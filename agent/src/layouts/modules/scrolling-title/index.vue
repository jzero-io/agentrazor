<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, onUpdated, ref, watch } from 'vue';

defineOptions({
  name: 'ScrollingTitle'
});

const props = defineProps<{ text: string }>();

const root = ref<HTMLElement | null>(null);
const overflowing = ref(false);
const distance = ref(0);
let observer: ResizeObserver | undefined;

function measure() {
  const container = root.value;
  const content = container?.querySelector<HTMLElement>('.scrolling-title-text');
  if (!container || !content) return;
  const overflow = Math.ceil(content.scrollWidth - container.clientWidth);
  overflowing.value = overflow > 2;
  distance.value = Math.max(0, overflow);
}

onMounted(() => {
  nextTick(measure);
  if ('ResizeObserver' in window && root.value) {
    observer = new ResizeObserver(measure);
    observer.observe(root.value);
  }
});
onUpdated(() => nextTick(measure));
onBeforeUnmount(() => observer?.disconnect());
watch(() => props.text, () => nextTick(measure));
</script>

<template>
  <span
    ref="root"
    class="scrolling-title"
    :class="{ 'is-overflowing': overflowing }"
    :style="{ '--scroll-distance': `${distance}px`, '--scroll-duration': `${Math.max(2.4, distance * 0.024).toFixed(2)}s` }"
  >
    <span class="scrolling-title-text">{{ text }}</span>
  </span>
</template>
