import { computed, reactive, ref, type ComputedRef, type Ref } from 'vue';
import type { Conversation, ConversationDetail, ThreadItem, Turn } from '../../service/api';
import {
  completedProcessSummary,
  formatTurnDuration,
  processDisplayItems,
  showCompletedProcessSummary,
  streamingProcessSummary,
  turnProcessItems,
  turnResultItems,
  type ProcessDisplayItem
} from '../../utils/processDisplay';

export interface TurnView {
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

interface BeginActiveTurnOptions {
  conversationId?: string;
  turn?: Turn;
  id?: string;
  status?: string;
  startedAt?: string;
  resetResultSeen?: boolean;
  restartTimer?: boolean;
}

interface UseConversationTurnsOptions {
  selectedConversationId: Ref<string>;
  conversations: Ref<Conversation[]>;
  activeDetail: ComputedRef<ConversationDetail | null>;
  detailsByConversation: Map<string, ConversationDetail>;
  detail: Ref<ConversationDetail | null>;
  locallyStoppedRunIds: Set<string>;
  locallyStoppedConversationIds: Set<string>;
  setConversationProcessing: (conversationId: string, processing: boolean) => void;
  touchConversationUpdatedAt: (conversationId: string, value?: string) => void;
  isConversationProcessing: (conversation: Conversation) => boolean;
  upsertConversationListItem: (conversation: Conversation) => void;
}

export function userItemText(item: ThreadItem) {
  if (!Array.isArray(item.content)) return '';
  return item.content
    .filter(part => typeof part === 'object' && part?.type === 'text')
    .map(part => typeof part === 'object' ? part.text || '' : '')
    .filter(Boolean)
    .join('\n');
}

export function cloneThreadItem(item: ThreadItem): ThreadItem {
  const next: ThreadItem = { ...item };
  if (Array.isArray(item.content)) next.content = [...item.content] as ThreadItem['content'];
  return next;
}

export function mergeTurnItems(current: ThreadItem[], incoming: ThreadItem[], options: { skipIncomingReasoning?: boolean } = {}) {
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

export function cloneTurnForStream(turn: Turn): Turn {
  return {
    ...turn,
    items: (turn.items || []).map(cloneThreadItem)
  };
}

export function useConversationTurns(options: UseConversationTurnsOptions) {
  const {
    selectedConversationId,
    conversations,
    activeDetail,
    detailsByConversation,
    detail,
    locallyStoppedRunIds,
    locallyStoppedConversationIds,
    setConversationProcessing,
    isConversationProcessing,
    upsertConversationListItem
  } = options;

  const sending = ref(false);
  const stopping = ref(false);
  const activeTurnsByConversation = reactive(new Map<string, Turn>());
  const activeTurnResultSeenByConversation = reactive(new Map<string, boolean>());
  const activeTurnStartedAtByConversation = reactive(new Map<string, number>());
  const processStartedAtByConversation = reactive(new Map<string, number>());
  const processActiveConversationIds = reactive(new Set<string>());
  const nowMs = ref(Date.now());
  let turnTimer: number | undefined;

  const currentStreamingTurn = computed(() =>
    selectedConversationId.value ? activeTurnsByConversation.get(selectedConversationId.value) || null : null
  );
  const currentTurnElapsedMs = computed(() => {
    if (!selectedConversationId.value) return 0;
    const startedAt = processStartedAtByConversation.get(selectedConversationId.value);
    return startedAt ? Math.max(1000, nowMs.value - startedAt) : 1000;
  });
  const currentTurnResultSeen = computed(() =>
    Boolean(selectedConversationId.value && activeTurnResultSeenByConversation.get(selectedConversationId.value))
  );

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
    return Boolean(selectedConversationId.value && processActiveConversationIds.has(selectedConversationId.value));
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

  const renderedTurns = computed(() => normalizedRenderedTurns(activeDetail.value?.turns || [], currentStreamingTurn.value));
  const renderedTurnViews = computed(() => renderedTurns.value.map(createTurnView));

  function cachedActiveTurn(conversationId: string) {
    return activeTurnsByConversation.get(conversationId) || null;
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

  function syncTurnItems(target: ThreadItem[], source: ThreadItem[], opts: { skipIncomingReasoning?: boolean } = {}) {
    const merged = mergeTurnItems(target, source, opts);
    const existingById = new Map(target.map(item => [item.id, item]));
    const nextItems = merged.map(item => {
      const existing = existingById.get(item.id);
      if (!existing) return cloneThreadItem(item);
      syncThreadItem(existing, item);
      return existing;
    });
    target.splice(0, target.length, ...nextItems);
  }

  function syncTurnForStream(target: Turn, source: Turn, opts: { keepStatus?: boolean; skipIncomingReasoning?: boolean } = {}) {
    const keepRestoredRunning = isRestoredRunningTurn(target) || isRestoredRunningTurn(source);
    target.id = source.id;
    if (!opts.keepStatus) target.status = source.status;
    if (source.startedAt) target.startedAt = source.startedAt;
    if (source.completedAt) target.completedAt = source.completedAt;
    if (source.durationMs !== undefined) target.durationMs = source.durationMs;
    if (source.error !== undefined) target.error = source.error;
    if (keepRestoredRunning) (target as Turn & { restoredRunning?: boolean }).restoredRunning = true;
    syncTurnItems(target.items, source.items || [], { skipIncomingReasoning: opts.skipIncomingReasoning });
  }

  function publishActiveTurn(conversationId: string, turn?: Turn) {
    if (!conversationId) return;
    const current = turn || cachedActiveTurn(conversationId);
    if (!current) return;
    if (activeTurnsByConversation.get(conversationId) !== current) activeTurnsByConversation.set(conversationId, current);
    if (activeTurnResultSeen(current) && !activeTurnResultSeenByConversation.get(conversationId)) {
      activeTurnResultSeenByConversation.set(conversationId, true);
    }
  }

  function beginActiveTurn(opts: BeginActiveTurnOptions = {}): Turn {
    const conversationId = opts.conversationId || selectedConversationId.value;
    let current: Turn | null = cachedActiveTurn(conversationId);

    if (opts.turn) {
      if (current) syncTurnForStream(current, opts.turn);
      else current = cloneTurnForStream(opts.turn);
    } else if (!current) {
      current = { id: opts.id || '', status: opts.status || 'inProgress', items: [] };
    }

    if (opts.id) current.id = opts.id;
    if (opts.status) current.status = opts.status;
    if (opts.startedAt) current.startedAt = opts.startedAt;
    if (conversationId && activeTurnsByConversation.get(conversationId) !== current) activeTurnsByConversation.set(conversationId, current);

    if (opts.resetResultSeen) activeTurnResultSeenByConversation.delete(conversationId);
    else if (activeTurnResultSeen(current) && !activeTurnResultSeenByConversation.get(conversationId)) activeTurnResultSeenByConversation.set(conversationId, true);

    setConversationProcessing(conversationId, true);
    if (conversationId === selectedConversationId.value) {
      sending.value = true;
      stopping.value = false;
    }
    if (opts.restartTimer || !activeTurnStartedAtByConversation.has(conversationId)) startTurnTimer(conversationId, current.startedAt || opts.startedAt);
    return current;
  }

  function setConversationDetail(snapshot: ConversationDetail) {
    detailsByConversation.set(snapshot.conversation.id, snapshot);
    if (selectedConversationId.value === snapshot.conversation.id) detail.value = snapshot;
  }

  function clearConversationDetail(id: string) {
    detailsByConversation.delete(id);
    if (detail.value?.conversation.id === id) detail.value = null;
  }

  function confirmSentTurn(conversationId: string, turn: Turn) {
    const targetDetail = detailsByConversation.get(conversationId);
    const persisted = targetDetail?.turns.find(item => item.id === turn.id);
    if (targetDetail && persisted && !isRunningTurnStatus(persisted)) {
      if (!persisted.startedAt) persisted.startedAt = turn.startedAt;
      persisted.items = mergeTurnItems(persisted.items, turn.items);
      setConversationDetail(targetDetail);
      setConversationProcessing(conversationId, false);
      if (conversationId === selectedConversationId.value) clearDisplayedActiveTurn();
      return;
    }

    const current = cachedActiveTurn(conversationId);
    if (!current) {
      beginActiveTurn({ conversationId, turn, resetResultSeen: true, restartTimer: true });
      return;
    }

    const hasStreamedItems = current.items.some(item => item.type !== 'userMessage');
    beginActiveTurn({
      conversationId,
      turn: {
        ...current,
        id: current.id && !current.id.startsWith('pending-') ? current.id : turn.id,
        status: 'inProgress',
        startedAt: current.startedAt || turn.startedAt,
        items: mergeTurnItems(current.items, turn.items)
      },
      resetResultSeen: !hasStreamedItems,
      restartTimer: true
    });
  }

  function createOptimisticTurn(content: string): Turn {
    const pendingId = `pending-${Date.now()}`;
    return {
      id: pendingId,
      status: 'pending',
      startedAt: new Date().toISOString(),
      items: [{
        id: `local-user-${pendingId}`,
        type: 'userMessage',
        content: [{ type: 'text', text: content }]
      }]
    };
  }

  function showOptimisticTurn(conversationId: string, turn: Turn) {
    activeTurnsByConversation.set(conversationId, cloneTurnForStream(turn));
    activeTurnResultSeenByConversation.delete(conversationId);
  }

  function moveOptimisticTurn(fromConversationId: string, toConversationId: string, pendingTurnId: string) {
    const turn = cachedActiveTurn(fromConversationId);
    if (!turn || turn.id !== pendingTurnId) return;
    activeTurnsByConversation.delete(fromConversationId);
    activeTurnResultSeenByConversation.delete(fromConversationId);
    activeTurnsByConversation.set(toConversationId, turn);
  }

  function discardOptimisticTurn(conversationId: string, pendingTurnId: string) {
    const turn = cachedActiveTurn(conversationId);
    if (!turn || turn.id !== pendingTurnId) return false;
    activeTurnsByConversation.delete(conversationId);
    activeTurnResultSeenByConversation.delete(conversationId);
    return true;
  }

  function activeTurnError(error: unknown) {
    return error instanceof Error && error.message.includes('active turn');
  }

  function syncDisplayedActiveTurnState() {
    const running = Boolean(selectedConversationId.value && cachedActiveTurn(selectedConversationId.value));
    sending.value = running;
    if (!running) stopping.value = false;
  }

  function clearDisplayedActiveTurn() {
    sending.value = false;
    stopping.value = false;
  }

  function resetActiveTurn(opts: { clearCache?: boolean } = {}) {
    if (opts.clearCache && selectedConversationId.value) {
      stopTurnTimer(selectedConversationId.value);
      processStartedAtByConversation.delete(selectedConversationId.value);
      processActiveConversationIds.delete(selectedConversationId.value);
      activeTurnsByConversation.delete(selectedConversationId.value);
      activeTurnResultSeenByConversation.delete(selectedConversationId.value);
    }
    clearDisplayedActiveTurn();
  }

  function activeTurnStartedAt(snapshot: ConversationDetail, turn?: Turn) {
    return snapshot.conversation.runningStartedAt
      || turn?.startedAt
      || snapshot.conversation.updatedAt
      || snapshot.conversation.createdAt
      || '';
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
          conversationId: snapshot.conversation.id,
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
      conversationId: snapshot.conversation.id,
      turn: activeTurn,
      startedAt,
      restartTimer: true
    });
  }

  function mergeTurnForDisplay(target: Turn, source: Turn, keepStatus = false) {
    if (!keepStatus) target.status = source.status;
    if (source.durationMs !== undefined) target.durationMs = source.durationMs;
    if (source.completedAt) target.completedAt = source.completedAt;
    if (source.error !== undefined) target.error = source.error;
    target.items = mergeTurnItems(target.items, source.items, { skipIncomingReasoning: true });
  }

  function finishActiveTurn(status: 'completed' | 'failed' | 'stopped', conversationId = selectedConversationId.value, error?: string) {
    const selected = conversationId === selectedConversationId.value;
    const durationMs = stopTurnTimer(conversationId);
    const finishedTurn = conversationId ? activeTurnsByConversation.get(conversationId) || null : null;
    if (!finishedTurn) {
      if (selected) resetActiveTurn();
      return null;
    }

    finishedTurn.status = status;
    if (error) finishedTurn.error = error;
    if (finishedTurn.durationMs === undefined && durationMs !== undefined) finishedTurn.durationMs = durationMs;
    if (status === 'stopped') {
      if (finishedTurn.id) locallyStoppedRunIds.add(finishedTurn.id);
      if (conversationId) locallyStoppedConversationIds.add(conversationId);
    }
    if (conversationId) {
      options.touchConversationUpdatedAt(conversationId);
      setConversationProcessing(conversationId, false);
      activeTurnsByConversation.delete(conversationId);
      activeTurnResultSeenByConversation.delete(conversationId);
    }

    const targetDetail = detailsByConversation.get(conversationId);
    if (targetDetail) {
      const persisted = targetDetail.turns.find(turn => turn.id === finishedTurn.id);
      if (persisted) mergeTurnForDisplay(persisted, finishedTurn, status === 'stopped');
      else targetDetail.turns.push({ ...finishedTurn, items: [...finishedTurn.items] });
      setConversationDetail(targetDetail);
    }

    if (selected) clearDisplayedActiveTurn();
    return finishedTurn;
  }

  function finalizeStoppedTurn(conversationId = selectedConversationId.value) {
    finishActiveTurn('stopped', conversationId);
  }

  function findStreamingItem(conversationId: string, id: string) {
    return cachedActiveTurn(conversationId)?.items.find(value => value.id === id);
  }

  function ensureStreamingItem(conversationId: string, id: string, type: string) {
    const turn = beginActiveTurn({ conversationId });
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

  function upsertStreamingItem(conversationId: string, item: ThreadItem) {
    const turn = beginActiveTurn({ conversationId });
    if (item.type === 'userMessage') {
      turn.items = mergeTurnItems(turn.items, [item]);
      publishActiveTurn(conversationId, turn);
      return;
    }
    const index = turn.items.findIndex(value => value.id === item.id);
    if (index >= 0) syncThreadItem(turn.items[index], normalizeStreamingItem(turn.items[index], item));
    else turn.items.push(normalizeStreamingItem(undefined, item));
    if (item.type === 'agentMessage' && item.phase === 'final_answer' && item.text) activeTurnResultSeenByConversation.set(conversationId, true);
    publishActiveTurn(conversationId, turn);
  }

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

  function startTurnTimer(conversationId: string, startedAt?: string) {
    if (!conversationId) return;
    const parsed = startedAt ? Date.parse(startedAt) : Number.NaN;
    activeTurnStartedAtByConversation.set(conversationId, Number.isFinite(parsed) ? parsed : Date.now());
    ensureTurnTicker();
  }

  function startProcessTimer(conversationId: string, startedAt?: string) {
    if (!conversationId) return;
    processActiveConversationIds.add(conversationId);
    if (processStartedAtByConversation.has(conversationId)) return;
    const parsed = startedAt ? Date.parse(startedAt) : Number.NaN;
    processStartedAtByConversation.set(conversationId, Number.isFinite(parsed) ? parsed : Date.now());
    ensureTurnTicker();
  }

  function restoreProcessState(conversationId: string, turn: Turn, startedAt?: string) {
    if (!conversationId || !hasVisibleProcessItems(turn)) return;
    startProcessTimer(conversationId, startedAt || turn.startedAt);
  }

  function markProcessActive(conversationId: string) {
    startProcessTimer(conversationId);
  }

  function resetProcessTimer(conversationId: string) {
    if (!conversationId) return;
    processStartedAtByConversation.delete(conversationId);
    processActiveConversationIds.delete(conversationId);
  }

  function setTurnElapsedDuration(conversationId: string, durationMs: number) {
    activeTurnStartedAtByConversation.set(conversationId, Date.now() - durationMs);
    ensureTurnTicker();
  }

  function stopTurnTimer(conversationId: string) {
    const startedAt = activeTurnStartedAtByConversation.get(conversationId);
    activeTurnStartedAtByConversation.delete(conversationId);
    processStartedAtByConversation.delete(conversationId);
    processActiveConversationIds.delete(conversationId);
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

  function clearAllActiveTurns() {
    activeTurnsByConversation.clear();
    activeTurnResultSeenByConversation.clear();
    stopAllTurnTimers();
  }

  return {
    sending,
    stopping,
    activeTurnsByConversation,
    activeTurnResultSeenByConversation,
    currentStreamingTurn,
    currentTurnElapsedMs,
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
    setTurnElapsedDuration,
    isVisibleProcessStreamItem,
    upsertStreamingItem,
    finishActiveTurn,
    finalizeStoppedTurn,
    mergeTurnForDisplay,
    stopTurnTimer,
    stopAllTurnTimers,
    clearAllActiveTurns
  };
}
