<script setup lang="ts">
import ConversationMetrics from './modules/conversation-metrics.vue';
import ConversationTrendChart from './modules/conversation-trend-chart.vue';
import TokenUsageChart from './modules/token-usage-chart.vue';
</script>

<template>
  <div class="home-page">
    <ConversationMetrics />
    <ConversationTrendChart />
    <TokenUsageChart />
  </div>
</template>

<style scoped>
.home-page {
  display: grid;
  height: 100%;
  min-height: 0;
  grid-template-rows: auto minmax(300px, 1fr) minmax(300px, 1fr);
  gap: 16px;
}

@media (max-width: 640px) {
  .home-page {
    grid-template-rows: auto repeat(2, minmax(360px, auto));
  }
}
</style>
