<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import Clipboard from 'clipboard';
import { NAlert, NButton, NCard, NForm, NFormItem, NInput, NModal, NPopconfirm, NSelect, NTag } from 'naive-ui';
import {
  GetAgentAccountStatus,
  GetAgentSettings,
  LoginAgentWithApiKey,
  LogoutAgentAccount,
  RestartAgentRuntime,
  SaveAgentProviderApiKey,
  SaveAgentSelection,
  StartAgentChatGPTLogin
} from '@/service/api';
import { $t } from '@/locales';
import { useAppStore } from '@/store/modules/app';
const appStore = useAppStore();

const savingSelection = ref(false);
const restarting = ref(false);
const apiKey = ref('');
const apiKeyLoading = ref(false);
const providerApiKey = ref('');
const chatGPTLoading = ref(false);
const settings = ref<Api.Manage.AgentSettings | null>(null);
const providerId = ref('');
const model = ref('');
const reasoningEffort = ref('');
const openAIAuthView = ref<'chatgpt' | 'apikey'>('chatgpt');
const loginModalVisible = ref(false);
const deviceLogin = ref<Api.Manage.AgentDeviceLogin | null>(null);
const deviceLoginRemainingSeconds = ref(0);
const deviceLoginContainerRef = ref<HTMLElement | null>(null);
let loginTimer: number | undefined;
let loginCountdownTimer: number | undefined;
let loginExpiresAt = 0;

const providers = computed(() => settings.value?.providers || []);
const activeProvider = computed(() => providers.value.find(item => item.id === providerId.value));
const providerOptions = computed(() => providers.value.map(item => ({ label: item.id, value: item.id })));
const modelOptions = computed(() =>
  (activeProvider.value?.models || []).map(item => ({
    label: item.name || item.id,
    value: item.id
  }))
);
const selectedModel = computed(() => activeProvider.value?.models.find(item => item.id === model.value));
const displayedOpenAIAuthMode = computed(() => {
  if (!settings.value?.account.loggedIn) return openAIAuthView.value;
  return settings.value.account.authMode === 'apikey' ? 'apikey' : 'chatgpt';
});
const deviceLoginExpired = computed(() => Boolean(deviceLogin.value) && deviceLoginRemainingSeconds.value <= 0);
const deviceLoginRemainingText = computed(() => {
  const minutes = Math.floor(deviceLoginRemainingSeconds.value / 60);
  const seconds = deviceLoginRemainingSeconds.value % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
});
const effortLabels = computed<Record<string, string>>(() => ({
  low: $t('page.agentConfig.effort.low'),
  medium: $t('page.agentConfig.effort.medium'),
  high: $t('page.agentConfig.effort.high'),
  xhigh: $t('page.agentConfig.effort.xhigh'),
  max: $t('page.agentConfig.effort.max'),
  ultra: $t('page.agentConfig.effort.ultra')
}));
const lastRestartText = computed(() => {
  const value = settings.value?.runtime.lastRestartTime;
  if (!value) return $t('page.agentConfig.noRestartRecord');
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString(appStore.locale, { hour12: false });
});
const selectionDirty = computed(() => {
  if (!settings.value) return false;
  return (
    providerId.value !== settings.value.activeProvider ||
    model.value !== settings.value.model ||
    reasoningEffort.value !== settings.value.reasoningEffort ||
    (providerId.value !== 'openai' && providerApiKey.value.trim() !== '')
  );
});
const effortOptions = computed(() => {
  const values = selectedModel.value?.reasoningEfforts?.length
    ? selectedModel.value.reasoningEfforts
    : ['low', 'medium', 'high', 'xhigh', 'max', 'ultra'];
  return values.map(value => ({
    label: effortLabels.value[value] || value,
    value
  }));
});
const runtimeText = computed(() => {
  if (!settings.value) return $t('page.agentConfig.status.unknown');
  if (settings.value.runtime.restarting) return $t('page.agentConfig.status.restarting');
  return settings.value.runtime.running ? $t('page.agentConfig.status.running') : $t('page.agentConfig.status.stopped');
});
const runtimeType = computed(() => {
  if (!settings.value) return 'default';
  if (settings.value.runtime.restarting) return 'warning';
  return settings.value.runtime.running ? 'success' : 'error';
});

