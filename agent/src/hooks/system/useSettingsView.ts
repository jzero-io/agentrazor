import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

export type SettingsSection = 'appearance' | 'api-keys' | 'archives';

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
      if (route.path === '/settings/api-keys') return 'api-keys';
      if (route.path === '/settings/archives') return 'archives';
      return 'appearance';
    },
    set(value) {
      const paths: Record<SettingsSection, string> = {
        appearance: '/settings/appearance',
        'api-keys': '/settings/api-keys',
        archives: '/settings/archives'
      };
      void router.push(paths[value]);
    }
  });

  const visible = computed<boolean>({
    get() {
      return route.path === '/settings/appearance' || route.path === '/settings/api-keys' || route.path === '/settings/archives';
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
    const path = id ? `/conversation/${encodeURIComponent(id)}` : '/';
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
