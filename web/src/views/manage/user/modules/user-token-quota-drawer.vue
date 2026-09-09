<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import {
  DeleteUserTokenQuota,
  GetTokenQuotaGlobal,
  GetUserTokenQuota,
  ResetUserTokenQuota,
  SaveUserTokenQuota,
  type TokenQuotaUser
} from '@/service/api';
import { useAuth } from '@/hooks/business/auth';
import { useAppStore } from '@/store/modules/app';
import { formatCompactNumber } from '@/utils/common';

defineOptions({ name: 'UserTokenQuotaModal' });

interface Props {
  userUuid: string;
}

const props = defineProps<Props>();
const emit = defineEmits<{ updated: [] }>();
const visible = defineModel<boolean>('visible', { default: false });
const { hasAuth } = useAuth();
const appStore = useAppStore();
const loading = ref(false);
const saving = ref(false);
const actionLoading = ref(false);
const quota = ref<TokenQuotaUser | null>(null);
const globalFiveHourLimit = ref<number | null>(null);
const model = reactive({
  enabled: true,
  fiveHourDisabled: false,
  fiveHourLimitTokens: null as number | null,
  sevenDayLimitTokens: null as number | null
});

type QuotaUnit = 'TOKEN' | 'K' | 'M' | 'B';

const quotaUnitMultipliers: Record<QuotaUnit, number> = {
  TOKEN: 1,
  K: 1_000,
  M: 1_000_000,
  B: 1_000_000_000
};
const quotaUnitOptions = [
  { label: 'Token（个）', value: 'TOKEN' },
  { label: 'K（千）', value: 'K' },
  { label: 'M（百万）', value: 'M' },
  { label: 'B（十亿）', value: 'B' }
] satisfies Array<{ label: string; value: QuotaUnit }>;
const fiveHourUnit = ref<QuotaUnit>('M');
const sevenDayUnit = ref<QuotaUnit>('M');

function quotaUnitFor(value: number | null) {
  if (!value) return 'M' as QuotaUnit;
  if (value >= quotaUnitMultipliers.B) return 'B' as QuotaUnit;
  if (value >= quotaUnitMultipliers.M) return 'M' as QuotaUnit;
  if (value >= quotaUnitMultipliers.K) return 'K' as QuotaUnit;
  return 'TOKEN' as QuotaUnit;
}

function quotaValue(tokens: number | null, unit: QuotaUnit) {
  if (tokens === null) return null;
  return tokens / quotaUnitMultipliers[unit];
}

const fiveHourQuotaValue = computed({
  get: () => quotaValue(model.fiveHourLimitTokens, fiveHourUnit.value),
  set: (value: number | null) => {
    model.fiveHourLimitTokens = value === null ? null : Math.round(value * quotaUnitMultipliers[fiveHourUnit.value]);
  }
});
const sevenDayQuotaValue = computed({
  get: () => quotaValue(model.sevenDayLimitTokens, sevenDayUnit.value),
  set: (value: number | null) => {
    model.sevenDayLimitTokens = value === null ? null : Math.round(value * quotaUnitMultipliers[sevenDayUnit.value]);
  }
});

const fiveHourEnabled = computed({
  get: () => !model.fiveHourDisabled,
  set: enabled => {
    model.fiveHourDisabled = !enabled;
  }
});
const canView = computed(() =>
  hasAuth([
    'v1:manage:agent:getUserTokenQuota',
    'v1:manage:agent:saveUserTokenQuota',
    'v1:manage:agent:deleteUserTokenQuota',
    'v1:manage:agent:resetUserTokenQuota'
  ])
);
const canSave = computed(() => hasAuth('v1:manage:agent:saveUserTokenQuota'));
const canRestore = computed(() => hasAuth('v1:manage:agent:deleteUserTokenQuota'));
const canReset = computed(() => hasAuth('v1:manage:agent:resetUserTokenQuota'));

function applyQuota(value: TokenQuotaUser) {
  quota.value = value;
  model.enabled = value.enabled;
  model.fiveHourDisabled = value.fiveHourDisabled;
  model.fiveHourLimitTokens =
    value.fiveHourLimitTokens ?? value.effectiveFiveHourLimit ?? globalFiveHourLimit.value ?? null;
  model.sevenDayLimitTokens = value.sevenDayLimitTokens ?? value.effectiveSevenDayLimit;
  fiveHourUnit.value = quotaUnitFor(model.fiveHourLimitTokens);
  sevenDayUnit.value = quotaUnitFor(model.sevenDayLimitTokens);
}

function usagePercent(used: number, limit: number) {
  if (!limit) return 0;
  return Math.min(100, Math.max(0, Math.round((used / limit) * 1000) / 10));
}

function formatToken(value: number) {
  return formatCompactNumber(value, appStore.locale);
}

function formatExactToken(value: number | null) {
  return value?.toLocaleString(appStore.locale) || '-';
}

function formatResetTime(value: string) {
  return new Date(value).toLocaleString(appStore.locale, { hour12: false });
}

