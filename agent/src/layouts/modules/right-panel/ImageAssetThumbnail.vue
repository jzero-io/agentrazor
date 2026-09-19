<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue';
import { Icon } from '@iconify/vue';
import type { ImageAsset } from '../../../service/api';

const props = defineProps<{
  asset: ImageAsset;
  loadImage: (name: string) => Promise<Blob>;
}>();

const emit = defineEmits<{
  open: [name: string];
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
  () => props.asset.name,
  async name => {
    const version = ++loadVersion;
    releaseObjectUrl();
    source.value = '';
    failed.value = false;
    loading.value = true;
    try {
      const blob = await props.loadImage(name);
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
    @click="emit('open', asset.name)"
  >
    <span class="image-asset-thumbnail-preview">
      <img v-if="source && !failed" :src="source" :alt="asset.name" />
      <Icon v-else-if="loading" class="image-asset-thumbnail-spinner" icon="solar:refresh-linear" />
      <Icon v-else icon="solar:gallery-linear" />
    </span>
  </button>
</template>
