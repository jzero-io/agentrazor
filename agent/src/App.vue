<script setup lang="ts">
import { computed, defineComponent, h, nextTick, onBeforeUnmount, onMounted, onUpdated, reactive, ref, watch, type PropType } from 'vue';
import { Icon } from '@iconify/vue';
import MarkdownIt from 'markdown-it';
import hljs from 'highlight.js/lib/common';
import 'highlight.js/styles/github.css';
import githubDarkTheme from 'highlight.js/styles/github-dark.css?raw';
import {
  NButton,
  NConfigProvider,
  NDropdown,
  NEmpty,
  NImage,
  NInput,
  NModal,
  NScrollbar,
  NSpin,
  NTooltip,
  createDiscreteApi,
  darkTheme,
  zhCN,
  dateZhCN
} from 'naive-ui';
import { authApi, clearRefreshToken, clearToken, conversationApi, conversationGroupApi, getToken, setAuthErrorHandler, setRefreshToken, setToken } from './api';
import type { Conversation, ConversationDetail, StreamEvent, ThreadItem, Turn, UserInfo } from './api';
import {
  activityIcon,
  activityTitle,
  completedProcessSummary,
  formatTurnDuration,
  processDisplayItems,
  showCompletedProcessSummary,
  streamingProcessSummary,
  turnProcessItems,
  turnResultItems,
  type ProcessDisplayItem
} from './processDisplay';

interface SidebarViewState {
  sidebarCollapsed?: boolean;
  pinnedExpanded?: boolean;
  groupsExpanded?: boolean;
  conversationsExpanded?: boolean;
  collapsedGroups?: Record<string, boolean>;
}

interface TurnView {
  renderKey: string;
  turn: Turn;
  userItems: ThreadItem[];
  resultItems: ThreadItem[];
  processDisplays: ProcessDisplayItem[];
  processMode: 'none' | 'thinking' | 'processing' | 'completed';
  processSummary: string;
  showTailThinking: boolean;
  streaming: boolean;
}

interface WorkspaceDescriptor {
  type: 'workspace';
  title: string;
  url: string;
}

interface ParsedAgentMessage {
  markdown: string;
  workspaces: WorkspaceDescriptor[];
}

const SIDEBAR_VIEW_KEY = 'agentrazor_sidebar_view';
const CONVERSATION_LIST_DROP_TARGET = 'conversation-list';
const DRAFT_CONVERSATION_ID = '__draft_conversation__';

// 深色模式使用 GitHub Dark 主题：把主题选择器作用域限制在 [data-theme="dark"] 下
function scopeCssSelectors(css: string, scope: string): string {
  const rules: string[] = [];
  let buffer = '';
  for (const line of css.split('\n')) {
    buffer += `${line}\n`;
    if (!line.includes('}')) continue;
    const rule = buffer;
    buffer = '';
    const openBrace = rule.indexOf('{');
    if (openBrace < 0) {
      rules.push(rule);
      continue;
    }
    const selectors = rule.slice(0, openBrace);
    const body = rule.slice(openBrace);
    const scoped = selectors
      .split(',')
      .map(selector => `${scope} ${selector.trim()}`)
      .join(', ');
    rules.push(`${scoped}${body}`);
  }
  if (buffer.trim()) rules.push(buffer);
  return rules.join('');
}

const darkThemeStyle = document.createElement('style');
darkThemeStyle.textContent = scopeCssSelectors(githubDarkTheme, ':root[data-theme="dark"]');
document.head.appendChild(darkThemeStyle);

const markdown = new MarkdownIt({
  html: false,
  breaks: true,
  linkify: true
});
markdown.renderer.rules.link_open = (tokens, index, options, _env, renderer) => {
  tokens[index].attrSet('target', '_blank');
  tokens[index].attrSet('rel', 'noopener noreferrer');
  return renderer.renderToken(tokens, index, options);
};

markdown.renderer.rules.fence = (tokens, index) => {
  const token = tokens[index];
  const info = (token.info || '').trim();
  const lang = info.split(/\s+/)[0]?.toLowerCase() || '';
  if (lang === 'mermaid') {
    const source = markdown.utils.escapeHtml(token.content);
    return [
      `<div class="mermaid-block" data-mermaid-source="${source}">`,
      mermaidCopyButtonHtml(),
      '<div class="mermaid-placeholder">正在渲染 Mermaid 图表</div>',
      `<pre class="mermaid-source"><code>${source}</code></pre>`,
      '</div>'
    ].join('');
  }
  const label = lang ? hljs.getLanguage(lang)?.name || lang : 'text';
  let code = '';
  try {
    code = lang && hljs.getLanguage(lang)
      ? hljs.highlight(token.content, { language: lang, ignoreIllegals: true }).value
      : markdown.utils.escapeHtml(token.content);
  } catch {
    code = markdown.utils.escapeHtml(token.content);
  }
  return [
    '<div class="code-block">',
    `<div class="code-block-head"><span class="code-lang">${markdown.utils.escapeHtml(label)}</span></div>`,
    `<pre><code class="hljs${lang ? ` language-${lang}` : ''}">${code}</code></pre>`,
    '</div>'
  ].join('');
};

function renderMarkdown(content: string) {
  return markdown.render(content);
}

let mermaidRenderSeq = 0;
let mermaidModulePromise: Promise<typeof import('mermaid')> | undefined;

function mermaidCopyButtonHtml() {
  return '<button type="button" class="mermaid-copy-button" data-mermaid-copy>复制源码</button>';
}

function loadMermaid() {
  if (!mermaidModulePromise) mermaidModulePromise = import('mermaid');
  return mermaidModulePromise;
}

function currentMermaidTheme() {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'default';
}

function initMermaid(instance: typeof import('mermaid')['default']) {
  instance.initialize({
    startOnLoad: false,
    securityLevel: 'strict',
    theme: currentMermaidTheme()
  });
}

async function renderMermaidBlocks(root: HTMLElement | null) {
  if (!root) return;
  const theme = currentMermaidTheme();
  const blocks = Array.from(root.querySelectorAll<HTMLElement>('.mermaid-block'));
  if (!blocks.length) return;
  const { default: mermaid } = await loadMermaid();
  initMermaid(mermaid);
  for (const block of blocks) {
    const source = block.dataset.mermaidSource || '';
    if (!source.trim()) continue;
    if (block.dataset.renderedSource === source && block.dataset.renderedTheme === theme) continue;
    block.dataset.renderedSource = source;
    block.dataset.renderedTheme = theme;
    block.classList.remove('is-error');
    block.classList.add('is-rendering');
    try {
      const id = `agent-mermaid-${++mermaidRenderSeq}`;
      const { svg } = await mermaid.render(id, source);
      if (!block.isConnected) continue;
      block.innerHTML = `${mermaidCopyButtonHtml()}<div class="mermaid-diagram">${svg}</div>`;
      block.classList.remove('is-rendering');
    } catch (error) {
      if (!block.isConnected) continue;
      block.classList.remove('is-rendering');
      block.classList.add('is-error');
      block.innerHTML = [
        mermaidCopyButtonHtml(),
        '<div class="mermaid-error">Mermaid 图表渲染失败</div>',
        `<pre class="mermaid-source"><code>${markdown.utils.escapeHtml(source)}</code></pre>`
      ].join('');
    }
  }
}

async function writeClipboardText(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  textarea.style.top = '0';
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const copied = document.execCommand('copy');
  document.body.removeChild(textarea);
  if (!copied) throw new Error('复制失败');
}

async function copyMermaidSource(event: MouseEvent) {
  const target = event.target instanceof HTMLElement ? event.target : null;
  const button = target?.closest<HTMLButtonElement>('[data-mermaid-copy]');
  if (!button) return;
  const block = button.closest<HTMLElement>('.mermaid-block');
  const source = block?.dataset.mermaidSource || '';
  if (!source) return;
  event.preventDefault();
  event.stopPropagation();
  try {
    await writeClipboardText(source);
    const previous = button.textContent || '复制源码';
    button.textContent = '已复制';
    window.setTimeout(() => {
      if (button.isConnected) button.textContent = previous;
    }, 1200);
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '复制失败');
  }
}

function parseAgentMessage(content: string, streaming: boolean): ParsedAgentMessage {
  if (streaming) return { markdown: content, workspaces: [] };
  const workspaces: WorkspaceDescriptor[] = [];
  const markdown = content.replace(/```json\s*([\s\S]*?)```/gi, (block, raw: string) => {
    try {
      const value = JSON.parse(raw.trim()) as Partial<WorkspaceDescriptor>;
      if (value.type !== 'workspace' || typeof value.title !== 'string' || typeof value.url !== 'string') return block;
      workspaces.push({ type: 'workspace', title: value.title, url: value.url });
      return '';
    } catch {
      return block;
    }
  });
  return { markdown: markdown.trim(), workspaces };
}

function openWorkspace(workspace: WorkspaceDescriptor) {
  if (!selectedId.value) return;
  workspaceExpanded.value = false;
  pinnedSummaryOpen.value = false;
  activeWorkspace.value = workspace;
  workspaceVisible.value = true;
}

function collapseWorkspace() {
  workspaceExpanded.value = false;
  workspaceVisible.value = false;
}

function toggleRightPanel() {
  if (workspaceVisible.value || pinnedSummaryOpen.value) {
    pinnedSummaryOpen.value = false;
    collapseWorkspace();
    return;
  }
  pinnedSummaryOpen.value = true;
}

function toggleWorkspaceExpanded() {
  workspaceExpanded.value = !workspaceExpanded.value;
}

function reloadWorkspace() {
  workspaceReloadVersion.value += 1;
}

function openPinnedWorkspace(workspace: WorkspaceDescriptor) {
  openWorkspace(workspace);
}


function loadSidebarViewState(): SidebarViewState {
  try {
    return JSON.parse(localStorage.getItem(SIDEBAR_VIEW_KEY) || '{}') as SidebarViewState;
  } catch {
    return {};
  }
}
const savedSidebarView = loadSidebarViewState();

const conversations = ref<Conversation[]>([]);
const selectedId = ref('');
const mainPanel = ref<HTMLElement | null>(null);
const workspaceWidth = ref<number | null>(null);
const workspaceExpanded = ref(false);
const workspaceReloadVersion = ref(0);
const pinnedSummaryOpen = ref(false);
const workspacePanelStyle = computed(() => workspaceWidth.value
  ? { '--workspace-width': `${workspaceWidth.value}px` }
  : {});
let workspaceResizeStart: { x: number; width: number } | null = null;
const workspacesByConversation = reactive(new Map<string, WorkspaceDescriptor>());
const workspaceVisibilityByConversation = reactive(new Map<string, boolean>());
const workspaceVisible = computed({
  get: () => Boolean(workspaceVisibilityByConversation.get(selectedId.value)),
  set: visible => {
    const conversationId = selectedId.value;
    if (!conversationId) return;
    workspaceVisibilityByConversation.set(conversationId, visible);
  }
});
const activeWorkspace = computed<WorkspaceDescriptor | null>({
  get: () => workspacesByConversation.get(selectedId.value) || null,
  set: workspace => {
    const conversationId = selectedId.value;
    if (!conversationId) return;
    if (workspace) workspacesByConversation.set(conversationId, workspace);
    else workspacesByConversation.delete(conversationId);
  }
});
const detail = ref<ConversationDetail | null>(null);
const detailsByConversation = reactive(new Map<string, ConversationDetail>());
const draftsByConversation = reactive(new Map<string, string>());
const loadingList = ref(false);
const loadingDetail = ref(false);
const loadingDetailId = ref('');
const sending = ref(false);
const sendingRequest = ref(false);
const creatingConversation = ref(false);
const processingConversationIds = ref(new Set<string>());
const activeTurnsByConversation = reactive(new Map<string, Turn>());
const activeTurnResultSeenByConversation = reactive(new Map<string, boolean>());
const activeTurnStartedAtByConversation = reactive(new Map<string, number>());
const processStartedAtByConversation = reactive(new Map<string, number>());
const processActiveConversationIds = reactive(new Set<string>());
const nowMs = ref(Date.now());
const stopping = ref(false);
let turnTimer: number | undefined;
const locallyStoppedRunIds = new Set<string>();
const locallyStoppedSessionIds = new Set<string>();
const sidebarCollapsed = ref(savedSidebarView.sidebarCollapsed ?? false);
const mobileSidebarOpen = ref(false);
const renameVisible = ref(false);
const renameValue = ref('');
const messagePane = ref<HTMLElement>();
const autoScrollEnabled = ref(true);
const currentUser = ref<UserInfo | null>(null);
const authChecking = ref(true);
const loginVisible = ref(false);
const loginUsername = ref('');
const loginPassword = ref('');
const loginLoading = ref(false);
const confirmVisible = ref(false);
const confirmTitle = ref('');
const confirmContent = ref('');
const confirmPositiveText = ref('删除');
const confirmLoading = ref(false);
let confirmAction: (() => Promise<void>) | undefined;
const settingsVisible = ref(false);
const settingsSection = ref<'appearance' | 'archives'>('appearance');
// 手机端设置侧栏的导航展开/收缩（默认展开）
const settingsNavExpanded = ref(true);
const userMenuVisible = ref(false);
const pinnedExpanded = ref(savedSidebarView.pinnedExpanded ?? true);
const groupsExpanded = ref(savedSidebarView.groupsExpanded ?? true);
const conversationsExpanded = ref(savedSidebarView.conversationsExpanded ?? true);
const archiveQuery = ref('');
interface ConversationGroup {
  id: string;
  name: string;
  collapsed: boolean;
}
const conversationGroups = ref<ConversationGroup[]>([]);
const groupEditorVisible = ref(false);
const groupEditorName = ref('');
const editingGroupId = ref('');
const conversationPreview = ref<Conversation | null>(null);
const conversationPreviewTop = ref(0);
const draggedConversationId = ref('');
const conversationDropTarget = ref('');
const userMenuWrap = ref<HTMLElement>();
type Appearance = 'system' | 'light' | 'dark';
const APPEARANCE_KEY = 'agentrazor_appearance';
const savedAppearance = localStorage.getItem(APPEARANCE_KEY);
const appearance = ref<Appearance>(
  savedAppearance === 'light' || savedAppearance === 'dark' || savedAppearance === 'system'
    ? savedAppearance
    : 'system'
);
const systemDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches);
const colorSchemeMedia = window.matchMedia('(prefers-color-scheme: dark)');
const appearanceOptions: Array<{ key: Appearance; label: string; icon: string }> = [
  { key: 'system', label: '跟随系统', icon: 'solar:monitor-smartphone-linear' },
  { key: 'light', label: '浅色', icon: 'solar:sun-2-linear' },
  { key: 'dark', label: '深色', icon: 'solar:moon-linear' }
];
const streamClosers = new Map<string, () => void>();
let conversationSelectionToken = 0;
const draftConversationGroupId = ref('');

const themeOverrides = {
  common: {
    fontSize: '16px',
    fontSizeMedium: '16px',
    fontSizeSmall: '14px',
    primaryColor: '#3186c7',
    primaryColorHover: '#2476b5',
    primaryColorPressed: '#1d6399',
    primaryColorSuppl: '#3186c7',
    borderRadius: '10px'
  }
};

