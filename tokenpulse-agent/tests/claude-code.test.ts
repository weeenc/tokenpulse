import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import { ClaudeCodeCollector } from '../src/collectors/claude-code/index.js';
import type { FileProgress } from '../src/types/usage.js';

const directories: string[] = [];

afterEach(async () => {
  await Promise.all(
    directories.splice(0).map((path) => rm(path, { recursive: true, force: true })),
  );
});

describe('Claude Code collector', () => {
  it('discovers Claude Desktop project logs without scanning audit logs', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'tokenpulse-claude-desktop-'));
    directories.push(directory);

    const desktopRoot = join(directory, 'local-agent-mode-sessions');
    const projectRoot = join(
      desktopRoot,
      'account',
      '00000000',
      'session',
      '.claude',
      'projects',
      'session',
    );
    await mkdir(projectRoot, { recursive: true });
    await writeFile(
      join(projectRoot, 'session.jsonl'),
      `${JSON.stringify({
        type: 'assistant',
        sessionId: 'desktop-session-1',
        timestamp: '2026-09-29T01:12:09.582Z',
        message: {
          id: 'desktop-message-1',
          model: 'claude-sonnet-5',
          usage: {
            input_tokens: 1200,
            output_tokens: 300,
            cache_read_input_tokens: 400,
          },
        },
      })}\n`,
    );
    await writeFile(
      join(desktopRoot, 'account', '00000000', 'session', 'audit.jsonl'),
      `${JSON.stringify({
        type: 'assistant',
        sessionId: 'audit-session-that-must-not-be-read',
        timestamp: '2026-09-29T01:12:09.582Z',
        message: {
          id: 'audit-message-1',
          model: 'claude-sonnet-5',
          usage: { input_tokens: 9999, output_tokens: 9999 },
        },
      })}\n`,
    );

    let progress: FileProgress | null = null;
    const warnings: string[] = [];
    const collector = new ClaudeCodeCollector(join(directory, 'missing-cli-root'), [desktopRoot]);

    await expect(collector.detect()).resolves.toBe(true);
    const events = await collector.collect({
      getProgress: (path) => (progress?.path === path ? progress : null),
      saveProgress: (value) => {
        progress = value;
      },
      warn: (message) => warnings.push(message),
    });

    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({
      source: 'claude-code',
      sessionId: 'desktop-session-1',
      inputTokens: 1200,
      outputTokens: 300,
      cachedInputTokens: 400,
      totalTokens: 1500,
    });
    expect(warnings).toEqual([]);
    expect(progress?.path).toBe(join(projectRoot, 'session.jsonl'));
  });
});
