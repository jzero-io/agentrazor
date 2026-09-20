<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { useAuthStore } from '@/store/modules/auth';
import {
  ADMIN_AUTH_SESSION_CHANGE_EVENT,
  ADMIN_AUTH_STORAGE_KEYS,
  ADMIN_AUTH_STORAGE_PREFIX
} from '@/utils/auth-session';
import { localStg } from '@/utils/storage';

defineOptions({
  name: 'AgentConversations'
});

const SESSION_CHANGED_MESSAGE = 'agentrazor:admin-auth-session-changed';
const SHARED_SESSION_CHANGED_MESSAGE = 'agentrazor:shared-auth-session-changed';
const AGENT_AUTH_READY_MESSAGE = 'agentrazor:agent-auth-ready';
const iframeRef = ref<HTMLIFrameElement | null>(null);
const authStore = useAuthStore();
let adminLogoutStarted = false;

function notifyAgentSessionChanged() {
  iframeRef.value?.contentWindow?.postMessage(
    { type: SESSION_CHANGED_MESSAGE, storagePrefix: ADMIN_AUTH_STORAGE_PREFIX },
    window.location.origin
  );
}

function handleAgentLoad() {
  notifyAgentSessionChanged();
}

function resetAdminSession() {
  if (adminLogoutStarted) return;
  adminLogoutStarted = true;
  authStore.resetStore().catch(() => {
    adminLogoutStarted = false;
  });
}

function handleStorage(event: StorageEvent) {
  if (event.storageArea !== window.localStorage || !event.key) return;
  if (event.key !== ADMIN_AUTH_STORAGE_KEYS[0]) return;

  // Storage events are emitted only in the other same-origin browsing context.
  // If Agent cleared the shared session, make Admin follow the same logout path.
  if (!localStg.get('token')) resetAdminSession();
}

function handleAgentMessage(event: MessageEvent<unknown>) {
  if (event.origin !== window.location.origin || event.source !== iframeRef.value?.contentWindow) return;
  if (!event.data || typeof event.data !== 'object') return;

  const message = event.data as { type?: unknown; authenticated?: unknown };
  if (message.type === AGENT_AUTH_READY_MESSAGE) {
    notifyAgentSessionChanged();
    return;
  }
  if (message.type !== SHARED_SESSION_CHANGED_MESSAGE || typeof message.authenticated !== 'boolean') return;
  if (!message.authenticated && !localStg.get('token')) resetAdminSession();
}

onMounted(() => {
  window.addEventListener(ADMIN_AUTH_SESSION_CHANGE_EVENT, notifyAgentSessionChanged);
  window.addEventListener('message', handleAgentMessage);
  window.addEventListener('storage', handleStorage);
});

onBeforeUnmount(() => {
  window.removeEventListener(ADMIN_AUTH_SESSION_CHANGE_EVENT, notifyAgentSessionChanged);
  window.removeEventListener('message', handleAgentMessage);
  window.removeEventListener('storage', handleStorage);
});
</script>

<template>
  <div class="h-full min-h-0 overflow-hidden rounded-8px bg-layout">
    <iframe
      ref="iframeRef"
      src="/agent-app/"
      class="size-full border-0"
      title="AgentRazor"
      referrerpolicy="same-origin"
      allow="clipboard-read; clipboard-write"
      @load="handleAgentLoad"
    ></iframe>
  </div>
</template>

<style scoped></style>
