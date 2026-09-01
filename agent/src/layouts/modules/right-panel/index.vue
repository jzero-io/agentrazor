<script setup lang="ts">
import { Icon } from '@iconify/vue';
import { NButton, NSpin } from 'naive-ui';
import type { FilePreview, WorkspaceDescriptor } from '../../../hooks/business/useWorkspacePanel';

defineOptions({
  name: 'RightPanel'
});

defineProps<{
  visible: boolean;
  contentReady: boolean;
  expanded: boolean;
  conversationId: string;
  workspace: WorkspaceDescriptor | null;
  filePreview: FilePreview | null;
  fileLoading: boolean;
  fileError: string;
  title: string;
  reloadVersion: number;
  fileBadge: string;
  fileBreadcrumbs: string[];
  fileLines: Array<{ number: number; html: string }>;
}>();

const emit = defineEmits<{
  resizeStart: [event: PointerEvent];
  reload: [];
  toggleExpanded: [];
  collapse: [];
  closeFile: [];
}>();
</script>

<template>
  <aside v-if="visible && contentReady" class="workspace-panel" :class="{ 'is-expanded': expanded, 'is-file-preview': !workspace }">
    <div class="workspace-resizer" aria-hidden="true" @pointerdown="emit('resizeStart', $event)" />
    <header v-if="workspace" class="workspace-panel-header">
      <strong>{{ title }}</strong>
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

    <iframe
      v-if="workspace"
      :key="`${conversationId}:${workspace.url}:${reloadVersion}`"
      :src="workspace.url"
      :title="workspace.title"
    />

    <section v-else class="file-preview-panel">
      <n-spin :show="fileLoading">
        <div v-if="fileError" class="file-preview-error">{{ fileError }}</div>
        <template v-else-if="filePreview">
          <div class="file-preview-tabs">
            <div class="file-preview-tab is-active">
              <span class="file-preview-type">{{ fileBadge }}</span>
              <span class="file-preview-tab-name">{{ filePreview.name }}</span>
              <button type="button" class="file-preview-tab-close" aria-label="关闭文件" @click="emit('closeFile')">
                <Icon icon="solar:close-circle-linear" />
              </button>
            </div>
          </div>
          <div class="file-preview-breadcrumb">
            <template v-for="(part, index) in fileBreadcrumbs" :key="`${part}-${index}`">
              <span>{{ part }}</span>
              <Icon v-if="index < fileBreadcrumbs.length - 1" icon="solar:alt-arrow-right-linear" />
            </template>
          </div>
          <div class="file-preview-code-view">
            <div
              v-for="line in fileLines"
              :key="line.number"
              class="file-preview-line"
              :class="{ 'is-current': line.number === 1 }"
            >
              <span class="file-preview-line-number">{{ line.number }}</span>
              <code class="file-preview-line-code hljs" v-html="line.html" />
            </div>
          </div>
        </template>
      </n-spin>
    </section>
  </aside>
</template>
