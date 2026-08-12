// 本文件实现用量汇总、趋势、分组、最近事件和费用估算查询。
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/tokenpulse/tokenpulse/server/internal/repository"
	"gorm.io/gorm"
)

// StatisticsFilter 描述所有统计接口共享的可选筛选条件。
type StatisticsFilter struct {
	StartTime *time.Time // StartTime 为包含的 UTC 起始时间。
	EndTime   *time.Time // EndTime 为不包含的 UTC 结束时间。
	DeviceID  *uint64    // DeviceID 限定单台逻辑设备。
	Source    string     // Source 限定采集来源。
	Model     string     // Model 限定模型名称。
	// TimezoneOffsetMinutes 遵循 JavaScript Date#getTimezoneOffset 语义：
	// UTC 减本地时间，因此上海为 -480。
	TimezoneOffsetMinutes int
}

// Totals 是不同统计视图共享的 Token 汇总字段。
type Totals struct {
	TotalTokens       uint64 `json:"totalTokens"`       // TotalTokens 为来源上报的总数；细分不可用时仍可非零。
	InputTokens       uint64 `json:"inputTokens"`       // InputTokens 为输入 Token 总数。
	OutputTokens      uint64 `json:"outputTokens"`      // OutputTokens 为输出 Token 总数。
	CachedInputTokens uint64 `json:"cachedInputTokens"` // CachedInputTokens 为缓存输入 Token 总数。
	ReasoningTokens   uint64 `json:"reasoningTokens"`   // ReasoningTokens 为推理 Token 总数。
}

// Summary 是统计概览，包含本地自然周期和筛选范围总计。
type Summary struct {
	Today            uint64  `json:"today"`            // Today 为用户本地当天 Token 数。
	Week             uint64  `json:"week"`             // Week 为用户本地本周 Token 数（周一开始）。
	Month            uint64  `json:"month"`            // Month 为用户本地本月 Token 数。
	EstimatedCostUSD float64 `json:"estimatedCostUsd"` // EstimatedCostUSD 为按历史生效价估算的美元成本。
	Totals                   // Totals 嵌入筛选范围内的各类 Token 总计。
}

// GroupTotal 表示一个设备、来源或模型维度的汇总行。
type GroupTotal struct {
	Key    string `json:"key"` // Key 为分组展示值。
	Totals        // Totals 嵌入该分组的各类 Token 总计。
}

// TrendPoint 表示用户本地某个自然日的用量汇总。
type TrendPoint struct {
	Date   string `json:"date"` // Date 格式为 YYYY-MM-DD。
	Totals        // Totals 嵌入该自然日的各类 Token 总计。
}

// RecentEvent 是最近用量列表返回的精简事件视图。
type RecentEvent struct {
	EventID     string    `json:"eventId"`     // EventID 为来源事件标识。
	Source      string    `json:"source"`      // Source 为采集来源。
	Model       *string   `json:"model"`       // Model 为可选模型名称。
	DeviceName  string    `json:"deviceName"`  // DeviceName 为来源设备展示名。
	TotalTokens uint64    `json:"totalTokens"` // TotalTokens 为该事件 Token 总数。
	OccurredAt  time.Time `json:"occurredAt"`  // OccurredAt 为事件发生时间。
}

// DayDetail 汇总单日用量、活动计数以及来源和模型分布。
type DayDetail struct {
	Totals
	EstimatedCostUSD float64      `json:"estimatedCostUsd"` // EstimatedCostUSD 为当天历史价格估算费用。
	Messages         int64        `json:"messages"`         // Messages 为具有消息标识的去重消息数。
	Sessions         int64        `json:"sessions"`         // Sessions 为具有会话标识的去重会话数。
	Events           int64        `json:"events"`           // Events 为当天采集事件数。
	Sources          []GroupTotal `json:"sources"`          // Sources 为当天按采集工具汇总的用量。
	Models           []GroupTotal `json:"models"`           // Models 为当天按模型汇总的用量。
}

// StatisticsService 提供只读统计查询能力。
type StatisticsService struct {
	store *repository.Store // store 为统计查询的数据库访问入口。
}

// NewStatisticsService 创建统计服务。
func NewStatisticsService(db *gorm.DB) *StatisticsService {
	return &StatisticsService{store: repository.New(db)}
}

