<script setup lang="ts">
import { computed } from 'vue';
import { Icon } from '@iconify/vue';
import type { WorkspaceTreeNode } from '../../../hooks/business/useWorkspaceFileTree';
import { fileVisualIcon, fileVisualKind } from './filePresentation';

defineOptions({
  name: 'FileTreeNode'
});

const props = defineProps<{
  node: WorkspaceTreeNode;
  depth: number;
  expandedPaths: Set<string>;
  activePath: string;
}>();

const fileKind = computed(() => fileVisualKind(props.node.name));
const visualKind = computed(() => props.node.type === 'directory' ? 'folder' : fileKind.value);
const entryIcon = computed(() => props.node.type === 'directory'
  ? (props.expandedPaths.has(props.node.path) ? 'solar:folder-open-linear' : 'solar:folder-linear')
  : fileVisualIcon(fileKind.value));

const emit = defineEmits<{
  toggle: [path: string];
  open: [path: string];
}>();

function activate() {
  if (props.node.type === 'directory') emit('toggle', props.node.path);
  else emit('open', props.node.path);
}
</script>

<template>
  <li class="workspace-tree-node">
    <button
      type="button"
      class="workspace-tree-row"
      :class="{ 'is-active': node.type === 'file' && node.path === activePath }"
      :style="{ paddingLeft: `${6 + depth * 16}px` }"
      :title="node.path"
      @click="activate"
    >
      <Icon
        v-if="node.type === 'directory'"
        class="workspace-tree-chevron"
        :icon="expandedPaths.has(node.path) ? 'solar:alt-arrow-down-linear' : 'solar:alt-arrow-right-linear'"
      />
      <span v-else class="workspace-tree-chevron is-placeholder" />
      <Icon
        :class="['workspace-tree-entry-icon', 'is-' + visualKind]"
        :icon="entryIcon"
      />
      <span class="workspace-tree-name">{{ node.name }}</span>
    </button>
    <ul v-if="node.type === 'directory' && expandedPaths.has(node.path)" class="workspace-tree-children">
      <FileTreeNode
        v-for="child in node.children"
        :key="child.path"
        :node="child"
        :depth="depth + 1"
        :expanded-paths="expandedPaths"
        :active-path="activePath"
        @toggle="path => emit('toggle', path)"
        @open="path => emit('open', path)"
      />
    </ul>
  </li>
</template>
