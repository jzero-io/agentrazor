<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue';
import { Icon } from '@iconify/vue';
import type { WorkspaceTreeNode } from '../../../hooks/business/useWorkspaceFileTree';

const props = defineProps<{
  asset: WorkspaceTreeNode;
  loadImage: (relativePath: string) => Promise<Blob>;
}>();

const emit = defineEmits<{
  open: [path: string];
}>();

const source = ref('');
const loading = ref(false);
const failed = ref(false);
let objectUrl = '';
let loadVersion = 0;

function releaseObjectUrl() {
  if (!objectUrl) return;
  URL.revokeObjectURL(objectUrl);
  objectUrl = '';
}

watch(
  () => props.asset.path,
  async path => {
    const version = ++loadVersion;
    releaseObjectUrl();
    source.value = '';
    failed.value = false;
    loading.value = true;
    try {
      const blob = await props.loadImage(path);
      if (version !== loadVersion) return;
      objectUrl = URL.createObjectURL(blob);
      source.value = objectUrl;
    } catch {
      if (version === loadVersion) failed.value = true;
    } finally {
      if (version === loadVersion) loading.value = false;
    }
  },
  { immediate: true }
);

onBeforeUnmount(() => {
  loadVersion += 1;
  releaseObjectUrl();
});
</script>

<template>
  <button
    type="button"
    class="image-asset-thumbnail"
    :title="asset.name"
    @click="emit('open', asset.path)"
  >
    <span class="image-asset-thumbnail-preview">
      <img v-if="source && !failed" :src="source" :alt="asset.name" />
      <Icon v-else-if="loading" class="image-asset-thumbnail-spinner" icon="solar:refresh-linear" />
      <Icon v-else icon="solar:gallery-linear" />
    </span>
  </button>
</template>
