import type { UsageEvent } from '../../types/usage.js';
import { numberValue, record, stableEventId, stringValue } from '../common.js';

export interface CodexParserState {
  sessionId?: string;
  model?: string;
}

export function parseCodexLines(
  lines: string[],
  initial: CodexParserState = {},
  warn: (message: string) => void = () => {},
): { events: UsageEvent[]; state: CodexParserState } {
  const state = { ...initial };
  const events: UsageEvent[] = [];
  for (const line of lines) {
    let root: Record<string, unknown> | null;
    try {
      root = record(JSON.parse(line));
    } catch {
      warn('Codex JSONL contains an invalid line; skipped.');
      continue;
    }
    if (!root) continue;
    const payload = record(root.payload);
    if (root.type === 'session_meta' && payload) {
      const sessionId = stringValue(payload.id) ?? stringValue(payload.session_id);
      if (sessionId) state.sessionId = sessionId;
    }
    if (root.type === 'turn_context' && payload) {
      const model = stringValue(payload.model);
      if (model) state.model = model;
    }
    if (root.type !== 'event_msg' || payload?.type !== 'token_count') continue;
    const usage = record(record(payload.info)?.last_token_usage);
    const occurredAt = stringValue(root.timestamp);
    if (!usage || !occurredAt || !state.sessionId) {
      warn('Codex token record is missing usage, timestamp, or session id; skipped.');
      continue;
    }
    const input = numberValue(usage.input_tokens);
    const output = numberValue(usage.output_tokens);
    const cached = numberValue(usage.cached_input_tokens);
    const reasoning = numberValue(usage.reasoning_output_tokens);
    const total = numberValue(usage.total_tokens) || input + output;
    const event: UsageEvent = {
      eventId: stableEventId(['codex', state.sessionId, occurredAt, state.model, input, output]),
      source: 'codex',
      sessionId: state.sessionId,
      inputTokens: input,
      outputTokens: output,
      cachedInputTokens: cached,
      reasoningTokens: reasoning,
      totalTokens: total,
      occurredAt,
      ...(state.model ? { model: state.model } : {}),
    };
    events.push(event);
  }
  return { events, state };
}
