import type { Envelope, LoginResponse } from './types';

export const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

const TOKEN_KEY = 'agentrazor_token';
const REFRESH_TOKEN_KEY = 'agentrazor_refresh_token';

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || '';
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

export function getRefreshToken() {
  return localStorage.getItem(REFRESH_TOKEN_KEY) || '';
}

export function setRefreshToken(refreshToken: string) {
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
}

export function clearRefreshToken() {
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

let authErrorHandler: (() => void) | null = null;

export function setAuthErrorHandler(fn: (() => void) | null) {
  authErrorHandler = fn;
}

type RefreshAccessTokenResult = 'refreshed' | 'expired' | 'failed';
let refreshTokenPromise: Promise<RefreshAccessTokenResult> | null = null;

export async function refreshAccessToken(): Promise<RefreshAccessTokenResult> {
  if (!refreshTokenPromise) {
    refreshTokenPromise = (async () => {
      const refreshToken = getRefreshToken();
      if (!refreshToken) return 'expired';
      try {
        const response = await fetch(`${apiBase}/api/v1/auth/refreshToken`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refreshToken })
        });
        const body = (await response.json().catch(() => null)) as Envelope<LoginResponse> | LoginResponse | null;
        const envelope = body !== null && isEnvelope<LoginResponse>(body) ? body : null;
        if (response.status === 401 || envelope?.code === 40102) return 'expired';
        if (!response.ok) return 'failed';
        if (envelope && envelope.code !== 200) return envelope.code === 40102 ? 'expired' : 'failed';
        const data = envelope ? envelope.data : (body as LoginResponse | null);
        if (!data?.token || !data.refreshToken) return 'failed';
        setToken(data.token);
        setRefreshToken(data.refreshToken);
        return 'refreshed';
      } catch {
        return 'failed';
      }
    })().finally(() => {
      refreshTokenPromise = null;
    });
  }
  return refreshTokenPromise;
}

export async function refreshAccessTokenOrExpire() {
  const result = await refreshAccessToken();
  if (result === 'expired') expireSession();
  return result === 'refreshed';
}

export function expireSession() {
  clearToken();
  clearRefreshToken();
  authErrorHandler?.();
}

export function withAuthHeaders(init?: RequestInit): Headers {
  const headers = new Headers(init?.headers);
  const hasBody = init?.body !== undefined && init.body !== null;
  if (hasBody && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const token = getToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  return headers;
}

export async function request<T>(path: string, init?: RequestInit, retried = false): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: withAuthHeaders(init)
  });
  const body = (await response.json().catch(() => null)) as Envelope<T> | T | null;
  const envelope = body !== null && isEnvelope<T>(body) ? body : null;
  const authFailed = response.status === 401 || envelope?.code === 40101;
  if (authFailed) {
    if (!retried) {
      const refreshResult = await refreshAccessToken();
      if (refreshResult === 'refreshed') return request<T>(path, init, true);
      if (refreshResult === 'failed') throw new Error('登录续期失败，请稍后重试');
    }
    expireSession();
    throw new Error('登录已过期，请重新登录');
  }
  if (!response.ok) throw new Error(`请求失败（${response.status}）`);
  if (envelope) {
    if (envelope.code !== 200) {
      if (envelope.code === 40102) expireSession();
      throw new Error(envelope.msg || '服务请求失败');
    }
    return envelope.data;
  }
  return body as T;
}

export function isEnvelope<T>(body: Envelope<T> | T): body is Envelope<T> {
  return typeof body === 'object' && body !== null && 'code' in body && 'data' in body;
}
