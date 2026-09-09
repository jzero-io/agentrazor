<script setup lang="ts">
defineOptions({
  name: 'SettingsView',
  inheritAttrs: false
});

import { Icon } from '@iconify/vue';
import { NButton, NInput, NModal, NSelect, NSpin } from 'naive-ui';
import { computed, ref, watch } from 'vue';
import { useConfirmDialog } from '../../hooks/system/useConfirmDialog';
import { apiKeyApi } from '../../service/api';
import type { AgentApiKey, Conversation, TokenQuotaStatus } from '../../service/api';
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

interface ConversationGroupOption {
  id: string;
  name: string;
}

const allArchiveGroups = '__all__';
const ungroupedArchiveGroup = '__ungrouped__';

const props = defineProps<{
  visible: boolean;
  navExpanded: boolean;
  section: 'usage' | 'appearance' | 'api-keys' | 'archives';
  tokenQuota: TokenQuotaStatus | null;
  quotaLoading: boolean;
  appearance: 'system' | 'light' | 'dark';
  appearanceOptions: AppearanceOption[];
  archivedConversations: Conversation[];
  archivedSections: ArchiveSection[];
  conversationGroups: ConversationGroupOption[];
  archiveQuery: string;
  loadingList: boolean;
  displayConversationTitle: (item: Conversation) => string;
  formatConversationDate: (value: string) => string;
}>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
  'update:navExpanded': [value: boolean];
  'update:section': [value: 'usage' | 'appearance' | 'api-keys' | 'archives'];
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
const archiveGroup = ref(allArchiveGroups);
const archiveGroupOptions = computed(() => [
  { label: '所有分组', value: allArchiveGroups },
  { label: '未分组', value: ungroupedArchiveGroup },
  ...props.conversationGroups.map(group => ({ label: group.name, value: group.id }))
]);
const visibleArchiveSections = computed(() => {
  if (archiveGroup.value === allArchiveGroups) return props.archivedSections;
  if (archiveGroup.value === ungroupedArchiveGroup) {
    return props.archivedSections.filter(section => !section.groupId);
  }
  return props.archivedSections.filter(section => section.groupId === archiveGroup.value);
});

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

function percentWidth(value: number) {
  return `${Math.max(0, Math.min(100, value))}%`;
}

