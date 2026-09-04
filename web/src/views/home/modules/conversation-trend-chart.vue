<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import { type ConversationTrendDimension, type ConversationTrendPoint, GetConversationTrend } from '@/service/api';
import { useEcharts } from '@/hooks/common/echarts';
import { useThemeStore } from '@/store/modules/theme';

defineOptions({
  name: 'ConversationTrendChart'
});

const themeStore = useThemeStore();
const dimension = ref<ConversationTrendDimension>('day');
const loading = ref(false);
const points = ref<ConversationTrendPoint[]>([]);
let requestSequence = 0;

function createOptions() {
  const conversationColor = themeStore.themeColors.primary;
  const archivedColor = '#64748b';

  return {
    color: [conversationColor, archivedColor],
    grid: { top: 52, right: 22, bottom: 16, left: 18, containLabel: true },
    legend: { top: 4, itemWidth: 20, itemHeight: 3 },
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
      axisLabel: { formatter: (value: number) => String(Math.round(value)) },
      splitLine: { lineStyle: { type: 'dashed' as const, opacity: 0.45 } }
    },
    series: [
      {
        name: '对话数量',
        type: 'line' as const,
        smooth: true,
        showSymbol: false,
        symbolSize: 7,
        data: points.value.map(point => point.totalConversations),
        lineStyle: { width: 3 },
        areaStyle: { opacity: 0.08 },
        emphasis: { focus: 'series' as const }
      },
      {
        name: '归档对话',
        type: 'line' as const,
        smooth: true,
        showSymbol: false,
        symbolSize: 7,
        data: points.value.map(point => point.archivedConversations),
        lineStyle: { width: 3 },
        emphasis: { focus: 'series' as const }
      }
    ]
  };
}

const { domRef, updateOptions } = useEcharts(createOptions, {
  onRender: chart => {
    if (loading.value) chart.showLoading({ color: themeStore.themeColors.primary });
  },
  onUpdated: chart => {
    if (loading.value) chart.showLoading({ color: themeStore.themeColors.primary });
    else chart.hideLoading();
  }
});

async function refreshChart() {
  await nextTick();
  await updateOptions(() => createOptions());
}

async function loadTrend() {
  requestSequence += 1;
  const sequence = requestSequence;
  loading.value = true;
  await refreshChart();

  const { data, error } = await GetConversationTrend(dimension.value);
  if (sequence !== requestSequence) return;

  points.value = error ? [] : data.points;
  loading.value = false;
  await refreshChart();
}

watch(dimension, loadTrend, { immediate: true });
</script>

<template>
  <NCard :bordered="false" size="small" class="conversation-chart card-wrapper">
    <div class="chart-header">
      <div class="chart-title-row">
        <div class="chart-title-icon">
          <SvgIcon icon="carbon:chart-line-data" />
        </div>
        <h2>对话趋势</h2>
      </div>

      <NRadioGroup v-model:value="dimension" size="small">
        <NRadioButton value="day" label="按天" />
        <NRadioButton value="month" label="按月" />
      </NRadioGroup>
    </div>

    <div ref="domRef" class="chart-canvas"></div>
  </NCard>
</template>

<style scoped>
.conversation-chart {
  height: 100%;
}

.conversation-chart :deep(.n-card__content) {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  padding: 18px;
}

.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 8px;
}

.chart-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chart-title-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 8px;
  color: #2f6fed;
  background: rgb(47 111 237 / 12%);
  font-size: 21px;
}

.chart-title-row h2 {
  margin: 0;
  color: var(--n-text-color);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
}

.chart-canvas {
  width: 100%;
  min-height: 0;
  flex: 1;
}

@media (max-width: 640px) {
  .chart-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .chart-canvas {
    min-height: 280px;
  }
}
</style>
