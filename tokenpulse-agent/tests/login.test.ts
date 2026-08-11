import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { login } from '../src/auth/device-auth.js';
import type { CredentialStore } from '../src/auth/credential-store.js';
import type { AgentPaths } from '../src/platform/paths.js';
import { ConfigStore } from '../src/storage/config.js';

const directories: string[] = [];

function envelope(status: number, code: number, message: string, data: unknown): Response {
  return new Response(JSON.stringify({ code, message, data }), { status });
}

async function fixture(): Promise<{ config: ConfigStore; credentials: CredentialStore }> {
  const baseDir = await mkdtemp(join(tmpdir(), 'tokenpulse-login-'));
  directories.push(baseDir);
  const paths: AgentPaths = {
    baseDir,
    config: join(baseDir, 'config.json'),
    database: join(baseDir, 'data.db'),
    credential: join(baseDir, 'credential.bin'),
    logs: join(baseDir, 'logs'),
  };
  return {
    config: new ConfigStore(paths),
    credentials: {
      saveDeviceToken: vi.fn(),
      getDeviceToken: vi.fn(),
      deleteDeviceToken: vi.fn(),
    },
  };
}

afterEach(async () => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  await Promise.all(
    directories.splice(0).map((path) => rm(path, { recursive: true, force: true })),
  );
});

describe('device login polling', () => {
  it('continues pending polling and stores an approved opaque token', async () => {
    const { config, credentials } = await fixture();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        envelope(201, 0, 'success', {
          deviceCode: 'dc_secret',
          userCode: 'ABCD-EFGH',
          verificationUriComplete: 'https://tokenpulse.example.com/device?code=ABCD-EFGH',
          expiresIn: 2,
          interval: 0,
        }),
      )
      .mockResolvedValueOnce(envelope(400, 40002, 'authorization_pending', null))
      .mockResolvedValueOnce(
        envelope(200, 0, 'success', {
          deviceToken: 'dt_secret',
          account: 'wenc',
          device: { deviceId: 'device-1', deviceName: 'Test Mac' },
        }),
      );
    vi.stubGlobal('fetch', fetchMock);
    await login(config, credentials, '0.1.0', {
      server: 'https://tokenpulse.example.com',
      browser: false,
    });
    expect(credentials.saveDeviceToken).toHaveBeenCalledWith('dt_secret');
    expect(await config.load()).toMatchObject({ deviceId: 'device-1', deviceName: 'Test Mac' });
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it.each([
    ['access_denied', 'denied'],
    ['expired_token', 'expired'],
  ])('reports terminal authorization state %s', async (state, message) => {
    const { config, credentials } = await fixture();
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(
          envelope(201, 0, 'success', {
            deviceCode: 'dc_secret',
            userCode: 'ABCD-EFGH',
            verificationUriComplete: 'https://example.com/device?code=ABCD-EFGH',
            expiresIn: 2,
            interval: 0,
          }),
        )
        .mockResolvedValueOnce(envelope(400, 40002, state, null)),
    );
    await expect(
      login(config, credentials, '0.1.0', {
        server: 'https://example.com',
        browser: false,
      }),
    ).rejects.toThrow(message);
    expect(credentials.saveDeviceToken).not.toHaveBeenCalled();
  });
});
