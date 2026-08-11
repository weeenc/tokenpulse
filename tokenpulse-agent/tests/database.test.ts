import { mkdtemp, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { afterEach, describe, expect, it } from 'vitest';
import { LocalDatabase } from '../src/storage/database.js';
import type { UsageEvent } from '../src/types/usage.js';

const dirs: string[] = [];
afterEach(async () =>
  Promise.all(dirs.splice(0).map((path) => rm(path, { recursive: true, force: true }))),
);

describe('local event storage', () => {
  it('deduplicates stable event ids and updates sync state', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'tokenpulse-'));
    dirs.push(dir);
    const db = new LocalDatabase(join(dir, 'data.db'));
    const event: UsageEvent = {
      eventId: 'a'.repeat(64),
      source: 'codex',
      inputTokens: 10,
      outputTokens: 2,
      cachedInputTokens: 3,
      reasoningTokens: 1,
      totalTokens: 12,
      occurredAt: '2026-08-07T00:00:00Z',
    };
    expect(db.addEvents([event, event])).toBe(1);
    expect(db.pending()).toHaveLength(1);
    db.markSynced([event.eventId]);
    expect(db.countPending()).toBe(0);
    db.close();
  });

  it('commits collected events and scan progress atomically', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'tokenpulse-atomic-'));
    dirs.push(dir);
    const db = new LocalDatabase(join(dir, 'data.db'));
    const event: UsageEvent = {
      eventId: 'b'.repeat(64),
      source: 'claude-code',
      inputTokens: 8,
      outputTokens: 2,
      cachedInputTokens: 0,
      reasoningTokens: 0,
      totalTokens: 10,
      occurredAt: '2026-08-07T00:00:00Z',
    };
    expect(
      db.addCollected(
        [event],
        [
          {
            path: '/fixture/session.jsonl',
            size: 100,
            mtimeMs: 1,
            offset: 100,
            fingerprint: 'fixture',
          },
        ],
      ),
    ).toBe(1);
    expect(db.pending()).toHaveLength(1);
    expect(db.getProgress('/fixture/session.jsonl')?.offset).toBe(100);
    db.close();
  });
});