async function refreshSettings(syncForm = true) {
  const { data, error } = await GetAgentSettings();
  if (error) return;
  settings.value = data;
  if (data.account.loggedIn) {
    openAIAuthView.value = data.account.authMode === 'apikey' ? 'apikey' : 'chatgpt';
  } else if (syncForm) {
    openAIAuthView.value = 'chatgpt';
  }
  if (syncForm) {
    providerId.value = data.activeProvider || data.providers[0]?.id || '';
    model.value = data.model;
    reasoningEffort.value = data.reasoningEffort;
    providerApiKey.value = '';
  }
}

function changeProvider() {
  providerApiKey.value = '';
  const firstModel = activeProvider.value?.models[0];
  model.value = firstModel?.id || '';
  reasoningEffort.value = firstModel?.defaultReasoningEffort || '';
}

function changeModel() {
  if (selectedModel.value?.defaultReasoningEffort) {
    reasoningEffort.value = selectedModel.value.defaultReasoningEffort;
  }
}

async function saveSelection() {
  if (!providerId.value || !model.value.trim()) {
    window.$message?.warning($t('page.agentConfig.message.selectProviderModel'));
    return;
  }
  const provider = activeProvider.value;
  const stagedApiKey = providerApiKey.value.trim();
  if (providerId.value !== 'openai' && !provider?.hasApiKey && !stagedApiKey) {
    window.$message?.warning(
      $t('page.agentConfig.message.providerApiKeyRequired', {
        provider: provider?.id || $t('page.agentConfig.supplier')
      })
    );
    return;
  }
  savingSelection.value = true;
  if (providerId.value !== 'openai' && stagedApiKey) {
    const { error: keyError } = await SaveAgentProviderApiKey({
      providerId: providerId.value,
      apiKey: stagedApiKey
    });
    if (keyError) {
      savingSelection.value = false;
      return;
    }
  }
  const { error } = await SaveAgentSelection({
    providerId: providerId.value,
    model: model.value.trim(),
    reasoningEffort: reasoningEffort.value
  });
  savingSelection.value = false;
  if (!error) {
    providerApiKey.value = '';
    window.$message?.success($t('page.agentConfig.message.applied'));
    await refreshSettings();
  }
}

async function restartRuntime() {
  restarting.value = true;
  const { error } = await RestartAgentRuntime();
  restarting.value = false;
  if (!error) {
    window.$message?.success($t('page.agentConfig.message.restarted'));
    await refreshSettings(false);
  }
}

async function loginApiKey() {
  if (!apiKey.value.trim()) {
    window.$message?.warning($t('page.agentConfig.message.openAIApiKeyRequired'));
    return;
  }
  apiKeyLoading.value = true;
  const { error } = await LoginAgentWithApiKey(apiKey.value.trim());
  apiKeyLoading.value = false;
  if (!error) {
    apiKey.value = '';
    window.$message?.success($t('page.agentConfig.message.openAIApiKeySaved'));
    await refreshSettings(false);
  }
}

async function startChatGPTLogin() {
  chatGPTLoading.value = true;
  const { data, error } = await StartAgentChatGPTLogin();
  chatGPTLoading.value = false;
  if (error) return;
  deviceLogin.value = data;
  deviceLoginRemainingSeconds.value = Math.max(0, data.expiresIn || 0);
  loginExpiresAt = Date.now() + deviceLoginRemainingSeconds.value * 1000;
  loginModalVisible.value = true;
  startLoginPolling();
}

