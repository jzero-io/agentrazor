<script setup lang="ts">
import { computed, reactive } from 'vue';
import { $t } from '@/locales';
import { loginModuleRecord } from '@/constants/app';
import { useRouterPush } from '@/hooks/common/router';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { useAuthStore } from '@/store/modules/auth';

defineOptions({
  name: 'PwdLogin'
});

const authStore = useAuthStore();
const { toggleLoginModule } = useRouterPush();
const { formRef, validate } = useNaiveForm();

interface FormModel {
  username: string;
  password: string;
}

const model: FormModel = reactive({
  username: '',
  password: ''
});

const rules = computed<Record<keyof FormModel, App.Global.FormRule[]>>(() => {
  // inside computed to make locale reactive, if not apply i18n, you can define it without computed
  const { formRules } = useFormRules();

  return {
    username: formRules.username,
    password: formRules.pwd
  };
});

async function handleSubmit() {
  await validate();
  await authStore.loginByPwd(model.username, model.password);
}
</script>

<template>
  <NForm ref="formRef" class="login-form" :model="model" :rules="rules" size="large" :show-label="false" autocomplete="off" @keyup.enter="handleSubmit">
    <NFormItem path="username">
      <NInput
        v-model:value="model.username"
        round
        autocomplete="off"
        :placeholder="$t('page.login.common.usernamePlaceholder')"
      >
        <template #prefix>
          <SvgIcon icon="carbon:user" class="text-18px text-gray-400" />
        </template>
      </NInput>
    </NFormItem>
    <NFormItem path="password">
      <NInput
        v-model:value="model.password"
        type="password"
        round
        autocomplete="new-password"
        show-password-on="click"
        :placeholder="$t('page.login.common.passwordPlaceholder')"
      >
        <template #prefix>
          <SvgIcon icon="carbon:locked" class="text-18px text-gray-400" />
        </template>
      </NInput>
    </NFormItem>
    <div class="mb-18px flex-y-center justify-end">
      <NButton quaternary size="small" class="px-0" @click="toggleLoginModule('reset-pwd')">
        {{ $t('page.login.pwdLogin.forgetPassword') }}
      </NButton>
    </div>
    <NSpace vertical :size="24">
      <NButton type="primary" size="large" round block :loading="authStore.loginLoading" @click="handleSubmit">
        {{ $t('common.confirm') }}
      </NButton>
      <div class="flex-y-center justify-between gap-12px">
        <NButton class="flex-1" block @click="toggleLoginModule('code-login')">
          {{ $t(loginModuleRecord['code-login'], { type: $t('page.login.codeLogin.emailType') }) }}
        </NButton>
        <NButton class="flex-1" block @click="toggleLoginModule('register')">
          {{ $t(loginModuleRecord.register) }}
        </NButton>
      </div>
    </NSpace>
  </NForm>
</template>

<style scoped>
.login-form :deep(.n-input) {
  box-shadow: none !important;
}

.login-form :deep(.n-input-wrapper) {
  padding-inline: 16px;
  background: transparent !important;
  box-shadow: none !important;
}

.login-form :deep(.n-input__input-el) {
  background: transparent !important;
  box-shadow: none !important;
}

.login-form :deep(input:-webkit-autofill),
.login-form :deep(input:-webkit-autofill:hover),
.login-form :deep(input:-webkit-autofill:focus),
.login-form :deep(input:-webkit-autofill:active) {
  background-color: transparent !important;
  background-image: none !important;
  box-shadow: none !important;
  -webkit-background-clip: text !important;
  background-clip: text !important;
  -webkit-text-fill-color: var(--n-text-color) !important;
  caret-color: var(--n-text-color);
  transition: background-color 999999s ease-out 0s;
}
</style>
