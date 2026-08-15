import { afterEach, describe, expect, it } from 'vitest';
import {
  AxiosError,
  AxiosHeaders,
  type AxiosAdapter,
  type InternalAxiosRequestConfig,
} from 'axios';
import { api, beginSessionEnd, isCanceledRequest, resumeSession } from '../src/api/client.js';

const originalAdapter = api.defaults.adapter;

function unauthorized(config: InternalAxiosRequestConfig): AxiosError {
  return new AxiosError('unauthorized', AxiosError.ERR_BAD_REQUEST, config, undefined, {
    data: { code: 40101, message: 'authentication required', data: null },
    status: 401,
    statusText: 'Unauthorized',
    headers: new AxiosHeaders(),
    config,
  });
}

afterEach(() => {
  api.defaults.adapter = originalAdapter;
  resumeSession();
});

describe('API session lifecycle', () => {
  it('refreshes an active session once and retries the original request', async () => {
    const urls: string[] = [];
    let deviceCalls = 0;
    api.defaults.adapter = (async (config) => {
      urls.push(config.url ?? '');
      if (config.url === '/devices' && deviceCalls++ === 0) throw unauthorized(config);
      return {
        data: {},
        status: 200,
        statusText: 'OK',
        headers: new AxiosHeaders(),
        config,
      };
    }) satisfies AxiosAdapter;

    resumeSession();
    await expect(api.get('/devices')).resolves.toMatchObject({ status: 200 });
    expect(urls).toEqual(['/devices', '/auth/refresh', '/devices']);
  });

  it('blocks business requests during logout but still permits the logout request', async () => {
    const urls: string[] = [];
    api.defaults.adapter = (async (config) => {
      urls.push(config.url ?? '');
      return {
        data: {},
        status: 200,
        statusText: 'OK',
        headers: new AxiosHeaders(),
        config,
      };
    }) satisfies AxiosAdapter;

    await beginSessionEnd();
    const businessRequest = api.get('/devices');
    await expect(businessRequest).rejects.toSatisfy(isCanceledRequest);
    await expect(api.post('/auth/logout')).resolves.toMatchObject({ status: 200 });
    expect(urls).toEqual(['/auth/logout']);
  });

  it('does not start a refresh when logout begins before a 401 response arrives', async () => {
    const urls: string[] = [];
    let rejectDevices: ((reason: AxiosError) => void) | undefined;
    let deviceConfig: InternalAxiosRequestConfig | undefined;
    api.defaults.adapter = (async (config) => {
      urls.push(config.url ?? '');
      if (config.url === '/devices') {
        deviceConfig = config;
        return new Promise((_, reject) => {
          rejectDevices = reject;
        });
      }
      throw new Error(`unexpected request: ${config.url}`);
    }) satisfies AxiosAdapter;

    resumeSession();
    const devicesRequest = api.get('/devices');
    await Promise.resolve();
    await beginSessionEnd();
    rejectDevices?.(unauthorized(deviceConfig!));

    await expect(devicesRequest).rejects.toMatchObject({ response: { status: 401 } });
    expect(urls).toEqual(['/devices']);
  });
});
