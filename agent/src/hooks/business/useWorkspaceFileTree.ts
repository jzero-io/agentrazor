import { computed, reactive, type Ref } from 'vue';
import type { WorkspaceEntry } from '../../service/api';

export interface WorkspaceTreeNode extends WorkspaceEntry {
  children: WorkspaceTreeNode[];
}

interface UseWorkspaceFileTreeOptions {
  selectedConversationId: Ref<string>;
  draftConversationId: string;
  fetchEntries: (conversationId: string) => Promise<WorkspaceEntry[]>;
  onError?: (error: unknown) => void;
}

function sortNodes(nodes: WorkspaceTreeNode[]) {
  nodes.sort((left, right) => {
    if (left.type !== right.type) return left.type === 'directory' ? -1 : 1;
    return left.name.localeCompare(right.name, undefined, { numeric: true, sensitivity: 'base' });
  });
  for (const node of nodes) sortNodes(node.children);
  return nodes;
}

function buildTree(entries: WorkspaceEntry[]) {
  const roots: WorkspaceTreeNode[] = [];
  const nodes = new Map<string, WorkspaceTreeNode>();

  for (const entry of entries) {
    nodes.set(entry.path, { ...entry, children: [] });
  }

  for (const node of nodes.values()) {
    const separator = node.path.lastIndexOf('/');
    if (separator < 0) {
      roots.push(node);
      continue;
    }
    const parent = nodes.get(node.path.slice(0, separator));
    if (parent?.type === 'directory') parent.children.push(node);
    else roots.push(node);
  }

  return sortNodes(roots);
}

export function useWorkspaceFileTree(options: UseWorkspaceFileTreeOptions) {
  const entriesByConversation = reactive(new Map<string, WorkspaceEntry[]>());
  const expandedByConversation = reactive(new Map<string, Set<string>>());
  const loadingConversationIds = reactive(new Set<string>());
  const loadedConversationIds = reactive(new Set<string>());
  const errorsByConversation = reactive(new Map<string, string>());

  const conversationId = computed(() => options.selectedConversationId.value);
  const entries = computed(() => entriesByConversation.get(conversationId.value) || []);
  const tree = computed(() => buildTree(entries.value));
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
            .filter(entry => entry.type === 'directory' && !entry.path.includes('/'))
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
    expandedPaths,
    loading,
    loaded,
    error,
    load,
    toggleDirectory,
    removeConversation
  };
}
