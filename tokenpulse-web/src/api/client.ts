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

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config;
    if (
      error.response?.status !== 401 ||
      !config ||
      config.url?.includes('/auth/refresh') ||
      config.url?.includes('/auth/login')
    )
      return Promise.reject(error);
    refreshing ??= api
      .post('/auth/refresh')
      .then(() => undefined)
      .finally(() => {
        refreshing = null;
      });
    await refreshing;
    return api.request(config);
  },
);

export async function data<T>(promise: Promise<{ data: Envelope<T> }>): Promise<T> {
  return (await promise).data.data;
}
export function errorMessage(error: unknown): string {
  if (axios.isAxiosError<Envelope<unknown>>(error))
    return error.response?.data?.message ?? error.message;
  return error instanceof Error ? error.message : String(error);
}