function conversationCreatedAtMs(item: Conversation) {
  const time = Date.parse(item.createdAt || '');
  return Number.isFinite(time) ? time : 0;
}

function sortConversationsByCreatedAt(items: Conversation[]) {
  return [...items].sort((left, right) => conversationCreatedAtMs(right) - conversationCreatedAtMs(left));
}

const visibleConversations = computed(() =>
  sortConversationsByCreatedAt(conversations.value.filter(item => item.status !== 'archived'))
);
const archivedConversations = computed(() =>
  sortConversationsByCreatedAt(conversations.value.filter(item => item.status === 'archived'))
);
const pinnedConversations = computed(() =>
  visibleConversations.value.filter(item => Boolean(item.pinnedAt))
);
const conversationList = computed(() =>
  visibleConversations.value.filter(item => !item.pinnedAt && !item.groupId)
);
const filteredArchivedConversations = computed(() => {
  const query = archiveQuery.value.trim().toLowerCase();
  return query
    ? archivedConversations.value.filter(item => item.title.toLowerCase().includes(query))
    : archivedConversations.value;
});
// 已归档对话按所属分组归类展示
const archivedConversationSections = computed(() => {
  const byGroup = new Map<string | undefined, Conversation[]>();
  for (const item of filteredArchivedConversations.value) {
    const key = item.groupId || undefined;
    if (!byGroup.has(key)) byGroup.set(key, []);
    byGroup.get(key)!.push(item);
  }
  const sections: { title: string; groupId?: string; items: Conversation[] }[] = [];
  for (const group of conversationGroups.value) {
    const items = byGroup.get(group.id);
    if (items?.length) sections.push({ title: group.name, groupId: group.id, items });
  }
  const ungrouped = byGroup.get(undefined);
  if (ungrouped?.length) sections.push({ title: '对话', items: ungrouped });
  return sections;
});
const activeConversation = computed(() =>
  conversations.value.find(item => item.id === selectedId.value)
);
const isArchivedActive = computed(() => activeConversation.value?.status === 'archived');
function isDraftConversation(id = selectedId.value) {
  return id === DRAFT_CONVERSATION_ID;
}

function createDraftConversationDetail(): ConversationDetail {
  const now = new Date().toISOString();
  return {
    conversation: {
      id: DRAFT_CONVERSATION_ID,
      title: '新对话',
      status: 'active',
      groupId: draftConversationGroupId.value || undefined,
      running: false,
      createdAt: now,
      updatedAt: now
    },
    eventCursor: 0,
    turns: []
  };
}

const activeDetail = computed(() => {
  if (!selectedId.value) return null;
  if (isDraftConversation()) return detail.value?.conversation.id === DRAFT_CONVERSATION_ID ? detail.value : createDraftConversationDetail();
  return detailsByConversation.get(selectedId.value)
    || (detail.value?.conversation.id === selectedId.value ? detail.value : null);
});
const currentStreamingTurn = computed(() =>
  selectedId.value ? activeTurnsByConversation.get(selectedId.value) || null : null
);
const currentTurnElapsedMs = computed(() => {
  if (!selectedId.value) return 0;
  const startedAt = processStartedAtByConversation.get(selectedId.value);
  return startedAt ? Math.max(1000, nowMs.value - startedAt) : 1000;
});
const currentTurnResultSeen = computed(() =>
  Boolean(selectedId.value && activeTurnResultSeenByConversation.get(selectedId.value))
);
const draft = computed({
  get() {
    return selectedId.value ? draftsByConversation.get(selectedId.value) || '' : '';
  },
  set(value: string) {
    if (!selectedId.value) return;
    setConversationDraft(selectedId.value, value);
  }
});

function setConversationDraft(conversationId: string, value: string) {
  if (!conversationId) return;
  if (value) draftsByConversation.set(conversationId, value);
  else draftsByConversation.delete(conversationId);
}
const userMessages = computed(() =>
  (activeDetail.value?.turns || []).flatMap(turn => turn.items.filter(item => item.type === 'userMessage'))
);
const renderedTurns = computed(() => normalizedRenderedTurns(activeDetail.value?.turns || [], currentStreamingTurn.value));
const pinnedSummaryWorkspaces = computed(() => {
  const byUrl = new Map<string, WorkspaceDescriptor>();
  for (const turn of renderedTurns.value) {
    for (const item of turn.items) {
      if (item.type !== 'agentMessage' || !item.text) continue;
      for (const workspace of parseAgentMessage(item.text, false).workspaces) {
        byUrl.set(workspace.url, workspace);
      }
    }
  }
  return [...byUrl.values()];
});
const renderedTurnViews = computed(() => renderedTurns.value.map(createTurnView));
const canSend = computed(() => Boolean(draft.value.trim() && selectedId.value && !creatingConversation.value && !sendingRequest.value && !isConversationRunning(selectedId.value)));
const composerActionPending = computed(() => creatingConversation.value || sendingRequest.value || stopping.value);
const composerActionDisabled = computed(() => sending.value ? stopping.value : !canSend.value);
const composerActionLabel = computed(() => {
  if (stopping.value) return '正在停止';
  if (sending.value) return '停止当前任务';
  if (composerActionPending.value) return '发送中';
  return '发送消息';
});
const composerActionIcon = computed(() => {
  if (stopping.value || !sending.value && composerActionPending.value) return 'solar:refresh-linear';
  if (sending.value) return 'solar:stop-bold';
  return 'solar:arrow-up-linear';
});
const loadingCurrentDetail = computed(() =>
  Boolean(selectedId.value)
  && !activeDetail.value
  && loadingDetail.value
  && loadingDetailId.value === selectedId.value
);
const conversationOpening = computed(() =>
  Boolean(selectedId.value)
  && !activeDetail.value
  && !currentStreamingTurn.value
);
const currentConversationRunning = computed(() =>
  Boolean(selectedId.value && !isDraftConversation() && (cachedActiveTurn(selectedId.value) || isConversationRunning(selectedId.value)))
);
const isNewChat = computed(() =>
  Boolean(selectedId.value)
  && !conversationOpening.value
  && Boolean(activeDetail.value)
  && !activeDetail.value!.turns?.length
  && !currentConversationRunning.value
  && !currentStreamingTurn.value
);
const userInitial = computed(() => currentUser.value?.username.trim().slice(0, 2).toUpperCase() || 'AR');
const isDarkAppearance = computed(() =>
  appearance.value === 'dark' || appearance.value === 'system' && systemDark.value
);
const activeTheme = computed(() => isDarkAppearance.value ? darkTheme : null);

// toast/message 跟随主题：深色模式下弹层不再出现白底
const toastProviderProps = reactive({
  theme: activeTheme.value,
  themeOverrides
});
const { message: toast } = createDiscreteApi(['message'], {
  configProviderProps: toastProviderProps
});

const renderIcon = (icon: string) => () => h(Icon, { icon });
const menuOptions = computed(() => {
  const conv = activeConversation.value;
  if (!conv) return [];
  const archived = conv.status === 'archived';
  const options = [
    { label: '重命名', key: 'rename', icon: renderIcon('solar:pen-2-outline'), disabled: archived },
    { label: conv.pinnedAt ? '取消置顶' : '置顶', key: 'pin', icon: renderIcon('solar:pin-bold') },
    { label: archived ? '取消归档' : '归档', key: 'archive', icon: renderIcon('solar:archive-linear') }
  ];
  // 只有归档的对话才能删除：未归档时直接不显示删除项
  if (archived) {
    options.push({ label: '删除', key: 'delete', icon: renderIcon('solar:trash-bin-trash-linear') });
  }
  return options;
});

function conversationsInGroup(groupId: string) {
  return visibleConversations.value
    .filter(item => item.groupId === groupId && !item.pinnedAt)
    .sort((left, right) => {
      if (Boolean(left.pinnedAt) !== Boolean(right.pinnedAt)) return left.pinnedAt ? -1 : 1;
      const createDiff = new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime();
      if (createDiff !== 0) return createDiff;
      return right.id.localeCompare(left.id);
    });
}

function conversationGroupName(item: Conversation) {
  if (!item.groupId) return '';
  return conversationGroups.value.find(group => group.id === item.groupId)?.name || '';
}

function revealConversationSection(item: Conversation) {
  if (item.pinnedAt) {
    pinnedExpanded.value = true;
    return;
  }

  if (item.groupId) {
    groupsExpanded.value = true;
    const group = conversationGroups.value.find(value => value.id === item.groupId);
    if (group) group.collapsed = false;
    return;
  }

  conversationsExpanded.value = true;
}

function setConversationProcessing(id: string, processing: boolean) {
  if (!id) return;
  const current = processingConversationIds.value.has(id);
  if (current === processing) return;
  const next = new Set(processingConversationIds.value);
  if (processing) next.add(id);
  else next.delete(id);
  processingConversationIds.value = next;
}

function clearConversationProcessing(ids: string[]) {
  if (!ids.length) return;
  const removed = new Set(ids);
  const next = new Set([...processingConversationIds.value].filter(id => !removed.has(id)));
  processingConversationIds.value = next;
}

function isConversationRunning(id: string) {
  return processingConversationIds.value.has(id)
    || Boolean(conversations.value.find(item => item.id === id)?.running);
}

function isConversationProcessing(item: Conversation) {
  return item.running || processingConversationIds.value.has(item.id);
}

function applyConversationList(next: Conversation[], preserveLocalProcessing = true) {
  const nextIds = new Set(next.map(item => item.id));
  const runningIds = new Set(next.filter(item => item.running).map(item => item.id));
  if (preserveLocalProcessing) {
    for (const id of processingConversationIds.value) {
      if (nextIds.has(id) && cachedActiveTurn(id)) {
        runningIds.add(id);
      }
    }
  }
  processingConversationIds.value = runningIds;
  conversations.value = next;
  for (const item of next) {
    if (item.running) ensureConversationStream(item.id);
  }
  closeIdleConversationStreams();
}

function upsertConversationListItem(item: Conversation) {
  const running = item.running || processingConversationIds.value.has(item.id);
  const index = conversations.value.findIndex(value => value.id === item.id);
  if (index >= 0) conversations.value[index] = { ...conversations.value[index], ...item, running };
  else conversations.value.unshift({ ...item, running });
  setConversationProcessing(item.id, running);
}

function closeConversationStream(id: string) {
  const close = streamClosers.get(id);
  if (!close) return;
  streamClosers.delete(id);
  close();
}

function closeAllConversationStreams() {
  for (const close of streamClosers.values()) close();
  streamClosers.clear();
}

function ensureConversationStream(id: string, afterId = 0) {
  if (!id || streamClosers.has(id)) return;
  const close = conversationApi.subscribe(
    id,
    afterId,
    event => void handleStreamEvent(event),
    () => {
      closeConversationStream(id);
      void reconcileConversationStream(id);
    },
    () => void reconcileConversationStream(id)
  );
  streamClosers.set(id, close);
}

async function reconcileConversationStream(id: string) {
  if (!id || !currentUser.value) return;
  await loadConversations(false, false);
  const conversation = conversations.value.find(item => item.id === id);
  if (conversation?.running) {
    if (selectedId.value === id) await refreshDetail({ restoreActiveTurn: true });
    return;
  }

  setConversationProcessing(id, false);
  stopTurnTimer(id);
  activeTurnsByConversation.delete(id);
  activeTurnResultSeenByConversation.delete(id);
  if (selectedId.value === id) {
    clearDisplayedActiveTurn();
    await refreshDetail({ restoreActiveTurn: false });
  }
}

function closeIdleConversationStreams(activeId = selectedId.value) {
  for (const id of [...streamClosers.keys()]) {
    if (id !== activeId && !processingConversationIds.value.has(id)) closeConversationStream(id);
  }
}

function showConversationPreview(item: Conversation, event: MouseEvent) {
  const target = event.currentTarget;
  if (!(target instanceof HTMLElement)) return;
  const rect = target.getBoundingClientRect();
  conversationPreview.value = item;
  conversationPreviewTop.value = Math.min(rect.top, window.innerHeight - 180);
}

function hideConversationPreview() {
  conversationPreview.value = null;
}

async function loadConversationGroups() {
  const previous = new Map(conversationGroups.value.map(group => [group.id, group.collapsed]));
  try {
    const groups = await conversationGroupApi.list();
    conversationGroups.value = groups.map(group => ({
      id: group.id,
      name: group.name,
      collapsed: previous.get(group.id) ?? savedSidebarView.collapsedGroups?.[group.id] ?? false
    }));
  } catch (error) {
    conversationGroups.value = [];
    showError(error);
  }
}

function openCreateGroup() {
  editingGroupId.value = '';
  groupEditorName.value = '';
  groupEditorVisible.value = true;
}

function openRenameGroup(group: ConversationGroup) {
  editingGroupId.value = group.id;
  groupEditorName.value = group.name;
  groupEditorVisible.value = true;
}

async function saveGroup() {
  const name = groupEditorName.value.trim();
  if (!name) return;
  if (editingGroupId.value) {
    await conversationGroupApi.update(editingGroupId.value, { name });
  } else {
    await conversationGroupApi.create(name);
  }
  await loadConversationGroups();
  groupEditorVisible.value = false;
}

function toggleGroup(group: ConversationGroup) {
  group.collapsed = !group.collapsed;
}

function deleteGroup(group: ConversationGroup) {
  openConfirm(
    '删除分组',
    `删除「${group.name}」后，其中的对话会移回“对话”列表。`,
    '删除分组',
    async () => {
      try {
        await conversationGroupApi.remove(group.id);
        await Promise.all([loadConversationGroups(), loadConversations()]);
      } catch (error) {
        showError(error);
      }
    }
  );
}

async function updateConversationGroup(item: Conversation, groupId: string) {
  const previousGroupId = item.groupId || '';
  const applyGroup = (conversation: Conversation) => {
    replaceConversation({ ...conversation, groupId: groupId || undefined });
  };

  applyGroup(item);
  try {
    const updated = await conversationApi.update(item.id, { groupId });
    applyGroup({ ...item, ...updated });
  } catch (error) {
    replaceConversation({ ...item, groupId: previousGroupId || undefined });
    showError(error);
  }
}

// 拖拽实现：鼠标走 Pointer 事件，触屏走长按 + touchmove（原生 HTML5 DnD 不支持触屏，
// 且 touch-action 会被浏览器当滚动吞掉手势，所以触屏用 preventDefault 接管）
let pointerDragStart: { item: Conversation; x: number; y: number } | null = null;
let dragGhostEl: HTMLElement | null = null;
let touchDrag: { item: Conversation; x: number; y: number; activated: boolean } | null = null;
let touchLongPressTimer: number | undefined;

function onConversationPointerDown(item: Conversation, event: PointerEvent) {
  if (event.button !== 0 || event.pointerType === 'touch') return;
  pointerDragStart = { item, x: event.clientX, y: event.clientY };
}

function activateDrag(item: Conversation, x: number, y: number) {
  draggedConversationId.value = item.id;
  conversationDropTarget.value = '';
  hideConversationPreview();
  document.body.classList.add('is-dragging');
  dragGhostEl = document.createElement('div');
  dragGhostEl.className = 'conversation-drag-ghost';
  dragGhostEl.textContent = item.title;
  document.body.appendChild(dragGhostEl);
  updateDragGhost(x, y);
}

