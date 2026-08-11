import { defineStore } from 'pinia';
import { ref } from 'vue';
import { api, data } from '../api/client.js';

export interface User {
  id: number;
  username: string;
  email?: string;
  status: string;
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  const checked = ref(false);
  async function me(): Promise<User | null> {
    try {
      user.value = await data<User>(api.get('/auth/me'));
    } catch {
      user.value = null;
    } finally {
      checked.value = true;
    }
    return user.value;
  }
  async function login(identity: string, password: string): Promise<void> {
    user.value = await data<User>(api.post('/auth/login', { identity, password }));
    checked.value = true;
  }
  async function register(username: string, email: string, password: string): Promise<void> {
    user.value = await data<User>(
      api.post('/auth/register', { username, email: email || undefined, password }),
    );
    checked.value = true;
  }
  async function logout(): Promise<void> {
    await api.post('/auth/logout');
    user.value = null;
  }
  return { user, checked, me, login, register, logout };
});
