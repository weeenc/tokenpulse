import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiClient, ApiError } from '../src/api/client.js';
import type { UsageEvent } from '../src/types/usage.js';

const event: UsageEvent = {
  eventId: 'a'.repeat(64),
  source: 'codex',
  inputTokens: 10,
  outputTokens: 2,
  cachedInputTokens: 3,
  reasoningTokens: 1,
  totalTokens: 12,
  occurredAt: '2026-08-07T00:00:00Z',
};

function response(status: number, code: number, message: string, data: unknown): Response {
  return new Response(JSON.stringify({ code, message, data }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('API client', () => {
  it('sends the opaque device credential in the authorization header', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response(200, 0, 'success', { deviceId: '1' }));
    vi.stubGlobal('fetch', fetchMock);
    await new ApiClient('https://tokenpulse.example.com', 'dt_secret').deviceMe();
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(fetchMock.mock.calls[0]?.[0].toString()).toBe(
      'https://tokenpulse.example.com/api/v1/devices/me',
    );
    expect(fetchMock.mock.calls[0]?.[1]?.headers).toMatchObject({
      Authorization: 'Bearer dt_secret',
    });
  });

  it('reports the running agent version with its heartbeat', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response(200, 0, 'success', {}));
    vi.stubGlobal('fetch', fetchMock);
    await new ApiClient('https://tokenpulse.example.com', 'dt_secret').heartbeat('0.1.1');
    expect(fetchMock.mock.calls[0]?.[0].toString()).toBe(
      'https://tokenpulse.example.com/api/v1/devices/heartbeat',
    );
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({
      agentVersion: '0.1.1',
    });
  });

  it('does not retry an authentication failure', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response(401, 40102, 'revoked', null));
    vi.stubGlobal('fetch', fetchMock);
    await expect(new ApiClient('https://example.com', 'dt_old').upload([event])).rejects.toEqual(
      expect.objectContaining<ApiError>({ status: 401, code: 40102 }),
    );
    expect(fetchMock).toHaveBeenCalledOnce();
  });

  it('retries a transient server error and then succeeds', async () => {
    vi.useFakeTimers();
    vi.spyOn(Math, 'random').mockReturnValue(0);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(response(500, 50000, 'temporary', null))
      .mockResolvedValueOnce(
        response(200, 0, 'success', { received: 1, inserted: 1, duplicated: 0 }),
      );
    vi.stubGlobal('fetch', fetchMock);
    const upload = new ApiClient('https://example.com', 'dt_valid').upload([event], 2);
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    await vi.advanceTimersByTimeAsync(1000);
    await expect(upload).resolves.toEqual({ received: 1, inserted: 1, duplicated: 0 });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('retries a rate limit response with exponential backoff', async () => {
    vi.useFakeTimers();
    vi.spyOn(Math, 'random').mockReturnValue(0);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(response(429, 42901, 'too many requests', null))
      .mockResolvedValueOnce(
        response(200, 0, 'success', { received: 1, inserted: 0, duplicated: 1 }),
      );
    vi.stubGlobal('fetch', fetchMock);
    const upload = new ApiClient('https://example.com', 'dt_valid').upload([event], 2);
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    await vi.advanceTimersByTimeAsync(1000);
    await expect(upload).resolves.toEqual({ received: 1, inserted: 0, duplicated: 1 });
  });

  it('reports invalid and unreachable server responses clearly', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('not-json', { status: 502 }));
    vi.stubGlobal('fetch', fetchMock);
    await expect(
      new ApiClient('https://example.com', 'dt_valid').upload([event], 1),
    ).rejects.toThrow('invalid response');

    fetchMock.mockRejectedValueOnce(new Error('offline'));
    await expect(
      new ApiClient('https://example.com', 'dt_valid').upload([event], 1),
    ).rejects.toThrow('Unable to reach https://example.com');
  });
});
