export interface StatisticsParams {
  startTime: string;
  endTime?: string;
  timezoneOffsetMinutes: number;
}

export interface DeviceStatisticsParams {
  deviceId?: number;
}

export function deviceStatisticsParams(deviceId?: number | null): DeviceStatisticsParams {
  return deviceId == null ? {} : { deviceId };
}

export function statisticsParams(
  days: number,
  range: [Date, Date] | undefined,
  timezoneOffsetMinutes: number,
  now = Date.now(),
): StatisticsParams {
  if (!range) {
    return {
      startTime: new Date(now - (days - 1) * 86_400_000).toISOString(),
      timezoneOffsetMinutes,
    };
  }
  const [start, end] = range;
  const exclusiveEnd = new Date(end);
  exclusiveEnd.setDate(exclusiveEnd.getDate() + 1);
  exclusiveEnd.setHours(0, 0, 0, 0);
  return {
    startTime: start.toISOString(),
    endTime: exclusiveEnd.toISOString(),
    timezoneOffsetMinutes,
  };
}
