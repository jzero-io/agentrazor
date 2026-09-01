import { computed, ref } from 'vue';
import { authApi, clearRefreshToken, clearToken, getToken, setAuthErrorHandler, setRefreshToken, setToken } from '../../service/api';
import type { UserInfo } from '../../service/api';

export function useAuthSession(options: {
  onAuthError: () => void;
  showError: (error: unknown) => void;
}) {
  const currentUser = ref<UserInfo | null>(null);
  const authChecking = ref(true);
  const hasAuthToken = ref(Boolean(getToken()));
  const loginVisible = ref(false);
  const loginUsername = ref('');
  const loginPassword = ref('');
  const loginLoading = ref(false);
  const userInitial = computed(() => currentUser.value?.username.trim().slice(0, 2).toUpperCase() || 'AR');

  function installAuthErrorHandler() {
    setAuthErrorHandler(() => {
      hasAuthToken.value = false;
      currentUser.value = null;
      options.onAuthError();
    });
  }

  async function restoreSession() {
    if (!getToken()) {
      authChecking.value = false;
      return null;
    }
    try {
      currentUser.value = await authApi.getUserInfo();
      hasAuthToken.value = true;
      return currentUser.value;
    } catch {
      currentUser.value = null;
      hasAuthToken.value = false;
      return null;
    } finally {
      authChecking.value = false;
    }
  }

  async function submitLogin() {
    const username = loginUsername.value.trim();
    const password = loginPassword.value;
    if (!username || !password) return null;
    loginLoading.value = true;
    try {
      const { token, refreshToken } = await authApi.pwdLogin(username, password);
      setToken(token);
      setRefreshToken(refreshToken);
      hasAuthToken.value = true;
      currentUser.value = await authApi.getUserInfo();
      loginVisible.value = false;
      loginUsername.value = '';
      loginPassword.value = '';
      return currentUser.value;
    } catch (error) {
      options.showError(error);
      return null;
    } finally {
      loginLoading.value = false;
    }
  }

  function openLogin() {
    loginVisible.value = true;
  }

  function clearSession() {
    clearToken();
    clearRefreshToken();
    hasAuthToken.value = false;
    currentUser.value = null;
  }

  function finishAuthChecking() {
    authChecking.value = false;
  }

  return {
    currentUser,
    authChecking,
    hasAuthToken,
    loginVisible,
    loginUsername,
    loginPassword,
    loginLoading,
    userInitial,
    installAuthErrorHandler,
    restoreSession,
    submitLogin,
    openLogin,
    clearSession,
    finishAuthChecking
  };
}
