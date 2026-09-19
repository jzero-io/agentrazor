import zhCN from './langs/zh-cn';
import enUS from './langs/en-us';

type LocaleMessages = Record<string, unknown>;

interface LocaleMergeContext {
  sourceFile: string;
  parentKey?: string;
}

function isLocaleMessages(value: unknown): value is LocaleMessages {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * Merge compile-time plugin locale bundles into the Admin locale tree.
 *
 * Each plugin owns files at `server/plugins/<plugin>/admin/locales/<lang>.json`. The Admin shell only supplies the
 * common loader; it contains no plugin copy.
 */
function mergeLocaleMessages(
  target: LocaleMessages,
  source: LocaleMessages,
  context: LocaleMergeContext
): LocaleMessages {
  const result = { ...target };
  const { sourceFile, parentKey = '' } = context;

  for (const [key, value] of Object.entries(source)) {
    if (key === '__proto__' || key === 'constructor' || key === 'prototype') {
      throw new Error(`Plugin locale ${sourceFile} contains an unsafe key: ${key}`);
    }

    const keyPath = parentKey ? `${parentKey}.${key}` : key;
    const current = result[key];
    if (isLocaleMessages(current) && isLocaleMessages(value)) {
      result[key] = mergeLocaleMessages(current, value, { sourceFile, parentKey: keyPath });
    } else {
      if (Object.hasOwn(result, key)) {
        throw new Error(`Plugin locale ${sourceFile} attempts to override ${keyPath}`);
      }
      result[key] = value;
    }
  }

  return result;
}

const pluginLocaleBundles = import.meta.glob('../../../server/plugins/*/admin/locales/*.json', {
  eager: true,
  import: 'default'
}) as Record<string, LocaleMessages>;

function withPluginLocales(language: App.I18n.LangType, coreLocale: App.I18n.Schema): App.I18n.Schema {
  const suffix = `/locales/${language}.json`;

  return Object.entries(pluginLocaleBundles)
    .filter(([file]) => file.endsWith(suffix))
    .sort(([left], [right]) => left.localeCompare(right))
    .reduce<LocaleMessages>(
      (messages, [file, pluginLocale]) => mergeLocaleMessages(messages, pluginLocale, { sourceFile: file }),
      coreLocale
    ) as App.I18n.Schema;
}

const locales: Record<App.I18n.LangType, App.I18n.Schema> = {
  'zh-CN': withPluginLocales('zh-CN', zhCN),
  'en-US': withPluginLocales('en-US', enUS)
};

export default locales;
