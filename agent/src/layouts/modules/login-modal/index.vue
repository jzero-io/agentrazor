<script setup lang="ts">
import { Icon } from '@iconify/vue';
import { NButton, NInput, NModal } from 'naive-ui';

defineProps<{
  visible: boolean;
  mode: 'password' | 'email';
  username: string;
  password: string;
  email: string;
  verificationCode: string;
  verificationReady: boolean;
  verificationSending: boolean;
  verificationCountdown: number;
  loading: boolean;
  offsetX: number;
}>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
  'update:mode': [value: 'password' | 'email'];
  'update:username': [value: string];
  'update:password': [value: string];
  'update:email': [value: string];
  'update:verificationCode': [value: string];
  submit: [];
  sendVerificationCode: [];
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

    <div class="login-brand">
      <img src="/agentrazor-icon.png" alt="" />
      <strong>AgentRazor</strong>
    </div>

    <div class="login-form">
      <h2 class="login-form-title">{{ mode === 'password' ? '密码登录' : '邮箱验证码登录' }}</h2>

      <div v-if="mode === 'password'" class="login-form-fields">
        <n-input
          :value="username"
          size="large"
          round
          placeholder="请输入用户名"
          autofocus
          aria-label="用户名"
          @update:value="value => emit('update:username', value)"
        >
          <template #prefix><Icon icon="solar:user-rounded-linear" /></template>
        </n-input>
        <n-input
          :value="password"
          type="password"
          size="large"
          round
          show-password-on="click"
          placeholder="请输入密码"
          aria-label="密码"
          @update:value="value => emit('update:password', value)"
          @keyup.enter="emit('submit')"
        >
          <template #prefix><Icon icon="solar:lock-keyhole-minimalistic-linear" /></template>
        </n-input>
      </div>

      <div v-else class="login-form-fields">
        <n-input
          :value="email"
          size="large"
          round
          placeholder="请输入邮箱"
          autofocus
          aria-label="邮箱"
          @update:value="value => emit('update:email', value)"
        >
          <template #prefix><Icon icon="solar:letter-linear" /></template>
        </n-input>
        <div class="login-code-row">
          <n-input
            :value="verificationCode"
            size="large"
            round
            maxlength="6"
            placeholder="请输入验证码"
            aria-label="邮箱验证码"
            @update:value="value => emit('update:verificationCode', value)"
            @keyup.enter="emit('submit')"
          >
            <template #prefix><Icon icon="solar:shield-keyhole-linear" /></template>
          </n-input>
          <n-button
            secondary
            type="primary"
            size="large"
            round
            :loading="verificationSending"
            :disabled="verificationCountdown > 0 || !email.trim()"
            @click="emit('sendVerificationCode')"
          >
            {{ verificationCountdown > 0 ? `${verificationCountdown}s` : '获取验证码' }}
          </n-button>
        </div>
      </div>

      <n-button
        class="login-submit"
        type="primary"
        size="large"
        round
        block
        :loading="loading"
        :disabled="
          mode === 'password'
            ? !username.trim() || !password
            : !email.trim() || !verificationCode.trim() || !verificationReady
        "
        @click="emit('submit')"
      >
        确认
      </n-button>

      <n-button
        class="login-alt-button"
        size="large"
        block
        :disabled="loading"
        @click="emit('update:mode', mode === 'password' ? 'email' : 'password')"
      >
        {{ mode === 'password' ? '邮箱验证码登录' : '返回密码登录' }}
      </n-button>
    </div>
  </n-modal>
</template>