function formatResetTime(value?: string) {
  if (!value) return '暂无';
  return new Date(value).toLocaleString('zh-CN', { hour12: false });
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
        <button :class="{ active: section === 'usage' }" @click="emit('update:section', 'usage')">
          <Icon icon="solar:graph-up-linear" />
          <span>使用情况</span>
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
      <section v-if="section === 'usage'" class="settings-content-inner usage-page">
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
            <h1>使用情况</h1>
          </div>
        </header>

        <n-spin :show="quotaLoading">
          <div v-if="tokenQuota && !tokenQuota.enabled" class="quota-disabled-banner">
            <Icon icon="solar:danger-triangle-linear" />
            <div>
              <strong>Agent 使用已禁用</strong>
              <span>当前账户不能发起新的 Turn，如需恢复请联系管理员。</span>
            </div>
          </div>

          <div
            v-if="tokenQuota"
            class="quota-window-grid"
            :class="{ 'quota-window-grid--single': !tokenQuota.fiveHour.limited }"
          >
            <article v-if="tokenQuota.fiveHour.limited" class="quota-window-card">
              <header>
                <div>
                  <span class="quota-window-icon"><Icon icon="solar:clock-circle-linear" /></span>
                  <div>
                    <strong>5 小时使用限额</strong>
                  </div>
                </div>
                <b><span>剩余</span>{{ tokenQuota.fiveHour.remainingPercent }}%</b>
              </header>
              <div class="quota-progress" aria-hidden="true">
                <span :style="{ width: percentWidth(tokenQuota.fiveHour.remainingPercent) }" />
              </div>
              <div class="quota-reset-time">
                <Icon icon="solar:restart-linear" />
                <span>重置时间</span>
                <time v-if="tokenQuota.fiveHour.resetAt" :datetime="tokenQuota.fiveHour.resetAt">
                  {{ formatResetTime(tokenQuota.fiveHour.resetAt) }}
                </time>
                <strong v-else>暂无</strong>
              </div>
            </article>

            <article class="quota-window-card">
              <header>
                <div>
                  <span class="quota-window-icon"><Icon icon="solar:calendar-linear" /></span>
                  <div>
                    <strong>每周使用限额</strong>
                  </div>
                </div>
                <b><span>剩余</span>{{ tokenQuota.sevenDay.remainingPercent }}%</b>
              </header>
              <div class="quota-progress" aria-hidden="true">
                <span :style="{ width: percentWidth(tokenQuota.sevenDay.remainingPercent) }" />
              </div>
              <div class="quota-reset-time">
                <Icon icon="solar:restart-linear" />
                <span>重置时间</span>
                <time v-if="tokenQuota.sevenDay.resetAt" :datetime="tokenQuota.sevenDay.resetAt">
                  {{ formatResetTime(tokenQuota.sevenDay.resetAt) }}
                </time>
                <strong v-else>暂无</strong>
              </div>
            </article>
          </div>

          <div v-if="!tokenQuota && !quotaLoading" class="quota-empty">
            <Icon icon="solar:graph-up-linear" />
            <span>暂时无法读取额度信息，请稍后再试。</span>
          </div>
        </n-spin>
      </section>

      <section v-else-if="section === 'appearance'" class="settings-content-inner appearance-page">
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
          </div>
          <n-button
            v-if="archivedConversations.length"
            tertiary
            type="error"
            class="archive-delete-all"
            @click="emit('deleteAllArchived')"
          >
            <template #icon><Icon icon="solar:trash-bin-trash-linear" /></template>
            全部删除
          </n-button>
        </header>

        <div class="archive-toolbar">
          <n-input
            :value="archiveQuery"
            size="large"
            clearable
            placeholder="搜索已归档对话"
            class="archive-search"
            @update:value="value => emit('update:archiveQuery', value)"
          >
            <template #prefix><Icon icon="solar:magnifer-linear" /></template>
          </n-input>
          <n-select
            v-model:value="archiveGroup"
            size="large"
            :options="archiveGroupOptions"
            class="archive-group-filter"
          />
        </div>

        <n-spin :show="loadingList">
          <div v-if="visibleArchiveSections.length" class="archive-list archive-page-list">
            <section
              v-for="archiveSection in visibleArchiveSections"
              :key="archiveSection.groupId || ungroupedArchiveGroup"
              class="archive-section"
            >
              <div class="archive-section-head">
                <h2 class="archive-section-title">
                  <Icon icon="solar:folder-linear" />
                  {{ archiveSection.title }}
                </h2>
                <div class="archive-section-meta">
                  <span>{{ archiveSection.items.length }} 个对话</span>
                  <n-button
                    v-if="archiveSection.groupId"
                    text
                    type="error"
                    size="small"
                    @click="emit('deleteGroupArchived', archiveSection)"
                  >
                    删除本组
                  </n-button>
                </div>
              </div>

              <div class="archive-section-list">
                <div v-for="item in archiveSection.items" :key="item.id" class="archive-item">
                  <div class="archive-item-copy">
                    <strong>{{ displayConversationTitle(item) }}</strong>
                    <span>{{ formatConversationDate(item.updatedAt) }}</span>
                  </div>
                  <n-button
                    quaternary
                    circle
                    class="archive-delete-button"
                    aria-label="删除"
                    @click="emit('deleteArchived', item)"
                  >
                    <template #icon><Icon icon="solar:trash-bin-trash-linear" /></template>
                  </n-button>
                  <n-button
                    secondary
                    class="archive-restore-button"
                    @click="emit('restoreArchived', item)"
                  >
                    取消归档
                  </n-button>
                </div>
              </div>
            </section>
          </div>
          <div v-else class="archive-empty">
            {{ archiveQuery.trim() || archiveGroup !== allArchiveGroups ? '没有匹配的归档对话' : '暂无已归档对话' }}
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

<style scoped>
.usage-page {
  width: min(920px, calc(100% - 64px));
}

.quota-disabled-banner {
  display: flex;
  align-items: center;
  gap: 13px;
  margin-bottom: 16px;
  padding: 15px 17px;
  border: 1px solid #f0c8c4;
  border-radius: 14px;
  color: #8f3934;
  background: #fff4f2;
}

.quota-disabled-banner > svg {
  flex: 0 0 auto;
  font-size: 24px;
}

.quota-disabled-banner > div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.quota-disabled-banner span {
  color: #aa5b55;
  font-size: 13px;
}

.quota-window-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.quota-window-grid--single {
  grid-template-columns: minmax(0, 1fr);
}

.quota-window-card {
  padding: 21px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--panel-bg);
  box-shadow: var(--shadow-panel);
}

