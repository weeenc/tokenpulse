import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock('../src/api/client.js', () => ({
  api: mocks,
  data: async <T>(promise: Promise<{ data: { data: T } }>) => (await promise).data.data,
}));

import { useAuthStore } from '../src/stores/auth.js';

const user = { id: 1, username: 'wenc', status: 'ACTIVE' };

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    mocks.get.mockReset();
    mocks.post.mockReset();
  });

  it('loads the current user', async () => {
    mocks.get.mockResolvedValue({ data: { data: user } });
    const store = useAuthStore();
    await expect(store.me()).resolves.toEqual(user);
    expect(store.checked).toBe(true);
  });

  it('clears state when the session is unavailable', async () => {
    mocks.get.mockRejectedValue(new Error('unauthorized'));
    const store = useAuthStore();
    await expect(store.me()).resolves.toBeNull();
    expect(store.user).toBeNull();
    expect(store.checked).toBe(true);
  });

  it('logs in and logs out', async () => {
    mocks.post.mockResolvedValueOnce({ data: { data: user } }).mockResolvedValueOnce({});
    const store = useAuthStore();
    await store.login('wenc', 'password');
    expect(store.user).toEqual(user);
    await store.logout();
    expect(store.user).toBeNull();
  });
});
