// 本文件实现 Agent Token 用量事件的批量、幂等入库。
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/tokenpulse/tokenpulse/server/internal/model"
	"github.com/tokenpulse/tokenpulse/server/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UsageInput 是 Handler 校验后传入业务层的单条用量事件。
type UsageInput struct {
	EventID           string    // EventID 为来源侧稳定事件标识。
	Source            string    // Source 为 codex、claude-code 等采集来源。
	Model             *string   // Model 为可选模型名称。
	SessionID         *string   // SessionID 为可选来源会话标识。
	MessageID         *string   // MessageID 为可选来源消息标识。
	InputTokens       uint64    // InputTokens 为输入 Token 数。
	OutputTokens      uint64    // OutputTokens 为输出 Token 数。
	CachedInputTokens uint64    // CachedInputTokens 为缓存输入 Token 数。
	ReasoningTokens   uint64    // ReasoningTokens 为推理 Token 数。
	TotalTokens       uint64    // TotalTokens 为来源上报的总数；细分不可用时仍可非零。
	OccurredAt        time.Time // OccurredAt 为事件实际发生时间。
}

// UsageService 提供用量事件写入能力。
type UsageService struct {
	store *repository.Store // store 为用量域的数据库访问入口。
}

// NewUsageService 创建用量服务。
func NewUsageService(db *gorm.DB) *UsageService { return &UsageService{store: repository.New(db)} }

// WithContext 返回绑定请求上下文的新用量服务实例。
func (s *UsageService) WithContext(ctx context.Context) *UsageService {
	return &UsageService{store: s.store.WithContext(ctx)}
}

// Batch 将输入映射为可信设备身份下的持久化事件，并按 500 条分批插入。
// 唯一键冲突使用 DoNothing 忽略，从而支持 Agent 安全重试和幂等同步。
func (s *UsageService) Batch(identity DeviceIdentity, inputs []UsageInput) (int64, error) {
	installationID := identity.InstallationID
	events := make([]model.UsageEvent, 0, len(inputs))
	for _, input := range inputs {
		events = append(events, model.UsageEvent{
			UserID: identity.UserID, DeviceID: identity.DeviceID, InstallationID: &installationID,
			EventID: input.EventID, Source: input.Source, Model: input.Model,
			SessionID: input.SessionID, MessageID: input.MessageID,
			InputTokens: input.InputTokens, OutputTokens: input.OutputTokens,
			CachedInputTokens: input.CachedInputTokens, ReasoningTokens: input.ReasoningTokens,
			TotalTokens: input.TotalTokens, OccurredAt: input.OccurredAt.UTC(),
		})
	}
	if len(events) == 0 {
		return 0, nil
	}
	result := s.store.Query().Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&events, 500)
	if result.Error != nil {
		return 0, fmt.Errorf("insert usage batch: %w", result.Error)
	}
	return result.RowsAffected, nil
}
