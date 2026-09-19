import { useAuthStore } from '@/store/modules/auth';
import { localStg } from '@/utils/storage';
import { RefreshToken } from '../api';
import type { RequestInstanceState } from './type';

export function getAuthorization() {
  const token = localStg.get('token');
  const Authorization = token ? `Bearer ${token}` : null;

  return Authorization;
}

export function getLanguage() {
  const language = localStg.get('lang') || 'zh-CN';

  return language;
}

/** refresh token */
async function handleRefreshToken() {
  const { resetStore } = useAuthStore();

  const rToken = localStg.get('refreshToken') || '';
  if (!rToken) {
    await resetStore();
    return false;
  }

  const { error, data } = await RefreshToken(rToken);
  if (!error) {
    localStg.set('token', data.token);
    localStg.set('refreshToken', data.refreshToken);
    return true;
  }

  await resetStore();

  return false;
}

export async function handleExpiredRequest(state: RequestInstanceState) {
  if (!state.refreshTokenFn) {
    state.refreshTokenFn = handleRefreshToken();
  }

  const refreshing = state.refreshTokenFn;
  try {
    return await refreshing;
  } finally {
    // Clear only the promise this request awaited. A timer left the old promise
    // reusable for one second and could replay an already failed refresh.
    if (state.refreshTokenFn === refreshing) {
      state.refreshTokenFn = null;
    }
  }
}

export function showErrorMsg(state: RequestInstanceState, message: string) {
  if (!state.errMsgStack?.length) {
    state.errMsgStack = [];
  }

  const isExist = state.errMsgStack.includes(message);

  if (!isExist) {
    state.errMsgStack.push(message);

    window.$message?.error(message, {
      onLeave: () => {
        state.errMsgStack = state.errMsgStack.filter(msg => msg !== message);

        setTimeout(() => {
          state.errMsgStack = [];
        }, 5000);
      }
    });
  }
}
