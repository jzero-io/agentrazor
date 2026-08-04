<script setup lang="ts">
import { computed, h, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { Icon } from '@iconify/vue';
import {
  NButton,
  NConfigProvider,
  NDropdown,
  NEmpty,
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
import { authApi, clearToken, conversationApi, conversationGroupApi, getToken, setAuthErrorHandler, setToken } from './api';
import type { Conversation, ConversationDetail, Message, StreamEvent, UserInfo } from './api';

interface SidebarViewState {
  sidebarCollapsed?: boolean;
  pinnedExpanded?: boolean;
  groupsExpanded?: boolean;
  conversationsExpanded?: boolean;
  collapsedGroups?: Record<string, boolean>;
}

const SIDEBAR_VIEW_KEY = 'agentrazor_sidebar_view';
const CONVERSATION_LIST_DROP_TARGET = 'conversation-list';
function loadSidebarViewState(): SidebarViewState {
  try {
    return JSON.parse(localStorage.getItem(SIDEBAR_VIEW_KEY) || '{}') as SidebarViewState;
  } catch {
    return {};
  }
}
const savedSidebarView = loadSidebarViewState();

const { message: toast } = createDiscreteApi(['message']);
const conversations = ref<Conversation[]>([]);
const selectedId = ref('');
const detail = ref<ConversationDetail | null>(null);
const draft = ref('');
const loadingList = ref(false);
const loadingDetail = ref(false);
const sending = ref(false);
const streamingMessage = ref<Message | null>(null);
const sidebarCollapsed = ref(savedSidebarView.sidebarCollapsed ?? false);
const renameVisible = ref(false);
const renameValue = ref('');
const messagePane = ref<HTMLElement>();
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
const userMenuVisible = ref(false);
const pinnedExpanded = ref(savedSidebarView.pinnedExpanded ?? true);
const groupsExpanded = ref(savedSidebarView.groupsExpanded ?? true);
const conversationsExpanded = ref(savedSidebarView.conversationsExpanded ?? true);
const archiveQuery = ref('');
interface ConversationGroup {
  id: string;
  name: string;
  pinnedAt?: string;
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
const activeConversation = computed(() =>
  conversations.value.find(item => item.id === selectedId.value)
);
const isArchivedActive = computed(() => activeConversation.value?.status === 'archived');
const userMessages = computed(() =>
  (detail.value?.messages || []).filter(item => item.role === 'user')
);
const canSend = computed(() => Boolean(draft.value.trim() && selectedId.value && !sending.value));
const isNewChat = computed(() =>
  Boolean(selectedId.value)
  && !detail.value?.messages?.length
  && !sending.value
  && !streamingMessage.value
);
const userInitial = computed(() => currentUser.value?.username.trim().slice(0, 2).toUpperCase() || 'AR');
const isDarkAppearance = computed(() =>
  appearance.value === 'dark' || appearance.value === 'system' && systemDark.value
);
const activeTheme = computed(() => isDarkAppearance.value ? darkTheme : null);

const renderIcon = (icon: string) => () => h(Icon, { icon });
const menuOptions = computed(() => {
  const conv = activeConversation.value;
  if (!conv) return [];
  const archived = conv.status === 'archived';
  return [
    { label: '重命名', key: 'rename', icon: renderIcon('solar:pen-2-outline'), disabled: archived },
    { label: conv.pinnedAt ? '取消置顶' : '置顶', key: 'pin', icon: renderIcon('solar:pin-bold') },
    { label: archived ? '取消归档' : '归档', key: 'archive', icon: renderIcon('solar:archive-linear') },
    { label: '删除', key: 'delete', icon: renderIcon('solar:trash-bin-trash-linear') }
  ];
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
      pinnedAt: group.pinnedAt,
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

function startConversationDrag(item: Conversation, event: DragEvent) {
  draggedConversationId.value = item.id;
  conversationDropTarget.value = '';
  hideConversationPreview();
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', item.id);
  }
}

function setConversationDropTarget(target: string, event: DragEvent) {
  if (!draggedConversationId.value) return;
  event.preventDefault();
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
  conversationDropTarget.value = target;
}

async function dropConversation(targetGroupId: string, event: DragEvent) {
  event.preventDefault();
  const conversationId = draggedConversationId.value || event.dataTransfer?.getData('text/plain') || '';
  const item = conversations.value.find(conversation => conversation.id === conversationId);
  if (!item || (item.groupId || '') === targetGroupId) {
    endConversationDrag();
    return;
  }
  if (targetGroupId) {
    const group = conversationGroups.value.find(candidate => candidate.id === targetGroupId);
    if (group) group.collapsed = false;
  }
  await updateConversationGroup(item, targetGroupId);
  endConversationDrag();
}

function endConversationDrag() {
  draggedConversationId.value = '';
  conversationDropTarget.value = '';
}

async function loadConversations(selectFirst = false) {
  loadingList.value = true;
  try {
    conversations.value = await conversationApi.list();
    if (selectFirst && !selectedId.value) {
      const requestedId = new URL(window.location.href).searchParams.get('conversation') || '';
      const preferred = visibleConversations.value.find(item => item.id === requestedId)
        || visibleConversations.value[0];
      if (preferred) await selectConversation(preferred.id);
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
  if (key === 'pin') void toggleGroupPinned(group);
  if (key === 'delete') deleteGroup(group);
}

async function toggleGroupPinned(group: ConversationGroup) {
  try {
    await conversationGroupApi.update(group.id, { pinned: !group.pinnedAt });
    await loadConversationGroups();
  } catch (error) {
    showError(error);
  }
}

async function selectConversation(id: string) {
  if (!id || id === selectedId.value && detail.value) return;
  selectedId.value = id;
  syncConversationUrl(id);
  detail.value = null;
  streamingMessage.value = null;
  sending.value = false;
  closeStream?.();
  closeStream = undefined;
  streamEventCutoff = Date.now();
  await refreshDetail();
  if (selectedId.value === id) {
    closeStream = conversationApi.subscribe(id, handleStreamEvent, () => undefined);
  }
}

function syncConversationUrl(id: string) {
  const url = new URL(window.location.href);
  if (id) url.searchParams.set('conversation', id);
  else url.searchParams.delete('conversation');
  window.history.replaceState(window.history.state, '', `${url.pathname}${url.search}${url.hash}`);
}

async function refreshDetail() {
  if (!selectedId.value) return;
  loadingDetail.value = true;
  try {
    detail.value = await conversationApi.get(selectedId.value);
    await scrollToBottom();
  } catch (error) {
    showError(error);
  } finally {
    loadingDetail.value = false;
  }
}

async function sendMessage() {
  const content = draft.value.trim();
  if (!canSend.value || !content) return;
  draft.value = '';
  sending.value = true;
  streamingMessage.value = null;

  const optimistic: Message = {
    id: `local-${Date.now()}`,
    runId: '',
    role: 'user',
    content,
    status: 'queued',
    createdAt: new Date().toISOString()
  };
  detail.value?.messages.push(optimistic);
  await scrollToBottom();

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
    detail.value?.messages.splice(detail.value.messages.indexOf(optimistic), 1);
    sending.value = false;
    showError(error);
  }
}

async function handleStreamEvent(event: StreamEvent) {
  if (event.sessionId !== selectedId.value) return;
  const eventTime = Date.parse(event.createdAt);
  if (Number.isFinite(eventTime) && eventTime <= streamEventCutoff) return;

  if (event.type === 'run.started') {
    sending.value = true;
    streamingMessage.value = {
      id: '',
      runId: event.runId || '',
      role: 'assistant',
      content: '',
      status: 'streaming',
      createdAt: new Date().toISOString()
    };
    await scrollToBottom();
    return;
  }

  if (event.type === 'codex.item.agentMessage.delta') {
    const params = (event.data as { params?: { delta?: string; itemId?: string } } | undefined)?.params;
    const delta = params?.delta ?? '';
    if (!delta) return;
    if (!streamingMessage.value) {
      streamingMessage.value = {
        id: params?.itemId || '',
        runId: event.runId || '',
        role: 'assistant',
        content: '',
        status: 'streaming',
        createdAt: new Date().toISOString()
      };
    }
    streamingMessage.value.content += delta;
    await scrollToBottom();
    return;
  }

  if (event.type === 'run.completed' || event.type === 'run.failed') {
    await Promise.all([refreshDetail(), loadConversations()]);
    sending.value = false;
    streamingMessage.value = null;
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
      streamingMessage.value = null;
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
    streamingMessage.value = null;
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

async function scrollToBottom() {
  await nextTick();
  messagePane.value?.scrollTo({ top: messagePane.value.scrollHeight, behavior: 'smooth' });
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
  setAuthErrorHandler(() => {
    currentUser.value = null;
    conversations.value = [];
    selectedId.value = '';
    syncConversationUrl('');
    detail.value = null;
    streamingMessage.value = null;
    closeStream?.();
  });
  try {
    if (getToken()) {
      currentUser.value = await authApi.getUserInfo();
      await Promise.all([loadConversationGroups(), loadConversations(true)]);
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
    const { token } = await authApi.pwdLogin(username, password);
    setToken(token);
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
  userMenuVisible.value = false;
  clearToken();
  currentUser.value = null;
  conversationGroups.value = [];
  closeStream?.();
  selectedId.value = '';
  syncConversationUrl('');
  detail.value = null;
  conversations.value = [];
  streamingMessage.value = null;
  sending.value = false;
}

function openSettings() {
  userMenuVisible.value = false;
  settingsSection.value = 'appearance';
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
  window.addEventListener('blur', hideConversationPreview);
  colorSchemeMedia.addEventListener('change', handleSystemAppearanceChange);
  void bootstrap();
});
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown);
  document.removeEventListener('pointerover', handleDocumentPointerOver);
  window.removeEventListener('blur', hideConversationPreview);
  colorSchemeMedia.removeEventListener('change', handleSystemAppearanceChange);
  closeStream?.();
});
watch(isDarkAppearance, value => {
  document.documentElement.dataset.theme = value ? 'dark' : 'light';
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
</script>

<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN" :theme="activeTheme" :theme-overrides="themeOverrides">
    <div v-if="authChecking" class="app-boot-screen" aria-label="正在恢复登录状态" />
    <div v-else class="app-shell" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
      <aside class="sidebar">
        <div class="brand">
          <div class="brand-mark"><img src="/agentrazor-icon.png" alt="" /></div>
          <div v-if="!sidebarCollapsed" class="brand-copy">
            <strong>AgentRazor</strong>
            <span>Agent Workspace</span>
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
                    draggable="true"
                    @dragstart="startConversationDrag(item, $event)"
                    @dragend="endConversationDrag"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)">
                      <Icon icon="solar:pin-bold" class="pin-icon" />
                      <span>{{ item.title }}</span>
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
                @dragover="setConversationDropTarget(group.id, $event)"
                @drop.stop="dropConversation(group.id, $event)"
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
                    <n-tooltip trigger="hover" placement="top">
                      <template #trigger>
                        <n-dropdown
                          trigger="click"
                          placement="right-start"
                          :options="[
                            { label: group.pinnedAt ? '取消置顶' : '置顶分组', key: 'pin', icon: renderIcon(group.pinnedAt ? 'solar:pin-bold' : 'solar:pin-linear') },
                            { label: '重命名分组', key: 'rename', icon: renderIcon('solar:pen-2-linear') },
                            { type: 'divider', key: 'divider' },
                            { label: '删除分组', key: 'delete', icon: renderIcon('solar:trash-bin-trash-linear') }
                          ]"
                          @select="key => handleGroupAction(group, String(key))"
                        >
                          <button type="button" aria-label="更多分组操作"><Icon icon="solar:menu-dots-bold" /></button>
                        </n-dropdown>
                      </template>
                      更多操作
                    </n-tooltip>
                  </div>
                </div>
                <template v-if="!group.collapsed">
                  <div
                    v-for="item in conversationsInGroup(group.id)"
                    :key="item.id"
                    class="conversation-row grouped-conversation previewable-conversation-row"
                    :class="{ selected: item.id === selectedId, dragging: item.id === draggedConversationId }"
                    draggable="true"
                    @dragstart="startConversationDrag(item, $event)"
                    @dragend="endConversationDrag"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)">
                      <span>{{ item.title }}</span>
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
                @dragover="setConversationDropTarget(CONVERSATION_LIST_DROP_TARGET, $event)"
                @drop.stop="dropConversation('', $event)"
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
                    draggable="true"
                    @dragstart="startConversationDrag(item, $event)"
                    @dragend="endConversationDrag"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)">
                      <span>{{ item.title }}</span>
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
              <n-button type="primary" circle :disabled="!canSend" @click="sendMessage">
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
              :aria-label="`跳转到第 ${index + 1} 条用户消息：${messagePreview(item.content)}`"
              :title="messagePreview(item.content)"
              :data-preview="messagePreview(item.content)"
              @click="jumpToMessage(item.id)"
            />
          </nav>
          <section ref="messagePane" class="message-pane">
            <n-spin :show="loadingDetail">
              <div class="message-column">
                <article
                  v-for="item in detail?.messages || []"
                  :key="item.id"
                  :id="`message-${item.id}`"
                  tabindex="-1"
                  class="message"
                  :class="item.role"
                >
                  <div v-if="item.role === 'assistant'" class="avatar">
                    <img src="/agentrazor-icon.png" alt="AgentRazor" />
                  </div>
                  <div class="message-content">{{ item.content }}</div>
                </article>

                <article v-if="streamingMessage || sending" class="message assistant">
                  <div class="avatar"><img src="/agentrazor-icon.png" alt="AgentRazor" /></div>
                  <div class="message-content">
                    <template v-if="streamingMessage?.content">{{ streamingMessage.content }}<span class="stream-cursor" /></template>
                    <div v-else class="thinking"><i /><i /><i /><span>正在处理</span></div>
                  </div>
                </article>
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
                <n-button type="primary" circle :disabled="!canSend" @click="sendMessage">
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

    <section v-if="settingsVisible" class="settings-shell">
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
          <button :class="{ active: settingsSection === 'archives' }" @click="openArchiveSettings">
            <Icon icon="solar:archive-linear" />
            <span>已归档对话</span>
          </button>
        </nav>
      </aside>

      <main class="settings-content">
        <section v-if="settingsSection === 'appearance'" class="settings-content-inner appearance-page">
          <header class="settings-page-header">
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
            <div v-if="filteredArchivedConversations.length" class="archive-list archive-page-list">
              <div v-for="item in filteredArchivedConversations" :key="item.id" class="archive-item">
                <div class="archive-item-copy">
                  <strong>{{ item.title }}</strong>
                  <span>{{ formatConversationDate(item.updatedAt) }}</span>
                </div>
                <n-button quaternary circle type="error" aria-label="删除" @click="confirmDeleteArchived(item)">
                  <template #icon><Icon icon="solar:trash-bin-trash-linear" /></template>
                </n-button>
                <n-button secondary @click="restoreArchived(item)">取消归档</n-button>
              </div>
            </div>
            <div v-else class="archive-empty">
              {{ archiveQuery.trim() ? '没有匹配的归档对话' : '暂无已归档对话' }}
            </div>
          </n-spin>
        </section>
      </main>
    </section>
  </n-config-provider>
</template>