function resetText(value: string | undefined, windowLabel: string) {
  return value ? `重置时间：${formatResetTime(value)}` : `产生用量后 ${windowLabel}滚动释放`;
}

async function loadQuota() {
  if (!props.userUuid || !canView.value) return;
  loading.value = true;
  if (canView.value) {
    const { data, error } = await GetTokenQuotaGlobal();
    if (!error) globalFiveHourLimit.value = data.quota.fiveHourLimitTokens;
  }
  const { data, error } = await GetUserTokenQuota(props.userUuid);
  if (!error) applyQuota(data.quota);
  loading.value = false;
}

async function saveQuota() {
  if (!canSave.value) return;
  if (!model.sevenDayLimitTokens || (fiveHourEnabled.value && !model.fiveHourLimitTokens)) {
    window.$message?.warning('请输入有效的 Token 额度');
    return;
  }
  saving.value = true;
  const { error } = await SaveUserTokenQuota(props.userUuid, {
    enabled: model.enabled,
    fiveHourDisabled: model.fiveHourDisabled,
    fiveHourLimitTokens: model.fiveHourDisabled ? undefined : model.fiveHourLimitTokens || undefined,
    sevenDayLimitTokens: model.sevenDayLimitTokens || undefined
  });
  saving.value = false;
  if (error) return;
  window.$message?.success('额度设置已保存');
  emit('updated');
  visible.value = false;
}

async function restoreGlobal() {
  if (!canRestore.value) return;
  actionLoading.value = true;
  const { error } = await DeleteUserTokenQuota(props.userUuid);
  actionLoading.value = false;
  if (error) return;
  window.$message?.success('已恢复全局额度');
  emit('updated');
  await loadQuota();
}

async function resetQuota() {
  if (!canReset.value) return;
  actionLoading.value = true;
  const { error } = await ResetUserTokenQuota(props.userUuid);
  actionLoading.value = false;
  if (error) return;
  window.$message?.success('额度用量已重置');
  emit('updated');
  await loadQuota();
}

watch(
  [visible, () => props.userUuid],
  ([show]) => {
    if (show) loadQuota();
  },
  { immediate: true }
);
</script>

<template>
  <NModal
    v-model:show="visible"
    display-directive="show"
    preset="card"
    title="额度设置"
    :bordered="false"
    :mask-closable="!saving && !actionLoading"
    class="user-quota-modal"
    style="width: min(560px, calc(100vw - 24px))"
  >
    <NSpin :show="loading">
      <div v-if="quota" class="quota-usage" :class="{ 'quota-usage--single': !quota.effectiveFiveHourLimit }">
        <article v-if="quota.effectiveFiveHourLimit" class="usage-window">
          <header>
            <span>5 小时已用</span>
            <strong>{{ usagePercent(quota.fiveHourUsedTokens, quota.effectiveFiveHourLimit) }}%</strong>
          </header>
          <NProgress
            :percentage="usagePercent(quota.fiveHourUsedTokens, quota.effectiveFiveHourLimit)"
            :show-indicator="false"
            :height="7"
            :border-radius="4"
          />
          <div class="usage-values">
            <span>{{ formatToken(quota.fiveHourUsedTokens) }} / {{ formatToken(quota.effectiveFiveHourLimit) }}</span>
            <time :datetime="quota.fiveHourResetAt">
              {{ resetText(quota.fiveHourResetAt, '5 小时') }}
            </time>
          </div>
        </article>

        <article class="usage-window">
          <header>
            <span>每周已用</span>
            <strong>{{ usagePercent(quota.sevenDayUsedTokens, quota.effectiveSevenDayLimit) }}%</strong>
          </header>
          <NProgress
            :percentage="usagePercent(quota.sevenDayUsedTokens, quota.effectiveSevenDayLimit)"
            :show-indicator="false"
            :height="7"
            :border-radius="4"
          />
          <div class="usage-values">
            <span>{{ formatToken(quota.sevenDayUsedTokens) }} / {{ formatToken(quota.effectiveSevenDayLimit) }}</span>
            <time :datetime="quota.sevenDayResetAt">
              {{ resetText(quota.sevenDayResetAt, '7 天') }}
            </time>
          </div>
        </article>
      </div>

      <div class="quota-settings">
        <div class="setting-line">
          <strong>Agent 使用</strong>
          <div class="setting-control">
            <NSwitch v-model:value="model.enabled" :disabled="!canSave" aria-label="Agent 使用" />
          </div>
        </div>

        <div class="quota-line">
          <header>
            <strong>5 小时使用限额</strong>
            <div class="setting-control">
              <NSwitch v-model:value="fiveHourEnabled" :disabled="!canSave" aria-label="5 小时使用限额" />
            </div>
          </header>
          <div v-if="fiveHourEnabled" class="quota-input-wrap">
            <NInputGroup class="quota-input-group">
              <NInputNumber
                v-model:value="fiveHourQuotaValue"
                :disabled="!canSave"
                :min="1"
                :precision="2"
                :show-button="false"
                placeholder="请输入数值"
              />
              <NSelect
                v-model:value="fiveHourUnit"
                :disabled="!canSave"
                :options="quotaUnitOptions"
                :consistent-menu-width="false"
              />
            </NInputGroup>
            <span class="quota-input-help">即 {{ formatExactToken(model.fiveHourLimitTokens) }} Token</span>
          </div>
        </div>

        <div class="quota-line">
          <header>
            <strong>每周使用限额</strong>
            <div class="setting-control">
              <NSwitch :value="true" disabled aria-label="每周使用限额" />
            </div>
          </header>
          <div class="quota-input-wrap">
            <NInputGroup class="quota-input-group">
              <NInputNumber
                v-model:value="sevenDayQuotaValue"
                :disabled="!canSave"
                :min="1"
                :precision="2"
                :show-button="false"
                placeholder="请输入数值"
              />
              <NSelect
                v-model:value="sevenDayUnit"
                :disabled="!canSave"
                :options="quotaUnitOptions"
                :consistent-menu-width="false"
              />
            </NInputGroup>
            <span class="quota-input-help">即 {{ formatExactToken(model.sevenDayLimitTokens) }} Token</span>
          </div>
        </div>
      </div>
    </NSpin>

    <template #footer>
      <div class="quota-modal-footer">
        <div class="footer-actions">
          <NPopconfirm v-if="canRestore && quota?.configured" @positive-click="restoreGlobal">
            <template #trigger>
              <NButton class="quota-action-button" secondary type="primary" :loading="actionLoading">
                <template #icon><SvgIcon icon="carbon:undo" /></template>
                恢复全局额度
              </NButton>
            </template>
            确认恢复全局额度？
          </NPopconfirm>
          <NPopconfirm v-if="canReset" @positive-click="resetQuota">
            <template #trigger>
              <NButton class="quota-action-button" secondary type="primary" :loading="actionLoading">
                <template #icon><SvgIcon icon="carbon:restart" /></template>
                重置用量
              </NButton>
            </template>
            确认重置 5 小时和每周用量？
          </NPopconfirm>
        </div>
        <div class="footer-actions">
          <NButton @click="visible = false">取消</NButton>
          <NButton v-if="canSave" type="primary" :loading="saving" @click="saveQuota">保存</NButton>
        </div>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.quota-usage {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 4px;
}

