import { record } from '../common.js';

export function recognizesCodexJsonl(lines: string[]): boolean {
  return lines.some((line) => {
    try {
      const root = record(JSON.parse(line));
      return ['session_meta', 'turn_context', 'event_msg'].includes(String(root?.type ?? ''));
    } catch {
      return false;
    }
  });
}
