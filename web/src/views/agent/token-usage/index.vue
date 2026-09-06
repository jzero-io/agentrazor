<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import {
  GetTokenUsageConversations,
  GetTokenUsageDetails,
  GetTokenUsageTrend,
  type TokenUsageAccount,
  type TokenUsageConversation,
  type TokenUsageDimension,
  type TokenUsageSummary,
  type TokenUsageTrendPoint
} from '@/service/api';
import { useEcharts } from '@/hooks/common/echarts';
import { useThemeStore } from '@/store/modules/theme';

defineOptions({ name: 'AgentTokenUsage' });

interface ConversationPageState {
  current: number;
  size: number;
  total: number;
  keyword: string;
  loading: boolean;
  loaded: boolean;
  conversations: TokenUsageConversation[];
}

const emptySummary = (): TokenUsageSummary => ({
  inputTokens: 0,
  cachedInputTokens: 0,
  cacheWriteInputTokens: 0,
  outputTokens: 0,
  reasoningOutputTokens: 0,
  totalTokens: 0
});

const themeStore = useThemeStore();
const dimension = ref<TokenUsageDimension>('day');
const trendLoading = ref(false);
const detailsLoading = ref(false);
const points = ref<TokenUsageTrendPoint[]>([]);
const accounts = ref<TokenUsageAccount[]>([]);
const tokenSummary = ref<TokenUsageSummary>(emptySummary());
const accountKeyword = ref('');
const accountPage = ref(1);
const accountPageSize = ref(10);
const accountTotal = ref(0);
const expandedAccountNames = ref<string[]>([]);
const conversationPages = ref<Record<string, ConversationPageState>>({});
let requestSequence = 0;

const tokenMetrics = computed(() => [
  { label: '输入', value: tokenSummary.value.inputTokens, icon: 'carbon:download', tone: 'blue' },
  { label: '缓存输入', value: tokenSummary.value.cachedInputTokens, icon: 'carbon:data-base', tone: 'cyan' },
  { label: '缓存写入', value: tokenSummary.value.cacheWriteInputTokens, icon: 'carbon:save', tone: 'orange' },
  { label: '输出', value: tokenSummary.value.outputTokens, icon: 'carbon:upload', tone: 'green' },
  { label: '推理输出', value: tokenSummary.value.reasoningOutputTokens, icon: 'carbon:idea', tone: 'violet' },
  { label: '总 Token', value: tokenSummary.value.totalTokens, icon: 'carbon:meter', tone: 'primary' }
]);