function updateDragGhost(x: number, y: number) {
  if (dragGhostEl) {
    dragGhostEl.style.left = `${x + 12}px`;
    dragGhostEl.style.top = `${y + 10}px`;
  }
}

function updateDropTarget(x: number, y: number) {
  const zone = document.elementFromPoint(x, y)
    ?.closest?.('.conversation-drop-zone') as HTMLElement | null | undefined;
  conversationDropTarget.value = zone ? String(zone.dataset.dropTarget || zone.dataset.groupId || '') : '';
}

function dropTargetGroupId(target: string) {
  return target === CONVERSATION_LIST_DROP_TARGET ? '' : target;
}

function finishDrag(item: Conversation) {
  const target = conversationDropTarget.value;
  conversationDropTarget.value = '';
  draggedConversationId.value = '';
  if (dragGhostEl) {
    dragGhostEl.remove();
    dragGhostEl = null;
  }
  document.body.classList.remove('is-dragging');
  if (!target) return;

  const targetGroupId = dropTargetGroupId(target);
  if ((item.groupId || '') === targetGroupId) return;
  if (targetGroupId) {
    const group = conversationGroups.value.find(candidate => candidate.id === targetGroupId);
    if (group) group.collapsed = false;
  } else {
    conversationsExpanded.value = true;
  }
  void updateConversationGroup(item, targetGroupId);
}

function onDragPointerMove(event: PointerEvent) {
  if (!pointerDragStart) return;
  const dx = event.clientX - pointerDragStart.x;
  const dy = event.clientY - pointerDragStart.y;
  if (!draggedConversationId.value) {
    // 移动超过阈值才算拖拽，避免影响点击
    if (Math.hypot(dx, dy) < 8) return;
    activateDrag(pointerDragStart.item, event.clientX, event.clientY);
  }
  updateDragGhost(event.clientX, event.clientY);
  updateDropTarget(event.clientX, event.clientY);
}

function onDragPointerUp() {
  if (!pointerDragStart) return;
  const item = pointerDragStart.item;
  pointerDragStart = null;
  finishDrag(item);
}

// 触屏：长按激活拖拽，激活后 touchmove 阻止页面滚动，保证手势不丢
function onRowTouchStart(item: Conversation, event: TouchEvent) {
  if (event.touches.length !== 1) return;
  const touch = event.touches[0];
  touchDrag = { item, x: touch.clientX, y: touch.clientY, activated: false };
  touchLongPressTimer = window.setTimeout(() => {
    if (touchDrag && !touchDrag.activated) {
      touchDrag.activated = true;
      activateDrag(touchDrag.item, touchDrag.x, touchDrag.y);
    }
  }, 350);
}

function onWindowTouchMove(event: TouchEvent) {
  if (!touchDrag || event.touches.length !== 1) return;
  const touch = event.touches[0];
  const dx = touch.clientX - touchDrag.x;
  const dy = touch.clientY - touchDrag.y;
  if (!touchDrag.activated) {
    // 长按前就移动：取消拖拽，恢复正常滚动
    if (Math.hypot(dx, dy) > 8) {
      window.clearTimeout(touchLongPressTimer);
      touchDrag = null;
    }
    return;
  }
  event.preventDefault();
  updateDragGhost(touch.clientX, touch.clientY);
  updateDropTarget(touch.clientX, touch.clientY);
}

function onWindowTouchEnd() {
  if (!touchDrag) return;
  window.clearTimeout(touchLongPressTimer);
  const item = touchDrag.item;
  touchDrag = null;
  finishDrag(item);
}

async function loadConversations(selectFirst = false, preserveLocalProcessing = true) {
  loadingList.value = true;
  try {
    applyConversationList(await conversationApi.list(), preserveLocalProcessing);
    if (selectFirst && !selectedId.value) {
      const requestedId = conversationIdFromPath(window.location.pathname);
      let preferred = visibleConversations.value.find(item => item.id === requestedId);
      if (!preferred && requestedId) {
        // 列表里还没有这个对话（例如刚创建、还没有消息），单独读取后恢复，
        // 避免刷新后直接跳到第一行对话。
        try {
          const detail = await conversationApi.get(requestedId);
          if (detail?.conversation) {
            preferred = detail.conversation;
            upsertConversationListItem(preferred);
          }
        } catch {
          preferred = undefined;
        }
      }
      const fallback = preferred || visibleConversations.value[0];
      if (fallback) await selectConversation(fallback.id);
      else syncConversationUrl('');
    }
  } catch (error) {
    showError(error);
  } finally {
    loadingList.value = false;
  }
}

function createConversation() {
  if (sendingRequest.value) return;
  if (!currentUser.value) {
    loginVisible.value = true;
    return;
  }
  mobileSidebarOpen.value = false;
  draftConversationGroupId.value = '';
  selectedId.value = DRAFT_CONVERSATION_ID;
  detail.value = createDraftConversationDetail();
  clearDisplayedActiveTurn();
  syncConversationUrl('');
  autoScrollEnabled.value = true;
  conversationsExpanded.value = true;
}

function openMobileSidebar() {
  // 移动端抽屉：确保侧栏内容展开（桌面端折叠状态不影响抽屉显示）
  if (sidebarCollapsed.value) sidebarCollapsed.value = false;
  mobileSidebarOpen.value = true;
}

function createConversationInGroup(group: ConversationGroup) {
  if (sendingRequest.value) return;
  if (!currentUser.value) {
    loginVisible.value = true;
    return;
  }
  mobileSidebarOpen.value = false;
  draftConversationGroupId.value = group.id;
  selectedId.value = DRAFT_CONVERSATION_ID;
  detail.value = createDraftConversationDetail();
  clearDisplayedActiveTurn();
  syncConversationUrl('');
  autoScrollEnabled.value = true;
  groupsExpanded.value = true;
  group.collapsed = false;
}

function handleGroupAction(group: ConversationGroup, key: string) {
  if (key === 'rename') openRenameGroup(group);
  if (key === 'archiveConversations') archiveGroupConversations(group);
  if (key === 'delete') deleteGroup(group);
}

function archiveGroupConversations(group: ConversationGroup) {
  openConfirm(
    '归档分组对话',
    `归档「${group.name}」中的所有对话？归档后可随时在“已归档”设置中恢复。`,
    '归档',
    async () => {
      try {
        const archivedIds = conversations.value.filter(item => item.groupId === group.id).map(item => item.id);
        await conversationGroupApi.archiveConversations(group.id);
        clearConversationProcessing(archivedIds);
        await Promise.all([loadConversationGroups(), loadConversations()]);
      } catch (error) {
        showError(error);
      }
    }
  );
}

async function selectConversation(id: string) {
  mobileSidebarOpen.value = false;
  if (!id) return;
  if (id === selectedId.value && activeDetail.value) {
    if (loadingDetailId.value !== id) loadingDetail.value = false;
    return;
  }

  const token = ++conversationSelectionToken;

  selectedId.value = id;
  syncConversationUrl(id);
  detail.value = detailsByConversation.get(id) || null;
  autoScrollEnabled.value = true;
  restoreRunningConversationFromList(id);
  syncDisplayedActiveTurnState();

  const snapshot = await refreshDetail({ forceScroll: true });
  if (conversationSelectionToken !== token || selectedId.value !== id) return;
  ensureConversationStream(id, snapshot?.eventCursor ?? 0);
  closeIdleConversationStreams(id);
}

function conversationIdFromPath(pathname: string): string {
  // /c/:id 以及后续子级路径 /c/:id/<sub> 都从首段解析
  const match = pathname.match(/^\/c\/([^/]+)/);
  return match ? decodeURIComponent(match[1]) : '';
}

// 打开设置前所在的路径，关闭设置时回到这里
const settingsReturnPath = ref('/');

// 设置页是独立路径：/settings/appearance（外观）、/settings/archives（已归档）
function syncSettingsUrl() {
  const path = settingsVisible.value
    ? settingsSection.value === 'archives' ? '/settings/archives' : '/settings/appearance'
    : settingsReturnPath.value || '/';
  window.history.replaceState(window.history.state, '', path);
}

function syncConversationUrl(id: string) {
  const path = id ? `/c/${encodeURIComponent(id)}` : '/';
  settingsReturnPath.value = path;
  // 设置页打开时 URL 保持设置路径，不覆盖
  if (settingsVisible.value) return;
  window.history.replaceState(window.history.state, '', path);
}

interface RefreshDetailOptions {
  forceScroll?: boolean;
  restoreActiveTurn?: boolean;
}

interface BeginActiveTurnOptions {
  sessionId?: string;
  turn?: Turn;
  id?: string;
  status?: string;
  startedAt?: string;
  resetResultSeen?: boolean;
  restartTimer?: boolean;
}

async function refreshDetail(options: RefreshDetailOptions = {}): Promise<ConversationDetail | null> {
  const id = selectedId.value;
  if (!id) return null;
  loadingDetailId.value = id;
  loadingDetail.value = true;
  try {
    const snapshot = await conversationApi.get(id);
    if (selectedId.value !== id) return null;
    restoreActiveTurn(snapshot, options.restoreActiveTurn !== false);
    return snapshot;
  } catch (error) {
    if (selectedId.value === id) showError(error);
    return null;
  } finally {
    if (loadingDetailId.value === id) {
      loadingDetail.value = false;
      loadingDetailId.value = '';
    }
    if (selectedId.value === id) {
      void scrollToBottom({ force: options.forceScroll });
    }
  }
}

function isActiveTurn(turn: Turn) {
  const normalized = String(turn.status || '').replace(/[-_\s]/g, '').toLowerCase();
  return normalized === 'inprogress' || normalized === 'running' || normalized === 'pending';
}

function markRestoredRunningTurn(turn: Turn) {
  return { ...turn, restoredRunning: true } as Turn & { restoredRunning: boolean };
}

function isRestoredRunningTurn(turn: Turn) {
  return Boolean((turn as Turn & { restoredRunning?: boolean }).restoredRunning);
}

function hasVisibleProcessItems(turn: Turn) {
  return turnProcessItems(turn).length > 0;
}

function completedProcessDisplays(turn: Turn) {
  return processDisplayItems(turn, false);
}

function hasActiveProcessState() {
  return Boolean(selectedId.value && processActiveConversationIds.has(selectedId.value));
}

function isVisibleProcessStreamItem(item: ThreadItem) {
  return item.type !== 'userMessage'
    && item.type !== 'imageGeneration'
    && item.type !== 'reasoning'
    && (item.type !== 'agentMessage' || item.phase !== 'final_answer');
}

function isRunningTurnStatus(turn: Turn) {
  return ['inprogress', 'running', 'pending', 'started'].includes(String(turn.status || '').replace(/[-_\s]/g, '').toLowerCase());
}

function isStreamingTurn(turn: Turn) {
  const current = currentStreamingTurn.value;
  return turn === current
    || Boolean(current && turn.id && current.id && turn.id === current.id)
    || isRestoredRunningTurn(turn)
    || isRunningTurnStatus(turn);
}

function rawStreamingProcessDisplays(turn: Turn) {
  return processDisplayItems(turn, true);
}

function stableTurnId(turn: Turn) {
  const id = String(turn.id || '');
  return id && !id.startsWith('running-') ? id : '';
}

function turnUserSignature(turn: Turn) {
  const userItem = turn.items.find(item => item.type === 'userMessage');
  return userItem ? userItemText(userItem) || userItem.id : '';
}

function turnRenderKey(turn: Turn) {
  const id = stableTurnId(turn);
  if (id) return `turn:${id}`;
  const user = turnUserSignature(turn);
  if (user) return `user:${user}`;
  return `turn:${turn.startedAt || turn.status || 'empty'}`;
}

function sameRenderedTurn(detailTurn: Turn, activeTurn: Turn, index: number, total: number) {
  const detailId = stableTurnId(detailTurn);
  const activeId = stableTurnId(activeTurn);
  if (detailId && activeId) return detailId === activeId;
  if (index !== total - 1) return false;
  const detailUser = turnUserSignature(detailTurn);
  const activeUser = turnUserSignature(activeTurn);
  return Boolean(detailUser && activeUser && detailUser === activeUser);
}

function mergeRenderedActiveTurn(detailTurn: Turn, activeTurn: Turn) {
  const merged = cloneTurnForStream(activeTurn);
  merged.id = stableTurnId(activeTurn) || stableTurnId(detailTurn) || activeTurn.id || detailTurn.id;
  merged.startedAt = activeTurn.startedAt || detailTurn.startedAt;
  merged.completedAt = activeTurn.completedAt || detailTurn.completedAt;
  if (merged.durationMs === undefined && detailTurn.durationMs !== undefined) merged.durationMs = detailTurn.durationMs;
  merged.items = mergeTurnItems(activeTurn.items || [], detailTurn.items || []);
  return merged;
}

function normalizedRenderedTurns(detailTurns: Turn[], activeTurn: Turn | null) {
  if (!activeTurn) return detailTurns;
  const duplicateIndex = detailTurns.findIndex((turn, index) => sameRenderedTurn(turn, activeTurn, index, detailTurns.length));
  if (duplicateIndex < 0) return [...detailTurns, activeTurn];
  const next = [...detailTurns];
  next[duplicateIndex] = mergeRenderedActiveTurn(next[duplicateIndex], activeTurn);
  return next;
}

function createStreamingTurnView(turn: Turn, userItems: ThreadItem[], resultItems: ThreadItem[]): TurnView {
  const rawProcessDisplays = rawStreamingProcessDisplays(turn);
  const hasProcessShell = isRestoredRunningTurn(turn) || hasActiveProcessState() || rawProcessDisplays.length > 0;
  const processDisplays = hasProcessShell ? rawProcessDisplays : [];
  const hasResult = resultItems.some(item =>
    item.type === 'imageGeneration'
    || item.type === 'agentMessage' && Boolean(item.text)
  );
  const hasLiveProcess = processDisplays.some(display => display.live);

  return {
    renderKey: turnRenderKey(turn),
    turn,
    userItems,
    resultItems,
    processDisplays,
    processMode: hasProcessShell ? 'processing' : 'thinking',
    processSummary: hasProcessShell ? streamingProcessSummary(turn, currentTurnElapsedMs.value) : '',
    showTailThinking: hasProcessShell && processDisplays.length > 0 && !hasResult && !hasLiveProcess,
    streaming: true
  };
}

function createCompletedTurnView(turn: Turn, userItems: ThreadItem[], resultItems: ThreadItem[]): TurnView {
  const processDisplays = showCompletedProcessSummary(turn) ? completedProcessDisplays(turn) : [];
  return {
    renderKey: turnRenderKey(turn),
    turn,
    userItems,
    resultItems,
    processDisplays,
    processMode: showCompletedProcessSummary(turn) ? 'completed' : 'none',
    processSummary: showCompletedProcessSummary(turn) ? completedProcessSummary(turn) : '',
    showTailThinking: false,
    streaming: false
  };
}

function createTurnView(turn: Turn): TurnView {
  const userItems = turn.items.filter(item => item.type === 'userMessage');
  const resultItems = turnResultItems(turn);
  return isStreamingTurn(turn)
    ? createStreamingTurnView(turn, userItems, resultItems)
    : createCompletedTurnView(turn, userItems, resultItems);
}

function cloneThreadItem(item: ThreadItem): ThreadItem {
  const next: ThreadItem = { ...item };
  if (Array.isArray(item.content)) next.content = [...item.content] as ThreadItem['content'];
  return next;
}

