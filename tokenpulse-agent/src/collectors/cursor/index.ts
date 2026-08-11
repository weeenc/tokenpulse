import Database from 'better-sqlite3';
import { stat } from 'node:fs/promises';
import { sourcePaths } from '../../platform/paths.js';
import type { CollectContext, UsageCollector, UsageEvent } from '../../types/usage.js';
import { stableEventId } from '../common.js';
import { CursorSourceDetector } from './detector.js';
import { normalizeCursorRow, type CursorUsageRow } from './normalizer.js';
import { detectCursorFormat } from './version-detector.js';

type DatabaseRow = {
  row_id: number;
  bubble_key: string;
  input_tokens: number | null;
  output_tokens: number | null;
  cached_input_tokens: number | null;
  reasoning_tokens: number | null;
  model: string | null;
  created_at: string | number | null;
  request_id: string | null;
};

const OVERLAP_ROWS = 2_000;

export class CursorCollector implements UsageCollector {
  readonly name = 'cursor';
  private readonly detector: CursorSourceDetector;

  constructor(private readonly databasePath = sourcePaths().cursor) {
    this.detector = new CursorSourceDetector(databasePath);
  }

  detect(): Promise<boolean> {
    return this.detector.detect();
  }

  async collect(context: CollectContext): Promise<UsageEvent[]> {
    const file = await cursorDatabaseState(this.databasePath);
    const previous = context.getProgress(this.databasePath);
    if (previous && previous.size === file.size && previous.mtimeMs === file.mtimeMs) return [];

    const database = new Database(this.databasePath, { readonly: true, fileMustExist: true });
    try {
      database.pragma('query_only = ON');
      database.pragma('busy_timeout = 2000');
      const format = detectCursorFormat(database);
      if (format === 'unknown') {
        context.warn('Cursor database schema is not recognized; no data was read.');
        return [];
      }
      const startRow = Math.max(0, (previous?.offset ?? 0) - OVERLAP_ROWS);
      const rows = database
        .prepare(
          `SELECT ROWID AS row_id, key AS bubble_key,
             json_extract(CAST(value AS TEXT), '$.tokenCount.inputTokens') AS input_tokens,
             json_extract(CAST(value AS TEXT), '$.tokenCount.outputTokens') AS output_tokens,
             json_extract(CAST(value AS TEXT), '$.tokenCount.cachedInputTokens') AS cached_input_tokens,
             json_extract(CAST(value AS TEXT), '$.tokenCount.reasoningTokens') AS reasoning_tokens,
             json_extract(CAST(value AS TEXT), '$.modelInfo.modelName') AS model,
             json_extract(CAST(value AS TEXT), '$.createdAt') AS created_at,
             json_extract(CAST(value AS TEXT), '$.requestId') AS request_id
           FROM cursorDiskKV
           WHERE ROWID > ? AND key LIKE 'bubbleId:%' AND json_valid(CAST(value AS TEXT))
           ORDER BY ROWID ASC`,
        )
        .all(startRow) as DatabaseRow[];
      const events = rows
        .map((row) =>
          normalizeCursorRow({
            bubbleKey: row.bubble_key,
            inputTokens: row.input_tokens,
            outputTokens: row.output_tokens,
            cachedInputTokens: row.cached_input_tokens,
            reasoningTokens: row.reasoning_tokens,
            model: row.model,
            createdAt: row.created_at,
            requestId: row.request_id,
          } satisfies CursorUsageRow),
        )
        .filter((event): event is UsageEvent => event !== null);
      const lastRow = rows.at(-1)?.row_id ?? previous?.offset ?? 0;
      context.saveProgress({
        path: this.databasePath,
        size: file.size,
        mtimeMs: file.mtimeMs,
        offset: lastRow,
        fingerprint: stableEventId([format, file.size, file.mtimeMs]),
        metadata: { format },
      });
      if (events.length === 0) {
        context.warn(
          rows.length === 0
            ? 'Cursor was detected, but no local usage records with verified token fields were found.'
            : 'Cursor records were found, but none contained settled, non-zero token counts.',
        );
      }
      return events;
    } catch (error) {
      context.warn(`Cursor database could not be parsed safely: ${String(error)}`);
      return [];
    } finally {
      database.close();
    }
  }
}

async function cursorDatabaseState(path: string): Promise<{ size: number; mtimeMs: number }> {
  const main = await stat(path);
  try {
    const wal = await stat(`${path}-wal`);
    return { size: main.size + wal.size, mtimeMs: Math.max(main.mtimeMs, wal.mtimeMs) };
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== 'ENOENT') throw error;
    return { size: main.size, mtimeMs: main.mtimeMs };
  }
}
