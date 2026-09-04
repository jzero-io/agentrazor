import { computed, nextTick, type ComputedRef, type Ref } from 'vue';
import { conversationApi, type ConversationDetail, type Turn } from '../../service/api';

interface UseConversationComposerOptions {
  selectedConversationId: Ref<string>;
  draftConversationId: string;
  draftConversationGroupId: Ref<string>;
  detailsByConversation: Map<string, ConversationDetail>;
  detail: Ref<ConversationDetail | null>;
  activeDetail: ComputedRef<ConversationDetail | null>;
  sendingRequest: Ref<boolean>;
  creatingConversation: Ref<boolean>;
  sending: Ref<boolean>;
  stopping: Ref<boolean>;
  draftValue: ComputedRef<string>;
  setConversationDraft: (conversationId: string, value: string) => void;
  isDraftConversation: (id?: string) => boolean;
  isConversationRunning: (id: string) => boolean;
  setConversationProcessing: (id: string, processing: boolean) => void;
  locallyStoppedConversationIds: Set<string>;
  createOptimisticTurn: (content: string) => Turn;
  showOptimisticTurn: (conversationId: string, turn: Turn) => void;
  moveOptimisticTurn: (fromConversationId: string, toConversationId: string, turnId: string) => void;
  discardOptimisticTurn: (conversationId: string, turnId: string) => boolean;
  activeTurnError: (error: unknown) => boolean;
  cachedActiveTurn: (conversationId: string) => Turn | null | undefined;
  confirmSentTurn: (conversationId: string, turn: Turn) => void;
  resetActiveTurn: () => void;
  finalizeStoppedTurn: (conversationId: string) => void;
  setConversationDetail: (detail: ConversationDetail) => void;
  upsertConversationListItem: (conversation: ConversationDetail['conversation']) => void;
  syncConversationMetadata: (conversation: ConversationDetail['conversation']) => void;
  revealConversationSection: (conversation: ConversationDetail['conversation']) => void;
  scheduleConversationTitleRefresh: (conversationId: string, conversation: ConversationDetail['conversation']) => void;
  ensureConversationStream: (conversationId: string) => Promise<void>;
  refreshDetail: () => Promise<ConversationDetail | null>;
  enableAutoScroll: () => void;
  scrollToBottom: (options?: { force?: boolean }) => Promise<void> | void;
  syncConversationUrl: (conversationId: string) => void;
  showError: (error: unknown) => void;
}