function cloneTurnForStream(turn: Turn): Turn {
  return {
    ...turn,
    items: (turn.items || []).map(cloneThreadItem)
  };
}

function cachedActiveTurn(sessionId: string) {
  return activeTurnsByConversation.get(sessionId) || null;
}

function activeTurnResultSeen(turn: Turn) {
  return turn.items.some(item =>
    item.type === 'agentMessage'
    && item.phase === 'final_answer'
    && Boolean(item.text)
  );
}

function syncThreadItem(target: ThreadItem, source: ThreadItem) {
  const next = cloneThreadItem(source);
  const mutableTarget = target as Record<string, unknown>;
  for (const key of Object.keys(target)) {
    if (!(key in next)) delete mutableTarget[key];
  }
  Object.assign(target, next);
}

function syncTurnItems(target: ThreadItem[], source: ThreadItem[], options: { skipIncomingReasoning?: boolean } = {}) {
  const merged = mergeTurnItems(target, source, options);
  const existingById = new Map(target.map(item => [item.id, item]));
  const nextItems = merged.map(item => {
    const existing = existingById.get(item.id);
    if (!existing) return cloneThreadItem(item);
    syncThreadItem(existing, item);
    return existing;
  });
  target.splice(0, target.length, ...nextItems);
}

function syncTurnForStream(target: Turn, source: Turn, options: { keepStatus?: boolean; skipIncomingReasoning?: boolean } = {}) {
  const keepRestoredRunning = isRestoredRunningTurn(target) || isRestoredRunningTurn(source);
  target.id = source.id;
  if (!options.keepStatus) target.status = source.status;
  if (source.startedAt) target.startedAt = source.startedAt;
  if (source.completedAt) target.completedAt = source.completedAt;
  if (source.durationMs !== undefined) target.durationMs = source.durationMs;
  if (source.error !== undefined) target.error = source.error;
  if (keepRestoredRunning) (target as Turn & { restoredRunning?: boolean }).restoredRunning = true;
  syncTurnItems(target.items, source.items || [], { skipIncomingReasoning: options.skipIncomingReasoning });
}

function publishActiveTurn(sessionId: string, turn?: Turn) {
  if (!sessionId) return;
  const current = turn || cachedActiveTurn(sessionId);
  if (!current) return;
  if (activeTurnsByConversation.get(sessionId) !== current) activeTurnsByConversation.set(sessionId, current);
  if (activeTurnResultSeen(current) && !activeTurnResultSeenByConversation.get(sessionId)) {
    activeTurnResultSeenByConversation.set(sessionId, true);
  }
}

function beginActiveTurn(options: BeginActiveTurnOptions = {}): Turn {
  const sessionId = options.sessionId || selectedId.value;
  let current: Turn | null = cachedActiveTurn(sessionId);

  if (options.turn) {
    if (current) syncTurnForStream(current, options.turn);
    else current = cloneTurnForStream(options.turn);
  } else if (!current) {
    current = { id: options.id || '', status: options.status || 'inProgress', items: [] };
  }

  if (options.id) current.id = options.id;
  if (options.status) current.status = options.status;
  if (options.startedAt) current.startedAt = options.startedAt;
  if (sessionId && activeTurnsByConversation.get(sessionId) !== current) activeTurnsByConversation.set(sessionId, current);

  if (options.resetResultSeen) activeTurnResultSeenByConversation.delete(sessionId);
  else if (activeTurnResultSeen(current) && !activeTurnResultSeenByConversation.get(sessionId)) activeTurnResultSeenByConversation.set(sessionId, true);

  setConversationProcessing(sessionId, true);
  if (sessionId === selectedId.value) {
    sending.value = true;
    stopping.value = false;
  }
  if (options.restartTimer || !activeTurnStartedAtByConversation.has(sessionId)) startTurnTimer(sessionId, current.startedAt || options.startedAt);
  return current;
}

function confirmSentTurn(sessionId: string, turn: Turn) {
  const current = cachedActiveTurn(sessionId);
  if (!current) {
    beginActiveTurn({ sessionId, turn, resetResultSeen: true, restartTimer: true });
    return;
  }

  const hasStreamedItems = current.items.some(item => item.type !== 'userMessage');
  beginActiveTurn({
    sessionId,
    turn: {
      ...current,
      id: current.id || turn.id,
      status: 'inProgress',
      startedAt: current.startedAt || turn.startedAt,
      items: mergeTurnItems(current.items, turn.items)
    },
    resetResultSeen: !hasStreamedItems,
    restartTimer: true
  });
}

function activeTurnError(error: unknown) {
  return error instanceof Error && error.message.includes('active turn');
}

function syncDisplayedActiveTurnState() {
  const running = Boolean(selectedId.value && cachedActiveTurn(selectedId.value));
  sending.value = running;
  if (!running) stopping.value = false;
}

function clearDisplayedActiveTurn() {
  sending.value = false;
  stopping.value = false;
}

function resetActiveTurn(options: { clearCache?: boolean } = {}) {
  if (options.clearCache && selectedId.value) {
    stopTurnTimer(selectedId.value);
    processStartedAtByConversation.delete(selectedId.value);
    processActiveConversationIds.delete(selectedId.value);
    activeTurnsByConversation.delete(selectedId.value);
    activeTurnResultSeenByConversation.delete(selectedId.value);
  }
  clearDisplayedActiveTurn();
}

function restoreRunningConversationFromList(sessionId: string) {
  const conversation = conversations.value.find(item => item.id === sessionId);
  if (!conversation || !isConversationProcessing(conversation)) return;
  setConversationProcessing(sessionId, true);
  const activeTurn = cachedActiveTurn(sessionId) || markRestoredRunningTurn({
    id: `running-${sessionId}`,
    status: 'inProgress',
    startedAt: conversation.runningStartedAt || conversation.updatedAt || conversation.createdAt,
    items: []
  });
  beginActiveTurn({
    sessionId,
    turn: activeTurn,
    resetResultSeen: !cachedActiveTurn(sessionId),
    restartTimer: true
  });
}

function activeTurnStartedAt(snapshot: ConversationDetail, turn?: Turn) {
  return snapshot.conversation.runningStartedAt
    || turn?.startedAt
    || snapshot.conversation.updatedAt
    || snapshot.conversation.createdAt
    || '';
}

function setConversationDetail(snapshot: ConversationDetail) {
  detailsByConversation.set(snapshot.conversation.id, snapshot);
  if (selectedId.value === snapshot.conversation.id) detail.value = snapshot;
}

function clearConversationDetail(id: string) {
  detailsByConversation.delete(id);
  if (detail.value?.conversation.id === id) detail.value = null;
}

function restoreActiveTurn(snapshot: ConversationDetail, enabled: boolean) {
  upsertConversationListItem(snapshot.conversation);
  const turns = snapshot.turns || [];
  let activeIndex = -1;
  if (enabled) {
    for (let index = turns.length - 1; index >= 0; index -= 1) {
      if (isActiveTurn(turns[index])) {
        activeIndex = index;
        break;
      }
    }
  }
  const cachedTurn = cachedActiveTurn(snapshot.conversation.id);
  if (activeIndex < 0) {
    setConversationDetail(snapshot);
    setConversationProcessing(snapshot.conversation.id, snapshot.conversation.running);
    if (enabled && snapshot.conversation.running) {
      const activeTurn = cachedTurn || markRestoredRunningTurn({
        id: '',
        status: 'inProgress',
        startedAt: activeTurnStartedAt(snapshot),
        items: []
      });
      restoreProcessState(snapshot.conversation.id, activeTurn, activeTurnStartedAt(snapshot, activeTurn));
      beginActiveTurn({
        sessionId: snapshot.conversation.id,
        turn: activeTurn,
        resetResultSeen: !cachedTurn,
        restartTimer: true
      });
    }
    return;
  }

  setConversationProcessing(snapshot.conversation.id, true);
  const detailActiveTurn = snapshot.conversation.running
    ? markRestoredRunningTurn(turns[activeIndex])
    : turns[activeIndex];
  const activeTurn = cachedTurn
    ? { ...cachedTurn, items: mergeTurnItems(cachedTurn.items, detailActiveTurn.items) }
    : detailActiveTurn;
  const displaySnapshot = {
    ...snapshot,
    turns: turns.filter((_, index) => index !== activeIndex)
  };
  setConversationDetail(displaySnapshot);
  const startedAt = activeTurnStartedAt(snapshot, activeTurn);
  restoreProcessState(snapshot.conversation.id, activeTurn, startedAt);
  beginActiveTurn({
    sessionId: snapshot.conversation.id,
    turn: activeTurn,
    startedAt,
    restartTimer: true
  });
}

async function sendMessage() {
  const draftKey = selectedId.value;
  const content = draft.value.trim();
  if (!content || sendingRequest.value) return;

  sendingRequest.value = true;
  autoScrollEnabled.value = true;
  let conversationId = selectedId.value;

  try {
    conversationId = selectedId.value;
    if (!conversationId) return;

    if (isDraftConversation(conversationId)) {
      creatingConversation.value = true;
      const created = await conversationApi.create();
      const nextConversation = draftConversationGroupId.value
        ? await conversationApi.update(created.id, { groupId: draftConversationGroupId.value })
        : created;
      draftConversationGroupId.value = '';
      upsertConversationListItem(nextConversation);
      revealConversationSection(nextConversation);
      conversationId = nextConversation.id;
      selectedId.value = conversationId;
      syncConversationUrl(conversationId);
      setConversationDetail({ conversation: nextConversation, eventCursor: 0, turns: [] });
    }

    if (isConversationRunning(conversationId)) return;

    setConversationDraft(draftKey, '');
    if (draftKey !== conversationId) setConversationDraft(conversationId, '');
    ensureConversationStream(conversationId, activeDetail.value?.conversation.id === conversationId ? activeDetail.value.eventCursor : 0);
    const sent = await conversationApi.send(conversationId, content);
    setConversationProcessing(conversationId, true);
    if (sent.conversation) replaceConversation(sent.conversation);
    if (selectedId.value === conversationId) {
      const selectedDetail = activeDetail.value;
      if (sent.conversation && selectedDetail?.conversation.id === conversationId) {
        selectedDetail.conversation = sent.conversation;
        setConversationDetail(selectedDetail);
      }
      const confirmedTurn: Turn = {
        id: sent.run?.id || `run-${Date.now()}`,
        status: 'inProgress',
        startedAt: sent.run?.createdAt || new Date().toISOString(),
        items: [{
          id: `local-user-${sent.run?.id || Date.now()}`,
          type: 'userMessage',
          content: [{ type: 'text', text: content }]
        }]
      };
      confirmSentTurn(conversationId, confirmedTurn);
      await scrollToBottom({ force: true });
    }
  } catch (error) {
    if (selectedId.value === conversationId) setConversationDraft(conversationId, content);
    else if (selectedId.value === draftKey) setConversationDraft(draftKey, content);
    if (activeTurnError(error)) {
      setConversationProcessing(conversationId, true);
      ensureConversationStream(conversationId, activeDetail.value?.conversation.id === conversationId ? activeDetail.value.eventCursor : 0);
      if (selectedId.value === conversationId) await refreshDetail();
      return;
    }
    setConversationProcessing(conversationId, false);
    if (selectedId.value === conversationId) resetActiveTurn();
    showError(error);
  } finally {
    creatingConversation.value = false;
    sendingRequest.value = false;
  }
}

async function cancelTurn() {
  const conversationId = selectedId.value;
  if (!conversationId || !isConversationRunning(conversationId) || stopping.value) return;
  stopping.value = true;
  finalizeStoppedTurn(conversationId);
  try {
    await conversationApi.cancelTurn(conversationId);
  } catch (error) {
    locallyStoppedSessionIds.delete(conversationId);
    setConversationProcessing(conversationId, false);
    showError(error);
  }
}

function handleComposerAction() {
  if (sending.value) {
    void cancelTurn();
    return;
  }
  void sendMessage();
}

function mergeTurnItems(current: ThreadItem[], incoming: ThreadItem[], options: { skipIncomingReasoning?: boolean } = {}) {
  if (!incoming.length) return current;

  const currentById = new Map(current.map(item => [item.id, item]));
  const pickCurrent = (incomingItem: ThreadItem): ThreadItem | undefined => {
    const byId = currentById.get(incomingItem.id);
    if (byId) return byId;
    if (incomingItem.type === 'agentMessage' && incomingItem.text) {
      return current.find(item => item.type === 'agentMessage' && item.text === incomingItem.text);
    }
    return undefined;
  };

  const merged: ThreadItem[] = [];
  const usedCurrent = new Set<ThreadItem>();
  const userMessage = incoming.find(item => item.type === 'userMessage')
    ?? current.find(item => item.type === 'userMessage');
  if (userMessage) {
    merged.push(userMessage);
    const currentUser = current.find(item => item.type === 'userMessage');
    if (currentUser) usedCurrent.add(currentUser);
  }

  for (const incomingItem of incoming) {
    if (incomingItem.type === 'userMessage') continue;
    if (options.skipIncomingReasoning && incomingItem.type === 'reasoning') continue;
    const currentItem = pickCurrent(incomingItem);
    if (currentItem) usedCurrent.add(currentItem);
    merged.push(currentItem ?? incomingItem);
  }

  for (const currentItem of current) {
    if (currentItem.type === 'userMessage' || usedCurrent.has(currentItem)) continue;
    merged.push(currentItem);
  }
  return merged;
}

function mergeTurnForDisplay(target: Turn, source: Turn, keepStatus = false) {
  if (!keepStatus) target.status = source.status;
  if (source.durationMs !== undefined) target.durationMs = source.durationMs;
  if (source.completedAt) target.completedAt = source.completedAt;
  target.items = mergeTurnItems(target.items, source.items, { skipIncomingReasoning: true });
}

function finishActiveTurn(status: 'completed' | 'failed' | 'stopped', sessionId = selectedId.value) {
  const selected = sessionId === selectedId.value;
  const durationMs = stopTurnTimer(sessionId);
  const finishedTurn = sessionId ? activeTurnsByConversation.get(sessionId) || null : null;
  if (!finishedTurn) {
    if (selected) resetActiveTurn();
    return null;
  }

  finishedTurn.status = status;
  if (finishedTurn.durationMs === undefined && durationMs !== undefined) finishedTurn.durationMs = durationMs;
  if (status === 'stopped') {
    if (finishedTurn.id) locallyStoppedRunIds.add(finishedTurn.id);
    if (sessionId) locallyStoppedSessionIds.add(sessionId);
  }
  if (sessionId) {
    setConversationProcessing(sessionId, false);
    activeTurnsByConversation.delete(sessionId);
    activeTurnResultSeenByConversation.delete(sessionId);
  }

  const targetDetail = detailsByConversation.get(sessionId);
  if (targetDetail) {
    const persisted = targetDetail.turns.find(turn => turn.id === finishedTurn.id);
    if (persisted) mergeTurnForDisplay(persisted, finishedTurn, status === 'stopped');
    else targetDetail.turns.push({ ...finishedTurn, items: [...finishedTurn.items] });
    setConversationDetail(targetDetail);
  }

  if (selected) clearDisplayedActiveTurn();
  return finishedTurn;
}

