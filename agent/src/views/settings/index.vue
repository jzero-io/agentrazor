<script setup lang="ts">
import { Icon } from '@iconify/vue';
import { NButton, NInput, NSpin } from 'naive-ui';
import type { Conversation } from '../../service/api';

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

defineProps<{
  visible: boolean;
  navExpanded: boolean;
  section: 'appearance' | 'archives';
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
  'update:section': [value: 'appearance' | 'archives'];
  'update:archiveQuery': [value: string];
  setAppearance: [value: 'system' | 'light' | 'dark'];
  deleteAllArchived: [];
  deleteGroupArchived: [section: ArchiveSection];
  deleteArchived: [item: Conversation];
  restoreArchived: [item: Conversation];
}>();

function openArchives() {
  emit('update:section', 'archives');
}
</script>

<template>
  <section v-if="visible" class="settings-shell" :class="{ 'settings-nav-collapsed': !navExpanded }">
    <aside class="settings-sidebar">
      <button class="settings-back" type="button" @click="emit('update:visible', false)">
        <Icon icon="solar:arrow-left-linear" />
        <span>返回应用</span>
      </button>
      <div class="settings-sidebar-title">设置</div>
      <nav aria-label="设置导航">
        <button :class="{ active: section === 'appearance' }" @click="emit('update:section', 'appearance')">
          <Icon icon="solar:sun-2-linear" />
          <span>外观</span>
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

      <section v-else class="settings-content-inner archives-page">
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
    </main>
  </section>
</template>
