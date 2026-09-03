import { computed, onBeforeUnmount, onMounted, reactive, ref, watch, type Ref } from 'vue';
import hljs from 'highlight.js/lib/common';
import type { WorkspaceEntry, WorkspaceFileBlob, WorkspaceFileContent } from '../../service/api';
import { useWorkspaceFileTree } from './useWorkspaceFileTree';

export interface WorkspaceDescriptor {
  type: 'workspace';
  title: string;
  url: string;
}

export interface WorkspaceTab extends WorkspaceDescriptor {
  tabId: string;
  order: number;
}

export interface FilePreview extends WorkspaceFileContent {
  kind: 'text' | 'image' | 'unsupported' | 'loading';
  language: string;
  markdown: boolean;
  objectUrl?: string;
  placeholder?: boolean;
  size?: number;
}

export interface FilePreviewTab extends FilePreview {
  tabId: string;
  order: number;
}

interface FileTab {
  tabId: string;
  path: string;
  order: number;
}

interface RightPanelConversationState {
  visible?: boolean;
  kind?: 'workspace' | 'file';
  expanded?: boolean;
  workspaceTabs?: WorkspaceTab[];
  activeWorkspaceTabId?: string;
  fileTabs?: FileTab[];
  activeFileTabId?: string;
}

interface RightPanelViewState {
  workspaceWidth?: number | null;
  conversations?: Record<string, RightPanelConversationState>;
}

const RIGHT_PANEL_VIEW_KEY = 'agentrazor_right_panel_view';

