import { computed, nextTick, onBeforeUnmount, reactive, type ComputedRef, type Ref } from 'vue';
import { conversationApi, type ConversationDetail, type MessageAttachment, type Turn } from '../../service/api';

const MAX_ATTACHMENTS = 10;
const MAX_ATTACHMENT_SIZE = 10 * 1024 * 1024;

export interface ComposerAttachment {
  id: string;
  file: File;
  name: string;
  size: number;
  kind: 'image' | 'file';
  previewUrl?: string;
  uploaded?: MessageAttachment;
  uploadedConversationId?: string;
}

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
  createOptimisticTurn: (content: string, attachments?: Array<{ name: string; kind: 'image' | 'file'; previewUrl?: string }>) => Turn;
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
  const attachmentsByConversation = reactive(new Map<string, ComposerAttachment[]>());
  const attachments = computed(() => attachmentsByConversation.get(options.selectedConversationId.value) || []);

  const canSend = computed(() => Boolean(
    (options.draftValue.value.trim() || attachments.value.length)
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

  function addAttachments(files: File[]) {
    const conversationId = options.selectedConversationId.value;
    if (!conversationId || files.length === 0) return;
    const current = [...attachments.value];
    for (const file of files) {
      if (current.length >= MAX_ATTACHMENTS) {
        options.showError(`每条消息最多添加 ${MAX_ATTACHMENTS} 个附件`);
        break;
      }
      if (file.size <= 0) {
        options.showError(`${file.name || '附件'} 是空文件`);
        continue;
      }
      if (file.size > MAX_ATTACHMENT_SIZE) {
        options.showError(`${file.name} 超过 10 MB`);
        continue;
      }
      const duplicate = current.some(item =>
        item.name === file.name && item.size === file.size && item.file.lastModified === file.lastModified
      );
      if (duplicate) continue;
      const kind = file.type.startsWith('image/') ? 'image' : 'file';
      current.push({
        id: typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : `attachment-${Date.now()}-${current.length}`,
        file,
        name: file.name || 'attachment',
        size: file.size,
        kind,
        previewUrl: kind === 'image' ? URL.createObjectURL(file) : undefined
      });
    }
    attachmentsByConversation.set(conversationId, current);
  }

  function revokeAttachment(attachment: ComposerAttachment) {
    if (attachment.previewUrl) URL.revokeObjectURL(attachment.previewUrl);
  }

  function removeAttachment(id: string) {
    const conversationId = options.selectedConversationId.value;
    const current = attachments.value;
    const target = current.find(item => item.id === id);
    if (target) revokeAttachment(target);
    const next = current.filter(item => item.id !== id);
    if (next.length) attachmentsByConversation.set(conversationId, next);
    else attachmentsByConversation.delete(conversationId);
  }

  function moveAttachments(fromConversationId: string, toConversationId: string) {
    const current = attachmentsByConversation.get(fromConversationId);
    if (!current?.length || fromConversationId === toConversationId) return;
    attachmentsByConversation.delete(fromConversationId);
    attachmentsByConversation.set(toConversationId, current);
  }

  function clearAttachments(conversationId: string) {
    const current = attachmentsByConversation.get(conversationId) || [];
    current.forEach(revokeAttachment);
    attachmentsByConversation.delete(conversationId);
  }

  async function uploadAttachments(conversationId: string, values: ComposerAttachment[]) {
    const uploaded: MessageAttachment[] = [];
    for (const attachment of values) {
      if (attachment.uploaded && attachment.uploadedConversationId === conversationId) {
        uploaded.push(attachment.uploaded);
        continue;
      }
      attachment.uploaded = await conversationApi.uploadAttachment(conversationId, attachment.file);
      attachment.uploadedConversationId = conversationId;
      uploaded.push(attachment.uploaded);
    }
    return uploaded;
  }

  async function sendMessage() {
    const draftKey = options.selectedConversationId.value;
    const content = options.draftValue.value.trim();
    const messageAttachments = [...attachments.value];
    if ((!content && messageAttachments.length === 0) || options.sendingRequest.value) return;

    const optimisticTurn = options.createOptimisticTurn(content, messageAttachments);

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
        moveAttachments(draftKey, conversationId);
        options.selectedConversationId.value = conversationId;
        options.syncConversationUrl(conversationId);
        const draftDetail = options.detail.value?.conversation.id === draftKey ? options.detail.value : null;
        options.setConversationDetail({
          conversation: createdConversation,
          streamPosition: draftDetail?.streamPosition || '',
          turns: draftDetail?.turns ?? []
        });
        options.upsertConversationListItem(createdConversation);
        options.revealConversationSection(createdConversation);
      } else {
        if (options.isConversationRunning(conversationId)) return;
      }

      await options.ensureConversationStream(conversationId);
      const uploadedAttachments = await uploadAttachments(conversationId, messageAttachments);
      const sent = await conversationApi.send(conversationId, content, uploadedAttachments);
      clearAttachments(conversationId);

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
        clearAttachments(conversationId);
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

  onBeforeUnmount(() => {
    for (const values of attachmentsByConversation.values()) values.forEach(revokeAttachment);
    attachmentsByConversation.clear();
  });

  return {
    attachments,
    addAttachments,
    removeAttachment,
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