function formatToken(value: number) {
  return value.toLocaleString('zh-CN');
}
function formatCompactToken(value: number) {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return String(value);
}
function formatTime(value: string) {
  return new Date(value).toLocaleString('zh-CN', { hour12: false });
}
function accountName(account: TokenUsageAccount) {
  return account.nickname || account.username || '未知账号';
}
function accountMeta(account: TokenUsageAccount) {
  const username = account.username ? `@${account.username}` : '';
  if (account.nickname && account.nickname !== account.username) {
    return [username, account.userUuid].filter(Boolean).join(' · ');
  }
  return account.userUuid;
}
function conversationPage(userUuid: string) {
  if (!conversationPages.value[userUuid]) {
    conversationPages.value[userUuid] = {
      current: 1,
      size: 10,
      total: 0,
      keyword: '',
      loading: false,
      loaded: false,
      conversations: []
    };
  }
  return conversationPages.value[userUuid];
}
function createOptions() {
  const color = themeStore.themeColors.primary;
  return {
    color: [color],
    grid: { top: 24, right: 22, bottom: 16, left: 18, containLabel: true },
    tooltip: { trigger: 'axis' as const },
    xAxis: {
      type: 'category' as const,
      boundaryGap: false,
      data: points.value.map(point => point.period),
      axisTick: { show: false },
      axisLabel: { hideOverlap: true }
    },
    yAxis: {
      type: 'value' as const,
      minInterval: 1,
      axisLabel: { formatter: (value: number) => formatCompactToken(value) },
      splitLine: { lineStyle: { type: 'dashed' as const, opacity: 0.45 } }
    },
    series: [
      {
        name: 'Token 消耗',
        type: 'line' as const,
        smooth: true,
        showSymbol: false,
        data: points.value.map(point => point.tokens),
        lineStyle: { width: 3 },
        areaStyle: { opacity: 0.12 },
        emphasis: { focus: 'series' as const }
      }
    ]
  };
}
const { domRef, updateOptions } = useEcharts(createOptions, {
  onRender: chart => {
    if (trendLoading.value) chart.showLoading({ color: themeStore.themeColors.primary });
  },
  onUpdated: chart => {
    if (trendLoading.value) chart.showLoading({ color: themeStore.themeColors.primary });
    else chart.hideLoading();
  }
});
async function refreshChart() {
  await nextTick();
  await updateOptions(() => createOptions());
}
async function loadDetails() {
  detailsLoading.value = true;
  const { data, error } = await GetTokenUsageDetails({
    current: accountPage.value,
    size: accountPageSize.value,
    username: accountKeyword.value.trim() || undefined
  });
  if (error) {
    accounts.value = [];
    accountTotal.value = 0;
    tokenSummary.value = emptySummary();
  } else {
    accounts.value = data.accounts;
    accountTotal.value = data.total;
    tokenSummary.value = data.summary;
    accounts.value.forEach(account => conversationPage(account.userUuid));
  }
  detailsLoading.value = false;
}
async function searchAccounts() {
  accountPage.value = 1;
  expandedAccountNames.value = [];
  await loadDetails();
}
async function changeAccountPageSize(size: number) {
  accountPageSize.value = size;
  accountPage.value = 1;
  expandedAccountNames.value = [];
  await loadDetails();
}
async function loadConversations(userUuid: string) {
  const state = conversationPage(userUuid);
  state.loading = true;
  const { data, error } = await GetTokenUsageConversations({
    current: state.current,
    size: state.size,
    userUuid,
    conversationId: state.keyword.trim() || undefined
  });
  if (error) {
    state.conversations = [];
    state.total = 0;
  } else {
    state.conversations = data.conversations;
    state.total = data.total;
  }
  state.loaded = true;
  state.loading = false;
}
async function searchConversations(userUuid: string) {
  conversationPage(userUuid).current = 1;
  await loadConversations(userUuid);
}
async function changeConversationPageSize(userUuid: string, size: number) {
  const state = conversationPage(userUuid);
  state.size = size;
  state.current = 1;
  await loadConversations(userUuid);
}
function handleAccountExpanded(names: Array<string | number> | string | number | null) {
  let normalized: string[] = [];
  if (Array.isArray(names)) normalized = names.map(String);
  else if (names !== null) normalized = [String(names)];
  expandedAccountNames.value = normalized;
  normalized.forEach(userUuid => {
    const state = conversationPage(userUuid);
    if (!state.loaded && !state.loading) loadConversations(userUuid);
  });
}
async function loadTrend() {
  requestSequence += 1;
  const sequence = requestSequence;
  trendLoading.value = true;
  await refreshChart();
  const { data, error } = await GetTokenUsageTrend(dimension.value);
  if (sequence !== requestSequence) return;
  points.value = error ? [] : data.points;
  trendLoading.value = false;
  await refreshChart();
}
onMounted(loadDetails);
watch(dimension, loadTrend, { immediate: true });
</script>

