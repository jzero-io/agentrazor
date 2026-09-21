import zhCN from './langs/zh-cn';
import enUS from './langs/en-us';

export type LocaleMessages = Record<string, unknown>;

interface LocaleMergeContext {
  sourceFile: string;
  parentKey?: string;
}

function isLocaleMessages(value: unknown): value is LocaleMessages {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function assertSafeLocaleValue(value: unknown, sourceFile: string, parentKey = ''): void {
  if (Array.isArray(value)) {
    value.forEach((item, index) => assertSafeLocaleValue(item, sourceFile, `${parentKey}[${index}]`));
    return;
  }

  if (!isLocaleMessages(value)) return;

  Object.entries(value).forEach(([key, child]) => {
    if (key === '__proto__' || key === 'constructor' || key === 'prototype') {
      throw new Error(`Plugin locale ${sourceFile} contains an unsafe key: ${key}`);
    }

    const keyPath = parentKey ? `${parentKey}.${key}` : key;
    assertSafeLocaleValue(child, sourceFile, keyPath);
  });
}

/** Merge a locale tree without allowing the source to replace an existing core value. */
export function mergeLocaleMessages(
  target: LocaleMessages,
  source: LocaleMessages,
  context: LocaleMergeContext
): LocaleMessages {
  const result = { ...target };
  const { sourceFile, parentKey = '' } = context;

  assertSafeLocaleValue(source, sourceFile, parentKey);

  for (const [key, value] of Object.entries(source)) {
    const keyPath = parentKey ? `${parentKey}.${key}` : key;
    const current = result[key];
    if (isLocaleMessages(current) && isLocaleMessages(value)) {
      result[key] = mergeLocaleMessages(current, value, { sourceFile, parentKey: keyPath });
    } else if (Object.hasOwn(result, key)) {
      throw new Error(`Plugin locale ${sourceFile} attempts to override ${keyPath}`);
    } else {
      result[key] = value;
    }
  }

  return result;
}

function cloneLocaleValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(cloneLocaleValue);

  if (isLocaleMessages(value)) {
    return Object.fromEntries(Object.entries(value).map(([key, child]) => [key, cloneLocaleValue(child)]));
  }

  return value;
}

function cloneLocaleMessages(messages: LocaleMessages): LocaleMessages {
  return cloneLocaleValue(messages) as LocaleMessages;
}

function freezeLocaleValue(value: unknown): void {
  if (Array.isArray(value)) {
    value.forEach(freezeLocaleValue);
    Object.freeze(value);
    return;
  }

  if (!isLocaleMessages(value)) return;

  Object.values(value).forEach(freezeLocaleValue);
  Object.freeze(value);
}

function createCoreLocaleSnapshot(messages: App.I18n.Schema): LocaleMessages {
  const snapshot = cloneLocaleMessages(messages as LocaleMessages);
  freezeLocaleValue(snapshot);
  return snapshot;
}

/** Immutable baselines used to rebuild locale state on every runtime refresh. */
const coreLocaleSnapshots: Readonly<Record<App.I18n.LangType, LocaleMessages>> = Object.freeze({
  'zh-CN': createCoreLocaleSnapshot(zhCN),
  'en-US': createCoreLocaleSnapshot(enUS)
});

export function getCoreLocaleMessages(locale: App.I18n.LangType): LocaleMessages {
  return cloneLocaleMessages(coreLocaleSnapshots[locale]);
}

const locales: Record<App.I18n.LangType, App.I18n.Schema> = {
  'zh-CN': getCoreLocaleMessages('zh-CN') as App.I18n.Schema,
  'en-US': getCoreLocaleMessages('en-US') as App.I18n.Schema
};

export default locales;
