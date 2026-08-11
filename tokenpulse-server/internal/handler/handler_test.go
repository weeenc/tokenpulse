// 本文件验证统计查询参数在 HTTP 边界的解析和拒绝规则。
package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// statisticsContext 创建带指定查询字符串的 Gin 测试上下文。
func statisticsContext(rawQuery string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/statistics?"+rawQuery, nil)
	return context, recorder
}

// TestStatisticsFilterValidatesRangeAndTimezone 覆盖时间格式、时区范围和有效区间。
func TestStatisticsFilterValidatesRangeAndTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, response := statisticsContext("startTime=invalid")
	if _, ok := statisticsFilter(context); ok || response.Code != 400 {
		t.Fatalf("invalid timestamp was accepted: status=%d", response.Code)
	}

	context, response = statisticsContext("timezoneOffsetMinutes=900")
	if _, ok := statisticsFilter(context); ok || response.Code != 400 {
		t.Fatalf("invalid timezone was accepted: status=%d", response.Code)
	}

	context, _ = statisticsContext("startTime=2026-08-01T00%3A00%3A00Z&endTime=2026-08-08T00%3A00%3A00Z&timezoneOffsetMinutes=-480")
	filter, ok := statisticsFilter(context)
	if !ok || filter.TimezoneOffsetMinutes != -480 || filter.StartTime == nil || filter.EndTime == nil {
		t.Fatalf("valid filter was rejected: %+v", filter)
	}
}

// TestValidTokenTotals 覆盖完整细分、仅总量和矛盾细分三类输入。
func TestValidTokenTotals(t *testing.T) {
	tests := []struct {
		name  string
		event usageEventRequest
		valid bool
	}{
		{name: "complete breakdown", event: usageEventRequest{InputTokens: 100, OutputTokens: 20, CachedInputTokens: 40, ReasoningTokens: 5, TotalTokens: 120}, valid: true},
		{name: "verified total only", event: usageEventRequest{TotalTokens: 12_178}, valid: true},
		{name: "inconsistent input and output", event: usageEventRequest{InputTokens: 100, OutputTokens: 20, TotalTokens: 130}, valid: false},
		{name: "inconsistent cached breakdown", event: usageEventRequest{CachedInputTokens: 10, TotalTokens: 100}, valid: false},
		{name: "inconsistent reasoning breakdown", event: usageEventRequest{ReasoningTokens: 10, TotalTokens: 100}, valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validTokenTotals(test.event); got != test.valid {
				t.Fatalf("validTokenTotals() = %v, want %v", got, test.valid)
			}
		})
	}
}
