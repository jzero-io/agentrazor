import type { App } from 'vue';
import { createI18n } from 'vue-i18n';
import { localStg } from '@/utils/storage';
import messages, { getCoreLocaleMessages, mergeLocaleMessages } from './locale';

const i18n = createI18n({
  locale: localStg.get('lang') || 'zh-CN',
  fallbackLocale: 'en-US',
  messages,
  legacy: false
});

/**
 * Setup plugin i18n
 *
 * @param app
 */
export function setupI18n(app: App) {
  app.use(i18n);
}

export const $t = i18n.global.t as App.I18n.$T;

export function translateOr(i18nKey: string | null | undefined, fallback: string) {
  if (!i18nKey) return fallback || '';

  const currentLocale = i18n.global.locale.value;
  const hasTranslation = i18n.global.te(i18nKey, currentLocale) || i18n.global.te(i18nKey, 'en-US');

  if (!hasTranslation) return fallback || i18nKey || '';

  return i18n.global.t(i18nKey);
}

/** Atomically replace runtime plugin messages using immutable core locale baselines. */
export function mergeRuntimeLocales(runtimeLocales: Api.Plugin.AdminLocalesResponse['locales']) {
  const supportedLocales: App.I18n.LangType[] = ['zh-CN', 'en-US'];
  const pendingLocales: [App.I18n.LangType, App.I18n.Schema][] = [];

  supportedLocales.forEach(locale => {
    const coreMessages = getCoreLocaleMessages(locale);
    const runtimeMessages = runtimeLocales[locale] || {};
    const nextMessages = mergeLocaleMessages(coreMessages, runtimeMessages, {
      sourceFile: `runtime:${locale}`
    }) as App.I18n.Schema;

    pendingLocales.push([locale, nextMessages]);
  });

  pendingLocales.forEach(([locale, nextMessages]) => {
    i18n.global.setLocaleMessage(locale, nextMessages);
  });
}

export function setLocale(locale: App.I18n.LangType) {
  i18n.global.locale.value = locale;
}