function startLoginPolling() {
  stopLoginPolling();
  updateDeviceLoginRemaining();
  if (deviceLoginExpired.value) return;
  loginCountdownTimer = window.setInterval(updateDeviceLoginRemaining, 1000);

  const poll = async () => {
    if (deviceLoginExpired.value) return;
    const { data, error } = await GetAgentAccountStatus();
    if (!error && settings.value) settings.value = { ...settings.value, account: data.account };
    if (!error && data.account.loggedIn) {
      stopLoginPolling();
      loginModalVisible.value = false;
      window.$message?.success($t('page.agentConfig.message.chatGPTLoginSuccess'));
      return;
    }
    if (!deviceLoginExpired.value) loginTimer = window.setTimeout(poll, 2500);
  };
  loginTimer = window.setTimeout(poll, 1000);
}

function updateDeviceLoginRemaining() {
  deviceLoginRemainingSeconds.value = Math.max(0, Math.ceil((loginExpiresAt - Date.now()) / 1000));
  if (deviceLoginRemainingSeconds.value === 0) stopLoginPolling();
}

function stopLoginPolling() {
  if (loginTimer) window.clearTimeout(loginTimer);
  if (loginCountdownTimer) window.clearInterval(loginCountdownTimer);
  loginTimer = undefined;
  loginCountdownTimer = undefined;
}

function copyDeviceLoginCode() {
  const code = deviceLogin.value?.userCode?.trim();
  const container = deviceLoginContainerRef.value;
  if (!code || !container) {
    window.$message?.error($t('page.agentConfig.login.copyFailed'));
    return;
  }
  try {
    Clipboard.copy(code, { container });
    window.$message?.success($t('page.agentConfig.login.copySuccess'));
  } catch {
    window.$message?.error($t('page.agentConfig.login.copyFailed'));
  }
}

async function logout() {
  const { error } = await LogoutAgentAccount();
  if (!error) {
    openAIAuthView.value = 'chatgpt';
    window.$message?.success($t('page.agentConfig.message.openAILogoutSuccess'));
    await refreshSettings(false);
  }
}

onMounted(() => refreshSettings());
onBeforeUnmount(stopLoginPolling);
</script>

