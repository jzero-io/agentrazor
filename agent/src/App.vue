<script setup lang="ts">
import { computed, defineComponent, h, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch, type PropType } from 'vue';
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

interface SidebarViewState {
  sidebarCollapsed?: boolean;
  pinnedExpanded?: boolean;
  groupsExpanded?: boolean;
  conversationsExpanded?: boolean;
  collapsedGroups?: Record<string, boolean>;
}

const SIDEBAR_VIEW_KEY = 'agentrazor_sidebar_view';
const CONVERSATION_LIST_DROP_TARGET = 'conversation-list';

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
const detail = ref<ConversationDetail | null>(null);
const draft = ref('');
const loadingList = ref(false);
const loadingDetail = ref(false);
const sending = ref(false);
const streamingTurn = ref<Turn | null>(null);
const processingStatus = ref('');
// 是否已开始输出最终回答：一旦有结果，就不再回退到"正在思考"
const streamResultSeen = ref(false);
const elapsedDurationMs = ref(0);
const stopping = ref(false);
let turnStartedAtMs = 0;
let turnTimer: number | undefined;
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
let closeStream: (() => void) | undefined;
let streamEventCutoff = 0;

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

const visibleConversations = computed(() =>
  conversations.value.filter(item => item.status !== 'archived')
);
const archivedConversations = computed(() =>
  conversations.value.filter(item => item.status === 'archived')
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
  if (ungrouped?.length) sections.push({ title: '未分组', items: ungrouped });
  return sections;
});
const activeConversation = computed(() =>
  conversations.value.find(item => item.id === selectedId.value)
);
const isArchivedActive = computed(() => activeConversation.value?.status === 'archived');
const userMessages = computed(() =>
  (detail.value?.turns || []).flatMap(turn => turn.items.filter(item => item.type === 'userMessage'))
);
const renderedTurns = computed(() => [
  ...(detail.value?.turns || []),
  ...(streamingTurn.value ? [streamingTurn.value] : [])
]);
const canSend = computed(() => Boolean(draft.value.trim() && selectedId.value && !sending.value));
const isNewChat = computed(() =>
  Boolean(selectedId.value)
  && !detail.value?.turns?.length
  && !sending.value
  && !streamingTurn.value
);
const userInitial = computed(() => currentUser.value?.username.trim().slice(0, 2).toUpperCase() || 'AR');
const isDarkAppearance = computed(() =>
  appearance.value === 'dark' || appearance.value === 'system' && systemDark.value
);
const activeTheme = computed(() => isDarkAppearance.value ? darkTheme : null);

interface ProcessDisplayItem {
  item: ThreadItem;
  label: string;
  icon: string;
  detail: string;
  live: boolean;
  expandable: boolean;
  className: string;
}

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
      return new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime();
    });
}

function conversationGroupName(item: Conversation) {
  if (!item.groupId) return '';
  return conversationGroups.value.find(group => group.id === item.groupId)?.name || '';
}

