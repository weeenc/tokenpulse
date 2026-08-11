import type { UsageEvent } from '../../types/usage.js';
import { numberValue, record, stableEventId, stringValue } from '../common.js';

export function parseClaudeLines(
  lines: string[],
  warn: (message: string) => void = () => {},
): UsageEvent[] {
  const events: UsageEvent[] = [];
  for (const line of lines) {
    let root: Record<string, unknown> | null;
    try {
      root = record(JSON.parse(line));
    } catch {
      warn('Claude Code JSONL contains an invalid line; skipped.');
      continue;
    }
    const message = record(root?.message);
    const usage = record(message?.usage);
    if (!root || root.type !== 'assistant' || !message || !usage) continue;
    const sessionId = stringValue(root.sessionId);
    const messageId = stringValue(message.id) ?? stringValue(root.uuid);
    const occurredAt = stringValue(root.timestamp);
    if (!sessionId || !messageId || !occurredAt) {
      warn('Claude Code usage record is missing stable identifiers; skipped.');
      continue;
    }
    const input = numberValue(usage.input_tokens);
    const output = numberValue(usage.output_tokens);
    const cached = numberValue(usage.cache_read_input_tokens);
    const model = stringValue(message.model);
    events.push({
      eventId: stableEventId(['claude-code', sessionId, messageId]),
      source: 'claude-code',
      sessionId,
      messageId,
      ...(model ? { model } : {}),
      inputTokens: input,
      outputTokens: output,
      cachedInputTokens: cached,
      reasoningTokens: 0,
      totalTokens: input + output,
      occurredAt,
    });
  }
  return events;
}
