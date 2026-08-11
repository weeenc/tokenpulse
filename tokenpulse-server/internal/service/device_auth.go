// 本文件实现设备码申请、用户审批、轮询换取设备凭据的完整状态机。
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tokenpulse/tokenpulse/server/internal/model"
	"github.com/tokenpulse/tokenpulse/server/internal/repository"
	"github.com/tokenpulse/tokenpulse/server/internal/security"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeviceAuthInput 是 Agent 发起设备授权时提交的安装元数据。
type DeviceAuthInput struct {
	DeviceName       string // DeviceName 为用户可识别的设备名称。
	Platform         string // Platform 为操作系统平台。
	Arch             string // Arch 为处理器架构。
	AgentVersion     string // AgentVersion 为 Agent 版本号。
	InstallationUUID string // InstallationUUID 为本地安装的稳定 UUID。
}

// DeviceAuthService 保存设备授权流程依赖及策略配置。
type DeviceAuthService struct {
	store      *repository.Store // store 为数据库访问入口。
	webBaseURL string            // webBaseURL 为用户审批页面根地址。
	expires    time.Duration     // expires 为授权请求有效期。
}

// DeviceTokenResult 是授权完成后一次性返回给 Agent 的凭据和账户信息。
type DeviceTokenResult struct {
	DeviceToken string       // DeviceToken 为仅返回一次的设备明文令牌。
	Device      model.Device // Device 为新建或重新连接的逻辑设备。
	Username    string       // Username 为授权账户展示名。
}

// NewDeviceAuthService 创建设备授权服务。
func NewDeviceAuthService(db *gorm.DB, webBaseURL string, expires time.Duration) *DeviceAuthService {
	return &DeviceAuthService{store: repository.New(db), webBaseURL: webBaseURL, expires: expires}
}

// WithContext 返回绑定请求上下文的新设备授权服务实例。
func (s *DeviceAuthService) WithContext(ctx context.Context) *DeviceAuthService {
	return &DeviceAuthService{store: s.store.WithContext(ctx), webBaseURL: s.webBaseURL, expires: s.expires}
}

// Request 生成高熵设备码和便于人工输入的用户码，并持久化 PENDING 请求。
func (s *DeviceAuthService) Request(input DeviceAuthInput) (*model.DeviceAuthRequest, string, error) {
	deviceCode, err := security.RandomToken("dc_", 32)
	if err != nil {
		return nil, "", err
	}
	var request *model.DeviceAuthRequest
	// 用户码空间较小，发生唯一键冲突时最多重新生成五次。
	for attempts := 0; attempts < 5; attempts++ {
		userCode, codeErr := security.UserCode()
		if codeErr != nil {
			return nil, "", codeErr
		}
		candidate := &model.DeviceAuthRequest{
			DeviceCodeHash: security.SHA256(deviceCode), UserCode: userCode,
			DeviceName: input.DeviceName, Platform: input.Platform, Arch: input.Arch,
			AgentVersion: input.AgentVersion, InstallationUUID: input.InstallationUUID,
			Status: "PENDING", ExpiresAt: time.Now().UTC().Add(s.expires),
		}
		if createErr := s.store.Query().Create(candidate).Error; createErr == nil {
			request = candidate
			break
		} else if !isDuplicate(createErr) {
			return nil, "", fmt.Errorf("create device auth request: %w", createErr)
		}
	}
	if request == nil {
		return nil, "", fmt.Errorf("could not generate unique user code")
	}
	return request, deviceCode, nil
}

// Info 返回待审批请求和当前用户可选择重新连接的有效设备。
func (s *DeviceAuthService) Info(userID uint64, userCode string) (*model.DeviceAuthRequest, []model.Device, error) {
	var request model.DeviceAuthRequest
	if err := s.store.Query().Where("user_code = ?", userCode).First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("find device auth request: %w", err)
	}
	if request.ExpiresAt.Before(time.Now().UTC()) {
		if err := s.expire(&request); err != nil {
			return nil, nil, err
		}
		return nil, nil, ErrExpired
	}
	if request.Status != "PENDING" {
		return nil, nil, ErrInvalidState
	}
	var devices []model.Device
	if err := s.store.Query().Where("user_id = ? AND status = ?", userID, "ACTIVE").Order("last_active_at DESC").Find(&devices).Error; err != nil {
		return nil, nil, fmt.Errorf("list devices: %w", err)
	}
	return &request, devices, nil
}

// Approve 在行锁保护下批准请求，并可绑定当前用户已有的目标设备。
func (s *DeviceAuthService) Approve(userID uint64, userCode string, targetDeviceID *uint64) error {
	return s.store.Transaction(func(transaction *repository.Store) error {
		tx := transaction.Query()
		var request model.DeviceAuthRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_code = ?", userCode).First(&request).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("lock device request: %w", err)
		}
		if request.ExpiresAt.Before(time.Now().UTC()) {
			tx.Model(&request).Update("status", "EXPIRED")
			return ErrExpired
		}
		if request.Status != "PENDING" {
			return ErrInvalidState
		}
		if targetDeviceID != nil {
			// 重新连接只允许选择审批用户本人拥有的有效设备。
			var count int64
			if err := tx.Model(&model.Device{}).Where("id = ? AND user_id = ? AND status = ?", *targetDeviceID, userID, "ACTIVE").Count(&count).Error; err != nil {
				return fmt.Errorf("validate target device: %w", err)
			}
			if count != 1 {
				return ErrForbidden
			}
		}
		now := time.Now().UTC()
		result := tx.Model(&request).Where("status = ?", "PENDING").Updates(map[string]any{
			"status": "APPROVED", "approved_user_id": userID, "target_device_id": targetDeviceID, "approved_at": now,
		})
		if result.Error != nil {
			return fmt.Errorf("approve device request: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrInvalidState
		}
		return nil
	})
}

