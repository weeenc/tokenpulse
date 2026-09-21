export function formatTokens(value: number): string {
  return Intl.NumberFormat('zh-CN').format(value);
}

function formatLocalDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export interface TokenPeriodLabels {
  today: string;
  week: string;
  month: string;
}

export function tokenPeriodLabels(now = new Date()): TokenPeriodLabels {
  const today = new Date(now);
  today.setHours(0, 0, 0, 0);

  const weekStart = new Date(today);
  const daysSinceMonday = (today.getDay() + 6) % 7;
  weekStart.setDate(today.getDate() - daysSinceMonday);
  const weekEnd = new Date(weekStart);
  weekEnd.setDate(weekStart.getDate() + 6);

  const monthStart = new Date(today.getFullYear(), today.getMonth(), 1);
  const monthEnd = new Date(today.getFullYear(), today.getMonth() + 1, 0);

  return {
    today: formatLocalDate(today),
    week: `${formatLocalDate(weekStart)}~${formatLocalDate(weekEnd)}`,
    month: `${formatLocalDate(monthStart)}~${formatLocalDate(monthEnd)}`,
  };
}

export function relativeTime(value: string, now = Date.now()): string {
  const timestamp = new Date(value).getTime();
  if (!Number.isFinite(timestamp)) return '未知时间';
  const minutes = Math.max(0, Math.round((now - timestamp) / 60_000));
  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes} 分钟前`;
  if (minutes < 1_440) return `${Math.floor(minutes / 60)} 小时前`;
  return new Date(timestamp).toLocaleDateString('zh-CN');
}

export function formatDateTime(value?: string): string {
  return value ? new Date(value).toLocaleString('zh-CN') : '尚未同步';
}
