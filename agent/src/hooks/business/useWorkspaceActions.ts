import { computed, type Ref } from 'vue';
import type { ThreadItem, Turn } from '../../service/api';
import type { FilePreview, WorkspaceDescriptor, useWorkspacePanel } from './useWorkspacePanel';

interface ParsedAgentMessage {
  markdown: string;
  workspaces: WorkspaceDescriptor[];
}

type WorkspacePanelState = ReturnType<typeof useWorkspacePanel>;

interface UseWorkspaceActionsOptions {
  renderedTurns: Ref<Turn[]>;
  workspacePanel: WorkspacePanelState;
  pinnedSummaryOpen: Ref<boolean>;
  workspaceVisible: Ref<boolean>;
  activeWorkspace: Ref<WorkspaceDescriptor | null>;
  activeFilePreview: Ref<FilePreview | null>;
}

export function parseWorkspaceLinks(content: string): ParsedAgentMessage {
  const workspaces: WorkspaceDescriptor[] = [];
  const markdown = content.replace(/```json\s*([\s\S]*?)```/gi, (block, raw: string) => {
    try {
      const value = JSON.parse(raw.trim()) as Partial<WorkspaceDescriptor>;
      if (value.type !== 'workspace' || typeof value.title !== 'string' || typeof value.url !== 'string') return block;
      workspaces.push({ type: 'workspace', title: value.title, url: value.url });
      return '';
    } catch {
      return block;
    }
  });
  return { markdown: markdown.trim(), workspaces };
}

function agentMessageWorkspaces(item: ThreadItem) {
  if (item.type !== 'agentMessage' || !item.text) return [];
  return parseWorkspaceLinks(item.text).workspaces;
}

export function useWorkspaceActions(options: UseWorkspaceActionsOptions) {
  const rightPanelOpen = computed(() => options.workspaceVisible.value || options.pinnedSummaryOpen.value);
  const rightPanelContentReady = computed(() => Boolean(
    options.workspaceVisible.value
    || options.activeWorkspace.value
    || options.activeFilePreview.value
    || options.workspacePanel.fileLoading.value
    || options.workspacePanel.fileError.value
  ));
  const pinnedSummaryWorkspaces = computed(() => {
    const byUrl = new Map<string, WorkspaceDescriptor>();
    for (const turn of options.renderedTurns.value) {
      for (const item of turn.items) {
        for (const workspace of agentMessageWorkspaces(item)) byUrl.set(workspace.url, workspace);
      }
    }
    return [...byUrl.values()];
  });

  function parseAgentMessage(content: string, _streaming = false): ParsedAgentMessage {
    return parseWorkspaceLinks(content);
  }

  function normalizeWorkspaceFilePath(href: string) {
    return options.workspacePanel.normalizeFilePath(href) || '';
  }

  function openWorkspace(workspace: WorkspaceDescriptor) {
    options.pinnedSummaryOpen.value = false;
    options.workspacePanel.replaceWithWorkspace(workspace);
  }

  async function openWorkspaceFile(path: string) {
    options.pinnedSummaryOpen.value = false;
    await options.workspacePanel.openFile(path);
  }

  function closeFilePreview(path?: string) {
    options.workspacePanel.closeFile(path);
  }

  function displayWorkspaceFilePath(file: FilePreview) {
    return options.workspacePanel.displayWorkspaceFilePath(file);
  }

  function displayWorkspaceProcessPath(filePath: string) {
    return options.workspacePanel.displayWorkspaceProcessPath(filePath);
  }

  function collapseWorkspace() {
    options.workspacePanel.collapse();
  }

  function toggleRightPanel() {
    if (rightPanelOpen.value) {
      options.pinnedSummaryOpen.value = false;
      collapseWorkspace();
      return;
    }
    options.pinnedSummaryOpen.value = false;
    if (!options.workspacePanel.activeKind.value) {
      options.workspacePanel.showLauncher();
      return;
    }
    options.workspaceVisible.value = true;
  }

  function toggleWorkspaceExpanded() {
    options.workspacePanel.toggleExpanded();
  }

  function reloadWorkspace() {
    options.workspacePanel.reload();
  }

  function openPinnedWorkspace(workspace: WorkspaceDescriptor) {
    openWorkspace(workspace);
  }

  return {
    rightPanelOpen,
    rightPanelContentReady,
    pinnedSummaryWorkspaces,
    parseAgentMessage,
    normalizeWorkspaceFilePath,
    openWorkspace,
    openWorkspaceFile,
    closeFilePreview,
    displayWorkspaceFilePath,
    displayWorkspaceProcessPath,
    collapseWorkspace,
    toggleRightPanel,
    toggleWorkspaceExpanded,
    reloadWorkspace,
    openPinnedWorkspace
  };
}
