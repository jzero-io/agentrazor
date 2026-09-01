import { ref } from 'vue';
import { conversationGroupApi } from '../../service/api';

export interface ConversationGroup {
  id: string;
  name: string;
  collapsed: boolean;
}

interface UseConversationGroupsOptions {
  initialCollapsedGroups?: Record<string, boolean>;
  showError: (error: unknown) => void;
}

export function useConversationGroups(options: UseConversationGroupsOptions) {
  const conversationGroups = ref<ConversationGroup[]>([]);
  const groupEditorVisible = ref(false);
  const groupEditorName = ref('');
  const editingGroupId = ref('');

  async function loadConversationGroups() {
    const previous = new Map(conversationGroups.value.map(group => [group.id, group.collapsed]));
    try {
      const groups = await conversationGroupApi.list();
      conversationGroups.value = groups.map(group => ({
        id: group.id,
        name: group.name,
        collapsed: previous.get(group.id) ?? options.initialCollapsedGroups?.[group.id] ?? false
      }));
    } catch (error) {
      conversationGroups.value = [];
      options.showError(error);
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

  function clearGroups() {
    conversationGroups.value = [];
    groupEditorVisible.value = false;
    groupEditorName.value = '';
    editingGroupId.value = '';
  }

  return {
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
  };
}
