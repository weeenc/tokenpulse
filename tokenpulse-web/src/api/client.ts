import axios, { AxiosError } from 'axios';

export interface Envelope<T> {
  code: number;
  message: string;
  data: T;
}

export const api = axios.create({
  baseURL: '/api/v1',
  withCredentials: true,
  withXSRFToken: true,
  xsrfCookieName: 'tp_csrf',
  xsrfHeaderName: 'X-CSRF-Token',
  timeout: 15_000,
});
let refreshing: Promise<void> | null = null;
let sessionEnding = false;

interface RetryableRequest {
  _retry?: boolean;
  url?: string;
}

function isAuthMutation(url = ''): boolean {
  return ['/auth/refresh', '/auth/login', '/auth/register', '/auth/logout'].some((path) =>
    url.includes(path),
  );
}

// 退出期间不再启动业务请求或刷新会话，避免已清理的 Cookie 被并发 refresh 重新写回。
api.interceptors.request.use((config) => {
  if (sessionEnding && !config.url?.includes('/auth/logout')) {
    return Promise.reject(new axios.CanceledError('session is ending'));
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as (typeof error.config & RetryableRequest) | undefined;
    if (
      sessionEnding ||
      error.response?.status !== 401 ||
      !config ||
      config._retry ||
      isAuthMutation(config.url)
    )
      return Promise.reject(error);
    config._retry = true;
    refreshing ??= api
      .post('/auth/refresh')
      .then(() => undefined)
      .finally(() => {
        refreshing = null;
      });
    try {
      await refreshing;
    } catch (refreshError) {
      if (sessionEnding) throw new axios.CanceledError('session is ending');
      throw refreshError;
    }
    if (sessionEnding) throw new axios.CanceledError('session is ending');
    return api.request(config);
  },
);

export async function beginSessionEnd(): Promise<void> {
  sessionEnding = true;
  try {
    await refreshing;
  } catch {
    // 无论并发刷新是否成功，都继续发送幂等退出请求以清理 Cookie。
  }
}

export function resumeSession(): void {
  sessionEnding = false;
}

export function isCanceledRequest(error: unknown): boolean {
  return axios.isCancel(error);
}

export async function data<T>(promise: Promise<{ data: Envelope<T> }>): Promise<T> {
  return (await promise).data.data;
}
export function errorMessage(error: unknown): string {
  if (axios.isAxiosError<Envelope<unknown>>(error))
    return error.response?.data?.message ?? error.message;
  return error instanceof Error ? error.message : String(error);
}
