<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import type { FormInst, FormRules } from 'naive-ui';
import {
  NAlert,
  NButton,
  NCard,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NInput,
  NInputNumber,
  NSwitch,
  NTag
} from 'naive-ui';
import { GetEmailConfig, SaveEmailConfig, TestEmailConfig } from '@/service/api';

const loading = ref(false);
const saving = ref(false);
const testing = ref(false);
const configured = ref(false);
const hasPassword = ref(false);
const formRef = ref<FormInst | null>(null);

const model = reactive<Api.Manage.SaveEmailConfigRequest>({
  from: '',
  host: '',
  port: 465,
  username: '',
  password: '',
  enableSsl: true,
  isVerify: true
});
const recipient = ref('');

const rules: FormRules = {
  from: [
    { required: true, message: '请输入发件邮箱', trigger: ['input', 'blur'] },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: ['input', 'blur'] }
  ],
  host: { required: true, message: '请输入 SMTP 服务器地址', trigger: ['input', 'blur'] },
  port: { required: true, type: 'number', message: '请输入 SMTP 端口', trigger: ['input', 'blur'] },
  username: { required: true, message: '请输入 SMTP 用户名', trigger: ['input', 'blur'] },
  password: {
    validator: (_rule, value: string) => {
      if (!configured.value && !value) return new Error('首次配置必须填写密码或授权码');
      return true;
    },
    trigger: ['input', 'blur']
  }
};

async function loadConfig() {
  loading.value = true;
  const { data, error } = await GetEmailConfig();
  loading.value = false;
  if (error) return;

  configured.value = data.configured;
  hasPassword.value = data.config.hasPassword;
  Object.assign(model, {
    from: data.config.from,
    host: data.config.host,
    port: data.config.port,
    username: data.config.username,
    password: '',
    enableSsl: data.config.enableSsl,
    isVerify: data.config.isVerify
  });
  if (!recipient.value) recipient.value = data.config.from;
}

async function saveConfig() {
  await formRef.value?.validate();
  saving.value = true;
  const { error } = await SaveEmailConfig(model);
  saving.value = false;
  if (error) return;

  window.$message?.success('邮箱配置已保存');
  await loadConfig();
}

async function testConfig() {
  if (!recipient.value) {
    window.$message?.warning('请输入测试收件邮箱');
    return;
  }
  testing.value = true;
  const { error } = await TestEmailConfig(recipient.value);
  testing.value = false;
  if (!error) window.$message?.success('测试邮件已发送，请检查收件箱');
}

onMounted(loadConfig);
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-auto">
    <NCard title="邮箱配置" :bordered="false" size="small" class="card-wrapper">
      <template #header-extra>
        <NTag :type="configured ? 'success' : 'warning'" round>
          {{ configured ? '已配置' : '未配置' }}
        </NTag>
      </template>

      <NAlert type="info" :bordered="false" class="mb-20px">
        此配置用于登录、注册和重置密码的验证码邮件。密码或授权码保存后不会由接口返回；留空表示保留原值。
      </NAlert>

      <NForm
        ref="formRef"
        :model="model"
        :rules="rules"
        label-placement="top"
        :disabled="loading"
        class="max-w-900px"
      >
        <NGrid :cols="24" :x-gap="20">
          <NGridItem :span="24">
            <NFormItem label="发件邮箱" path="from">
              <NInput v-model:value="model.from" placeholder="noreply@example.com" />
            </NFormItem>
          </NGridItem>
          <NGridItem :span="24" :m-span="16">
            <NFormItem label="SMTP 服务器" path="host">
              <NInput v-model:value="model.host" placeholder="smtp.example.com" />
            </NFormItem>
          </NGridItem>
          <NGridItem :span="24" :m-span="8">
            <NFormItem label="端口" path="port">
              <NInputNumber v-model:value="model.port" :min="1" :max="65535" class="w-full" />
            </NFormItem>
          </NGridItem>
          <NGridItem :span="24">
            <NFormItem label="用户名" path="username">
              <NInput v-model:value="model.username" placeholder="通常与发件邮箱相同" />
            </NFormItem>
          </NGridItem>
          <NGridItem :span="24">
            <NFormItem label="密码 / 授权码" path="password">
              <NInput
                v-model:value="model.password"
                type="password"
                show-password-on="click"
                :placeholder="hasPassword ? '已保存，留空则不修改' : '请输入 SMTP 密码或授权码'"
              />
            </NFormItem>
          </NGridItem>
          <NGridItem :span="24" :m-span="12">
            <NFormItem label="启用 SSL">
              <NSwitch v-model:value="model.enableSsl" />
            </NFormItem>
          </NGridItem>
          <NGridItem :span="24" :m-span="12">
            <NFormItem label="校验 TLS 证书">
              <div class="flex items-center gap-12px">
                <NSwitch v-model:value="model.isVerify" />
                <span class="text-13px text-gray-500">生产环境建议始终开启</span>
              </div>
            </NFormItem>
          </NGridItem>
        </NGrid>

        <NAlert v-if="!model.isVerify" type="warning" :bordered="false" class="mb-20px">
          关闭证书校验会降低连接安全性，仅建议用于可信内网的自签名证书。
        </NAlert>

        <div class="flex flex-wrap items-center gap-12px">
          <NButton type="primary" :loading="saving" @click="saveConfig">保存配置</NButton>
        </div>
      </NForm>
    </NCard>

    <NCard title="发送测试邮件" :bordered="false" size="small" class="card-wrapper">
      <div class="max-w-900px">
        <div class="mb-8px text-14px">测试收件邮箱</div>
        <div class="flex items-center gap-12px lt-sm:flex-col lt-sm:items-stretch">
          <NInput v-model:value="recipient" class="flex-1" placeholder="recipient@example.com" />
          <NButton
            class="shrink-0"
            type="primary"
            secondary
            :disabled="!configured"
            :loading="testing"
            @click="testConfig"
          >
            发送测试邮件
          </NButton>
        </div>
      </div>
    </NCard>
  </div>
</template>

<style scoped></style>
