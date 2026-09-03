import { request } from './request';
import type { LoginResponse, UserInfo } from './types';

export const authApi = {
  pwdLogin(username: string, password: string) {
    return request<LoginResponse>('/api/v1/auth/pwd-login', {
      method: 'POST',
      body: JSON.stringify({ username, password })
    });
  },
  sendEmailVerificationCode(email: string) {
    const params = new URLSearchParams({ email, verificationType: 'email' });
    return request<{ verificationUuid: string }>('/api/v1/auth/sendVerificationCode?' + params.toString());
  },
  codeLogin(email: string, verificationUuid: string, verificationCode: string) {
    return request<LoginResponse>('/api/v1/auth/code-login', {
      method: 'POST',
      body: JSON.stringify({ email, verificationUuid, verificationCode })
    });
  },
  getUserInfo() {
    return request<UserInfo>('/api/v1/auth/getUserInfo');
  }
};
