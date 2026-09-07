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
import { $t } from '@/locales';
import { useAppStore } from '@/store/modules/app';
import { useThemeStore } from '@/store/modules/theme';
import { formatCompactNumber } from '@/utils/common';

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
const appStore = useAppStore();
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
  {
    label: $t('page.agentTokenUsage.input'),
    value: tokenSummary.value.inputTokens,
    icon: 'carbon:download',
    tone: 'blue'
  },
  {
    label: $t('page.agentTokenUsage.cachedInput'),
    value: tokenSummary.value.cachedInputTokens,
    icon: 'carbon:data-base',
    tone: 'cyan'
  },
  {
    label: $t('page.agentTokenUsage.cacheWrite'),
    value: tokenSummary.value.cacheWriteInputTokens,
    icon: 'carbon:save',
    tone: 'orange'
  },
  {
    label: $t('page.agentTokenUsage.output'),
    value: tokenSummary.value.outputTokens,
    icon: 'carbon:upload',
    tone: 'green'
  },
  {
    label: $t('page.agentTokenUsage.reasoningOutput'),
    value: tokenSummary.value.reasoningOutputTokens,
    icon: 'carbon:idea',
    tone: 'violet'
  },
  {
    label: $t('page.agentTokenUsage.totalToken'),
    value: tokenSummary.value.totalTokens,
    icon: 'carbon:meter',
    tone: 'primary'
  }
]);