// WithContext 返回绑定请求上下文的新统计服务实例。
func (s *StatisticsService) WithContext(ctx context.Context) *StatisticsService {
	return &StatisticsService{store: s.store.WithContext(ctx)}
}

// Summary 计算自然日、周、月、筛选范围 Token 总计和历史价格成本。
func (s *StatisticsService) Summary(userID uint64, filter StatisticsFilter) (Summary, error) {
	now := time.Now().UTC()
	today, week, month := summaryBoundaries(now, filter.TimezoneOffsetMinutes)
	var result Summary
	query := s.filtered(userID, filter).Select(`
		COALESCE(SUM(CASE WHEN occurred_at >= ? THEN total_tokens ELSE 0 END), 0) AS today,
		COALESCE(SUM(CASE WHEN occurred_at >= ? THEN total_tokens ELSE 0 END), 0) AS week,
		COALESCE(SUM(CASE WHEN occurred_at >= ? THEN total_tokens ELSE 0 END), 0) AS month,
		COALESCE(SUM(total_tokens), 0) AS total_tokens,
		COALESCE(SUM(input_tokens), 0) AS input_tokens,
		COALESCE(SUM(output_tokens), 0) AS output_tokens,
		COALESCE(SUM(cached_input_tokens), 0) AS cached_input_tokens,
		COALESCE(SUM(reasoning_tokens), 0) AS reasoning_tokens`, today, week, month)
	if err := query.Scan(&result).Error; err != nil {
		return Summary{}, fmt.Errorf("statistics summary: %w", err)
	}
	costQuery := s.filtered(userID, filter).Joins(`LEFT JOIN model_prices ON model_prices.id = (
		-- 优先匹配最长模型前缀，并取事件发生时已经生效的最新价格。
		SELECT price.id FROM model_prices AS price
		WHERE (price.model = usage_events.model OR usage_events.model LIKE CONCAT(price.model, '-%'))
		  AND price.provider = CASE usage_events.source
		    WHEN 'codex' THEN 'openai'
		    WHEN 'claude-code' THEN 'anthropic'
		    ELSE price.provider END
		  AND price.effective_at <= usage_events.occurred_at
		ORDER BY CHAR_LENGTH(price.model) DESC, price.effective_at DESC LIMIT 1
	)`).Select(`COALESCE(SUM(
		IF(usage_events.input_tokens >= usage_events.cached_input_tokens, usage_events.input_tokens - usage_events.cached_input_tokens, 0) * COALESCE(model_prices.input_price_per_million, 0) / 1000000 +
		usage_events.cached_input_tokens * COALESCE(model_prices.cached_input_price_per_million, model_prices.input_price_per_million, 0) / 1000000 +
		usage_events.output_tokens * COALESCE(model_prices.output_price_per_million, 0) / 1000000
	), 0)`)
	if err := costQuery.Scan(&result.EstimatedCostUSD).Error; err != nil {
		return Summary{}, fmt.Errorf("statistics estimated cost: %w", err)
	}
	return result, nil
}

// summaryBoundaries 把浏览器时区中的日、周一和月初边界转换为 UTC。
func summaryBoundaries(now time.Time, timezoneOffsetMinutes int) (time.Time, time.Time, time.Time) {
	location := time.FixedZone("browser", -timezoneOffsetMinutes*60)
	localNow := now.In(location)
	todayLocal := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	today := todayLocal.UTC()
	week := todayLocal.AddDate(0, 0, -((int(todayLocal.Weekday()) + 6) % 7)).UTC()
	month := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, location).UTC()
	return today, week, month
}

// Trend 按浏览器本地自然日聚合 Token，默认查询最近 30 天。
func (s *StatisticsService) Trend(userID uint64, filter StatisticsFilter) ([]TrendPoint, error) {
	if filter.StartTime == nil {
		start := time.Now().UTC().AddDate(0, 0, -29)
		filter.StartTime = &start
	}
	localMinutes := -filter.TimezoneOffsetMinutes
	dateExpression := fmt.Sprintf("DATE_FORMAT(DATE_ADD(occurred_at, INTERVAL %d MINUTE), '%%Y-%%m-%%d')", localMinutes)
	var rows []TrendPoint
	err := s.filtered(userID, filter).
		Select(dateExpression + " AS date, SUM(total_tokens) AS total_tokens, SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens, SUM(cached_input_tokens) AS cached_input_tokens, SUM(reasoning_tokens) AS reasoning_tokens").
		Group(dateExpression).Order("date ASC").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("statistics trend: %w", err)
	}
	return rows, nil
}