<template>
  <div class="token-page">
    <NCard title="Token 概览" :bordered="false" size="small" class="summary-card card-wrapper">
      <NSpin :show="detailsLoading">
        <div class="metric-grid">
          <div
            v-for="metric in tokenMetrics"
            :key="metric.label"
            class="metric-card"
            :class="`metric-card--${metric.tone}`"
          >
            <div class="metric-head">
              <div class="metric-icon"><SvgIcon :icon="metric.icon" /></div>
              <span class="metric-label">{{ metric.label }}</span>
            </div>
            <div class="metric-value">{{ formatToken(metric.value) }}</div>
            <div class="metric-unit">Token</div>
          </div>
        </div>
      </NSpin>
    </NCard>

    <NCard title="Token 消耗趋势" :bordered="false" size="small" class="chart-card card-wrapper">
      <template #header-extra>
        <NRadioGroup v-model:value="dimension" size="small">
          <NRadioButton value="day" label="按天" />
          <NRadioButton value="month" label="按月" />
        </NRadioGroup>
      </template>
      <div ref="domRef" class="chart-canvas"></div>
    </NCard>

    <NCard title="用量明细" :bordered="false" size="small" class="details-card card-wrapper">
      <template #header-extra>
        <NInputGroup class="account-search">
          <NInput
            v-model:value="accountKeyword"
            clearable
            placeholder="输入用户名查询"
            @keyup.enter="searchAccounts"
            @clear="searchAccounts"
          >
            <template #prefix><SvgIcon icon="carbon:search" /></template>
          </NInput>
          <NButton type="primary" @click="searchAccounts">查询</NButton>
        </NInputGroup>
      </template>
      <p class="details-tip">账号分页展示，展开账号后按需加载对话和 Turn</p>

      <NSpin :show="detailsLoading">
        <NEmpty v-if="!accounts.length" description="暂无匹配的 Token 用量记录" class="empty-state" />
        <NCollapse
          v-else
          :expanded-names="expandedAccountNames"
          class="account-list"
          @update:expanded-names="handleAccountExpanded"
        >
          <NCollapseItem v-for="account in accounts" :key="account.userUuid" :name="account.userUuid">
            <template #header>
              <div class="account-header">
                <div class="account-avatar"><SvgIcon icon="carbon:user-avatar" /></div>
                <div class="account-name">
                  <strong>{{ accountName(account) }}</strong>
                  <span>{{ accountMeta(account) }}</span>
                </div>
                <div class="account-stats">
                  <div>
                    <span>对话</span>
                    <b>{{ account.conversationCount }}</b>
                  </div>
                  <div>
                    <span>Turn</span>
                    <b>{{ account.turnCount }}</b>
                  </div>
                  <div class="account-token">
                    <span>总 Token</span>
                    <strong>{{ formatToken(account.totalTokens) }}</strong>
                  </div>
                </div>
              </div>
            </template>

            <div v-if="conversationPages[account.userUuid]" class="conversation-panel">
              <div class="conversation-toolbar">
                <NInputGroup class="conversation-search">
                  <NInput
                    v-model:value="conversationPages[account.userUuid].keyword"
                    clearable
                    size="small"
                    placeholder="输入 Conversation ID 查询"
                    @keyup.enter="searchConversations(account.userUuid)"
                    @clear="searchConversations(account.userUuid)"
                  >
                    <template #prefix><SvgIcon icon="carbon:search" /></template>
                  </NInput>
                  <NButton size="small" secondary type="primary" @click="searchConversations(account.userUuid)">
                    查询
                  </NButton>
                </NInputGroup>
                <span>共 {{ conversationPages[account.userUuid].total }} 个对话</span>
              </div>

              <NSpin :show="conversationPages[account.userUuid].loading">
                <NEmpty
                  v-if="
                    conversationPages[account.userUuid].loaded &&
                    !conversationPages[account.userUuid].conversations.length
                  "
                  description="暂无匹配的对话"
                  class="conversation-empty"
                />
                <NCollapse v-else accordion class="conversation-list">
                  <NCollapseItem
                    v-for="conversation in conversationPages[account.userUuid].conversations"
                    :key="conversation.conversationId"
                    :name="conversation.conversationId"
                  >
                    <template #header>
                      <div class="conversation-header">
                        <div class="conversation-main">
                          <div class="conversation-icon"><SvgIcon icon="carbon:chat" /></div>
                          <div class="conversation-info">
                            <code>{{ conversation.conversationId }}</code>
                            <span>最近使用 {{ formatTime(conversation.lastUsedAt) }}</span>
                          </div>
                        </div>
                        <div class="conversation-stats">
                          <span>
                            <b>{{ conversation.turnCount }}</b>
                            Turns
                          </span>
                          <div>
                            <small>总 Token</small>
                            <strong>{{ formatToken(conversation.totalTokens) }}</strong>
                          </div>
                        </div>
                      </div>
                    </template>
                    <div class="turn-table-wrap">
                      <NTable striped size="small" :single-line="false" class="turn-table">
                        <thead>
                          <tr>
                            <th class="turn-id-col">Turn ID</th>
                            <th class="time-col">时间</th>
                            <th class="number">输入</th>
                            <th class="number">缓存输入</th>
                            <th class="number">缓存写入</th>
                            <th class="number">输出</th>
                            <th class="number">推理输出</th>
                            <th class="number total-col">总 Token</th>
                            <th class="number context-col">上下文窗口</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="turn in conversation.turns" :key="turn.turnId">
                            <td class="turn-id-cell">
                              <code :title="turn.turnId">{{ turn.turnId }}</code>
                            </td>
                            <td class="time-cell">{{ formatTime(turn.updatedAt) }}</td>
                            <td class="number">{{ formatToken(turn.inputTokens) }}</td>
                            <td class="number">{{ formatToken(turn.cachedInputTokens) }}</td>
                            <td class="number">{{ formatToken(turn.cacheWriteInputTokens) }}</td>
                            <td class="number">{{ formatToken(turn.outputTokens) }}</td>
                            <td class="number">{{ formatToken(turn.reasoningOutputTokens) }}</td>
                            <td class="number total-cell">{{ formatToken(turn.totalTokens) }}</td>
                            <td class="number context-cell">
                              {{ turn.modelContextWindow ? formatToken(turn.modelContextWindow) : '-' }}
                            </td>
                          </tr>
                        </tbody>
                      </NTable>
                    </div>
                  </NCollapseItem>
                </NCollapse>
                <div
                  v-if="conversationPages[account.userUuid].total > 0"
                  class="conversation-pagination pagination-row"
                >
                  <NPagination
                    v-model:page="conversationPages[account.userUuid].current"
                    :page-size="conversationPages[account.userUuid].size"
                    :item-count="conversationPages[account.userUuid].total"
                    :page-sizes="[5, 10, 20, 50]"
                    show-size-picker
                    @update:page="loadConversations(account.userUuid)"
                    @update:page-size="size => changeConversationPageSize(account.userUuid, size)"
                  />
                </div>
              </NSpin>
            </div>
          </NCollapseItem>
        </NCollapse>

        <div v-if="accountTotal > 0" class="pagination-row account-pagination">
          <span>共 {{ accountTotal }} 个账号</span>
          <NPagination
            v-model:page="accountPage"
            :page-size="accountPageSize"
            :item-count="accountTotal"
            :page-sizes="[5, 10, 20, 50]"
            show-size-picker
            @update:page="loadDetails"
            @update:page-size="changeAccountPageSize"
          />
        </div>
      </NSpin>
    </NCard>
  </div>
