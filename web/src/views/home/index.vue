<template>
  <div class="home-page">
    <NCard :bordered="false" class="welcome-card card-wrapper">
      <div class="welcome-icon"><SvgIcon icon="carbon:bot" /></div>
      <div class="welcome-content">
        <span>AGENTRAZOR</span>
        <h1>欢迎使用 AgentRazor</h1>
        <p>智能、便捷的 Agent 管理平台</p>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.home-page {
  height: 100%;
  min-height: 0;
}

.welcome-card {
  overflow: hidden;
  background: linear-gradient(110deg, color-mix(in srgb, var(--primary-color) 9%, #fff), #fff 68%);
}

.welcome-card :deep(.n-card__content) {
  display: flex;
  min-height: 132px;
  align-items: center;
  gap: 20px;
  padding: 28px 32px;
}

.welcome-icon {
  display: grid;
  width: 64px;
  height: 64px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 18px;
  color: var(--primary-color);
  background: color-mix(in srgb, var(--primary-color) 14%, transparent);
  font-size: 34px;
}

.welcome-content span {
  color: var(--primary-color);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.18em;
}

.welcome-content h1 {
  margin: 5px 0 6px;
  color: #1f2937;
  font-size: 26px;
  line-height: 1.2;
}

.welcome-content p {
  margin: 0;
  color: #6b7280;
  font-size: 14px;
}

.dark .welcome-card {
  background: linear-gradient(110deg, color-mix(in srgb, var(--primary-color) 12%, #18181c), #18181c 68%);
}

.dark .welcome-content h1 {
  color: #f3f4f6;
}
.dark .welcome-content p {
  color: #9ca3af;
}

@media (max-width: 640px) {
  .welcome-card :deep(.n-card__content) {
    min-height: 118px;
    padding: 22px;
  }
  .welcome-icon {
    width: 54px;
    height: 54px;
    font-size: 28px;
  }
  .welcome-content h1 {
    font-size: 21px;
  }
}
</style>
