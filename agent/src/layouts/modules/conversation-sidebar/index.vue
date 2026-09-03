<script setup lang="ts">
import { computed, h, type ComponentPublicInstance } from 'vue';
import { Icon } from '@iconify/vue';
import { NButton, NDropdown, NScrollbar, NSpin, NTooltip } from 'naive-ui';
import type { Conversation, UserInfo } from '../../../service/api';
import ScrollingTitle from '../scrolling-title/index.vue';

defineOptions({
  name: 'ConversationSidebar'
});

interface SidebarGroup {
  id: string;
  name: string;
  collapsed: boolean;
}

const props = defineProps<{
  mobileSidebarOpen: boolean;
  sidebarExpanded: boolean;
  sidebarCollapsed: boolean;
  currentUser: UserInfo | null;
  userInitial: string;
  userMenuVisible: boolean;
  pinnedExpanded: boolean;
  groupsExpanded: boolean;
  conversationsExpanded: boolean;
  loadingList: boolean;
  pinnedConversations: Conversation[];
  conversationGroups: SidebarGroup[];
  conversationList: Conversation[];
  selectedConversationId: string;
  draggedConversationId: string;
  conversationDropTarget: string;
  conversationListDropTarget: string;
  displayConversationTitle: (item?: Pick<Conversation, 'title'> | null) => string;
  isConversationProcessing: (item: Conversation) => boolean;
  conversationsInGroup: (groupId: string) => Conversation[];
  setUserMenuWrap: (el: Element | ComponentPublicInstance | null) => void;
  closeSidebarHover: () => void;
  toggleSidebarPinned: () => void;
  createConversation: () => void;
  createConversationInGroup: (group: SidebarGroup) => void;
  openCreateGroup: () => void;
  toggleGroup: (group: SidebarGroup) => void;
  handleGroupAction: (group: SidebarGroup, key: string) => void;
  hideConversationPreview: () => void;
  showConversationPreview: (item: Conversation, event: MouseEvent) => void;
  selectConversation: (id: string) => void;
  renameConversation: (item: Conversation) => void;
  onConversationPointerDown: (item: Conversation, event: PointerEvent) => void;
  onRowTouchStart: (item: Conversation, event: TouchEvent) => void;
  toggleConversationPinned: (item: Conversation) => void;
  archiveConversation: (item: Conversation) => void;
  openSettings: () => void;
  logout: () => void;
  openLogin: () => void;
  startSidebarResize: (event: PointerEvent) => void;
}>();

const emit = defineEmits<{
  'update:userMenuVisible': [value: boolean];
  'update:pinnedExpanded': [value: boolean];
  'update:groupsExpanded': [value: boolean];
  'update:conversationsExpanded': [value: boolean];
}>();

const userMenuVisible = computed({
  get: () => props.userMenuVisible,
  set: value => emit('update:userMenuVisible', value)
});
const pinnedExpanded = computed({
  get: () => props.pinnedExpanded,
  set: value => emit('update:pinnedExpanded', value)
});
const groupsExpanded = computed({
  get: () => props.groupsExpanded,
  set: value => emit('update:groupsExpanded', value)
});
const conversationsExpanded = computed({
  get: () => props.conversationsExpanded,
  set: value => emit('update:conversationsExpanded', value)
});

const renderIcon = (icon: string) => () => h(Icon, { icon });

function selectGroupAction(group: SidebarGroup, key: string | number) {
  props.handleGroupAction(group, String(key));
}

</script>

