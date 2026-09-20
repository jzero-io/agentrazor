import { localStg } from '@/utils/storage';
import { emitAdminAuthSessionChange } from '@/utils/auth-session';

/** Get token */
export function getToken() {
  return localStg.get('token') || '';
}

/** Clear auth storage */
export function clearAuthStorage() {
  localStg.remove('token');
  localStg.remove('refreshToken');
  emitAdminAuthSessionChange('clear');
}
