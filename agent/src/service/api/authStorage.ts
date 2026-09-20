const STANDALONE_TOKEN_KEY = 'agentrazor_token';
const STANDALONE_REFRESH_TOKEN_KEY = 'agentrazor_refresh_token';
const DEFAULT_ADMIN_STORAGE_PREFIX = 'AGENTRAZOR_ADMIN_';
const EMBEDDED_APP_BASE = '/agent-app/';

let adminStoragePrefix = DEFAULT_ADMIN_STORAGE_PREFIX;

export const SHARED_AUTH_SESSION_CHANGED = 'agentrazor:shared-auth-session-changed';

export interface AuthTokenPair {
  token: string;
  refreshToken: string;
}

export type AdminThemeScheme = 'light' | 'dark' | 'auto';

function getAdminStorageKeys() {
  return {
    token: `${adminStoragePrefix}token`,
    refreshToken: `${adminStoragePrefix}refreshToken`,
    themeSettings: `${adminStoragePrefix}themeSettings`
  };
}

/**
 * The parent only sends this non-secret namespace after exact same-origin and
 * source validation. This keeps custom Admin storage prefixes compatible.
 */
export function setAdminStoragePrefix(prefix: string) {
  if (prefix.length > 256 || prefix === adminStoragePrefix) return false;
  adminStoragePrefix = prefix;
  return true;
}

export function isAdminAuthStorageKey(key: string | null) {
  const keys = getAdminStorageKeys();
  return key === keys.token || key === keys.refreshToken;
}

export function isAdminThemeStorageKey(key: string | null) {
  return key === getAdminStorageKeys().themeSettings;
}

/**
 * The Admin token store is shared only with the same-origin Agent iframe that
 * is mounted below /agent-app/. A cross-origin frame must never receive or
 * persist the Admin session.
 */
export function isAdminEmbeddedAuthContext() {
  if (window.parent === window) return false;
  if (window.location.pathname !== EMBEDDED_APP_BASE.slice(0, -1)
    && !window.location.pathname.startsWith(EMBEDDED_APP_BASE)) {
    return false;
  }

  try {
    return window.parent.location.origin === window.location.origin;
  } catch {
    return false;
  }
}

export function decodeAdminStorageValue(value: string | null) {
  if (!value) return '';
  try {
    const decoded = JSON.parse(value) as unknown;
    return typeof decoded === 'string' ? decoded : '';
  } catch {
    return '';
  }
}

/** Read the Admin appearance from the same-origin shared local storage. */
export function getAdminThemeScheme(): AdminThemeScheme | null {
  if (!isAdminEmbeddedAuthContext()) return null;
  try {
    const value = JSON.parse(localStorage.getItem(getAdminStorageKeys().themeSettings) || 'null') as unknown;
    if (!value || typeof value !== 'object') return null;
    const scheme = (value as { themeScheme?: unknown }).themeScheme;
    return scheme === 'light' || scheme === 'dark' || scheme === 'auto' ? scheme : null;
  } catch {
    return null;
  }
}

function getStorageKeys() {
  if (isAdminEmbeddedAuthContext()) {
    const keys = getAdminStorageKeys();
    return {
      token: keys.token,
      refreshToken: keys.refreshToken,
      jsonEncoded: true
    };
  }

  return {
    token: STANDALONE_TOKEN_KEY,
    refreshToken: STANDALONE_REFRESH_TOKEN_KEY,
    jsonEncoded: false
  };
}

function readValue(key: string, jsonEncoded: boolean) {
  const value = localStorage.getItem(key);
  return jsonEncoded ? decodeAdminStorageValue(value) : value || '';
}

function writeValue(key: string, value: string, jsonEncoded: boolean) {
  localStorage.setItem(key, jsonEncoded ? JSON.stringify(value) : value);
}

function notifyAdminSessionChanged(authenticated: boolean) {
  if (!isAdminEmbeddedAuthContext()) return;
  window.parent.postMessage(
    { type: SHARED_AUTH_SESSION_CHANGED, authenticated },
    window.location.origin
  );
}

export function getStoredToken() {
  const keys = getStorageKeys();
  return readValue(keys.token, keys.jsonEncoded);
}

export function getStoredRefreshToken() {
  const keys = getStorageKeys();
  return readValue(keys.refreshToken, keys.jsonEncoded);
}

export function setStoredToken(token: string) {
  const keys = getStorageKeys();
  writeValue(keys.token, token, keys.jsonEncoded);
  notifyAdminSessionChanged(Boolean(token));
}

export function setStoredRefreshToken(refreshToken: string) {
  const keys = getStorageKeys();
  writeValue(keys.refreshToken, refreshToken, keys.jsonEncoded);
}

export function setStoredAuthTokens({ token, refreshToken }: AuthTokenPair) {
  const keys = getStorageKeys();
  writeValue(keys.token, token, keys.jsonEncoded);
  writeValue(keys.refreshToken, refreshToken, keys.jsonEncoded);
  notifyAdminSessionChanged(Boolean(token));
}

export function clearStoredToken() {
  const keys = getStorageKeys();
  localStorage.removeItem(keys.token);
  notifyAdminSessionChanged(false);
}

export function clearStoredRefreshToken() {
  const keys = getStorageKeys();
  localStorage.removeItem(keys.refreshToken);
}

export function clearStoredAuthTokens() {
  const keys = getStorageKeys();
  localStorage.removeItem(keys.token);
  localStorage.removeItem(keys.refreshToken);
  notifyAdminSessionChanged(false);
}
