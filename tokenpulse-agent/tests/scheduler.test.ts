import { describe, expect, it } from 'vitest';
import { intervalSeconds, launchAgentXml, windowsTaskCommand } from '../src/scheduler/index.js';

describe('autosubmit interval', () => {
  it('parses minutes and hours', () => {
    expect(intervalSeconds('30m')).toBe(1800);
    expect(intervalSeconds('1h')).toBe(3600);
  });
  it('rejects ambiguous values', () => {
    expect(() => intervalSeconds('3600')).toThrow();
  });
  it('quotes Windows executable paths containing spaces', () => {
    expect(
      windowsTaskCommand(
        'C:\\Program Files\\nodejs\\node.exe',
        'C:\\Users\\Wenc Chao\\AppData\\Roaming\\npm\\node_modules\\tokenpulse\\dist\\cli.js',
      ),
    ).toBe(
      '"C:\\Program Files\\nodejs\\node.exe" "C:\\Users\\Wenc Chao\\AppData\\Roaming\\npm\\node_modules\\tokenpulse\\dist\\cli.js" sync',
    );
  });
  it('escapes launchd XML paths', () => {
    const xml = launchAgentXml('/usr/local/bin/node', '/Users/a&b/tokenpulse cli.js', 3600);
    expect(xml).toContain('<string>/Users/a&amp;b/tokenpulse cli.js</string>');
    expect(xml).toContain('<integer>3600</integer>');
  });
});
