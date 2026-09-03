import { computed, onScopeDispose, ref, watch } from 'vue';
import { authApi, clearRefreshToken, clearToken, getToken, setAuthErrorHandler, setRefreshToken, setToken } from '../../service/api';
import type { UserInfo } from '../../service/api';

export function useAuthSession(options: {
  onAuthError: () => void;
  showError: (error: unknown) => void;
  showSuccess: (message: string) => void;
}) {
  const currentUser = ref<UserInfo | null>(null);
  const authChecking = ref(true);
  const hasAuthToken = ref(Boolean(getToken()));
  const loginVisible = ref(false);
  const loginMode = ref<'password' | 'email'>('password');
  const loginUsername = ref('');
  const loginPassword = ref('');
  const loginEmail = ref('');
  const loginVerificationCode = ref('');
  const loginVerificationUuid = ref('');
  const verificationEmail = ref('');
  const verificationSending = ref(false);
  const verificationCountdown = ref(0);
  const loginLoading = ref(false);
  const userInitial = computed(() => currentUser.value?.username.trim().slice(0, 2).toUpperCase() || 'AR');

  let countdownTimer: ReturnType<typeof setInterval> | undefined;

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

  function stopVerificationCountdown() {
    if (countdownTimer) clearInterval(countdownTimer);
    countdownTimer = undefined;
  }

  function startVerificationCountdown() {
    stopVerificationCountdown();
    verificationCountdown.value = 30;
    countdownTimer = setInterval(() => {
      verificationCountdown.value -= 1;
      if (verificationCountdown.value <= 0) stopVerificationCountdown();
    }, 1000);
  }

  watch(loginEmail, value => {
    if (value.trim() === verificationEmail.value) return;
    loginVerificationUuid.value = '';
    loginVerificationCode.value = '';
  });

  onScopeDispose(stopVerificationCountdown);

  async function sendLoginVerificationCode() {
    const email = loginEmail.value.trim();
    if (!/^\S+@\S+\.\S+$/.test(email)) {
      options.showError(new Error('请输入正确的邮箱地址'));
      return false;
    }
    if (verificationSending.value || verificationCountdown.value > 0) return false;

    verificationSending.value = true;
    try {
      const { verificationUuid } = await authApi.sendEmailVerificationCode(email);
      loginVerificationUuid.value = verificationUuid;
      verificationEmail.value = email;
      startVerificationCountdown();
      options.showSuccess("验证码已发送，请检查邮箱");
      return true;
    } catch (error) {
      options.showError(error);
      return false;
    } finally {
      verificationSending.value = false;
    }
  }

  async function submitLogin() {
    const mode = loginMode.value;
    const username = loginUsername.value.trim();
    const password = loginPassword.value;
    const email = loginEmail.value.trim();
    const verificationCode = loginVerificationCode.value.trim();

    if (mode === 'password' && (!username || !password)) return null;
    if (mode === 'email' && (!email || !verificationCode || !loginVerificationUuid.value)) {
      options.showError(new Error(loginVerificationUuid.value ? '请输入邮箱验证码' : '请先获取邮箱验证码'));
      return null;
    }

    loginLoading.value = true;
    try {
      const { token, refreshToken } = mode === 'password'
        ? await authApi.pwdLogin(username, password)
        : await authApi.codeLogin(email, loginVerificationUuid.value, verificationCode);
      setToken(token);
      setRefreshToken(refreshToken);
      hasAuthToken.value = true;
      currentUser.value = await authApi.getUserInfo();
      loginVisible.value = false;
      loginUsername.value = '';
      loginPassword.value = '';
      loginEmail.value = '';
      loginVerificationCode.value = '';
      loginVerificationUuid.value = '';
      verificationEmail.value = '';
      verificationCountdown.value = 0;
      stopVerificationCountdown();
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
    loginMode,
    loginUsername,
    loginPassword,
    loginEmail,
    loginVerificationCode,
    loginVerificationUuid,
    verificationSending,
    verificationCountdown,
    loginLoading,
    userInitial,
    installAuthErrorHandler,
    restoreSession,
    submitLogin,
    sendLoginVerificationCode,
    openLogin,
    clearSession,
    finishAuthChecking
  };
}
