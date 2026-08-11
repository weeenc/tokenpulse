import { setTimeout as delay } from 'node:timers/promises';
import { randomUUID } from 'node:crypto';
import openBrowser from 'open';
import { ApiClient, ApiError } from '../api/client.js';
import type { CredentialStore } from './credential-store.js';
import type { AgentConfig, ConfigStore } from '../storage/config.js';
import { systemInfo } from '../platform/system-info.js';

type DeviceRequest = {
  deviceCode: string;
  userCode: string;
  verificationUriComplete: string;
  expiresIn: number;
  interval: number;
};
type DeviceToken = {
  deviceToken: string;
  account: string;
  device: { deviceId: string; deviceName: string };
};

export async function login(
  configStore: ConfigStore,
  credentials: CredentialStore,
  version: string,
  options: { server?: string; browser: boolean },
): Promise<void> {
  const current = await configStore.loadOrCreate(options.server);
  const config: AgentConfig = { ...current, installationId: randomUUID() };
  await configStore.save(config);
  const client = new ApiClient(config.serverUrl);
  const requested = (await client.requestDevice(
    systemInfo(version, config.installationId),
  )) as DeviceRequest;
  console.log(`\nDevice code: ${requested.userCode}\n`);
  if (options.browser) {
    console.log('Opening browser...');
    try {
      await openBrowser(requested.verificationUriComplete);
    } catch {
      console.warn('⚠ Browser could not be opened automatically.');
    }
  }
  console.log(
    `If the browser does not open automatically, visit:\n\n${requested.verificationUriComplete}\n\nWaiting for authorization...`,
  );
  const deadline = Date.now() + requested.expiresIn * 1000;
  while (Date.now() < deadline) {
    await delay(requested.interval * 1000);
    try {
      const result = (await client.pollDeviceToken(requested.deviceCode)) as DeviceToken;
      await credentials.saveDeviceToken(result.deviceToken);
      const updated: AgentConfig = {
        ...config,
        deviceId: result.device.deviceId,
        deviceName: result.device.deviceName,
      };
      await configStore.save(updated);
      console.log(
        `\n✓ Device connected successfully.\n\nAccount:\n${result.account}\n\nDevice:\n${result.device.deviceName}\n\nDevice ID:\n${result.device.deviceId}`,
      );
      return;
    } catch (error) {
      if (error instanceof ApiError && error.message === 'authorization_pending') continue;
      if (error instanceof ApiError && error.message === 'access_denied')
        throw new Error('Device authorization was denied.', { cause: error });
      if (error instanceof ApiError && error.message === 'expired_token')
        throw new Error('Device authorization expired. Run tokenpulse login again.', {
          cause: error,
        });
      throw error;
    }
  }
  throw new Error('Device authorization expired. Run tokenpulse login again.');
}

export async function recoverConfig(
  configStore: ConfigStore,
  credentials: CredentialStore,
  serverOverride?: string,
): Promise<{ config: AgentConfig; token: string } | null> {
  const token = await credentials.getDeviceToken();
  const existing = await configStore.load();
  if (!token) return null;
  const serverUrl =
    serverOverride ??
    process.env.TOKENPULSE_SERVER_URL ??
    existing?.serverUrl ??
    'http://localhost:8080';
  try {
    const data = (await new ApiClient(serverUrl, token).deviceMe()) as {
      deviceId: string;
      deviceName: string;
      installationId: string;
    };
    const config: AgentConfig = {
      ...(existing ?? { serverUrl, installationId: data.installationId }),
      serverUrl,
      installationId: data.installationId,
      deviceId: data.deviceId,
      deviceName: data.deviceName,
    };
    if (
      !existing ||
      existing.deviceId !== data.deviceId ||
      existing.installationId !== data.installationId
    )
      await configStore.save(config);
    return { config, token };
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) return null;
    throw error;
  }
}
