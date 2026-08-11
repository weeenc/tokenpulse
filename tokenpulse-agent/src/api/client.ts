import { setTimeout as delay } from 'node:timers/promises';
import { z } from 'zod';
import type { UsageEvent } from '../types/usage.js';

const envelopeSchema = z.object({ code: z.number(), message: z.string(), data: z.unknown() });

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: number,
    message: string,
    readonly data?: unknown,
  ) {
    super(message);
  }
}

export class ApiClient {
  constructor(
    readonly serverUrl: string,
    private readonly deviceToken?: string,
  ) {}

  async requestDevice(input: object) {
    return this.request('/api/v1/device-auth/request', { method: 'POST', body: input });
  }
  async pollDeviceToken(deviceCode: string) {
    return this.request('/api/v1/device-auth/token', { method: 'POST', body: { deviceCode } });
  }
  async deviceMe() {
    return this.request('/api/v1/devices/me', { method: 'GET', authenticated: true });
  }
  async heartbeat() {
    return this.request('/api/v1/devices/heartbeat', {
      method: 'POST',
      body: {},
      authenticated: true,
    });
  }

  async upload(
    events: UsageEvent[],
    attempts = 4,
  ): Promise<{ received: number; inserted: number; duplicated: number }> {
    let lastError: unknown;
    for (let attempt = 0; attempt < attempts; attempt++) {
      try {
        return (await this.request('/api/v1/usage/batch', {
          method: 'POST',
          body: { events },
          authenticated: true,
        })) as { received: number; inserted: number; duplicated: number };
      } catch (error) {
        lastError = error;
        if (
          error instanceof ApiError &&
          (error.status === 401 || (error.status < 500 && error.status !== 429))
        )
          throw error;
        if (attempt + 1 < attempts)
          await delay(2 ** attempt * 1000 + Math.floor(Math.random() * 250));
      }
    }
    throw lastError;
  }

  private async request(
    path: string,
    options: { method: string; body?: object; authenticated?: boolean },
  ): Promise<unknown> {
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (options.body) headers['Content-Type'] = 'application/json';
    if (options.authenticated) {
      if (!this.deviceToken) throw new Error('Device token is unavailable');
      headers.Authorization = `Bearer ${this.deviceToken}`;
    }
    let response: Response;
    try {
      response = await fetch(new URL(path, this.serverUrl), {
        method: options.method,
        headers,
        ...(options.body ? { body: JSON.stringify(options.body) } : {}),
        signal: AbortSignal.timeout(15_000),
      });
    } catch (error) {
      throw new Error(`Unable to reach ${this.serverUrl}: ${String(error)}`, { cause: error });
    }
    const parsed = envelopeSchema.safeParse(await response.json().catch(() => null));
    if (!parsed.success)
      throw new Error(`Server returned an invalid response (${response.status})`);
    if (!response.ok || parsed.data.code !== 0)
      throw new ApiError(response.status, parsed.data.code, parsed.data.message, parsed.data.data);
    return parsed.data.data;
  }
}
