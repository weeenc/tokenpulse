import Database from 'better-sqlite3';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CursorCollector } from '../src/collectors/cursor/index.js';
import { normalizeCursorRow } from '../src/collectors/cursor/normalizer.js';
import type { FileProgress } from '../src/types/usage.js';

const directories: string[] = [];

afterEach(async () => {
  await Promise.all(
    directories.splice(0).map((path) => rm(path, { recursive: true, force: true })),
  );
});

describe('Cursor collector', () => {
  it('normalizes only explicit, settled token counters without reading content', () => {
    const event = normalizeCursorRow(
      {
        bubbleKey: 'bubbleId:conversation-1:bubble-1',
        inputTokens: 800,
        outputTokens: 200,
        cachedInputTokens: 100,
        reasoningTokens: 25,
        model: 'gpt-5',
        createdAt: '2026-08-07T00:00:00Z',
        requestId: 'request-1',
      },
      Date.parse('2026-08-07T01:00:00Z'),
    );
    expect(event).toMatchObject({
      source: 'cursor',
      sessionId: 'conversation-1',
      messageId: 'request-1',
      inputTokens: 800,
      outputTokens: 200,
      totalTokens: 1000,
    });
    expect(event?.eventId).toMatch(/^[a-f0-9]{64}$/);
  });

  it('reads the verified cursorDiskKV tokenCount shape from a fixture database', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'tokenpulse-cursor-'));
    directories.push(directory);
    const path = join(directory, 'state.vscdb');
    const database = new Database(path);
    database.exec('CREATE TABLE cursorDiskKV (key TEXT UNIQUE, value BLOB)');
    database.prepare('INSERT INTO cursorDiskKV(key, value) VALUES (?, ?)').run(
      'bubbleId:conversation-2:bubble-2',
      JSON.stringify({
        tokenCount: { inputTokens: 1200, outputTokens: 300, cachedInputTokens: 400 },
        modelInfo: { modelName: 'claude-sonnet' },
        createdAt: '2026-08-07T00:00:00Z',
        requestId: 'request-2',
        text: 'private content that the query must never select',
      }),
    );
    database.close();

    let progress: FileProgress | null = null;
    const warnings: string[] = [];
    const events = await new CursorCollector(path).collect({
      getProgress: () => progress,
      saveProgress: (value) => {
        progress = value;
      },
      warn: (message) => warnings.push(message),
    });
    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({ inputTokens: 1200, outputTokens: 300 });
    expect(progress).toMatchObject({ path, offset: 1, metadata: { format: 'cursor-disk-kv-v1' } });
    expect(warnings).toEqual([]);
  });

  it('diagnoses databases that do not expose verified token records', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'tokenpulse-cursor-empty-'));
    directories.push(directory);
    const path = join(directory, 'state.vscdb');
    const database = new Database(path);
    database.exec('CREATE TABLE cursorDiskKV (key TEXT UNIQUE, value BLOB)');
    database.close();
    const warn = vi.fn();
    await expect(
      new CursorCollector(path).collect({
        getProgress: () => null,
        saveProgress: () => undefined,
        warn,
      }),
    ).resolves.toEqual([]);
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('no local usage records'));
  });
});