<template>
  <aside class="sidebar" :class="{ 'mobile-open': mobileSidebarOpen }" @mouseleave="closeSidebarHover">
        <div class="brand">
          <div class="brand-mark"><img src="/agentrazor-icon.png" alt="" /></div>
          <div v-if="sidebarExpanded" class="brand-copy">
            <strong>AgentRazor</strong>
          </div>
          <n-button v-if="currentUser && sidebarExpanded" quaternary circle class="collapse-button" @click="toggleSidebarPinned">
            <template #icon>
              <Icon icon="lucide:panel-left" />
            </template>
          </n-button>
        </div>

        <n-button class="new-chat" secondary @click="createConversation">
          <template #icon><Icon icon="solar:pen-new-square-outline" /></template>
          <span v-if="sidebarExpanded">新对话</span>
        </n-button>

        <div v-if="sidebarExpanded" class="sidebar-section">
          <n-spin :show="loadingList">
            <n-scrollbar class="conversation-scroll" @scroll="hideConversationPreview">
              <div v-if="pinnedConversations.length" class="conversation-group">
                <button class="conversation-group-toggle" @click="pinnedExpanded = !pinnedExpanded">
                  <span>置顶</span>
                  <Icon :icon="pinnedExpanded ? 'solar:alt-arrow-down-linear' : 'solar:alt-arrow-right-linear'" />
                </button>
                <template v-if="pinnedExpanded">
                  <div
                    v-for="item in pinnedConversations"
                    :key="item.id"
                    class="conversation-row previewable-conversation-row"
                    :class="{ selected: item.id === selectedConversationId, dragging: item.id === draggedConversationId }"
                    @pointerdown="onConversationPointerDown(item, $event)"
                    @touchstart="onRowTouchStart(item, $event)"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)" @dblclick.stop="renameConversation(item)">
                      <Icon icon="solar:pin-bold" class="pin-icon" />
                      <ScrollingTitle :key="`${item.id}:${displayConversationTitle(item)}`" :text="displayConversationTitle(item)" />
                    </button>
                    <button
                      v-if="isConversationProcessing(item)"
                      type="button"
                      class="conversation-processing-button"
                      aria-label="处理中"
                      title="处理中"
                      disabled
                    >
                      <Icon icon="solar:refresh-linear" />
                    </button>
                    <div class="conversation-actions">
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="取消置顶" @click="toggleConversationPinned(item)">
                            <Icon icon="solar:pin-bold" />
                          </button>
                        </template>
                        取消置顶
                      </n-tooltip>
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="归档" @click="archiveConversation(item)">
                            <Icon icon="solar:archive-linear" />
                          </button>
                        </template>
                        归档
                      </n-tooltip>
                    </div>
                  </div>
                </template>
              </div>
              <div class="group-section-heading">
                <button class="group-section-toggle" type="button" @click="groupsExpanded = !groupsExpanded">
                  <span>分组</span>
                  <Icon :icon="groupsExpanded ? 'solar:alt-arrow-down-linear' : 'solar:alt-arrow-right-linear'" />
                </button>
                <div class="group-actions">
                  <n-tooltip trigger="hover" placement="top">
                    <template #trigger>
                      <button class="group-add-button" type="button" aria-label="新增分组" @click="openCreateGroup()">
                        <Icon icon="solar:add-circle-linear" />
                      </button>
                    </template>
                    新增分组
                  </n-tooltip>
                </div>
              </div>
              <template v-if="groupsExpanded">
              <div
                v-for="group in conversationGroups"
                :key="group.id"
                class="conversation-group custom-group conversation-drop-zone"
                :class="{ 'drag-over': conversationDropTarget === group.id }"
                :data-group-id="group.id"
              >
                <div class="custom-group-heading">
                  <button class="conversation-group-toggle" @click="toggleGroup(group)">
                    <Icon
                      :icon="group.collapsed ? 'lucide:folder' : 'lucide:folder-open'"
                      class="group-folder-icon"
                    />
                    <span>{{ group.name }}</span>
                  </button>
                  <div class="group-actions">
                    <n-dropdown
                      trigger="click"
                      placement="right-start"
                      :options="[
                        { label: '重命名分组', key: 'rename', icon: renderIcon('solar:pen-2-linear') },
                        { label: '归档对话', key: 'archiveConversations', icon: renderIcon('solar:archive-linear') },
                        { type: 'divider', key: 'divider' },
                        { label: '删除分组', key: 'delete', icon: renderIcon('solar:trash-bin-trash-linear') }
                      ]"
                      @select="key => selectGroupAction(group, key)"
                    >
                      <button type="button" aria-label="更多分组操作"><Icon icon="lucide:ellipsis" /></button>
                    </n-dropdown>
                    <n-tooltip trigger="hover" placement="top">
                      <template #trigger>
                        <button type="button" aria-label="在分组中新增对话" @click="createConversationInGroup(group)">
                          <Icon icon="solar:pen-new-square-outline" />
                        </button>
                      </template>
                      新增对话
                    </n-tooltip>
                  </div>
                </div>
                <template v-if="!group.collapsed">
                  <div
                    v-for="item in conversationsInGroup(group.id)"
                    :key="item.id"
                    class="conversation-row grouped-conversation previewable-conversation-row"
                    :class="{ selected: item.id === selectedConversationId, dragging: item.id === draggedConversationId }"
                    @pointerdown="onConversationPointerDown(item, $event)"
                    @touchstart="onRowTouchStart(item, $event)"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)" @dblclick.stop="renameConversation(item)">
                      <ScrollingTitle :key="`${item.id}:${displayConversationTitle(item)}`" :text="displayConversationTitle(item)" />
                    </button>
                    <button
                      v-if="isConversationProcessing(item)"
                      type="button"
                      class="conversation-processing-button"
                      aria-label="处理中"
                      title="处理中"
                      disabled
                    >
                      <Icon icon="solar:refresh-linear" />
                    </button>
                    <div class="conversation-actions">
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button
                            type="button"
                            :aria-label="item.pinnedAt ? '取消置顶' : '置顶'"
                            @click="toggleConversationPinned(item)"
                          >
                            <Icon :icon="item.pinnedAt ? 'solar:pin-bold' : 'solar:pin-linear'" />
                          </button>
                        </template>
                        {{ item.pinnedAt ? '取消置顶' : '置顶' }}
                      </n-tooltip>
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="归档" @click="archiveConversation(item)"><Icon icon="solar:archive-linear" /></button>
                        </template>
                        归档
                      </n-tooltip>
                    </div>
                  </div>
                  <div v-if="!conversationsInGroup(group.id).length" class="sidebar-empty-state">
                    <Icon icon="solar:inbox-line-outline" />
                    <span>暂无对话</span>
                  </div>
                </template>
              </div>
              </template>
              <div
                class="conversation-group conversation-list-group conversation-drop-zone"
                :class="{ 'drag-over': conversationDropTarget === conversationListDropTarget }"
                :data-drop-target="conversationListDropTarget"
              >
                <div class="conversation-list-heading">
                  <button class="conversation-group-toggle" @click="conversationsExpanded = !conversationsExpanded">
                    <span>对话</span>
                    <Icon :icon="conversationsExpanded ? 'solar:alt-arrow-down-linear' : 'solar:alt-arrow-right-linear'" />
                  </button>
                  <div class="group-actions">
                    <n-tooltip trigger="hover" placement="top">
                      <template #trigger>
                        <button class="group-add-button" type="button" aria-label="新对话" @click="createConversation">
                          <Icon icon="solar:pen-new-square-outline" />
                        </button>
                      </template>
                      新对话
                    </n-tooltip>
                  </div>
                </div>
                <template v-if="conversationsExpanded">
                  <div
                    v-for="item in conversationList"
                    :key="item.id"
                    class="conversation-row previewable-conversation-row"
                    :class="{ selected: item.id === selectedConversationId, dragging: item.id === draggedConversationId }"
                    @pointerdown="onConversationPointerDown(item, $event)"
                    @touchstart="onRowTouchStart(item, $event)"
                    @mouseenter="showConversationPreview(item, $event)"
                    @mouseleave="hideConversationPreview"
                  >
                    <button class="conversation-item" @click="hideConversationPreview(); selectConversation(item.id)" @dblclick.stop="renameConversation(item)">
                      <ScrollingTitle :key="`${item.id}:${displayConversationTitle(item)}`" :text="displayConversationTitle(item)" />
                    </button>
                    <button
                      v-if="isConversationProcessing(item)"
                      type="button"
                      class="conversation-processing-button"
                      aria-label="处理中"
                      title="处理中"
                      disabled
                    >
                      <Icon icon="solar:refresh-linear" />
                    </button>
                    <div class="conversation-actions">
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="置顶" @click="toggleConversationPinned(item)">
                            <Icon icon="solar:pin-linear" />
                          </button>
                        </template>
                        置顶
                      </n-tooltip>
                      <n-tooltip trigger="hover" placement="top">
                        <template #trigger>
                          <button type="button" aria-label="归档" @click="archiveConversation(item)">
                            <Icon icon="solar:archive-linear" />
                          </button>
                        </template>
                        归档
                      </n-tooltip>
                    </div>
                  </div>
                </template>
              </div>
              <div v-if="conversationsExpanded && !loadingList && !conversationList.length" class="sidebar-empty-state">
                <Icon icon="solar:inbox-line-outline" />
                <span>暂无对话</span>
              </div>
            </n-scrollbar>
          </n-spin>
        </div>

        <div v-if="sidebarExpanded" class="sidebar-footer" :class="{ 'login-footer': !currentUser }">
          <template v-if="currentUser">
            <div :ref="setUserMenuWrap" class="user-menu-wrap" @keydown.esc="userMenuVisible = false">
              <div v-if="userMenuVisible" class="user-popover" role="menu">
                <div class="user-popover-account">
                  <span class="user-avatar">{{ userInitial }}</span>
                  <div>
                    <strong>{{ currentUser.username }}</strong>
                    <span>当前登录账号</span>
                  </div>
                </div>
                <button type="button" role="menuitem" @click="openSettings">
                  <Icon icon="solar:settings-linear" />
                  <span>设置</span>
                </button>
                <button type="button" role="menuitem" class="logout-menu-item" @click="logout">
                  <Icon icon="solar:logout-2-linear" />
                  <span>退出登录</span>
                </button>
              </div>
              <button
                class="user-menu-trigger"
                type="button"
                aria-haspopup="menu"
                :aria-expanded="userMenuVisible"
                @click="userMenuVisible = !userMenuVisible"
              >
                <span class="user-avatar">{{ userInitial }}</span>
                <span class="user-name">{{ currentUser.username }}</span>
              </button>
            </div>
          </template>
          <button v-else type="button" class="login-entry" @click="openLogin">
            <span class="login-entry-icon">
              <Icon icon="solar:login-3-linear" />
            </span>
            <span class="login-entry-copy">
              <strong>登录 AgentRazor</strong>
              <span>同步你的对话</span>
            </span>
          </button>
        </div>
        <div v-if="currentUser && !sidebarCollapsed" class="sidebar-resize-handle" role="separator" aria-orientation="vertical" title="拖动调整侧边栏宽度" @pointerdown="startSidebarResize" />
  </aside>


</template>