export function useConversationComposer(options: UseConversationComposerOptions) {
  const canSend = computed(() => Boolean(
    options.draftValue.value.trim()
    && options.selectedConversationId.value
    && !options.creatingConversation.value
    && !options.sendingRequest.value
    && !options.isConversationRunning(options.selectedConversationId.value)
  ));

  const composerActionPending = computed(() => options.creatingConversation.value || options.sendingRequest.value || options.stopping.value);
  const composerActionDisabled = computed(() => options.sending.value ? options.stopping.value : !canSend.value);
  const composerActionLabel = computed(() => {
    if (options.stopping.value) return '正在停止';
    if (options.sending.value) return '停止当前任务';
    if (composerActionPending.value) return '发送中';
    return '发送消息';
  });
  const composerActionIcon = computed(() => {
    if (options.stopping.value || !options.sending.value && composerActionPending.value) return 'solar:refresh-linear';
    if (options.sending.value) return 'solar:stop-bold';
    return 'solar:arrow-up-linear';
  });

  async function sendMessage() {
    const draftKey = options.selectedConversationId.value;
    const content = options.draftValue.value.trim();
    if (!content || options.sendingRequest.value) return;

    const optimisticTurn = options.createOptimisticTurn(content);

    options.sendingRequest.value = true;
    options.enableAutoScroll();
    let conversationId = options.selectedConversationId.value;
    let createdConversation: ConversationDetail['conversation'] | null = null;
    options.showOptimisticTurn(conversationId, optimisticTurn);
    options.setConversationDraft(draftKey, '');
    await nextTick();
    await options.scrollToBottom({ force: true });

    try {
      conversationId = options.selectedConversationId.value;
      if (!conversationId) return;

      const creatingFromDraft = options.isDraftConversation(conversationId);
      if (creatingFromDraft) {
        options.creatingConversation.value = true;
        createdConversation = await conversationApi.create(options.draftConversationGroupId.value);
        conversationId = createdConversation.id;
        if (!conversationId) throw new Error('conversation id is required');

        options.draftConversationGroupId.value = '';
        options.moveOptimisticTurn(draftKey, conversationId, optimisticTurn.id);
        options.selectedConversationId.value = conversationId;
        options.syncConversationUrl(conversationId);
        const draftDetail = options.detail.value?.conversation.id === draftKey ? options.detail.value : null;
        options.setConversationDetail({
          conversation: createdConversation,
          turns: draftDetail?.turns ?? []
        });
        options.upsertConversationListItem(createdConversation);
        options.revealConversationSection(createdConversation);
      } else {
        if (options.isConversationRunning(conversationId)) return;
      }

      await options.ensureConversationStream(conversationId);
      const sent = await conversationApi.send(conversationId, content);

      if (createdConversation) options.scheduleConversationTitleRefresh(conversationId, createdConversation);

      options.setConversationProcessing(conversationId, true);
      if (draftKey !== conversationId) options.setConversationDraft(conversationId, '');
      options.ensureConversationStream(conversationId);

      if (options.selectedConversationId.value === conversationId) {
        const confirmedTurn: Turn = {
          id: sent.id || `turn-${Date.now()}`,
          status: 'inProgress',
          startedAt: sent.startedAt || new Date().toISOString(),
          items: optimisticTurn.items
        };
        options.confirmSentTurn(conversationId, confirmedTurn);
        await options.scrollToBottom({ force: true });
      }
    } catch (error) {
      if (options.activeTurnError(error)) {
        options.setConversationProcessing(conversationId, true);
        options.ensureConversationStream(conversationId);
        if (options.selectedConversationId.value === conversationId) await options.refreshDetail();
        return;
      }

      const discarded = options.discardOptimisticTurn(conversationId, optimisticTurn.id)
        || (conversationId !== draftKey && options.discardOptimisticTurn(draftKey, optimisticTurn.id));
      if (!discarded && options.cachedActiveTurn(conversationId)) {
        options.setConversationProcessing(conversationId, true);
        options.ensureConversationStream(conversationId);
        return;
      }
      if (discarded) {
        if (options.selectedConversationId.value === conversationId) options.setConversationDraft(conversationId, content);
        else if (options.selectedConversationId.value === draftKey) options.setConversationDraft(draftKey, content);
      }
      options.setConversationProcessing(conversationId, false);
      if (options.selectedConversationId.value === conversationId) options.resetActiveTurn();
      options.showError(error);
    } finally {
      options.creatingConversation.value = false;
      options.sendingRequest.value = false;
    }
  }

  async function cancelTurn() {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId || !options.isConversationRunning(conversationId) || options.stopping.value) return;
    options.stopping.value = true;
    options.finalizeStoppedTurn(conversationId);
    try {
      await conversationApi.cancelTurn(conversationId);
    } catch (error) {
      options.locallyStoppedConversationIds.delete(conversationId);
      options.setConversationProcessing(conversationId, false);
      options.showError(error);
    }
  }

  function handleComposerAction() {
    if (options.sending.value) {
      void cancelTurn();
      return;
    }
    void sendMessage();
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
      options.setConversationDraft(
        options.selectedConversationId.value,
        `${textarea.value.slice(0, start)}\n${textarea.value.slice(end)}`
      );
      void nextTick(() => textarea.setSelectionRange(start + 1, start + 1));
      return;
    }

    if (event.altKey || event.ctrlKey || event.metaKey) return;
    event.preventDefault();
    event.stopPropagation();
    void sendMessage();
  }

  return {
    canSend,
    composerActionPending,
    composerActionDisabled,
    composerActionLabel,
    composerActionIcon,
    sendMessage,
    cancelTurn,
    handleComposerAction,
    handleComposerKeydown
  };
}
