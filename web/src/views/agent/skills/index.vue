<script setup lang="tsx">
import { computed, onMounted, ref } from 'vue';
import { NButton, NCard, NEmpty, NInput, NPopconfirm, NTree, NUpload } from 'naive-ui';
import type { TreeOption, UploadCustomRequestOptions } from 'naive-ui';
import { DeleteAgentSkill, GetAgentSkillDetail, GetAgentSkills, UpdateAgentSkillFile, UploadAgentSkill } from '@/service/api';

const loading = ref(false);
const detailLoading = ref(false);
const uploading = ref(false);
const deletingName = ref('');
const editing = ref(false);
const saving = ref(false);
const editContent = ref('');
const searchKeyword = ref('');
const selectedName = ref('');
const selectedFile = ref('');
const expandedKeys = ref<string[]>([]);
const skills = ref<Api.Manage.AgentSkill[]>([]);
const selectedDetail = ref<Api.Manage.SkillDetailResponse | null>(null);

const selectedSkill = computed(() => skills.value.find(item => item.name === selectedName.value));
const visibleContent = computed(() => selectedDetail.value?.content || '');
const currentFile = computed(() => selectedDetail.value?.currentFile || selectedFile.value || 'SKILL.md');
const filteredSkills = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase();
  if (!keyword) return skills.value;

  return skills.value.filter(item => item.name.toLowerCase().includes(keyword));
});
const visibleFiles = computed(() => normalizeTreeFiles(selectedDetail.value?.files ?? []));
const treeData = computed<TreeOption[]>(() => buildTreeOptions(visibleFiles.value));

function normalizeTreeFiles(files: Api.Manage.AgentSkillFile[]) {
  const skillName = selectedSkill.value?.name;
  if (!skillName) return files;

  return files.flatMap(item => {
    if (item.type === 'directory' && item.name === skillName) {
      return item.children ?? [];
    }
    return [item];
  });
}

function buildTreeOptions(files: Api.Manage.AgentSkillFile[]): TreeOption[] {
  return files.map(item => ({
    key: item.path,
    label: item.name,
    isLeaf: item.type === 'file',
    disabled: item.type === 'directory',
    prefix: () => (item.type === 'directory' ? <icon-mdi-folder-outline class="text-icon text-gray-400" /> : <icon-mdi-file-document-outline class="text-icon text-gray-400" />),
    children: item.children?.length ? buildTreeOptions(item.children) : undefined
  }));
}

function directoryKeys(files: Api.Manage.AgentSkillFile[]) {
  const result: string[] = [];
  const walk = (items: Api.Manage.AgentSkillFile[]) => {
    items.forEach(item => {
      if (item.type === 'directory') {
        result.push(item.path);
        walk(item.children ?? []);
      }
    });
  };
  walk(files);
  return result;
}

async function getData() {
  loading.value = true;
  const { data, error } = await GetAgentSkills();
  loading.value = false;
  if (error) return;

  skills.value = data.skills;
  if (!selectedName.value && skills.value.length) {
    await selectSkill(skills.value[0]);
    return;
  }

  if (selectedName.value) {
    const next = skills.value.find(item => item.name === selectedName.value);
    if (next) await selectSkill(next, selectedFile.value);
    else selectedName.value = '';
  }
}

async function selectSkill(skill: Api.Manage.AgentSkill, file = '') {
  selectedName.value = skill.name;
  selectedFile.value = file;
  detailLoading.value = true;
  const { data, error } = await GetAgentSkillDetail(skill.name, file || undefined);
  detailLoading.value = false;
  if (!error) {
    selectedDetail.value = data;
    selectedFile.value = data.currentFile;
    expandedKeys.value = directoryKeys(data.files).slice(0, 24);
    editing.value = false;
    editContent.value = data.content;
  }
}

async function selectFile(keys: Array<string | number>) {
  const key = String(keys[0] ?? '');
  if (!key || !selectedSkill.value || key === selectedFile.value) return;
  await selectSkill(selectedSkill.value, key);
}

