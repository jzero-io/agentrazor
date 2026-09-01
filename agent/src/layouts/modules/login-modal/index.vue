<script setup lang="ts">
import { Icon } from '@iconify/vue';
import { NButton, NInput, NModal } from 'naive-ui';

defineProps<{
  visible: boolean;
  username: string;
  password: string;
  loading: boolean;
  offsetX: number;
}>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
  'update:username': [value: string];
  'update:password': [value: string];
  submit: [];
}>();
</script>

<template>
  <n-modal
    :show="visible"
    preset="card"
    :bordered="false"
    :mask-closable="!loading"
    class="login-modal"
    :style="{ '--login-offset-x': `${offsetX}px` }"
    transform-origin="center"
    @update:show="value => emit('update:visible', value)"
  >
    <n-button quaternary circle class="login-close" aria-label="关闭登录窗口" @click="emit('update:visible', false)">
      <template #icon><Icon icon="solar:close-circle-linear" /></template>
    </n-button>
    <div class="login-hero">
      <img src="/agentrazor-icon.png" alt="" />
      <div>
        <span>AGENTRAZOR</span>
        <h2>欢迎回来</h2>
        <p>登录后继续你的 Agent 对话与任务。</p>
      </div>
    </div>
    <div class="login-form">
      <label>
        <span>用户名</span>
        <n-input :value="username" size="large" placeholder="输入用户名" autofocus @update:value="value => emit('update:username', value)">
          <template #prefix><Icon icon="solar:user-rounded-linear" /></template>
        </n-input>
      </label>
      <label>
        <span>密码</span>
        <n-input
          :value="password"
          type="password"
          size="large"
          show-password-on="click"
          placeholder="输入密码"
          @update:value="value => emit('update:password', value)"
          @keyup.enter="emit('submit')"
        >
          <template #prefix><Icon icon="solar:lock-keyhole-minimalistic-linear" /></template>
        </n-input>
      </label>
      <n-button
        type="primary"
        size="large"
        block
        :loading="loading"
        :disabled="!username.trim() || !password"
        @click="emit('submit')"
      >
        登录 AgentRazor
      </n-button>
    </div>
  </n-modal>
</template>
