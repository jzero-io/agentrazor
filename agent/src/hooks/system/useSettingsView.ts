import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

export type SettingsSection = 'appearance' | 'archives';

export function useSettingsView(options: {
  onOpen?: () => void;
}) {
  const route = useRoute();
  const router = useRouter();
  const navExpanded = ref(true);
  const archiveQuery = ref('');
  const returnPath = ref('/');

  const section = computed<SettingsSection>({
    get() {
      return route.path === '/settings/archives' ? 'archives' : 'appearance';
    },
    set(value) {
      void router.push(value === 'archives' ? '/settings/archives' : '/settings/appearance');
    }
  });

  const visible = computed<boolean>({
    get() {
      return route.path === '/settings/appearance' || route.path === '/settings/archives';
    },
    set(value) {
      if (value) openAppearance();
      else close();
    }
  });

  function isSettingsPath(path: string) {
    return path === '/settings' || path.startsWith('/settings/');
  }

  function rememberReturnPath(path = route.fullPath) {
    if (!isSettingsPath(path)) returnPath.value = path || '/';
  }

  function syncConversationUrl(id: string) {
    const path = id ? `/c/${encodeURIComponent(id)}` : '/';
    returnPath.value = path;
    if (!visible.value) void router.replace(path);
  }

  function restoreFromPath() {
    rememberReturnPath(route.fullPath);
    return visible.value;
  }

  function openAppearance(currentPath = route.fullPath) {
    rememberReturnPath(currentPath);
    navExpanded.value = true;
    void router.push('/settings/appearance');
  }

  function openArchives() {
    void router.push('/settings/archives');
  }

  function close() {
    void router.push(returnPath.value || '/');
  }

  watch(
    () => route.fullPath,
    path => {
      if (visible.value) options.onOpen?.();
      else rememberReturnPath(path);
    },
    { immediate: true }
  );

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
