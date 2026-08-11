import type { CredentialStore } from './auth/credential-store.js';
import { recoverConfig } from './auth/device-auth.js';
import { ApiClient, ApiError } from './api/client.js';
import { collectors } from './collectors/index.js';
import { agentPaths } from './platform/paths.js';
import { ConfigStore, type AgentConfig } from './storage/config.js';
import { LocalDatabase } from './storage/database.js';
import type { FileProgress } from './types/usage.js';

export interface AppDependencies {
  configStore: ConfigStore;
  credentials: CredentialStore;
  agentVersion: string;
}

export async function collect(
  configStore: ConfigStore,
  verbose = false,
): Promise<{ found: number; inserted: number; warnings: string[] }> {
  const paths = agentPaths();
  const database = new LocalDatabase(paths.database);
  const warnings: string[] = [];
  const progresses: FileProgress[] = [];
  try {
    const context = {
      getProgress: (path: string) => database.getProgress(path),
      saveProgress: (progress: FileProgress) => progresses.push(progress),
      warn: (message: string) => {
        warnings.push(message);
        if (verbose) console.warn(`⚠ ${message}`);
      },
    };
    const events = [];
    for (const collector of collectors) {
      if (!(await collector.detect())) {
        if (verbose) console.log(`- ${collector.name}: not detected`);
        continue;
      }
      const collected = await collector.collect(context);
      events.push(...collected);
      if (verbose) console.log(`✓ ${collector.name}: ${collected.length} usage records scanned`);
    }
    // Event rows and source offsets advance atomically so a crash cannot skip
    // records that were scanned but not yet persisted.
    const inserted = database.addCollected(events, progresses);
    const config = await configStore.loadOrCreate();
    await configStore.save({ ...config, lastCollect: new Date().toISOString() });
    return { found: events.length, inserted, warnings };
  } finally {
    database.close();
  }
}

export async function sync(
  dependencies: AppDependencies,
  verbose = false,
  serverOverride?: string,
): Promise<{ collected: number; uploaded: number; duplicated: number }> {
  const auth = await recoverConfig(
    dependencies.configStore,
    dependencies.credentials,
    serverOverride,
  );
  if (!auth)
    throw new Error('Device authentication expired or revoked.\n\nRun:\n\ntokenpulse login');
  const collection = await collect(dependencies.configStore, verbose);
  const database = new LocalDatabase(agentPaths().database);
  let uploaded = 0;
  let duplicated = 0;
  try {
    const client = new ApiClient(auth.config.serverUrl, auth.token);
    while (true) {
      const batch = database.pending(500);
      if (batch.length === 0) break;
      try {
        const result = await client.upload(batch);
        database.markSynced(batch.map((event) => event.eventId));
        uploaded += result.inserted;
        duplicated += result.duplicated;
      } catch (error) {
        database.markFailed(batch.map((event) => event.eventId));
        if (error instanceof ApiError && error.status === 401)
          throw new Error('Device authentication expired or revoked.\n\nRun:\n\ntokenpulse login', {
            cause: error,
          });
        throw error;
      }
    }
    await client.heartbeat(dependencies.agentVersion);
    const current = await dependencies.configStore.loadOrCreate();
    await dependencies.configStore.save({ ...current, lastSync: new Date().toISOString() });
    return { collected: collection.inserted, uploaded, duplicated };
  } finally {
    database.close();
  }
}

export async function status(
  dependencies: AppDependencies,
  version: string,
  serverOverride?: string,
): Promise<void> {
  const auth = await recoverConfig(
    dependencies.configStore,
    dependencies.credentials,
    serverOverride,
  );
  const config = auth?.config ?? (await dependencies.configStore.load());
  const database = new LocalDatabase(agentPaths().database);
  const pending = database.countPending();
  database.close();
  if (!config || !auth) {
    console.log('Not authenticated.\n\nRun:\n\ntokenpulse login');
    return;
  }
  const me = (await new ApiClient(config.serverUrl, auth.token).deviceMe()) as {
    user: { username: string };
    platform: string;
    arch: string;
  };
  console.log(
    `Server:\n${config.serverUrl}\n\nAccount:\n${me.user.username}\n\nDevice:\n${config.deviceName ?? '-'}\n\nPlatform:\n${me.platform} ${me.arch}\n\nAgent:\n${version}\n\nLast collect:\n${formatDate(config.lastCollect)}\n\nLast sync:\n${formatDate(config.lastSync)}\n\nPending:\n${pending} events`,
  );
}

export async function requireConfig(configStore: ConfigStore): Promise<AgentConfig> {
  return configStore.loadOrCreate();
}
function formatDate(value?: string): string {
  return value ? new Date(value).toLocaleString() : 'Never';
}
