<script setup lang="tsx">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NPagination,
  NPopconfirm,
  NTabPane,
  NTabs,
  NTag
} from 'naive-ui';
import type { DataTableColumns, FormInst, FormRules } from 'naive-ui';
import {
  CreateBuiltinAgentCharacter,
  DeleteBuiltinAgentCharacter,
  GetAgentCharacters,
  UpdateBuiltinAgentCharacter
} from '@/service/api/character';
import { useAuth } from '@/hooks/business/auth';

defineOptions({ name: 'AgentCharacters' });

const { hasAuth } = useAuth();
const canCreate = computed(() => hasAuth('v1:manage:character:create'));
const canUpdate = computed(() => hasAuth('v1:manage:character:update'));
const canDelete = computed(() => hasAuth('v1:manage:character:delete'));

const activeKind = ref<Api.Manage.AgentCharacterKind>('builtin');
const keyword = ref('');
const records = ref<Api.Manage.AgentCharacter[]>([]);
const loading = ref(false);
const saving = ref(false);
const deletingId = ref('');
const modalVisible = ref(false);
const editingId = ref('');
const formRef = ref<FormInst | null>(null);
const pagination = reactive({ current: 1, size: 10, total: 0 });
const form = reactive<Api.Manage.SaveBuiltinAgentCharacterRequest>({
  name: '',
  description: '',
  prompt: '',
  sort: 0
});

const rules: FormRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: ['input', 'blur'] }],
  prompt: [{ required: true, message: '请输入角色提示词', trigger: ['input', 'blur'] }]
};

