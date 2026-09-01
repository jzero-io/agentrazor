import { computed, onBeforeUnmount, onMounted, reactive, ref, watch, type Ref } from 'vue';
import hljs from 'highlight.js/lib/common';
import type { WorkspaceFileContent } from '../../service/api';

export interface WorkspaceDescriptor {
  type: 'workspace';
  title: string;
  url: string;
}

export interface FilePreview extends WorkspaceFileContent {
  language: string;
  markdown: boolean;
}

interface RightPanelConversationState {
  visible?: boolean;
  kind?: 'workspace' | 'file';
  expanded?: boolean;
  workspace?: WorkspaceDescriptor;
  filePath?: string;
}

interface RightPanelViewState {
  workspaceWidth?: number | null;
  conversations?: Record<string, RightPanelConversationState>;
}

const RIGHT_PANEL_VIEW_KEY = 'agentrazor_right_panel_view';

function sanitizeRightPanelState(value: RightPanelConversationState | undefined | null): RightPanelConversationState | null {
  if (!value || value.kind !== 'workspace' && value.kind !== 'file') return null;
  const state: RightPanelConversationState = {
    visible: Boolean(value.visible),
    kind: value.kind,
    expanded: Boolean(value.expanded)
  };
  if (value.kind === 'workspace') {
    const workspace = value.workspace;
    if (!workspace || workspace.type !== 'workspace' || !workspace.title || !workspace.url) return null;
    state.workspace = { type: 'workspace', title: workspace.title, url: workspace.url };
  } else {
    if (!value.filePath) return null;
    state.filePath = value.filePath;
  }
  return state;
}

function loadRightPanelViewState(): RightPanelViewState {
  try {
    const raw = JSON.parse(localStorage.getItem(RIGHT_PANEL_VIEW_KEY) || '{}') as RightPanelViewState;
    const conversations: Record<string, RightPanelConversationState> = {};
    for (const [id, value] of Object.entries(raw.conversations || {})) {
      const state = sanitizeRightPanelState(value);
      if (id && state) conversations[id] = state;
    }
    return {
      workspaceWidth: typeof raw.workspaceWidth === 'number' ? raw.workspaceWidth : null,
      conversations
    };
  } catch {
    return { workspaceWidth: null, conversations: {} };
  }
}

export function languageFromFilename(name: string, contentType: string) {
  const lower = name.toLowerCase();
  const ext = lower.includes('.') ? lower.slice(lower.lastIndexOf('.') + 1) : '';
  if (ext === 'api') return 'api';
  if (ext === 'yml' || ext === 'yaml') return 'yaml';
  if (ext === 'md' || ext === 'markdown') return 'markdown';
  if (ext === 'toml') return 'toml';
  if (ext === 'json') return 'json';
  if (ext === 'go') return 'go';
  if (ext === 'ts' || ext === 'tsx') return 'typescript';
  if (ext === 'js' || ext === 'jsx') return 'javascript';
  if (ext === 'vue') return 'vue';
  if (ext === 'sql') return 'sql';
  if (contentType.includes('html')) return 'html';
  return ext || 'text';
}

export function safeWorkspaceFileURL(conversationId: string, relativePath: string) {
  const clean = relativePath
    .replace(/\\/g, '/')
    .split('/')
    .filter(part => part && part !== '.')
    .join('/');
  if (!clean || clean.split('/').some(part => part === '..')) return '';
  if (!/\.[a-z0-9][a-z0-9_-]*$/i.test(clean)) return '';
  return `/dist/data/agentrazor-home/${encodeURIComponent(conversationId)}/${clean.split('/').map(encodeURIComponent).join('/')}`;
}

