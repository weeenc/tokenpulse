import { record } from '../common.js';

export function recognizesClaudeJsonl(lines: string[]): boolean {
  return lines.some((line) => {
    try {
      const root = record(JSON.parse(line));
      return ['assistant', 'user', 'system', 'summary'].includes(String(root?.type ?? ''));
    } catch {
      return false;
    }
  });
}