.quota-window-card > header,
.quota-window-card > header > div {
  display: flex;
  align-items: center;
}

.quota-window-card > header {
  justify-content: space-between;
  gap: 16px;
}

.quota-window-card > header > div {
  min-width: 0;
  gap: 11px;
}

.quota-window-card > header > div > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.quota-window-card header strong {
  color: var(--text-strong);
  font-size: 16px;
}

.quota-window-card header b {
  display: flex;
  align-items: baseline;
  gap: 7px;
  color: var(--accent);
  font-size: 26px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.5px;
}

.quota-window-card header b span {
  color: var(--muted);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0;
}

.quota-window-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 11px;
  color: var(--accent);
  background: var(--accent-soft);
  font-size: 20px;
}

.quota-progress {
  overflow: hidden;
  height: 8px;
  margin-top: 20px;
  border-radius: 999px;
  background: var(--border-soft);
}

.quota-progress > span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #4ca2d2, #2b82bc);
  transition: width 220ms ease;
}

.quota-reset-time {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 17px;
  padding-top: 14px;
  border-top: 1px solid var(--border-soft);
  color: var(--muted);
  font-size: 14px;
}

.quota-reset-time svg {
  flex: 0 0 auto;
  color: var(--accent);
  font-size: 18px;
}

.quota-reset-time time,
.quota-reset-time strong {
  margin-left: auto;
  color: var(--text-strong);
  font-size: 14px;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.quota-empty {
  display: grid;
  min-height: 220px;
  place-items: center;
  align-content: center;
  gap: 10px;
  border: 1px dashed #dce4e7;
  border-radius: 15px;
  color: #8b989d;
}

.quota-empty svg {
  font-size: 28px;
}

:global(:root[data-theme="dark"]) .quota-window-card {
  border-color: #383838;
  background: #222;
  box-shadow: none;
}

:global(:root[data-theme="dark"]) .quota-window-card header strong,
:global(:root[data-theme="dark"]) .quota-reset-time time,
:global(:root[data-theme="dark"]) .quota-reset-time strong {
  color: #ededed;
}

:global(:root[data-theme="dark"]) .quota-window-card header b span,
:global(:root[data-theme="dark"]) .quota-reset-time {
  color: #a1a1a1;
}

:global(:root[data-theme="dark"]) .quota-window-icon {
  color: #83c0eb;
  background: #293942;
}

:global(:root[data-theme="dark"]) .quota-progress {
  background: #3a3a3a;
}

:global(:root[data-theme="dark"]) .quota-reset-time {
  border-top-color: #3d3d3d;
}

:global(:root[data-theme="dark"]) .quota-disabled-banner {
  border-color: #67423e;
  color: #ffb4ab;
  background: #382421;
}

:global(:root[data-theme="dark"]) .quota-disabled-banner span {
  color: #d9948d;
}

:global(:root[data-theme="dark"]) .quota-empty {
  border-color: #414141;
  color: #999;
}

@media (max-width: 860px) {
  .quota-window-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 720px) {
  .usage-page {
    width: calc(100% - 28px);
    padding: 12px 0 48px;
  }

  .quota-window-card {
    padding: 18px;
  }

  .quota-window-card header b {
    font-size: 22px;
  }
}

.archives-page {
  width: min(1120px, calc(100% - 64px));
}

.archive-page-header {
  align-items: center;
  margin-bottom: 52px;
}

.archive-page-header h1 {
  margin-bottom: 0;
  font-size: 30px;
  font-weight: 600;
}

.archive-delete-all {
  border-radius: 12px;
  background: #fff1f1;
}

.archive-toolbar {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) 250px;
  gap: 14px;
  margin-bottom: 44px;
}