<template>
  <div class="config-page">
    <NCard :bordered="false" size="small" class="hero-card card-wrapper">
      <div class="hero-top">
        <div class="hero-heading">
          <div class="hero-icon"><SvgIcon icon="carbon:settings-adjust" /></div>
          <h2>{{ $t('page.agentConfig.title') }}</h2>
        </div>
        <div class="hero-actions">
          <NTag :type="runtimeType" round size="small">
            <span class="status-dot" :class="{ 'status-dot--running': settings?.runtime.running }"></span>
            {{ runtimeText }}
          </NTag>
          <span class="last-restart">
            {{ $t('page.agentConfig.lastRestart', { time: lastRestartText }) }}
          </span>
        </div>
      </div>

      <NForm label-placement="top" class="configuration-form">
        <div class="model-form-grid">
          <NFormItem :label="$t('page.agentConfig.supplier')">
            <NSelect v-model:value="providerId" :options="providerOptions" @update:value="changeProvider" />
          </NFormItem>
          <NFormItem :label="$t('page.agentConfig.model')">
            <NSelect
              v-model:value="model"
              :options="modelOptions"
              :input-props="{
                autocomplete: 'new-password',
                name: 'agent-model-selector'
              }"
              filterable
              tag
              :placeholder="$t('page.agentConfig.modelPlaceholder')"
              @update:value="changeModel"
            />
          </NFormItem>
          <NFormItem :label="$t('page.agentConfig.reasoningEffort')">
            <NSelect
              v-model:value="reasoningEffort"
              clearable
              :options="effortOptions"
              :placeholder="$t('page.agentConfig.defaultReasoningEffort')"
            />
          </NFormItem>
        </div>
      </NForm>

      <section v-if="providerId === 'openai'" class="provider-section">
        <div class="provider-heading">
          <div class="provider-name">
            {{ $t('page.agentConfig.openAI.title') }}
          </div>
          <NTag
            :type="
              settings?.account.loggedIn && settings.activeProvider === 'openai'
                ? 'success'
                : settings?.account.loggedIn
                  ? 'info'
                  : 'warning'
            "
            round
            class="provider-status"
          >
            {{
              settings?.account.loggedIn && settings.activeProvider === 'openai'
                ? $t('page.agentConfig.status.inUse')
                : settings?.account.loggedIn
                  ? $t('page.agentConfig.status.configured')
                  : $t('page.agentConfig.status.pending')
            }}
          </NTag>
        </div>

        <div class="auth-panel">
          <div class="auth-option-head">
            <div
              class="auth-option-icon"
              :class="displayedOpenAIAuthMode === 'chatgpt' ? 'auth-option-icon--chatgpt' : 'auth-option-icon--key'"
            >
              <SvgIcon :icon="displayedOpenAIAuthMode === 'chatgpt' ? 'carbon:user-avatar' : 'carbon:password'" />
            </div>
            <strong>
              {{
                displayedOpenAIAuthMode === 'chatgpt'
                  ? $t('page.agentConfig.openAI.chatGPTAccount')
                  : $t('page.agentConfig.openAI.apiKey')
              }}
            </strong>
          </div>

          <template v-if="settings?.account.loggedIn && displayedOpenAIAuthMode === 'chatgpt'">
            <div class="account-summary">
              <div class="account-detail-card">
                <span>{{ $t('page.agentConfig.openAI.email') }}</span>
                <strong>{{ settings.account.email || '-' }}</strong>
              </div>
              <div class="account-detail-card">
                <span>{{ $t('page.agentConfig.openAI.planType') }}</span>
                <strong>{{ settings.account.planType || '-' }}</strong>
              </div>
            </div>
            <div class="auth-action-row">
              <NPopconfirm @positive-click="logout">
                <template #trigger>
                  <NButton tertiary type="error">{{ $t('page.agentConfig.openAI.logout') }}</NButton>
                </template>
                {{ $t('page.agentConfig.openAI.logoutConfirm') }}
              </NPopconfirm>
            </div>
          </template>

          <template v-else-if="settings?.account.loggedIn">
            <div class="credential-state">
              <SvgIcon icon="carbon:checkmark-filled" />
              <span>{{ $t('page.agentConfig.openAI.apiKeyConfigured') }}</span>
            </div>
            <div class="auth-action-row">
              <NPopconfirm @positive-click="logout">
                <template #trigger>
                  <NButton tertiary type="error">{{ $t('page.agentConfig.openAI.clearApiKey') }}</NButton>
                </template>
                {{ $t('page.agentConfig.openAI.clearApiKeyConfirm') }}
              </NPopconfirm>
            </div>
          </template>

          <template v-else-if="openAIAuthView === 'chatgpt'">
            <div class="auth-action-row">
              <NButton size="large" type="primary" :loading="chatGPTLoading" @click="startChatGPTLogin">
                <template #icon><SvgIcon icon="carbon:login" /></template>
                {{ $t('page.agentConfig.openAI.loginChatGPT') }}
              </NButton>
              <NButton size="large" quaternary type="primary" @click="openAIAuthView = 'apikey'">
                {{ $t('page.agentConfig.openAI.useApiKey') }}
              </NButton>
            </div>
          </template>

          <template v-else>
            <div class="api-key-form">
              <label>{{ $t('page.agentConfig.openAI.apiKey') }}</label>
              <div class="key-row">
                <NInput
                  v-model:value="apiKey"
                  size="large"
                  type="text"
                  :input-props="{
                    autocomplete: 'off',
                    name: 'agent-openai-credential',
                    class: 'api-key-input-element'
                  }"
                  placeholder="sk-..."
                  @keyup.enter="loginApiKey"
                >
                  <template #prefix><SvgIcon icon="carbon:password" /></template>
                </NInput>
                <NButton size="large" type="primary" :loading="apiKeyLoading" @click="loginApiKey">
                  {{ $t('page.agentConfig.openAI.saveAndUse') }}
                </NButton>
              </div>
            </div>
            <div class="auth-action-row auth-action-row--compact">
              <NButton text type="primary" @click="openAIAuthView = 'chatgpt'">
                {{ $t('page.agentConfig.openAI.backToChatGPT') }}
              </NButton>
            </div>
          </template>
        </div>
      </section>

      <section v-else-if="activeProvider" class="provider-section">
        <div class="provider-heading">
          <div class="provider-name">
            {{
              $t('page.agentConfig.external.title', {
                provider: activeProvider.id
              })
            }}
          </div>
          <NTag
            :type="
              activeProvider.hasApiKey && settings?.activeProvider === activeProvider.id
                ? 'success'
                : activeProvider.hasApiKey
                  ? 'info'
                  : 'warning'
            "
            round
            class="provider-status"
          >
            {{
              activeProvider.hasApiKey && settings?.activeProvider === activeProvider.id
                ? $t('page.agentConfig.status.inUse')
                : activeProvider.hasApiKey
                  ? $t('page.agentConfig.status.configured')
                  : $t('page.agentConfig.status.pending')
            }}
          </NTag>
        </div>

        <NAlert
          v-if="!activeProvider.hasApiKey && !providerApiKey"
          type="info"
          :bordered="false"
          class="provider-guide"
        >
          {{
            $t('page.agentConfig.external.guide', {
              provider: activeProvider.id
            })
          }}
        </NAlert>

        <div class="external-config-grid">
          <div class="external-config-field">
            <label>{{ $t('page.agentConfig.external.apiAddress') }}</label>
            <div class="endpoint-value">
              <SvgIcon icon="carbon:link" />
              <span>{{ activeProvider.baseUrl || '-' }}</span>
            </div>
          </div>
          <div class="api-key-form external-config-field">
            <label>{{ $t('page.agentConfig.openAI.apiKey') }}</label>
            <NInput
              v-model:value="providerApiKey"
              size="large"
              type="text"
              :input-props="{
                autocomplete: 'off',
                name: 'agent-provider-credential',
                class: 'api-key-input-element'
              }"
              :disabled="savingSelection"
              :placeholder="
                activeProvider.hasApiKey
                  ? $t('page.agentConfig.external.replaceKey')
                  : $t('page.agentConfig.external.keyPlaceholder', {
                      provider: activeProvider.id
                    })
              "
              @keyup.enter="saveSelection"
            >
              <template #prefix><SvgIcon icon="carbon:password" /></template>
            </NInput>
          </div>
        </div>
      </section>

      <div class="form-actions">
        <NButton
          type="primary"
          size="large"
          :loading="savingSelection"
          :disabled="!selectionDirty"
          @click="saveSelection"
        >
          <template #icon><SvgIcon icon="carbon:checkmark" /></template>
          {{ $t('page.agentConfig.saveAndRestart') }}
        </NButton>
        <NButton secondary size="large" :loading="restarting" @click="restartRuntime">
          <template #icon><SvgIcon icon="carbon:restart" /></template>
          {{ $t('page.agentConfig.restartAgent') }}
        </NButton>
      </div>
    </NCard>

    <NModal
      v-model:show="loginModalVisible"
      preset="card"
      :title="$t('page.agentConfig.login.title')"
      class="max-w-[calc(100vw-32px)] w-480px"
      @after-leave="stopLoginPolling"
    >
      <div ref="deviceLoginContainerRef">
        <NAlert :type="deviceLoginExpired ? 'warning' : 'info'" :bordered="false">
          {{ deviceLoginExpired ? $t('page.agentConfig.login.expired') : $t('page.agentConfig.login.guide') }}
        </NAlert>

        <div class="device-login-steps" :class="{ 'device-login-steps--expired': deviceLoginExpired }">
          <div class="device-login-step">
            <span class="device-login-step-index">1</span>
            <div class="device-login-step-content">
              <strong>{{ $t('page.agentConfig.login.openStep') }}</strong>
              <NButton
                tag="a"
                type="primary"
                :href="deviceLoginExpired ? undefined : deviceLogin?.verificationUrl"
                target="_blank"
                rel="noopener noreferrer"
                :disabled="deviceLoginExpired"
              >
                <template #icon><SvgIcon icon="carbon:launch" /></template>
                {{ $t('page.agentConfig.login.openPage') }}
              </NButton>
            </div>
          </div>

          <div class="device-login-step">
            <span class="device-login-step-index">2</span>
            <div class="device-login-step-content">
              <strong>{{ $t('page.agentConfig.login.codeStep') }}</strong>
              <div class="device-login-code-card">
                <div class="device-login-code">
                  <code>{{ deviceLogin?.userCode }}</code>
                  <NButton
                    quaternary
                    type="primary"
                    size="small"
                    :disabled="deviceLoginExpired"
                    @click="copyDeviceLoginCode"
                  >
                    <template #icon><SvgIcon icon="carbon:copy" /></template>
                    {{ $t('page.agentConfig.login.copyCode') }}
                  </NButton>
                </div>
                <div class="device-login-expiry" :class="{ 'device-login-expiry--expired': deviceLoginExpired }">
                  <SvgIcon icon="carbon:time" />
                  <span>{{ $t('page.agentConfig.login.expiresIn', { time: deviceLoginRemainingText }) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="device-login-footer">
          <NButton v-if="deviceLoginExpired" type="primary" :loading="chatGPTLoading" @click="startChatGPTLogin">
            {{ $t('page.agentConfig.login.retry') }}
          </NButton>
          <span v-else>{{ $t('page.agentConfig.login.waiting') }}</span>
        </div>
      </div>
    </NModal>
  </div>
</template>

<style scoped>
.config-page {
  --config-surface: rgb(var(--container-bg-color));
  --config-text: rgb(var(--base-text-color));
  --config-muted: color-mix(in srgb, var(--config-text) 58%, transparent);
  --config-border: color-mix(in srgb, var(--config-text) 14%, transparent);
  --config-divider: color-mix(in srgb, var(--config-text) 9%, transparent);
  --config-soft: color-mix(in srgb, var(--config-text) 4%, var(--config-surface));
  --config-primary-soft: color-mix(in srgb, rgb(var(--primary-color)) 12%, var(--config-surface));
  --config-success-soft: color-mix(in srgb, rgb(var(--success-color)) 10%, var(--config-surface));

  display: flex;
  width: 100%;
  min-height: 100%;
  flex-direction: column;
  color: var(--config-text);
}

.hero-card {
  flex: 1;
  overflow: hidden;
}

.hero-card :deep(.n-card__content) {
  padding: 22px 24px 20px;
}

.hero-top,
.hero-heading,
.hero-actions,
.provider-heading,
.form-actions {
  display: flex;
  align-items: center;
}

.hero-top {
  justify-content: space-between;
  gap: 24px;
}

.hero-heading {
  min-width: 0;
  gap: 12px;
}

.hero-icon {
  display: grid;
  width: 40px;
  height: 40px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 10px;
  color: rgb(var(--primary-color));
  background: var(--config-primary-soft);
  font-size: 21px;
}

.hero-heading h2 {
  margin: 0;
  color: var(--config-text);
  font-size: 19px;
  font-weight: 700;
  line-height: 1.35;
}

.hero-actions {
  flex: 0 0 auto;
  gap: 10px;
}

.last-restart {
  color: var(--config-muted);
  font-size: 12px;
  white-space: nowrap;
}

.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-right: 5px;
  border-radius: 50%;
  background: currentColor;
}

.status-dot--running {
  box-shadow: 0 0 0 3px color-mix(in srgb, rgb(var(--success-color)) 16%, transparent);
}

.configuration-form {
  display: flex;
  margin-top: 22px;
  padding-top: 20px;
  border-top: 1px solid var(--config-divider);
  flex-direction: column;
}

.model-form-grid {
  display: grid;
  grid-template-columns: minmax(180px, 0.8fr) minmax(300px, 1.5fr) minmax(180px, 0.8fr);
  gap: 14px;
}

.model-form-grid :deep(.n-form-item) {
  margin-bottom: 0;
}

.provider-section {
  margin-top: 22px;
  padding-top: 22px;
  border-top: 1px solid var(--config-divider);
}

.provider-heading {
  gap: 14px;
}

.provider-name {
  color: var(--config-text);
  font-size: 17px;
  font-weight: 700;
}

.provider-guide {
  margin-top: 14px;
}

.auth-panel {
  margin-top: 18px;
  padding: 18px;
  border: 1px solid var(--config-divider);
  border-radius: 12px;
  background: color-mix(in srgb, var(--config-text) 2.5%, var(--config-surface));
}

.auth-option-head {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.auth-option-head strong {
  color: var(--config-text);
  font-size: 16px;
  font-weight: 700;
}

.auth-option-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 10px;
  font-size: 20px;
}

.auth-option-icon--chatgpt {
  color: rgb(var(--success-color));
  border: 1px solid color-mix(in srgb, rgb(var(--success-color)) 12%, transparent);
  background: color-mix(in srgb, rgb(var(--success-color)) 7%, var(--config-surface));
}

.auth-option-icon--key {
  color: rgb(var(--primary-color));
  background: var(--config-primary-soft);
}

.account-summary {
  display: grid;
  max-width: 760px;
  margin-top: 18px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.account-detail-card {
  display: flex;
  min-width: 0;
  padding: 14px 16px;
  border: 1px solid var(--config-divider);
  border-radius: 9px;
  background: var(--config-surface);
  box-shadow: 0 1px 2px color-mix(in srgb, var(--config-text) 4%, transparent);
  flex-direction: column;
}

.account-detail-card span {
  color: var(--config-muted);
  font-size: 12px;
}

.account-detail-card strong {
  overflow: hidden;
  margin-top: 4px;
  color: var(--config-text);
  font-size: 14px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.credential-state {
  display: flex;
  width: fit-content;
  align-items: center;
  gap: 8px;
  margin-top: 18px;
  padding: 10px 12px;
  border-radius: 8px;
  color: rgb(var(--success-color));
  background: var(--config-success-soft);
  font-size: 14px;
  font-weight: 600;
}

.auth-action-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 18px;
}

.auth-action-row--compact {
  margin-top: 12px;
}

.api-key-form {
  margin-top: 18px;
}

.api-key-form label,
.external-config-field > label {
  display: block;
  margin-bottom: 8px;
  color: var(--config-text);
  font-size: 14px;
  font-weight: 600;
}

.key-row {
  display: flex;
  max-width: 980px;
  align-items: center;
  gap: 9px;
}

.key-row .n-button {
  flex: 0 0 auto;
}

.external-config-grid {
  display: grid;
  margin-top: 18px;
  grid-template-columns: minmax(260px, 0.9fr) minmax(320px, 1.1fr);
  gap: 18px;
}

.external-config-field {
  min-width: 0;
}

.api-key-form.external-config-field {
  margin-top: 0;
}

.endpoint-value {
  display: flex;
  min-height: 40px;
  align-items: center;
  gap: 9px;
  padding: 0 14px;
  border: 1px solid var(--config-border);
  border-radius: 3px;
  color: var(--config-text);
  background: var(--config-surface);
}

.endpoint-value svg {
  flex: 0 0 auto;
  color: var(--config-muted);
  font-size: 16px;
}

.endpoint-value span {
  overflow: hidden;
  font-size: 14px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.form-actions {
  justify-content: flex-start;
  gap: 16px;
  margin-top: 18px;
  padding-top: 15px;
  border-top: 1px solid var(--config-divider);
}

.device-login-steps {
  display: flex;
  margin-top: 20px;
  flex-direction: column;
  gap: 18px;
}

.device-login-step {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.device-login-step-index {
  display: grid;
  width: 28px;
  height: 28px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 50%;
  color: rgb(var(--primary-color));
  background: var(--config-primary-soft);
  font-size: 13px;
  font-weight: 700;
}

.device-login-step-content {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 10px;
}

.device-login-step-content strong {
  color: var(--config-text);
  font-size: 14px;
  line-height: 28px;
}

.device-login-step-content .n-button {
  width: fit-content;
}

.device-login-code-card {
  display: flex;
  overflow: hidden;
  width: min(100%, 360px);
  border: 1px solid color-mix(in srgb, rgb(var(--primary-color)) 28%, var(--config-border));
  border-radius: 12px;
  background: var(--config-primary-soft);
  flex-direction: column;
}

.device-login-code-card code {
  color: rgb(var(--primary-color));
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: clamp(20px, 5vw, 25px);
  font-weight: 750;
  font-variant-numeric: tabular-nums;
  letter-spacing: 2.5px;
  line-height: 1;
  white-space: nowrap;
}

.device-login-code {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 14px 14px 12px 16px;
}

.device-login-code .n-button {
  flex: 0 0 auto;
}

.device-login-expiry {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-top: 1px solid color-mix(in srgb, rgb(var(--primary-color)) 16%, transparent);
  color: var(--config-muted);
  background: color-mix(in srgb, var(--config-surface) 40%, transparent);
  font-size: 12px;
  font-weight: 600;
}

.device-login-expiry svg {
  flex: 0 0 auto;
  font-size: 15px;
}

.device-login-expiry--expired {
  color: rgb(var(--error-color));
}

.device-login-steps--expired {
  opacity: 0.55;
}

.device-login-footer {
  display: flex;
  min-height: 34px;
  align-items: center;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--config-divider);
  color: var(--config-muted);
  font-size: 13px;
}

:global(.api-key-input-element) {
  -webkit-text-security: disc;
}

@media (max-width: 1000px) {
  .external-config-grid {
    grid-template-columns: 1fr;
  }

  .model-form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .model-form-grid :deep(.n-form-item:nth-child(2)) {
    grid-column: span 2;
    grid-row: 1;
  }
}

@media (max-width: 760px) {
  .config-page {
    min-height: 100%;
    overflow-x: hidden;
  }

  .hero-card {
    flex: none;
  }

  .hero-top {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }

  .hero-actions {
    width: 100%;
    flex-wrap: wrap;
    justify-content: flex-start;
  }

  .last-restart {
    flex: 1;
    white-space: normal;
  }

  .provider-heading {
    justify-content: space-between;
    flex-wrap: wrap;
  }
}

@media (max-width: 560px) {
  .hero-card :deep(.n-card__content) {
    padding: 16px 14px;
  }

  .hero-icon {
    width: 36px;
    height: 36px;
    font-size: 19px;
  }

  .hero-heading h2 {
    font-size: 18px;
  }

  .configuration-form {
    margin-top: 18px;
    padding-top: 16px;
  }

  .model-form-grid,
  .account-summary {
    grid-template-columns: 1fr;
  }

  .model-form-grid :deep(.n-form-item:nth-child(2)) {
    grid-column: auto;
    grid-row: auto;
  }

  .auth-panel {
    margin-top: 14px;
    padding: 14px;
  }

  .provider-section {
    margin-top: 18px;
    padding-top: 18px;
  }

  .account-summary {
    margin-top: 14px;
    gap: 8px;
  }

  .form-actions,
  .key-row {
    align-items: stretch;
    flex-direction: column;
  }

  .form-actions .n-button,
  .key-row .n-button {
    width: 100%;
  }

  .form-actions {
    gap: 10px;
  }

  .auth-action-row {
    flex-wrap: wrap;
  }

  .device-login-code-card {
    width: 100%;
  }

  .device-login-code-card code {
    font-size: 20px;
    letter-spacing: 1.5px;
  }
}

@media (max-width: 400px) {
  .auth-action-row {
    align-items: stretch;
    flex-direction: column;
  }

  .auth-action-row .n-button {
    width: 100%;
  }

  .device-login-step {
    gap: 8px;
  }

  .device-login-step-index {
    width: 24px;
    height: 24px;
  }

  .device-login-code {
    align-items: stretch;
    flex-direction: column;
  }

  .device-login-code .n-button {
    width: 100%;
  }
}
</style>
