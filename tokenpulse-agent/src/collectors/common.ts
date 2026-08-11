import { createHash } from 'node:crypto';
import { open, readdir, stat } from 'node:fs/promises';
import { join } from 'node:path';
import type { CollectContext, FileProgress } from '../types/usage.js';

export function stableEventId(parts: Array<string | number | undefined>): string {
  return createHash('sha256')
    .update(parts.map((value) => value ?? '').join('\u001f'))
    .digest('hex');
}

export async function jsonlFiles(root: string): Promise<string[]> {
  const result: string[] = [];
  async function walk(path: string): Promise<void> {
    let entries;
    try {
      entries = await readdir(path, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      const full = join(path, entry.name);
      if (entry.isDirectory()) await walk(full);
      else if (entry.isFile() && entry.name.endsWith('.jsonl')) result.push(full);
    }
  }
  await walk(root);
  return result.sort();
}

export async function readIncrementalLines(
  path: string,
  context: CollectContext,
): Promise<{ lines: string[]; progress: FileProgress }> {
  const fileStat = await stat(path);
  const previous = context.getProgress(path);
  const handle = await open(path, 'r');
  try {
    const head = Buffer.alloc(Math.min(fileStat.size, 4096));
    if (head.length > 0) await handle.read(head, 0, head.length, 0);
    const firstNewline = head.indexOf(0x0a);
    const stableHead = firstNewline >= 0 ? head.subarray(0, firstNewline + 1) : head;
    const fingerprint = createHash('sha256').update(stableHead).digest('hex');
    const sameFile = Boolean(
      previous &&
      previous.size <= fileStat.size &&
      previous.fingerprint === fingerprint &&
      previous.offset <= fileStat.size,
    );
    const offset = sameFile ? (previous?.offset ?? 0) : 0;
    const length = fileStat.size - offset;
    if (length <= 0)
      return {
        lines: [],
        progress: {
          path,
          size: fileStat.size,
          mtimeMs: fileStat.mtimeMs,
          offset,
          fingerprint,
          ...(sameFile && previous?.metadata ? { metadata: previous.metadata } : {}),
        },
      };
    const buffer = Buffer.alloc(length);
    await handle.read(buffer, 0, length, offset);
    const lastNewline = buffer.lastIndexOf(0x0a);
    if (lastNewline < 0)
      return {
        lines: [],
        progress: {
          path,
          size: fileStat.size,
          mtimeMs: fileStat.mtimeMs,
          offset,
          fingerprint,
          ...(sameFile && previous?.metadata ? { metadata: previous.metadata } : {}),
        },
      };
    const complete = buffer.subarray(0, lastNewline + 1);
    return {
      lines: complete.toString('utf8').split('\n').filter(Boolean),
      progress: {
        path,
        size: fileStat.size,
        mtimeMs: fileStat.mtimeMs,
        offset: offset + complete.length,
        fingerprint,
        ...(sameFile && previous?.metadata ? { metadata: { ...previous.metadata } } : {}),
      },
    };
  } finally {
    await handle.close();
  }
}

export function record(value: unknown): Record<string, unknown> | null {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;
}
export function stringValue(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined;
}
export function numberValue(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 ? Math.trunc(value) : 0;
}