export function displayWorkspaceProcessPath(filePath: string, conversationId = '') {
  if (conversationId) {
    const marker = `/agentrazor-home/${conversationId}/`;
    const index = filePath.indexOf(marker);
    if (index >= 0) return filePath.slice(index + marker.length);
  }
  return filePath.replace(/^\/dist\/data\/agentrazor-home\/[^/]+\//, '');
}

export function useWorkspacePanel(options: {
  selectedConversationId: Ref<string>;
  draftConversationId: string;
  fetchFile: (path: string) => Promise<WorkspaceFileContent>;
  containerRef?: Ref<HTMLElement | null>;
  onError?: (error: unknown) => void;
}) {
  const saved = loadRightPanelViewState();
  const statesByConversation = reactive(new Map<string, RightPanelConversationState>(Object.entries(saved.conversations || {})));
  const filePreviewsByConversation = reactive(new Map<string, FilePreview>());
  const width = ref<number | null>(saved.workspaceWidth ?? null);
  const expanded = ref(false);
  const reloadVersion = ref(0);
  const fileLoading = ref(false);
  const fileError = ref('');
  let resizeStart: { x: number; width: number } | null = null;

  const activeState = computed(() => statesByConversation.get(options.selectedConversationId.value) || null);
  const visible = computed({
    get: () => Boolean(activeState.value?.visible),
    set: value => {
      setState(options.selectedConversationId.value, { visible: value });
    }
  });
  const activeWorkspace = computed(() => {
    const state = activeState.value;
    return state?.kind === 'workspace' ? state.workspace || null : null;
  });
  const activeFilePreview = computed(() => {
    const state = activeState.value;
    if (state?.kind !== 'file') return null;
    return filePreviewsByConversation.get(options.selectedConversationId.value) || null;
  });
  const title = computed(() => activeFilePreview.value?.name || activeWorkspace.value?.title || '');
  const filePath = computed(() => activeFilePreview.value ? displayWorkspaceFilePath(activeFilePreview.value) : '');
  const fileBreadcrumbs = computed(() => filePath.value.split('/').filter(Boolean));
  const fileBadge = computed(() => {
    const language = activeFilePreview.value?.language || 'text';
    if (language === 'typescript') return 'TS';
    if (language === 'javascript') return 'JS';
    if (language === 'markdown') return 'MD';
    if (language === 'yaml') return 'YML';
    if (language === 'text') return 'TXT';
    return language.slice(0, 3).toUpperCase();
  });
  const fileLines = computed(() => {
    const file = activeFilePreview.value;
    if (!file) return [] as Array<{ number: number; html: string }>;
    const rawLines = file.content.replace(/\r\n/g, '\n').split('\n');
    if (rawLines.length > 1 && rawLines[rawLines.length - 1] === '') rawLines.pop();
    const language = hljs.getLanguage(file.language) ? file.language : 'plaintext';
    return rawLines.map((line, index) => ({
      number: index + 1,
      html: line
        ? hljs.highlight(line, { language, ignoreIllegals: true }).value
        : '&nbsp;'
    }));
  });
  const panelStyle = computed(() => width.value
    ? { '--workspace-width': `${width.value}px` }
    : {});

  function persist() {
    const conversations: Record<string, RightPanelConversationState> = {};
    for (const [id, state] of statesByConversation.entries()) {
      const sanitized = sanitizeRightPanelState(state);
      if (sanitized) conversations[id] = sanitized;
    }
    localStorage.setItem(RIGHT_PANEL_VIEW_KEY, JSON.stringify({ workspaceWidth: width.value, conversations } satisfies RightPanelViewState));
  }

  function setState(conversationId: string, patch: Partial<RightPanelConversationState>) {
    if (!conversationId || conversationId === options.draftConversationId) return;
    const next = { ...(statesByConversation.get(conversationId) || {}), ...patch } as RightPanelConversationState;
    const sanitized = sanitizeRightPanelState(next);
    if (sanitized) statesByConversation.set(conversationId, sanitized);
    else statesByConversation.delete(conversationId);
    persist();
  }

  function normalizeFilePath(href: string) {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId) return '';
    const trimmed = href.trim();
    if (!trimmed || trimmed.startsWith('#')) return '';

    let pathname = trimmed;
    try {
      const url = new URL(trimmed, window.location.origin);
      if (url.protocol !== 'http:' && url.protocol !== 'https:' && url.protocol !== 'file:') return '';
      if (url.protocol !== 'file:' && url.origin !== window.location.origin && !url.pathname.includes('/agentrazor-home/')) return '';
      pathname = decodeURIComponent(url.pathname);
    } catch {
      pathname = trimmed.split('#')[0]?.split('?')[0] || '';
    }

    const marker = `/agentrazor-home/${conversationId}/`;
    const markerIndex = pathname.indexOf(marker);
    if (markerIndex >= 0) {
      return safeWorkspaceFileURL(conversationId, pathname.slice(markerIndex + marker.length));
    }

    if (pathname.startsWith('/')) return '';
    return safeWorkspaceFileURL(conversationId, pathname);
  }

  function openWorkspace(workspace: WorkspaceDescriptor) {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId) return;
    expanded.value = false;
    fileError.value = '';
    filePreviewsByConversation.delete(conversationId);
    setState(conversationId, { visible: true, kind: 'workspace', expanded: false, workspace });
  }

  async function openFile(path: string, restoreExpanded = false) {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId) return;
    expanded.value = restoreExpanded ? expanded.value : false;
    fileError.value = '';
    fileLoading.value = true;
    setState(conversationId, { visible: true, kind: 'file', expanded: expanded.value, filePath: path });
    try {
      const file = await options.fetchFile(path);
      if (options.selectedConversationId.value !== conversationId) return;
      filePreviewsByConversation.set(conversationId, {
        ...file,
        language: languageFromFilename(file.name, file.contentType),
        markdown: /\.md(?:own)?$/i.test(file.name)
      });
    } catch (error) {
      if (options.selectedConversationId.value === conversationId) {
        fileError.value = error instanceof Error ? error.message : '文件读取失败';
        options.onError?.(error);
      }
    } finally {
      if (options.selectedConversationId.value === conversationId) fileLoading.value = false;
    }
  }

  function collapse() {
    const conversationId = options.selectedConversationId.value;
    expanded.value = false;
    fileLoading.value = false;
    if (conversationId) setState(conversationId, { visible: false, expanded: false });
  }

  function toggleExpanded() {
    expanded.value = !expanded.value;
    setState(options.selectedConversationId.value, { expanded: expanded.value });
  }

  function reload() {
    const state = activeState.value;
    if (state?.kind === 'file' && state.filePath) {
      void openFile(state.filePath, true);
      return;
    }
    reloadVersion.value += 1;
  }

  function closeFile() {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId) return;
    filePreviewsByConversation.delete(conversationId);
    fileError.value = '';
    setState(conversationId, { visible: false });
  }

  function restore(conversationId: string) {
    const state = statesByConversation.get(conversationId);
    expanded.value = Boolean(state?.expanded);
    fileLoading.value = false;
    fileError.value = '';
    if (state?.visible && state.kind === 'file' && state.filePath && !filePreviewsByConversation.has(conversationId)) {
      void openFile(state.filePath, true);
    }
  }

  function removeConversation(conversationId: string) {
    statesByConversation.delete(conversationId);
    filePreviewsByConversation.delete(conversationId);
    persist();
  }

  function displayWorkspaceFilePath(file: FilePreview) {
    return displayWorkspaceProcessPath(file.path, options.selectedConversationId.value);
  }

  function startResize(event: PointerEvent) {
    if (window.matchMedia('(max-width: 720px)').matches) return;
    const panel = options.containerRef?.value?.querySelector<HTMLElement>('.workspace-panel');
    if (!panel) return;
    resizeStart = { x: event.clientX, width: panel.getBoundingClientRect().width };
    document.body.classList.add('workspace-resizing');
    event.preventDefault();
  }

  function resize(event: PointerEvent) {
    if (!resizeStart) return;
    const availableWidth = options.containerRef?.value?.clientWidth || window.innerWidth;
    const maxWidth = Math.max(360, availableWidth - 320);
    width.value = Math.min(
      maxWidth,
      Math.max(360, resizeStart.width + resizeStart.x - event.clientX)
    );
  }

  function stopResize() {
    if (!resizeStart) return;
    resizeStart = null;
    document.body.classList.remove('workspace-resizing');
  }

  watch(width, persist);

  onMounted(() => {
    window.addEventListener('pointermove', resize);
    window.addEventListener('pointerup', stopResize);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('pointermove', resize);
    window.removeEventListener('pointerup', stopResize);
    stopResize();
  });

  return {
    width,
    expanded,
    reloadVersion,
    visible,
    activeWorkspace,
    activeFilePreview,
    fileLoading,
    fileError,
    title,
    filePath,
    fileBreadcrumbs,
    fileBadge,
    fileLines,
    panelStyle,
    normalizeFilePath,
    displayWorkspaceFilePath,
    displayWorkspaceProcessPath: (path: string) => displayWorkspaceProcessPath(path, options.selectedConversationId.value),
    openWorkspace,
    openFile,
    closeFile,
    collapse,
    toggleExpanded,
    reload,
    restore,
    removeConversation,
    startResize
  };
}