// DayDetail 返回指定筛选范围的详细用量；调用方通常传入一个本地自然日的 UTC 边界。
func (s *StatisticsService) DayDetail(userID uint64, filter StatisticsFilter) (DayDetail, error) {
	summary, err := s.Summary(userID, filter)
	if err != nil {
		return DayDetail{}, err
	}
	detail := DayDetail{Totals: summary.Totals, EstimatedCostUSD: summary.EstimatedCostUSD}
	var activity struct {
		Messages int64
		Sessions int64
		Events   int64
	}
	if err := s.filtered(userID, filter).Select(`
		COUNT(*) AS events,
		COUNT(DISTINCT NULLIF(message_id, '')) AS messages,
		COUNT(DISTINCT NULLIF(session_id, '')) AS sessions`).Scan(&activity).Error; err != nil {
		return DayDetail{}, fmt.Errorf("statistics day activity: %w", err)
	}
	detail.Messages, detail.Sessions, detail.Events = activity.Messages, activity.Sessions, activity.Events
	if detail.Sources, err = s.By(userID, filter, "source"); err != nil {
		return DayDetail{}, err
	}
	if detail.Models, err = s.By(userID, filter, "model"); err != nil {
		return DayDetail{}, err
	}
	if detail.Sources == nil {
		detail.Sources = []GroupTotal{}
	}
	if detail.Models == nil {
		detail.Models = []GroupTotal{}
	}
	return detail, nil
}

// By 按 device、source 或 model 聚合，并限制最多返回前 100 组。
func (s *StatisticsService) By(userID uint64, filter StatisticsFilter, group string) ([]GroupTotal, error) {
	var selectKey, groupBy string
	query := s.filtered(userID, filter)
	switch group {
	case "device":
		query = query.Joins("JOIN devices ON devices.id = usage_events.device_id")
		selectKey, groupBy = "devices.device_name", "devices.id, devices.device_name"
	case "source":
		selectKey, groupBy = "usage_events.source", "usage_events.source"
	case "model":
		selectKey, groupBy = "COALESCE(usage_events.model, 'unknown')", "usage_events.model"
	default:
		return nil, fmt.Errorf("unsupported group")
	}
	var rows []GroupTotal
	err := query.Select(selectKey + " AS `key`, SUM(total_tokens) AS total_tokens, SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens, SUM(cached_input_tokens) AS cached_input_tokens, SUM(reasoning_tokens) AS reasoning_tokens").
		Group(groupBy).Order("total_tokens DESC").Limit(100).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("statistics by %s: %w", group, err)
	}
	return rows, nil
}

// Recent 返回按发生时间倒序排列的最近用量事件，数量限制为 1 到 100。
func (s *StatisticsService) Recent(userID uint64, filter StatisticsFilter, limit int) ([]RecentEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []RecentEvent
	err := s.filtered(userID, filter).Joins("JOIN devices ON devices.id = usage_events.device_id").
		Select("usage_events.event_id, usage_events.source, usage_events.model, devices.device_name, usage_events.total_tokens, usage_events.occurred_at").
		Order("usage_events.occurred_at DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("recent usage: %w", err)
	}
	return rows, nil
}

// filtered 构造带用户隔离和通用筛选条件的 usage_events 基础查询。
func (s *StatisticsService) filtered(userID uint64, filter StatisticsFilter) *gorm.DB {
	query := s.store.Query().Table("usage_events").Where("usage_events.user_id = ?", userID)
	if filter.StartTime != nil {
		query = query.Where("usage_events.occurred_at >= ?", filter.StartTime.UTC())
	}
	if filter.EndTime != nil {
		query = query.Where("usage_events.occurred_at < ?", filter.EndTime.UTC())
	}
	if filter.DeviceID != nil {
		query = query.Where("usage_events.device_id = ?", *filter.DeviceID)
	}
	if filter.Source != "" {
		query = query.Where("usage_events.source = ?", filter.Source)
	}
	if filter.Model != "" {
		query = query.Where("usage_events.model = ?", filter.Model)
	}
	return query
}
