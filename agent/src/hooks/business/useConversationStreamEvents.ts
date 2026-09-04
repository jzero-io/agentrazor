import { nextTick, type Ref } from 'vue';
import type { Conversation, ConversationDetail, StreamEvent, ThreadItem, Turn } from '../../service/api';
import {
  agentMessageDeltaParams,
  codexTurnError,
  codexTurnPayload,
  itemParams,
  nonNegativeNumber,
  taskPayload,
  textDeltaParams,
  unixSecondsToIso
} from '../../utils/streamEvents';

interface BeginActiveTurnOptions {
  conversationId?: string;
  turn?: Turn;
  id?: string;
  status?: string;
  startedAt?: string;
  observedAt?: string;
  resetResultSeen?: boolean;
  restartTimer?: boolean;
}

interface ConversationStreamEventsOptions {
  selectedConversationId: Ref<string>;
  conversations: Ref<Conversation[]>;
  detailsByConversation: Map<string, ConversationDetail>;
  locallyStoppedTurnIds: Set<string>;
  locallyStoppedConversationIds: Set<string>;
  activeTurnResultSeenByConversation: Map<string, boolean>;
  resetProcessTimer: (conversationId: string) => void;
  setConversationProcessing: (conversationId: string, running: boolean) => void;
  beginActiveTurn: (options?: BeginActiveTurnOptions) => Turn;
  cachedActiveTurn: (conversationId: string) => Turn | null;
  mergeTurnItems: (current: ThreadItem[], incoming: ThreadItem[], options?: { skipIncomingReasoning?: boolean }) => ThreadItem[];
  publishActiveTurn: (conversationId: string, turn?: Turn) => void;
  findStreamingItem: (conversationId: string, id: string) => ThreadItem | undefined;
  ensureStreamingItem: (conversationId: string, id: string, type: string) => ThreadItem;
  markProcessActive: (conversationId: string) => void;
  isVisibleProcessStreamItem: (item: ThreadItem) => boolean;
  upsertStreamingItem: (conversationId: string, item: ThreadItem) => void;
  finishActiveTurn: (status: 'completed' | 'failed' | 'stopped', conversationId?: string, error?: string) => Turn | null;
  mergeTurnForDisplay: (target: Turn, source: Turn, keepStatus?: boolean) => void;
  setConversationDetail: (snapshot: ConversationDetail) => void;
  scrollToBottom: () => Promise<void>;
  closeIdleConversationStreams: () => void;
}

interface PendingStreamDelta {
  conversationId: string;
  itemId: string;
  kind: 'reasoning' | 'agentMessage';
  contentIndex: number;
  phase: string;
  text: string;
}

function streamAgentMessagePhase(params: { phase?: string | null; item?: ThreadItem } | undefined, existing?: ThreadItem) {
  if (params?.phase) return params.phase;
  if (params?.item?.phase) return params.item.phase;
  return existing?.phase ?? 'commentary';
}

