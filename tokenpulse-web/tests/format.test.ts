import { describe, expect, it } from 'vitest';
import { formatTokens, relativeTime, tokenPeriodLabels } from '../src/utils/format.js';

describe('display formatting', () => {
  it('formats token counts', () => {
    expect(formatTokens(1_250_000)).toMatch(/1[,，]250[,，]000/);
  });

  it('formats today, week, and month token periods from the local date', () => {
    expect(tokenPeriodLabels(new Date(2026, 8, 21))).toEqual({
      today: '2026-09-21',
      week: '2026-09-21~2026-09-27',
      month: '2026-09-01~2026-09-30',
    });
  });

  it('starts the weekly period on Monday for dates later in the week', () => {
    expect(tokenPeriodLabels(new Date(2026, 8, 23)).week).toBe('2026-09-21~2026-09-27');
  });

  it('formats relative timestamps deterministically', () => {
    const now = Date.parse('2026-08-07T03:00:00Z');
    expect(relativeTime('2026-08-07T02:55:00Z', now)).toBe('5 分钟前');
    expect(relativeTime('2026-08-07T01:00:00Z', now)).toBe('2 小时前');
    expect(relativeTime('invalid', now)).toBe('未知时间');
  });
});
