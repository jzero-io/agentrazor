import type { AxiosResponse } from 'axios';
import { BACKEND_ERROR_CODE, createFlatRequest } from '@sa/axios';
import { useAuthStore } from '@/store/modules/auth';
import { $t } from '@/locales';
import { getServiceBaseURL } from '@/utils/service';
import { getAuthorization, getLanguage, handleExpiredRequest, showErrorMsg } from './shared';
import type { RequestInstanceState } from './type';

const isHttpProxy = import.meta.env.DEV && import.meta.env.VITE_HTTP_PROXY === 'Y';
const { baseURL } = getServiceBaseURL(import.meta.env, isHttpProxy);
const refreshTokenUrl = '/api/v1/auth/refreshToken';

function isRefreshTokenRequest(url?: string) {
  return url?.split('?')[0] === refreshTokenUrl;
}

export const request = createFlatRequest<App.Service.Response, RequestInstanceState>(
  {
    baseURL,
    headers: {},
    // The authentication middleware returns a bare HTTP 401 when an access
    // token has expired. Let it reach `onBackendFail` so the refresh flow can
    // renew the session instead of treating every expired access token as a
    // logout.
    validateStatus(status) {
      return (status >= 200 && status < 300) || status === 401;
    }
  },
  {
    async onRequest(config) {
      const Authorization = getAuthorization();
      const language = getLanguage();

      Object.assign(config.headers, { Authorization, 'Accept-Language': language });

      return config;
    },
    isBackendSuccess(response) {
      // when the backend response code matches the configured success code, the request is successful
      // to change this logic by yourself, you can modify the `VITE_SERVICE_SUCCESS_CODE` in `.env` file
      return (
        response.status >= 200 &&
        response.status < 300 &&
        String(response.data?.code) === import.meta.env.VITE_SERVICE_SUCCESS_CODE
      );
    },
    async onBackendFail(response, instance) {
      const authStore = useAuthStore();
      const isRefreshRequest = isRefreshTokenRequest(response.config.url);
      const responseCode = String(response.data?.code || '');

      function handleLogout() {
        authStore.resetStore();
      }

      function logoutAndCleanup() {
        handleLogout();
        window.removeEventListener('beforeunload', handleLogout);

        request.state.errMsgStack = request.state.errMsgStack.filter(msg => msg !== response.data?.msg);
      }

      async function retryWithRefreshedToken() {
        const success = await handleExpiredRequest(request.state);
        if (!success) return null;

        const Authorization = getAuthorization();
        Object.assign(response.config.headers, { Authorization });

        return instance.request(response.config) as Promise<AxiosResponse>;
      }

      function showModalLogout() {
        const modalLogoutCodes = import.meta.env.VITE_SERVICE_MODAL_LOGOUT_CODES?.split(',') || [];
        const message = response.data?.msg || $t('common.error');
        if (!modalLogoutCodes.includes(responseCode) || request.state.errMsgStack?.includes(message)) return false;

        request.state.errMsgStack = [...(request.state.errMsgStack || []), message];
        window.addEventListener('beforeunload', handleLogout);
        window.$dialog?.error({
          title: $t('common.error'),
          content: message,
          positiveText: $t('common.confirm'),
          maskClosable: false,
          closeOnEsc: false,
          onPositiveClick: logoutAndCleanup,
          onClose: logoutAndCleanup
        });

        return true;
      }

      async function retryExpiredBackendResponse() {
        const expiredTokenCodes = import.meta.env.VITE_SERVICE_EXPIRED_TOKEN_CODES?.split(',') || [];
        if (!expiredTokenCodes.includes(responseCode)) return null;

        return retryWithRefreshedToken();
      }

      // Do not attempt to refresh a refresh-token request. Its failure is
      // handled by `handleRefreshToken`, which clears the local session once.
      if (isRefreshRequest) return null;

      // The auth middleware intentionally replies with a raw HTTP 401 for an
      // expired access token, so there is no application response code here.
      if (response.status === 401) {
        return retryWithRefreshedToken();
      }

      // when the backend response code is in `logoutCodes`, it means the user will be logged out and redirected to login page
      const logoutCodes = import.meta.env.VITE_SERVICE_LOGOUT_CODES?.split(',') || [];
      if (logoutCodes.includes(responseCode)) {
        handleLogout();
        return null;
      }

      // when the backend response code is in `modalLogoutCodes`, it means the user will be logged out by displaying a modal
      if (showModalLogout()) return null;

      // when the backend response code is in `expiredTokenCodes`, it means the token is expired, and refresh token
      // the api `refreshToken` can not return error code in `expiredTokenCodes`, otherwise it will be a dead loop, should return `logoutCodes` or `modalLogoutCodes`
      return retryExpiredBackendResponse();
    },
    transformBackendResponse(response) {
      return response.data.data;
    },
    onError(error) {
      // when the request is fail, you can show error message

      let message = error.message;
      let backendErrorCode = '';

      const httpStatus = error.response?.status ?? error.status;

      if (httpStatus === 401) {
        const authStore = useAuthStore();
        authStore.resetStore();
      }

      // get backend error message and code
      if (error.code === BACKEND_ERROR_CODE) {
        message = error.response?.data?.msg || message;
        backendErrorCode = String(error.response?.data?.code || '');
      }

      if (httpStatus === 401) {
        message = $t('request.unauthorized');
      } else if (httpStatus === 403) {
        message = $t('request.forbidden');
      }

      // the error message is displayed in the modal
      const modalLogoutCodes = import.meta.env.VITE_SERVICE_MODAL_LOGOUT_CODES?.split(',') || [];
      if (modalLogoutCodes.includes(backendErrorCode)) {
        return;
      }

      // when the token is expired, refresh token and retry request, so no need to show error message
      const expiredTokenCodes = import.meta.env.VITE_SERVICE_EXPIRED_TOKEN_CODES?.split(',') || [];
      if (expiredTokenCodes.includes(backendErrorCode)) {
        return;
      }

      showErrorMsg(request.state, message);
    }
  }
);
