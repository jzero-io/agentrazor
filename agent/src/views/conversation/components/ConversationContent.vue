<script setup lang="ts">
defineOptions({
  name: 'ConversationContent',
  inheritAttrs: false
});

import type { ComponentPublicInstance } from 'vue';
import { Icon } from '@iconify/vue';
import type { Conversation, ThreadItem, Turn } from '../../../service/api';
import type { ProcessDisplayItem } from '../../../utils/processDisplay';
import type { WorkspaceDescriptor } from '../../../hooks/business/useWorkspacePanel';
import ComposerBox from '../../../layouts/modules/composer-box/index.vue';
import MarkdownBlock from './MarkdownBlock.vue';
import ProcessItemCard from './ProcessItemCard.vue';

interface TurnView {
  renderKey: string;
  turn: Turn;
  userItems: ThreadItem[];
  resultItems: ThreadItem[];
  processDisplays: ProcessDisplayItem[];
  processMode: 'none' | 'thinking' | 'processing' | 'completed';
  processSummary: string;
  showTailThinking: boolean;
  streaming: boolean;
}

interface MessagePreview {
  id: string;
  title: string;
  finalAnswerHtml?: string;
}

interface ParsedAgentMessage {
  markdown: string;
  workspaces: WorkspaceDescriptor[];
}

defineProps<{
  selectedConversationId: string;
  isNewChat: boolean;
  currentUser: unknown;
  isArchivedActive: boolean;
  loadingCurrentDetail: boolean;
  renderedTurnViews: TurnView[];
  messagePreviews: MessagePreview[];
  selectedPreviewMessageId: string;
  hoveredPreviewMessageId: string;
  draft: string;
  composerActionPending: boolean;
  sending: boolean;
  composerActionDisabled: boolean;
  composerActionLabel: string;
  composerActionIcon: string;
  copiedMessageId: string;
  normalizeWorkspaceFilePath: (href: string) => string;
  displayWorkspaceProcessPath: (filePath: string) => string;
  openWorkspaceFile: (path: string) => void;
  setMessagePane: (el: Element | ComponentPublicInstance | null) => void;
  handleMessageScroll: (event: Event) => void;
  previewTickClasses: (index: number) => unknown;
  jumpToMessage: (id: string) => void;
  userItemText: (item: ThreadItem) => string;
  formatMessageTime: (value?: string) => string;
  assistantMessageTime: (turn: Turn) => string;
  messageCopyText: (item: ThreadItem) => string;
  copyMessage: (item: ThreadItem) => void;
  parseAgentMessage: (content: string, streaming: boolean) => ParsedAgentMessage;
  openWorkspace: (workspace: WorkspaceDescriptor) => void;
  activityIcon: (item: ThreadItem) => string;
  activityTitle: (item: ThreadItem) => string;
  openLogin: () => void;
}>();

const emit = defineEmits<{
  'update:draft': [value: string];
  'update:hoveredPreviewMessageId': [value: string];
  composerKeydown: [event: KeyboardEvent];
  composerAction: [];
  error: [message: string];
}>();

function emitError(message: string) {
  emit('error', message);
}
</script>

