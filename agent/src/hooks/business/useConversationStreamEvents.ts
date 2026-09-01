import { nextTick, type Ref } from 'vue';
import type { Conversation, ConversationDetail, StreamEvent, ThreadItem, Turn } from '../../service/api';
import {
  agentMessageDeltaParams,
  codexTurnPayload,
  itemParams,
  nonNegativeNumber,
  streamErrorMessage,
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
  resetResultSeen?: boolean;
  restartTimer?: boolean;
}

interface ConversationStreamEventsOptions {
  selectedConversationId: Ref<string>;
  conversations: Ref<Conversation[]>;
  detailsByConversation: Map<string, ConversationDetail>;
  locallyStoppedRunIds: Set<string>;
  locallyStoppedConversationIds: Set<string>;
  activeTurnResultSeenByConversation: Map<string, boolean>;
  resetProcessTimer: (conversationId: string) => void;
  setTurnElapsedDuration: (conversationId: string, durationMs: number) => void;
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
  scheduleConversationTitleRefresh: (id: string, item?: Pick<Conversation, 'title'> | null) => void;
}

function streamAgentMessagePhase(params: { phase?: string | null; item?: ThreadItem } | undefined, existing?: ThreadItem) {
  if (params?.phase) return params.phase;
  if (params?.item?.phase) return params.item.phase;
  return existing?.phase ?? 'commentary';
}

