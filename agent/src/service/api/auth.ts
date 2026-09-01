import { request } from './request';
import type { LoginResponse, UserInfo } from './types';

export const authApi = {
  pwdLogin(username: string, password: string) {
    return request<LoginResponse>('/api/v1/auth/pwd-login', {
      method: 'POST',
      body: JSON.stringify({ username, password })
    });
  },
  getUserInfo() {
    return request<UserInfo>('/api/v1/auth/getUserInfo');
  }
};
