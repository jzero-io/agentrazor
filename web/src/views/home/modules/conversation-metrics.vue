<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { GetConversationStats, type ConversationStats } from '@/service/api';

defineOptions({
  name: 'ConversationMetrics'
});

const loading = ref(false);
const stats = ref<ConversationStats | null>(null);

const cards = computed(() => [
  { label: '对话数量', value: stats.value?.totalConversations ?? 0, icon: 'carbon:chat', tone: 'blue' },
  { label: '正在进行中', value: stats.value?.runningConversations ?? 0, icon: 'carbon:in-progress', tone: 'green' },
  { label: '归档对话', value: stats.value?.archivedConversations ?? 0, icon: 'carbon:archive', tone: 'slate' },
  {
    label: 'Token 消耗',
    value: stats.value?.tokenUsageAvailable ? formatToken(stats.value.totalTokens) : '未记录',
    icon: 'carbon:meter',
    tone: 'violet'
  }
]);

function formatToken(value: number) {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return String(value);
}

async function loadStats() {
  loading.value = true;
  const { data, error } = await GetConversationStats();
  if (!error) stats.value = data;
  loading.value = false;
}

onMounted(loadStats);
</script>

<template>
  <NCard :bordered="false" size="small" class="metrics-panel card-wrapper">
    <div class="metrics-header">
      <div class="metrics-title">
        <div class="metrics-title__icon">
          <SvgIcon icon="carbon:dashboard" />
        </div>
        <h2>Agent 指标</h2>
      </div>
    </div>

    <NSpin :show="loading && !stats">
      <NGrid :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
        <NGi v-for="card in cards" :key="card.label" span="24 s:12 m:6">
          <div class="metric-card" :class="`metric-card--${card.tone}`">
            <div class="metric-card__bar"></div>
            <div class="metric-icon">
              <SvgIcon :icon="card.icon" />
            </div>
            <span class="metric-label">{{ card.label }}</span>
            <strong class="metric-value">{{ card.value }}</strong>
          </div>
        </NGi>
      </NGrid>
    </NSpin>
  </NCard>
</template>

<style scoped>
.metrics-panel {
  --metrics-panel-border: var(--n-border-color);
  --metrics-title-color: var(--n-text-color);
  --metrics-card-bg: var(--n-color);
  --metrics-card-border: var(--n-border-color);
  --metrics-card-shadow: none;
}

.dark .metrics-panel {
  --metrics-panel-border: var(--n-border-color);
  --metrics-title-color: var(--n-text-color);
  --metrics-card-bg: var(--n-color);
  --metrics-card-border: var(--n-border-color);
  --metrics-card-shadow: none;
}

.metrics-panel :deep(.n-card__content) {
  padding: 18px;
}

.metrics-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.metrics-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.metrics-title__icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 8px;
  color: #2f6fed;
  background: color-mix(in srgb, #2f6fed 12%, transparent);
  font-size: 21px;
}

.metrics-title h2 {
  margin: 0;
  color: var(--metrics-title-color);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
}

.metric-card {
  position: relative;
  display: grid;
  min-height: 88px;
  grid-template-columns: 44px minmax(0, 1fr) auto;
  align-items: center;
  gap: 14px;
  overflow: hidden;
  padding: 18px;
  border: 1px solid var(--metrics-card-border);
  border-radius: 8px;
  background: var(--metrics-card-bg);
}

.metric-card__bar {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
  background: var(--metric-color);
}

.metric-card--blue { --metric-color: #2f6fed; --metric-bg: rgb(47 111 237 / 12%); }
.metric-card--green { --metric-color: #12956c; --metric-bg: rgb(18 149 108 / 13%); }
.metric-card--slate { --metric-color: #64748b; --metric-bg: rgb(100 116 139 / 13%); }
.metric-card--violet { --metric-color: #7557d8; --metric-bg: rgb(117 87 216 / 13%); }

.metric-icon {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border-radius: 8px;
  color: var(--metric-color);
  background: var(--metric-bg);
  font-size: 24px;
}

.metric-label {
  min-width: 0;
  overflow: hidden;
  color: var(--metrics-title-color);
  font-size: 14px;
  font-weight: 600;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.metric-value {
  min-width: 72px;
  color: var(--metrics-title-color);
  font-size: 30px;
  font-weight: 750;
  line-height: 1;
  text-align: right;
}

@media (max-width: 640px) {
  .metrics-header {
    align-items: flex-start;
  }

  .metric-card {
    grid-template-columns: 44px minmax(0, 1fr);
  }

  .metric-value {
    grid-column: 2;
    min-width: 0;
    text-align: left;
  }
}
</style>
