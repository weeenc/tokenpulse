// 本文件验证统计自然周期边界的时区转换规则。
package service

import (
	"testing"
	"time"
)

// TestSummaryBoundariesUseBrowserTimezone 验证上海时区的日、周、月起点均转换为正确 UTC。
func TestSummaryBoundariesUseBrowserTimezone(t *testing.T) {
	now := time.Date(2026, 8, 6, 17, 30, 0, 0, time.UTC) // 2026-08-07 01:30 in Shanghai.
	today, week, month := summaryBoundaries(now, -480)
	if !today.Equal(time.Date(2026, 8, 6, 16, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected local day boundary: %s", today)
	}
	if !week.Equal(time.Date(2026, 8, 2, 16, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected local week boundary: %s", week)
	}
	if !month.Equal(time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected local month boundary: %s", month)
	}
}
