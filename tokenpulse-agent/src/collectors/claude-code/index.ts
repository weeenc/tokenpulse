import { access } from 'node:fs/promises';
import { sourcePaths } from '../../platform/paths.js';
import type { CollectContext, UsageCollector, UsageEvent } from '../../types/usage.js';
import { jsonlFiles, readIncrementalLines } from '../common.js';
import { parseClaudeLines } from './parser-jsonl.js';
import { recognizesClaudeJsonl } from './version-detector.js';

export class ClaudeCodeCollector implements UsageCollector {
  readonly name = 'claude-code';
  constructor(private readonly root = sourcePaths().claude) {}
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
        if (lines.length > 0 && !recognizesClaudeJsonl(lines)) {
          context.warn(`Claude Code file ${path} uses an unrecognized JSONL format; skipped.`);
          context.saveProgress(progress);
          continue;
        }
        events.push(...parseClaudeLines(lines, context.warn));
        context.saveProgress(progress);
      } catch (error) {
        context.warn(`Claude Code file ${path} could not be parsed: ${String(error)}`);
      }
    }
    return events;
  }
}