.archive-toolbar .archive-search {
  margin-bottom: 0;
}

.archive-search :deep(.n-input),
.archive-group-filter :deep(.n-base-selection) {
  border-radius: 14px;
}

.archive-list.archive-page-list {
  display: block;
  overflow: visible;
}

.archive-section + .archive-section {
  margin-top: 34px;
}

.archive-section-head {
  min-height: 28px;
  margin: 0 4px 14px;
}

.archive-section-title {
  display: flex;
  align-items: center;
  gap: 9px;
  color: #30383c;
  font-size: 15px;
  font-weight: 600;
}

.archive-section-title svg {
  flex: 0 0 auto;
  font-size: 18px;
}

.archive-section-meta {
  display: flex;
  align-items: center;
  gap: 14px;
  color: #899399;
  font-size: 13px;
}

.archive-section-list {
  overflow: hidden;
  border: 1px solid #e5e9eb;
  border-radius: 16px;
  background: #fff;
}

.archive-page-list .archive-item {
  min-height: 82px;
  gap: 12px;
  padding: 14px 18px 14px 22px;
  border: 0;
  border-bottom: 1px solid #edf0f1;
  border-radius: 0;
  background: transparent;
}

.archive-page-list .archive-item:last-child {
  border-bottom: 0;
}

.archive-item-copy {
  gap: 5px;
}

.archive-item-copy strong {
  color: #202729;
  font-size: 15px;
  font-weight: 600;
}

.archive-item-copy span {
  font-size: 13px;
}

.archive-delete-button {
  color: #98a1a5;
}

.archive-restore-button {
  min-width: 96px;
  height: 40px;
  border-radius: 12px;
}

.archive-restore-button :deep(.n-button__border) {
  border: 0;
}

.archive-empty {
  min-height: 180px;
  border-radius: 16px;
  font-size: 14px;
}

:global(:root[data-theme="dark"]) .archive-delete-all {
  background: rgb(239 104 104 / 12%);
}

:global(:root[data-theme="dark"]) .archive-section-title {
  color: #edf1f2;
}

:global(:root[data-theme="dark"]) .archive-section-list {
  border-color: #3d3d3d;
  background: #222;
}

:global(:root[data-theme="dark"]) .archive-page-list .archive-item {
  border-bottom-color: #353535;
  background: transparent;
}

:global(:root[data-theme="dark"]) .archive-delete-button {
  color: #989898;
}

@media (max-width: 720px) {
  .archives-page {
    width: calc(100% - 28px);
  }

  .archive-page-header {
    flex-direction: row;
    align-items: center;
    margin-bottom: 28px;
  }

  .archive-page-header > div {
    flex: 1;
  }

  .archive-page-header h1 {
    font-size: 24px;
  }

  .archive-toolbar {
    grid-template-columns: minmax(0, 1fr);
    gap: 10px;
    margin-bottom: 28px;
  }

  .archive-page-list .archive-item {
    min-height: 76px;
    padding: 12px 12px 12px 16px;
  }

  .archive-restore-button {
    min-width: auto;
  }

  .archive-section-meta > span {
    display: none;
  }
}
</style>
