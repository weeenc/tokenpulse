import { describe, expect, it } from 'vitest';
import { deviceStatisticsParams, statisticsParams } from '../src/utils/statistics.js';

describe('statistics query parameters', () => {
  it('builds a deterministic preset range and passes the browser offset', () => {
    expect(statisticsParams(7, undefined, -480, Date.parse('2026-08-07T12:00:00Z'))).toEqual({
      startTime: '2026-08-01T12:00:00.000Z',
      timezoneOffsetMinutes: -480,
    });
  });

  it('uses an exclusive day boundary for a custom local date range', () => {
    const result = statisticsParams(
      30,
      [new Date(2026, 7, 1), new Date(2026, 7, 7)],
      new Date().getTimezoneOffset(),
    );
    expect(new Date(result.startTime).getDate()).toBe(1);
    expect(new Date(result.endTime ?? '').getDate()).toBe(8);
  });

  it('only sends a device filter after a device is selected', () => {
    expect(deviceStatisticsParams()).toEqual({});
    expect(deviceStatisticsParams(null)).toEqual({});
    expect(deviceStatisticsParams(42)).toEqual({ deviceId: 42 });
  });
});
