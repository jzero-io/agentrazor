<script setup lang="ts">
import { computed, h, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch, type ComponentPublicInstance } from 'vue';
import { RouterView, useRoute } from 'vue-router';
import { Icon } from '@iconify/vue';
import 'highlight.js/styles/github.css';
import githubDarkTheme from 'highlight.js/styles/github-dark.css?raw';
import {
  NButton,
  NConfigProvider,
  NDropdown,
  NInput,
  NModal,
  createDiscreteApi,
  zhCN,
  dateZhCN
} from 'naive-ui';
import { conversationApi, conversationGroupApi } from '../../service/api';
import type { Conversation, ConversationDetail, ConversationMetadata, StreamEvent, ThreadItem, Turn } from '../../service/api';
import { activityIcon, activityTitle } from '../../utils/processDisplay';
import { useAppearance } from '../../hooks/system/useAppearance';
import { useConfirmDialog } from '../../hooks/system/useConfirmDialog';
import { useAuthSession } from '../../hooks/system/useAuthSession';
import { useSettingsView } from '../../hooks/system/useSettingsView';
import { useWorkspacePanel } from '../../hooks/business/useWorkspacePanel';
import { loadSidebarViewState, useSidebarState } from '../../hooks/layout/useSidebarState';
import { useConversationStreams } from '../../hooks/business/useConversationStreams';
import { useConversationStreamEvents } from '../../hooks/business/useConversationStreamEvents';
import { useConversationDrag } from '../../hooks/business/useConversationDrag';
import { useConversationGroups, type ConversationGroup } from '../../hooks/business/useConversationGroups';
import { useConversationList } from '../../hooks/business/useConversationList';
import { useConversationTitleRefresh } from '../../hooks/business/useConversationTitleRefresh';
import { useMessageActions } from '../../hooks/business/useMessageActions';
import { useConversationPreview } from '../../hooks/business/useConversationPreview';
import { useMessagePaneScroll } from '../../hooks/business/useMessagePaneScroll';
import { useConversationTurns, userItemText } from '../../hooks/business/useConversationTurns';
import { useConversationComposer } from '../../hooks/business/useConversationComposer';
import { useWorkspaceActions } from '../../hooks/business/useWorkspaceActions';
import {
  conversationIdFromPath,
  displayConversationTitle,
  formatConversationDate,
  formatMessageTime,
  mergeConversationSnapshot
} from '../../utils/conversation';
import { installScopedCss } from '../../utils/scopedCss';
import ConversationSidebar from '../modules/conversation-sidebar/index.vue';
import LoginModal from '../modules/login-modal/index.vue';
import RightPanel from '../modules/right-panel/index.vue';


installScopedCss(githubDarkTheme, ':root[data-theme="dark"]');

const CONVERSATION_LIST_DROP_TARGET = 'conversation-list';
const DRAFT_CONVERSATION_ID = '__draft_conversation__';

const savedSidebarView = loadSidebarViewState();
const route = useRoute();
const requestedConversationId = computed(() => conversationIdFromPath(route.path));
const isSettingsRoute = computed(() => route.path.startsWith('/settings'));

const selectedConversationId = ref('');
const mainPanel = ref<HTMLElement | null>(null);
const pinnedSummaryOpen = ref(false);
const workspacePanel = useWorkspacePanel({
  selectedConversationId,
  draftConversationId: DRAFT_CONVERSATION_ID,
  fetchFile: path => conversationApi.fetchWorkspaceFile(path),
  fetchBlob: path => conversationApi.fetchWorkspaceBlob(path),
  fetchEntries: conversationId => conversationApi.workspaceFiles(conversationId),
  containerRef: mainPanel
});
const workspaceExpanded = workspacePanel.expanded;
const workspaceReloadVersion = workspacePanel.reloadVersion;
const workspacePanelStyle = workspacePanel.panelStyle;
const startWorkspaceResize = workspacePanel.startResize;
const workspaceVisible = workspacePanel.visible;
const activeWorkspace = workspacePanel.activeWorkspace;
const activeWorkspaceTabs = workspacePanel.activeWorkspaceTabs;
const activeWorkspaceTabId = workspacePanel.activeWorkspaceTabId;
const activeWorkspaceUrl = workspacePanel.activeWorkspaceUrl;
const activeRightPanelKind = workspacePanel.activeKind;
const hasRightPanelWorkspace = workspacePanel.hasWorkspace;
const hasRightPanelFiles = workspacePanel.hasFiles;
const activeFilePath = workspacePanel.activeFilePath;
const activeFileTabId = workspacePanel.activeFileTabId;
const activeFileTabs = workspacePanel.activeFileTabs;
const activeFilePreview = workspacePanel.activeFilePreview;
const filePreviewLoading = workspacePanel.fileLoading;
const filePreviewError = workspacePanel.fileError;
const sidePanelTitle = workspacePanel.title;
const workspaceFileTree = workspacePanel.fileTree;
const workspaceFileTreeExpandedPaths = workspacePanel.fileTreeExpandedPaths;
const workspaceFileTreeLoading = workspacePanel.fileTreeLoading;
const workspaceFileTreeLoaded = workspacePanel.fileTreeLoaded;
const workspaceFileTreeError = workspacePanel.fileTreeError;
const toggleWorkspaceFileTreeDirectory = workspacePanel.toggleFileTreeDirectory;
const activeFilePreviewPath = workspacePanel.filePath;
const activeFilePreviewBreadcrumbs = workspacePanel.fileBreadcrumbs;
const activeFilePreviewBadge = workspacePanel.fileBadge;
const activeFilePreviewLines = workspacePanel.fileLines;
const selectWorkspaceFile = workspacePanel.selectFile;
const reorderWorkspaceFile = workspacePanel.reorderFile;
const detail = ref<ConversationDetail | null>(null);
const detailsByConversation = reactive(new Map<string, ConversationDetail>());
const draftsByConversation = reactive(new Map<string, string>());
const loadingList = ref(false);
const loadingDetail = ref(false);
const loadingDetailId = ref('');
const sendingRequest = ref(false);
const creatingConversation = ref(false);
const locallyStoppedTurnIds = new Set<string>();
const locallyStoppedConversationIds = new Set<string>();
const renameVisible = ref(false);
const renameConversationId = ref('');
const renameValue = ref('');
const renameInput = ref<HTMLInputElement | null>(null);
const renameSaving = ref(false);
const headingRenameVisible = ref(false);
const headingRenameValue = ref('');
const headingRenameInput = ref<HTMLInputElement | null>(null);
const headingRenameSaving = ref(false);

