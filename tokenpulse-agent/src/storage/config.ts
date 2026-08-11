import { randomUUID } from 'node:crypto';
import { mkdir, readFile, rename, writeFile } from 'node:fs/promises';
import type { AgentPaths } from '../platform/paths.js';

export interface AgentConfig {
  serverUrl: string;
  deviceId?: string;
  deviceName?: string;
  installationId: string;
  lastCollect?: string;
  lastSync?: string;
}

const DEFAULT_SERVER = 'http://localhost:8080';

export class ConfigStore {
  constructor(private readonly paths: AgentPaths) {}

  async load(): Promise<AgentConfig | null> {
    try {
      return JSON.parse(await readFile(this.paths.config, 'utf8')) as AgentConfig;
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === 'ENOENT') return null;
      throw new Error(`Unable to read config: ${String(error)}`, { cause: error });
    }
  }

  async loadOrCreate(serverOverride?: string): Promise<AgentConfig> {
    const existing = await this.load();
    if (existing) {
      return {
        ...existing,
        serverUrl: serverOverride ?? process.env.TOKENPULSE_SERVER_URL ?? existing.serverUrl,
      };
    }
    const created: AgentConfig = {
      serverUrl: serverOverride ?? process.env.TOKENPULSE_SERVER_URL ?? DEFAULT_SERVER,
      installationId: randomUUID(),
    };
    await this.save(created);
    return created;
  }

  async save(config: AgentConfig): Promise<void> {
    await mkdir(this.paths.baseDir, { recursive: true, mode: 0o700 });
    const temporary = `${this.paths.config}.tmp`;
    await writeFile(temporary, `${JSON.stringify(config, null, 2)}\n`, {
      encoding: 'utf8',
      mode: 0o600,
    });
    await rename(temporary, this.paths.config);
  }
}
