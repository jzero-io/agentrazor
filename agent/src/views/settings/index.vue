<script setup lang="ts">
defineOptions({
  name: 'SettingsView',
  inheritAttrs: false
});

import { Icon } from '@iconify/vue';
import { NButton, NInput, NModal, NSpin } from 'naive-ui';
import { ref, watch } from 'vue';
import { useConfirmDialog } from '../../hooks/system/useConfirmDialog';
import { apiKeyApi } from '../../service/api';
import type { AgentApiKey, Conversation } from '../../service/api';
import { writeClipboardText } from '../../utils/clipboard';

interface AppearanceOption {
  key: 'system' | 'light' | 'dark';
  label: string;
  icon: string;
}

interface ArchiveSection {
  title: string;
  groupId?: string;
  items: Conversation[];
}

const props = defineProps<{
  visible: boolean;
  navExpanded: boolean;
  section: 'appearance' | 'api-keys' | 'archives';
  appearance: 'system' | 'light' | 'dark';
  appearanceOptions: AppearanceOption[];
  archivedConversations: Conversation[];
  archivedSections: ArchiveSection[];
  archiveQuery: string;
  loadingList: boolean;
  displayConversationTitle: (item: Conversation) => string;
  formatConversationDate: (value: string) => string;
}>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
  'update:navExpanded': [value: boolean];
  'update:section': [value: 'appearance' | 'api-keys' | 'archives'];
  'update:archiveQuery': [value: string];
  setAppearance: [value: 'system' | 'light' | 'dark'];
  deleteAllArchived: [];
  deleteGroupArchived: [section: ArchiveSection];
  deleteArchived: [item: Conversation];
  restoreArchived: [item: Conversation];
  resizeStart: [event: PointerEvent];
}>();

const apiKeys = ref<AgentApiKey[]>([]);
const loadingKeys = ref(false);
const creatingKey = ref(false);
const deletingKeyId = ref('');
const createdKey = ref('');
const keyError = ref('');
const copiedKey = ref(false);
const keyConfirmDialog = useConfirmDialog();
const keyConfirmVisible = keyConfirmDialog.visible;
const keyConfirmTitle = keyConfirmDialog.title;
const keyConfirmContent = keyConfirmDialog.content;
const keyConfirmPositiveText = keyConfirmDialog.positiveText;
const keyConfirmLoading = keyConfirmDialog.loading;

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : '操作失败，请稍后重试';
}

async function loadApiKeys() {
  loadingKeys.value = true;
  keyError.value = '';
  try {
    const response = await apiKeyApi.list();
    apiKeys.value = response.keys;
  } catch (error) {
    keyError.value = errorMessage(error);
  } finally {
    loadingKeys.value = false;
  }
}

async function createApiKey() {
  creatingKey.value = true;
  keyError.value = '';
  try {
    const response = await apiKeyApi.create();
    createdKey.value = response.key;
    copiedKey.value = false;
  } catch (error) {
    keyError.value = errorMessage(error);
  } finally {
    creatingKey.value = false;
  }
}

function deleteApiKey(item: AgentApiKey) {
  keyConfirmDialog.open(
    '删除密钥',
    `确定删除密钥 ${item.keyHint} 吗？使用它的客户端将立即无法访问。`,
    '删除',
    async () => {
      deletingKeyId.value = item.id;
      keyError.value = '';
      try {
        await apiKeyApi.delete(item.id);
        await loadApiKeys();
      } catch (error) {
        keyError.value = errorMessage(error);
      } finally {
        deletingKeyId.value = '';
      }
    }
  );
}

async function acknowledgeCreatedKey() {
  createdKey.value = '';
  copiedKey.value = false;
  await loadApiKeys();
}

async function copyCreatedKey() {
  try {
    await writeClipboardText(createdKey.value);
    copiedKey.value = true;
  } catch (error) {
    keyError.value = errorMessage(error);
  }
}

function openArchives() {
  emit('update:section', 'archives');
}

