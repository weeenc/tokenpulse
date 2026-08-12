export interface ContributionPoint {
  date: string;
  totalTokens: number;
  inputTokens?: number;
  outputTokens?: number;
  cachedInputTokens?: number;
  reasoningTokens?: number;
}

export interface ContributionDay {
  date: string;
  totalTokens: number;
  inputTokens: number;
  outputTokens: number;
  cachedInputTokens: number;
  reasoningTokens: number;
  level: number;
  barHeight: number;
}

export interface ContributionMonth {
  label: string;
  week: number;
}

export interface ContributionCalendar {
  days: ContributionDay[];
  leadingDays: number;
  trailingDays: number;
  weeks: number;
  months: ContributionMonth[];
  activeDays: number;
  totalTokens: number;
  peak: ContributionDay | null;
}

const MONTH_LABELS = [
  '1月',
  '2月',
  '3月',
  '4月',
  '5月',
  '6月',
  '7月',
  '8月',
  '9月',
  '10月',
  '11月',
  '12月',
];

function startOfLocalDay(value: Date): Date {
  return new Date(value.getFullYear(), value.getMonth(), value.getDate());
}

function dateKey(value: Date): string {
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, '0');
  const day = String(value.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function addDays(value: Date, amount: number): Date {
  const result = new Date(value);
  result.setDate(result.getDate() + amount);
  return result;
}

export function contributionLevel(value: number, maximum: number): number {
  if (value <= 0 || maximum <= 0) return 0;
  const ratio = Math.sqrt(value / maximum);
  return Math.max(1, Math.min(4, Math.ceil(ratio * 4)));
}

export function buildContributionCalendar(
  points: ContributionPoint[],
  endDate = new Date(),
): ContributionCalendar {
  const end = startOfLocalDay(endDate);
  const start = addDays(end, -364);
  const values = new Map(points.map((point) => [point.date, point]));
  const totals = Array.from({ length: 365 }, (_, index) => {
    const point = values.get(dateKey(addDays(start, index)));
    return Math.max(0, point?.totalTokens ?? 0);
  });
  const maximum = Math.max(0, ...totals);
  const days = totals.map((totalTokens, index) => {
    const date = dateKey(addDays(start, index));
    const point = values.get(date);
    return {
      date,
      totalTokens,
      inputTokens: Math.max(0, point?.inputTokens ?? 0),
      outputTokens: Math.max(0, point?.outputTokens ?? 0),
      cachedInputTokens: Math.max(0, point?.cachedInputTokens ?? 0),
      reasoningTokens: Math.max(0, point?.reasoningTokens ?? 0),
      level: contributionLevel(totalTokens, maximum),
      barHeight:
        totalTokens > 0 && maximum > 0
          ? Math.max(5, Math.round(Math.pow(totalTokens / maximum, 0.45) * 74))
          : 0,
    };
  });
  const leadingDays = start.getDay();
  const trailingDays = 6 - end.getDay();
  const weeks = Math.ceil((leadingDays + days.length + trailingDays) / 7);
  const months: ContributionMonth[] = [];

  days.forEach((day, index) => {
    const date = addDays(start, index);
    if (index !== 0 && date.getDate() !== 1) return;
    const week = Math.floor((leadingDays + index) / 7);
    if (months.at(-1)?.week === week) return;
    months.push({ label: MONTH_LABELS[date.getMonth()], week });
  });

  const activeDays = days.filter((day) => day.totalTokens > 0).length;
  const totalTokens = days.reduce((total, day) => total + day.totalTokens, 0);
  const peak = maximum > 0 ? (days.find((day) => day.totalTokens === maximum) ?? null) : null;

  return {
    days,
    leadingDays,
    trailingDays,
    weeks,
    months,
    activeDays,
    totalTokens,
    peak,
  };
}

export function contributionDateRange(endDate = new Date()): [Date, Date] {
  const end = startOfLocalDay(endDate);
  return [addDays(end, -364), end];
}
