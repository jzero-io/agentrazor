<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue';
import { NImage, NSpin } from 'naive-ui';

const props = defineProps<{
  name: string;
  path: string;
  dataUrl?: string;
  loadImage: (path: string) => Promise<Blob>;
}>();

const source = ref('');
const failed = ref(false);
const loading = ref(false);
let objectUrl = '';
let loadVersion = 0;

function releaseObjectUrl() {
  if (!objectUrl) return;
  URL.revokeObjectURL(objectUrl);
  objectUrl = '';
}

watch(
  () => [props.path, props.dataUrl] as const,
  async ([path, dataUrl]) => {
    const version = ++loadVersion;
    failed.value = false;
    releaseObjectUrl();

    const inlineSource = typeof dataUrl === 'string' ? dataUrl.trim() : '';
    if (inlineSource) {
      source.value = inlineSource;
      loading.value = false;
      return;
    }

    source.value = '';
    if (!path) {
      failed.value = true;
      return;
    }

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
  <div class="generated-image-asset">
    <n-spin v-if="loading" size="small" />
    <n-image
      v-else-if="source && !failed"
      class="generated-image"
      :src="source"
      :alt="name"
      object-fit="contain"
      lazy
      @error="failed = true"
    />
    <div v-else class="generated-image-error">图片资产暂时无法读取</div>
  </div>
</template>
