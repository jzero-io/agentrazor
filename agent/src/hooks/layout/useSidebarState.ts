import { computed, onBeforeUnmount, onMounted, watch, type Ref } from 'vue';
import { ref } from 'vue';
import type { ConversationGroup } from '../business/useConversationGroups';

export interface SidebarViewState {
  sidebarCollapsed?: boolean;
  sidebarWidth?: number;
  pinnedExpanded?: boolean;
  groupsExpanded?: boolean;
  conversationsExpanded?: boolean;
  collapsedGroups?: Record<string, boolean>;
}

const SIDEBAR_VIEW_KEY = 'agentrazor_sidebar_view';
const DEFAULT_SIDEBAR_WIDTH = 320;
const MIN_SIDEBAR_WIDTH = 280;

export function loadSidebarViewState(): SidebarViewState {
  try {
    return JSON.parse(localStorage.getItem(SIDEBAR_VIEW_KEY) || '{}') as SidebarViewState;
  } catch {
    return {};
  }
}

interface UseSidebarStateOptions {
  savedSidebarView: SidebarViewState;
  conversationGroups: Ref<ConversationGroup[]>;
}

export function useSidebarState(options: UseSidebarStateOptions) {
  const sidebarCollapsed = ref(options.savedSidebarView.sidebarCollapsed ?? false);
  const sidebarWidth = ref(Math.max(MIN_SIDEBAR_WIDTH, options.savedSidebarView.sidebarWidth || DEFAULT_SIDEBAR_WIDTH));
  const sidebarHoverOpen = ref(false);
  const mobileSidebarOpen = ref(false);
  const pinnedExpanded = ref(options.savedSidebarView.pinnedExpanded ?? true);
  const groupsExpanded = ref(options.savedSidebarView.groupsExpanded ?? true);
  const conversationsExpanded = ref(options.savedSidebarView.conversationsExpanded ?? true);
  const appShellStyle = computed(() => ({ '--sidebar-width': `${sidebarWidth.value}px` }));
  const sidebarExpanded = computed(() => !sidebarCollapsed.value || sidebarHoverOpen.value || mobileSidebarOpen.value);
  let sidebarResizeStart: { x: number; width: number } | null = null;

  function closeMobileSidebar() {
    mobileSidebarOpen.value = false;
  }

  function openMobileSidebar() {
    if (sidebarCollapsed.value) sidebarCollapsed.value = false;
    sidebarHoverOpen.value = false;
    mobileSidebarOpen.value = true;
  }

  function openSidebarHover() {
    if (!sidebarCollapsed.value || window.matchMedia('(max-width: 720px)').matches) return;
    sidebarHoverOpen.value = true;
  }

  function closeSidebarHover() {
    if (!sidebarCollapsed.value) return;
    sidebarHoverOpen.value = false;
  }

  function toggleSidebarPinned() {
    if (sidebarCollapsed.value) {
      sidebarCollapsed.value = false;
      sidebarHoverOpen.value = false;
      return;
    }
    sidebarCollapsed.value = true;
  }

  function expandSidebar() {
    sidebarCollapsed.value = false;
    sidebarHoverOpen.value = false;
  }

  function startSidebarResize(event: PointerEvent) {
    if (sidebarCollapsed.value || window.matchMedia('(max-width: 720px)').matches) return;
    sidebarResizeStart = { x: event.clientX, width: sidebarWidth.value };
    document.body.classList.add('sidebar-resizing');
    event.preventDefault();
  }

  function resizeSidebar(event: PointerEvent) {
    if (!sidebarResizeStart) return;
    sidebarWidth.value = Math.max(
      MIN_SIDEBAR_WIDTH,
      sidebarResizeStart.width + event.clientX - sidebarResizeStart.x
    );
  }

  function stopSidebarResize() {
    if (!sidebarResizeStart) return;
    sidebarResizeStart = null;
    document.body.classList.remove('sidebar-resizing');
  }

  onMounted(() => {
    window.addEventListener('pointermove', resizeSidebar);
    window.addEventListener('pointerup', stopSidebarResize);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('pointermove', resizeSidebar);
    window.removeEventListener('pointerup', stopSidebarResize);
    stopSidebarResize();
  });

  watch(
    [sidebarCollapsed, sidebarWidth, pinnedExpanded, groupsExpanded, conversationsExpanded, options.conversationGroups],
    () => {
      const collapsedGroups = Object.fromEntries(
        options.conversationGroups.value.map(group => [group.id, group.collapsed])
      );
      localStorage.setItem(SIDEBAR_VIEW_KEY, JSON.stringify({
        sidebarCollapsed: sidebarCollapsed.value,
        sidebarWidth: sidebarWidth.value,
        pinnedExpanded: pinnedExpanded.value,
        groupsExpanded: groupsExpanded.value,
        conversationsExpanded: conversationsExpanded.value,
        collapsedGroups
      } satisfies SidebarViewState));
    },
    { deep: true }
  );

  return {
    sidebarCollapsed,
    sidebarWidth,
    sidebarHoverOpen,
    mobileSidebarOpen,
    pinnedExpanded,
    groupsExpanded,
    conversationsExpanded,
    appShellStyle,
    sidebarExpanded,
    closeMobileSidebar,
    openMobileSidebar,
    openSidebarHover,
    closeSidebarHover,
    toggleSidebarPinned,
    expandSidebar,
    startSidebarResize
  };
}
