import { request } from '../request';

/** Get every Admin plugin locale bundle. */
export function GetPluginAdminLocales() {
  return request<Api.Plugin.AdminLocalesResponse>({
    url: '/api/v1/plugin/admin-locales',
    timeout: 3000,
    silent: true
  });
}
