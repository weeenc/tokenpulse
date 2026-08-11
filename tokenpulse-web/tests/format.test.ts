import { describe, expect, it } from 'vitest';
import { formatTokens, relativeTime } from '../src/utils/format.js';

describe('display formatting', () => {
  it('formats token counts', () => {
    expect(formatTokens(1_250_000)).toMatch(/1[,，]250[,，]000/);
  });

  it('formats relative timestamps deterministically', () => {
    const now = Date.parse('2026-08-07T03:00:00Z');
    expect(relativeTime('2026-08-07T02:55:00Z', now)).toBe('5 分钟前');
    expect(relativeTime('2026-08-07T01:00:00Z', now)).toBe('2 小时前');
    expect(relativeTime('invalid', now)).toBe('未知时间');
  });
});
