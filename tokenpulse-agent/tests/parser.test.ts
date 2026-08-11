import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';
import { parseCodexLines } from '../src/collectors/codex/parser-jsonl.js';
import { parseClaudeLines } from '../src/collectors/claude-code/parser-jsonl.js';

async function fixture(path: string): Promise<string[]> {
  return (await readFile(new URL(`fixtures/${path}`, import.meta.url), 'utf8')).trim().split('\n');
}

describe('verified JSONL parsers', () => {
  it('normalizes Codex last-token usage without content fields', async () => {
    const result = parseCodexLines(await fixture('codex/session.jsonl'));
    expect(result.events).toHaveLength(1);
    expect(result.events[0]).toMatchObject({
      source: 'codex',
      model: 'gpt-5',
      sessionId: 'codex-session-1',
      inputTokens: 1200,
      outputTokens: 300,
      cachedInputTokens: 400,
      reasoningTokens: 50,
      totalTokens: 1500,
    });
    expect(result.events[0]?.eventId).toMatch(/^[a-f0-9]{64}$/);
  });

  it('uses Claude message id so streamed snapshots remain idempotent', async () => {
    const events = parseClaudeLines(await fixture('claude-code/session.jsonl'));
    expect(events).toHaveLength(2);
    expect(events[0]?.eventId).toBe(events[1]?.eventId);
    expect(events[0]).toMatchObject({
      source: 'claude-code',
      model: 'claude-sonnet',
      totalTokens: 1000,
    });
  });
});
