<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { GetHomeOverview } from '@/service/api';
import type { HomeOverview } from '@/service/api';
import { $t } from '@/locales';
import { useAppStore } from '@/store/modules/app';
import { useAuthStore } from '@/store/modules/auth';
import { formatCompactNumber } from '@/utils/common';

defineOptions({ name: 'HomePage' });

const appStore = useAppStore();
const authStore = useAuthStore();

const loading = ref(false);
const overview = ref<HomeOverview | null>(null);

const runtimeLabel = computed(() => {
  if (!overview.value) return $t('page.home.dashboard.unavailable');
  return overview.value.agentRunning ? $t('page.home.dashboard.agentOnline') : $t('page.home.dashboard.agentOffline');
});
const activeModelName = computed(() => overview.value?.modelName || overview.value?.model || '-');

const metrics = computed<Array<{ label: string; value: string; icon: string; tone: string }>>(() => [
  {
    label: $t('page.home.dashboard.agentStatus'),
    value: runtimeLabel.value,
    icon: 'carbon:cloud-service-management',
    tone: overview.value?.agentRunning ? 'green' : 'orange'
  },
  {
    label: $t('page.home.dashboard.currentModel'),
    value: activeModelName.value,
    icon: 'carbon:cube',
    tone: 'violet'
  },
  {
    label: $t('page.home.dashboard.totalTokens'),
    value: overview.value ? formatCompactNumber(overview.value.totalTokens, appStore.locale) : '-',
    icon: 'carbon:meter',
    tone: 'blue'
  },
  {
    label: $t('page.home.dashboard.installedSkills'),
    value: overview.value ? overview.value.skillCount.toLocaleString(appStore.locale) : '-',
    icon: 'carbon:skill-level-basic',
    tone: 'cyan'
  }
]);

async function loadOverview() {
  loading.value = true;
  const { data, error } = await GetHomeOverview();
  if (!error) overview.value = data;
  loading.value = false;
}

onMounted(loadOverview);
</script>

<template>
  <div class="home-page min-h-500px">
    <NCard :bordered="false" size="small" class="hero-card card-wrapper">
      <div class="hero-content">
        <div class="hero-copy">
          <div class="brand-mark">AGENTRAZOR</div>
          <h1>
            {{ $t('page.home.dashboard.welcome', { username: authStore.userInfo.username || 'Admin' }) }}
          </h1>
          <p>{{ $t('page.home.dashboard.description') }}</p>
        </div>

        <div class="hero-visual" aria-hidden="true">
          <div class="orbit orbit--outer"></div>
          <div class="orbit orbit--inner"></div>
          <span class="orbit-node orbit-node--one"></span>
          <span class="orbit-node orbit-node--two"></span>
          <div class="bot-core"><SvgIcon icon="carbon:bot" /></div>
        </div>
      </div>
    </NCard>

    <NSpin :show="loading" class="overview-spin">
      <div class="metric-grid">
        <NCard
          v-for="metric in metrics"
          :key="metric.label"
          :bordered="false"
          size="small"
          class="metric-card card-wrapper"
        >
          <div class="metric-top">
            <div class="metric-icon" :class="`metric-icon--${metric.tone}`">
              <SvgIcon :icon="metric.icon" />
            </div>
            <span>{{ metric.label }}</span>
          </div>
          <strong :title="metric.value">{{ metric.value }}</strong>
        </NCard>
      </div>
    </NSpin>
  </div>
</template>

<style scoped>
.home-page {
  --home-surface: rgb(var(--container-bg-color));
  --home-text: rgb(var(--base-text-color));
  --home-muted: color-mix(in srgb, var(--home-text) 58%, transparent);
  --home-primary: rgb(var(--primary-color));

  display: flex;
  flex-direction: column;
  gap: 16px;
  color: var(--home-text);
}

.hero-card {
  position: relative;
  overflow: hidden;
  background: radial-gradient(
      circle at 80% 10%,
      color-mix(in srgb, var(--home-primary) 13%, transparent),
      transparent 28%
    ),
    linear-gradient(120deg, color-mix(in srgb, var(--home-primary) 7%, var(--home-surface)), var(--home-surface) 62%);
}

