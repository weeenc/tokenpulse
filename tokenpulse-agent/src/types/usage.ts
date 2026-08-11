export interface UsageEvent {
  eventId: string;
  source: string;
  model?: string;
  sessionId?: string;
  messageId?: string;
  inputTokens: number;
  outputTokens: number;
  cachedInputTokens: number;
  reasoningTokens: number;
  totalTokens: number;
  occurredAt: string;
}

export interface CollectContext {
  getProgress(path: string): FileProgress | null;
  saveProgress(progress: FileProgress): void;
  warn(message: string): void;
}

export interface FileProgress {
  path: string;
  size: number;
  mtimeMs: number;
  offset: number;
  fingerprint: string;
  metadata?: Record<string, string>;
}

export interface UsageCollector {
  readonly name: string;
  detect(): Promise<boolean>;
  collect(context: CollectContext): Promise<UsageEvent[]>;
}