function finalizeStoppedTurn(sessionId = selectedId.value) {
  finishActiveTurn('stopped', sessionId);
}

function findStreamingItem(sessionId: string, id: string) {
  return cachedActiveTurn(sessionId)?.items.find(value => value.id === id);
}

function ensureStreamingItem(sessionId: string, id: string, type: string) {
  const turn = beginActiveTurn({ sessionId });
  let item = turn.items.find(value => value.id === id);
  if (!item) {
    item = { id, type };
    turn.items.push(item);
  }
  return item;
}

function normalizeStreamingItem(existing: ThreadItem | undefined, incoming: ThreadItem): ThreadItem {
  const next = { ...existing, ...incoming };
  if (incoming.type === 'agentMessage' && (next.phase === undefined || next.phase === null)) {
    next.phase = existing?.phase ?? 'commentary';
  }
  return next;
}

function streamAgentMessagePhase(params: { phase?: string | null; item?: ThreadItem } | undefined, existing?: ThreadItem) {
  if (params?.phase) return params.phase;
  if (params?.item?.phase) return params.item.phase;
  return existing?.phase ?? 'commentary';
}

function upsertStreamingItem(sessionId: string, item: ThreadItem) {
  const turn = beginActiveTurn({ sessionId });
  if (item.type === 'userMessage') {
    turn.items = mergeTurnItems(turn.items, [item]);
    publishActiveTurn(sessionId, turn);
    return;
  }
  const index = turn.items.findIndex(value => value.id === item.id);
  if (index >= 0) syncThreadItem(turn.items[index], normalizeStreamingItem(turn.items[index], item));
  else turn.items.push(normalizeStreamingItem(undefined, item));
  if (item.type === 'agentMessage' && item.phase === 'final_answer' && item.text) activeTurnResultSeenByConversation.set(sessionId, true);
  publishActiveTurn(sessionId, turn);
}

function userItemText(item: ThreadItem) {
  if (!Array.isArray(item.content)) return '';
  return item.content
    .filter(part => typeof part === 'object' && part?.type === 'text')
    .map(part => typeof part === 'object' ? part.text || '' : '')
    .filter(Boolean)
    .join('\n');
}

function reasoningText(item: ThreadItem) {
  const values = [...(item.summary || []), ...(Array.isArray(item.content) ? item.content.filter(value => typeof value === 'string') : [])];
  return values.join('\n\n');
}

const MarkdownBlock = defineComponent({
  name: 'MarkdownBlock',
  props: {
    content: { type: String, required: true },
    streaming: { type: Boolean, default: false }
  },
  setup(props) {
    const root = ref<HTMLElement | null>(null);
    const html = computed(() => renderMarkdown(props.content));
    const renderMermaid = () => {
      void nextTick(() => renderMermaidBlocks(root.value));
    };
    onMounted(renderMermaid);
    onUpdated(renderMermaid);
    return () => h('div', {
      ref: root,
      class: ['markdown-body', props.streaming ? 'streaming-markdown' : ''],
      onClick: copyMermaidSource,
      innerHTML: html.value
    });
  }
});

// 按 item 类型渲染不同的过程卡片，live 与完成态共用
const ProcessItemCard = defineComponent({
  name: 'ProcessItemCard',
  props: {
    display: { type: Object as PropType<ProcessDisplayItem>, required: true }
  },
  setup(props) {
    return () => {
      const display = props.display;
      const item = display.item;
      const icon = h(Icon, { icon: display.icon });

      if (item.type === 'webSearchGroup') {
        const queries = item.searchQueries || [];
        return h('details', { class: ['process-entry', 'web-search-group-entry', display.live ? 'process-entry-live' : ''] }, [
          h('summary', [
            icon,
            h('span', { class: 'entry-label' }, display.label)
          ]),
          h('div', { class: ['process-detail-list', 'web-search-group-list'] }, queries.map(query =>
            h('div', { class: ['process-detail-list-item', 'web-search-group-item'] }, [
              h(Icon, { icon: 'solar:global-linear' }),
              h('span', `已搜索网页： ${query}`)
            ])
          ))
        ]);
      }

      if (item.type === 'skillReadGroup') {
        const files = item.skillFiles || [];
        return h('details', { class: ['process-entry', 'skill-read-entry', display.live ? 'process-entry-live' : ''] }, [
          h('summary', [
            icon,
            h('span', { class: 'entry-label' }, display.label)
          ]),
          h('div', { class: ['process-detail-list', 'skill-read-list'] }, files.map(file =>
            h('div', { class: ['process-detail-list-item', 'skill-read-list-item'] }, [
              h(Icon, { icon: 'solar:document-text-linear' }),
              h('span', file)
            ])
          ))
        ]);
      }

      if (item.type === 'agentMessage' && item.text) {
        return h(MarkdownBlock, { content: item.text, streaming: display.live, class: 'process-commentary' });
      }

      const classes = ['process-entry', display.className, display.live ? 'process-entry-live' : ''];
      const title = [
        icon,
        h('span', { class: 'entry-label' }, display.label)
      ];

      if (!display.expandable) {
        return h('div', { class: [...classes, 'process-entry-title'] }, title);
      }

      const bodyClass = item.type === 'commandExecution'
        ? 'command-output'
        : item.type === 'fileChange'
          ? 'file-path-list'
          : item.type === 'webSearch'
            ? 'search-result'
            : 'activity-body';

      return h('details', { class: classes }, [
        h('summary', title),
        h(item.type === 'webSearch' ? 'div' : 'pre', { class: bodyClass }, display.detail)
      ]);
    };
  }
});

function ensureTurnTicker() {
  nowMs.value = Date.now();
  if (turnTimer !== undefined) return;
  turnTimer = window.setInterval(() => {
    nowMs.value = Date.now();
  }, 1000);
}

function stopTurnTickerIfIdle() {
  if (activeTurnStartedAtByConversation.size > 0 || processStartedAtByConversation.size > 0 || turnTimer === undefined) return;
  window.clearInterval(turnTimer);
  turnTimer = undefined;
}

function startTurnTimer(sessionId: string, startedAt?: string) {
  if (!sessionId) return;
  const parsed = startedAt ? Date.parse(startedAt) : Number.NaN;
  activeTurnStartedAtByConversation.set(sessionId, Number.isFinite(parsed) ? parsed : Date.now());
  ensureTurnTicker();
}

function startProcessTimer(sessionId: string, startedAt?: string) {
  if (!sessionId) return;
  processActiveConversationIds.add(sessionId);
  if (processStartedAtByConversation.has(sessionId)) return;
  const parsed = startedAt ? Date.parse(startedAt) : Number.NaN;
  processStartedAtByConversation.set(sessionId, Number.isFinite(parsed) ? parsed : Date.now());
  ensureTurnTicker();
}

function restoreProcessState(sessionId: string, turn: Turn, startedAt?: string) {
  if (!sessionId || !hasVisibleProcessItems(turn)) return;
  startProcessTimer(sessionId, startedAt || turn.startedAt);
}

function markProcessActive(sessionId: string) {
  startProcessTimer(sessionId);
}

function resetProcessTimer(sessionId: string) {
  if (!sessionId) return;
  processStartedAtByConversation.delete(sessionId);
  processActiveConversationIds.delete(sessionId);
}

function stopTurnTimer(sessionId: string) {
  const startedAt = activeTurnStartedAtByConversation.get(sessionId);
  activeTurnStartedAtByConversation.delete(sessionId);
  processStartedAtByConversation.delete(sessionId);
  processActiveConversationIds.delete(sessionId);
  stopTurnTickerIfIdle();
  return startedAt ? Math.max(0, Date.now() - startedAt) : undefined;
}

function stopAllTurnTimers() {
  activeTurnStartedAtByConversation.clear();
  processStartedAtByConversation.clear();
  processActiveConversationIds.clear();
  if (turnTimer !== undefined) window.clearInterval(turnTimer);
  turnTimer = undefined;
}

async function handleStreamEvent(event: StreamEvent) {
  const isSelectedConversation = event.sessionId === selectedId.value;
  const locallyStopped = Boolean(event.runId && locallyStoppedRunIds.has(event.runId)) || locallyStoppedSessionIds.has(event.sessionId);

  if (event.type === 'run.started') {
    resetProcessTimer(event.sessionId);
    setConversationProcessing(event.sessionId, true);
  }

  if (locallyStopped && event.type !== 'run.completed' && event.type !== 'run.failed') return;

  if (event.type === 'run.started') {
    setConversationProcessing(event.sessionId, true);
    const data = event.data as { startedAt?: string } | undefined;
    beginActiveTurn({
      sessionId: event.sessionId,
      id: event.runId || undefined,
      status: 'inProgress',
      startedAt: data?.startedAt || event.createdAt,
      resetResultSeen: true
    });
    if (isSelectedConversation) await scrollToBottom();
    return;
  }

  if (event.type === 'codex.turn.started') {
    const turn = (event.data as { params?: { turn?: Record<string, unknown> } } | undefined)?.params?.turn;
    if (turn) {
      const startedAt = Number(turn.startedAt);
      const startedAtIso = Number.isFinite(startedAt) && startedAt > 0 ? new Date(startedAt * 1000).toISOString() : undefined;
      beginActiveTurn({
        sessionId: event.sessionId,
        id: String(turn.id || cachedActiveTurn(event.sessionId)?.id || ''),
        status: String(turn.status || 'inProgress'),
        startedAt: startedAtIso,
        restartTimer: Boolean(startedAtIso)
      });
    }
    return;
  }

  if (event.type === 'codex.turn.completed') {
    const turn = (event.data as { params?: { turn?: Record<string, unknown> } } | undefined)?.params?.turn;
    if (turn) {
      const activeTurn = beginActiveTurn({
        sessionId: event.sessionId,
        id: String(turn.id || cachedActiveTurn(event.sessionId)?.id || ''),
        status: String(turn.status || 'completed')
      });
      const durationMs = Number(turn.durationMs);
      if (Number.isFinite(durationMs) && durationMs >= 0) activeTurn.durationMs = durationMs;
      const completedAt = Number(turn.completedAt);
      if (Number.isFinite(completedAt) && completedAt > 0) {
        activeTurn.completedAt = new Date(completedAt * 1000).toISOString();
      }
      if (Array.isArray(turn.items)) {
        // turn.completed 快照只含 userMessage/agentMessage，统一走 mergeTurnItems，
        // 让服务端 userMessage 替换本地确认消息，同时保留过程项。
        activeTurn.items = mergeTurnItems(activeTurn.items, turn.items as ThreadItem[]);
      }
      publishActiveTurn(event.sessionId, activeTurn);
    }
    return;
  }

  if (event.type.includes('task_started')) {
    const data = event.data as { params?: Record<string, unknown> } | undefined;
    const payload = ((data?.params?.msg || data?.params?.payload || data?.params) ?? {}) as Record<string, unknown>;
    const startedAt = Number(payload.started_at);
    if (Number.isFinite(startedAt) && startedAt > 0) {
      beginActiveTurn({ sessionId: event.sessionId, startedAt: new Date(startedAt * 1000).toISOString() });
    }
    return;
  }

  if (event.type.includes('task_complete')) {
    const data = event.data as { params?: Record<string, unknown> } | undefined;
    const payload = ((data?.params?.msg || data?.params?.payload || data?.params) ?? {}) as Record<string, unknown>;
    const durationMs = Number(payload.duration_ms);
    if (Number.isFinite(durationMs) && durationMs >= 0) {
      const startedAt = Date.now() - durationMs;
      activeTurnStartedAtByConversation.set(event.sessionId, startedAt);
      ensureTurnTicker();
    }
  }

  if (event.type === 'codex.item.reasoning.textDelta') {
    const params = (event.data as { params?: { delta?: string; itemId?: string; contentIndex?: number } } | undefined)?.params;
    const delta = params?.delta ?? '';
    if (!delta) return;
    const itemId = params?.itemId || `stream-reasoning-${event.runId || 'active'}`;
    const item = ensureStreamingItem(event.sessionId, itemId, 'reasoning');
    const content = (Array.isArray(item.content) ? item.content : []) as unknown as string[];
    const index = Number(params?.contentIndex) || 0;
    while (content.length <= index) content.push('');
    content[index] = `${content[index] || ''}${delta}`;
    item.content = content;
    publishActiveTurn(event.sessionId);
    if (isSelectedConversation) await scrollToBottom();
    return;
  }

  if (event.type === 'codex.item.agentMessage.delta') {
    const params = (event.data as { params?: { delta?: string; itemId?: string; phase?: string | null; item?: ThreadItem } } | undefined)?.params;
    const delta = params?.delta ?? '';
    if (!delta) return;
    const itemId = params?.itemId || params?.item?.id || `stream-agent-${event.runId || 'active'}`;
    const existingItem = findStreamingItem(event.sessionId, itemId);
    const phase = streamAgentMessagePhase(params, existingItem);
    if (phase !== 'final_answer') markProcessActive(event.sessionId);
    const item = ensureStreamingItem(event.sessionId, itemId, 'agentMessage');
    item.phase = phase;
    item.text = `${item.text || ''}${delta}`;
    publishActiveTurn(event.sessionId);
    if (item.phase === 'final_answer' && item.text) activeTurnResultSeenByConversation.set(event.sessionId, true);
    if (isSelectedConversation) await scrollToBottom();
    return;
  }

  if (event.type === 'codex.item.started') {
    const params = (event.data as { params?: { item?: ThreadItem } } | undefined)?.params;
    const streamedItem = params?.item as ThreadItem | undefined;
    if (streamedItem) {
      if (isVisibleProcessStreamItem(streamedItem)) markProcessActive(event.sessionId);
      upsertStreamingItem(event.sessionId, { ...streamedItem, streamStatus: 'running' });
      if (streamedItem.type === 'agentMessage' && streamedItem.phase === 'final_answer' && streamedItem.text) {
        activeTurnResultSeenByConversation.set(event.sessionId, true);
      }
    }
    if (isSelectedConversation) await scrollToBottom();
    return;
  }

  if (event.type === 'codex.item.completed') {
    const params = (event.data as { params?: { item?: ThreadItem } } | undefined)?.params;
    if (params?.item) {
      const completedItem = { ...(params.item as ThreadItem), streamStatus: 'completed' };
      if (isVisibleProcessStreamItem(completedItem)) markProcessActive(event.sessionId);
      upsertStreamingItem(event.sessionId, completedItem);
      if (completedItem.type === 'agentMessage' && completedItem.phase === 'final_answer' && completedItem.text) {
        activeTurnResultSeenByConversation.set(event.sessionId, true);
      }
    }
    if (isSelectedConversation) await scrollToBottom();
    return;
  }

  if (event.type === 'run.completed' || event.type === 'run.failed') {
    if (locallyStopped) {
      if (event.runId) locallyStoppedRunIds.delete(event.runId);
      locallyStoppedSessionIds.delete(event.sessionId);
      setConversationProcessing(event.sessionId, false);
      await loadConversations();
      return;
    }

    const completedSessionId = event.sessionId;
    setConversationProcessing(completedSessionId, false);
    const completedTurn = finishActiveTurn(event.type === 'run.completed' ? 'completed' : 'failed', completedSessionId);
    await nextTick();
    await loadConversations();
    closeIdleConversationStreams();
    if (completedTurn) {
      const targetDetail = detailsByConversation.get(completedSessionId);
      if (targetDetail) {
        const persisted = targetDetail.turns.find(turn => turn.id === completedTurn.id);
        if (persisted) mergeTurnForDisplay(persisted, completedTurn);
        else targetDetail.turns.push(completedTurn);
        setConversationDetail(targetDetail);
      }
    }
  }
}