const confirmDialog = useConfirmDialog();
const confirmVisible = confirmDialog.visible;
const confirmTitle = confirmDialog.title;
const confirmContent = confirmDialog.content;
const confirmPositiveText = confirmDialog.positiveText;
const confirmLoading = confirmDialog.loading;
const openConfirm = confirmDialog.open;
const closeConfirm = confirmDialog.close;
const submitConfirm = confirmDialog.submit;
const userMenuVisible = ref(false);
const settingsView = useSettingsView({
  onOpen: () => void loadConversations()
});
const {
  visible: settingsVisible,
  section: settingsSection,
  navExpanded: settingsNavExpanded,
  archiveQuery,
  syncConversationUrl,
  restoreFromPath: restoreSettingsFromPath,
  openAppearance: openSettingsView,
  openArchives: openArchiveSettingsView
} = settingsView;
const conversationGroupsState = useConversationGroups({
  initialCollapsedGroups: savedSidebarView.collapsedGroups,
  showError
});
const {
  conversationGroups,
  groupEditorVisible,
  groupEditorName,
  editingGroupId,
  loadConversationGroups,
  openCreateGroup,
  openRenameGroup,
  saveGroup,
  toggleGroup,
  clearGroups
} = conversationGroupsState;
const sidebarState = useSidebarState({
  savedSidebarView,
  conversationGroups
});
const {
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
} = sidebarState;
const userMenuWrap = ref<HTMLElement>();
const loginModalOffsetX = computed(() => {
  if (sidebarCollapsed.value || mobileSidebarOpen.value) return 0;
  if (typeof window !== 'undefined' && window.matchMedia('(max-width: 720px)').matches) return 0;
  return sidebarWidth.value / 2;
});
const { appearance, appearanceOptions, isDarkAppearance, activeTheme, setAppearance } = useAppearance();
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
  },
  Tooltip: {
    boxShadow: 'none'
  }
};

const conversationListState = useConversationList({
  selectedConversationId,
  conversationGroups,
  archiveQuery,
  draftConversationId: DRAFT_CONVERSATION_ID
});
const {
  conversations,
  processingConversationIds,
  visibleConversations,
  archivedConversations,
  pinnedConversations,
  conversationList,
  archivedConversationSections,
  activeConversation,
  isArchivedActive,
  isDraftConversation,
  setConversationProcessing,
  clearConversationProcessing,
  touchConversationUpdatedAt,
  isConversationRunning,
  isConversationProcessing,
  replaceConversation,
  upsertConversationListItem,
  applyConversationList: applyConversationListState,
  clearConversations,
  conversationDetailForDraft
} = conversationListState;

const activeDetail = computed(() => {
  if (!selectedConversationId.value) return null;
  if (isDraftConversation()) return detail.value?.conversation.id === DRAFT_CONVERSATION_ID ? detail.value : conversationDetailForDraft(draftConversationGroupId.value);
  return detailsByConversation.get(selectedConversationId.value)
    || (detail.value?.conversation.id === selectedConversationId.value ? detail.value : null);
});
const conversationTurns = useConversationTurns({
  selectedConversationId,
  conversations,
  activeDetail,
  detailsByConversation,
  detail,
  locallyStoppedTurnIds,
  locallyStoppedConversationIds,
  setConversationProcessing,
  touchConversationUpdatedAt,
  isConversationProcessing,
  upsertConversationListItem
});
const {
  sending,
  stopping,
  activeTurnsByConversation,
  activeTurnResultSeenByConversation,
  currentStreamingTurn,
  currentTurnResultSeen,
  renderedTurns,
  renderedTurnViews,
  beginActiveTurn,
  confirmSentTurn,
  createOptimisticTurn,
  showOptimisticTurn,
  moveOptimisticTurn,
  discardOptimisticTurn,
  activeTurnError,
  syncDisplayedActiveTurnState,
  clearDisplayedActiveTurn,
  resetActiveTurn,
  restoreActiveTurn,
  setConversationDetail,
  clearConversationDetail,
  cachedActiveTurn,
  mergeTurnItems,
  publishActiveTurn,
  findStreamingItem,
  ensureStreamingItem,
  markProcessActive,
  resetProcessTimer,
  isVisibleProcessStreamItem,
  upsertStreamingItem,
  finishActiveTurn,
  finalizeStoppedTurn,
  mergeTurnForDisplay,
  stopTurnTimer,
  stopAllTurnTimers,
  clearAllActiveTurns
} = conversationTurns;
const draft = computed({
  get() {
    return selectedConversationId.value ? draftsByConversation.get(selectedConversationId.value) || '' : '';
  },
  set(value: string) {
    if (!selectedConversationId.value) return;
    setConversationDraft(selectedConversationId.value, value);
  }
});

