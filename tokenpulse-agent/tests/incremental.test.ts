import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import { readIncrementalLines } from '../src/collectors/common.js';
import type { FileProgress } from '../src/types/usage.js';

const directories: string[] = [];

afterEach(async () => {
  await Promise.all(
    directories.splice(0).map((path) => rm(path, { recursive: true, force: true })),
  );
});

describe('incremental JSONL reader', () => {
  it('reads appended lines and resets when a file is replaced at the same path', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'tokenpulse-incremental-'));
    directories.push(directory);
    const path = join(directory, 'session.jsonl');
    let progress: FileProgress | null = null;
    const context = {
      getProgress: () => progress,
      saveProgress: () => undefined,
      warn: () => undefined,
    };

    await writeFile(path, '{"id":1}\n');
    const first = await readIncrementalLines(path, context);
    progress = first.progress;
    expect(first.lines).toEqual(['{"id":1}']);

    await writeFile(path, '{"id":1}\n{"id":2}\n');
    const appended = await readIncrementalLines(path, context);
    progress = appended.progress;
    expect(appended.lines).toEqual(['{"id":2}']);

    await writeFile(path, '{"replacement":true}\n{"id":3}\n');
    const replaced = await readIncrementalLines(path, context);
    expect(replaced.lines).toEqual(['{"replacement":true}', '{"id":3}']);
  });
});