function openRename() {
  if (!activeConversation.value) return;
  renameValue.value = activeConversation.value.title;
  renameVisible.value = true;
}

async function saveRename() {
  const title = renameValue.value.trim();
  if (!activeConversation.value || !title) return;
  try {
    const conversationId = activeConversation.value.id;
    const updated = await conversationApi.update(conversationId, { title });
    replaceConversation(updated);
    if (detail.value?.conversation.id === conversationId) {
      detail.value.conversation = { ...detail.value.conversation, ...updated };
    }
    renameVisible.value = false;
  } catch (error) {
    showError(error);
  }
}

async function togglePinned() {
  if (!activeConversation.value) return;
  await toggleConversationPinned(activeConversation.value);
}

async function toggleConversationPinned(item: Conversation) {
  try {
    const nextPinned = !item.pinnedAt;
    const updated = await conversationApi.update(item.id, {
      pinned: nextPinned
    });
    const merged = { ...item, ...updated, pinnedAt: nextPinned ? updated.pinnedAt || item.pinnedAt || new Date().toISOString() : undefined };
    replaceConversation(merged);
    if (detail.value?.conversation.id === item.id) {
      detail.value.conversation = { ...detail.value.conversation, ...merged };
    }
    await loadConversations();
    const refreshed = conversations.value.find(value => value.id === item.id);
    revealConversationSection({ ...merged, ...refreshed, pinnedAt: nextPinned ? refreshed?.pinnedAt || merged.pinnedAt : undefined });
  } catch (error) {
    showError(error);
  }
}

async function toggleArchived() {
  if (!activeConversation.value) return;
  await archiveConversation(activeConversation.value);
}

async function archiveConversation(item: Conversation) {
  const wasSelected = item.id === selectedId.value;
  try {
    await conversationApi.update(item.id, { archived: true });
    clearConversationDetail(item.id);
    setConversationProcessing(item.id, false);
    if (wasSelected) {
      closeConversationStream(item.id);
      selectedId.value = '';
      syncConversationUrl('');
      detail.value = null;
      resetActiveTurn();
    }
    await loadConversations(wasSelected);
  } catch (error) {
    showError(error);
  }
}

function confirmDelete() {
  if (!activeConversation.value) return;
  openConfirm(
    '删除对话',
    `确定删除「${activeConversation.value.title}」吗？此操作不可撤销。`,
    '删除',
    deleteConversation
  );
}

async function deleteConversation() {
  if (!activeConversation.value) return;
  try {
    const removedId = activeConversation.value.id;
    await conversationApi.remove(removedId);
    clearConversationDetail(removedId);
    workspacesByConversation.delete(removedId);
    workspaceVisibilityByConversation.delete(removedId);
    setConversationProcessing(removedId, false);
    closeConversationStream(removedId);
    selectedId.value = '';
    draftConversationGroupId.value = '';
    syncConversationUrl('');
    detail.value = null;
    resetActiveTurn();
    await loadConversations(true);
  } catch (error) {
    showError(error);
  }
}

function replaceConversation(updated: Conversation) {
  const index = conversations.value.findIndex(item => item.id === updated.id);
  if (index < 0) return;
  const current = conversations.value[index];
  const running = updated.running || current.running || processingConversationIds.value.has(updated.id);
  conversations.value[index] = { ...current, ...updated, running };
}

function handleComposerKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter') return;
  if (event.isComposing || event.keyCode === 229) return;

  if (event.shiftKey) {
    event.preventDefault();
    event.stopPropagation();
    const textarea = event.target;
    if (!(textarea instanceof HTMLTextAreaElement) || textarea.value.length >= 12000) return;
    const start = textarea.selectionStart ?? textarea.value.length;
    const end = textarea.selectionEnd ?? start;
    draft.value = `${textarea.value.slice(0, start)}\n${textarea.value.slice(end)}`;
    void nextTick(() => textarea.setSelectionRange(start + 1, start + 1));
    return;
  }

  if (event.altKey || event.ctrlKey || event.metaKey) return;
  event.preventDefault();
  event.stopPropagation();
  void sendMessage();
}

function showError(error: unknown) {
  toast.error(error instanceof Error ? error.message : '操作失败');
}

function isMessagePaneNearBottom(pane: HTMLElement, threshold = 96) {
  return pane.scrollHeight - pane.scrollTop - pane.clientHeight <= threshold;
}

function handleMessageScroll(event: Event) {
  const pane = event.currentTarget;
  if (pane instanceof HTMLElement) {
    autoScrollEnabled.value = isMessagePaneNearBottom(pane);
  }
}

async function scrollToBottom(options: { force?: boolean } = {}) {
  await nextTick();
  await new Promise(resolve => requestAnimationFrame(resolve));
  // 首次进入对话时消息面板可能还没渲染出来，轮询等待它挂载后再滚动
  let pane = messagePane.value || document.querySelector<HTMLElement>('.message-pane');
  for (let i = 0; i < 80 && !pane; i++) {
    await new Promise(resolve => setTimeout(resolve, 50));
    pane = messagePane.value || document.querySelector<HTMLElement>('.message-pane');
  }
  if (!pane) return;
  if (!options.force && !autoScrollEnabled.value && !isMessagePaneNearBottom(pane)) return;
  pane.scrollTo({ top: pane.scrollHeight, behavior: 'auto' });
  autoScrollEnabled.value = true;
}

function messagePreview(content: string) {
  const summary = content.replace(/\s+/g, ' ').trim();
  return summary.length > 72 ? `${summary.slice(0, 72)}…` : summary;
}

function jumpToMessage(id: string) {
  const target = document.getElementById(`message-${id}`);
  if (!target) return;
  target.scrollIntoView({ behavior: 'smooth', block: 'start' });
  target.focus({ preventScroll: true });
}

async function bootstrap() {
  // 先记住 URL 上的设置页路径（/settings、/settings/archives）
  const settingsPath = window.location.pathname.match(/^\/settings(?:\/([^/]+))?/);
  setAuthErrorHandler(() => {
    currentUser.value = null;
    conversations.value = [];
    settingsVisible.value = false;
    selectedId.value = '';
    syncConversationUrl('');
    detail.value = null;
    loadingDetail.value = false;
    loadingDetailId.value = '';
    detailsByConversation.clear();
    sendingRequest.value = false;
    creatingConversation.value = false;
    resetActiveTurn();
    activeTurnsByConversation.clear();
    activeTurnResultSeenByConversation.clear();
    stopAllTurnTimers();
    processingConversationIds.value = new Set();
    closeAllConversationStreams();
  });
  try {
    if (getToken()) {
      currentUser.value = await authApi.getUserInfo();
      await Promise.all([loadConversationGroups(), loadConversations(true)]);
      // 刷新后从 URL 恢复设置页
      if (settingsPath) {
        settingsSection.value = settingsPath[1] === 'archives' ? 'archives' : 'appearance';
        settingsVisible.value = true;
      }
    }
  } catch {
    currentUser.value = null;
  } finally {
    authChecking.value = false;
  }
}

async function submitLogin() {
  const username = loginUsername.value.trim();
  const password = loginPassword.value;
  if (!username || !password) return;
  loginLoading.value = true;
  try {
    const { token, refreshToken } = await authApi.pwdLogin(username, password);
    setToken(token);
    setRefreshToken(refreshToken);
    currentUser.value = await authApi.getUserInfo();
    await loadConversationGroups();
    loginVisible.value = false;
    loginUsername.value = '';
    loginPassword.value = '';
    await loadConversations(true);
  } catch (error) {
    showError(error);
  } finally {
    loginLoading.value = false;
  }
}

function logout() {
  mobileSidebarOpen.value = false;
  userMenuVisible.value = false;
  clearToken();
  clearRefreshToken();
  settingsVisible.value = false;
  currentUser.value = null;
  conversationGroups.value = [];
  closeAllConversationStreams();
  selectedId.value = '';
  draftConversationGroupId.value = '';
  syncConversationUrl('');
  detail.value = null;
  loadingDetail.value = false;
  loadingDetailId.value = '';
  detailsByConversation.clear();
  conversations.value = [];
  sendingRequest.value = false;
  creatingConversation.value = false;
  resetActiveTurn();
  activeTurnsByConversation.clear();
  activeTurnResultSeenByConversation.clear();
  stopAllTurnTimers();
  processingConversationIds.value = new Set();
}

function openSettings() {
  mobileSidebarOpen.value = false;
  userMenuVisible.value = false;
  settingsReturnPath.value = location.pathname;
  settingsSection.value = 'appearance';
  settingsNavExpanded.value = true;
  settingsVisible.value = true;
  void loadConversations();
}

function openArchiveSettings() {
  settingsSection.value = 'archives';
  void loadConversations();
}

async function restoreArchived(item: Conversation) {
  try {
    await conversationApi.update(item.id, { archived: false });
    await loadConversations();
    toast.success('对话已恢复');
  } catch (error) {
    showError(error);
  }
}

function confirmDeleteArchived(item: Conversation) {
  openConfirm(
    '删除归档对话',
    `确定删除「${item.title}」吗？此操作不可撤销。`,
    '删除',
    async () => {
      try {
        await conversationApi.remove(item.id);
        clearConversationDetail(item.id);
        await loadConversations();
      } catch (error) {
        showError(error);
      }
    }
  );
}

function openLogin() {
  loginVisible.value = true;
}

function setAppearance(value: Appearance) {
  appearance.value = value;
  localStorage.setItem(APPEARANCE_KEY, value);
}

function formatConversationDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? ''
    : new Intl.DateTimeFormat('zh-CN', {
        year: 'numeric', month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit'
      }).format(date);
}

function isSameDay(left: Date, right: Date) {
  return left.getFullYear() === right.getFullYear()
    && left.getMonth() === right.getMonth()
    && left.getDate() === right.getDate();
}

function formatMessageTime(value?: string) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const now = new Date();
  const options: Intl.DateTimeFormatOptions = isSameDay(date, now)
    ? { hour: '2-digit', minute: '2-digit' }
    : date.getFullYear() === now.getFullYear()
      ? { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }
      : { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' };
  return new Intl.DateTimeFormat('zh-CN', options).format(date);
}

function confirmDeleteAllArchived() {
  if (!archivedConversations.value.length) return;
  openConfirm(
    '删除全部归档对话',
    `确定永久删除全部 ${archivedConversations.value.length} 个归档对话吗？此操作不可撤销。`,
    '全部删除',
    async () => {
      try {
        const removedIds = archivedConversations.value.map(item => item.id);
        await Promise.all(removedIds.map(id => conversationApi.remove(id)));
        removedIds.forEach(id => clearConversationDetail(id));
        clearConversationProcessing(removedIds);
        await loadConversations();
      } catch (error) {
        showError(error);
      }
    }
  );
}

function confirmDeleteGroupArchived(section: { title: string; groupId?: string; items: Conversation[] }) {
  if (!section.groupId || !section.items.length) return;
  openConfirm(
    '删除分组归档对话',
    `确定永久删除「${section.title}」下的全部 ${section.items.length} 个归档对话吗？此操作不可撤销。`,
    '全部删除',
    async () => {
      try {
        await conversationGroupApi.deleteArchivedConversations(section.groupId!);
        await loadConversations();
      } catch (error) {
        showError(error);
      }
    }
  );
}

function openConfirm(title: string, content: string, positiveText: string, action: () => Promise<void>) {
  confirmTitle.value = title;
  confirmContent.value = content;
  confirmPositiveText.value = positiveText;
  confirmAction = action;
  confirmVisible.value = true;
}

function closeConfirm() {
  if (confirmLoading.value) return;
  confirmVisible.value = false;
  confirmAction = undefined;
}

async function submitConfirm() {
  if (!confirmAction || confirmLoading.value) return;
  confirmLoading.value = true;
  try {
    await confirmAction();
    confirmVisible.value = false;
    confirmAction = undefined;
  } finally {
    confirmLoading.value = false;
  }
}

function handleSystemAppearanceChange(event: MediaQueryListEvent) {
  systemDark.value = event.matches;
}

function handleDocumentPointerDown(event: PointerEvent) {
  if (!userMenuVisible.value) return;
  const target = event.target;
  if (target instanceof Node && !userMenuWrap.value?.contains(target)) {
    userMenuVisible.value = false;
  }
}

function handleDocumentPointerOver(event: PointerEvent) {
  if (!conversationPreview.value) return;
  const target = event.target;
  if (!(target instanceof Element) || !target.closest('.previewable-conversation-row')) {
    hideConversationPreview();
  }
}

function startWorkspaceResize(event: PointerEvent) {
  if (window.matchMedia('(max-width: 720px)').matches) return;
  const panel = mainPanel.value?.querySelector<HTMLElement>('.workspace-panel');
  if (!panel) return;
  workspaceResizeStart = { x: event.clientX, width: panel.getBoundingClientRect().width };
  document.body.classList.add('workspace-resizing');
  event.preventDefault();
}

function resizeWorkspace(event: PointerEvent) {
  if (!workspaceResizeStart) return;
  const availableWidth = mainPanel.value?.clientWidth || window.innerWidth;
  const maxWidth = Math.max(360, availableWidth - 320);
  workspaceWidth.value = Math.min(
    maxWidth,
    Math.max(360, workspaceResizeStart.width + workspaceResizeStart.x - event.clientX)
  );
}

function stopWorkspaceResize() {
  if (!workspaceResizeStart) return;
  workspaceResizeStart = null;
  document.body.classList.remove('workspace-resizing');
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown);
  document.addEventListener('pointerover', handleDocumentPointerOver);
  window.addEventListener('pointermove', onDragPointerMove);
  window.addEventListener('pointermove', resizeWorkspace);
  window.addEventListener('pointerup', onDragPointerUp);
  window.addEventListener('pointerup', stopWorkspaceResize);
  window.addEventListener('touchmove', onWindowTouchMove, { passive: false });
  window.addEventListener('touchend', onWindowTouchEnd);
  window.addEventListener('touchcancel', onWindowTouchEnd);
  window.addEventListener('blur', hideConversationPreview);
  colorSchemeMedia.addEventListener('change', handleSystemAppearanceChange);
  void bootstrap();
  // 防止登录恢复流程异常卡死导致一直白屏：超时后强制结束启动态
  window.setTimeout(() => {
    if (authChecking.value) authChecking.value = false;
  }, 10000);
});
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown);
  document.removeEventListener('pointerover', handleDocumentPointerOver);
  window.removeEventListener('pointermove', onDragPointerMove);
  window.removeEventListener('pointermove', resizeWorkspace);
  window.removeEventListener('pointerup', onDragPointerUp);
  window.removeEventListener('pointerup', stopWorkspaceResize);
  window.removeEventListener('touchmove', onWindowTouchMove);
  window.removeEventListener('touchend', onWindowTouchEnd);
  window.removeEventListener('touchcancel', onWindowTouchEnd);
  window.removeEventListener('blur', hideConversationPreview);
  colorSchemeMedia.removeEventListener('change', handleSystemAppearanceChange);
  closeAllConversationStreams();
  stopAllTurnTimers();
});
watch(selectedId, () => {
  workspaceExpanded.value = false;
  pinnedSummaryOpen.value = false;
});
watch(isDarkAppearance, value => {
  document.documentElement.dataset.theme = value ? 'dark' : 'light';
  toastProviderProps.theme = activeTheme.value;
}, { immediate: true });
watch(
  [sidebarCollapsed, pinnedExpanded, groupsExpanded, conversationsExpanded, conversationGroups],
  () => {
    const collapsedGroups = Object.fromEntries(
      conversationGroups.value.map(group => [group.id, group.collapsed])
    );
    localStorage.setItem(SIDEBAR_VIEW_KEY, JSON.stringify({
      sidebarCollapsed: sidebarCollapsed.value,
      pinnedExpanded: pinnedExpanded.value,
      groupsExpanded: groupsExpanded.value,
      conversationsExpanded: conversationsExpanded.value,
      collapsedGroups
    } satisfies SidebarViewState));
  },
  { deep: true }
);
watch([settingsVisible, settingsSection], () => {
  syncSettingsUrl();
});
</script>