export function useConversationStreamEvents(options: ConversationStreamEventsOptions) {
  const pendingDeltas = new Map<string, PendingStreamDelta>();
  let pendingFrame: number | undefined;

  function queueDelta(delta: PendingStreamDelta) {
    const key = `${delta.conversationId}\u0000${delta.kind}\u0000${delta.itemId}\u0000${delta.contentIndex}\u0000${delta.phase}`;
    const pending = pendingDeltas.get(key);
    if (pending) pending.text += delta.text;
    else pendingDeltas.set(key, delta);
    if (pendingFrame !== undefined) return;
    pendingFrame = window.requestAnimationFrame(() => {
      pendingFrame = undefined;
      flushPendingDeltas();
    });
  }

  function flushPendingDeltas(conversationId?: string) {
    const changedConversations = new Set<string>();
    for (const [key, pending] of pendingDeltas) {
      if (conversationId && pending.conversationId !== conversationId) continue;
      pendingDeltas.delete(key);
      if (!options.cachedActiveTurn(pending.conversationId)) continue;

      const item = options.ensureStreamingItem(pending.conversationId, pending.itemId, pending.kind);
      if (pending.kind === 'reasoning') {
        const content = (Array.isArray(item.content) ? item.content : []) as unknown as string[];
        while (content.length <= pending.contentIndex) content.push('');
        content[pending.contentIndex] = `${content[pending.contentIndex] || ''}${pending.text}`;
        item.content = content;
      } else {
        item.phase = pending.phase;
        item.text = `${item.text || ''}${pending.text}`;
        if (item.phase === 'final_answer' && item.text) {
          options.activeTurnResultSeenByConversation.set(pending.conversationId, true);
        }
      }
      changedConversations.add(pending.conversationId);
    }

    for (const id of changedConversations) options.publishActiveTurn(id);
    if (changedConversations.has(options.selectedConversationId.value)) void options.scrollToBottom();
  }

  async function handleStreamEvent(event: StreamEvent) {
    const isSelectedConversation = event.conversationId === options.selectedConversationId.value;
    const locallyStopped = Boolean(event.turnId && options.locallyStoppedTurnIds.has(event.turnId))
      || options.locallyStoppedConversationIds.has(event.conversationId);

    if (locallyStopped && event.type !== 'turn.completed') return;
    if (event.type !== 'item.reasoning.textDelta' && event.type !== 'item.agentMessage.delta') {
      flushPendingDeltas(event.conversationId);
    }

    if (event.type === 'turn.started') {
      const turn = codexTurnPayload(event.data);
      const startedAt = unixSecondsToIso(turn?.startedAt) || event.createdAt;
      options.resetProcessTimer(event.conversationId);
      options.setConversationProcessing(event.conversationId, true);
      options.beginActiveTurn({
        conversationId: event.conversationId,
        id: String(turn?.id || event.turnId || options.cachedActiveTurn(event.conversationId)?.id || ''),
        status: String(turn?.status || 'inProgress'),
        startedAt,
        observedAt: event.createdAt,
        resetResultSeen: true,
        restartTimer: true
      });
      if (isSelectedConversation) void options.scrollToBottom();
      return;
    }

    if (event.type === 'turn.completed') {
      if (locallyStopped) {
        if (event.turnId) options.locallyStoppedTurnIds.delete(event.turnId);
        options.locallyStoppedConversationIds.delete(event.conversationId);
        options.setConversationProcessing(event.conversationId, false);
        options.closeIdleConversationStreams();
        return;
      }

      const turn = codexTurnPayload(event.data);
      if (turn) {
        const activeTurn = options.cachedActiveTurn(event.conversationId) || options.beginActiveTurn({
          conversationId: event.conversationId,
          id: String(turn.id || event.turnId || ''),
          status: String(turn.status || 'completed')
        });
        if (turn.id || event.turnId) activeTurn.id = String(turn.id || event.turnId);
        activeTurn.status = String(turn.status || 'completed');
        const durationMs = nonNegativeNumber(turn.durationMs);
        if (durationMs !== undefined) activeTurn.durationMs = durationMs;
        const completedAt = unixSecondsToIso(turn.completedAt);
        if (completedAt) activeTurn.completedAt = completedAt;
        if (Array.isArray(turn.items)) {
          activeTurn.items = options.mergeTurnItems(activeTurn.items, turn.items as ThreadItem[]);
        }
        options.publishActiveTurn(event.conversationId, activeTurn);
      }

      const normalizedStatus = String(turn?.status || 'completed').toLowerCase();
      const completedTurn = options.finishActiveTurn(
        normalizedStatus === 'interrupted' ? 'stopped' : normalizedStatus === 'failed' ? 'failed' : 'completed',
        event.conversationId,
        codexTurnError(turn) || undefined
      );
      await nextTick();
      options.closeIdleConversationStreams();
      if (completedTurn) {
        const targetDetail = options.detailsByConversation.get(event.conversationId);
        if (targetDetail) {
          const persisted = targetDetail.turns.find(item => item.id === completedTurn.id);
          if (persisted) options.mergeTurnForDisplay(persisted, completedTurn);
          else targetDetail.turns.push(completedTurn);
          options.setConversationDetail(targetDetail);
        }
      }
      return;
    }

    // Item events are only meaningful while their turn is active. Ignoring
    // late delivery here prevents a completed turn from being recreated.
    if (!options.cachedActiveTurn(event.conversationId)) return;

    if (event.type.includes('task_started')) {
      const payload = taskPayload(event.data);
      const startedAt = unixSecondsToIso(payload.started_at);
      if (startedAt) options.beginActiveTurn({ conversationId: event.conversationId, startedAt });
      return;
    }

    if (event.type === 'item.reasoning.textDelta') {
      const params = textDeltaParams(event.data);
      const delta = params?.delta ?? '';
      if (!delta) return;
      const itemId = params?.itemId || `stream-reasoning-${event.turnId || 'active'}`;
      const index = Number(params?.contentIndex) || 0;
      queueDelta({
        conversationId: event.conversationId,
        itemId,
        kind: 'reasoning',
        contentIndex: index,
        phase: '',
        text: delta
      });
      return;
    }

    if (event.type === 'item.agentMessage.delta') {
      const params = agentMessageDeltaParams(event.data);
      const delta = params?.delta ?? '';
      if (!delta) return;
      const itemId = params?.itemId || params?.item?.id || `stream-agent-${event.turnId || 'active'}`;
      const existingItem = options.findStreamingItem(event.conversationId, itemId);
      const phase = streamAgentMessagePhase(params, existingItem);
      if (phase !== 'final_answer') options.markProcessActive(event.conversationId);
      queueDelta({
        conversationId: event.conversationId,
        itemId,
        kind: 'agentMessage',
        contentIndex: 0,
        phase,
        text: delta
      });
      return;
    }

    if (event.type === 'item.started') {
      const params = itemParams(event.data);
      const streamedItem = params?.item;
      if (streamedItem) {
        if (options.isVisibleProcessStreamItem(streamedItem)) options.markProcessActive(event.conversationId);
        options.upsertStreamingItem(event.conversationId, { ...streamedItem, streamStatus: 'running' });
        if (streamedItem.type === 'agentMessage' && streamedItem.phase === 'final_answer' && streamedItem.text) {
          options.activeTurnResultSeenByConversation.set(event.conversationId, true);
        }
      }
      if (isSelectedConversation) void options.scrollToBottom();
      return;
    }

    if (event.type === 'item.completed') {
      const params = itemParams(event.data);
      if (params?.item) {
        const completedItem = { ...params.item, streamStatus: 'completed' };
        if (options.isVisibleProcessStreamItem(completedItem)) options.markProcessActive(event.conversationId);
        options.upsertStreamingItem(event.conversationId, completedItem);
        const mergedItem = options.findStreamingItem(event.conversationId, completedItem.id) || completedItem;
        if (mergedItem.type === 'agentMessage' && mergedItem.phase === 'final_answer') {
          if (mergedItem.text) options.activeTurnResultSeenByConversation.set(event.conversationId, true);
        }
      }
      if (isSelectedConversation) void options.scrollToBottom();
      return;
    }

  }

  return {
    handleStreamEvent
  };
}
