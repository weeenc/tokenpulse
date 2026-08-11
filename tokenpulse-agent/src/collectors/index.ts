import { ClaudeCodeCollector } from './claude-code/index.js';
import { CodexCollector } from './codex/index.js';
import { CursorCollector } from './cursor/index.js';

export const collectors = [new CodexCollector(), new ClaudeCodeCollector(), new CursorCollector()];
