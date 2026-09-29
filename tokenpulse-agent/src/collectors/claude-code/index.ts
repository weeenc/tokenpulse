import { access, readdir } from 'node:fs/promises';
import { basename, join } from 'node:path';
import { sourcePaths } from '../../platform/paths.js';
import type { CollectContext, UsageCollector, UsageEvent } from '../../types/usage.js';
import { jsonlFiles, readIncrementalLines } from '../common.js';
import { parseClaudeLines } from './parser-jsonl.js';
import { recognizesClaudeJsonl } from './version-detector.js';

export class ClaudeCodeCollector implements UsageCollector {
  readonly name = 'claude-code';
  constructor(
    private readonly root = sourcePaths().claude,
    private readonly desktopRoots = sourcePaths().claudeDesktop,
  ) {}
  async detect(): Promise<boolean> {
    try {
      await access(this.root);
      return true;
    } catch {
      return (await this.projectRoots()).length > 1;
    }
  }
  async collect(context: CollectContext): Promise<UsageEvent[]> {
    const events: UsageEvent[] = [];
    for (const root of await this.projectRoots()) {
      for (const path of await jsonlFiles(root)) {
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
    }
    return events;
  }

  private async projectRoots(): Promise<string[]> {
    const roots = new Set([this.root]);
    for (const desktopRoot of this.desktopRoots) {
      for (const projectRoot of await findProjectRoots(desktopRoot)) roots.add(projectRoot);
    }
    return [...roots];
  }
}

/**
 * Claude Desktop keeps its embedded Claude Code projects below a dynamic
 * account/session directory. Discover only `.claude/projects` directories so
 * collection never scans desktop audit logs, caches, or other application
 * files that may contain private content.
 */
async function findProjectRoots(root: string): Promise<string[]> {
  const result: string[] = [];

  async function walk(path: string): Promise<void> {
    let entries;
    try {
      entries = await readdir(path, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      const full = join(path, entry.name);
      if (entry.name === 'projects' && basename(path) === '.claude') {
        result.push(full);
        continue;
      }
      await walk(full);
    }
  }

  await walk(root);
  return result.sort();
}
