// Package model 定义与 MySQL 表一一映射的核心持久化模型。
package model

import "time"

// User 表示可登录 Web 控制台的用户账户。
type User struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`                          // ID 为账户自增主键。
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`  // Username 为唯一登录用户名。
	Email        *string   `gorm:"size:128;uniqueIndex" json:"email,omitempty"`   // Email 为可选且唯一的登录邮箱。
	PasswordHash string    `gorm:"size:255;not null" json:"-"`                    // PasswordHash 保存 bcrypt 哈希，禁止输出到 JSON。
	Status       string    `gorm:"size:16;not null;default:ACTIVE" json:"status"` // Status 表示账户是否可用。
	CreatedAt    time.Time `json:"createdAt"`                                     // CreatedAt 为账户创建时间（UTC）。
	UpdatedAt    time.Time `json:"updatedAt"`                                     // UpdatedAt 为账户最后更新时间（UTC）。
}

// Device 表示用户账户下稳定存在的逻辑设备。
type Device struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`                                            // ID 为设备自增主键。
	DeviceUUID   string     `gorm:"column:device_uuid;size:36;uniqueIndex;not null" json:"deviceId"` // DeviceUUID 是暴露给客户端的稳定设备标识。
	UserID       uint64     `gorm:"index;not null" json:"-"`                                         // UserID 为设备所属账户。
	DeviceName   string     `gorm:"size:128;not null" json:"deviceName"`                             // DeviceName 为用户可修改的展示名称。
	Platform     string     `gorm:"size:32;not null" json:"platform"`                                // Platform 为操作系统平台。
	Arch         string     `gorm:"size:32;not null" json:"arch"`                                    // Arch 为处理器架构。
	Status       string     `gorm:"size:16;not null;default:ACTIVE" json:"status"`                   // Status 表示设备有效或已撤销。
	LastActiveAt *time.Time `json:"lastActiveAt"`                                                    // LastActiveAt 为最近心跳时间。
	CreatedAt    time.Time  `json:"createdAt"`                                                       // CreatedAt 为设备创建时间。
	UpdatedAt    time.Time  `json:"updatedAt"`                                                       // UpdatedAt 为设备更新时间。
}

// DeviceInstallation 表示某次 Agent 安装及其可轮换的设备凭据。
type DeviceInstallation struct {
	ID               uint64     `gorm:"primaryKey"`                                            // ID 为安装记录自增主键。
	DeviceID         uint64     `gorm:"index;not null"`                                        // DeviceID 关联逻辑设备。
	InstallationUUID string     `gorm:"column:installation_uuid;size:36;uniqueIndex;not null"` // InstallationUUID 标识本地 Agent 安装实例。
	AgentVersion     string     `gorm:"size:32"`                                               // AgentVersion 为授权时上报的 Agent 版本。
	CredentialHash   *string    `gorm:"size:64;uniqueIndex"`                                   // CredentialHash 为设备令牌 SHA-256 摘要。
	CredentialStatus string     `gorm:"size:16;not null;default:ACTIVE"`                       // CredentialStatus 表示凭据是否仍可用。
	LastActiveAt     *time.Time // LastActiveAt 为该安装最近活动时间。
	CreatedAt        time.Time  // CreatedAt 为安装授权完成时间。
	RevokedAt        *time.Time // RevokedAt 为凭据撤销时间。
}

// DeviceAuthRequest 表示 OAuth 设备授权风格的一次短期授权流程。
type DeviceAuthRequest struct {
	ID               uint64     `gorm:"primaryKey"`                       // ID 为授权请求自增主键。
	DeviceCodeHash   string     `gorm:"size:64;uniqueIndex;not null"`     // DeviceCodeHash 为设备码摘要，不保存明文。
	UserCode         string     `gorm:"size:16;uniqueIndex;not null"`     // UserCode 为用户在 Web 端输入的短码。
	DeviceName       string     `gorm:"size:128;not null"`                // DeviceName 为 Agent 上报的设备名称。
	Platform         string     `gorm:"size:32;not null"`                 // Platform 为 Agent 上报的操作系统。
	Arch             string     `gorm:"size:32;not null"`                 // Arch 为 Agent 上报的处理器架构。
	AgentVersion     string     `gorm:"size:32"`                          // AgentVersion 为发起授权的版本。
	InstallationUUID string     `gorm:"size:36;not null"`                 // InstallationUUID 为发起授权的安装标识。
	Status           string     `gorm:"size:16;not null;default:PENDING"` // Status 为授权流程当前状态。
	ApprovedUserID   *uint64    // ApprovedUserID 为批准或拒绝请求的用户。
	TargetDeviceID   *uint64    // TargetDeviceID 非空时表示重新连接已有设备。
	ExpiresAt        time.Time  // ExpiresAt 为授权请求失效时间。
	CreatedAt        time.Time  // CreatedAt 为授权请求创建时间。
	ApprovedAt       *time.Time // ApprovedAt 为用户批准时间。
}

// UsageEvent 表示 Agent 上报的一条不可变 Token 使用事件。
type UsageEvent struct {
	ID                uint64    `gorm:"primaryKey"`                                                                          // ID 为事件自增主键。
	UserID            uint64    `gorm:"uniqueIndex:uk_usage_event,priority:1;index:idx_usage_user_time,priority:1;not null"` // UserID 为事件所属用户，也是幂等键的一部分。
	DeviceID          uint64    `gorm:"index:idx_usage_device_time,priority:1;not null"`                                     // DeviceID 为事件来源逻辑设备。
	InstallationID    *uint64   // InstallationID 为实际上报事件的 Agent 安装。
	EventID           string    `gorm:"size:64;uniqueIndex:uk_usage_event,priority:3;not null"`                                                                                                            // EventID 为 Agent 生成的事件摘要标识。
	Source            string    `gorm:"size:64;uniqueIndex:uk_usage_event,priority:2;index:idx_usage_source_time,priority:1;not null"`                                                                     // Source 为 codex、claude-code 等来源。
	Model             *string   `gorm:"size:128;index:idx_usage_model_time,priority:1"`                                                                                                                    // Model 为模型名称，无法识别时为空。
	SessionID         *string   `gorm:"size:255"`                                                                                                                                                          // SessionID 为来源工具的会话标识。
	MessageID         *string   `gorm:"size:255"`                                                                                                                                                          // MessageID 为来源工具的消息标识。
	InputTokens       uint64    `gorm:"not null;default:0"`                                                                                                                                                // InputTokens 为输入 Token 数。
	OutputTokens      uint64    `gorm:"not null;default:0"`                                                                                                                                                // OutputTokens 为输出 Token 数。
	CachedInputTokens uint64    `gorm:"not null;default:0"`                                                                                                                                                // CachedInputTokens 为命中缓存的输入 Token 数。
	ReasoningTokens   uint64    `gorm:"not null;default:0"`                                                                                                                                                // ReasoningTokens 为输出中的推理 Token 数。
	TotalTokens       uint64    `gorm:"not null;default:0"`                                                                                                                                                // TotalTokens 为来源上报的 Token 总数；细分不可用时仍可非零。
	OccurredAt        time.Time `gorm:"index:idx_usage_user_time,priority:2;index:idx_usage_device_time,priority:2;index:idx_usage_source_time,priority:2;index:idx_usage_model_time,priority:2;not null"` // OccurredAt 为来源事件实际发生时间。
	CreatedAt         time.Time // CreatedAt 为服务端入库时间。
}

// ModelPrice 表示某模型从指定生效时间开始使用的每百万 Token 价格。
type ModelPrice struct {
	ID                         uint64    `gorm:"primaryKey"`                                         // ID 为价格记录自增主键。
	Provider                   string    `gorm:"size:64;index:idx_model_price,priority:1;not null"`  // Provider 为模型供应商。
	Model                      string    `gorm:"size:128;index:idx_model_price,priority:2;not null"` // Model 为价格适用的模型名称或前缀。
	InputPricePerMillion       *float64  // InputPricePerMillion 为每百万输入 Token 的美元价格。
	OutputPricePerMillion      *float64  // OutputPricePerMillion 为每百万输出 Token 的美元价格。
	CachedInputPricePerMillion *float64  // CachedInputPricePerMillion 为每百万缓存输入 Token 的美元价格。
	EffectiveAt                time.Time `gorm:"index:idx_model_price,priority:3;not null"` // EffectiveAt 为价格开始生效的 UTC 时间。
	CreatedAt                  time.Time // CreatedAt 为价格记录创建时间。
}

// RefreshSession 表示可轮换、可撤销并具备重放检测能力的刷新令牌会话。
type RefreshSession struct {
	ID         uint64     `gorm:"primaryKey"`                   // ID 为刷新会话自增主键。
	UserID     uint64     `gorm:"index;not null"`               // UserID 为会话所属用户。
	FamilyID   string     `gorm:"size:36;index;not null"`       // FamilyID 关联同一轮换链，用于检测重放后整族撤销。
	TokenHash  string     `gorm:"size:64;uniqueIndex;not null"` // TokenHash 为刷新令牌 SHA-256 摘要。
	ExpiresAt  time.Time  `gorm:"index;not null"`               // ExpiresAt 为令牌绝对失效时间。
	CreatedAt  time.Time  `gorm:"not null"`                     // CreatedAt 为会话创建时间。
	LastUsedAt *time.Time // LastUsedAt 为最后轮换或撤销时间。
	RevokedAt  *time.Time `gorm:"index"` // RevokedAt 非空表示会话已失效。
}

// TableName 固定 User 与既有 users 表的映射。
func (User) TableName() string { return "users" }

// TableName 固定 Device 与既有 devices 表的映射。
func (Device) TableName() string { return "devices" }

// TableName 固定 DeviceInstallation 与既有 device_installations 表的映射。
func (DeviceInstallation) TableName() string { return "device_installations" }

// TableName 固定 DeviceAuthRequest 与既有 device_auth_requests 表的映射。
func (DeviceAuthRequest) TableName() string { return "device_auth_requests" }

// TableName 固定 UsageEvent 与既有 usage_events 表的映射。
func (UsageEvent) TableName() string { return "usage_events" }

// TableName 固定 ModelPrice 与既有 model_prices 表的映射。
func (ModelPrice) TableName() string { return "model_prices" }

// TableName 固定 RefreshSession 与既有 refresh_sessions 表的映射。
func (RefreshSession) TableName() string { return "refresh_sessions" }
