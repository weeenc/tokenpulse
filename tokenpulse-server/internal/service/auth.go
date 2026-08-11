// Package service 承载认证、设备、用量和统计等核心业务规则。
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tokenpulse/tokenpulse/server/internal/model"
	"github.com/tokenpulse/tokenpulse/server/internal/repository"
	"github.com/tokenpulse/tokenpulse/server/internal/security"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 领域错误用于在 Service 与 HTTP Handler 之间稳定映射业务失败类型。
var (
	ErrConflict     = errors.New("conflict")      // ErrConflict 表示唯一资源冲突。
	ErrUnauthorized = errors.New("unauthorized")  // ErrUnauthorized 表示凭据无效或主体未认证。
	ErrNotFound     = errors.New("not found")     // ErrNotFound 表示目标资源不存在。
	ErrForbidden    = errors.New("forbidden")     // ErrForbidden 表示主体无权操作目标资源。
	ErrExpired      = errors.New("expired")       // ErrExpired 表示短期凭据或流程已过期。
	ErrInvalidState = errors.New("invalid state") // ErrInvalidState 表示资源当前状态不允许该操作。
)

// AuthService 实现用户账户和刷新会话业务。
type AuthService struct {
	store *repository.Store // store 为认证域的数据库访问入口。
}

// NewAuthService 创建认证服务。
func NewAuthService(db *gorm.DB) *AuthService { return &AuthService{store: repository.New(db)} }

// WithContext 返回绑定请求上下文的新认证服务实例。
func (s *AuthService) WithContext(ctx context.Context) *AuthService {
	return &AuthService{store: s.store.WithContext(ctx)}
}

// Register 规范化用户输入、哈希密码并创建唯一账户。
func (s *AuthService) Register(username, email, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	hash, err := security.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &model.User{Username: username, PasswordHash: hash, Status: "ACTIVE"}
	if normalized := strings.TrimSpace(strings.ToLower(email)); normalized != "" {
		user.Email = &normalized
	}
	if err := s.store.Query().Create(user).Error; err != nil {
		if isDuplicate(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// Login 支持使用用户名或邮箱登录，并拒绝非 ACTIVE 账户。
func (s *AuthService) Login(identity, password string) (*model.User, error) {
	var user model.User
	query := s.store.Query().Where("username = ?", identity)
	if strings.Contains(identity, "@") {
		query = s.store.Query().Where("email = ?", strings.ToLower(identity))
	}
	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user.Status != "ACTIVE" || !security.CheckPassword(user.PasswordHash, password) {
		return nil, ErrUnauthorized
	}
	return &user, nil
}

// User 按主键查询用户，不存在时返回统一的 ErrNotFound。
func (s *AuthService) User(id uint64) (*model.User, error) {
	var user model.User
	if err := s.store.Query().First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	return &user, nil
}

// CreateRefreshSession 创建新的随机刷新令牌，仅把摘要持久化到数据库。
func (s *AuthService) CreateRefreshSession(userID uint64, ttl time.Duration) (string, error) {
	token, err := security.RandomToken("rt_", 32)
	if err != nil {
		return "", err
	}
	session := model.RefreshSession{
		UserID: userID, FamilyID: uuid.NewString(), TokenHash: security.SHA256(token), ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if err := s.store.Query().Create(&session).Error; err != nil {
		return "", fmt.Errorf("create refresh session: %w", err)
	}
	return token, nil
}

// RotateRefreshSession 以事务和行锁完成一次性刷新令牌轮换。
// 如果检测到已撤销令牌被重放，会撤销同一 family 的全部有效会话。
func (s *AuthService) RotateRefreshSession(token string, ttl time.Duration) (uint64, string, error) {
	var userID uint64
	var replacement string
	var replayDetected bool
	err := s.store.Transaction(func(transaction *repository.Store) error {
		tx := transaction.Query()
		var session model.RefreshSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", security.SHA256(token)).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUnauthorized
			}
			return fmt.Errorf("find refresh session: %w", err)
		}
		now := time.Now().UTC()
		if session.RevokedAt != nil {
			// 已使用令牌再次出现即视为泄露，整条轮换链全部失效。
			if err := tx.Model(&model.RefreshSession{}).Where("family_id = ? AND revoked_at IS NULL", session.FamilyID).
				Updates(map[string]any{"revoked_at": now, "last_used_at": now}).Error; err != nil {
				return fmt.Errorf("revoke replayed refresh family: %w", err)
			}
			replayDetected = true
			return nil
		}
		if !session.ExpiresAt.After(now) {
			return ErrUnauthorized
		}
		var user model.User
		if err := tx.Where("id = ? AND status = ?", session.UserID, "ACTIVE").First(&user).Error; err != nil {
			return ErrUnauthorized
		}
		newToken, err := security.RandomToken("rt_", 32)
		if err != nil {
			return err
		}
		if err := tx.Model(&session).Updates(map[string]any{"revoked_at": now, "last_used_at": now}).Error; err != nil {
			return fmt.Errorf("revoke refresh session: %w", err)
		}
		newSession := model.RefreshSession{
			// 新会话继承 family ID，使后续可以追踪同一轮换链。
			UserID: session.UserID, FamilyID: session.FamilyID, TokenHash: security.SHA256(newToken), ExpiresAt: now.Add(ttl),
		}
		if err := tx.Create(&newSession).Error; err != nil {
			return fmt.Errorf("rotate refresh session: %w", err)
		}
		userID, replacement = session.UserID, newToken
		return nil
	})
	if err == nil && replayDetected {
		return 0, "", ErrUnauthorized
	}
	return userID, replacement, err
}

// DeleteExpiredRefreshSessions 删除超过保留期的过期或已撤销会话。
func (s *AuthService) DeleteExpiredRefreshSessions(retention time.Duration) error {
	cutoff := time.Now().UTC().Add(-retention)
	if err := s.store.Query().Where("expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", cutoff, cutoff).
		Delete(&model.RefreshSession{}).Error; err != nil {
		return fmt.Errorf("delete expired refresh sessions: %w", err)
	}
	return nil
}

// RevokeRefreshSession 幂等撤销给定刷新令牌；空令牌直接视为成功。
func (s *AuthService) RevokeRefreshSession(token string) error {
	if token == "" {
		return nil
	}
	now := time.Now().UTC()
	if err := s.store.Query().Model(&model.RefreshSession{}).
		Where("token_hash = ? AND revoked_at IS NULL", security.SHA256(token)).
		Updates(map[string]any{"revoked_at": now, "last_used_at": now}).Error; err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	return nil
}

// isDuplicate 兼容 MySQL 与测试数据库的唯一约束错误文本。
func isDuplicate(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate") || strings.Contains(text, "unique constraint")
}
