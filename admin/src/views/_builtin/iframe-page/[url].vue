<script setup lang="ts">
import { computed, onBeforeMount, onBeforeUnmount, ref, watch } from 'vue';
import { request } from '@/service/request';
import { useAppStore } from '@/store/modules/app';
import { useThemeStore } from '@/store/modules/theme';

interface Props {
  url: string;
}

type PluginMethod = 'get' | 'post' | 'put' | 'patch' | 'delete';

interface PluginApiRequest {
  type: 'agentrazor:plugin-api-request';
  requestId: string;
  method?: string;
  path: string;
  body?: unknown;
  params?: Record<string, unknown>;
}

interface PluginHostContext {
  type: 'agentrazor:host-context';
  theme: 'light' | 'dark';
  locale: App.I18n.LangType;
}

const props = defineProps<Props>();
const iframeRef = ref<HTMLIFrameElement | null>(null);
const appStore = useAppStore();
const themeStore = useThemeStore();

// Only same-origin plugin static pages get the host API bridge. Regular iframe
// pages continue to work as plain embeds.
const apiPrefix = computed(() => {
  const match = /^\/plugins\/([a-z0-9_-]+)\/admin\/?$/i.exec(props.url);
  return match ? `/api/v1/manage/plugin/${match[1]}` : '';
});

function validPluginRequest(value: unknown): value is PluginApiRequest {
  if (!value || typeof value !== 'object') return false;
  const message = value as Partial<PluginApiRequest>;
  return (
    message.type === 'agentrazor:plugin-api-request' &&
    typeof message.requestId === 'string' &&
    message.requestId.length > 0 &&
    typeof message.path === 'string'
  );
}

function allowedMethod(method?: string): PluginMethod | null {
  const normalized = (method || 'get').toLowerCase();
  return ['get', 'post', 'put', 'patch', 'delete'].includes(normalized) ? (normalized as PluginMethod) : null;
}

function postPluginResponse(target: Window, requestId: string, result: { data?: unknown; error?: string }) {
  target.postMessage({ type: 'agentrazor:plugin-api-response', requestId, ...result }, window.location.origin);
}

function postPluginHostContext() {
  const frame = iframeRef.value;
  if (!apiPrefix.value || !frame?.contentWindow) return;

  const context: PluginHostContext = {
    type: 'agentrazor:host-context',
    theme: themeStore.darkMode ? 'dark' : 'light',
    locale: appStore.locale
  };
  frame.contentWindow.postMessage(context, window.location.origin);
}

async function handlePluginApiRequest(event: MessageEvent<unknown>) {
  const frame = iframeRef.value;
  if (
    !apiPrefix.value ||
    !frame?.contentWindow ||
    event.origin !== window.location.origin ||
    event.source !== frame.contentWindow
  ) {
    return;
  }
  if (!validPluginRequest(event.data)) return;

  const message = event.data;
  const method = allowedMethod(message.method);
  const permittedPath = message.path === apiPrefix.value || message.path.startsWith(`${apiPrefix.value}/`);
  if (!method || !permittedPath) {
    postPluginResponse(frame.contentWindow, message.requestId, { error: '插件请求不在允许范围内' });
    return;
  }

  const { data, error } = await request<unknown>({
    url: message.path,
    method,
    params: method === 'get' || method === 'delete' ? message.params : undefined,
    data: method === 'get' || method === 'delete' ? undefined : message.body
  });
  postPluginResponse(frame.contentWindow, message.requestId, {
    data,
    error: error ? '请求失败，请检查权限或稍后重试' : undefined
  });
}

onBeforeMount(() => window.addEventListener('message', handlePluginApiRequest));
onBeforeUnmount(() => window.removeEventListener('message', handlePluginApiRequest));

watch([() => themeStore.darkMode, () => appStore.locale], postPluginHostContext, { immediate: true });
</script>

<template>
  <div class="h-full">
    <iframe
      id="iframePage"
      ref="iframeRef"
      class="size-full border-0"
      :src="url"
      @load="postPluginHostContext"
    ></iframe>
  </div>
</template>

<style scoped></style>
