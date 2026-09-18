import { computed, reactive, type Ref } from 'vue';
import type { WorkspaceEntry } from '../../service/api';

export interface WorkspaceTreeNode extends WorkspaceEntry {
  children: WorkspaceTreeNode[];
}

export const GENERATED_IMAGES_ASSET_DIRECTORY = '__generated_images__';

interface UseWorkspaceFileTreeOptions {
  selectedConversationId: Ref<string>;
  draftConversationId: string;
  fetchEntries: (conversationId: string) => Promise<WorkspaceEntry[]>;
  onError?: (error: unknown) => void;
}

export function useWorkspaceFileTree(options: UseWorkspaceFileTreeOptions) {
  const entriesByConversation = reactive(new Map<string, WorkspaceEntry[]>());
  const expandedByConversation = reactive(new Map<string, Set<string>>());
  const loadingConversationIds = reactive(new Set<string>());
  const loadedConversationIds = reactive(new Set<string>());
  const errorsByConversation = reactive(new Map<string, string>());

  const conversationId = computed(() => options.selectedConversationId.value);
  const entries = computed(() => entriesByConversation.get(conversationId.value) || []);
  const imageAssets = computed(() => {
    const root = entries.value.find(entry =>
      entry.type === 'directory' && entry.path === GENERATED_IMAGES_ASSET_DIRECTORY
    );
    return (root?.children || []).filter(entry => entry.type === 'file') as WorkspaceTreeNode[];
  });
  const tree = computed(() => entries.value.filter(entry =>
    entry.path !== GENERATED_IMAGES_ASSET_DIRECTORY
  ) as WorkspaceTreeNode[]);
  const expandedPaths = computed(() => expandedByConversation.get(conversationId.value) || new Set<string>());
  const loading = computed(() => loadingConversationIds.has(conversationId.value));
  const loaded = computed(() => loadedConversationIds.has(conversationId.value));
  const error = computed(() => errorsByConversation.get(conversationId.value) || '');

  async function load(force = false) {
    const id = conversationId.value;
    if (!id || id === options.draftConversationId || loadingConversationIds.has(id)) return;
    if (!force && loadedConversationIds.has(id)) return;

    loadingConversationIds.add(id);
    errorsByConversation.delete(id);
    try {
      const nextEntries = await options.fetchEntries(id);
      if (conversationId.value !== id && !force) return;
      entriesByConversation.set(id, nextEntries);
      loadedConversationIds.add(id);
      if (!expandedByConversation.has(id)) {
        expandedByConversation.set(id, new Set(
          nextEntries
            .filter(entry =>
              entry.type === 'directory'
              && entry.path !== GENERATED_IMAGES_ASSET_DIRECTORY
              && !entry.path.includes('/')
            )
            .map(entry => entry.path)
        ));
      }
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : '文件列表加载失败';
      errorsByConversation.set(id, message);
      options.onError?.(loadError);
    } finally {
      loadingConversationIds.delete(id);
    }
  }

  function toggleDirectory(path: string) {
    const id = conversationId.value;
    if (!id) return;
    const expanded = expandedByConversation.get(id) || new Set<string>();
    if (expanded.has(path)) expanded.delete(path);
    else expanded.add(path);
    expandedByConversation.set(id, expanded);
  }

  function removeConversation(id: string) {
    entriesByConversation.delete(id);
    expandedByConversation.delete(id);
    loadingConversationIds.delete(id);
    loadedConversationIds.delete(id);
    errorsByConversation.delete(id);
  }

  return {
    entries,
    tree,
    imageAssets,
    expandedPaths,
    loading,
    loaded,
    error,
    load,
    toggleDirectory,
    removeConversation
  };
}
