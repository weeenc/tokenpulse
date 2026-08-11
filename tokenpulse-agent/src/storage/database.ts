import Database from 'better-sqlite3';
import { mkdirSync } from 'node:fs';
import { dirname } from 'node:path';
import type { FileProgress, UsageEvent } from '../types/usage.js';

type EventRow = {
  event_id: string;
  source: string;
  model: string | null;
  session_id: string | null;
  message_id: string | null;
  input_tokens: number;
  output_tokens: number;
  cached_input_tokens: number;
  reasoning_tokens: number;
  total_tokens: number;
  occurred_at: string;
};

export class LocalDatabase {
  private readonly db: Database.Database;

  constructor(path: string) {
    mkdirSync(dirname(path), { recursive: true, mode: 0o700 });
    this.db = new Database(path);
    this.db.pragma('journal_mode = WAL');
    this.db.pragma('busy_timeout = 5000');
    this.migrate();
  }

  close(): void {
    this.db.close();
  }

  addEvents(events: UsageEvent[]): number {
    return this.insertEvents(events);
  }

  addCollected(events: UsageEvent[], progresses: FileProgress[]): number {
    const transaction = this.db.transaction(() => {
      const inserted = this.insertEvents(events);
      for (const progress of progresses) this.upsertProgress(progress);
      return inserted;
    });
    return transaction();
  }

  private insertEvents(events: UsageEvent[]): number {
    const insert = this.db.prepare(`INSERT OR IGNORE INTO usage_events
      (event_id, source, model, session_id, message_id, input_tokens, output_tokens, cached_input_tokens, reasoning_tokens, total_tokens, occurred_at, sync_status, retry_count, created_at)
      VALUES (@eventId, @source, @model, @sessionId, @messageId, @inputTokens, @outputTokens, @cachedInputTokens, @reasoningTokens, @totalTokens, @occurredAt, 'PENDING', 0, @createdAt)`);
    const now = new Date().toISOString();
    return events.reduce(
      (count, event) =>
        count +
        Number(
          insert.run({
            ...event,
            model: event.model ?? null,
            sessionId: event.sessionId ?? null,
            messageId: event.messageId ?? null,
            createdAt: now,
          }).changes,
        ),
      0,
    );
  }

  pending(limit = 500): UsageEvent[] {
    const rows = this.db
      .prepare(
        "SELECT event_id, source, model, session_id, message_id, input_tokens, output_tokens, cached_input_tokens, reasoning_tokens, total_tokens, occurred_at FROM usage_events WHERE sync_status IN ('PENDING','FAILED') ORDER BY occurred_at LIMIT ?",
      )
      .all(limit) as EventRow[];
    return rows.map(rowToEvent);
  }

  markSynced(eventIds: string[]): void {
    const update = this.db.prepare(
      "UPDATE usage_events SET sync_status = 'SYNCED' WHERE event_id = ?",
    );
    this.db.transaction((ids: string[]) => ids.forEach((id) => update.run(id)))(eventIds);
  }

  markFailed(eventIds: string[]): void {
    const update = this.db.prepare(
      "UPDATE usage_events SET sync_status = 'FAILED', retry_count = retry_count + 1 WHERE event_id = ?",
    );
    this.db.transaction((ids: string[]) => ids.forEach((id) => update.run(id)))(eventIds);
  }

  countPending(): number {
    return (
      this.db
        .prepare("SELECT COUNT(*) AS count FROM usage_events WHERE sync_status != 'SYNCED'")
        .get() as { count: number }
    ).count;
  }

  getProgress(path: string): FileProgress | null {
    const row = this.db
      .prepare(
        'SELECT path, size, mtime_ms, offset, fingerprint, metadata FROM file_progress WHERE path = ?',
      )
      .get(path) as
      | {
          path: string;
          size: number;
          mtime_ms: number;
          offset: number;
          fingerprint: string;
          metadata: string | null;
        }
      | undefined;
    if (!row) return null;
    return {
      path: row.path,
      size: row.size,
      mtimeMs: row.mtime_ms,
      offset: row.offset,
      fingerprint: row.fingerprint,
      ...(row.metadata ? { metadata: JSON.parse(row.metadata) as Record<string, string> } : {}),
    };
  }

  saveProgress(progress: FileProgress): void {
    this.upsertProgress(progress);
  }

  private upsertProgress(progress: FileProgress): void {
    this.db
      .prepare(
        `INSERT INTO file_progress(path,size,mtime_ms,offset,fingerprint,metadata,updated_at)
      VALUES(?,?,?,?,?,?,?) ON CONFLICT(path) DO UPDATE SET size=excluded.size,mtime_ms=excluded.mtime_ms,offset=excluded.offset,fingerprint=excluded.fingerprint,metadata=excluded.metadata,updated_at=excluded.updated_at`,
      )
      .run(
        progress.path,
        progress.size,
        progress.mtimeMs,
        progress.offset,
        progress.fingerprint,
        progress.metadata ? JSON.stringify(progress.metadata) : null,
        new Date().toISOString(),
      );
  }

  private migrate(): void {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS usage_events (
        event_id TEXT PRIMARY KEY, source TEXT NOT NULL, model TEXT, session_id TEXT, message_id TEXT,
        input_tokens INTEGER NOT NULL, output_tokens INTEGER NOT NULL, cached_input_tokens INTEGER NOT NULL,
        reasoning_tokens INTEGER NOT NULL, total_tokens INTEGER NOT NULL, occurred_at TEXT NOT NULL,
        sync_status TEXT NOT NULL, retry_count INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL
      );
      CREATE INDEX IF NOT EXISTS idx_usage_sync ON usage_events(sync_status, occurred_at);
      CREATE TABLE IF NOT EXISTS file_progress (
        path TEXT PRIMARY KEY, size INTEGER NOT NULL, mtime_ms REAL NOT NULL, offset INTEGER NOT NULL,
        fingerprint TEXT NOT NULL, metadata TEXT, updated_at TEXT NOT NULL
      );
    `);
  }
}

function rowToEvent(row: EventRow): UsageEvent {
  return {
    eventId: row.event_id,
    source: row.source,
    ...(row.model ? { model: row.model } : {}),
    ...(row.session_id ? { sessionId: row.session_id } : {}),
    ...(row.message_id ? { messageId: row.message_id } : {}),
    inputTokens: row.input_tokens,
    outputTokens: row.output_tokens,
    cachedInputTokens: row.cached_input_tokens,
    reasoningTokens: row.reasoning_tokens,
    totalTokens: row.total_tokens,
    occurredAt: row.occurred_at,
  };
}