.quota-usage--single {
  grid-template-columns: 1fr;
}

.usage-window {
  min-width: 0;
  padding: 14px 0;
}

.usage-window header,
.usage-values {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.usage-window header {
  margin-bottom: 9px;
  color: rgba(var(--base-text-color), 0.68);
  font-size: 13px;
}

.usage-window header strong {
  color: rgb(var(--primary-color));
  font-size: 16px;
  font-variant-numeric: tabular-nums;
}

.usage-values {
  align-items: flex-start;
  flex-direction: column;
  gap: 3px;
  margin-top: 8px;
  color: rgba(var(--base-text-color), 0.52);
  font-size: 13px;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}

.usage-values time {
  display: block;
  width: 100%;
  max-width: 100%;
  white-space: normal;
  overflow-wrap: break-word;
}

.quota-settings {
  border-top: 1px solid rgba(var(--base-text-color), 0.09);
}

.setting-line,
.quota-line header,
.setting-control,
.quota-modal-footer,
.footer-actions {
  display: flex;
  align-items: center;
}

.setting-line,
.quota-line {
  padding: 16px 0;
  border-bottom: 1px solid rgba(var(--base-text-color), 0.09);
}

.setting-line,
.quota-line header,
.quota-modal-footer {
  justify-content: space-between;
}

.setting-line strong,
.quota-line strong {
  font-size: 14px;
  font-weight: 600;
}

.setting-control,
.footer-actions {
  gap: 10px;
}
.quota-action-button {
  height: 38px;
  padding: 0 15px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  transition: box-shadow 160ms ease, transform 160ms ease;
}

.quota-action-button:hover {
  box-shadow: 0 6px 16px rgba(var(--primary-color), 0.16);
  transform: translateY(-1px);
}


.setting-control span {
  color: rgb(var(--primary-color));
  font-size: 12px;
}

.setting-control span.danger {
  color: rgb(var(--error-color));
}

.quota-line header {
  margin-bottom: 11px;
}

.quota-input-group {
  width: 100%;
}

.quota-input-group :deep(.n-input-number) {
  width: 0;
  min-width: 0;
  flex: 1;
}

.quota-input-group :deep(.n-select) {
  width: 132px;
  flex: 0 0 132px;
}

.quota-input-help {
  display: block;
  margin-top: 7px;
  color: rgba(var(--base-text-color), 0.48);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 600px) {
  .quota-usage {
    grid-template-columns: 1fr;
  }

  .quota-modal-footer {
    align-items: stretch;
    flex-direction: column-reverse;
    gap: 12px;
  }

  .footer-actions {
    justify-content: flex-end;
  }
}
</style>