.hero-card :deep(.n-card__content) {
  padding: 0;
}

.hero-content {
  display: flex;
  min-height: 236px;
  align-items: center;
  justify-content: space-between;
  gap: 32px;
  padding: 34px 42px;
}

.hero-copy {
  position: relative;
  z-index: 2;
  max-width: 720px;
}

.brand-mark {
  color: var(--home-primary);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.22em;
}

.hero-copy h1 {
  margin: 10px 0 12px;
  color: var(--home-text);
  font-size: clamp(28px, 3vw, 40px);
  font-weight: 750;
  letter-spacing: -0.025em;
  line-height: 1.15;
}

.hero-copy p {
  max-width: 620px;
  margin: 0;
  color: var(--home-muted);
  font-size: 15px;
  line-height: 1.7;
}

.hero-visual {
  position: relative;
  width: 220px;
  height: 180px;
  flex: 0 0 auto;
}

.orbit {
  position: absolute;
  top: 50%;
  left: 50%;
  border: 1px solid color-mix(in srgb, var(--home-primary) 23%, transparent);
  border-radius: 50%;
  transform: translate(-50%, -50%) rotate(-18deg);
}

.orbit--outer {
  width: 206px;
  height: 122px;
}

.orbit--inner {
  width: 150px;
  height: 88px;
  transform: translate(-50%, -50%) rotate(28deg);
}

.orbit-node {
  position: absolute;
  width: 10px;
  height: 10px;
  border: 3px solid var(--home-surface);
  border-radius: 50%;
  background: var(--home-primary);
  box-shadow: 0 4px 14px color-mix(in srgb, var(--home-primary) 36%, transparent);
}

.orbit-node--one {
  top: 34px;
  right: 24px;
}

.orbit-node--two {
  bottom: 29px;
  left: 26px;
}

.bot-core {
  position: absolute;
  top: 50%;
  left: 50%;
  display: grid;
  width: 82px;
  height: 82px;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--home-primary) 18%, transparent);
  border-radius: 24px;
  color: var(--home-primary);
  background: color-mix(in srgb, var(--home-primary) 13%, var(--home-surface));
  box-shadow: 0 18px 46px color-mix(in srgb, var(--home-primary) 17%, transparent);
  font-size: 43px;
  transform: translate(-50%, -50%) rotate(-4deg);
}

.overview-spin :deep(.n-spin-content) {
  display: block;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.metric-card :deep(.n-card__content) {
  padding: 18px 20px;
}

.metric-top {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--home-muted);
  font-size: 13px;
  font-weight: 600;
}

.metric-icon {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 9px;
  font-size: 18px;
}

.metric-icon--green {
  color: #16a36a;
  background: color-mix(in srgb, #16a36a 12%, var(--home-surface));
}

.metric-icon--orange {
  color: #e98221;
  background: color-mix(in srgb, #e98221 12%, var(--home-surface));
}

.metric-icon--violet {
  color: #7c5ce5;
  background: color-mix(in srgb, #7c5ce5 12%, var(--home-surface));
}

.metric-icon--blue {
  color: #3478f6;
  background: color-mix(in srgb, #3478f6 12%, var(--home-surface));
}

.metric-icon--cyan {
  color: #0891b2;
  background: color-mix(in srgb, #0891b2 12%, var(--home-surface));
}

.metric-card strong {
  display: block;
  overflow: hidden;
  margin-top: 16px;
  color: var(--home-text);
  font-size: 22px;
  font-weight: 720;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1280px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .hero-content {
    min-height: 210px;
    padding: 30px;
  }

  .hero-visual {
    width: 180px;
    transform: scale(0.86);
  }
}

@media (max-width: 640px) {
  .hero-content {
    min-height: 0;
    padding: 26px 22px;
  }

  .hero-visual {
    display: none;
  }

  .hero-copy h1 {
    font-size: 27px;
  }

  .metric-grid {
    grid-template-columns: 1fr;
  }

  .metric-card :deep(.n-card__content) {
    padding: 16px 18px;
  }
}
</style>
