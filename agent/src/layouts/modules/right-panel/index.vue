<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { Icon } from '@iconify/vue';
import { NButton, NSpin } from 'naive-ui';
import type { FilePreview, FilePreviewTab, WorkspaceDescriptor, WorkspaceTab } from '../../../hooks/business/useWorkspacePanel';
import type { WorkspaceTreeNode } from '../../../hooks/business/useWorkspaceFileTree';
import FileTreeNode from './FileTreeNode.vue';
import { fileVisualKind, fileVisualLabel } from './filePresentation';

defineOptions({
  name: 'RightPanel'
});

const props = defineProps<{
  visible: boolean;
  contentReady: boolean;
  expanded: boolean;
  conversationId: string;
  workspace: WorkspaceDescriptor | null;
  workspaceTabs: WorkspaceTab[];
  activeWorkspaceTabId: string;
  activeWorkspaceUrl: string;
  availableWorkspaces: WorkspaceDescriptor[];
  activeKind: string;
  hasWorkspace: boolean;
  hasFiles: boolean;
  filePreview: FilePreview | null;
  fileTabs: FilePreviewTab[];
  activeFileTabId: string;
  activeFilePath: string;
  fileLoading: boolean;
  fileError: string;
  title: string;
  reloadVersion: number;
  fileBadge: string;
  fileBreadcrumbs: string[];
  fileLines: Array<{ number: number; html: string }>;
  fileTree: WorkspaceTreeNode[];
  fileTreeExpandedPaths: Set<string>;
  fileTreeLoading: boolean;
  fileTreeLoaded: boolean;
  fileTreeError: string;
}>();

const emit = defineEmits<{
  resizeStart: [event: PointerEvent];
  reload: [];
  openWorkspace: [workspace: WorkspaceDescriptor, forceNewTab?: boolean];
  selectWorkspace: [tabId: string];
  closeWorkspace: [tabId?: string];
  switchFiles: [];
  toggleExpanded: [];
  collapse: [];
  selectFile: [tabId: string];
  closeFile: [tabId?: string];
  reorderFile: [fromTabId: string, toTabId: string];
  toggleFileTreeDirectory: [path: string];
  openTreeFile: [path: string, forceNewTab?: boolean];
}>();

const draggingFileTabId = ref('');
const fileTreeVisible = ref(false);
const addFileTabMode = ref(false);
const pickerOpen = ref(false);
const pickerSection = ref<'root' | 'workspace'>('root');
const panelElement = ref<HTMLElement | null>(null);
const pickerElement = ref<HTMLElement | null>(null);
const addButtonElement = ref<HTMLElement | null>(null);
const pickerPosition = ref({ left: 10, top: 64, width: 320 });
const breadcrumbPopupElement = ref<HTMLElement | null>(null);
const breadcrumbPopupOpen = ref(false);
const breadcrumbPopupPath = ref('');
const breadcrumbPopupExpandedPaths = ref<Set<string>>(new Set());
const breadcrumbPopupPosition = ref({ left: 10, top: 112, width: 360 });

watch(
  () => props.conversationId,
  () => {
    fileTreeVisible.value = false;
    addFileTabMode.value = false;
    pickerOpen.value = false;
    pickerSection.value = 'root';
    breadcrumbPopupOpen.value = false;
  }
);

watch(pickerOpen, open => {
  if (!open) pickerSection.value = 'root';
});

const activeTreePath = computed(() => {
  const decodedPath = decodeURIComponent((props.activeFilePath || '').split('?')[0] || '');
  const marker = `/agentrazor-home/${props.conversationId}/`;
  const markerIndex = decodedPath.indexOf(marker);
  if (markerIndex >= 0) return decodedPath.slice(markerIndex + marker.length);
  return '';
});

function findTreeNode(nodes: WorkspaceTreeNode[], path: string): WorkspaceTreeNode | null {
  for (const node of nodes) {
    if (node.path === path) return node;
    const child = findTreeNode(node.children, path);
    if (child) return child;
  }
  return null;
}

