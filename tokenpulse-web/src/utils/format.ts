export function formatTokens(value: number): string {
  return Intl.NumberFormat('zh-CN').format(value);
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
