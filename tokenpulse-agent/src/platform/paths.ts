import { homedir } from 'node:os';
import { join } from 'node:path';

export interface AgentPaths {
  baseDir: string;
  config: string;
  database: string;
  credential: string;
  logs: string;
}

export function agentPaths(platform: NodeJS.Platform = process.platform): AgentPaths {
  const baseDir =
    platform === 'win32'
      ? join(process.env.APPDATA ?? join(homedir(), 'AppData', 'Roaming'), 'tokenpulse-agent')
      : join(homedir(), '.tokenpulse');
  return {
    baseDir,
    config: join(baseDir, 'config.json'),
    database: join(baseDir, 'data.db'),
    credential: join(baseDir, 'credential.bin'),
    logs: join(baseDir, 'logs'),
  };
}

export function sourcePaths(): { codex: string; claude: string; cursor: string } {
  const cursor =
    process.platform === 'win32'
      ? join(process.env.APPDATA ?? join(homedir(), 'AppData', 'Roaming'), 'Cursor')
      : process.platform === 'darwin'
        ? join(homedir(), 'Library', 'Application Support', 'Cursor')
        : join(process.env.XDG_CONFIG_HOME ?? join(homedir(), '.config'), 'Cursor');
  return {
    codex: join(homedir(), '.codex', 'sessions'),
    claude: join(homedir(), '.claude', 'projects'),
    cursor: join(cursor, 'User', 'globalStorage', 'state.vscdb'),
  };
}