</template>

<style scoped>
.token-page {
  display: flex;
  min-height: 100%;
  flex-direction: column;
  gap: 16px;
}

.card-wrapper {
  border: 1px solid rgb(232 235 240);
  border-radius: 10px;
}

.summary-card :deep(.n-card__content),
.details-card :deep(.n-card__content) {
  padding: 16px;
}

.chart-card :deep(.n-card__content) {
  padding: 10px 16px 14px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 12px;
}

.metric-card {
  min-width: 0;
  padding: 15px 16px 14px;
  border: 1px solid rgb(229 231 235);
  border-radius: 9px;
  background: #fff;
}

.metric-head {
  display: flex;
  align-items: center;
  gap: 9px;
}

.metric-icon {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 8px;
  font-size: 17px;
}

.metric-label {
  color: rgb(100 116 139);
  font-size: 13px;
  font-weight: 600;
}

.metric-value {
  overflow: hidden;
  margin-top: 13px;
  color: rgb(31 41 55);
  font-size: clamp(18px, 1.55vw, 24px);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1.15;
  text-overflow: ellipsis;
}

.metric-unit {
  margin-top: 4px;
  color: rgb(148 163 184);
  font-size: 11px;
}

.metric-card--blue .metric-icon {
  color: #3478f6;
  background: #edf4ff;
}
.metric-card--cyan .metric-icon {
  color: #0891b2;
  background: #ecfeff;
}
.metric-card--orange .metric-icon {
  color: #e98221;
  background: #fff7ed;
}
.metric-card--green .metric-icon {
  color: #16a36a;
  background: #ecfdf5;
}
.metric-card--violet .metric-icon {
  color: #7c5ce5;
  background: #f3f0ff;
}
.metric-card--primary {
  border-color: rgb(99 102 241 / 25%);
  background: rgb(99 102 241 / 3%);
}
.metric-card--primary .metric-icon {
  color: #6366f1;
  background: #eeefff;
}
.metric-card--primary .metric-value {
  color: #6366f1;
}

.chart-canvas {
  width: 100%;
  height: 360px;
}

.account-search {
  width: min(360px, 40vw);
}

