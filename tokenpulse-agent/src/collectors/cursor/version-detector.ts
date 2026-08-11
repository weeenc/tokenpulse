import type Database from 'better-sqlite3';

export type CursorFormat = 'cursor-disk-kv-v1' | 'unknown';

export function detectCursorFormat(database: Database.Database): CursorFormat {
  const row = database
    .prepare(
      "SELECT COUNT(*) AS count FROM sqlite_master WHERE type = 'table' AND name = 'cursorDiskKV'",
    )
    .get() as { count: number };
  return row.count === 1 ? 'cursor-disk-kv-v1' : 'unknown';
}