watch(
  () => [props.visible, props.section] as const,
  ([visible, section]) => {
    if (visible && section === 'api-keys') void loadApiKeys();
  },
  { immediate: true }
);
</script>
<template>
  <section v-if="visible" class="settings-shell" :class="{ 'settings-nav-collapsed': !navExpanded }">
    <aside class="settings-sidebar">
      <button class="settings-back" type="button" @click="emit('update:visible', false)">
        <Icon icon="solar:arrow-left-linear" />
        <span>返回应用</span>
      </button>
      <div class="settings-sidebar-title">设置</div>
      <div class="sidebar-resize-handle settings-sidebar-resize-handle" role="separator" aria-orientation="vertical" title="拖动调整侧边栏宽度" @pointerdown="emit('resizeStart', $event)" />
      <nav aria-label="设置导航">
        <button :class="{ active: section === 'appearance' }" @click="emit('update:section', 'appearance')">
          <Icon icon="solar:sun-2-linear" />
          <span>外观</span>
        </button>
        <button :class="{ active: section === 'api-keys' }" @click="emit('update:section', 'api-keys')">
          <Icon icon="solar:key-minimalistic-square-linear" />
          <span>密钥管理</span>
        </button>
        <button :class="{ active: section === 'archives' }" @click="openArchives">
          <Icon icon="solar:archive-linear" />
          <span>已归档对话</span>
        </button>
      </nav>
    </aside>

    <div
      v-if="navExpanded"
      class="settings-backdrop"
      @click="emit('update:navExpanded', false)"
    />

    <main class="settings-content">
      <section v-if="section === 'appearance'" class="settings-content-inner appearance-page">
        <header class="settings-page-header">
          <button
            class="settings-menu-button"
            type="button"
            aria-label="打开设置菜单"
            @click="emit('update:navExpanded', true)"
          >
            <Icon icon="lucide:panel-left" />
          </button>
          <div>
            <h1>外观</h1>
            <p>选择 AgentRazor 的显示方式。</p>
          </div>
        </header>
        <div class="appearance-options" role="radiogroup" aria-label="外观">
          <button
            v-for="option in appearanceOptions"
            :key="option.key"
            type="button"
            role="radio"
            :aria-checked="appearance === option.key"
            :class="{ active: appearance === option.key }"
            @click="emit('setAppearance', option.key)"
          >
            <Icon :icon="option.icon" />
            <span>{{ option.label }}</span>
          </button>
        </div>
      </section>

      <section v-else-if="section === 'archives'" class="settings-content-inner archives-page">
        <header class="settings-page-header archive-page-header">
          <button
            class="settings-menu-button"
            type="button"
            aria-label="打开设置菜单"
            @click="emit('update:navExpanded', true)"
          >
            <Icon icon="lucide:panel-left" />
          </button>
          <div>
            <h1>已归档的对话</h1>
            <p>归档对话不能查看内容，恢复后才会重新出现在主页面。</p>
          </div>
          <n-button v-if="archivedConversations.length" tertiary type="error" @click="emit('deleteAllArchived')">
            全部删除
          </n-button>
        </header>
        <n-input :value="archiveQuery" size="large" clearable placeholder="搜索已归档对话" class="archive-search" @update:value="value => emit('update:archiveQuery', value)">
          <template #prefix><Icon icon="solar:magnifer-linear" /></template>
        </n-input>
        <n-spin :show="loadingList">
          <div v-if="archivedSections.length" class="archive-list archive-page-list">
            <template v-for="archiveSection in archivedSections" :key="archiveSection.title">
              <div class="archive-section-head">
                <h2 class="archive-section-title">{{ archiveSection.title }}</h2>
                <n-button
                  v-if="archiveSection.groupId"
                  text
                  type="error"
                  size="small"
                  @click="emit('deleteGroupArchived', archiveSection)"
                >
                  全部删除
                </n-button>
              </div>
              <div v-for="item in archiveSection.items" :key="item.id" class="archive-item">
                <div class="archive-item-copy">
                  <strong>{{ displayConversationTitle(item) }}</strong>
                  <span>{{ formatConversationDate(item.updatedAt) }}</span>
                </div>
                <n-button quaternary circle type="error" aria-label="删除" @click="emit('deleteArchived', item)">
                  <template #icon><Icon icon="solar:trash-bin-trash-linear" /></template>
                </n-button>
                <n-button secondary @click="emit('restoreArchived', item)">取消归档</n-button>
              </div>
            </template>
          </div>
          <div v-if="!archivedSections.length" class="archive-empty">
            {{ archiveQuery.trim() ? '没有匹配的归档对话' : '暂无已归档对话' }}
          </div>
        </n-spin>
      </section>

      <section v-else class="settings-content-inner api-keys-page">
        <header class="settings-page-header">
          <button
            class="settings-menu-button"
            type="button"
            aria-label="打开设置菜单"
            @click="emit('update:navExpanded', true)"
          >
            <Icon icon="lucide:panel-left" />
          </button>
          <div>
            <h1>密钥管理</h1>
            <p>使用 API 密钥调用 Agent 接口。每个账户最多可以创建三个。</p>
          </div>
          <n-button type="primary" :loading="creatingKey" :disabled="apiKeys.length >= 3 || Boolean(createdKey)" @click="createApiKey">
            生成密钥
          </n-button>
        </header>

        <div v-if="keyError" class="api-key-message api-key-error">{{ keyError }}</div>
        <div v-if="createdKey" class="api-key-created">
          <div>
            <strong>请立即保存这个密钥</strong>
            <span>出于安全考虑，关闭后将无法再次查看完整密钥。</span>
          </div>
          <code>{{ createdKey }}</code>
          <div class="api-key-created-actions">
            <n-button secondary @click="copyCreatedKey">{{ copiedKey ? '已复制' : '复制密钥' }}</n-button>
            <n-button quaternary @click="acknowledgeCreatedKey">我已保存</n-button>
          </div>
        </div>

        <n-spin :show="loadingKeys">
          <div v-if="apiKeys.length" class="api-key-list">
            <div v-for="item in apiKeys" :key="item.id" class="api-key-item">
              <div class="api-key-icon"><Icon icon="solar:key-minimalistic-square-linear" /></div>
              <div class="api-key-copy">
                <code>{{ item.keyHint }}</code>
                <span>创建于 {{ formatConversationDate(item.createdAt) }}</span>
              </div>
              <n-button
                quaternary
                circle
                type="error"
                aria-label="删除密钥"
                :loading="deletingKeyId === item.id"
                @click="deleteApiKey(item)"
              >
                <template #icon><Icon icon="solar:trash-bin-trash-linear" /></template>
              </n-button>
            </div>
          </div>
          <div v-else-if="!loadingKeys" class="api-key-empty">
            <Icon icon="solar:key-minimalistic-square-linear" />
            <strong>还没有 API 密钥</strong>
            <span>生成后可通过 X-API-Key: ar-... 调用 Agent 接口。</span>
          </div>
        </n-spin>
      </section>
    </main>

    <n-modal
      v-model:show="keyConfirmVisible"
      preset="card"
      :bordered="false"
      :mask-closable="!keyConfirmLoading"
      :close-on-esc="!keyConfirmLoading"
      class="confirm-modal"
    >
      <div class="confirm-modal-body">
        <span class="confirm-modal-icon"><Icon icon="solar:trash-bin-trash-linear" /></span>
        <div>
          <h2>{{ keyConfirmTitle }}</h2>
          <p>{{ keyConfirmContent }}</p>
        </div>
      </div>
      <template #footer>
        <div class="modal-actions">
          <n-button :disabled="keyConfirmLoading" @click="keyConfirmDialog.close">取消</n-button>
          <n-button type="error" :loading="keyConfirmLoading" @click="keyConfirmDialog.submit">
            {{ keyConfirmPositiveText }}
          </n-button>
        </div>
      </template>
    </n-modal>
  </section>
</template>