async function uploadSkill(options: UploadCustomRequestOptions) {
  const file = options.file.file;
  if (!file) return;

  uploading.value = true;
  const form = new FormData();
  form.append('file', file);
  const { data, error } = await UploadAgentSkill(form);
  uploading.value = false;
  if (error) {
    options.onError();
    return;
  }

  options.onFinish();
  await getData();
  const next = skills.value.find(item => item.name === data.name);
  if (next) await selectSkill(next);
}

async function deleteSkill(skill: Api.Manage.AgentSkill) {
  deletingName.value = skill.name;
  const { error } = await DeleteAgentSkill(skill.name);
  deletingName.value = '';
  if (error) return;

  if (selectedName.value === skill.name) {
    selectedName.value = '';
    selectedFile.value = '';
    selectedDetail.value = null;
  }
  await getData();
}

function startEdit() {
  editContent.value = visibleContent.value;
  editing.value = true;
}

function cancelEdit() {
  editContent.value = visibleContent.value;
  editing.value = false;
}

async function saveFile() {
  if (!selectedSkill.value || !currentFile.value) return;

  saving.value = true;
  const { error } = await UpdateAgentSkillFile(selectedSkill.value.name, currentFile.value, editContent.value);
  saving.value = false;
  if (error) return;

  editing.value = false;
  await selectSkill(selectedSkill.value, currentFile.value);
}