function setConversationDraft(conversationId: string, value: string) {
  if (!conversationId) return;
  if (value) draftsByConversation.set(conversationId, value);
  else draftsByConversation.delete(conversationId);
}
const conversationPreviewState = useConversationPreview({ renderedTurns });
const {
  conversationPreview,
  conversationPreviewTop,
  messagePreviews,
  selectedPreviewMessageId,
  hoveredPreviewMessageId,
  previewTickClasses,
  showConversationPreview,
  hideConversationPreview,
  clearMessagePreviewSelection
} = conversationPreviewState;
const messagePaneScroll = useMessagePaneScroll({ selectedPreviewMessageId });
const {
  setMessagePane,
  handleMessageScroll,
  scrollToBottom,
  jumpToMessage,
  enableAutoScroll
} = messagePaneScroll;

const workspaceActions = useWorkspaceActions({
  renderedTurns,
  workspacePanel,
  pinnedSummaryOpen,
  workspaceVisible,
  activeWorkspace,
  activeFilePreview
});
const {
  rightPanelOpen,
  rightPanelContentReady,
  pinnedSummaryWorkspaces,
  parseAgentMessage,
  normalizeWorkspaceFilePath,
  openWorkspace,
  openWorkspaceFile,
  closeFilePreview,
  displayWorkspaceProcessPath,
  toggleRightPanel,
  toggleWorkspaceExpanded,
  reloadWorkspace,
  openPinnedWorkspace
} = workspaceActions;
const loadingCurrentDetail = computed(() =>
  Boolean(selectedConversationId.value)
  && !isDraftConversation()
  && loadingDetail.value
  && loadingDetailId.value === selectedConversationId.value
);
const conversationOpening = computed(() =>
  Boolean(selectedConversationId.value)
  && !activeDetail.value
  && !currentStreamingTurn.value
);
const currentConversationRunning = computed(() =>
  Boolean(selectedConversationId.value && !isDraftConversation() && (cachedActiveTurn(selectedConversationId.value) || isConversationRunning(selectedConversationId.value)))
);
const isNewChat = computed(() =>
  Boolean(selectedConversationId.value)
  && !conversationOpening.value
  && Boolean(activeDetail.value)
  && !activeDetail.value!.turns?.length
  && !currentConversationRunning.value
  && !currentStreamingTurn.value
);
// toast/message 跟随主题：深色模式下弹层不再出现白底
const toastProviderProps = reactive({
  theme: activeTheme.value,
  themeOverrides
});
const { message: toast } = createDiscreteApi(['message'], {
  configProviderProps: toastProviderProps
});
const authSession = useAuthSession({
  onAuthError: resetApplicationState,
  showError,
  showSuccess: message => toast.success(message)
});
const {
  currentUser,
  authChecking,
  hasAuthToken,
  loginVisible,
  loginMode,
  loginUsername,
  loginPassword,
  loginEmail,
  loginVerificationCode,
  loginVerificationUuid,
  verificationSending,
  verificationCountdown,
  loginLoading,
  userInitial,
  installAuthErrorHandler,
  restoreSession,
  submitLogin: submitLoginSession,
  sendLoginVerificationCode,
  openLogin,
  clearSession,
  finishAuthChecking
} = authSession;

const messageActions = useMessageActions({
  parseAgentMarkdown: content => parseAgentMessage(content, false).markdown,
  showError: message => toast.error(message)
});
const {
  copiedMessageId,
  messageCopyText,
  copyMessage,
  assistantMessageTime,
  clearCopiedMessageTimer
} = messageActions;

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
      const updateDiff = new Date(right.updatedAt || right.createdAt).getTime() - new Date(left.updatedAt || left.createdAt).getTime();
      if (updateDiff !== 0) return updateDiff;
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

function syncConversationMetadata(metadata: ConversationMetadata) {
  const item = conversations.value.find(conversation => conversation.id === metadata.id);
  if (item) replaceConversation({ ...item, ...metadata });
  const currentDetail = detailsByConversation.get(metadata.id);
  if (!currentDetail) return;
  setConversationDetail({
    ...currentDetail,
    conversation: {
      ...currentDetail.conversation,
      title: metadata.title,
      updatedAt: metadata.updatedAt
    }
  });
}

const { scheduleConversationTitleRefresh, stopAllConversationTitleRefresh } = useConversationTitleRefresh({
  syncConversationMetadata
});

const conversationStreamEvents = useConversationStreamEvents({
  selectedConversationId,
  conversations,
  detailsByConversation,
  locallyStoppedTurnIds,
  locallyStoppedConversationIds,
  activeTurnResultSeenByConversation,
  resetProcessTimer,
  setConversationProcessing,
  beginActiveTurn,
  cachedActiveTurn,
  mergeTurnItems,
  publishActiveTurn,
  findStreamingItem,
  ensureStreamingItem,
  markProcessActive,
  isVisibleProcessStreamItem,
  upsertStreamingItem,
  finishActiveTurn,
  mergeTurnForDisplay,
  setConversationDetail,
  scrollToBottom,
  closeIdleConversationStreams
});

const conversationStreams = useConversationStreams({
  onEvent: conversationStreamEvents.handleStreamEvent,
  onBufferedEvents: conversationStreamEvents.handleBufferedStreamEvents,
  onStreamError: async id => {
    await reconcileConversationStream(id);
  },
  onReconnect: reconcileConversationStream
});

function closeConversationStream(id: string) {
  conversationStreams.closeStream(id);
}

function closeAllConversationStreams() {
  conversationStreams.closeAll();
}