.details-tip {
  margin: -4px 0 14px;
  color: rgb(148 163 184);
  font-size: 12px;
}

.empty-state {
  padding: 48px 0;
}

.account-list {
  padding-top: 1px;
}

.account-list :deep(> .n-collapse-item) {
  overflow: hidden;
  margin-bottom: 10px;
  border: 1px solid rgb(226 232 240);
  border-radius: 10px;
  background: #fff;
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}

.account-list :deep(> .n-collapse-item:hover) {
  border-color: rgb(199 210 254);
}

.account-list :deep(> .n-collapse-item.n-collapse-item--active) {
  border-color: rgb(199 210 254);
  box-shadow: 0 4px 16px rgb(15 23 42 / 5%);
}

.account-list :deep(> .n-collapse-item > .n-collapse-item__header) {
  min-height: 76px;
  padding: 0 18px;
  background: #fff;
}

.account-list :deep(> .n-collapse-item > .n-collapse-item__header:hover) {
  background: rgb(248 250 252);
}

.account-list :deep(> .n-collapse-item > .n-collapse-item__content-wrapper > .n-collapse-item__content-inner) {
  padding: 0;
}

.account-header {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  gap: 12px;
  padding-right: 4px;
}

.account-avatar {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 9px;
  color: #6366f1;
  background: #eeefff;
  font-size: 21px;
}

