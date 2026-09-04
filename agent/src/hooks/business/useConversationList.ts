import { computed, ref, type Ref } from 'vue';
import type { Conversation, ConversationDetail } from '../../service/api';
import { displayConversationTitle, mergeConversationSnapshot, sortConversationsByUpdatedAt } from '../../utils/conversation';
import type { ConversationGroup } from './useConversationGroups';

export function useConversationList(options: {
  selectedConversationId: Ref<string>;
  conversationGroups: Ref<ConversationGroup[]>;
  archiveQuery: Ref<string>;
  draftConversationId: string;
}) {
  const conversations = ref<Conversation[]>([]);
  const processingConversationIds = ref(new Set<string>());

  const visibleConversations = computed(() =>
    sortConversationsByUpdatedAt(conversations.value.filter(item => item.status !== 'archived'))
  );
  const archivedConversations = computed(() =>
    sortConversationsByUpdatedAt(conversations.value.filter(item => item.status === 'archived'))
  );
  const pinnedConversations = computed(() =>
    visibleConversations.value.filter(item => Boolean(item.pinnedAt))
  );
  const conversationList = computed(() =>
    visibleConversations.value.filter(item => !item.pinnedAt && !item.groupId)
  );
  const filteredArchivedConversations = computed(() => {
    const query = options.archiveQuery.value.trim().toLowerCase();
    return query
      ? archivedConversations.value.filter(item => displayConversationTitle(item).toLowerCase().includes(query))
      : archivedConversations.value;
  });
  const archivedConversationSections = computed(() => {
    const byGroup = new Map<string | undefined, Conversation[]>();
    for (const item of filteredArchivedConversations.value) {
      const key = item.groupId || undefined;
      if (!byGroup.has(key)) byGroup.set(key, []);
      byGroup.get(key)!.push(item);
    }
    const sections: { title: string; groupId?: string; items: Conversation[] }[] = [];
    for (const group of options.conversationGroups.value) {
      const items = byGroup.get(group.id);
      if (items?.length) sections.push({ title: group.name, groupId: group.id, items });
    }
    const ungrouped = byGroup.get(undefined);
    if (ungrouped?.length) sections.push({ title: '对话', items: ungrouped });
    return sections;
  });
  const activeConversation = computed(() =>
    conversations.value.find(item => item.id === options.selectedConversationId.value)
  );
  const activeConversationGroup = computed(() => {
    const groupId = activeConversation.value?.groupId;
    if (!groupId) return null;
    return options.conversationGroups.value.find(group => group.id === groupId) || null;
  });
  const activeConversationGroupCount = computed(() => {
    const groupId = activeConversation.value?.groupId;
    if (!groupId) return 0;
    return visibleConversations.value.filter(item => item.groupId === groupId && !item.pinnedAt).length;
  });
  const isArchivedActive = computed(() => activeConversation.value?.status === 'archived');

  function isDraftConversation(id = options.selectedConversationId.value) {
    return id === options.draftConversationId;
  }

  function setConversationProcessing(id: string, processing: boolean) {
    if (!id) return;
    if (processing) touchConversationUpdatedAt(id);
    const current = processingConversationIds.value.has(id);
    if (current !== processing) {
      const next = new Set(processingConversationIds.value);
      if (processing) next.add(id);
      else next.delete(id);
      processingConversationIds.value = next;
    }

    const index = conversations.value.findIndex(item => item.id === id);
    if (index >= 0 && conversations.value[index].running !== processing) {
      conversations.value[index] = { ...conversations.value[index], running: processing };
    }
  }

  function clearConversationProcessing(ids: string[]) {
    if (!ids.length) return;
    const removed = new Set(ids);
    const next = new Set([...processingConversationIds.value].filter(id => !removed.has(id)));
    processingConversationIds.value = next;
  }

  function touchConversationUpdatedAt(id: string, value = new Date().toISOString()) {
    const index = conversations.value.findIndex(item => item.id === id);
    if (index < 0) return;
    conversations.value[index] = mergeConversationSnapshot(conversations.value[index], {
      ...conversations.value[index],
      updatedAt: value
    });
  }

  function isConversationRunning(id: string) {
    return processingConversationIds.value.has(id)
      || Boolean(conversations.value.find(item => item.id === id)?.running);
  }

  function isConversationProcessing(item: Conversation) {
    return item.running || processingConversationIds.value.has(item.id);
  }

  function replaceConversation(updated: Conversation) {
    const index = conversations.value.findIndex(item => item.id === updated.id);
    if (index < 0) return;
    const current = conversations.value[index];
    const running = updated.running || current.running || processingConversationIds.value.has(updated.id);
    conversations.value[index] = mergeConversationSnapshot(current, { ...updated, running });
  }

  function upsertConversationListItem(item: Conversation) {
    const running = item.running || processingConversationIds.value.has(item.id);
    const index = conversations.value.findIndex(value => value.id === item.id);
    if (index >= 0) conversations.value[index] = mergeConversationSnapshot(conversations.value[index], { ...item, running });
    else conversations.value.unshift({ ...item, running });
    setConversationProcessing(item.id, running);
  }

  function applyConversationList(next: Conversation[], preserveLocalProcessing: boolean, shouldPreserveProcessing: (id: string) => boolean) {
    const nextIds = new Set(next.map(item => item.id));
    const runningIds = new Set(next.filter(item => item.running).map(item => item.id));
    if (preserveLocalProcessing) {
      for (const id of processingConversationIds.value) {
        if (nextIds.has(id) && shouldPreserveProcessing(id)) runningIds.add(id);
      }
    }
    processingConversationIds.value = runningIds;
    const previous = new Map(conversations.value.map(item => [item.id, item]));
    conversations.value = next.map(item => mergeConversationSnapshot(previous.get(item.id), item));
    return conversations.value.filter(item => item.running).map(item => item.id);
  }

  function setConversations(next: Conversation[]) {
    conversations.value = next;
  }

  function clearConversations() {
    conversations.value = [];
    processingConversationIds.value = new Set();
  }

  function conversationDetailForDraft(groupId: string): ConversationDetail {
    const now = new Date().toISOString();
    return {
      conversation: {
        id: options.draftConversationId,
        title: '新对话',
        status: 'active',
        groupId: groupId || undefined,
        running: false,
        createdAt: now,
        updatedAt: now
      },
      streamPosition: '',
      turns: []
    };
  }

  return {
    conversations,
    processingConversationIds,
    visibleConversations,
    archivedConversations,
    pinnedConversations,
    conversationList,
    filteredArchivedConversations,
    archivedConversationSections,
    activeConversation,
    activeConversationGroup,
    activeConversationGroupCount,
    isArchivedActive,
    isDraftConversation,
    setConversationProcessing,
    clearConversationProcessing,
    touchConversationUpdatedAt,
    isConversationRunning,
    isConversationProcessing,
    replaceConversation,
    upsertConversationListItem,
    applyConversationList,
    setConversations,
    clearConversations,
    conversationDetailForDraft
  };
}