const breadcrumbPopupRoot = computed(() => findTreeNode(props.fileTree, breadcrumbPopupPath.value));

function toggleFileTree() {
  breadcrumbPopupOpen.value = false;
  const nextVisible = !fileTreeVisible.value;
  fileTreeVisible.value = nextVisible;
  addFileTabMode.value = false;
  if (nextVisible) emit('switchFiles');
}

function openFiles(forceNewTab: boolean) {
  pickerOpen.value = false;
  fileTreeVisible.value = true;
  addFileTabMode.value = forceNewTab;
  emit('switchFiles');
}

function openTreeFile(path: string) {
  emit('openTreeFile', path, addFileTabMode.value);
  addFileTabMode.value = false;
  breadcrumbPopupOpen.value = false;
}

function openBreadcrumbDirectory(index: number, event: MouseEvent) {
  const path = props.fileBreadcrumbs.slice(0, index + 1).join('/');
  if (!path) return;
  addFileTabMode.value = false;
  emit('switchFiles');
  breadcrumbPopupPath.value = path;
  breadcrumbPopupExpandedPaths.value = new Set([path]);
  const panelRect = panelElement.value?.getBoundingClientRect();
  const targetRect = (event.currentTarget as HTMLElement).getBoundingClientRect();
  if (panelRect) {
    const width = Math.max(260, Math.min(420, panelRect.width - 20));
    breadcrumbPopupPosition.value = {
      left: Math.max(panelRect.left + 10, Math.min(targetRect.left, panelRect.right - width - 10)),
      top: targetRect.bottom + 6,
      width
    };
  }
  breadcrumbPopupOpen.value = true;
}

function toggleBreadcrumbPopupDirectory(path: string) {
  const expandedPaths = new Set(breadcrumbPopupExpandedPaths.value);
  if (expandedPaths.has(path)) expandedPaths.delete(path);
  else expandedPaths.add(path);
  breadcrumbPopupExpandedPaths.value = expandedPaths;
}

function openWorkspaceChoice(workspace: WorkspaceDescriptor) {
  pickerOpen.value = false;
  pickerSection.value = 'root';
  emit('openWorkspace', workspace, true);
}

function togglePicker() {
  if (!pickerOpen.value) {
    const panelRect = panelElement.value?.getBoundingClientRect();
    const buttonRect = addButtonElement.value?.getBoundingClientRect();
    if (!panelRect || !buttonRect) return;
    const panelWidth = panelRect.width;
    const width = Math.max(180, Math.min(360, panelWidth - 20));
    const left = Math.max(panelRect.left + 10, Math.min(buttonRect.left - 48, panelRect.right - width - 10));
    pickerPosition.value = { left, top: buttonRect.bottom + 6, width };
  }
  pickerOpen.value = !pickerOpen.value;
}

function handlePickerOutside(event: PointerEvent) {
  const target = event.target as Node;
  const targetElement = target instanceof Element ? target : target.parentElement;
  if (!pickerElement.value?.contains(target) && !addButtonElement.value?.contains(target)) {
    pickerOpen.value = false;
  }
  if (!breadcrumbPopupElement.value?.contains(target) && !targetElement?.closest('.file-preview-breadcrumb-part')) {
    breadcrumbPopupOpen.value = false;
  }
}

onMounted(() => document.addEventListener('pointerdown', handlePickerOutside));
onBeforeUnmount(() => document.removeEventListener('pointerdown', handlePickerOutside));

function openFirstWorkspace(event: MouseEvent) {
  if (props.availableWorkspaces.length === 1) {
    emit('openWorkspace', props.availableWorkspaces[0]);
    return;
  }
  if (props.availableWorkspaces.length > 1) {
    const panelRect = panelElement.value?.getBoundingClientRect();
    if (!panelRect) return;
    const width = Math.max(180, Math.min(360, panelRect.width - 20));
    const estimatedHeight = 50 + props.availableWorkspaces.length * 42;
    const belowTop = event.clientY + 12;
    const top = belowTop + estimatedHeight <= panelRect.bottom - 10
      ? belowTop
      : Math.max(panelRect.top + 10, event.clientY - estimatedHeight - 12);
    pickerPosition.value = {
      left: Math.max(panelRect.left + 10, Math.min(event.clientX - width / 2, panelRect.right - width - 10)),
      top,
      width
    };
    pickerSection.value = 'workspace';
    pickerOpen.value = true;
  }
}