function createTabId() {
  return globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function sanitizeRightPanelState(value: RightPanelConversationState | undefined | null): RightPanelConversationState | null {
  if (!value || (value.kind && value.kind !== 'workspace' && value.kind !== 'file')) return null;
  const state: RightPanelConversationState = {
    visible: Boolean(value.visible),
    kind: value.kind,
    expanded: Boolean(value.expanded)
  };
  const workspaceTabs = Array.isArray(value.workspaceTabs)
    ? value.workspaceTabs
      .filter(workspace => workspace?.type === 'workspace' && workspace.title && workspace.url)
      .map((workspace, index) => ({
        type: 'workspace' as const,
        title: workspace.title,
        url: workspace.url,
        tabId: typeof workspace.tabId === 'string' && workspace.tabId ? workspace.tabId : createTabId(),
        order: typeof workspace.order === 'number' ? workspace.order : index
      }))
    : [];
  const activeWorkspaceTabId = typeof value.activeWorkspaceTabId === 'string'
    ? value.activeWorkspaceTabId
    : workspaceTabs[0]?.tabId;
  if (workspaceTabs.length && activeWorkspaceTabId && workspaceTabs.some(workspace => workspace.tabId === activeWorkspaceTabId)) {
    state.workspaceTabs = workspaceTabs;
    state.activeWorkspaceTabId = activeWorkspaceTabId;
  }
  const fileTabs = Array.isArray(value.fileTabs)
    ? value.fileTabs
      .filter(tab => typeof tab?.path === 'string' && tab.path.trim())
      .map((tab, index) => ({
        tabId: typeof tab.tabId === 'string' && tab.tabId ? tab.tabId : createTabId(),
        path: tab.path,
        order: typeof tab.order === 'number' ? tab.order : workspaceTabs.length + index
      }))
    : [];
  const activeFileTabId = typeof value.activeFileTabId === 'string' ? value.activeFileTabId : fileTabs[0]?.tabId;
  if (fileTabs.length && activeFileTabId && fileTabs.some(tab => tab.tabId === activeFileTabId)) {
    state.fileTabs = fileTabs;
    state.activeFileTabId = activeFileTabId;
  }
  if (state.kind === 'workspace' && !state.workspaceTabs?.length) {
    if (state.fileTabs?.length) state.kind = 'file';
    else return null;
  }
  if (!state.kind && (state.workspaceTabs?.length || state.fileTabs?.length)) {
    state.kind = state.workspaceTabs?.length ? 'workspace' : 'file';
  }
  if (!state.kind && !state.visible) return null;
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

const IMAGE_EXTENSIONS = new Set(['apng', 'avif', 'bmp', 'gif', 'ico', 'jpeg', 'jpg', 'png', 'svg', 'webp']);
const TEXT_EXTENSIONS = new Set([
  'api', 'c', 'cc', 'conf', 'cpp', 'css', 'csv', 'dockerfile', 'env', 'go', 'h', 'html', 'java', 'js', 'json',
  'jsx', 'log', 'md', 'markdown', 'mod', 'proto', 'py', 'rs', 'sh', 'sql', 'sum', 'toml', 'ts', 'tsx', 'txt',
  'vue', 'xml', 'yaml', 'yml'
]);

function extensionFromName(name: string) {
  const lower = name.toLowerCase();
  return lower.includes('.') ? lower.slice(lower.lastIndexOf('.') + 1) : '';
}

function isImageFile(name: string, contentType = '') {
  return contentType.startsWith('image/') || IMAGE_EXTENSIONS.has(extensionFromName(name));
}

function isTextFile(name: string, contentType = '') {
  if (contentType.startsWith('text/')) return true;
  if (/json|xml|yaml|toml|javascript|typescript/.test(contentType)) return true;
  return TEXT_EXTENSIONS.has(extensionFromName(name));
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
  fetchBlob: (path: string) => Promise<WorkspaceFileBlob>;
  fetchEntries: (conversationId: string) => Promise<WorkspaceEntry[]>;
  containerRef?: Ref<HTMLElement | null>;
  onError?: (error: unknown) => void;
}) {
  const saved = loadRightPanelViewState();
  const statesByConversation = reactive(new Map<string, RightPanelConversationState>(Object.entries(saved.conversations || {})));
  const filePreviewsByConversation = reactive(new Map<string, Map<string, FilePreview>>());
  const width = ref<number | null>(saved.workspaceWidth ?? null);
  const expanded = ref(false);
  const reloadVersion = ref(0);
  const loadingFilePaths = reactive(new Set<string>());
  const fileError = ref('');
  let resizeStart: { x: number; width: number } | null = null;
  const fileTree = useWorkspaceFileTree({
    selectedConversationId: options.selectedConversationId,
    draftConversationId: options.draftConversationId,
    fetchEntries: options.fetchEntries,
    onError: options.onError
  });

  const activeState = computed(() => statesByConversation.get(options.selectedConversationId.value) || null);
  const visible = computed({
    get: () => Boolean(activeState.value?.visible),
    set: value => {
      setState(options.selectedConversationId.value, { visible: value });
    }
  });
  const activeKind = computed(() => activeState.value?.kind || '');
  const activeWorkspaceTabs = computed(() => activeState.value?.workspaceTabs || []);
  const activeWorkspaceTabId = computed(() => activeState.value?.activeWorkspaceTabId || '');
  const hasWorkspace = computed(() => Boolean(activeWorkspaceTabs.value.length));
  const hasFiles = computed(() => Boolean(options.selectedConversationId.value && options.selectedConversationId.value !== options.draftConversationId));
  const activeWorkspace = computed(() => {
    const state = activeState.value;
    return state?.kind === 'workspace'
      ? state.workspaceTabs?.find(workspace => workspace.tabId === state.activeWorkspaceTabId) || null
      : null;
  });
  const activeWorkspaceUrl = computed(() => activeWorkspace.value?.url || '');
  const activeFileTabId = computed(() => activeState.value?.activeFileTabId || '');
  const activeFilePath = computed(() => {
    const state = activeState.value;
    if (state?.kind !== 'file') return '';
    return state.fileTabs?.find(tab => tab.tabId === state.activeFileTabId)?.path || '';
  });
  const activeFileTabs = computed(() => {
    const state = activeState.value;
    if (!state) return [] as FilePreviewTab[];
    const previews = filePreviewsByConversation.get(options.selectedConversationId.value);
    return (state.fileTabs || [])
      .map(tab => {
        const preview = previews?.get(tab.path);
        return preview ? { ...preview, tabId: tab.tabId } : null;
      })
      .filter(Boolean) as FilePreviewTab[];
  });
  const activeFilePreview = computed(() => {
    if (!activeFilePath.value) return null;
    return filePreviewsByConversation.get(options.selectedConversationId.value)?.get(activeFilePath.value) || null;
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
    if (!file || file.kind !== 'text') return [] as Array<{ number: number; html: string }>;
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
  const fileLoading = computed(() => Boolean(activeFilePath.value && loadingFilePaths.has(activeFilePath.value)));
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

  function previewsForConversation(conversationId: string) {
    let previews = filePreviewsByConversation.get(conversationId);
    if (!previews) {
      previews = reactive(new Map<string, FilePreview>()) as Map<string, FilePreview>;
      filePreviewsByConversation.set(conversationId, previews);
    }
    return previews;
  }

  function nextTabOrder(state: RightPanelConversationState | undefined) {
    const orders = [
      ...(state?.workspaceTabs || []).map(tab => tab.order),
      ...(state?.fileTabs || []).map(tab => tab.order)
    ];
    return orders.length ? Math.max(...orders) + 1 : 0;
  }

  function fileNameFromPath(path: string) {
    return decodeURIComponent(path.split('?')[0] || '').split('/').filter(Boolean).pop() || '文件';
  }

  function previewBadge(file: FilePreview) {
    const language = file.language || 'text';
    if (file.kind === 'image') return 'IMG';
    if (language === 'typescript') return 'TS';
    if (language === 'javascript') return 'JS';
    if (language === 'markdown') return 'MD';
    if (language === 'yaml') return 'YML';
    if (language === 'text') return 'TXT';
    return language.slice(0, 3).toUpperCase();
  }

  function revokeFilePreview(conversationId: string, path?: string) {
    const previews = filePreviewsByConversation.get(conversationId);
    if (!previews) return;
    const paths = path ? [path] : [...previews.keys()];
    for (const itemPath of paths) {
      const preview = previews.get(itemPath);
      if (preview?.objectUrl) URL.revokeObjectURL(preview.objectUrl);
      previews.delete(itemPath);
    }
    if (!previews.size) filePreviewsByConversation.delete(conversationId);
  }

  function openWorkspace(workspace: WorkspaceDescriptor, forceNewTab = false) {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId) return;
    expanded.value = false;
    fileError.value = '';
    const state = statesByConversation.get(conversationId);
    const currentTabs = state?.workspaceTabs || [];
    const existing = !forceNewTab ? currentTabs.find(tab => tab.url === workspace.url) : null;
    const tab = existing || { ...workspace, tabId: createTabId(), order: nextTabOrder(state) };
    const workspaceTabs = existing ? currentTabs : [...currentTabs, tab];
    setState(conversationId, {
      visible: true,
      kind: 'workspace',
      expanded: false,
      workspaceTabs,
      activeWorkspaceTabId: tab.tabId
    });
  }

  function replaceWithWorkspace(workspace: WorkspaceDescriptor) {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId) return;
    expanded.value = false;
    fileError.value = '';
    const tab = { ...workspace, tabId: createTabId(), order: 0 };
    setState(conversationId, {
      visible: true,
      kind: 'workspace',
      expanded: false,
      workspaceTabs: [tab],
      activeWorkspaceTabId: tab.tabId
    });
  }

  function showLauncher() {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId || conversationId === options.draftConversationId) return;
    expanded.value = false;
    fileError.value = '';
    setState(conversationId, { visible: true, kind: undefined, expanded: false });
  }

  async function openFile(path: string, restoreExpanded = false, forceNewTab = false) {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId) return;
    expanded.value = restoreExpanded ? expanded.value : false;
    fileError.value = '';
    const state = statesByConversation.get(conversationId);
    const currentTabs = state?.fileTabs || [];
    const existing = !forceNewTab ? currentTabs.find(tab => tab.path === path) : null;
    const tab = existing || { tabId: createTabId(), path, order: nextTabOrder(state) };
    const fileTabs = existing ? currentTabs : [...currentTabs, tab];
    const previews = previewsForConversation(conversationId);
    if (!previews.has(path)) {
      const name = fileNameFromPath(path);
      previews.set(path, {
        path,
        name,
        content: '',
        contentType: '',
        kind: 'loading',
        language: languageFromFilename(name, ''),
        markdown: /\.md(?:own)?$/i.test(name),
        placeholder: true
      });
    }
    loadingFilePaths.add(path);
    setState(conversationId, {
      visible: true,
      kind: 'file',
      expanded: expanded.value,
      fileTabs,
      activeFileTabId: tab.tabId
    });
    try {
      const name = fileNameFromPath(path);
      if (isImageFile(name)) {
        const file = await options.fetchBlob(path);
        if (options.selectedConversationId.value !== conversationId) return;
        revokeFilePreview(conversationId, path);
        previewsForConversation(conversationId).set(path, {
          path: file.path,
          name: file.name,
          content: '',
          contentType: file.contentType,
          kind: 'image',
          language: languageFromFilename(file.name, file.contentType),
          markdown: false,
          objectUrl: URL.createObjectURL(file.blob),
          size: file.blob.size
        });
        return;
      }

      const file = await options.fetchFile(path);
      if (options.selectedConversationId.value !== conversationId) return;
      revokeFilePreview(conversationId, path);
      const textFile = isTextFile(file.name, file.contentType);
      previewsForConversation(conversationId).set(path, {
        ...file,
        kind: textFile ? 'text' : 'unsupported',
        language: languageFromFilename(file.name, file.contentType),
        markdown: /\.md(?:own)?$/i.test(file.name)
      });
    } catch (error) {
      if (options.selectedConversationId.value === conversationId) {
        revokeFilePreview(conversationId, path);
        fileError.value = error instanceof Error ? error.message : '文件读取失败';
        options.onError?.(error);
      }
    } finally {
      loadingFilePaths.delete(path);
    }
  }

  function collapse() {
    const conversationId = options.selectedConversationId.value;
    expanded.value = false;
    loadingFilePaths.clear();
    if (conversationId) setState(conversationId, { visible: false, expanded: false });
  }

  function toggleExpanded() {
    expanded.value = !expanded.value;
    setState(options.selectedConversationId.value, { expanded: expanded.value });
  }

  function switchToWorkspace() {
    const state = activeState.value;
    if (!state?.workspaceTabs?.length) return;
    fileError.value = '';
    setState(options.selectedConversationId.value, {
      visible: true,
      kind: 'workspace',
      activeWorkspaceTabId: state.activeWorkspaceTabId || state.workspaceTabs[0].tabId
    });
  }

  function selectWorkspace(tabId: string) {
    const state = activeState.value;
    if (!state?.workspaceTabs?.some(workspace => workspace.tabId === tabId)) return;
    fileError.value = '';
    setState(options.selectedConversationId.value, { visible: true, kind: 'workspace', activeWorkspaceTabId: tabId });
  }

  function closeWorkspace(tabId?: string) {
    const conversationId = options.selectedConversationId.value;
    const state = activeState.value;
    if (!conversationId || !state?.workspaceTabs?.length) return;
    if (!tabId) {
      setState(conversationId, { visible: true, kind: undefined, workspaceTabs: [], activeWorkspaceTabId: '' });
      return;
    }
    const closedIndex = state.workspaceTabs.findIndex(workspace => workspace.tabId === tabId);
    if (closedIndex < 0) return;
    const workspaceTabs = state.workspaceTabs.filter(workspace => workspace.tabId !== tabId);
    if (!workspaceTabs.length) {
      const hasFileTabs = Boolean(state.fileTabs?.length);
      setState(conversationId, {
        visible: hasFileTabs,
        kind: hasFileTabs ? 'file' : undefined,
        workspaceTabs: [],
        activeWorkspaceTabId: ''
      });
      return;
    }
    const activeWorkspaceTabId = state.activeWorkspaceTabId === tabId
      ? workspaceTabs[Math.min(closedIndex, workspaceTabs.length - 1)].tabId
      : state.activeWorkspaceTabId;
    setState(conversationId, { workspaceTabs, activeWorkspaceTabId, visible: true, kind: 'workspace' });
  }

  function switchToFiles() {
    const state = activeState.value;
    const activeTabId = state?.activeFileTabId || state?.fileTabs?.[0]?.tabId || '';
    const activePath = state?.fileTabs?.find(tab => tab.tabId === activeTabId)?.path || '';
    fileError.value = '';
    setState(options.selectedConversationId.value, { visible: true, kind: 'file', activeFileTabId: activeTabId });
    void fileTree.load();
    if (!activePath) return;
    const preview = filePreviewsByConversation.get(options.selectedConversationId.value)?.get(activePath);
    if (!preview || preview.placeholder) void openFile(activePath, true);
  }

  function reload() {
    const state = activeState.value;
    if (state?.kind === 'file') {
      void fileTree.load(true);
      if (activeFilePath.value) void openFile(activeFilePath.value, true);
      return;
    }
    reloadVersion.value += 1;
  }

  function selectFile(tabId: string) {
    const state = activeState.value;
    const tab = state?.kind === 'file' ? state.fileTabs?.find(item => item.tabId === tabId) : null;
    if (!tab) return;
    fileError.value = '';
    setState(options.selectedConversationId.value, { visible: true, kind: 'file', activeFileTabId: tabId });
    const preview = filePreviewsByConversation.get(options.selectedConversationId.value)?.get(tab.path);
    if (!preview || preview.placeholder) void openFile(tab.path, true);
  }
  function openTreeFile(relativePath: string, forceNewTab = false) {
    const path = safeWorkspaceFileURL(options.selectedConversationId.value, relativePath);
    if (path) void openFile(path, false, forceNewTab);
  }


  function reorderFile(fromTabId: string, toTabId: string) {
    const state = activeState.value;
    if (state?.kind !== 'file' || fromTabId === toTabId) return;
    const fileTabs = (state.fileTabs || []).map(tab => ({ ...tab }));
    const from = fileTabs.find(tab => tab.tabId === fromTabId);
    const to = fileTabs.find(tab => tab.tabId === toTabId);
    if (!from || !to) return;
    [from.order, to.order] = [to.order, from.order];
    setState(options.selectedConversationId.value, { fileTabs });
  }

  function closeFile(tabId = activeFileTabId.value) {
    const conversationId = options.selectedConversationId.value;
    const state = activeState.value;
    if (!conversationId || !state?.fileTabs?.length || !tabId) return;
    const closedIndex = (state.fileTabs || []).findIndex(tab => tab.tabId === tabId);
    if (closedIndex < 0) return;
    const closedTab = state.fileTabs![closedIndex];
    const fileTabs = state.fileTabs!.filter(tab => tab.tabId !== tabId);
    if (!fileTabs.some(tab => tab.path === closedTab.path)) revokeFilePreview(conversationId, closedTab.path);
    fileError.value = '';
    if (!fileTabs.length) {
      const hasWorkspaceTabs = Boolean(state.workspaceTabs?.length);
      setState(conversationId, {
        visible: hasWorkspaceTabs,
        kind: hasWorkspaceTabs ? 'workspace' : undefined,
        fileTabs: [],
        activeFileTabId: ''
      });
      return;
    }
    const activeFileTabId = state.activeFileTabId === tabId
      ? fileTabs[Math.min(closedIndex, fileTabs.length - 1)].tabId
      : state.activeFileTabId;
    setState(conversationId, { fileTabs, activeFileTabId, visible: true, kind: 'file' });
  }

  function restore(conversationId: string) {
    const state = statesByConversation.get(conversationId);
    expanded.value = Boolean(state?.expanded);
    loadingFilePaths.clear();
    fileError.value = '';
    if (state?.visible && state.kind === 'file') void fileTree.load();
    if (state?.visible && state.fileTabs?.length) {
      const previews = previewsForConversation(conversationId);
      for (const { path } of state.fileTabs || []) {
        if (!previews.has(path)) {
          const name = fileNameFromPath(path);
          previews.set(path, {
            path,
            name,
            content: '',
            contentType: '',
            kind: 'loading',
            language: languageFromFilename(name, ''),
            markdown: /\.md(?:own)?$/i.test(name),
            placeholder: true
          });
        }
      }
      if (state.kind === 'file' && state.activeFileTabId) {
        const activePath = state.fileTabs.find(tab => tab.tabId === state.activeFileTabId)?.path;
        if (activePath && previews.get(activePath)?.kind === 'loading') void openFile(activePath, true);
      }
    }
  }

  function removeConversation(conversationId: string) {
    statesByConversation.delete(conversationId);
    revokeFilePreview(conversationId);
    fileTree.removeConversation(conversationId);
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
    for (const id of filePreviewsByConversation.keys()) revokeFilePreview(id);
  });

  return {
    width,
    expanded,
    reloadVersion,
    visible,
    activeKind,
    hasWorkspace,
    hasFiles,
    activeWorkspaceTabs,
    activeWorkspaceTabId,
    activeWorkspaceUrl,
    activeWorkspace,
    activeFilePath,
    activeFileTabId,
    activeFileTabs,
    activeFilePreview,
    fileLoading,
    fileError,
    fileTree: fileTree.tree,
    fileTreeExpandedPaths: fileTree.expandedPaths,
    fileTreeLoading: fileTree.loading,
    fileTreeLoaded: fileTree.loaded,
    fileTreeError: fileTree.error,
    toggleFileTreeDirectory: fileTree.toggleDirectory,
    title,
    filePath,
    fileBreadcrumbs,
    fileBadge,
    fileLines,
    previewBadge,
    panelStyle,
    normalizeFilePath,
    displayWorkspaceFilePath,
    displayWorkspaceProcessPath: (path: string) => displayWorkspaceProcessPath(path, options.selectedConversationId.value),
    openWorkspace,
    replaceWithWorkspace,
    showLauncher,
    openFile,
    switchToWorkspace,
    selectWorkspace,
    closeWorkspace,
    switchToFiles,
    selectFile,
    reorderFile,
    closeFile,
    collapse,
    openTreeFile,
    toggleExpanded,
    reload,
    restore,
    removeConversation,
    startResize
  };
}