<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN" :theme="activeTheme" :theme-overrides="themeOverrides">
    <div v-if="authChecking" class="app-boot-screen" aria-label="正在加载对话">
      <div class="app-boot-card">
        <img class="app-boot-logo" src="/agentrazor-icon.png" alt="" />
        <div class="app-boot-copy">
          <div class="app-boot-title">AgentRazor</div>
          <div class="app-boot-text">正在加载对话</div>
        </div>
      </div>
    </div>
    <div v-else class="app-shell" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
      <aside class="sidebar" :class="{ 'mobile-open': mobileSidebarOpen }">
        <div class="brand">
          <div class="brand-mark"><img src="/agentrazor-icon.png" alt="" /></div>
          <div v-if="!sidebarCollapsed" class="brand-copy">
            <strong>AgentRazor</strong>
          </div>
          <n-button v-if="!sidebarCollapsed" quaternary circle class="collapse-button" @click="sidebarCollapsed = true">
            <template #icon>
              <Icon icon="solar:sidebar-code-outline" />
            </template>
          </n-button>
        </div>

        <n-button class="new-chat" secondary @click="createConversation">
          <template #icon><Icon icon="solar:pen-new-square-outline" /></template>
          <span v-if="!sidebarCollapsed">新建对话</span>
        </n-button>

        <div v-if="!sidebarCollapsed" class="sidebar-section">
          <n-spin :show="loadingList">
            <n-scrollbar class="conversation-scroll" @scroll="hideConversationPreview">
              <div v-if="pinnedConversations.length" class="conversation-group">
                <button class="conversation-group-toggle" @click="pinnedExpanded = !pinnedExpanded">
                  <span>置顶</span>
                  <Icon :icon="pinnedExpanded ? 'solar:alt-arrow-down-linear' : 'solar:alt-arrow-right-linear'" />
                </button>
                <template v-if="pinnedExpanded">
                  <div
                    v-for="item in pinnedConversations"
                    :key="item.id"
                    class="conversation-row previewable-conversation-row"
                    :class="{ selected: item.id === selectedId, dragging: item.id === draggedConversationId }"
                    @pointerdown="onConversationPointerDown(item, $event)"
                    @touchstart="onRowTouchStart(item, $event)"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)">
                      <Icon icon="solar:pin-bold" class="pin-icon" />
                      <span>{{ item.title }}</span>
                    </button>
                    <button
                      v-if="isConversationProcessing(item)"
                      type="button"
                      class="conversation-processing-button"
                      aria-label="处理中"
                      title="处理中"
                      disabled
                    >
                      <Icon icon="solar:refresh-linear" />
                    </button>
                    <div class="conversation-actions">
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="取消置顶" @click="toggleConversationPinned(item)">
                            <Icon icon="solar:pin-bold" />
                          </button>
                        </template>
                        取消置顶
                      </n-tooltip>
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="归档" @click="archiveConversation(item)">
                            <Icon icon="solar:archive-linear" />
                          </button>
                        </template>
                        归档
                      </n-tooltip>
                    </div>
                  </div>
                </template>
              </div>
              <div class="group-section-heading">
                <button class="group-section-toggle" type="button" @click="groupsExpanded = !groupsExpanded">
                  <span>分组</span>
                  <Icon :icon="groupsExpanded ? 'solar:alt-arrow-down-linear' : 'solar:alt-arrow-right-linear'" />
                </button>
                <n-tooltip trigger="hover" placement="top">
                  <template #trigger>
                    <button class="group-add-button" type="button" aria-label="新增分组" @click="openCreateGroup()">
                      <Icon icon="solar:add-circle-linear" />
                    </button>
                  </template>
                  新增分组
                </n-tooltip>
              </div>
              <template v-if="groupsExpanded">
              <div
                v-for="group in conversationGroups"
                :key="group.id"
                class="conversation-group custom-group conversation-drop-zone"
                :class="{ 'drag-over': conversationDropTarget === group.id }"
                :data-group-id="group.id"
              >
                <div class="custom-group-heading">
                  <button class="conversation-group-toggle" @click="toggleGroup(group)">
                    <Icon icon="solar:folder-linear" class="group-folder-icon" />
                    <span>{{ group.name }}</span>
                  </button>
                  <div class="group-actions">
                    <n-tooltip trigger="hover" placement="top">
                      <template #trigger>
                        <button type="button" aria-label="在分组中新增对话" @click="createConversationInGroup(group)">
                          <Icon icon="solar:pen-new-square-outline" />
                        </button>
                      </template>
                      新增对话
                    </n-tooltip>
                    <n-dropdown
                      trigger="click"
                      placement="right-start"
                      :options="[
                        { label: '重命名分组', key: 'rename', icon: renderIcon('solar:pen-2-linear') },
                        { label: '归档对话', key: 'archiveConversations', icon: renderIcon('solar:archive-linear') },
                        { type: 'divider', key: 'divider' },
                        { label: '删除分组', key: 'delete', icon: renderIcon('solar:trash-bin-trash-linear') }
                      ]"
                      @select="key => handleGroupAction(group, String(key))"
                    >
                      <button type="button" aria-label="更多分组操作"><Icon icon="solar:menu-dots-bold" /></button>
                    </n-dropdown>
                  </div>
                </div>
                <template v-if="!group.collapsed">
                  <div
                    v-for="item in conversationsInGroup(group.id)"
                    :key="item.id"
                    class="conversation-row grouped-conversation previewable-conversation-row"
                    :class="{ selected: item.id === selectedId, dragging: item.id === draggedConversationId }"
                    @pointerdown="onConversationPointerDown(item, $event)"
                    @touchstart="onRowTouchStart(item, $event)"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)">
                      <span>{{ item.title }}</span>
                    </button>
                    <button
                      v-if="isConversationProcessing(item)"
                      type="button"
                      class="conversation-processing-button"
                      aria-label="处理中"
                      title="处理中"
                      disabled
                    >
                      <Icon icon="solar:refresh-linear" />
                    </button>
                    <div class="conversation-actions">
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button
                            type="button"
                            :aria-label="item.pinnedAt ? '取消置顶' : '置顶'"
                            @click="toggleConversationPinned(item)"
                          >
                            <Icon :icon="item.pinnedAt ? 'solar:pin-bold' : 'solar:pin-linear'" />
                          </button>
                        </template>
                        {{ item.pinnedAt ? '取消置顶' : '置顶' }}
                      </n-tooltip>
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="归档" @click="archiveConversation(item)"><Icon icon="solar:archive-linear" /></button>
                        </template>
                        归档
                      </n-tooltip>
                    </div>
                  </div>
                  <div v-if="!conversationsInGroup(group.id).length" class="sidebar-empty-state">
                    <Icon icon="solar:inbox-line-outline" />
                    <span>暂无对话</span>
                  </div>
                </template>
              </div>
              </template>
              <div
                class="conversation-group conversation-list-group conversation-drop-zone"
                :class="{ 'drag-over': conversationDropTarget === CONVERSATION_LIST_DROP_TARGET }"
                :data-drop-target="CONVERSATION_LIST_DROP_TARGET"
              >
                <div class="conversation-list-heading">
                  <button class="conversation-group-toggle" @click="conversationsExpanded = !conversationsExpanded">
                    <span>对话</span>
                    <Icon :icon="conversationsExpanded ? 'solar:alt-arrow-down-linear' : 'solar:alt-arrow-right-linear'" />
                  </button>
                  <n-tooltip trigger="hover" placement="top">
                    <template #trigger>
                      <button class="group-add-button" type="button" aria-label="新建对话" @click="createConversation">
                        <Icon icon="solar:pen-new-square-outline" />
                      </button>
                    </template>
                    新建对话
                  </n-tooltip>
                </div>
                <template v-if="conversationsExpanded">
                  <div
                    v-for="item in conversationList"
                    :key="item.id"
                    class="conversation-row previewable-conversation-row"
                    :class="{ selected: item.id === selectedId, dragging: item.id === draggedConversationId }"
                    @pointerdown="onConversationPointerDown(item, $event)"
                    @touchstart="onRowTouchStart(item, $event)"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)">
                      <span>{{ item.title }}</span>
                    </button>
                    <button
                      v-if="isConversationProcessing(item)"
                      type="button"
                      class="conversation-processing-button"
                      aria-label="处理中"
                      title="处理中"
                      disabled
                    >
                      <Icon icon="solar:refresh-linear" />
                    </button>
                    <div class="conversation-actions">
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="置顶" @click="toggleConversationPinned(item)">
                            <Icon icon="solar:pin-linear" />
                          </button>
                        </template>
                        置顶
                      </n-tooltip>
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="归档" @click="archiveConversation(item)">
                            <Icon icon="solar:archive-linear" />
                          </button>
                        </template>
                        归档
                      </n-tooltip>
                    </div>
                  </div>
                </template>
              </div>
              <div v-if="conversationsExpanded && !loadingList && !conversationList.length" class="sidebar-empty-state">
                <Icon icon="solar:inbox-line-outline" />
                <span>暂无对话</span>
              </div>
            </n-scrollbar>
          </n-spin>
        </div>

        <div
          v-if="!sidebarCollapsed"
          class="sidebar-footer"
          :class="{ 'login-footer': !currentUser }"
        >
          <template v-if="currentUser">
            <div ref="userMenuWrap" class="user-menu-wrap" @keydown.esc="userMenuVisible = false">
              <div v-if="userMenuVisible" class="user-popover" role="menu">
                <div class="user-popover-account">
                  <span class="user-avatar">{{ userInitial }}</span>
                  <div>
                    <strong>{{ currentUser.username }}</strong>
                    <span>当前登录账号</span>
                  </div>
                </div>
                <button type="button" role="menuitem" @click="openSettings">
                  <Icon icon="solar:settings-linear" />
                  <span>设置</span>
                </button>
                <button type="button" role="menuitem" class="logout-menu-item" @click="logout">
                  <Icon icon="solar:logout-2-linear" />
                  <span>退出登录</span>
                </button>
              </div>
              <button
                class="user-menu-trigger"
                type="button"
                aria-haspopup="menu"
                :aria-expanded="userMenuVisible"
                @click="userMenuVisible = !userMenuVisible"
              >
                <span class="user-avatar">{{ userInitial }}</span>
                <span class="user-name">{{ currentUser.username }}</span>
              </button>
            </div>
          </template>
          <button v-else type="button" class="login-entry" @click.stop="openLogin">
            <span class="login-entry-icon"><Icon icon="solar:login-3-linear" /></span>
            <span class="login-entry-copy">
              <strong>登录 AgentRazor</strong>
              <span>同步你的对话</span>
            </span>
          </button>
        </div>
      </aside>

      <div
        v-if="mobileSidebarOpen"
        class="sidebar-backdrop"
        @click="mobileSidebarOpen = false"
      />

      <div
        v-if="conversationPreview && !sidebarCollapsed"
        class="conversation-hover-card"
        :style="{ top: `${conversationPreviewTop}px` }"
      >
        <strong>{{ conversationPreview.title }}</strong>
        <div class="pinned-preview-meta">
          <Icon icon="solar:clock-circle-linear" />
          <span>{{ formatConversationDate(conversationPreview.updatedAt) }}</span>
        </div>
        <div v-if="conversationGroupName(conversationPreview)" class="pinned-preview-meta">
          <Icon icon="solar:folder-linear" />
          <span>{{ conversationGroupName(conversationPreview) }}</span>
        </div>
      </div>

      <main
        ref="mainPanel"
        class="main-panel"
        :class="{ 'workspace-open': activeWorkspace && workspaceVisible, 'workspace-expanded': workspaceExpanded }"
        :style="workspacePanelStyle"
      >
        <header class="topbar">
          <n-button quaternary circle class="mobile-menu-button" @click="openMobileSidebar">
            <template #icon><Icon icon="solar:sidebar-minimalistic-outline" /></template>
          </n-button>
          <n-button v-if="sidebarCollapsed" quaternary circle class="topbar-sidebar-toggle" @click="sidebarCollapsed = false">
            <template #icon><Icon icon="solar:sidebar-minimalistic-outline" /></template>
          </n-button>
          <div class="topbar-inner">
            <div v-if="activeConversation && activeConversation.title !== '新对话'" class="conversation-heading">
              <Icon icon="solar:chat-round-dots-outline" />
              <strong>{{ activeConversation?.title || 'AgentRazor' }}</strong>
              <n-dropdown
                v-if="activeConversation"
                trigger="click"
                placement="bottom-start"
                :options="menuOptions"
                @select="key => key === 'rename' ? openRename() : key === 'delete' ? confirmDelete() : key === 'pin' ? togglePinned() : toggleArchived()"
              >
                <n-button quaternary circle><template #icon><Icon icon="solar:menu-dots-bold" /></template></n-button>
              </n-dropdown>
            </div>
          </div>
        </header>

        <n-button
          v-if="pinnedSummaryWorkspaces.length && !workspaceVisible"
          quaternary
          class="right-panel-toggle topbar-right-panel-toggle"
          aria-label="Toggle pinned summary"
          title="Toggle pinned summary"
          @click="toggleRightPanel"
        >
          <template #icon><Icon icon="solar:sidebar-minimalistic-outline" /></template>
        </n-button>

        <aside v-if="pinnedSummaryOpen" class="pinned-summary-panel">
          <header>
            <strong>Pinned summary</strong>
          </header>
          <button
            v-for="workspace in pinnedSummaryWorkspaces"
            :key="workspace.url"
            type="button"
            class="pinned-summary-item"
            @click="openPinnedWorkspace(workspace)"
          >
            <Icon icon="solar:widget-5-linear" />
            <span><strong>{{ workspace.title }}</strong><small>{{ workspace.url }}</small></span>
            <Icon icon="solar:arrow-right-linear" />
          </button>
        </aside>

        <section v-if="conversationOpening" class="conversation-opening" aria-label="正在加载对话">
          <div class="conversation-opening-indicator">
            <Icon icon="solar:chat-round-dots-outline" />
          </div>
          <div class="conversation-opening-copy">
            <strong>正在加载对话</strong>
            <span>正在同步消息和处理状态</span>
          </div>
        </section>

        <section v-else-if="selectedId && isNewChat" class="chat-empty">
          <div class="welcome">
            <div class="welcome-icon"><img src="/agentrazor-icon.png" alt="" /></div>
            <h1>今天想完成什么？</h1>
            <p>输入目标，AgentRazor 会在当前对话中持续处理。</p>
          </div>
          <div class="composer">
            <n-input
              v-model:value="draft"
              type="textarea"
              autosize
              :maxlength="12000"
              placeholder="给 AgentRazor 发送消息"
              @keydown="handleComposerKeydown"
            />
            <div class="composer-footer">
              <n-button
                type="primary"
                circle
                class="composer-action-button"
                :class="{ 'is-pending': composerActionPending, 'is-running': sending }"
                :disabled="composerActionDisabled"
                :aria-label="composerActionLabel"
                :title="composerActionLabel"
                @click="handleComposerAction"
              >
                <template #icon>
                  <Transition name="composer-action-icon" mode="out-in">
                    <Icon :key="composerActionIcon" :icon="composerActionIcon" />
                  </Transition>
                </template>
              </n-button>
            </div>
          </div>
        </section>

        <template v-else-if="selectedId">
          <nav v-if="userMessages.length" class="conversation-preview" aria-label="用户消息导航">
            <button
              v-for="(item, index) in userMessages"
              :key="item.id"
              class="preview-tick"
              :aria-label="`跳转到第 ${index + 1} 条用户消息：${messagePreview(userItemText(item))}`"
              :title="messagePreview(userItemText(item))"
              :data-preview="messagePreview(userItemText(item))"
              @click="jumpToMessage(item.id)"
            />
          </nav>
          <section :key="selectedId" ref="messagePane" class="message-pane" @scroll="handleMessageScroll">
            <n-spin class="message-spin" :show="loadingCurrentDetail">
              <div class="message-column">
                <section v-for="view in renderedTurnViews" :key="view.renderKey" class="turn" :data-status="view.turn.status">
                  <article
                    v-for="item in view.userItems"
                    :key="item.id"
                    :id="`message-${item.id}`"
                    tabindex="-1"
                    class="message user"
                  >
                    <div class="message-stack">
                      <div class="message-content">{{ userItemText(item) }}</div>
                      <time v-if="formatMessageTime(view.turn.startedAt)" class="message-time" :datetime="view.turn.startedAt">
                        {{ formatMessageTime(view.turn.startedAt) }}
                      </time>
                    </div>
                  </article>

                  <div v-if="view.processMode === 'thinking'" class="turn-live-status">
                    <span class="status-pulse">正在思考</span>
                  </div>

                  <div v-else-if="view.processMode === 'processing'" class="turn-process">
                    <div class="turn-process-summary">
                      <span>{{ view.processSummary }}</span>
                    </div>
                    <div class="turn-process-content">
                      <ProcessItemCard
                        v-for="display in view.processDisplays"
                        :key="display.item.id"
                        :display="display"
                      />
                      <div v-if="view.showTailThinking" class="turn-inline-thinking">
                        <span class="status-pulse">正在思考</span>
                      </div>
                    </div>
                  </div>

                  <details
                    v-else-if="view.processMode === 'completed'"
                    class="turn-process turn-process-done"
                  >
                    <summary>
                      <span>{{ view.processSummary }}</span>
                    </summary>
                    <div v-if="view.processDisplays.length" class="turn-process-content">
                      <ProcessItemCard
                        v-for="display in view.processDisplays"
                        :key="display.item.id"
                        :display="display"
                      />
                    </div>
                  </details>

                  <template v-for="item in view.resultItems" :key="item.id">
                    <article v-if="item.type === 'agentMessage'" class="message assistant">
                      <div class="message-content">
                        <MarkdownBlock
                          v-if="item.text && parseAgentMessage(item.text, view.streaming).markdown"
                          :content="parseAgentMessage(item.text, view.streaming).markdown"
                          :streaming="view.streaming"
                        />
                        <button
                          v-for="workspace in parseAgentMessage(item.text || '', view.streaming).workspaces"
                          :key="workspace.url"
                          type="button"
                          class="workspace-card"
                          @click="openWorkspace(workspace)"
                        >
                          <Icon icon="solar:widget-5-linear" />
                          <span><strong>{{ workspace.title }}</strong><small>在右侧打开工作台</small></span>
                          <Icon icon="solar:arrow-right-linear" />
                        </button>
                      </div>
                    </article>

                    <article v-else-if="item.type === 'imageGeneration'" class="activity-card image-item">
                      <div class="activity-heading"><Icon :icon="activityIcon(item)" /><span>{{ activityTitle(item) }}</span></div>
                      <n-image
                        v-if="item.dataUrl"
                        class="generated-image"
                        :src="item.dataUrl"
                        :alt="item.alt || String(item.result || '生成的图片')"
                        object-fit="contain"
                        lazy
                      />
                    </article>
                  </template>

                  <article v-if="view.turn.error" class="turn-error">{{ view.turn.error }}</article>
                </section>
              </div>
            </n-spin>
          </section>

          <footer v-if="!isArchivedActive" class="composer-wrap">
            <div class="composer">
              <n-input
                v-model:value="draft"
                type="textarea"
                autosize
                :maxlength="12000"
                placeholder="给 AgentRazor 发送消息"
                @keydown="handleComposerKeydown"
              />
              <div class="composer-footer">
                <n-button
                  type="primary"
                  circle
                  class="composer-action-button"
                  :class="{ 'is-pending': composerActionPending, 'is-running': sending }"
                  :disabled="composerActionDisabled"
                  :aria-label="composerActionLabel"
                  :title="composerActionLabel"
                  @click="handleComposerAction"
                >
                  <template #icon>
                    <Transition name="composer-action-icon" mode="out-in">
                      <Icon :key="composerActionIcon" :icon="composerActionIcon" />
                    </Transition>
                  </template>
                </n-button>
              </div>
            </div>
          </footer>
        </template>

        <section v-else-if="!currentUser" class="empty-state">
          <div class="welcome-icon"><Icon icon="solar:lock-keyhole-minimalistic-bold-duotone" /></div>
          <h1>登录后开始</h1>
          <p>登录账号即可使用你的专属 Agent 对话。</p>
          <n-button type="primary" size="large" @click="loginVisible = true">登录</n-button>
        </section>

        <section v-else class="empty-state">
          <div class="welcome-icon"><img src="/agentrazor-icon.png" alt="" /></div>
          <h1>从一个想法开始</h1>
          <p>创建对话，让 AgentRazor 持续处理你的目标。</p>
          <n-button type="primary" size="large" @click="createConversation">开始新对话</n-button>
        </section>

        <aside v-if="activeWorkspace && workspaceVisible" class="workspace-panel" :class="{ 'is-expanded': workspaceExpanded }">
          <div class="workspace-resizer" aria-hidden="true" @pointerdown="startWorkspaceResize" />
          <header class="workspace-panel-header">
            <strong>{{ activeWorkspace.title }}</strong>
            <div class="workspace-panel-actions">
              <n-button
                quaternary
                circle
                class="workspace-action-button"
                aria-label="Refresh workspace"
                title="Refresh workspace"
                @click="reloadWorkspace"
              >
                <template #icon><Icon icon="solar:refresh-linear" /></template>
              </n-button>
              <n-button
                quaternary
                circle
                class="workspace-action-button workspace-maximize-button"
                :aria-label="workspaceExpanded ? 'Restore workspace size' : 'Expand workspace'"
                :title="workspaceExpanded ? 'Restore workspace size' : 'Expand workspace'"
                @click="toggleWorkspaceExpanded"
              >
                <template #icon>
                  <Icon :icon="workspaceExpanded ? 'solar:minimize-square-3-linear' : 'solar:maximize-square-3-linear'" />
                </template>
              </n-button>
              <n-button
                quaternary
                circle
                class="workspace-action-button workspace-right-panel-toggle"
                aria-label="Toggle pinned summary"
                title="Toggle pinned summary"
                @click="toggleRightPanel"
              >
                <template #icon><Icon icon="solar:sidebar-minimalistic-outline" /></template>
              </n-button>
            </div>
          </header>
          <iframe :key="`${selectedId}:${workspaceReloadVersion}`" :src="activeWorkspace.url" :title="activeWorkspace.title" />
        </aside>
      </main>
    </div>

    <n-modal v-model:show="renameVisible" preset="card" title="重命名对话" class="rename-modal">
      <n-input v-model:value="renameValue" maxlength="80" autofocus @keyup.enter="saveRename" />
      <template #footer>
        <div class="modal-actions">
          <n-button @click="renameVisible = false">取消</n-button>
          <n-button type="primary" :disabled="!renameValue.trim()" @click="saveRename">保存</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal
      v-model:show="groupEditorVisible"
      preset="card"
      :title="editingGroupId ? '重命名分组' : '新建分组'"
      class="rename-modal"
    >
      <n-input v-model:value="groupEditorName" maxlength="40" autofocus placeholder="输入分组名称" @keyup.enter="saveGroup" />
      <template #footer>
        <div class="modal-actions">
          <n-button @click="groupEditorVisible = false">取消</n-button>
          <n-button type="primary" :disabled="!groupEditorName.trim()" @click="saveGroup">保存</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal
      v-model:show="confirmVisible"
      preset="card"
      :bordered="false"
      :mask-closable="!confirmLoading"
      :close-on-esc="!confirmLoading"
      class="confirm-modal"
    >
      <div class="confirm-modal-body">
        <span class="confirm-modal-icon"><Icon icon="solar:trash-bin-trash-linear" /></span>
        <div>
          <h2>{{ confirmTitle }}</h2>
          <p>{{ confirmContent }}</p>
        </div>
      </div>
      <template #footer>
        <div class="modal-actions">
          <n-button :disabled="confirmLoading" @click="closeConfirm">取消</n-button>
          <n-button type="error" :loading="confirmLoading" @click="submitConfirm">{{ confirmPositiveText }}</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="loginVisible" preset="card" :bordered="false" :mask-closable="!loginLoading" class="login-modal">
      <n-button quaternary circle class="login-close" aria-label="关闭登录窗口" @click="loginVisible = false">
        <template #icon><Icon icon="solar:close-circle-linear" /></template>
      </n-button>
      <div class="login-hero">
        <img src="/agentrazor-icon.png" alt="" />
        <div>
          <span>AGENTRAZOR</span>
          <h2>欢迎回来</h2>
          <p>登录后继续你的 Agent 对话与任务。</p>
        </div>
      </div>
      <div class="login-form">
        <label>
          <span>用户名</span>
          <n-input v-model:value="loginUsername" size="large" placeholder="输入用户名" autofocus>
            <template #prefix><Icon icon="solar:user-rounded-linear" /></template>
          </n-input>
        </label>
        <label>
          <span>密码</span>
          <n-input
            v-model:value="loginPassword"
            type="password"
            size="large"
            show-password-on="click"
            placeholder="输入密码"
            @keyup.enter="submitLogin"
          >
            <template #prefix><Icon icon="solar:lock-keyhole-minimalistic-linear" /></template>
          </n-input>
        </label>
        <n-button
          type="primary"
          size="large"
          block
          :loading="loginLoading"
          :disabled="!loginUsername.trim() || !loginPassword"
          @click="submitLogin"
        >
          登录 AgentRazor
        </n-button>
      </div>
    </n-modal>

    <section v-if="settingsVisible" class="settings-shell" :class="{ 'settings-nav-collapsed': !settingsNavExpanded }">
      <aside class="settings-sidebar">
        <button class="settings-back" type="button" @click="settingsVisible = false">
          <Icon icon="solar:arrow-left-linear" />
          <span>返回应用</span>
        </button>
        <div class="settings-sidebar-title">设置</div>
        <nav aria-label="设置导航">
          <button :class="{ active: settingsSection === 'appearance' }" @click="settingsSection = 'appearance'">
            <Icon icon="solar:sun-2-linear" />
            <span>外观</span>
          </button>
          <button :class="{ active: settingsSection === 'archives' }" @click="openArchiveSettings()">
            <Icon icon="solar:archive-linear" />
            <span>已归档对话</span>
          </button>
        </nav>
      </aside>

      <div
        v-if="settingsNavExpanded"
        class="settings-backdrop"
        @click="settingsNavExpanded = false"
      />

      <main class="settings-content">
        <section v-if="settingsSection === 'appearance'" class="settings-content-inner appearance-page">
          <header class="settings-page-header">
            <button
              class="settings-menu-button"
              type="button"
              aria-label="打开设置菜单"
              @click="settingsNavExpanded = true"
            >
              <Icon icon="solar:sidebar-minimalistic-outline" />
            </button>
            <div>
              <h1>外观</h1>
              <p>选择 AgentRazor 的显示方式。</p>
            </div>
          </header>
          <div class="appearance-options" role="radiogroup" aria-label="外观">
            <button
              v-for="option in appearanceOptions"
              :key="option.key"
              type="button"
              role="radio"
              :aria-checked="appearance === option.key"
              :class="{ active: appearance === option.key }"
              @click="setAppearance(option.key)"
            >
              <Icon :icon="option.icon" />
              <span>{{ option.label }}</span>
            </button>
          </div>
        </section>

        <section v-else class="settings-content-inner archives-page">
          <header class="settings-page-header archive-page-header">
            <button
              class="settings-menu-button"
              type="button"
              aria-label="打开设置菜单"
              @click="settingsNavExpanded = true"
            >
              <Icon icon="solar:sidebar-minimalistic-outline" />
            </button>
            <div>
              <h1>已归档的对话</h1>
              <p>归档对话不能查看内容，恢复后才会重新出现在主页面。</p>
            </div>
            <n-button v-if="archivedConversations.length" tertiary type="error" @click="confirmDeleteAllArchived">
              全部删除
            </n-button>
          </header>
          <n-input v-model:value="archiveQuery" size="large" clearable placeholder="搜索已归档对话" class="archive-search">
            <template #prefix><Icon icon="solar:magnifer-linear" /></template>
          </n-input>
          <n-spin :show="loadingList">
            <div v-if="archivedConversationSections.length" class="archive-list archive-page-list">
              <template v-for="section in archivedConversationSections" :key="section.title">
                <div class="archive-section-head">
                  <h2 class="archive-section-title">{{ section.title }}</h2>
                  <n-button
                    v-if="section.groupId"
                    text
                    type="error"
                    size="small"
                    @click="confirmDeleteGroupArchived(section)"
                  >
                    全部删除
                  </n-button>
                </div>
                <div v-for="item in section.items" :key="item.id" class="archive-item">
                  <div class="archive-item-copy">
                    <strong>{{ item.title }}</strong>
                    <span>{{ formatConversationDate(item.updatedAt) }}</span>
                  </div>
                  <n-button quaternary circle type="error" aria-label="删除" @click="confirmDeleteArchived(item)">
                    <template #icon><Icon icon="solar:trash-bin-trash-linear" /></template>
                  </n-button>
                  <n-button secondary @click="restoreArchived(item)">取消归档</n-button>
                </div>
              </template>
            </div>
            <div v-if="!archivedConversationSections.length" class="archive-empty">
              {{ archiveQuery.trim() ? '没有匹配的归档对话' : '暂无已归档对话' }}
            </div>
          </n-spin>
        </section>
      </main>
    </section>
  </n-config-provider>
</template>