function formatToken(value: number) {
  return value.toLocaleString(appStore.locale);
}
function formatCompactToken(value: number) {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return String(value);
}
function formatTime(value: string) {
  return new Date(value).toLocaleString(appStore.locale, { hour12: false });
}
function accountName(account: TokenUsageAccount) {
  return account.username || $t('page.agentTokenUsage.unknownAccount');
}
function accountMeta(account: TokenUsageAccount) {
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
        name: $t('page.agentTokenUsage.seriesName'),
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
watch(() => appStore.locale, refreshChart);
</script>

<template>
  <div class="token-page">
    <NCard
      :title="$t('page.agentTokenUsage.summaryTitle')"
      :bordered="false"
      size="small"
      class="summary-card card-wrapper"
    >
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
            <div class="metric-value" :title="formatToken(metric.value)">
              {{ formatCompactNumber(metric.value, appStore.locale) }}
            </div>
            <div class="metric-unit">{{ $t('page.agentTokenUsage.tokenUnit') }}</div>
          </div>
        </div>
      </NSpin>
    </NCard>

    <NCard
      :title="$t('page.agentTokenUsage.trendTitle')"
      :bordered="false"
      size="small"
      class="chart-card card-wrapper"
    >
      <template #header-extra>
        <NRadioGroup v-model:value="dimension" size="small">
          <NRadioButton value="day" :label="$t('page.agentTokenUsage.byDay')" />
          <NRadioButton value="month" :label="$t('page.agentTokenUsage.byMonth')" />
        </NRadioGroup>
      </template>
      <div ref="domRef" class="chart-canvas"></div>
    </NCard>

    <NCard
      :title="$t('page.agentTokenUsage.detailsTitle')"
      :bordered="false"
      size="small"
      class="details-card card-wrapper"
    >
      <template #header-extra>
        <NInputGroup class="account-search">
          <NInput
            v-model:value="accountKeyword"
            clearable
            :placeholder="$t('page.agentTokenUsage.accountSearchPlaceholder')"
            @keyup.enter="searchAccounts"
            @clear="searchAccounts"
          >
            <template #prefix><SvgIcon icon="carbon:search" /></template>
          </NInput>
          <NButton type="primary" @click="searchAccounts">{{ $t('common.search') }}</NButton>
        </NInputGroup>
      </template>
      <p class="details-tip">{{ $t('page.agentTokenUsage.detailsTip') }}</p>

      <NSpin :show="detailsLoading">
        <NEmpty v-if="!accounts.length" :description="$t('page.agentTokenUsage.emptyRecords')" class="empty-state" />
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
                    <span>{{ $t('page.agentTokenUsage.conversation') }}</span>
                    <b>{{ account.conversationCount }}</b>
                  </div>
                  <div>
                    <span>{{ $t('page.agentTokenUsage.turn') }}</span>
                    <b>{{ account.turnCount }}</b>
                  </div>
                  <div class="account-token">
                    <span>{{ $t('page.agentTokenUsage.totalToken') }}</span>
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
                    :placeholder="$t('page.agentTokenUsage.conversationSearchPlaceholder')"
                    @keyup.enter="searchConversations(account.userUuid)"
                    @clear="searchConversations(account.userUuid)"
                  >
                    <template #prefix><SvgIcon icon="carbon:search" /></template>
                  </NInput>
                  <NButton size="small" secondary type="primary" @click="searchConversations(account.userUuid)">
                    {{ $t('common.search') }}
                  </NButton>
                </NInputGroup>
                <span>
                  {{
                    $t('page.agentTokenUsage.conversationCount', {
                      count: conversationPages[account.userUuid].total
                    })
                  }}
                </span>
              </div>

              <NSpin :show="conversationPages[account.userUuid].loading">
                <NEmpty
                  v-if="
                    conversationPages[account.userUuid].loaded &&
                    !conversationPages[account.userUuid].conversations.length
                  "
                  :description="$t('page.agentTokenUsage.emptyConversations')"
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
                            <span>
                              {{
                                $t('page.agentTokenUsage.lastUsed', {
                                  time: formatTime(conversation.lastUsedAt)
                                })
                              }}
                            </span>
                          </div>
                        </div>
                        <div class="conversation-stats">
                          <span>
                            <b>{{ conversation.turnCount }}</b>
                            {{ $t('page.agentTokenUsage.turns') }}
                          </span>
                          <div>
                            <small>{{ $t('page.agentTokenUsage.totalToken') }}</small>
                            <strong>{{ formatToken(conversation.totalTokens) }}</strong>
                          </div>
                        </div>
                      </div>
                    </template>
                    <div class="turn-table-wrap">
                      <NTable striped size="small" :single-line="false" class="turn-table">
                        <thead>
                          <tr>
                            <th class="turn-id-col">{{ $t('page.agentTokenUsage.turnId') }}</th>
                            <th class="time-col">{{ $t('page.agentTokenUsage.time') }}</th>
                            <th class="number">{{ $t('page.agentTokenUsage.input') }}</th>
                            <th class="number">{{ $t('page.agentTokenUsage.cachedInput') }}</th>
                            <th class="number">{{ $t('page.agentTokenUsage.cacheWrite') }}</th>
                            <th class="number">{{ $t('page.agentTokenUsage.output') }}</th>
                            <th class="number">{{ $t('page.agentTokenUsage.reasoningOutput') }}</th>
                            <th class="number total-col">{{ $t('page.agentTokenUsage.totalToken') }}</th>
                            <th class="number context-col">{{ $t('page.agentTokenUsage.contextWindow') }}</th>
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
                    :page-slot="appStore.isMobile ? 3 : 9"
                    :show-size-picker="!appStore.isMobile"
                    @update:page="loadConversations(account.userUuid)"
                    @update:page-size="size => changeConversationPageSize(account.userUuid, size)"
                  />
                </div>
              </NSpin>
            </div>
          </NCollapseItem>
        </NCollapse>

        <div v-if="accountTotal > 0" class="pagination-row account-pagination">
          <span>{{ $t('page.agentTokenUsage.accountCount', { count: accountTotal }) }}</span>
          <NPagination
            v-model:page="accountPage"
            :page-size="accountPageSize"
            :item-count="accountTotal"
            :page-sizes="[5, 10, 20, 50]"
            :page-slot="appStore.isMobile ? 3 : 9"
            :show-size-picker="!appStore.isMobile"
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
  --token-surface: rgb(var(--container-bg-color));
  --token-text: rgb(var(--base-text-color));
  --token-text-secondary: color-mix(in srgb, var(--token-text) 76%, transparent);
  --token-muted: color-mix(in srgb, var(--token-text) 52%, transparent);
  --token-border: color-mix(in srgb, var(--token-text) 15%, transparent);
  --token-divider: color-mix(in srgb, var(--token-text) 9%, transparent);
  --token-soft: color-mix(in srgb, var(--token-text) 5%, var(--token-surface));
  --token-primary: rgb(var(--primary-color));
  --token-primary-soft: color-mix(in srgb, var(--token-primary) 11%, var(--token-surface));

  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 100%;
  overflow-x: hidden;
  flex-direction: column;
  gap: 16px;
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
  border: 1px solid var(--token-border);
  border-radius: 9px;
  background: var(--token-surface);
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
  color: var(--token-text-secondary);
  font-size: 13px;
  font-weight: 600;
}

