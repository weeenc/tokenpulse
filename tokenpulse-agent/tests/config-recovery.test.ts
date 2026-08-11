import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { recoverConfig } from '../src/auth/device-auth.js';
import type { CredentialStore } from '../src/auth/credential-store.js';
import type { AgentPaths } from '../src/platform/paths.js';
import { ConfigStore } from '../src/storage/config.js';

const dirs: string[] = [];

async function store(): Promise<{ config: ConfigStore; paths: AgentPaths }> {
  const baseDir = await mkdtemp(join(tmpdir(), 'tokenpulse-config-'));
  dirs.push(baseDir);
  const paths: AgentPaths = {
    baseDir,
    config: join(baseDir, 'config.json'),
    database: join(baseDir, 'data.db'),
    credential: join(baseDir, 'credential.bin'),
    logs: join(baseDir, 'logs'),
  };
  return { config: new ConfigStore(paths), paths };
}

function credentials(token: string | null): CredentialStore {
  return {
    saveDeviceToken: vi.fn(),
    getDeviceToken: vi.fn().mockResolvedValue(token),
    deleteDeviceToken: vi.fn(),
  };
}

afterEach(async () => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  await Promise.all(dirs.splice(0).map((path) => rm(path, { recursive: true, force: true })));
});

describe('configuration recovery', () => {
  it('restores a deleted config from a valid secure credential', async () => {
    const { config, paths } = await store();
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          code: 0,
          message: 'success',
          data: {
            deviceId: 'device-id',
            deviceName: 'MacBook Pro',
            installationId: 'installation-id',
          },
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal('fetch', fetchMock);
    const recovered = await recoverConfig(
      config,
      credentials('dt_valid'),
      'https://tokenpulse.example.com',
    );
    expect(recovered?.config).toMatchObject({
      serverUrl: 'https://tokenpulse.example.com',
      deviceId: 'device-id',
      installationId: 'installation-id',
    });
    expect(JSON.parse(await readFile(paths.config, 'utf8'))).toMatchObject({
      deviceName: 'MacBook Pro',
    });
  });

  it('returns unauthenticated when the credential is missing or revoked', async () => {
    const { config } = await store();
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    await expect(recoverConfig(config, credentials(null))).resolves.toBeNull();
    expect(fetchMock).not.toHaveBeenCalled();

    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 40102, message: 'revoked', data: null }), {
        status: 401,
      }),
    );
    await expect(recoverConfig(config, credentials('dt_revoked'))).resolves.toBeNull();
  });

  it('honors an explicit server override when creating config', async () => {
    const { config } = await store();
    vi.stubEnv('TOKENPULSE_SERVER_URL', 'https://environment.example.com');
    const created = await config.loadOrCreate('https://explicit.example.com');
    expect(created.serverUrl).toBe('https://explicit.example.com');
    expect(created.installationId).toMatch(/^[0-9a-f-]{36}$/);
  });
});
