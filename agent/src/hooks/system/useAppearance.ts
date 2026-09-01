import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { darkTheme } from 'naive-ui';

export type Appearance = 'system' | 'light' | 'dark';

const APPEARANCE_KEY = 'agentrazor_appearance';

export const appearanceOptions: Array<{ key: Appearance; label: string; icon: string }> = [
  { key: 'system', label: '跟随系统', icon: 'solar:monitor-smartphone-linear' },
  { key: 'light', label: '浅色', icon: 'solar:sun-2-linear' },
  { key: 'dark', label: '深色', icon: 'solar:moon-linear' }
];

function loadAppearance(): Appearance {
  const saved = localStorage.getItem(APPEARANCE_KEY);
  return saved === 'light' || saved === 'dark' || saved === 'system' ? saved : 'system';
}

export function useAppearance() {
  const colorSchemeMedia = window.matchMedia('(prefers-color-scheme: dark)');
  const appearance = ref<Appearance>(loadAppearance());
  const systemDark = ref(colorSchemeMedia.matches);

  const isDarkAppearance = computed(() =>
    appearance.value === 'dark' || appearance.value === 'system' && systemDark.value
  );
  const activeTheme = computed(() => isDarkAppearance.value ? darkTheme : null);

  function setAppearance(value: Appearance) {
    appearance.value = value;
    localStorage.setItem(APPEARANCE_KEY, value);
  }

  function handleSystemAppearanceChange(event: MediaQueryListEvent) {
    systemDark.value = event.matches;
  }

  onMounted(() => {
    colorSchemeMedia.addEventListener('change', handleSystemAppearanceChange);
  });

  onBeforeUnmount(() => {
    colorSchemeMedia.removeEventListener('change', handleSystemAppearanceChange);
  });

  return {
    appearance,
    appearanceOptions,
    isDarkAppearance,
    activeTheme,
    setAppearance
  };
}
