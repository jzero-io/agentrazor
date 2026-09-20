export const ADMIN_AUTH_SESSION_CHANGE_EVENT = 'agentrazor:admin-auth-session-change';

/** Public storage-key namespace shared with the same-origin Agent iframe. */
export const ADMIN_AUTH_STORAGE_PREFIX = import.meta.env.VITE_STORAGE_PREFIX || '';

export const ADMIN_AUTH_STORAGE_KEYS = [
  `${ADMIN_AUTH_STORAGE_PREFIX}token`,
  `${ADMIN_AUTH_STORAGE_PREFIX}refreshToken`
] as const;

export type AdminAuthSessionChangeReason = 'login' | 'refresh' | 'clear';

/** Notify same-window consumers after the Admin authentication session changes. */
export function emitAdminAuthSessionChange(reason: AdminAuthSessionChangeReason) {
  window.dispatchEvent(
    new CustomEvent<AdminAuthSessionChangeReason>(ADMIN_AUTH_SESSION_CHANGE_EVENT, {
      detail: reason
    })
  );
}
