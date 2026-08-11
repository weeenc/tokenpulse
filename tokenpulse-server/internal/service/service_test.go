// 本文件使用真实 MySQL 测试库覆盖跨服务的核心业务流程和事务约束。
package service

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tokenpulse/tokenpulse/server/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB 连接 MYSQL_TEST_DSN，重建隔离表结构；未配置时跳过集成测试。
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not configured; integration test skipped")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), NowFunc: func() time.Time { return time.Now().UTC() }})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&model.RefreshSession{}, &model.ModelPrice{}, &model.UsageEvent{}, &model.DeviceAuthRequest{}, &model.DeviceInstallation{}, &model.Device{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Device{}, &model.DeviceInstallation{}, &model.DeviceAuthRequest{}, &model.UsageEvent{}, &model.ModelPrice{}, &model.RefreshSession{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// user 创建测试用户并在失败时立即终止当前测试。
func user(t *testing.T, db *gorm.DB, name string) *model.User {
	t.Helper()
	created, err := NewAuthService(db).Register(name, name+"@example.com", "a-secure-password")
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// request 创建一条测试设备授权请求并返回请求记录和明文设备码。
func request(t *testing.T, service *DeviceAuthService, installation string) (*model.DeviceAuthRequest, string) {
	t.Helper()
	created, deviceCode, err := service.Request(DeviceAuthInput{DeviceName: "Test Mac", Platform: "darwin", Arch: "arm64", AgentVersion: "0.1.0", InstallationUUID: installation})
	if err != nil {
		t.Fatal(err)
	}
	return created, deviceCode
}

// TestUserRegisterAndLogin 覆盖注册唯一约束、正常登录和错误密码拒绝。
func TestUserRegisterAndLogin(t *testing.T) {
	db := testDB(t)
	auth := NewAuthService(db)
	created, err := auth.Register("alice", "alice@example.com", "a-secure-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Register("alice", "different@example.com", "a-secure-password"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	loggedIn, err := auth.Login("alice", "a-secure-password")
	if err != nil || loggedIn.ID != created.ID {
		t.Fatalf("login failed: %v", err)
	}
	if _, err := auth.Login("alice", "wrong-password"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

// TestRefreshSessionRotatesAndCannotBeReused 验证轮换、重放整族撤销和主动注销。
func TestRefreshSessionRotatesAndCannotBeReused(t *testing.T) {
	db := testDB(t)
	owner := user(t, db, "refresh-owner")
	auth := NewAuthService(db)
	first, err := auth.CreateRefreshSession(owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	userID, second, err := auth.RotateRefreshSession(first, time.Hour)
	if err != nil || userID != owner.ID || second == "" || second == first {
		t.Fatalf("refresh rotation failed: user=%d err=%v", userID, err)
	}
	if _, _, err := auth.RotateRefreshSession(first, time.Hour); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old refresh token was reusable: %v", err)
	}
	if _, _, err := auth.RotateRefreshSession(second, time.Hour); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("refresh family survived replay detection: %v", err)
	}
	second, err = auth.CreateRefreshSession(owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.RevokeRefreshSession(second); err != nil {
		t.Fatal(err)
	}
	if _, _, err := auth.RotateRefreshSession(second, time.Hour); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("revoked refresh token was accepted: %v", err)
	}
}

// TestDeviceAuthorizationLifecycle 覆盖设备授权从等待、批准、换取令牌到凭据认证的全流程。
func TestDeviceAuthorizationLifecycle(t *testing.T) {
	db := testDB(t)
	owner := user(t, db, "owner")
	service := NewDeviceAuthService(db, "http://localhost", 10*time.Minute)
	pending, code := request(t, service, uuid.NewString())
	if _, _, err := service.Info(owner.ID, pending.UserCode); err != nil {
		t.Fatal(err)
	}
	if _, state, err := service.Token(code); !errors.Is(err, ErrInvalidState) || state != "authorization_pending" {
		t.Fatalf("unexpected pending result: %s %v", state, err)
	}
	if err := service.Approve(owner.ID, pending.UserCode, nil); err != nil {
		t.Fatal(err)
	}
	result, state, err := service.Token(code)
	if err != nil || state != "" || result.DeviceToken == "" || result.Device.UserID != owner.ID {
		t.Fatalf("token exchange failed: %s %v", state, err)
	}
	if _, state, err := service.Token(code); !errors.Is(err, ErrExpired) || state != "expired_token" {
		t.Fatalf("device code was consumed twice: %s %v", state, err)
	}
	identity, err := NewDeviceService(db).Authenticate(result.DeviceToken)
	if err != nil || identity.DeviceID != result.Device.ID {
		t.Fatalf("device auth failed: %v", err)
	}
}

// TestDeviceAuthorizationDenyAndExpire 验证拒绝和过期状态不会签发设备凭据。
func TestDeviceAuthorizationDenyAndExpire(t *testing.T) {
	db := testDB(t)
	owner := user(t, db, "owner")
	service := NewDeviceAuthService(db, "http://localhost", 10*time.Minute)
	denied, deniedCode := request(t, service, uuid.NewString())
	if err := service.Deny(owner.ID, denied.UserCode); err != nil {
		t.Fatal(err)
	}
	if _, state, err := service.Token(deniedCode); !errors.Is(err, ErrForbidden) || state != "access_denied" {
		t.Fatalf("unexpected denied result: %s %v", state, err)
	}
	expired, expiredCode := request(t, service, uuid.NewString())
	db.Model(expired).Update("expires_at", time.Now().UTC().Add(-time.Minute))
	if err := service.Approve(owner.ID, expired.UserCode, nil); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired code approved: %v", err)
	}
	if _, state, err := service.Token(expiredCode); !errors.Is(err, ErrExpired) || state != "expired_token" {
		t.Fatalf("unexpected expired result: %s %v", state, err)
	}
}

// TestReconnectOwnershipRotationAndRevoke 验证设备归属隔离、重连凭据轮换和整机撤销。
func TestReconnectOwnershipRotationAndRevoke(t *testing.T) {
	db := testDB(t)
	owner := user(t, db, "owner")
	other := user(t, db, "other")
	auth := NewDeviceAuthService(db, "http://localhost", 10*time.Minute)
	devices := NewDeviceService(db)
	firstRequest, firstCode := request(t, auth, uuid.NewString())
	if err := auth.Approve(owner.ID, firstRequest.UserCode, nil); err != nil {
		t.Fatal(err)
	}
	first, _, err := auth.Token(firstCode)
	if err != nil {
		t.Fatal(err)
	}
	reconnect, reconnectCode := request(t, auth, uuid.NewString())
	if err := auth.Approve(other.ID, reconnect.UserCode, &first.Device.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other user reconnected device: %v", err)
	}
	if err := auth.Approve(owner.ID, reconnect.UserCode, &first.Device.ID); err != nil {
		t.Fatal(err)
	}
	second, _, err := auth.Token(reconnectCode)
	if err != nil {
		t.Fatal(err)
	}
	if second.Device.DeviceUUID != first.Device.DeviceUUID {
		t.Fatal("reconnect changed logical device id")
	}
	if _, err := devices.Authenticate(first.DeviceToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("old credential remained active")
	}
	if _, err := devices.Authenticate(second.DeviceToken); err != nil {
		t.Fatalf("new credential not active: %v", err)
	}
	if err := devices.Revoke(owner.ID, second.Device.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := devices.Authenticate(second.DeviceToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("revoked credential still authenticates")
	}
	listed, err := devices.List(owner.ID)
	if err != nil || len(listed) != 1 || listed[0].AgentVersion != "0.1.0" {
		t.Fatalf("device list omitted the latest agent version: %+v %v", listed, err)
	}
}

// TestUsageBatchDeduplicatesAndStatisticsAggregate 验证幂等入库、统计聚合和费用估算。
func TestUsageBatchDeduplicatesAndStatisticsAggregate(t *testing.T) {
	db := testDB(t)
	owner := user(t, db, "owner")
	device := model.Device{DeviceUUID: uuid.NewString(), UserID: owner.ID, DeviceName: "Mac", Platform: "darwin", Arch: "arm64", Status: "ACTIVE"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	installation := model.DeviceInstallation{DeviceID: device.ID, InstallationUUID: uuid.NewString(), CredentialStatus: "ACTIVE"}
	if err := db.Create(&installation).Error; err != nil {
		t.Fatal(err)
	}
	identity := DeviceIdentity{UserID: owner.ID, DeviceID: device.ID, InstallationID: installation.ID}
	input := UsageInput{EventID: "a" + fmt.Sprintf("%063d", 0), Source: "codex", InputTokens: 100, OutputTokens: 20, CachedInputTokens: 40, ReasoningTokens: 5, TotalTokens: 120, OccurredAt: time.Now().UTC()}
	modelName := "gpt-test-priced"
	input.Model = &modelName
	if err := db.Create(&model.ModelPrice{Provider: "openai", Model: modelName, InputPricePerMillion: floatPointer(2), OutputPricePerMillion: floatPointer(10), CachedInputPricePerMillion: floatPointer(0.2), EffectiveAt: time.Now().UTC().Add(-time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	usage := NewUsageService(db)
	if inserted, err := usage.Batch(identity, []UsageInput{input}); err != nil || inserted != 1 {
		t.Fatalf("first upload: %d %v", inserted, err)
	}
	if inserted, err := usage.Batch(identity, []UsageInput{input}); err != nil || inserted != 0 {
		t.Fatalf("duplicate upload: %d %v", inserted, err)
	}
	var count int64
	db.Model(&model.UsageEvent{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected one event, got %d", count)
	}
	summary, err := NewStatisticsService(db).Summary(owner.ID, StatisticsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalTokens != 120 || summary.Today != 120 || summary.InputTokens != 100 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.EstimatedCostUSD <= 0 {
		t.Fatalf("expected a non-zero estimated cost: %+v", summary)
	}
	groups, err := NewStatisticsService(db).By(owner.ID, StatisticsFilter{}, "device")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Key != "Mac" || groups[0].TotalTokens != 120 {
		t.Fatalf("unexpected device stats: %+v", groups)
	}
}

// floatPointer 为测试价格字段创建 float64 指针。
func floatPointer(value float64) *float64 { return &value }
