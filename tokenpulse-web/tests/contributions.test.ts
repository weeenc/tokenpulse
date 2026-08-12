import { describe, expect, it } from 'vitest';
import {
  buildContributionCalendar,
  contributionDateRange,
  contributionLevel,
} from '../src/utils/contributions.js';

describe('contribution calendar', () => {
  it('builds a complete, week-aligned local calendar year', () => {
    const calendar = buildContributionCalendar([], new Date(2026, 7, 12));

    expect(calendar.days).toHaveLength(365);
    expect(calendar.days[0].date).toBe('2025-08-13');
    expect(calendar.days.at(-1)?.date).toBe('2026-08-12');
    expect(calendar.leadingDays).toBe(3);
    expect(calendar.trailingDays).toBe(3);
    expect(calendar.weeks).toBe(53);
  });

  it('fills missing days and calculates totals, levels and the peak day', () => {
    const calendar = buildContributionCalendar(
      [
        {
          date: '2026-08-10',
          totalTokens: 10,
          inputTokens: 7,
          outputTokens: 3,
          cachedInputTokens: 2,
          reasoningTokens: 1,
        },
        { date: '2026-08-12', totalTokens: 10_000 },
      ],
      new Date(2026, 7, 12),
    );

    expect(calendar.activeDays).toBe(2);
    expect(calendar.totalTokens).toBe(10_010);
    expect(calendar.peak).toMatchObject({ date: '2026-08-12', totalTokens: 10_000, level: 4 });
    expect(calendar.peak?.barHeight).toBe(74);
    expect(calendar.days.at(-2)?.totalTokens).toBe(0);
    expect(calendar.days.at(-2)?.barHeight).toBe(0);
    expect(calendar.days.at(-3)?.level).toBeGreaterThan(0);
    expect(calendar.days.at(-3)).toMatchObject({
      inputTokens: 7,
      outputTokens: 3,
      cachedInputTokens: 2,
      reasoningTokens: 1,
    });
  });

  it('uses relative intensity buckets and local date boundaries', () => {
    expect(contributionLevel(0, 1_000)).toBe(0);
    expect(contributionLevel(1_000, 1_000)).toBe(4);
    expect(contributionLevel(1, 1_000)).toBeGreaterThan(0);

    const [start, end] = contributionDateRange(new Date(2026, 7, 12));
    expect(start.getDate()).toBe(13);
    expect(end.getDate()).toBe(12);
  });
});
