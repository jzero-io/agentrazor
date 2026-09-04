<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import { GetTokenUsageTrend, type TokenUsageDimension, type TokenUsageTrendPoint } from '@/service/api';
import { useEcharts } from '@/hooks/common/echarts';
import { useThemeStore } from '@/store/modules/theme';

defineOptions({
  name: 'TokenUsageChart'
});

const themeStore = useThemeStore();
const dimension = ref<TokenUsageDimension>('day');
const loading = ref(false);
const points = ref<TokenUsageTrendPoint[]>([]);
let requestSequence = 0;

function formatToken(value: number) {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return String(value);
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
      axisLabel: { formatter: (value: number) => formatToken(value) },
      splitLine: { lineStyle: { type: 'dashed' as const, opacity: 0.45 } }
    },
    series: [
      {
        name: 'Token 消耗',
        type: 'line' as const,
        smooth: true,
        showSymbol: false,
        symbolSize: 7,
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

  const { data, error } = await GetTokenUsageTrend(dimension.value);
  if (sequence !== requestSequence) return;

  points.value = error ? [] : data.points;
  loading.value = false;
  await refreshChart();
}

watch(dimension, loadTrend, { immediate: true });
</script>

<template>
  <NCard :bordered="false" size="small" class="token-chart card-wrapper">
    <div class="chart-header">
      <div>
        <div class="chart-title-row">
          <div class="chart-title-icon">
            <SvgIcon icon="carbon:chart-line" />
          </div>
          <div>
            <h2>Token 消耗趋势</h2>
          </div>
        </div>
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
.token-chart {
  height: 100%;
}

.token-chart :deep(.n-card__content) {
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
  color: #7557d8;
  background: rgb(117 87 216 / 13%);
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
