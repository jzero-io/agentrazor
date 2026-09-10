<script lang="ts" setup>
import agentrazorLogo from '@/assets/imgs/agentrazor.png';

defineOptions({ name: 'SystemLogo' });
</script>

<template>
  <img :src="agentrazorLogo" alt="AgentRazor" class="system-logo" />
</template>

<style scoped>
.system-logo {
  display: block;
  width: 1em;
  height: 1em;
  object-fit: contain;
}
</style>
