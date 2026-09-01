import { ref, watch } from 'vue';

export type SettingsSection = 'appearance' | 'archives';

export function useSettingsView(options: {
  onOpen?: () => void;
}) {
  const visible = ref(false);
  const section = ref<SettingsSection>('appearance');
  const navExpanded = ref(true);
  const archiveQuery = ref('');
  const returnPath = ref('/');

  function syncUrl() {
    const path = visible.value
      ? section.value === 'archives' ? '/settings/archives' : '/settings/appearance'
      : returnPath.value || '/';
    window.history.replaceState(window.history.state, '', path);
  }

  function syncConversationUrl(id: string) {
    const path = id ? `/c/${encodeURIComponent(id)}` : '/';
    returnPath.value = path;
    if (visible.value) return;
    window.history.replaceState(window.history.state, '', path);
  }

  function restoreFromPath(pathname = window.location.pathname) {
    const settingsPath = pathname.match(/^\/settings(?:\/([^/]+))?/);
    if (!settingsPath) return false;
    section.value = settingsPath[1] === 'archives' ? 'archives' : 'appearance';
    visible.value = true;
    return true;
  }

  function openAppearance(currentPath = location.pathname) {
    returnPath.value = currentPath;
    section.value = 'appearance';
    navExpanded.value = true;
    visible.value = true;
    options.onOpen?.();
  }

  function openArchives() {
    section.value = 'archives';
    options.onOpen?.();
  }

  function close() {
    visible.value = false;
  }

  watch([visible, section], syncUrl);

  return {
    visible,
    section,
    navExpanded,
    archiveQuery,
    returnPath,
    syncConversationUrl,
    restoreFromPath,
    openAppearance,
    openArchives,
    close
  };
}
