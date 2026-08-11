// 本文件实现设备凭据认证、设备管理和 Agent 心跳业务。
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tokenpulse/tokenpulse/server/internal/model"
	"github.com/tokenpulse/tokenpulse/server/internal/repository"
	"github.com/tokenpulse/tokenpulse/server/internal/security"
	"gorm.io/gorm"
)

// DeviceIdentity 是设备令牌认证后的可信主体信息。
type DeviceIdentity struct {
	UserID         uint64 // UserID 为设备所属用户。
	DeviceID       uint64 // DeviceID 为逻辑设备主键。
	InstallationID uint64 // InstallationID 为当前凭据所属安装记录主键。
}

// DeviceService 提供设备查询、认证和状态变更能力。
type DeviceService struct {
	store *repository.Store // store 为设备域的数据库访问入口。
}

// DeviceView 在设备基础信息上附加最近一次安装的 Agent 版本。
type DeviceView struct {
	model.Device        // Device 为逻辑设备基础字段。
	AgentVersion string `json:"agentVersion"` // AgentVersion 为该设备最近安装记录中的版本。
}

// NewDeviceService 创建设备服务。
func NewDeviceService(db *gorm.DB) *DeviceService { return &DeviceService{store: repository.New(db)} }

// WithContext 返回绑定请求上下文的新设备服务实例。
func (s *DeviceService) WithContext(ctx context.Context) *DeviceService {
	return &DeviceService{store: s.store.WithContext(ctx)}
}

// Authenticate 通过设备令牌摘要查找有效安装，并确认逻辑设备仍处于 ACTIVE 状态。
func (s *DeviceService) Authenticate(token string) (*DeviceIdentity, error) {
	var installation model.DeviceInstallation
	if err := s.store.Query().Where("credential_hash = ? AND credential_status = ?", security.SHA256(token), "ACTIVE").First(&installation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("find device credential: %w", err)
	}
	var device model.Device
	if err := s.store.Query().Where("id = ? AND status = ?", installation.DeviceID, "ACTIVE").First(&device).Error; err != nil {
		return nil, ErrUnauthorized
	}
	return &DeviceIdentity{UserID: device.UserID, DeviceID: device.ID, InstallationID: installation.ID}, nil
}

// List 查询用户的全部设备，并通过子查询附加每台设备最新的 Agent 版本。
func (s *DeviceService) List(userID uint64) ([]DeviceView, error) {
	var devices []DeviceView
	if err := s.store.Query().Table("devices").
		Select(`devices.*,
			COALESCE((SELECT agent_version FROM device_installations
			 WHERE device_installations.device_id = devices.id
			 ORDER BY device_installations.created_at DESC LIMIT 1), '') AS agent_version`).
		Where("devices.user_id = ?", userID).Order("devices.created_at DESC").Scan(&devices).Error; err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return devices, nil
}

// Get 查询属于指定用户的单台设备，避免跨用户枚举设备。
func (s *DeviceService) Get(userID, deviceID uint64) (*DeviceView, error) {
	var device DeviceView
	if err := s.store.Query().Table("devices").
		Select(`devices.*,
			COALESCE((SELECT agent_version FROM device_installations
			 WHERE device_installations.device_id = devices.id
			 ORDER BY device_installations.created_at DESC LIMIT 1), '') AS agent_version`).
		Where("devices.id = ? AND devices.user_id = ?", deviceID, userID).Scan(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get device: %w", err)
	}
	if device.ID == 0 {
		return nil, ErrNotFound
	}
	return &device, nil
}

// Rename 仅允许重命名当前用户拥有且仍有效的设备。
func (s *DeviceService) Rename(userID, deviceID uint64, name string) error {
	result := s.store.Query().Model(&model.Device{}).Where("id = ? AND user_id = ? AND status = ?", deviceID, userID, "ACTIVE").Update("device_name", name)
	if result.Error != nil {
		return fmt.Errorf("rename device: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return ErrNotFound
	}
	return nil
}

// Revoke 在同一事务中撤销逻辑设备及其全部有效安装凭据。
func (s *DeviceService) Revoke(userID, deviceID uint64) error {
	return s.store.Transaction(func(transaction *repository.Store) error {
		tx := transaction.Query()
		result := tx.Model(&model.Device{}).Where("id = ? AND user_id = ? AND status = ?", deviceID, userID, "ACTIVE").Update("status", "REVOKED")
		if result.Error != nil {
			return fmt.Errorf("revoke device: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrNotFound
		}
		now := time.Now().UTC()
		if err := tx.Model(&model.DeviceInstallation{}).Where("device_id = ? AND credential_status = ?", deviceID, "ACTIVE").Updates(map[string]any{"credential_status": "REVOKED", "revoked_at": now}).Error; err != nil {
			return fmt.Errorf("revoke credentials: %w", err)
		}
		return nil
	})
}

// Me 汇总当前 Agent 所需的设备、安装版本和账户展示信息。
func (s *DeviceService) Me(identity DeviceIdentity) (map[string]any, error) {
	var device model.Device
	var installation model.DeviceInstallation
	var user model.User
	if err := s.store.Query().First(&device, identity.DeviceID).Error; err != nil {
		return nil, err
	}
	if err := s.store.Query().First(&installation, identity.InstallationID).Error; err != nil {
		return nil, err
	}
	if err := s.store.Query().First(&user, identity.UserID).Error; err != nil {
		return nil, err
	}
	return map[string]any{"deviceId": device.DeviceUUID, "deviceName": device.DeviceName, "installationId": installation.InstallationUUID, "platform": device.Platform, "arch": device.Arch, "agentVersion": installation.AgentVersion, "user": map[string]any{"username": user.Username}}, nil
}

// Heartbeat 在同一事务中更新逻辑设备和当前安装的最近活动时间。
func (s *DeviceService) Heartbeat(identity DeviceIdentity) error {
	now := time.Now().UTC()
	return s.store.Transaction(func(transaction *repository.Store) error {
		tx := transaction.Query()
		if err := tx.Model(&model.Device{}).Where("id = ? AND status = ?", identity.DeviceID, "ACTIVE").Update("last_active_at", now).Error; err != nil {
			return err
		}
		return tx.Model(&model.DeviceInstallation{}).Where("id = ? AND credential_status = ?", identity.InstallationID, "ACTIVE").Update("last_active_at", now).Error
	})
}
