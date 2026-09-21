import { GetPluginAdminLocales } from '@/service/api';
import { mergeRuntimeLocales } from '.';

/** Load all plugin languages during auth-route initialization without blocking the shell on failure. */
export async function loadPluginAdminLocales() {
  try {
    const { data, error } = await GetPluginAdminLocales();
    if (error || !data?.locales) return;

    mergeRuntimeLocales(data.locales);
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to load Admin plugin locales', error);
  }
}