function formatTime(value: string) {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

function promptPreview(prompt: string) {
  const compact = prompt.replace(/\s+/g, ' ').trim();
  return compact.length > 90 ? `${compact.slice(0, 90)}…` : compact;
}

async function loadCharacters() {
  loading.value = true;
  const { data, error } = await GetAgentCharacters({
    kind: activeKind.value,
    keyword: keyword.value.trim() || undefined,
    current: pagination.current,
    size: pagination.size
  });
  loading.value = false;
  if (error) return;
  records.value = data.records;
  pagination.total = data.total;
}

function search() {
  pagination.current = 1;
  loadCharacters();
}

function resetSearch() {
  keyword.value = '';
  pagination.current = 1;
  loadCharacters();
}

function resetForm() {
  editingId.value = '';
  Object.assign(form, { name: '', description: '', prompt: '', sort: 0 });
}

function openCreate() {
  resetForm();
  modalVisible.value = true;
}

function openEdit(row: Api.Manage.AgentCharacter) {
  editingId.value = row.id;
  Object.assign(form, {
    name: row.name,
    description: row.description,
    prompt: row.prompt,
    sort: row.sort
  });
  modalVisible.value = true;
}

async function saveBuiltin() {
  await formRef.value?.validate();
  saving.value = true;
  const payload = { ...form };
  const { error } = editingId.value
    ? await UpdateBuiltinAgentCharacter(editingId.value, payload)
    : await CreateBuiltinAgentCharacter(payload);
  saving.value = false;
  if (error) return;
  window.$message?.success(editingId.value ? '内置角色已更新' : '内置角色已创建');
  modalVisible.value = false;
  await loadCharacters();
}

async function deleteBuiltin(row: Api.Manage.AgentCharacter) {
  deletingId.value = row.id;
  const { error } = await DeleteBuiltinAgentCharacter(row.id);
  deletingId.value = '';
  if (error) return;
  window.$message?.success('内置角色已删除');
  if (records.value.length === 1 && pagination.current > 1) pagination.current -= 1;
  await loadCharacters();
}

const columns = computed<DataTableColumns<Api.Manage.AgentCharacter>>(() => {
  const base: DataTableColumns<Api.Manage.AgentCharacter> = [
    { title: '角色名称', key: 'name', minWidth: 150 },
    {
      title: '类型',
      key: 'builtin',
      width: 110,
      render: row => <NTag type={row.builtin ? 'success' : 'info'}>{row.builtin ? '系统内置' : '用户自定义'}</NTag>
    },
    { title: '描述', key: 'description', minWidth: 220, ellipsis: { tooltip: true } },
    {
      title: '提示词',
      key: 'prompt',
      minWidth: 320,
      ellipsis: { tooltip: true },
      render: row => promptPreview(row.prompt)
    }
  ];

  if (activeKind.value === 'custom') {
    base.push({
      title: '所属用户',
      key: 'owner',
      minWidth: 160,
      render: row => row.owner?.username || '-'
    });
  } else {
    base.push({ title: '排序', key: 'sort', width: 90, align: 'center' });
  }

  base.push({
    title: '更新时间',
    key: 'updatedAt',
    width: 190,
    render: row => formatTime(row.updatedAt)
  });

  if (activeKind.value === 'builtin' && (canUpdate.value || canDelete.value)) {
    base.push({
      title: '操作',
      key: 'operate',
      width: 160,
      fixed: 'right',
      render: row => (
        <div class="flex gap-8px">
          {canUpdate.value && (
            <NButton size="small" type="primary" ghost onClick={() => openEdit(row)}>
              编辑
            </NButton>
          )}
          {canDelete.value && (
            <NPopconfirm onPositiveClick={() => deleteBuiltin(row)}>
              {{
                default: () => '删除后，新会话将不能再选择该角色。确认删除吗？',
                trigger: () => (
                  <NButton size="small" type="error" ghost loading={deletingId.value === row.id}>
                    删除
                  </NButton>
                )
              }}
            </NPopconfirm>
          )}
        </div>
      )
    });
  }
  return base;
});

watch(activeKind, () => {
  keyword.value = '';
  pagination.current = 1;
  loadCharacters();
});

onMounted(loadCharacters);
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard :bordered="false" size="small" class="card-wrapper">
      <NCollapse>
        <NCollapseItem title="搜索" name="character-search">
          <NForm label-placement="left" :label-width="80">
            <NGrid responsive="screen" item-responsive>
              <NFormItemGi span="24 s:12 m:6" label="角色名称" class="pr-24px">
                <NInput v-model:value="keyword" clearable placeholder="请输入角色名称" @keyup.enter="search" />
              </NFormItemGi>
              <NFormItemGi span="24 m:18" class="pr-24px">
                <NSpace class="w-full" justify="end">
                  <NButton @click="resetSearch">
                    <template #icon>
                      <icon-ic-round-refresh class="text-icon" />
                    </template>
                    重置
                  </NButton>
                  <NButton type="primary" ghost @click="search">
                    <template #icon>
                      <icon-ic-round-search class="text-icon" />
                    </template>
                    搜索
                  </NButton>
                </NSpace>
              </NFormItemGi>
            </NGrid>
          </NForm>
        </NCollapseItem>
      </NCollapse>
    </NCard>

    <NCard title="Agent 角色管理" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper">
      <template #header-extra>
        <NSpace>
          <NButton v-if="activeKind === 'builtin' && canCreate" size="small" type="primary" ghost @click="openCreate">
            <template #icon>
              <icon-ic-round-plus class="text-icon" />
            </template>
            新增内置角色
          </NButton>
          <NButton size="small" @click="loadCharacters">
            <template #icon>
              <icon-mdi-refresh class="text-icon" />
            </template>
            刷新
          </NButton>
        </NSpace>
      </template>

      <div class="mb-16px flex items-center">
        <NTabs v-model:value="activeKind" type="segment" class="max-w-360px">
          <NTabPane name="builtin" tab="系统内置角色" />
          <NTabPane name="custom" tab="用户自定义角色" />
        </NTabs>
      </div>

      <NDataTable
        :columns="columns"
        :data="records"
        :loading="loading"
        :row-key="row => row.id"
        :scroll-x="activeKind === 'custom' ? 1450 : 1100"
        flex-height
        class="min-h-360px sm:h-[calc(100%-104px)]"
      />
      <div class="mt-16px flex justify-end">
        <NPagination
          v-model:page="pagination.current"
          v-model:page-size="pagination.size"
          show-size-picker
          :page-sizes="[10, 20, 50]"
          :item-count="pagination.total"
          @update:page="loadCharacters"
          @update:page-size="
            pagination.current = 1;
            loadCharacters();
          "
        />
      </div>
    </NCard>

    <NModal
      v-model:show="modalVisible"
      preset="card"
      :title="editingId ? '编辑内置角色' : '新增内置角色'"
      class="max-w-[calc(100vw-32px)] w-720px"
      :mask-closable="false"
    >
      <NForm ref="formRef" :model="form" :rules="rules" label-placement="top">
        <div class="grid grid-cols-[1fr_120px] gap-16px lt-sm:grid-cols-1">
          <NFormItem label="角色名称" path="name">
            <NInput v-model:value="form.name" maxlength="80" show-count />
          </NFormItem>
          <NFormItem label="排序" path="sort">
            <NInputNumber v-model:value="form.sort" class="w-full" :min="-9999" :max="9999" />
          </NFormItem>
        </div>
        <NFormItem label="角色描述" path="description">
          <NInput v-model:value="form.description" maxlength="240" show-count />
        </NFormItem>
        <NFormItem label="角色提示词" path="prompt">
          <NInput
            v-model:value="form.prompt"
            type="textarea"
            :autosize="{ minRows: 10, maxRows: 18 }"
            maxlength="16000"
            show-count
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-8px">
          <NButton @click="modalVisible = false">取消</NButton>
          <NButton type="primary" :loading="saving" @click="saveBuiltin">保存</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped></style>
