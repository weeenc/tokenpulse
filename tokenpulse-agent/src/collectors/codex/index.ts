import { access } from 'node:fs/promises';
import { sourcePaths } from '../../platform/paths.js';
import type { CollectContext, UsageCollector, UsageEvent } from '../../types/usage.js';
import { jsonlFiles, readIncrementalLines } from '../common.js';
import { parseCodexLines, type CodexParserState } from './parser-jsonl.js';
import { recognizesCodexJsonl } from './version-detector.js';

export class CodexCollector implements UsageCollector {
  readonly name = 'codex';
  constructor(private readonly root = sourcePaths().codex) {}
  async detect(): Promise<boolean> {
    try {
      await access(this.root);
      return true;
    } catch {
      return false;
    }
  }
  async collect(context: CollectContext): Promise<UsageEvent[]> {
    const events: UsageEvent[] = [];
    for (const path of await jsonlFiles(this.root)) {
      try {
        const { lines, progress } = await readIncrementalLines(path, context);
        if (lines.length > 0 && !recognizesCodexJsonl(lines)) {
          context.warn(`Codex file ${path} uses an unrecognized JSONL format; skipped.`);
          context.saveProgress(progress);
          continue;
        }
        const parsed = parseCodexLines(
          lines,
          progress.metadata as CodexParserState | undefined,
          context.warn,
        );
        progress.metadata = {
          ...(parsed.state.sessionId ? { sessionId: parsed.state.sessionId } : {}),
          ...(parsed.state.model ? { model: parsed.state.model } : {}),
        };
        context.saveProgress(progress);
        events.push(...parsed.events);
      } catch (error) {
        context.warn(`Codex file ${path} could not be parsed: ${String(error)}`);
      }
    }
    return events;
  }
}