onMounted(getData);
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard title="Skills 管理" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper skills-card">
      <div class="min-h-0 flex-1 grid grid-cols-[320px_minmax(0,1fr)] gap-16px lt-lg:grid-cols-1">
        <section class="min-h-0 flex flex-col overflow-hidden rounded-6px border border-gray-100 bg-white p-16px dark:border-gray-700 dark:bg-#18181c">
          <div class="mb-12px flex items-center justify-between gap-12px">
            <div class="font-medium">已安装 Skills</div>
            <div class="flex shrink-0 items-center gap-8px">
              <NUpload :show-file-list="false" accept=".zip" :custom-request="uploadSkill">
                <NButton type="primary" size="small" :loading="uploading">
                  <template #icon><icon-ic-round-plus class="text-icon" /></template>
                  上传 zip
                </NButton>
              </NUpload>
              <NButton :loading="loading" size="small" secondary @click="getData">
                <template #icon><icon-mdi-refresh class="text-icon" /></template>
                刷新
              </NButton>
            </div>
          </div>

          <NInput v-model:value="searchKeyword" clearable placeholder="搜索名称">
            <template #prefix><icon-ic-round-search class="text-icon text-gray-400" /></template>
          </NInput>

          <div v-if="filteredSkills.length" class="mt-12px min-h-0 flex-1 overflow-auto pr-4px lt-lg:max-h-420px">
            <button
              v-for="item in filteredSkills"
              :key="item.name"
              type="button"
              class="group mb-8px w-full rounded-6px border px-12px py-11px text-left transition-colors last:mb-0"
              :class="selectedName === item.name ? 'border-primary bg-primary bg-opacity-8' : 'border-gray-200 bg-white hover:border-primary/60 dark:border-gray-700 dark:bg-#101014'"
              @click="selectSkill(item)"
            >
              <div class="flex items-center justify-between gap-8px">
                <span class="truncate text-14px font-medium">{{ item.name }}</span>
                <div class="flex shrink-0 items-center gap-6px">
                  <NPopconfirm @positive-click="deleteSkill(item)">
                    <template #trigger>
                      <NButton size="tiny" quaternary type="error" :loading="deletingName === item.name" @click.stop>
                        <template #icon><icon-material-symbols-delete-outline class="text-icon" /></template>
                      </NButton>
                    </template>
                    删除 {{ item.name }}？
                  </NPopconfirm>
                </div>
              </div>
            </button>
          </div>
          <NEmpty v-else class="py-44px" :description="skills.length ? '没有匹配的 skill' : '暂无 skills'" />
        </section>

        <section class="min-h-0 min-w-0 overflow-hidden rounded-6px border border-gray-100 bg-white dark:border-gray-700 dark:bg-#18181c">
          <div v-if="selectedSkill" class="grid h-full min-h-0 grid-cols-[280px_minmax(0,1fr)] lt-xl:grid-cols-[240px_minmax(0,1fr)] lt-lg:grid-cols-1">
            <aside class="min-h-0 flex flex-col border-r border-gray-100 p-16px dark:border-gray-700 lt-lg:border-b lt-lg:border-r-0">
              <div class="mb-12px shrink-0 truncate text-15px font-semibold">{{ selectedSkill.name }}</div>
              <div class="skill-file-tree-scroll min-h-0 flex-1 overflow-auto rounded-6px bg-gray-50 p-8px dark:bg-#141418 lt-lg:h-280px">
                <NTree
                  class="skill-file-tree"
                  :data="treeData"
                  :selected-keys="selectedFile ? [selectedFile] : []"
                  :expanded-keys="expandedKeys"
                  block-line
                  selectable
                  @update:selected-keys="selectFile"
                  @update:expanded-keys="keys => (expandedKeys = keys as string[])"
                />
              </div>
            </aside>

            <div class="min-h-0 min-w-0 flex flex-col p-16px">
              <div class="mb-12px flex min-w-0 shrink-0 items-start justify-between gap-12px">
                <div class="min-w-0">
                  <div class="truncate text-15px font-semibold">{{ currentFile }}</div>
                </div>
                <div class="flex shrink-0 items-center gap-8px">
                  <NButton v-if="!editing" size="small" secondary @click="startEdit">
                    <template #icon><icon-material-symbols-edit-outline class="text-icon" /></template>
                    编辑
                  </NButton>
                  <template v-else>
                    <NButton size="small" secondary @click="cancelEdit">取消</NButton>
                    <NButton size="small" type="primary" :loading="saving" @click="saveFile">
                      <template #icon><icon-material-symbols-save-outline class="text-icon" /></template>
                      保存
                    </NButton>
                  </template>
                </div>
              </div>

              <div class="min-h-0 flex-1 flex flex-col overflow-hidden rounded-6px border border-gray-200 dark:border-gray-700">
                <NInput
                  v-if="editing"
                  v-model:value="editContent"
                  type="textarea"
                  :autosize="false"
                  class="min-h-0 flex-1 skill-editor"
                  placeholder="编辑当前文件内容"
                />
                <pre v-else-if="!detailLoading" class="m-0 min-h-0 flex-1 overflow-auto whitespace-pre-wrap break-words bg-white p-16px text-13px text-gray-800 leading-6 dark:bg-#141418 dark:text-gray-200"><code>{{ visibleContent }}</code></pre>
                <div v-else class="min-h-0 flex-1 p-16px text-13px text-gray-500">加载中...</div>
              </div>
            </div>
          </div>
          <NEmpty v-else class="py-120px" description="选择左侧 skill 查看内容" />
        </section>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.skills-card :deep(.n-card__content) {
  min-height: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
}

.skill-editor {
  height: 100%;
}

.skill-editor :deep(.n-input) {
  height: 100%;
}

.skill-editor :deep(.n-input-wrapper) {
  height: 100%;
  background-color: #fff;
}

.dark .skill-editor :deep(.n-input-wrapper) {
  background-color: #141418;
}

.skill-editor :deep(textarea) {
  height: 100% !important;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 13px;
  line-height: 1.5;
}

.skill-file-tree {
  min-width: max-content;
}

.skill-file-tree :deep(.n-tree-node) {
  min-width: 100%;
  width: max-content;
}

.skill-file-tree :deep(.n-tree-node-content),
.skill-file-tree :deep(.n-tree-node-content__text) {
  white-space: nowrap;
}
</style>