.metric-value {
  overflow: hidden;
  margin-top: 13px;
  color: var(--token-text);
  font-size: clamp(18px, 1.55vw, 24px);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1.15;
  text-overflow: ellipsis;
}

.metric-unit {
  margin-top: 4px;
  color: var(--token-muted);
  font-size: 11px;
}

.metric-card--blue .metric-icon {
  color: #3478f6;
  background: color-mix(in srgb, #3478f6 12%, var(--token-surface));
}
.metric-card--cyan .metric-icon {
  color: #0891b2;
  background: color-mix(in srgb, #0891b2 12%, var(--token-surface));
}
.metric-card--orange .metric-icon {
  color: #e98221;
  background: color-mix(in srgb, #e98221 12%, var(--token-surface));
}
.metric-card--green .metric-icon {
  color: #16a36a;
  background: color-mix(in srgb, #16a36a 12%, var(--token-surface));
}
.metric-card--violet .metric-icon {
  color: #7c5ce5;
  background: color-mix(in srgb, #7c5ce5 12%, var(--token-surface));
}
.metric-card--primary {
  border-color: color-mix(in srgb, var(--token-primary) 30%, transparent);
  background: color-mix(in srgb, var(--token-primary) 7%, var(--token-surface));
}
.metric-card--primary .metric-icon {
  color: var(--token-primary);
  background: var(--token-primary-soft);
}
.metric-card--primary .metric-value {
  color: var(--token-primary);
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
  color: var(--token-muted);
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
  margin-left: 0;
  margin-bottom: 10px;
  border: 1px solid var(--token-border);
  border-radius: 10px;
  background: var(--token-surface);
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}

.account-list :deep(> .n-collapse-item:hover) {
  border-color: color-mix(in srgb, var(--token-primary) 32%, transparent);
}

.account-list :deep(> .n-collapse-item.n-collapse-item--active) {
  border-color: color-mix(in srgb, var(--token-primary) 38%, transparent);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--token-text) 7%, transparent);
}

.account-list :deep(> .n-collapse-item > .n-collapse-item__header) {
  min-height: 76px;
  padding: 0 18px;
  background: var(--token-surface);
}

.account-list :deep(> .n-collapse-item > .n-collapse-item__header .n-collapse-item__header-main),
.conversation-list :deep(> .n-collapse-item > .n-collapse-item__header .n-collapse-item__header-main) {
  min-width: 0;
}

.account-list :deep(> .n-collapse-item > .n-collapse-item__header:hover) {
  background: var(--token-soft);
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
  color: var(--token-primary);
  background: var(--token-primary-soft);
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
  color: var(--token-text);
  font-size: 14px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-name span {
  overflow: hidden;
  color: var(--token-muted);
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
  border-left: 1px solid var(--token-divider);
}

.account-stats span {
  color: var(--token-muted);
  font-size: 11px;
  line-height: 1.2;
}

.account-stats b,
.account-stats strong {
  margin-top: 3px;
  color: var(--token-text-secondary);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.account-stats .account-token strong {
  color: var(--token-primary);
  font-size: 14px;
}

.conversation-panel {
  padding: 14px 16px 16px;
  border-top: 1px solid var(--token-divider);
  background: var(--token-soft);
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
  color: var(--token-muted);
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
  border: 1px solid var(--token-border);
  border-radius: 9px;
  background: var(--token-surface);
}

.conversation-list :deep(> .n-collapse-item) {
  margin: 0;
  border-top: 1px solid var(--token-divider);
}

.conversation-list :deep(> .n-collapse-item:first-child) {
  border-top: 0;
}

.conversation-list :deep(> .n-collapse-item > .n-collapse-item__header) {
  min-height: 66px;
  padding: 0 16px;
  background: var(--token-surface);
}

.conversation-list :deep(> .n-collapse-item > .n-collapse-item__header:hover),
.conversation-list :deep(> .n-collapse-item.n-collapse-item--active > .n-collapse-item__header) {
  background: var(--token-soft);
}

.conversation-list :deep(> .n-collapse-item > .n-collapse-item__content-wrapper > .n-collapse-item__content-inner) {
  padding: 0 12px 12px;
  background: var(--token-soft);
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
  color: var(--token-text-secondary);
  background: var(--token-soft);
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
  color: var(--token-text-secondary);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conversation-info span {
  color: var(--token-muted);
  font-size: 11px;
}

.conversation-stats {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: 90px 150px;
  align-items: center;
}

.conversation-stats > span {
  color: var(--token-text-secondary);
  font-size: 12px;
  text-align: right;
}

.conversation-stats > span b {
  color: var(--token-text);
  font-variant-numeric: tabular-nums;
}

.conversation-stats > div {
  display: flex;
  align-items: flex-end;
  flex-direction: column;
  margin-left: 16px;
  padding-left: 16px;
  border-left: 1px solid var(--token-border);
}

.conversation-stats small {
  color: var(--token-muted);
  font-size: 10px;
}

.conversation-stats strong {
  margin-top: 2px;
  color: var(--token-primary);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.turn-table-wrap {
  overflow-x: auto;
  border: 1px solid var(--token-border);
  border-radius: 8px;
  background: var(--token-surface);
}

.turn-table {
  min-width: 1160px;
}

.turn-table :deep(th) {
  padding: 10px 12px;
  color: var(--token-text-secondary);
  background: var(--token-soft);
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.turn-table :deep(td) {
  padding: 11px 12px;
  color: var(--token-text-secondary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.turn-table :deep(tbody tr:hover td) {
  background: var(--token-soft);
}

.turn-id-col,
.turn-id-cell {
  width: 260px;
  min-width: 260px;
}

.turn-id-cell code {
  display: block;
  overflow: hidden;
  color: var(--token-text-secondary);
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
  color: var(--token-primary) !important;
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
  color: var(--token-muted);
  font-size: 12px;
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
  .chart-card :deep(.n-card-header),
  .details-card :deep(.n-card-header) {
    align-items: stretch;
    flex-wrap: wrap;
    gap: 10px;
  }

  .chart-card :deep(.n-card-header__main),
  .details-card :deep(.n-card-header__main),
  .chart-card :deep(.n-card-header__extra),
  .details-card :deep(.n-card-header__extra) {
    width: 100%;
  }

  .chart-card :deep(.n-card-header__extra),
  .details-card :deep(.n-card-header__extra) {
    margin-left: 0;
  }

  .chart-card :deep(.n-radio-group) {
    display: flex;
  }

  .chart-card :deep(.n-radio-button) {
    flex: 1;
  }

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

  .pagination-row :deep(.n-pagination) {
    max-width: 100%;
  }
}

@media (max-width: 640px) {
  .summary-card :deep(.n-card-header),
  .chart-card :deep(.n-card-header),
  .details-card :deep(.n-card-header) {
    padding-right: 14px;
    padding-left: 14px;
  }

  .summary-card :deep(.n-card__content),
  .details-card :deep(.n-card__content) {
    padding: 12px;
  }

  .chart-card :deep(.n-card__content) {
    padding: 6px 8px 12px;
  }

  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .metric-card {
    padding: 12px;
  }

  .metric-head {
    align-items: flex-start;
  }

  .metric-value {
    margin-top: 10px;
  }

  .chart-canvas {
    height: 270px;
  }

  .details-tip {
    margin-top: 0;
  }

  .account-list :deep(> .n-collapse-item > .n-collapse-item__header) {
    padding: 14px 12px 16px;
  }

  .conversation-list :deep(> .n-collapse-item > .n-collapse-item__header) {
    padding: 12px 10px;
  }

  .account-list :deep(> .n-collapse-item > .n-collapse-item__header .n-collapse-item__header-main) {
    align-items: flex-start;
  }

  .account-list :deep(> .n-collapse-item > .n-collapse-item__header .n-collapse-item-arrow) {
    margin-top: 9px;
  }

  .account-header {
    display: grid;
    grid-template-columns: 38px minmax(0, 1fr);
    align-items: center;
    gap: 12px;
    padding-right: 0;
  }

  .account-name {
    width: auto;
    min-width: 0;
  }

  .account-stats {
    grid-column: 1 / -1;
    padding: 11px 12px;
    border-radius: 8px;
    background: var(--token-soft);
  }

  .account-stats > div {
    min-width: 0;
    min-height: 0;
    padding: 0 8px;
  }

  .account-stats > div:first-child {
    padding-left: 0;
  }

  .account-stats > div + div {
    border-left: 1px solid var(--token-divider);
  }

  .account-stats b,
  .account-stats strong {
    overflow: hidden;
    max-width: 100%;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .conversation-panel {
    padding: 12px 10px;
  }

  .conversation-list :deep(> .n-collapse-item > .n-collapse-item__content-wrapper > .n-collapse-item__content-inner) {
    padding: 0 8px 8px;
  }

  .conversation-stats > div {
    margin-left: 8px;
    padding-left: 8px;
  }
}
</style>
