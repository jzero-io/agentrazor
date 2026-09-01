<script setup lang="ts">
import { Icon } from '@iconify/vue';
import type { ProcessDisplayItem } from '../../../utils/processDisplay';
import MarkdownBlock from './MarkdownBlock.vue';

defineProps<{
  display: ProcessDisplayItem;
  displayWorkspaceProcessPath: (filePath: string) => string;
  normalizeWorkspaceFilePath: (href: string) => string;
}>();

const emit = defineEmits<{
  openWorkspaceFile: [path: string];
  error: [message: string];
}>();
</script>

<template>
  <details
    v-if="display.item.type === 'webSearchGroup'"
    class="process-entry web-search-group-entry"
    :class="{ 'process-entry-live': display.live }"
  >
    <summary>
      <Icon :icon="display.icon" />
      <span class="entry-label">{{ display.label }}</span>
    </summary>
    <div class="process-detail-list web-search-group-list">
      <div v-for="query in display.item.searchQueries || []" :key="query" class="process-detail-list-item web-search-group-item">
        <Icon icon="solar:global-linear" />
        <span>已搜索网页： {{ query }}</span>
      </div>
    </div>
  </details>

  <details
    v-else-if="display.item.type === 'skillReadGroup'"
    class="process-entry skill-read-entry"
    :class="{ 'process-entry-live': display.live }"
  >
    <summary>
      <Icon :icon="display.icon" />
      <span class="entry-label">{{ display.label }}</span>
    </summary>
    <div class="process-detail-list skill-read-list">
      <div v-for="file in display.item.skillFiles || []" :key="file" class="process-detail-list-item skill-read-list-item">
        <Icon icon="solar:document-text-linear" />
        <span>{{ file }}</span>
      </div>
    </div>
  </details>

  <MarkdownBlock
    v-else-if="display.item.type === 'agentMessage' && display.item.text"
    class="process-commentary"
    :content="display.item.text"
    :streaming="display.live"
    :normalize-workspace-file-path="normalizeWorkspaceFilePath"
    @open-workspace-file="path => emit('openWorkspaceFile', path)"
    @error="message => emit('error', message)"
  />

  <template v-else-if="display.item.type === 'fileChange'">
    <div
      v-if="!display.detail.split('\n').map(path => path.trim()).filter(Boolean).length"
      class="process-entry process-entry-title file-change-entry"
      :class="[display.className, { 'process-entry-live': display.live }]"
    >
      <Icon :icon="display.icon" />
      <span class="entry-label">{{ display.label }}</span>
    </div>
    <details
      v-else
      class="process-entry file-change-entry"
      :class="[display.className, { 'process-entry-live': display.live }]"
    >
      <summary>
        <Icon :icon="display.icon" />
        <span class="entry-label">{{ display.label }}</span>
      </summary>
      <div class="file-change-list">
        <div
          v-for="filePath in display.detail.split('\n').map(path => path.trim()).filter(Boolean)"
          :key="filePath"
          class="file-change-row"
          :title="filePath"
        >
          <Icon icon="solar:document-text-linear" />
          <span>{{ displayWorkspaceProcessPath(filePath) }}</span>
        </div>
      </div>
    </details>
  </template>

  <div
    v-else-if="!display.expandable"
    class="process-entry process-entry-title"
    :class="[display.className, { 'process-entry-live': display.live }]"
  >
    <Icon :icon="display.icon" />
    <span class="entry-label">{{ display.label }}</span>
  </div>

  <details
    v-else
    class="process-entry"
    :class="[display.className, { 'process-entry-live': display.live }]"
  >
    <summary>
      <Icon :icon="display.icon" />
      <span class="entry-label">{{ display.label }}</span>
    </summary>
    <div v-if="display.item.type === 'webSearch'" class="search-result">{{ display.detail }}</div>
    <pre v-else class="command-output">{{ display.detail }}</pre>
  </details>
</template>
