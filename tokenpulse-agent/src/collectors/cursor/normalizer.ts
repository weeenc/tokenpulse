import type { UsageEvent } from '../../types/usage.js';
import { stableEventId } from '../common.js';

export interface CursorUsageRow {
  bubbleKey: string;
  inputTokens: number | null;
  outputTokens: number | null;
  cachedInputTokens: number | null;
  reasoningTokens: number | null;
  model: string | null;
  createdAt: string | number | null;
  requestId: string | null;
}

const SETTLE_TIME_MS = 2 * 60 * 1000;

export function normalizeCursorRow(row: CursorUsageRow, now = Date.now()): UsageEvent | null {
  const input = tokenCount(row.inputTokens);
  const output = tokenCount(row.outputTokens);
  if (input + output === 0) return null;
  const occurredAt = cursorTimestamp(row.createdAt);
  if (!occurredAt || now - Date.parse(occurredAt) < SETTLE_TIME_MS) return null;
  const parts = row.bubbleKey.split(':');
  const sessionId = parts.length >= 3 ? parts[1] : undefined;
  const messageId = row.requestId || parts.at(-1) || row.bubbleKey;
  return {
    eventId: stableEventId(['cursor', sessionId, messageId]),
    source: 'cursor',
    ...(row.model ? { model: row.model } : {}),
    ...(sessionId ? { sessionId } : {}),
    messageId,
    inputTokens: input,
    outputTokens: output,
    cachedInputTokens: tokenCount(row.cachedInputTokens),
    reasoningTokens: tokenCount(row.reasoningTokens),
    totalTokens: input + output,
    occurredAt,
  };
}

function tokenCount(value: number | null): number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 ? Math.trunc(value) : 0;
}

function cursorTimestamp(value: string | number | null): string | null {
  if (value === null || value === '') return null;
  const numeric = typeof value === 'number' ? value : /^\d+$/.test(value) ? Number(value) : NaN;
  const date = new Date(Number.isFinite(numeric) ? numeric : value);
  return Number.isFinite(date.getTime()) ? date.toISOString() : null;
}