<template>
  <section v-if="!currentUser" class="empty-state guest-empty-state">
    <div class="welcome-icon"><Icon icon="solar:lock-keyhole-minimalistic-bold-duotone" /></div>
    <h1>登录后开始</h1>
    <p>登录账号即可使用你的专属 Agent 对话。</p>
    <n-button type="primary" size="large" class="guest-login-button" @click="openLogin">登录</n-button>
  </section>

  <template v-else-if="selectedConversationId && isNewChat">
    <section class="chat-empty">
      <div class="welcome">
        <div class="welcome-icon"><img src="/agentrazor-icon.png" alt="" /></div>
        <h1>今天想完成什么？</h1>
        <p>输入目标，AgentRazor 会在当前对话中持续处理。</p>
      </div>
    </section>
    <ComposerBox
      :model-value="draft"
      :pending="composerActionPending"
      :running="sending"
      :disabled="composerActionDisabled"
      :label="composerActionLabel"
      :icon="composerActionIcon"
      @update:model-value="value => emit('update:draft', value)"
      @keydown="event => emit('composerKeydown', event)"
      @action="emit('composerAction')"
    />
  </template>

  <template v-else-if="selectedConversationId">
    <nav v-if="messagePreviews.length" class="conversation-preview" aria-label="用户消息导航">
      <button
        v-for="(item, index) in messagePreviews"
        :key="item.id"
        class="preview-tick"
        :class="previewTickClasses(index)"
        :aria-current="item.id === selectedPreviewMessageId ? 'true' : undefined"
        :aria-label="`跳转到第 ${index + 1} 条用户消息：${item.title}`"
        @mouseenter="emit('update:hoveredPreviewMessageId', item.id)"
        @mouseleave="emit('update:hoveredPreviewMessageId', '')"
        @focus="emit('update:hoveredPreviewMessageId', item.id)"
        @blur="emit('update:hoveredPreviewMessageId', '')"
        @click="jumpToMessage(item.id)"
      >
        <span class="preview-card" aria-hidden="true">
          <strong>{{ item.title }}</strong>
          <span v-if="item.finalAnswerHtml" class="preview-card-markdown markdown-body" v-html="item.finalAnswerHtml" />
        </span>
      </button>
    </nav>
    <section :key="selectedConversationId" :ref="setMessagePane" class="message-pane" @scroll="handleMessageScroll">
      <n-spin class="message-spin" :show="loadingCurrentDetail">
        <div class="message-column">
          <section v-for="view in renderedTurnViews" :key="view.renderKey" class="turn" :data-status="view.turn.status">
            <article
              v-for="item in view.userItems"
              :key="item.id"
              :id="`message-${item.id}`"
              tabindex="-1"
              class="message user"
            >
              <div class="message-stack">
                <div class="message-content">{{ userItemText(item) }}</div>
                <div class="message-meta">
                  <time v-if="formatMessageTime(view.turn.startedAt)" class="message-time" :datetime="view.turn.startedAt">
                    {{ formatMessageTime(view.turn.startedAt) }}
                  </time>
                  <button
                    v-if="messageCopyText(item)"
                    type="button"
                    class="message-copy-button"
                    :class="{ copied: copiedMessageId === item.id }"
                    aria-label="复制消息"
                    title="复制消息"
                    @click="copyMessage(item)"
                  >
                    <span v-if="copiedMessageId === item.id" class="message-copy-tip">已复制</span>
                    <Icon :key="copiedMessageId === item.id ? 'copied' : 'copy'" :icon="copiedMessageId === item.id ? 'solar:check-circle-linear' : 'solar:copy-linear'" />
                  </button>
                </div>
              </div>
            </article>

            <div v-if="view.processMode === 'thinking'" class="turn-live-status">
              <span class="status-pulse">正在思考</span>
            </div>

            <div v-else-if="view.processMode === 'processing'" class="turn-process">
              <div class="turn-process-summary">
                <span>{{ view.processSummary }}</span>
              </div>
              <div class="turn-process-content">
                <ProcessItemCard
                  v-for="display in view.processDisplays"
                  :key="display.item.id"
                  :display="display"
                  :display-workspace-process-path="displayWorkspaceProcessPath"
                  :normalize-workspace-file-path="normalizeWorkspaceFilePath"
                  @open-workspace-file="openWorkspaceFile"
                  @error="emitError"
                />
                <div v-if="view.showTailThinking" class="turn-inline-thinking">
                  <span class="status-pulse">正在思考</span>
                </div>
              </div>
            </div>

            <details
              v-else-if="view.processMode === 'completed'"
              class="turn-process turn-process-done"
            >
              <summary>
                <span>{{ view.processSummary }}</span>
              </summary>
              <div v-if="view.processDisplays.length" class="turn-process-content">
                <ProcessItemCard
                  v-for="display in view.processDisplays"
                  :key="display.item.id"
                  :display="display"
                  :display-workspace-process-path="displayWorkspaceProcessPath"
                  :normalize-workspace-file-path="normalizeWorkspaceFilePath"
                  @open-workspace-file="openWorkspaceFile"
                  @error="emitError"
                />
              </div>
            </details>

            <template v-for="item in view.resultItems" :key="item.id">
              <article v-if="item.type === 'agentMessage'" class="message assistant">
                <div class="message-content">
                  <MarkdownBlock
                    v-if="item.text && parseAgentMessage(item.text, view.streaming).markdown"
                    :content="parseAgentMessage(item.text, view.streaming).markdown"
                    :streaming="view.streaming"
                    :normalize-workspace-file-path="normalizeWorkspaceFilePath"
                    @open-workspace-file="openWorkspaceFile"
                    @error="emitError"
                  />
                  <button
                    v-for="workspace in parseAgentMessage(item.text || '', view.streaming).workspaces"
                    :key="workspace.url"
                    type="button"
                    class="workspace-card"
                    @click="openWorkspace(workspace)"
                  >
                    <Icon icon="solar:widget-5-linear" />
                    <span><strong>{{ workspace.title }}</strong><small>在右侧打开工作台</small></span>
                    <Icon icon="solar:arrow-right-linear" />
                  </button>
                  <div class="message-meta">
                    <time v-if="assistantMessageTime(view.turn)" class="message-time" :datetime="view.turn.completedAt || view.turn.startedAt">
                      {{ assistantMessageTime(view.turn) }}
                    </time>
                    <button
                      v-if="messageCopyText(item)"
                      type="button"
                      class="message-copy-button"
                      :class="{ copied: copiedMessageId === item.id }"
                      aria-label="复制消息"
                      title="复制消息"
                      @click="copyMessage(item)"
                    >
                      <span v-if="copiedMessageId === item.id" class="message-copy-tip">已复制</span>
                      <Icon :key="copiedMessageId === item.id ? 'copied' : 'copy'" :icon="copiedMessageId === item.id ? 'solar:check-circle-linear' : 'solar:copy-linear'" />
                    </button>
                  </div>
                </div>
              </article>

              <article v-else-if="item.type === 'imageGeneration'" class="activity-card image-item">
                <div class="activity-heading"><Icon :icon="activityIcon(item)" /><span>{{ activityTitle(item) }}</span></div>
                <n-image
                  v-if="item.dataUrl"
                  class="generated-image"
                  :src="item.dataUrl"
                  :alt="item.alt || String(item.result || '生成的图片')"
                  object-fit="contain"
                  lazy
                />
              </article>
            </template>

            <article v-if="view.turn.error" class="turn-error">{{ view.turn.error }}</article>
          </section>
        </div>
      </n-spin>
    </section>

    <ComposerBox
      v-if="!isArchivedActive"
      :model-value="draft"
      :pending="composerActionPending"
      :running="sending"
      :disabled="composerActionDisabled"
      :label="composerActionLabel"
      :icon="composerActionIcon"
      @update:model-value="value => emit('update:draft', value)"
      @keydown="event => emit('composerKeydown', event)"
      @action="emit('composerAction')"
    />
  </template>

  <section v-else class="chat-empty" aria-hidden="true"></section>
</template>