function ensureConversationStream(id: string): Promise<void> {
  return conversationStreams.ensureStream(id);
}

async function reconcileConversationStream(id: string) {
  if (!id || !currentUser.value) return;
  await loadConversations(false, false);
  const conversation = conversations.value.find(item => item.id === id);
  if (conversation?.running) {
    if (selectedConversationId.value === id) {
      return (await refreshDetailForSynchronization(id, { restoreActiveTurn: true })).streamPosition;
    }
    return;
  }

  setConversationProcessing(id, false);
  stopTurnTimer(id);
  activeTurnsByConversation.delete(id);
  activeTurnResultSeenByConversation.delete(id);
  if (selectedConversationId.value === id) {
    clearDisplayedActiveTurn();
    return (await refreshDetailForSynchronization(id, { restoreActiveTurn: false })).streamPosition;
  }
}

function closeIdleConversationStreams(activeId = selectedConversationId.value) {
  conversationStreams.closeInactive(activeId);
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
const conversationDrag = useConversationDrag({
  conversationListDropTarget: CONVERSATION_LIST_DROP_TARGET,
  conversationGroups,
  conversationsExpanded,
  displayConversationTitle,
  hideConversationPreview,
  selectConversation: item => selectConversation(item.id),
  updateConversationGroup
});
const {
  draggedConversationId,
  conversationDropTarget,
  onConversationPointerDown,
  onRowTouchStart
} = conversationDrag;

async function loadConversations(selectInitial = false, preserveLocalProcessing = true) {
  loadingList.value = true;
  try {
    applyConversationListState(await conversationApi.list(), preserveLocalProcessing, id => Boolean(cachedActiveTurn(id)));
    closeIdleConversationStreams();
    if (selectInitial && !selectedConversationId.value) await selectInitialConversation();
  } catch (error) {
    showError(error);
  } finally {
    loadingList.value = false;
  }
}

async function selectInitialConversation() {
  const requestedId = requestedConversationId.value;
  if (!requestedId) {
    showDraftConversation();
    return;
  }

  const target = await resolveRequestedConversation(requestedId);
  const fallback = target || visibleConversations.value[0];
  if (fallback) await selectConversation(fallback.id);
  else syncConversationUrl('');
}

async function resolveRequestedConversation(id: string) {
  const listed = conversations.value.find(item => item.id === id);
  if (listed) return listed;

  try {
    const snapshot = await conversationApi.get(id);
    if (!snapshot?.conversation) return undefined;
    upsertConversationListItem(snapshot.conversation);
    return snapshot.conversation;
  } catch {
    return undefined;
  }
}

function showDraftConversation(groupId = '') {
  draftConversationGroupId.value = groupId;
  selectedConversationId.value = DRAFT_CONVERSATION_ID;
  detail.value = conversationDetailForDraft(groupId);
  clearDisplayedActiveTurn();
  syncConversationUrl('');
  enableAutoScroll();
}

function beginDraftConversation(groupId = '') {
  if (sendingRequest.value) return false;
  if (!currentUser.value) {
    loginVisible.value = true;
    return false;
  }
  closeMobileSidebar();
  showDraftConversation(groupId);
  return true;
}

function createConversation() {
  if (!beginDraftConversation()) return;
  conversationsExpanded.value = true;
}

function createConversationInGroup(group: ConversationGroup) {
  if (!beginDraftConversation(group.id)) return;
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
  closeMobileSidebar();
  if (!id) return;
  if (id === selectedConversationId.value && activeDetail.value) {
    if (loadingDetailId.value !== id) loadingDetail.value = false;
    return;
  }

  const token = ++conversationSelectionToken;

  loadingDetailId.value = id;
  loadingDetail.value = true;
  selectedConversationId.value = id;
  syncConversationUrl(id);
  detail.value = detailsByConversation.get(id) || null;
  enableAutoScroll();
  syncDisplayedActiveTurnState();

  const archived = conversations.value.find(item => item.id === id)?.status === 'archived';
  if (archived) {
    await refreshDetail({ forceScroll: true });
  } else {
    try {
      await conversationStreams.synchronizeStream(id, async () => {
        if (conversationSelectionToken !== token || selectedConversationId.value !== id) return;
        return (await refreshDetailForSynchronization(id, { forceScroll: true })).streamPosition;
      });
    } catch (error) {
      if (conversationSelectionToken !== token || selectedConversationId.value !== id) return;
      showError(error);
    }
  }
  if (conversationSelectionToken !== token || selectedConversationId.value !== id) return;
  closeIdleConversationStreams(id);
}

interface RefreshDetailOptions {
  forceScroll?: boolean;
  restoreActiveTurn?: boolean;
  reportError?: boolean;
  throwOnError?: boolean;
}

async function refreshDetail(options: RefreshDetailOptions = {}): Promise<ConversationDetail | null> {
  const id = selectedConversationId.value;
  if (!id) return null;
  loadingDetailId.value = id;
  loadingDetail.value = true;
  try {
    const snapshot = await conversationApi.get(id);
    if (selectedConversationId.value !== id) return null;
    const listed = conversations.value.find(item => item.id === id);
    if (listed?.status === 'archived') {
      snapshot.conversation = {
        ...snapshot.conversation,
        status: 'archived',
        archivedAt: listed.archivedAt
      };
    }
    restoreActiveTurn(snapshot, options.restoreActiveTurn !== false);
    return snapshot;
  } catch (error) {
    if (selectedConversationId.value === id && options.reportError !== false) showError(error);
    if (options.throwOnError) throw error;
    return null;
  } finally {
    if (loadingDetailId.value === id) {
      loadingDetail.value = false;
      loadingDetailId.value = '';
    }
    if (selectedConversationId.value === id) {
      void scrollToBottom({ force: options.forceScroll });
    }
  }
}

const snapshotRetryDelays = [0, 400, 1000];

async function refreshDetailForSynchronization(
  id: string,
  options: Omit<RefreshDetailOptions, 'reportError' | 'throwOnError'> = {}
): Promise<ConversationDetail> {
  let lastError: unknown = new Error('读取会话快照失败');
  for (const delay of snapshotRetryDelays) {
    if (selectedConversationId.value !== id) throw lastError;
    if (delay > 0) await new Promise(resolve => window.setTimeout(resolve, delay));
    if (selectedConversationId.value !== id) throw lastError;
    try {
      const snapshot = await refreshDetail({
        ...options,
        reportError: false,
        throwOnError: true
      });
      if (snapshot?.streamPosition) return snapshot;
      lastError = new Error('会话快照缺少同步位置');
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError;
}

async function synchronizeSelectedConversationDetail(): Promise<ConversationDetail | null> {
  const id = selectedConversationId.value;
  if (!id) return null;
  if (conversations.value.find(item => item.id === id)?.status === 'archived') return refreshDetail();
  let snapshot: ConversationDetail | null = null;
  try {
    await conversationStreams.synchronizeStream(id, async () => {
      if (selectedConversationId.value !== id) return;
      snapshot = await refreshDetailForSynchronization(id);
      return snapshot.streamPosition;
    });
    return snapshot;
  } catch (error) {
    if (selectedConversationId.value === id) showError(error);
    return null;
  }
}

const conversationComposer = useConversationComposer({
  selectedConversationId,
  draftConversationId: DRAFT_CONVERSATION_ID,
  draftConversationGroupId,
  detailsByConversation,
  detail,
  activeDetail,
  sendingRequest,
  creatingConversation,
  sending,
  stopping,
  draftValue: draft,
  setConversationDraft,
  isDraftConversation,
  isConversationRunning,
  setConversationProcessing,
  locallyStoppedConversationIds,
  createOptimisticTurn,
  showOptimisticTurn,
  moveOptimisticTurn,
  discardOptimisticTurn,
  activeTurnError,
  cachedActiveTurn,
  confirmSentTurn,
  resetActiveTurn,
  finalizeStoppedTurn,
  setConversationDetail,
  upsertConversationListItem,
  syncConversationMetadata,
  revealConversationSection,
  scheduleConversationTitleRefresh,
  ensureConversationStream,
  refreshDetail: synchronizeSelectedConversationDetail,
  enableAutoScroll,
  scrollToBottom,
  syncConversationUrl,
  showError
});
const {
  canSend,
  composerActionPending,
  composerActionDisabled,
  composerActionLabel,
  composerActionIcon,
  sendMessage,
  handleComposerAction,
  handleComposerKeydown
} = conversationComposer;


function openRename(item: Conversation | null | undefined = activeConversation.value) {
  if (!item || isDraftConversation(item.id)) return;
  renameConversationId.value = item.id;
  renameValue.value = item.title || '';
  renameVisible.value = true;
  void nextTick(() => {
    renameInput.value?.focus();
    renameInput.value?.select();
  });
}

function cancelRename() {
  renameVisible.value = false;
  renameConversationId.value = '';
  renameValue.value = '';
}

async function saveRename() {
  const title = renameValue.value.trim();
  if (renameSaving.value) return;
  const conversation = conversations.value.find(item => item.id === renameConversationId.value);
  if (!conversation) {
    cancelRename();
    return;
  }
  if (!title) return;
  if (title === conversation.title) {
    cancelRename();
    return;
  }
  renameSaving.value = true;
  try {
    const conversationId = conversation.id;
    const updated = await conversationApi.update(conversationId, { title });
    replaceConversation(updated);
    if (detail.value?.conversation.id === conversationId) {
      detail.value.conversation = { ...detail.value.conversation, ...updated };
    }
    cancelRename();
  } catch (error) {
    showError(error);
  } finally {
    renameSaving.value = false;
  }
}

function openHeadingRename() {
  if (!activeConversation.value || isDraftConversation(activeConversation.value.id)) return;
  headingRenameValue.value = activeConversation.value.title || '';
  headingRenameVisible.value = true;
  void nextTick(() => {
    headingRenameInput.value?.focus();
    headingRenameInput.value?.select();
  });
}

function cancelHeadingRename() {
  headingRenameVisible.value = false;
  headingRenameValue.value = '';
}

async function saveHeadingRename() {
  const conversation = activeConversation.value;
  const title = headingRenameValue.value.trim();
  if (headingRenameSaving.value || !conversation) return;
  if (!title || title === conversation.title) {
    cancelHeadingRename();
    return;
  }

  headingRenameSaving.value = true;
  try {
    const updated = await conversationApi.update(conversation.id, { title });
    replaceConversation(updated);
    if (detail.value?.conversation.id === conversation.id) {
      detail.value.conversation = { ...detail.value.conversation, ...updated };
    }
    cancelHeadingRename();
  } catch (error) {
    showError(error);
  } finally {
    headingRenameSaving.value = false;
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
  const wasSelected = item.id === selectedConversationId.value;
  try {
    await conversationApi.update(item.id, { archived: true });
    clearConversationDetail(item.id);
    setConversationProcessing(item.id, false);
    if (wasSelected) {
      closeConversationStream(item.id);
      selectedConversationId.value = '';
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
    `确定删除「${displayConversationTitle(activeConversation.value)}」吗？此操作不可撤销。`,
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
    workspacePanel.removeConversation(removedId);
    setConversationProcessing(removedId, false);
    closeConversationStream(removedId);
    selectedConversationId.value = '';
    draftConversationGroupId.value = '';
    syncConversationUrl('');
    detail.value = null;
    resetActiveTurn();
    await loadConversations(true);
  } catch (error) {
    showError(error);
  }
}


function showError(error: unknown) {
  toast.error(error instanceof Error ? error.message : '操作失败');
}

function resetApplicationState() {
  clearConversations();
  settingsVisible.value = false;
  selectedConversationId.value = '';
  syncConversationUrl('');
  detail.value = null;
  loadingDetail.value = false;
  loadingDetailId.value = '';
  detailsByConversation.clear();
  sendingRequest.value = false;
  creatingConversation.value = false;
  resetActiveTurn();
  clearAllActiveTurns();
  stopAllConversationTitleRefresh();
  closeAllConversationStreams();
}

function finishBootScreen() {
  void nextTick(() => {
    requestAnimationFrame(() => document.body.classList.add('app-ready'));
  });
}

async function bootstrap() {
  const restoredSettings = restoreSettingsFromPath();
  installAuthErrorHandler();
  const user = await restoreSession();
  if (!user) {
    finishBootScreen();
    return;
  }
  await Promise.all([loadConversationGroups(), loadConversations(!restoredSettings)]);
  finishBootScreen();
}

async function submitLogin() {
  const user = await submitLoginSession();
  if (!user) return;
  await loadConversationGroups();
  await loadConversations(!settingsVisible.value);
}

function logout() {
  closeMobileSidebar();
  userMenuVisible.value = false;
  clearSession();
  clearGroups();
  draftConversationGroupId.value = '';
  resetApplicationState();
}

function openSettings() {
  closeMobileSidebar();
  userMenuVisible.value = false;
  openSettingsView(location.pathname);
}

function openArchiveSettings() {
  openArchiveSettingsView();
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
    `确定删除「${displayConversationTitle(item)}」吗？此操作不可撤销。`,
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

function setUserMenuWrap(el: Element | ComponentPublicInstance | null) {
  userMenuWrap.value = el instanceof HTMLElement ? el : undefined;
}

function handleDocumentPointerDown(event: PointerEvent) {
  if (!userMenuVisible.value) return;
  const target = event.target;
  if (target instanceof Node && !userMenuWrap.value?.contains(target)) {
    userMenuVisible.value = false;
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown);
  void bootstrap();
  // 防止登录恢复流程异常卡死导致一直白屏：超时后强制结束启动态
  window.setTimeout(() => {
    if (authChecking.value) {
      finishAuthChecking();
      finishBootScreen();
    }
  }, 10000);
});
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown);
  closeAllConversationStreams();
  stopAllTurnTimers();
  stopAllConversationTitleRefresh();
  clearCopiedMessageTimer();
});
watch(
  () => route.path,
  path => {
    if (settingsVisible.value || authChecking.value || !currentUser.value) return;
    const conversationId = conversationIdFromPath(path);
    if (conversationId) {
      if (conversationId !== selectedConversationId.value) void selectConversation(conversationId);
      return;
    }
    if (selectedConversationId.value !== DRAFT_CONVERSATION_ID) showDraftConversation();
  }
);
let initialConversationSelection = true;
watch(selectedConversationId, id => {
  clearMessagePreviewSelection();
  pinnedSummaryOpen.value = false;
  const restorePanel = initialConversationSelection
    && performance.getEntriesByType('navigation').some(entry => (entry as PerformanceNavigationTiming).type === 'reload');
  initialConversationSelection = false;
  if (id) {
    if (restorePanel) workspacePanel.restore(id);
    else workspacePanel.collapse();
  } else {
    workspaceExpanded.value = false;
  }
});
watch(isDarkAppearance, value => {
  document.documentElement.dataset.theme = value ? 'dark' : 'light';
  toastProviderProps.theme = activeTheme.value;
}, { immediate: true });
</script>

<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN" :theme="activeTheme" :theme-overrides="themeOverrides">
    <div class="app-shell" :class="{ 'sidebar-collapsed': Boolean(currentUser) && sidebarCollapsed, 'sidebar-hover-open': Boolean(currentUser) && sidebarHoverOpen }" :style="appShellStyle">
      <ConversationSidebar
        v-model:user-menu-visible="userMenuVisible"
        v-model:pinned-expanded="pinnedExpanded"
        v-model:groups-expanded="groupsExpanded"
        v-model:conversations-expanded="conversationsExpanded"
        :mobile-sidebar-open="mobileSidebarOpen"
        :sidebar-expanded="currentUser ? sidebarExpanded : true"
        :sidebar-collapsed="currentUser ? sidebarCollapsed : false"
        :current-user="currentUser"
        :user-initial="userInitial"
        :loading-list="Boolean(currentUser) && loadingList"
        :pinned-conversations="currentUser ? pinnedConversations : []"
        :conversation-groups="currentUser ? conversationGroups : []"
        :conversation-list="currentUser ? conversationList : []"
        :selected-conversation-id="selectedConversationId"
        :dragged-conversation-id="draggedConversationId"
        :conversation-drop-target="conversationDropTarget"
        :conversation-list-drop-target="CONVERSATION_LIST_DROP_TARGET"
        :display-conversation-title="displayConversationTitle"
        :is-conversation-processing="isConversationProcessing"
        :conversations-in-group="currentUser ? conversationsInGroup : (() => [])"
        :set-user-menu-wrap="setUserMenuWrap"
        :close-sidebar-hover="closeSidebarHover"
        :toggle-sidebar-pinned="toggleSidebarPinned"
        :create-conversation="createConversation"
        :create-conversation-in-group="createConversationInGroup"
        :open-create-group="openCreateGroup"
        :toggle-group="toggleGroup"
        :handle-group-action="handleGroupAction"
        :hide-conversation-preview="hideConversationPreview"
        :show-conversation-preview="showConversationPreview"
        :select-conversation="selectConversation"
        :rename-conversation="openRename"
        :on-conversation-pointer-down="onConversationPointerDown"
        :on-row-touch-start="onRowTouchStart"
        :toggle-conversation-pinned="toggleConversationPinned"
        :archive-conversation="archiveConversation"
        :open-settings="openSettings"
        :logout="logout"
        :open-login="openLogin"
        :start-sidebar-resize="startSidebarResize"
      />

      <div
        v-if="mobileSidebarOpen"
        class="sidebar-backdrop"
        @click="closeMobileSidebar"
      />

      <div
        v-if="currentUser && conversationPreview && sidebarExpanded"
        class="conversation-hover-card"
        :style="{ top: `${conversationPreviewTop}px` }"
      >
        <strong>{{ displayConversationTitle(conversationPreview) }}</strong>
        <div class="pinned-preview-meta">
          <Icon icon="solar:clock-circle-linear" />
          <span>{{ formatConversationDate(conversationPreview.updatedAt) }}</span>
        </div>
        <div v-if="conversationGroupName(conversationPreview)" class="pinned-preview-meta">
          <Icon icon="lucide:folder" />
          <span>{{ conversationGroupName(conversationPreview) }}</span>
        </div>
      </div>

      <main
        ref="mainPanel"
        class="main-panel"
        :class="{
          'workspace-open':
            !isSettingsRoute && Boolean(selectedConversationId) && selectedConversationId !== DRAFT_CONVERSATION_ID && workspaceVisible && rightPanelContentReady,
          'workspace-expanded': !isSettingsRoute && selectedConversationId !== DRAFT_CONVERSATION_ID && workspaceExpanded
        }"
        :style="workspacePanelStyle"
      >
        <header class="topbar">
          <n-button quaternary circle class="mobile-menu-button" aria-label="打开左侧边栏" title="打开左侧边栏" @click="openMobileSidebar">
            <template #icon><Icon icon="lucide:panel-left" /></template>
          </n-button>
          <div v-if="currentUser && sidebarCollapsed" class="topbar-left-actions">
            <n-button quaternary class="topbar-sidebar-toggle" aria-label="打开左侧边栏" title="打开左侧边栏" @mouseenter="openSidebarHover" @click="expandSidebar">
              <template #icon><Icon icon="lucide:panel-left" /></template>
            </n-button>
            <n-button quaternary circle class="topbar-new-chat" aria-label="新对话" title="新对话" @click="createConversation">
              <template #icon><Icon icon="solar:pen-new-square-outline" /></template>
            </n-button>
          </div>
          <div class="topbar-inner">
            <div v-if="activeConversation && !isDraftConversation(activeConversation.id)" class="conversation-heading">
              <form v-if="headingRenameVisible" class="conversation-heading-editor" @submit.prevent="saveHeadingRename">
                <span class="conversation-heading-editor-sizer" aria-hidden="true">{{ headingRenameValue || ' ' }}</span>
                <input
                  ref="headingRenameInput"
                  v-model="headingRenameValue"
                  size="1"
                  maxlength="80"
                  aria-label="修改对话标题"
                  @blur="saveHeadingRename"
                  @keydown.esc.prevent="cancelHeadingRename"
                />
              </form>
              <button
                v-else
                type="button"
                class="conversation-heading-title"
                :title="`修改标题：${displayConversationTitle(activeConversation)}`"
                :aria-label="`修改标题：${displayConversationTitle(activeConversation)}`"
                @click="openHeadingRename"
              >
                <span>{{ displayConversationTitle(activeConversation) }}</span>
              </button>
              <n-dropdown
                v-if="activeConversation"
                trigger="click"
                placement="bottom-start"
                :options="menuOptions"
                @select="key => key === 'rename' ? openHeadingRename() : key === 'delete' ? confirmDelete() : key === 'pin' ? togglePinned() : toggleArchived()"
              >
                <n-button quaternary circle class="conversation-heading-more"><template #icon><Icon icon="lucide:ellipsis" /></template></n-button>
              </n-dropdown>
            </div>
          </div>
          <n-button
            v-if="!isSettingsRoute && currentUser && selectedConversationId && selectedConversationId !== DRAFT_CONVERSATION_ID && !workspaceVisible"
            quaternary
            class="right-panel-toggle topbar-right-panel-toggle"
            :aria-label="rightPanelOpen ? '收起右侧面板' : '打开右侧面板'"
            :title="rightPanelOpen ? '收起右侧面板' : '打开右侧面板'"
            @click="toggleRightPanel"
          >
            <template #icon><Icon icon="lucide:panel-right" /></template>
          </n-button>
        </header>

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

        <RouterView v-slot="{ Component }">
          <component
            :is="Component"
            v-model:draft="draft"
            v-model:hovered-preview-message-id="hoveredPreviewMessageId"
            v-model:visible="settingsVisible"
            v-model:nav-expanded="settingsNavExpanded"
            v-model:section="settingsSection"
            v-model:archive-query="archiveQuery"
            :selected-conversation-id="selectedConversationId"
            :is-new-chat="isNewChat"
            :current-user="currentUser"
            :is-archived-active="isArchivedActive"
            :loading-current-detail="loadingCurrentDetail"
            :rendered-turn-views="renderedTurnViews"
            :message-previews="messagePreviews"
            :selected-preview-message-id="selectedPreviewMessageId"
            :composer-action-pending="composerActionPending"
            :sending="sending"
            :composer-action-disabled="composerActionDisabled"
            :composer-action-label="composerActionLabel"
            :composer-action-icon="composerActionIcon"
            :copied-message-id="copiedMessageId"
            :set-message-pane="setMessagePane"
            :handle-message-scroll="handleMessageScroll"
            :preview-tick-classes="previewTickClasses"
            :jump-to-message="jumpToMessage"
            :user-item-text="userItemText"
            :format-message-time="formatMessageTime"
            :assistant-message-time="assistantMessageTime"
            :message-copy-text="messageCopyText"
            :copy-message="copyMessage"
            :parse-agent-message="parseAgentMessage"
            :normalize-workspace-file-path="normalizeWorkspaceFilePath"
            :display-workspace-process-path="displayWorkspaceProcessPath"
            :open-workspace="openWorkspace"
            :open-workspace-file="openWorkspaceFile"
            :activity-icon="activityIcon"
            :activity-title="activityTitle"
            :open-login="openLogin"
            :appearance="appearance"
            :appearance-options="appearanceOptions"
            :archived-conversations="archivedConversations"
            :archived-sections="archivedConversationSections"
            :loading-list="loadingList"
            :display-conversation-title="displayConversationTitle"
            :format-conversation-date="formatConversationDate"
            @composer-keydown="handleComposerKeydown"
            @composer-action="handleComposerAction"
            @error="toast.error"
            @set-appearance="setAppearance"
            @delete-all-archived="confirmDeleteAllArchived"
            @delete-group-archived="confirmDeleteGroupArchived"
            @delete-archived="confirmDeleteArchived"
            @restore-archived="restoreArchived"
            @resize-start="startSidebarResize"
          />
        </RouterView>

        <RightPanel
          :visible="!isSettingsRoute && Boolean(currentUser) && Boolean(selectedConversationId) && selectedConversationId !== DRAFT_CONVERSATION_ID && workspaceVisible"
          :content-ready="rightPanelContentReady"
          :expanded="workspaceExpanded"
          :conversation-id="selectedConversationId"
          :workspace="activeWorkspace"
          :workspace-tabs="activeWorkspaceTabs"
          :active-workspace-tab-id="activeWorkspaceTabId"
          :active-workspace-url="activeWorkspaceUrl"
          :available-workspaces="pinnedSummaryWorkspaces"
          :active-kind="activeRightPanelKind"
          :has-workspace="hasRightPanelWorkspace || pinnedSummaryWorkspaces.length > 0"
          :has-files="hasRightPanelFiles"
          :file-preview="activeFilePreview"
          :file-tabs="activeFileTabs"
          :active-file-tab-id="activeFileTabId"
          :active-file-path="activeFilePath"
          :file-loading="filePreviewLoading"
          :file-error="filePreviewError"
          :title="sidePanelTitle"
          :reload-version="workspaceReloadVersion"
          :file-badge="activeFilePreviewBadge"
          :file-breadcrumbs="activeFilePreviewBreadcrumbs"
          :file-tree="workspaceFileTree"
          :file-tree-expanded-paths="workspaceFileTreeExpandedPaths"
          :file-tree-loading="workspaceFileTreeLoading"
          :file-tree-loaded="workspaceFileTreeLoaded"
          :file-tree-error="workspaceFileTreeError"
          :file-lines="activeFilePreviewLines"
          @resize-start="startWorkspaceResize"
          @reload="reloadWorkspace"
          @open-workspace="workspacePanel.openWorkspace"
          @select-workspace="workspacePanel.selectWorkspace"
          @close-workspace="workspacePanel.closeWorkspace"
          @switch-files="workspacePanel.switchToFiles"
          @toggle-expanded="toggleWorkspaceExpanded"
          @collapse="toggleRightPanel"
          @select-file="selectWorkspaceFile"
          @reorder-file="reorderWorkspaceFile"
          @close-file="closeFilePreview"
          @toggle-file-tree-directory="toggleWorkspaceFileTreeDirectory"
          @open-tree-file="workspacePanel.openTreeFile"
        />
      </main>
    </div>

    <n-modal
      v-model:show="renameVisible"
      preset="card"
      :bordered="false"
      :mask-closable="!renameSaving"
      :close-on-esc="!renameSaving"
      class="conversation-rename-modal"
      @after-leave="cancelRename"
    >
      <div class="conversation-rename-header">
        <h2>重命名聊天</h2>
        <p>保持简短且易于识别</p>
      </div>
      <input
        ref="renameInput"
        v-model="renameValue"
        class="conversation-rename-input"
        maxlength="80"
        aria-label="对话标题"
        @keydown.enter.prevent="saveRename"
      />
      <template #footer>
        <div class="modal-actions">
          <n-button :disabled="renameSaving" @click="cancelRename">取消</n-button>
          <n-button type="primary" :loading="renameSaving" :disabled="!renameValue.trim()" @click="saveRename">保存</n-button>
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

    <LoginModal
      v-model:visible="loginVisible"
      v-model:mode="loginMode"
      v-model:username="loginUsername"
      v-model:password="loginPassword"
      v-model:email="loginEmail"
      v-model:verification-code="loginVerificationCode"
      :verification-ready="Boolean(loginVerificationUuid)"
      :verification-sending="verificationSending"
      :verification-countdown="verificationCountdown"
      :loading="loginLoading"
      :offset-x="loginModalOffsetX"
      @submit="submitLogin"
      @send-verification-code="sendLoginVerificationCode"
    />


  </n-config-provider>
</template>
