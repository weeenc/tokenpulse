import { ApiClient } from '../api/client.js';
import { credentialStoreAvailable, type CredentialStore } from '../auth/credential-store.js';
import { collectors } from '../collectors/index.js';
import { agentPaths } from '../platform/paths.js';
import { autosubmitStatus } from '../scheduler/index.js';
import type { ConfigStore } from '../storage/config.js';
import { LocalDatabase } from '../storage/database.js';

export async function doctor(
  configStore: ConfigStore,
  credentials: CredentialStore,
  serverOverride?: string,
): Promise<void> {
  const results: Array<[boolean, string]> = [
    [Number(process.versions.node.split('.')[0]) >= 20, `Node.js ${process.versions.node}`],
  ];
  const config = await configStore.loadOrCreate(serverOverride);
  try {
    const response = await fetch(new URL('/health', config.serverUrl), {
      signal: AbortSignal.timeout(5000),
    });
    results.push([response.ok, 'Server reachable']);
  } catch {
    results.push([false, 'Server reachable']);
  }
  const token = await credentials.getDeviceToken();
  if (token) {
    try {
      await new ApiClient(config.serverUrl, token).deviceMe();
      results.push([true, 'Device authenticated']);
    } catch {
      results.push([false, 'Device authenticated']);
    }
  } else results.push([false, 'Device authenticated']);
  for (const collector of collectors)
    results.push([await collector.detect(), `${collector.name} detected`]);
  try {
    const db = new LocalDatabase(agentPaths().database);
    db.close();
    results.push([true, 'Local database']);
  } catch {
    results.push([false, 'Local database']);
  }
  results.push([
    await credentialStoreAvailable(),
    process.platform === 'darwin'
      ? 'Keychain available'
      : process.platform === 'win32'
        ? 'DPAPI available'
        : 'Credential store unsupported',
  ]);
  results.push([await autosubmitStatus(), 'Autosubmit enabled']);
  for (const [ok, message] of results) console.log(`${ok ? '✓' : '✗'} ${message}`);
}
