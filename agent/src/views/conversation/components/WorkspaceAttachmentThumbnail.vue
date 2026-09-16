<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue';

const props = defineProps<{
  name: string;
  path: string;
  previewUrl?: string;
  loadWorkspaceImage: (path: string) => Promise<Blob>;
}>();

const source = ref('');
const failed = ref(false);
const expanded = ref(false);
let objectUrl = '';
let loadVersion = 0;

function releaseObjectUrl() {
  if (!objectUrl) return;
  URL.revokeObjectURL(objectUrl);
  objectUrl = '';
}

watch(
  () => [props.path, props.previewUrl] as const,
  async ([path, previewUrl]) => {
    const version = ++loadVersion;
    failed.value = false;
    expanded.value = false;
    releaseObjectUrl();

    if (previewUrl) {
      source.value = previewUrl;
      return;
    }
    source.value = '';
    if (!path) return;

    try {
      const blob = await props.loadWorkspaceImage(path);
      if (version !== loadVersion) return;
      objectUrl = URL.createObjectURL(blob);
      source.value = objectUrl;
    } catch {
      if (version === loadVersion) failed.value = true;
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
  <img
    v-if="source && !failed"
    class="user-message-attachment-preview-image"
    :src="source"
    :alt="name"
    loading="lazy"
    decoding="async"
    title="点击放大图片"
    @click="expanded = true"
    @error="failed = true"
  >
  <Teleport to="body">
    <div
      v-if="expanded && source && !failed"
      class="attachment-image-lightbox"
      role="dialog"
      aria-modal="true"
      :aria-label="`查看图片 ${name}`"
      tabindex="-1"
      @click.self="expanded = false"
      @keydown.esc="expanded = false"
    >
      <button type="button" class="attachment-image-lightbox-close" aria-label="关闭图片预览" @click="expanded = false">×</button>
      <img :src="source" :alt="name" @click.stop>
    </div>
  </Teleport>
</template>
