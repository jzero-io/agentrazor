<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { NAlert, NButton, NCard, NEmpty, NInput, NPopconfirm, NTag } from 'naive-ui';
import {
  GetAgentConfigFile,
  GetAgentConfigFiles,
  GetAgentRuntimeStatus,
  RestartAgentRuntime,
  UpdateAgentConfigFile
} from '@/service/api';

const loading = ref(false);
const fileLoading = ref(false);
const saving = ref(false);
const restarting = ref(false);
const files = ref<Api.Manage.AgentConfigFile[]>([]);
const selectedName = ref('config.toml');
const content = ref('');
const runtime = ref<Api.Manage.AgentRuntimeStatus | null>(null);

const selectedFile = computed(() => files.value.find(item => item.name === selectedName.value));
const runtimeText = computed(() => {
  if (!runtime.value) return '未知';
  if (runtime.value.restarting) return '重启中';
  return runtime.value.running ? '运行中' : '未运行';
});
const runtimeType = computed(() => {
  if (!runtime.value) return 'default';
  if (runtime.value.restarting) return 'warning';
  return runtime.value.running ? 'success' : 'error';
});
const languageLabel = computed(() => {
  if (selectedName.value.endsWith('.toml')) return 'TOML';
  if (selectedName.value.endsWith('.json')) return 'JSON';
  return 'TEXT';
});

async function refreshStatus() {
  const { data, error } = await GetAgentRuntimeStatus();
  if (!error) runtime.value = data;
}

async function getFiles() {
  loading.value = true;
  const { data, error } = await GetAgentConfigFiles();
  loading.value = false;
  if (error) return;

  files.value = data.files;
  if (!files.value.some(item => item.name === selectedName.value)) {
    selectedName.value = files.value[0]?.name || 'config.toml';
  }
  await selectFile(selectedName.value);
}

async function selectFile(name: string) {
  if (!name) return;
  selectedName.value = name;
  fileLoading.value = true;
  const { data, error } = await GetAgentConfigFile(name);
  fileLoading.value = false;
  if (error) return;

  content.value = data.content;
}

async function saveFile() {
  if (!selectedName.value) return;

  saving.value = true;
  const { data, error } = await UpdateAgentConfigFile(selectedName.value, content.value, false);
  saving.value = false;
  if (error) return;

  runtime.value = data.runtime;
  await selectFile(selectedName.value);
}

async function restartRuntime() {
  restarting.value = true;
  const { data, error } = await RestartAgentRuntime();
  restarting.value = false;
  if (!error) runtime.value = data;
}

onMounted(async () => {
  await Promise.all([getFiles(), refreshStatus()]);
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard title="配置管理" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper config-card">
      <div class="mb-16px flex flex-wrap items-center justify-between gap-12px">
        <div class="flex items-center gap-10px">
          <NTag :type="runtimeType" round>{{ runtimeText }}</NTag>
          <span class="text-13px text-gray-500 dark:text-gray-400">活跃任务 {{ runtime?.activeTurnCount ?? 0 }}</span>
          <span v-if="runtime?.lastRestartTime" class="text-13px text-gray-500 dark:text-gray-400">最近重启 {{ runtime.lastRestartTime }}</span>
        </div>
        <div class="flex items-center gap-8px">
          <NButton size="small" secondary :loading="loading" @click="getFiles">
            <template #icon><icon-mdi-refresh class="text-icon" /></template>
            刷新
          </NButton>
          <NPopconfirm @positive-click="restartRuntime">
            <template #trigger>
              <NButton size="small" secondary type="warning" :loading="restarting">
                <template #icon><icon-mdi-restart class="text-icon" /></template>
                重启 app-server
              </NButton>
            </template>
            如果有正在运行的任务，后端会拒绝重启。确认重启？
          </NPopconfirm>
        </div>
      </div>

      <div class="min-h-0 flex-1 grid grid-cols-[260px_minmax(0,1fr)] gap-16px lt-lg:grid-cols-1">
        <section class="min-h-0 flex flex-col overflow-hidden rounded-6px border border-gray-100 bg-white p-16px dark:border-gray-700 dark:bg-#18181c">
          <div class="mb-12px font-medium">Codex 配置文件</div>
          <div v-if="files.length" class="min-h-0 flex-1 overflow-auto pr-4px">
            <button
              v-for="item in files"
              :key="item.name"
              type="button"
              class="mb-8px w-full rounded-6px border px-12px py-11px text-left transition-colors last:mb-0"
              :class="selectedName === item.name ? 'border-primary bg-primary bg-opacity-8' : 'border-gray-200 bg-white hover:border-primary/60 dark:border-gray-700 dark:bg-#101014'"
              @click="selectFile(item.name)"
            >
              <div class="flex items-center gap-8px">
                <icon-mdi-file-document-outline class="text-icon text-gray-400" />
                <span class="truncate text-14px font-medium">{{ item.name }}</span>
              </div>
            </button>
          </div>
          <NEmpty v-else class="py-44px" description="暂无配置文件" />
        </section>

        <section class="min-h-0 min-w-0 flex flex-col overflow-hidden rounded-6px border border-gray-100 bg-white dark:border-gray-700 dark:bg-#18181c">
          <div class="flex shrink-0 items-center justify-between gap-12px border-b border-gray-100 p-16px dark:border-gray-700">
            <div class="min-w-0 flex items-center gap-10px">
              <span class="truncate text-15px font-semibold">{{ selectedFile?.name || selectedName }}</span>
              <NTag size="small" round>{{ languageLabel }}</NTag>
            </div>
            <div class="flex shrink-0 items-center gap-8px">
              <NButton size="small" secondary type="primary" :loading="saving" @click="saveFile">
                <template #icon><icon-material-symbols-save-outline class="text-icon" /></template>
                保存
              </NButton>
            </div>
          </div>

          <NAlert v-if="runtime?.activeTurnCount" type="warning" :bordered="false" class="mx-16px mt-16px">
            当前有 {{ runtime.activeTurnCount }} 个任务正在运行，重启会被后端拒绝。
          </NAlert>

          <div class="min-h-0 flex-1 p-16px">
            <NInput
              v-model:value="content"
              type="textarea"
              :autosize="false"
              :loading="fileLoading"
              class="config-editor h-full"
              placeholder="编辑配置文件内容"
            />
          </div>
        </section>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.config-card :deep(.n-card__content) {
  min-height: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
}

.config-editor :deep(.n-input) {
  height: 100%;
}

.config-editor :deep(.n-input-wrapper) {
  height: 100%;
  background-color: #fff;
}

.dark .config-editor :deep(.n-input-wrapper) {
  background-color: #141418;
}

.config-editor :deep(textarea) {
  height: 100% !important;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 13px;
  line-height: 1.5;
}
</style>