function isConversationProcessing(item: Conversation) {
  return item.id === selectedId.value && (sending.value || Boolean(streamingTurn.value));
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
  try {
    const updated = await conversationApi.update(item.id, { groupId });
    replaceConversation(updated);
  } catch (error) {
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
  conversationDropTarget.value = zone ? String(zone.dataset.groupId ?? '') : '';
}

function finishDrag(item: Conversation) {
  const targetGroupId = conversationDropTarget.value;
  conversationDropTarget.value = '';
  draggedConversationId.value = '';
  if (dragGhostEl) {
    dragGhostEl.remove();
    dragGhostEl = null;
  }
  document.body.classList.remove('is-dragging');
  if (!targetGroupId && targetGroupId !== '') return;
  if ((item.groupId || '') === targetGroupId) return;
  if (targetGroupId) {
    const group = conversationGroups.value.find(candidate => candidate.id === targetGroupId);
    if (group) group.collapsed = false;
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

async function loadConversations(selectFirst = false) {
  loadingList.value = true;
  try {
    conversations.value = await conversationApi.list();
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
            const exists = conversations.value.some(item => item.id === preferred!.id);
            if (!exists) {
              const index = conversations.value.findIndex(
                item => (item.updatedAt || '') < (preferred!.updatedAt || '')
              );
              if (index < 0) conversations.value.push(preferred);
              else conversations.value.splice(index, 0, preferred);
            }
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

async function createConversation() {
  if (!currentUser.value) {
    loginVisible.value = true;
    return;
  }
  try {
    const created = await conversationApi.create();
    conversations.value.unshift(created);
    await selectConversation(created.id);
  } catch (error) {
    showError(error);
  }
}

function openMobileSidebar() {
  // 移动端抽屉：确保侧栏内容展开（桌面端折叠状态不影响抽屉显示）
  if (sidebarCollapsed.value) sidebarCollapsed.value = false;
  mobileSidebarOpen.value = true;
}

async function createConversationInGroup(group: ConversationGroup) {
  if (!currentUser.value) {
    loginVisible.value = true;
    return;
  }
  try {
    const created = await conversationApi.create();
    const grouped = await conversationApi.update(created.id, { groupId: group.id });
    conversations.value.unshift(grouped);
    group.collapsed = false;
    await selectConversation(grouped.id);
  } catch (error) {
    showError(error);
    await loadConversations();
  }
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
        await conversationGroupApi.archiveConversations(group.id);
        await Promise.all([loadConversationGroups(), loadConversations()]);
      } catch (error) {
        showError(error);
      }
    }
  );
}

async function selectConversation(id: string) {
  mobileSidebarOpen.value = false;
  if (!id || id === selectedId.value && detail.value) return;
  selectedId.value = id;
  syncConversationUrl(id);
  detail.value = null;
  resetActiveTurn();
  closeStream?.();
  closeStream = undefined;
  autoScrollEnabled.value = true;
  streamEventCutoff = Date.now();
  const snapshot = await refreshDetail({ forceScroll: true });
  if (selectedId.value === id) {
    closeStream = conversationApi.subscribe(id, snapshot?.eventCursor ?? 0, handleStreamEvent, () => undefined);
  }
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
  turn?: Turn;
  id?: string;
  status?: string;
  startedAt?: string;
  label?: string;
  resetResultSeen?: boolean;
  restartTimer?: boolean;
}

async function refreshDetail(options: RefreshDetailOptions = {}): Promise<ConversationDetail | null> {
  const id = selectedId.value;
  if (!id) return null;
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
    if (selectedId.value === id) {
      loadingDetail.value = false;
      void scrollToBottom({ force: options.forceScroll });
    }
  }
}

function isActiveTurn(turn: Turn) {
  const normalized = String(turn.status || '').replace(/[-_\s]/g, '').toLowerCase();
  return normalized === 'inprogress' || normalized === 'running' || normalized === 'pending';
}

function beginActiveTurn(options: BeginActiveTurnOptions = {}) {
  const current = options.turn
    ? { ...options.turn, items: [...(options.turn.items || [])] }
    : streamingTurn.value || { id: options.id || '', status: options.status || 'inProgress', items: [] };
  if (options.id) current.id = options.id;
  if (options.status) current.status = options.status;
  if (options.startedAt) current.startedAt = options.startedAt;
  streamingTurn.value = current;
  sending.value = true;
  stopping.value = false;
  if (options.resetResultSeen) streamResultSeen.value = false;
  else streamResultSeen.value = streamResultSeen.value || current.items.some(item => item.type === 'agentMessage' && item.phase === 'final_answer');
  if (options.restartTimer || !turnStartedAtMs) startTurnTimer(current.startedAt || options.startedAt);
  if (options.label !== undefined) {
    processingStatus.value = options.label;
  } else if (!processingStatus.value && !streamResultSeen.value) {
    processingStatus.value = processingLabel(turnActiveProcessItem(current) || undefined);
  }
}

function resetActiveTurn() {
  streamingTurn.value = null;
  sending.value = false;
  stopping.value = false;
  processingStatus.value = '';
  streamResultSeen.value = false;
  stopTurnTimer();
}

function restoreActiveTurn(snapshot: ConversationDetail, enabled: boolean) {
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
  if (activeIndex < 0) {
    detail.value = snapshot;
    return;
  }

  const activeTurn = turns[activeIndex];
  detail.value = {
    ...snapshot,
    turns: turns.filter((_, index) => index !== activeIndex)
  };
  beginActiveTurn({
    turn: activeTurn,
    startedAt: activeTurn.startedAt || snapshot.conversation.updatedAt || snapshot.conversation.createdAt
  });
}

async function sendMessage() {
  const content = draft.value.trim();
  if (!canSend.value || !content) return;
  draft.value = '';
  autoScrollEnabled.value = true;

  const optimistic: Turn = {
    id: `local-${Date.now()}`,
    status: 'inProgress',
    startedAt: new Date().toISOString(),
    items: [{
      id: `local-user-${Date.now()}`,
      type: 'userMessage',
      content: [{ type: 'text', text: content }]
    }]
  };
  beginActiveTurn({ turn: optimistic, label: '正在思考', resetResultSeen: true, restartTimer: true });
  await scrollToBottom({ force: true });

  try {
    if (activeConversation.value?.title === '新对话') {
      const title = content.replace(/\s+/g, ' ').slice(0, 28);
      const updated = await conversationApi.update(selectedId.value, { title });
      replaceConversation(updated);
      if (detail.value) detail.value.conversation = updated;
    }
    await conversationApi.send(selectedId.value, content);
  } catch (error) {
    draft.value = content;
    resetActiveTurn();
    showError(error);
  }
}

async function cancelTurn() {
  if (!selectedId.value || !sending.value || stopping.value) return;
  stopping.value = true;
  processingStatus.value = '正在停止';
  try {
    await conversationApi.cancelTurn(selectedId.value);
  } catch (error) {
    stopping.value = false;
    processingStatus.value = '正在处理';
    showError(error);
  }
}

function ensureStreamingItem(id: string, type: string) {
  beginActiveTurn();
  let item = streamingTurn.value!.items.find(value => value.id === id);
  if (!item) {
    item = { id, type };
    streamingTurn.value!.items.push(item);
  }
  return item;
}

function upsertStreamingItem(item: ThreadItem) {
  beginActiveTurn();
  let index = streamingTurn.value!.items.findIndex(value => value.id === item.id);
  if (index < 0 && item.type === 'userMessage') {
    index = streamingTurn.value!.items.findIndex(value =>
      value.type === 'userMessage' && value.id.startsWith('local-user-')
    );
  }
  if (index >= 0) streamingTurn.value!.items[index] = item;
  else streamingTurn.value!.items.push(item);
}

function processingLabel(item?: ThreadItem) {
  if (!item) return '正在处理';
  const toolName = `${item.server || ''} ${item.tool || ''}`.toLowerCase();
  const truncate = (value: string, max = 48) => value.length > max ? `${value.slice(0, max)}…` : value;
  switch (item.type) {
    case 'reasoning':
      return '正在思考';
    case 'commandExecution':
      if (isSkillReadItem(item)) {
        const skill = skillReadName(item);
        const display = skill ? humanizeSkillName(skill) : '';
        return display ? `正在读取 ${display} 技能` : '正在读取技能';
      }
      return item.pluginId
        ? `正在使用 ${item.pluginId}`
        : item.command ? `正在运行命令：${truncate(item.command)}` : '正在运行命令';
    case 'fileChange':
      {
        const paths = (item.changes || []).map(change => (change as Record<string, unknown>).path).filter(Boolean) as string[];
        return paths.length ? `正在修改文件：${truncate(paths.join('、'))}` : '正在修改文件';
      }
    case 'webSearch':
      return webSearchTitle(item, true);
    case 'imageView':
      return '正在查看图片';
    case 'imageGeneration':
      return '正在生成图片';
    case 'mcpToolCall':
    case 'dynamicToolCall':
      return toolName.includes('image')
        ? '正在生成图片'
        : item.tool ? `正在调用工具：${item.tool}` : '正在调用工具';
    case 'collabAgentToolCall':
    case 'subAgentActivity':
      return '正在协作处理';
    case 'contextCompaction':
      return '正在整理上下文';
    case 'sleep':
      return '正在等待';
    case 'agentMessage':
    case 'userMessage':
      return '';
    default:
      return '正在处理';
  }
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

function activityTitle(item: ThreadItem, live = false) {
  switch (item.type) {
    case 'commandExecution':
      if (isSkillReadItem(item)) {
        const skill = skillReadName(item);
        const display = skill ? humanizeSkillName(skill) : '';
        return `${live ? '正在读取' : '读取'}${display ? ` ${display} 技能` : '技能'}`;
      }
      return live ? '正在运行命令' : '运行命令';
    case 'fileChange': return fileChangeTitle(item, live);
    case 'webSearch': return webSearchTitle(item, live);
    case 'imageView': return live ? '正在查看图片' : '查看图片';
    case 'imageGeneration': return live ? '正在生成图片' : '生成图片';
    case 'mcpToolCall': return `${live ? '正在调用' : '调用'} ${item.server || 'MCP'} · ${item.tool || '工具'}`;
    case 'dynamicToolCall': return `${live ? '正在调用' : '调用'} ${item.tool || '工具'}`;
    case 'collabAgentToolCall': return live ? '正在协作处理' : '协作处理';
    case 'contextCompaction': return live ? '正在整理上下文' : '整理上下文';
    case 'plan': return '计划';
    default: return live ? '正在处理' : String(item.type);
  }
}

function activityIcon(item: ThreadItem) {
  switch (item.type) {
    case 'commandExecution': return isSkillReadItem(item) ? 'solar:document-text-linear' : 'solar:command-linear';
    case 'fileChange': return 'solar:document-add-linear';
    case 'webSearch': return webSearchActionType(item) === 'openPage' ? 'solar:link-round-angle-linear' : 'solar:magnifer-linear';
    case 'imageGeneration': return 'solar:gallery-add-linear';
    case 'reasoning': return 'solar:lightbulb-bolt-linear';
    default: return 'solar:widget-5-linear';
  }
}

function activityDetail(item: ThreadItem) {
  if (item.type === 'commandExecution') return [item.command, item.aggregatedOutput].filter(Boolean).join('\n\n');
  if (item.type === 'plan') return item.text || '';
  if (item.type === 'fileChange') return fileChangeDetail(item);
  if (item.type === 'webSearch') return item.result != null ? String(item.result) : '';
  if (item.type === 'mcpToolCall' || item.type === 'dynamicToolCall') {
    return JSON.stringify({ arguments: item.arguments, result: item.result }, null, 2);
  }
  return '';
}

function webSearchActionType(item: ThreadItem) {
  return (item.action as { type?: string } | undefined)?.type || '';
}

function webSearchText(item: ThreadItem) {
  const action = item.action as Record<string, unknown> | undefined;
  const values = [
    item.query,
    action?.title,
    action?.url,
    action?.query
  ].filter(value => typeof value === 'string' && value.trim()) as string[];
  return values.length ? cleanSearchQuery(values[0]) : '';
}

function webSearchTitle(item: ThreadItem, live = false) {
  const openPage = webSearchActionType(item) === 'openPage';
  const prefix = live
    ? openPage ? '正在打开网页' : '正在搜索网页'
    : openPage ? '打开网页' : '搜索网页';
  const text = webSearchText(item);
  return text ? `${prefix}：${text}` : prefix;
}

function fileChangeTitle(item: ThreadItem, live = false) {
  const changes = Array.isArray(item.changes) ? item.changes as Array<Record<string, unknown>> : [];
  const prefix = live ? '正在修改文件' : '修改文件';
  return changes.length ? `${prefix}：${changes.length} 个` : prefix;
}

function fileChangeDetail(item: ThreadItem) {
  const changes = Array.isArray(item.changes) ? item.changes as Array<Record<string, unknown>> : [];
  return changes.map(change => String(change.path || '')).filter(Boolean).join('\n');
}

function isIntermediateAgentMessage(item: ThreadItem) {
  return item.type === 'agentMessage' && item.phase === 'commentary';
}

function turnProcessItems(turn: Turn) {
  return turn.items.filter(item =>
    item.type !== 'userMessage'
    && item.type !== 'imageGeneration'
    && item.type !== 'reasoning'
    && (item.type !== 'agentMessage' || isIntermediateAgentMessage(item))
  );
}

function turnReasoningItems(turn: Turn) {
  return turn.items.filter(item => item.type === 'reasoning');
}

// 真正执行了工具/动作的条目（排除纯文字说明），用于判断任务是否"复杂"
function turnWorkedItems(turn: Turn) {
  return turnProcessItems(turn).filter(item => item.type !== 'agentMessage');
}

// 判断命令是否是在读取 skill：使用接口返回的结构化 commandActions
// （type=read 且 path 指向 skills 目录），而不是解析命令字符串
function isSkillReadItem(item: ThreadItem): boolean {
  if (item.type !== 'commandExecution' || !Array.isArray(item.commandActions)) return false;
  return item.commandActions.some(action =>
    action?.type === 'read'
    && typeof action.path === 'string'
    && action.path.includes('skills')
  );
}

// 从 commandActions 的 path 提取技能名（skills/ 后的第一个目录段）
function skillReadName(item: ThreadItem): string {
  if (!Array.isArray(item.commandActions)) return '';
  const action = item.commandActions.find(a =>
    a?.type === 'read' && typeof a.path === 'string' && a.path.includes('skills')
  );
  const match = action?.path?.match(/\/skills\/([^/]+)/);
  return match ? match[1] : '';
}

// 读取技能时具体读取的文件名（commandActions path 的 basename）
function skillReadFileName(item: ThreadItem): string {
  if (!Array.isArray(item.commandActions)) return '';
  const action = item.commandActions.find(a =>
    a?.type === 'read' && typeof a.path === 'string' && a.path.includes('skills')
  );
  const path = action?.path ?? '';
  return path.split('/').filter(Boolean).pop() ?? '';
}

// 展示的过程条目：同一技能的连续读取会合并技能标题，后续文件只展示文件名。
function turnDisplayItems(turn: Turn): ThreadItem[] {
  const result: ThreadItem[] = [];
  let lastSkill = '';
  for (const item of turnProcessItems(turn)) {
    if (isSkillReadItem(item)) {
      const skill = skillReadName(item);
      if (skill && skill === lastSkill) {
        result.push({ ...item, skillFileName: skillReadFileName(item) });
        continue;
      }
      lastSkill = skill;
    } else {
      lastSkill = '';
    }
    result.push(item);
  }
  return result;
}

function turnActiveProcessItem(turn: Turn): ThreadItem | null {
  const items = turnDisplayItems(turn).filter(item => item.type !== 'agentMessage');
  return items.length ? items[items.length - 1] : null;
}

function processClassName(item: ThreadItem) {
  if (item.type === 'webSearch') return 'search-entry';
  if (item.type === 'commandExecution') return isSkillReadItem(item) ? 'skill-read-entry' : 'command-entry';
  if (item.type === 'fileChange') return 'file-entry';
  return 'tool-entry';
}

function processDisplayItems(turn: Turn): ProcessDisplayItem[] {
  const active = turn === streamingTurn.value ? turnActiveProcessItem(turn) : null;
  return turnDisplayItems(turn)
    .filter(item => !(item.type === 'reasoning'))
    .map(item => {
      const live = Boolean(active && item.id === active.id);
      const skillFile = typeof item.skillFileName === 'string' && item.skillFileName.trim() ? item.skillFileName.trim() : '';
      const label = skillFile ? `读取 ${skillFile}` : activityTitle(item, live);
      const detail = skillFile ? '' : activityDetail(item);
      return {
        item,
        label,
        icon: activityIcon(item),
        detail,
        live,
        expandable: Boolean(detail),
        className: processClassName(item)
      };
    });
}

function turnResultItems(turn: Turn) {
  return turn.items.filter(item =>
    item.type === 'imageGeneration'
    || item.type === 'agentMessage' && !isIntermediateAgentMessage(item)
  );
}

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

      if (item.type === 'agentMessage' && item.text) {
        return h('div', { class: 'process-commentary markdown-body', innerHTML: renderMarkdown(item.text) });
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

function formatTurnDuration(durationMs = 0) {
  const totalSeconds = Math.max(0, Math.round(durationMs / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const parts: string[] = [];
  if (hours) parts.push(`${hours}小时`);
  if (minutes) parts.push(`${minutes}分`);
  if (seconds || parts.length === 0) parts.push(`${seconds}秒`);
  return parts.join('');
}

// 技能名美化：jzero-skills → Jzero Skills，与 desktop 展示一致
function humanizeSkillName(name: string): string {
  return name
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

// 搜索查询清洗：去掉内部追踪参数（#ws_call_id=...），避免标签出现长尾巴
function cleanSearchQuery(query: string): string {
  return query.replace(/#ws_call_id=.*$/, '').trim();
}

function startTurnTimer(startedAt?: string) {
  const parsed = startedAt ? Date.parse(startedAt) : Number.NaN;
  turnStartedAtMs = Number.isFinite(parsed) ? parsed : Date.now();
  // 正在处理时从 1s 开始计时，避免刚启动时显示 0s
  elapsedDurationMs.value = Math.max(1000, Date.now() - turnStartedAtMs);
  if (turnTimer !== undefined) window.clearInterval(turnTimer);
  turnTimer = window.setInterval(() => {
    elapsedDurationMs.value = Math.max(1000, Date.now() - turnStartedAtMs);
  }, 1000);
}

function stopTurnTimer() {
  if (turnTimer !== undefined) window.clearInterval(turnTimer);
  turnTimer = undefined;
  const duration = turnStartedAtMs ? Math.max(0, Date.now() - turnStartedAtMs) : undefined;
  turnStartedAtMs = 0;
  return duration;
}

async function handleStreamEvent(event: StreamEvent) {
  if (event.sessionId !== selectedId.value) return;
  const eventTime = Date.parse(event.createdAt);
  if (Number.isFinite(eventTime) && eventTime <= streamEventCutoff) return;

  if (event.type === 'run.started') {
    const data = event.data as { startedAt?: string } | undefined;
    beginActiveTurn({
      id: event.runId || undefined,
      status: 'inProgress',
      startedAt: data?.startedAt || event.createdAt,
      label: '正在思考',
      resetResultSeen: true
    });
    await scrollToBottom();
    return;
  }

  if (event.type === 'codex.turn.started') {
    const turn = (event.data as { params?: { turn?: Record<string, unknown> } } | undefined)?.params?.turn;
    if (turn) {
      const startedAt = Number(turn.startedAt);
      const startedAtIso = Number.isFinite(startedAt) && startedAt > 0 ? new Date(startedAt * 1000).toISOString() : undefined;
      beginActiveTurn({
        id: String(turn.id || streamingTurn.value?.id || ''),
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
      beginActiveTurn({
        id: String(turn.id || streamingTurn.value?.id || ''),
        status: String(turn.status || 'completed')
      });
      const activeTurn = streamingTurn.value!;
      const durationMs = Number(turn.durationMs);
      if (Number.isFinite(durationMs) && durationMs >= 0) activeTurn.durationMs = durationMs;
      const completedAt = Number(turn.completedAt);
      if (Number.isFinite(completedAt) && completedAt > 0) {
        activeTurn.completedAt = new Date(completedAt * 1000).toISOString();
      }
      if (Array.isArray(turn.items)) {
        // turn.completed 快照只含 userMessage/agentMessage，合并而非替换，
        // 保留流式阶段收集的 reasoning 等条目
        const merged = new Map(activeTurn.items.map(item => [item.id, item]));
        for (const item of turn.items as ThreadItem[]) merged.set(item.id, item);
        activeTurn.items = Array.from(merged.values());
      }
    }
    return;
  }

  if (event.type.includes('task_started')) {
    const data = event.data as { params?: Record<string, unknown> } | undefined;
    const payload = ((data?.params?.msg || data?.params?.payload || data?.params) ?? {}) as Record<string, unknown>;
    const startedAt = Number(payload.started_at);
    if (Number.isFinite(startedAt) && startedAt > 0) {
      beginActiveTurn({ startedAt: new Date(startedAt * 1000).toISOString() });
    }
    return;
  }

  if (event.type.includes('task_complete')) {
    const data = event.data as { params?: Record<string, unknown> } | undefined;
    const payload = ((data?.params?.msg || data?.params?.payload || data?.params) ?? {}) as Record<string, unknown>;
    const durationMs = Number(payload.duration_ms);
    if (Number.isFinite(durationMs) && durationMs >= 0) elapsedDurationMs.value = durationMs;
  }

  if (event.type === 'codex.item.reasoning.textDelta') {
    const params = (event.data as { params?: { delta?: string; itemId?: string; contentIndex?: number } } | undefined)?.params;
    const delta = params?.delta ?? '';
    if (!delta) return;
    const itemId = params?.itemId || `stream-reasoning-${event.runId || 'active'}`;
    const item = ensureStreamingItem(itemId, 'reasoning');
    const content = (Array.isArray(item.content) ? item.content : []) as unknown as string[];
    const index = Number(params?.contentIndex) || 0;
    while (content.length <= index) content.push('');
    content[index] = `${content[index] || ''}${delta}`;
    item.content = content;
    await scrollToBottom();
    return;
  }

  if (event.type === 'codex.item.agentMessage.delta') {
    const params = (event.data as { params?: { delta?: string; itemId?: string } } | undefined)?.params;
    const delta = params?.delta ?? '';
    if (!delta) return;
    const itemId = params?.itemId || `stream-agent-${event.runId || 'active'}`;
    const item = ensureStreamingItem(itemId, 'agentMessage');
    item.text = `${item.text || ''}${delta}`;
    if (item.phase === 'final_answer') streamResultSeen.value = true;
    await scrollToBottom();
    return;
  }

  if (event.type === 'codex.item.started') {
    const params = (event.data as { params?: { item?: ThreadItem } } | undefined)?.params;
    const streamedItem = params?.item as ThreadItem | undefined;
    if (streamedItem) {
      upsertStreamingItem(streamedItem);
      if (streamedItem.type === 'agentMessage' && streamedItem.phase === 'final_answer') {
        streamResultSeen.value = true;
      }
    }
    if (!stopping.value) {
      const label = processingLabel(params?.item);
      // agentMessage 等空标签不覆盖当前状态，避免执行过程中状态闪空
      // 已经有结果后，后续 reasoning 不再把状态拉回"正在思考"
      if (label && !(streamResultSeen.value && params?.item?.type === 'reasoning')) processingStatus.value = label;
    }
    await scrollToBottom();
    return;
  }

  if (event.type === 'codex.item.completed') {
    const params = (event.data as { params?: { item?: ThreadItem } } | undefined)?.params;
    if (params?.item) {
      const completedItem = params.item as ThreadItem;
      upsertStreamingItem(completedItem);
      if (completedItem.type === 'agentMessage' && completedItem.phase === 'final_answer') {
        streamResultSeen.value = true;
      }
    }
    await scrollToBottom();
    return;
  }

  if (event.type === 'run.completed' || event.type === 'run.failed') {
    stopTurnTimer();
    const completedTurn = streamingTurn.value;
    await Promise.all([refreshDetail({ restoreActiveTurn: false }), loadConversations()]);
    if (completedTurn && detail.value) {
      const persisted = detail.value.turns.find(turn => turn.id === completedTurn.id);
      if (persisted) {
        // 服务端持久化的 turn 只含 userMessage/agentMessage，工具/命令等过程
        // 条目在流式阶段按真实时间顺序收集。以流式顺序为骨架交错合并，
        // 避免命令全部堆到过程说明前面；同 id 或同文本的说明/最终回答
        // 优先采用服务端版本（带权威 phase）。
        const persistedById = new Map(persisted.items.map(item => [item.id, item]));
        const pickPersisted = (streamItem: ThreadItem): ThreadItem | undefined => {
          if (persistedById.has(streamItem.id)) return persistedById.get(streamItem.id);
          if (streamItem.type === 'agentMessage' && streamItem.text) {
            return persisted.items.find(existing =>
              existing.type === 'agentMessage' && existing.text === streamItem.text
            );
          }
          return undefined;
        };
        const merged: ThreadItem[] = [];
        const usedPersisted = new Set<ThreadItem>();
        const userMessage = persisted.items.find(item => item.type === 'userMessage');
        if (userMessage) merged.push(userMessage);
        for (const streamItem of completedTurn.items) {
          if (streamItem.type === 'userMessage' || streamItem.type === 'reasoning') continue;
          const existing = pickPersisted(streamItem);
          if (existing) usedPersisted.add(existing);
          merged.push(existing ?? streamItem);
        }
        for (const item of persisted.items) {
          if (item.type === 'userMessage' || usedPersisted.has(item)) continue;
          merged.push(item);
        }
        persisted.items = merged;
      } else {
        detail.value.turns.push(completedTurn);
      }
    }
    resetActiveTurn();
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
    const updated = await conversationApi.update(activeConversation.value.id, { title });
    replaceConversation(updated);
    if (detail.value) detail.value.conversation = updated;
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
    const updated = await conversationApi.update(item.id, {
      pinned: !item.pinnedAt
    });
    replaceConversation(updated);
    await loadConversations();
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
    if (wasSelected) {
      closeStream?.();
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
    await conversationApi.remove(activeConversation.value.id);
    closeStream?.();
    selectedId.value = '';
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
  if (index >= 0) conversations.value[index] = updated;
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
    resetActiveTurn();
    closeStream?.();
    closeStream = undefined;
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
  closeStream?.();
  closeStream = undefined;
  selectedId.value = '';
  syncConversationUrl('');
  detail.value = null;
  loadingDetail.value = false;
  conversations.value = [];
  resetActiveTurn();
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

function confirmDeleteAllArchived() {
  if (!archivedConversations.value.length) return;
  openConfirm(
    '删除全部归档对话',
    `确定永久删除全部 ${archivedConversations.value.length} 个归档对话吗？此操作不可撤销。`,
    '全部删除',
    async () => {
      try {
        await Promise.all(archivedConversations.value.map(item => conversationApi.remove(item.id)));
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

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown);
  document.addEventListener('pointerover', handleDocumentPointerOver);
  window.addEventListener('pointermove', onDragPointerMove);
  window.addEventListener('pointerup', onDragPointerUp);
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
  window.removeEventListener('pointerup', onDragPointerUp);
  window.removeEventListener('touchmove', onWindowTouchMove);
  window.removeEventListener('touchend', onWindowTouchEnd);
  window.removeEventListener('touchcancel', onWindowTouchEnd);
  window.removeEventListener('blur', hideConversationPreview);
  colorSchemeMedia.removeEventListener('change', handleSystemAppearanceChange);
  closeStream?.();
  stopTurnTimer();
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
    <div v-if="authChecking" class="app-boot-screen" aria-label="正在恢复登录状态">
      <div class="app-boot-card">
        <img class="app-boot-logo" src="/agentrazor-icon.png" alt="" />
        <div class="app-boot-copy">
          <div class="app-boot-title">AgentRazor</div>
          <div class="app-boot-text">正在恢复会话</div>
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
                data-group-id=""
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

      <main class="main-panel">
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

        <section v-if="selectedId && isNewChat" class="chat-empty">
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
                v-if="sending"
                type="primary"
                circle
                :loading="stopping"
                aria-label="停止当前任务"
                title="停止当前任务"
                @click="cancelTurn"
              >
                <template #icon><Icon icon="solar:stop-bold" /></template>
              </n-button>
              <n-button v-else type="primary" circle :disabled="!canSend" @click="sendMessage">
                <template #icon><Icon icon="solar:arrow-up-linear" /></template>
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
          <section ref="messagePane" class="message-pane" @scroll="handleMessageScroll">
            <n-spin :show="loadingDetail">
              <div class="message-column">
                <section v-for="turn in renderedTurns" :key="turn.id" class="turn" :data-status="turn.status">
                  <article
                    v-for="item in turn.items.filter(value => value.type === 'userMessage')"
                    :key="item.id"
                    :id="`message-${item.id}`"
                    tabindex="-1"
                    class="message user"
                  >
                    <div class="message-content">{{ userItemText(item) }}</div>
                  </article>

                  <!-- 阶段一：处理中，结果卡片按到达顺序逐条追加平铺，思考时只显示"正在思考" -->
                  <div v-if="turn === streamingTurn" class="turn-process">
                    <div v-if="turnWorkedItems(turn).length" class="turn-process-summary">
                      <span>正在处理 {{ formatTurnDuration(elapsedDurationMs) }}</span>
                    </div>
                    <div class="turn-process-content">
                      <template
                        v-for="display in processDisplayItems(turn)"
                        :key="display.item.id"
                      >
                        <ProcessItemCard :display="display" />
                      </template>
                    </div>
                  </div>
                  <!-- 阶段二：过程说明 + 工具/过程卡片都收进"已处理 Xs"折叠区 -->
                  <template v-else>
                    <details
                      v-if="turnProcessItems(turn).length && turn.durationMs !== undefined"
                      class="turn-process turn-process-done"
                    >
                      <summary>
                        <span>已处理 {{ formatTurnDuration(turn.durationMs) }}</span>
                      </summary>
                      <div class="turn-process-content">
                        <ProcessItemCard
                          v-for="display in processDisplayItems(turn)"
                          :key="display.item.id"
                          :display="display"
                        />
                      </div>
                    </details>
                  </template>
                  <div
                    v-if="turn === streamingTurn && processingStatus && !streamResultSeen && (!turnActiveProcessItem(turn) || turnActiveProcessItem(turn)!.type === 'reasoning')"
                    class="turn-live-status"
                  >
                    <span :class="{ 'status-pulse': processingStatus === '正在思考' }">{{ processingStatus }}</span>
                  </div>

                  <template v-for="item in turnResultItems(turn)" :key="item.id">
                    <article v-if="item.type === 'agentMessage'" class="message assistant">
                      <div class="message-content">
                        <div
                          v-if="item.text"
                          class="markdown-body"
                          :class="{ 'streaming-markdown': turn === streamingTurn && turn.status === 'inProgress' }"
                          v-html="renderMarkdown(item.text)"
                        />
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

                  <article v-if="turn.error" class="turn-error">{{ turn.error }}</article>
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
                  v-if="sending"
                  type="primary"
                  circle
                  :loading="stopping"
                  aria-label="停止当前任务"
                  title="停止当前任务"
                  @click="cancelTurn"
                >
                  <template #icon><Icon icon="solar:stop-bold" /></template>
                </n-button>
                <n-button v-else type="primary" circle :disabled="!canSend" @click="sendMessage">
                  <template #icon><Icon icon="solar:arrow-up-linear" /></template>
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