// Deny 将仍在有效期内的 PENDING 请求原子更新为 DENIED。
func (s *DeviceAuthService) Deny(userID uint64, userCode string) error {
	result := s.store.Query().Model(&model.DeviceAuthRequest{}).
		Where("user_code = ? AND status = ? AND expires_at > ?", userCode, "PENDING", time.Now().UTC()).
		Updates(map[string]any{"status": "DENIED", "approved_user_id": userID})
	if result.Error != nil {
		return fmt.Errorf("deny device request: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return ErrInvalidState
	}
	return nil
}

// Token 供 Agent 轮询设备码状态，并在 APPROVED 后一次性签发设备凭据。
func (s *DeviceAuthService) Token(deviceCode string) (*DeviceTokenResult, string, error) {
	var result *DeviceTokenResult
	state := ""
	err := s.store.Transaction(func(transaction *repository.Store) error {
		tx := transaction.Query()
		var request model.DeviceAuthRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("device_code_hash = ?", security.SHA256(deviceCode)).First(&request).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUnauthorized
			}
			return fmt.Errorf("find device request: %w", err)
		}
		if request.ExpiresAt.Before(time.Now().UTC()) {
			tx.Model(&request).Update("status", "EXPIRED")
			state = "expired_token"
			return ErrExpired
		}
		switch request.Status {
		// state 使用设备授权协议风格的稳定错误字符串，便于 Agent 决定是否继续轮询。
		case "PENDING":
			state = "authorization_pending"
			return ErrInvalidState
		case "DENIED":
			state = "access_denied"
			return ErrForbidden
		case "CONSUMED":
			state = "expired_token"
			return ErrExpired
		case "APPROVED":
		default:
			state = "expired_token"
			return ErrInvalidState
		}
		if request.ApprovedUserID == nil {
			return fmt.Errorf("approved request has no user")
		}
		var device model.Device
		if request.TargetDeviceID == nil {
			// 首次授权创建逻辑设备；重新连接则复用设备 UUID 并轮换全部旧凭据。
			device = model.Device{DeviceUUID: uuid.NewString(), UserID: *request.ApprovedUserID, DeviceName: request.DeviceName, Platform: request.Platform, Arch: request.Arch, Status: "ACTIVE"}
			if err := tx.Create(&device).Error; err != nil {
				return fmt.Errorf("create device: %w", err)
			}
		} else {
			if err := tx.Where("id = ? AND user_id = ? AND status = ?", *request.TargetDeviceID, *request.ApprovedUserID, "ACTIVE").First(&device).Error; err != nil {
				return ErrForbidden
			}
			now := time.Now().UTC()
			if err := tx.Model(&model.DeviceInstallation{}).Where("device_id = ? AND credential_status = ?", device.ID, "ACTIVE").Updates(map[string]any{"credential_status": "REVOKED", "revoked_at": now}).Error; err != nil {
				return fmt.Errorf("revoke old credentials: %w", err)
			}
		}
		token, err := security.RandomToken("dt_", 32)
		if err != nil {
			return err
		}
		hash := security.SHA256(token)
		// 数据库只保存摘要，明文设备令牌只存在于本次响应中。
		installation := model.DeviceInstallation{DeviceID: device.ID, InstallationUUID: request.InstallationUUID, AgentVersion: request.AgentVersion, CredentialHash: &hash, CredentialStatus: "ACTIVE"}
		if err := tx.Create(&installation).Error; err != nil {
			return fmt.Errorf("create installation: %w", err)
		}
		if err := tx.Model(&request).Where("status = ?", "APPROVED").Update("status", "CONSUMED").Error; err != nil {
			return fmt.Errorf("consume device request: %w", err)
		}
		var user model.User
		if err := tx.First(&user, *request.ApprovedUserID).Error; err != nil {
			return fmt.Errorf("find approved user: %w", err)
		}
		result = &DeviceTokenResult{DeviceToken: token, Device: device, Username: user.Username}
		return nil
	})
	return result, state, err
}

// ExpirePending 批量标记所有已到期但仍处于等待状态的请求。
func (s *DeviceAuthService) ExpirePending() error {
	return s.store.Query().Model(&model.DeviceAuthRequest{}).Where("status = ? AND expires_at <= ?", "PENDING", time.Now().UTC()).Update("status", "EXPIRED").Error
}

// expire 幂等地将单条等待中的授权请求标记为过期。
func (s *DeviceAuthService) expire(request *model.DeviceAuthRequest) error {
	if err := s.store.Query().Model(request).Where("status = ?", "PENDING").Update("status", "EXPIRED").Error; err != nil {
		return fmt.Errorf("expire device request: %w", err)
	}
	return nil
}