.account-name {
  display: flex;
  min-width: 180px;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.account-name strong {
  overflow: hidden;
  color: rgb(30 41 59);
  font-size: 14px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-name span {
  overflow: hidden;
  color: rgb(148 163 184);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-stats {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: 86px 86px 170px;
}

.account-stats > div {
  display: flex;
  min-height: 38px;
  align-items: flex-end;
  justify-content: center;
  flex-direction: column;
  padding: 0 18px;
  border-left: 1px solid rgb(241 245 249);
}

.account-stats span {
  color: rgb(148 163 184);
  font-size: 11px;
  line-height: 1.2;
}

.account-stats b,
.account-stats strong {
  margin-top: 3px;
  color: rgb(51 65 85);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.account-stats .account-token strong {
  color: #6366f1;
  font-size: 14px;
}

.conversation-panel {
  padding: 14px 16px 16px;
  border-top: 1px solid rgb(241 245 249);
  background: rgb(248 250 252);
}

.conversation-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.conversation-toolbar > span {
  flex: 0 0 auto;
  color: rgb(148 163 184);
  font-size: 12px;
}

.conversation-search {
  width: min(360px, 48vw);
}

.conversation-empty {
  padding: 30px 0;
}

.conversation-list {
  overflow: hidden;
  border: 1px solid rgb(226 232 240);
  border-radius: 9px;
  background: #fff;
}

.conversation-list :deep(> .n-collapse-item) {
  margin: 0;
  border-top: 1px solid rgb(241 245 249);
}

.conversation-list :deep(> .n-collapse-item:first-child) {
  border-top: 0;
}

.conversation-list :deep(> .n-collapse-item > .n-collapse-item__header) {
  min-height: 66px;
  padding: 0 16px;
  background: #fff;
}

.conversation-list :deep(> .n-collapse-item > .n-collapse-item__header:hover),
.conversation-list :deep(> .n-collapse-item.n-collapse-item--active > .n-collapse-item__header) {
  background: rgb(248 250 252);
}

.conversation-list :deep(> .n-collapse-item > .n-collapse-item__content-wrapper > .n-collapse-item__content-inner) {
  padding: 0 12px 12px;
  background: rgb(248 250 252);
}

.conversation-header {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding-right: 2px;
}

.conversation-main {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 10px;
}

.conversation-icon {
  display: grid;
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 7px;
  color: rgb(100 116 139);
  background: rgb(241 245 249);
  font-size: 16px;
}

.conversation-info {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.conversation-info code {
  overflow: hidden;
  color: rgb(51 65 85);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conversation-info span {
  color: rgb(148 163 184);
  font-size: 11px;
}

.conversation-stats {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: 90px 150px;
  align-items: center;
}

.conversation-stats > span {
  color: rgb(100 116 139);
  font-size: 12px;
  text-align: right;
}

.conversation-stats > span b {
  color: rgb(51 65 85);
  font-variant-numeric: tabular-nums;
}

.conversation-stats > div {
  display: flex;
  align-items: flex-end;
  flex-direction: column;
  margin-left: 16px;
  padding-left: 16px;
  border-left: 1px solid rgb(226 232 240);
}

.conversation-stats small {
  color: rgb(148 163 184);
  font-size: 10px;
}

.conversation-stats strong {
  margin-top: 2px;
  color: #6366f1;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.turn-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(226 232 240);
  border-radius: 8px;
  background: #fff;
}

.turn-table {
  min-width: 1160px;
}

.turn-table :deep(th) {
  padding: 10px 12px;
  color: rgb(100 116 139);
  background: rgb(248 250 252);
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.turn-table :deep(td) {
  padding: 11px 12px;
  color: rgb(71 85 105);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.turn-table :deep(tbody tr:hover td) {
  background: rgb(248 250 252);
}

.turn-id-col,
.turn-id-cell {
  width: 260px;
  min-width: 260px;
}

.turn-id-cell code {
  display: block;
  overflow: hidden;
  color: rgb(71 85 105);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.time-col,
.time-cell {
  width: 158px;
  min-width: 158px;
  white-space: nowrap;
}

.number {
  min-width: 88px;
  text-align: right;
  white-space: nowrap;
}

.total-col,
.total-cell {
  min-width: 104px;
}

.context-col,
.context-cell {
  min-width: 112px;
}

.total-cell {
  color: #6366f1 !important;
  font-weight: 700;
}

.pagination-row {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  margin-top: 14px;
}

.conversation-pagination {
  padding: 0 2px;
}

.account-pagination {
  justify-content: space-between;
  padding-top: 4px;
}

.account-pagination > span {
  color: rgb(148 163 184);
  font-size: 12px;
}

:global(.dark) .card-wrapper,
:global(.dark) .metric-card,
:global(.dark) .account-list :deep(> .n-collapse-item),
:global(.dark) .account-list :deep(> .n-collapse-item > .n-collapse-item__header),
:global(.dark) .conversation-list,
:global(.dark) .conversation-list :deep(> .n-collapse-item > .n-collapse-item__header),
:global(.dark) .turn-table-wrap {
  border-color: rgb(55 65 81);
  background: #18181c;
}

:global(.dark) .account-list :deep(> .n-collapse-item > .n-collapse-item__header:hover),
:global(.dark) .conversation-panel,
:global(.dark) .conversation-list :deep(> .n-collapse-item > .n-collapse-item__header:hover),
:global(.dark) .conversation-list :deep(> .n-collapse-item.n-collapse-item--active > .n-collapse-item__header),
:global(.dark)
  .conversation-list
  :deep(> .n-collapse-item > .n-collapse-item__content-wrapper > .n-collapse-item__content-inner),
:global(.dark) .turn-table :deep(th),
:global(.dark) .turn-table :deep(tbody tr:hover td) {
  background: #202024;
}

:global(.dark) .account-name strong,
:global(.dark) .account-stats b,
:global(.dark) .conversation-info code,
:global(.dark) .conversation-stats > span b,
:global(.dark) .metric-value,
:global(.dark) .turn-table :deep(td),
:global(.dark) .turn-id-cell code {
  color: rgb(226 232 240);
}

:global(.dark) .metric-card--primary {
  background: rgb(99 102 241 / 8%);
}

@media (max-width: 1280px) {
  .metric-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .account-stats {
    grid-template-columns: 72px 72px 150px;
  }
  .account-stats > div {
    padding: 0 12px;
  }
}

@media (max-width: 800px) {
  .account-search,
  .conversation-search {
    width: 100%;
  }

  .account-header,
  .conversation-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .account-stats {
    width: 100%;
    grid-template-columns: repeat(3, 1fr);
  }

  .account-stats > div {
    align-items: flex-start;
    padding: 8px 10px 0 0;
    border-left: 0;
  }

  .conversation-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .conversation-stats {
    width: 100%;
    grid-template-columns: 1fr 1fr;
  }

  .conversation-stats > span {
    text-align: left;
  }
  .conversation-stats > div {
    align-items: flex-start;
  }

  .account-pagination {
    align-items: flex-end;
    flex-direction: column;
    gap: 10px;
  }
}

@media (max-width: 640px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .chart-canvas {
    height: 300px;
  }
}
</style>