function startTabDrag(event: DragEvent, tabId: string) {
  draggingFileTabId.value = tabId;
  event.dataTransfer?.setData('text/plain', tabId);
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
}

function enterTabDrag(event: DragEvent) {
  event.preventDefault();
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
}

function dropTab(event: DragEvent, tabId: string) {
  event.preventDefault();
  const fromTabId = draggingFileTabId.value || event.dataTransfer?.getData('text/plain') || '';
  draggingFileTabId.value = '';
  if (fromTabId && fromTabId !== tabId) emit('reorderFile', fromTabId, tabId);
}
</script>

<template>
  <aside ref="panelElement" v-if="visible && contentReady" class="workspace-panel" :class="{ 'is-expanded': expanded, 'is-file-preview': !workspace }">
    <div class="workspace-resizer" aria-hidden="true" @pointerdown="emit('resizeStart', $event)" />
    <header class="workspace-panel-header">
      <div v-if="workspaceTabs.length || fileTabs.length" class="file-preview-tabs">
        <button
          v-for="tab in workspaceTabs"
          :key="tab.tabId"
          type="button"
          class="file-preview-tab workspace-preview-tab"
          :class="{ 'is-active': activeKind === 'workspace' && tab.tabId === activeWorkspaceTabId }"
          :style="{ order: tab.order }"
          @click="emit('selectWorkspace', tab.tabId)"
        >
          <Icon class="workspace-preview-tab-icon" icon="solar:widget-5-linear" />
          <span class="file-preview-tab-name">{{ tab.title }}</span>
          <span
            role="button"
            tabindex="0"
            class="file-preview-tab-close"
            aria-label="关闭工作台"
            @click.stop="emit('closeWorkspace', tab.tabId)"
            @keydown.enter.stop.prevent="emit('closeWorkspace', tab.tabId)"
            @keydown.space.stop.prevent="emit('closeWorkspace', tab.tabId)"
          >
            <Icon icon="lucide:x" />
          </span>
        </button>
        <button
          v-for="tab in fileTabs"
          :key="tab.tabId"
          type="button"
          class="file-preview-tab"
          :class="{
            'is-active': activeKind === 'file' && tab.tabId === activeFileTabId,
            'is-dragging': draggingFileTabId === tab.tabId
          }"
          :style="{ order: tab.order }"
          draggable="true"
          @click="emit('selectFile', tab.tabId)"
          @dragstart="startTabDrag($event, tab.tabId)"
          @dragend="draggingFileTabId = ''"
          @dragover="enterTabDrag"
          @drop="dropTab($event, tab.tabId)"
        >
          <span class="file-preview-type" :class="'is-' + fileVisualKind(tab.name, tab.language)">{{ fileVisualLabel(fileVisualKind(tab.name, tab.language), tab.name, tab.language) }}</span>
          <span class="file-preview-tab-name">{{ tab.name }}</span>
          <span
            role="button"
            tabindex="0"
            class="file-preview-tab-close"
            aria-label="关闭文件"
            @click.stop="emit('closeFile', tab.tabId)"
            @keydown.enter.stop.prevent="emit('closeFile', tab.tabId)"
            @keydown.space.stop.prevent="emit('closeFile', tab.tabId)"
          >
            <Icon icon="lucide:x" />
          </span>
        </button>
      </div>
      <button
        v-if="activeKind"
        ref="addButtonElement"
        type="button"
        class="workspace-panel-add-button"
        aria-label="打开右侧内容"
        title="打开右侧内容"
        @click="togglePicker"
      >
        <Icon icon="lucide:plus" />
      </button>
      <div
        v-if="pickerOpen"
        ref="pickerElement"
        class="workspace-panel-picker-popover"
        :style="{ left: `${pickerPosition.left}px`, top: `${pickerPosition.top}px`, width: `${pickerPosition.width}px` }"
      >
        <div class="workspace-panel-picker">
          <template v-if="pickerSection === 'root'">
            <button
              v-if="hasWorkspace"
              type="button"
              class="workspace-panel-picker-item"
              @click="pickerSection = 'workspace'"
            >
              <Icon icon="solar:widget-5-linear" />
              <span>工作台</span>
              <Icon class="workspace-panel-picker-next" icon="lucide:chevron-right" />
            </button>
            <button type="button" class="workspace-panel-picker-item" @click="openFiles(true)">
              <Icon icon="lucide:folders" />
              <span>文件</span>
            </button>
          </template>
          <template v-else>
            <div class="workspace-panel-picker-header">
              <button type="button" class="workspace-panel-picker-back" aria-label="返回" title="返回" @click="pickerSection = 'root'">
                <Icon icon="lucide:chevron-left" />
              </button>
              <span>选择工作台</span>
            </div>
            <div class="workspace-panel-picker-list">
            <button
              v-for="item in availableWorkspaces"
              :key="item.url"
              type="button"
              class="workspace-panel-picker-item"
              :class="{ 'is-active': item.url === activeWorkspaceUrl }"
              @click="openWorkspaceChoice(item)"
            >
              <Icon icon="solar:widget-5-linear" />
              <span>{{ item.title }}</span>
              <Icon v-if="item.url === activeWorkspaceUrl" class="workspace-panel-picker-check" icon="lucide:check" />
            </button>
            </div>
          </template>
        </div>
      </div>
    </header>
    <div class="workspace-panel-actions">
      <n-button
        quaternary
        circle
        class="workspace-action-button"
        aria-label="刷新右侧面板"
        title="刷新右侧面板"
        @click="emit('reload')"
      >
        <template #icon><Icon icon="solar:refresh-linear" /></template>
      </n-button>
      <n-button
        quaternary
        circle
        class="workspace-action-button workspace-maximize-button"
        :aria-label="expanded ? '恢复右侧面板' : '放大右侧面板'"
        :title="expanded ? '恢复右侧面板' : '放大右侧面板'"
        @click="emit('toggleExpanded')"
      >
        <template #icon>
          <Icon :icon="expanded ? 'solar:minimize-square-3-linear' : 'solar:maximize-square-3-linear'" />
        </template>
      </n-button>
      <n-button
        quaternary
        circle
        class="workspace-action-button workspace-collapse-button"
        aria-label="收起右侧面板"
        title="收起右侧面板"
        @click="emit('collapse')"
      >
        <template #icon><Icon icon="lucide:panel-right" /></template>
      </n-button>
    </div>

    <div class="workspace-panel-content">
      <div v-if="!activeKind" class="workspace-panel-launcher">
        <div class="workspace-panel-launcher-items">
          <button
            v-if="hasWorkspace"
            type="button"
            class="workspace-panel-launcher-item"
            @click="openFirstWorkspace"
          >
            <Icon icon="solar:widget-5-linear" />
            <span>工作台</span>
          </button>
          <button type="button" class="workspace-panel-launcher-item" @click="openFiles(false)">
            <Icon icon="lucide:folders" />
            <span>文件</span>
          </button>
        </div>
      </div>

      <iframe
        v-else-if="workspace"
        :key="`${conversationId}:${workspace.url}:${reloadVersion}`"
        :src="workspace.url"
        :title="workspace.title"
      />

      <section v-else class="file-preview-panel">
        <div class="file-preview-pathbar">
          <div class="file-preview-breadcrumb">
            <template v-for="(part, index) in fileBreadcrumbs" :key="part + index">
              <button
                v-if="index < fileBreadcrumbs.length - 1"
                type="button"
                class="file-preview-breadcrumb-part"
                @click="openBreadcrumbDirectory(index, $event)"
              >
                {{ part }}
              </button>
              <span v-else class="file-preview-breadcrumb-current">{{ part }}</span>
              <Icon v-if="index < fileBreadcrumbs.length - 1" icon="solar:alt-arrow-right-linear" />
            </template>
          </div>
          <n-button
            quaternary
            circle
            class="file-preview-tree-toggle"
            :class="{ active: fileTreeVisible }"
            :aria-pressed="fileTreeVisible"
            aria-label="显示或收起文件树"
            title="显示或收起文件树"
            @click="toggleFileTree"
          >
            <template #icon><Icon icon="lucide:folders" /></template>
          </n-button>
        </div>

        <div
          v-if="breadcrumbPopupOpen"
          ref="breadcrumbPopupElement"
          class="file-preview-breadcrumb-popup"
          :style="{
            left: `${breadcrumbPopupPosition.left}px`,
            top: `${breadcrumbPopupPosition.top}px`,
            width: `${breadcrumbPopupPosition.width}px`
          }"
        >
          <div v-if="fileTreeLoading && !fileTreeLoaded" class="workspace-tree-status">
            <Icon class="workspace-tree-spinner" icon="solar:refresh-linear" />
          </div>
          <div v-else-if="fileTreeError" class="workspace-tree-status is-error">{{ fileTreeError }}</div>
          <div v-else-if="fileTreeLoaded && !breadcrumbPopupRoot" class="workspace-tree-status">目录不存在</div>
          <ul v-else class="workspace-tree file-preview-breadcrumb-list">
            <FileTreeNode
              v-if="breadcrumbPopupRoot"
              :node="breadcrumbPopupRoot"
              :depth="0"
              :expanded-paths="breadcrumbPopupExpandedPaths"
              :active-path="activeTreePath"
              @toggle="toggleBreadcrumbPopupDirectory"
              @open="openTreeFile"
            />
          </ul>
        </div>

        <div class="file-preview-body">
          <div class="file-preview-main">
            <n-spin :show="fileLoading">
              <div v-if="fileError" class="file-preview-error">{{ fileError }}</div>
              <template v-else-if="filePreview">
                <template v-if="filePreview.kind !== 'loading'">
                  <div v-if="filePreview.kind === 'image'" class="file-preview-image-view">
                    <div class="file-preview-image-frame">
                      <img v-if="filePreview.objectUrl" :src="filePreview.objectUrl" :alt="filePreview.name" />
                    </div>
                  </div>
                  <div v-else-if="filePreview.kind === 'unsupported'" class="file-preview-empty">
                    <Icon icon="solar:file-corrupted-linear" />
                    <span>此文件暂不支持预览</span>
                  </div>
                  <div v-else class="file-preview-code-view">
                    <div
                      v-for="line in fileLines"
                      :key="line.number"
                      class="file-preview-line"
                    >
                      <span class="file-preview-line-number">{{ line.number }}</span>
                      <code class="file-preview-line-code hljs" v-html="line.html" />
                    </div>
                  </div>
                </template>
              </template>
              <div v-else class="file-preview-empty">
                <Icon icon="solar:document-text-linear" />
                <span>选择文件进行预览</span>
              </div>
            </n-spin>
          </div>

          <aside v-if="fileTreeVisible" class="workspace-tree-panel" aria-label="工作区文件">
            <div v-if="fileTreeLoading && !fileTreeLoaded" class="workspace-tree-status">
              <Icon class="workspace-tree-spinner" icon="solar:refresh-linear" />
              <span>正在读取文件</span>
            </div>
            <div v-else-if="fileTreeError" class="workspace-tree-status is-error">{{ fileTreeError }}</div>
            <div v-else-if="fileTreeLoaded && !fileTree.length" class="workspace-tree-status">工作区暂无文件</div>
            <ul v-else class="workspace-tree">
              <FileTreeNode
                v-for="node in fileTree"
                :key="node.path"
                :node="node"
                :depth="0"
                :expanded-paths="fileTreeExpandedPaths"
                :active-path="activeTreePath"
                @toggle="path => emit('toggleFileTreeDirectory', path)"
                @open="openTreeFile"
              />
            </ul>
          </aside>
        </div>
      </section>
    </div>
  </aside>
</template>