export function useConversationStreamEvents(options: ConversationStreamEventsOptions) {
  async function handleStreamEvent(event: StreamEvent) {
    const isSelectedConversation = event.conversationId === options.selectedConversationId.value;
    const locallyStopped = Boolean(event.runId && options.locallyStoppedRunIds.has(event.runId))
      || options.locallyStoppedConversationIds.has(event.conversationId);

    if (event.type === 'run.started') {
      options.resetProcessTimer(event.conversationId);
      options.setConversationProcessing(event.conversationId, true);
    }

    if (locallyStopped && event.type !== 'run.completed' && event.type !== 'run.failed') return;

    if (event.type === 'run.started') {
      options.setConversationProcessing(event.conversationId, true);
      const data = event.data as { startedAt?: string } | undefined;
      options.beginActiveTurn({
        conversationId: event.conversationId,
        id: event.runId || undefined,
        status: 'inProgress',
        startedAt: data?.startedAt || event.createdAt,
        resetResultSeen: true
      });
      if (isSelectedConversation) await options.scrollToBottom();
      return;
    }

    if (event.type === 'codex.turn.started') {
      const turn = codexTurnPayload(event.data);
      if (turn) {
        const startedAtIso = unixSecondsToIso(turn.startedAt);
        options.beginActiveTurn({
          conversationId: event.conversationId,
          id: String(turn.id || options.cachedActiveTurn(event.conversationId)?.id || ''),
          status: String(turn.status || 'inProgress'),
          startedAt: startedAtIso,
          restartTimer: Boolean(startedAtIso)
        });
      }
      return;
    }

    if (event.type === 'codex.turn.completed') {
      const turn = codexTurnPayload(event.data);
      if (turn) {
        const activeTurn = options.beginActiveTurn({
          conversationId: event.conversationId,
          id: String(turn.id || options.cachedActiveTurn(event.conversationId)?.id || ''),
          status: String(turn.status || 'completed')
        });
        const durationMs = nonNegativeNumber(turn.durationMs);
        if (durationMs !== undefined) activeTurn.durationMs = durationMs;
        const completedAt = unixSecondsToIso(turn.completedAt);
        if (completedAt) activeTurn.completedAt = completedAt;
        if (Array.isArray(turn.items)) {
          activeTurn.items = options.mergeTurnItems(activeTurn.items, turn.items as ThreadItem[]);
        }
        options.publishActiveTurn(event.conversationId, activeTurn);
      }
      return;
    }

    if (event.type.includes('task_started')) {
      const payload = taskPayload(event.data);
      const startedAt = unixSecondsToIso(payload.started_at);
      if (startedAt) options.beginActiveTurn({ conversationId: event.conversationId, startedAt });
      return;
    }

    if (event.type.includes('task_complete')) {
      const payload = taskPayload(event.data);
      const durationMs = nonNegativeNumber(payload.duration_ms);
      if (durationMs !== undefined) options.setTurnElapsedDuration(event.conversationId, durationMs);
    }

    if (event.type === 'codex.item.reasoning.textDelta') {
      const params = textDeltaParams(event.data);
      const delta = params?.delta ?? '';
      if (!delta) return;
      const itemId = params?.itemId || `stream-reasoning-${event.runId || 'active'}`;
      const item = options.ensureStreamingItem(event.conversationId, itemId, 'reasoning');
      const content = (Array.isArray(item.content) ? item.content : []) as unknown as string[];
      const index = Number(params?.contentIndex) || 0;
      while (content.length <= index) content.push('');
      content[index] = `${content[index] || ''}${delta}`;
      item.content = content;
      options.publishActiveTurn(event.conversationId);
      if (isSelectedConversation) await options.scrollToBottom();
      return;
    }

    if (event.type === 'codex.item.agentMessage.delta') {
      const params = agentMessageDeltaParams(event.data);
      const delta = params?.delta ?? '';
      if (!delta) return;
      const itemId = params?.itemId || params?.item?.id || `stream-agent-${event.runId || 'active'}`;
      const existingItem = options.findStreamingItem(event.conversationId, itemId);
      const phase = streamAgentMessagePhase(params, existingItem);
      if (phase !== 'final_answer') options.markProcessActive(event.conversationId);
      const item = options.ensureStreamingItem(event.conversationId, itemId, 'agentMessage');
      item.phase = phase;
      item.text = `${item.text || ''}${delta}`;
      options.publishActiveTurn(event.conversationId);
      if (item.phase === 'final_answer' && item.text) options.activeTurnResultSeenByConversation.set(event.conversationId, true);
      if (isSelectedConversation) await options.scrollToBottom();
      return;
    }

    if (event.type === 'codex.item.started') {
      const params = itemParams(event.data);
      const streamedItem = params?.item;
      if (streamedItem) {
        if (options.isVisibleProcessStreamItem(streamedItem)) options.markProcessActive(event.conversationId);
        options.upsertStreamingItem(event.conversationId, { ...streamedItem, streamStatus: 'running' });
        if (streamedItem.type === 'agentMessage' && streamedItem.phase === 'final_answer' && streamedItem.text) {
          options.activeTurnResultSeenByConversation.set(event.conversationId, true);
        }
      }
      if (isSelectedConversation) await options.scrollToBottom();
      return;
    }

    if (event.type === 'codex.item.completed') {
      const params = itemParams(event.data);
      if (params?.item) {
        const completedItem = { ...params.item, streamStatus: 'completed' };
        if (options.isVisibleProcessStreamItem(completedItem)) options.markProcessActive(event.conversationId);
        options.upsertStreamingItem(event.conversationId, completedItem);
        if (completedItem.type === 'agentMessage' && completedItem.phase === 'final_answer' && completedItem.text) {
          options.activeTurnResultSeenByConversation.set(event.conversationId, true);
        }
      }
      if (isSelectedConversation) await options.scrollToBottom();
      return;
    }

    if (event.type === 'run.completed' || event.type === 'run.failed') {
      if (locallyStopped) {
        if (event.runId) options.locallyStoppedRunIds.delete(event.runId);
        options.locallyStoppedConversationIds.delete(event.conversationId);
        options.setConversationProcessing(event.conversationId, false);
        options.closeIdleConversationStreams();
        return;
      }

      const completedConversationId = event.conversationId;
      options.setConversationProcessing(completedConversationId, false);
      const completedTurn = options.finishActiveTurn(
        event.type === 'run.completed' ? 'completed' : 'failed',
        completedConversationId,
        event.type === 'run.failed' ? streamErrorMessage(event.data) : undefined
      );
      await nextTick();
      options.closeIdleConversationStreams();
      options.scheduleConversationTitleRefresh(
        completedConversationId,
        options.conversations.value.find(item => item.id === completedConversationId)
      );
      if (completedTurn) {
        const targetDetail = options.detailsByConversation.get(completedConversationId);
        if (targetDetail) {
          const persisted = targetDetail.turns.find(turn => turn.id === completedTurn.id);
          if (persisted) options.mergeTurnForDisplay(persisted, completedTurn);
          else targetDetail.turns.push(completedTurn);
          options.setConversationDetail(targetDetail);
        }
      }
    }
  }

  return {
    handleStreamEvent
  };
}
