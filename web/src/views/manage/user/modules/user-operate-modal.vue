<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useLoading } from '@sa/hooks';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { AddUser, EditUser, GetAllRoles } from '@/service/api';
import { $t } from '@/locales';
import { enableStatusOptions } from '@/constants/business';

defineOptions({
  name: 'UserOperateDrawer'
});

interface Props {
  /** the type of operation */
  operateType: NaiveUI.TableOperateType;
  /** the edit row data */
  rowData?: Api.Manage.User | null;
}

const props = defineProps<Props>();

interface Emits {
  (e: 'submitted'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const { formRef, validate, restoreValidation } = useNaiveForm();
const { defaultRequiredRule, createRequiredRule, patternRules } = useFormRules();
const { loading: confirmLoading, startLoading: confirmStartLoding, endLoading: confirmEndLoading } = useLoading();

const title = computed(() => {
  const titles: Record<NaiveUI.TableOperateType, string> = {
    add: $t('page.manage.user.addUser'),
    edit: $t('page.manage.user.editUser')
  };
  return titles[props.operateType];
});

type Model = Pick<
  Api.Manage.AddUserRequest,
  'username' | 'nickName' | 'userPhone' | 'userEmail' | 'userRoles' | 'status' | 'password'
>;

const model: Model = reactive(createDefaultModel());

function createDefaultModel(): Model {
  return {
    username: '',
    password: '',
    nickName: '',
    userPhone: '',
    userEmail: '',
    userRoles: [],
    status: null
  };
}

type RuleKey = Extract<keyof Model, 'username' | 'status' | 'password' | 'userRoles'>;

const rules: Record<RuleKey, App.Global.FormRule | App.Global.FormRule[]> = {
  username: [createRequiredRule($t('form.required')), patternRules.username],
  status: defaultRequiredRule,
  password: defaultRequiredRule,
  userRoles: {
    type: 'array',
    required: true,
    min: 1,
    message: $t('form.required'),
    trigger: 'change'
  }
};

type RoleOption = CommonType.Option<string> & { disabled?: boolean };

/** the enabled role options */
const roleOptions = ref<RoleOption[]>([]);

function appendCurrentUserRoleOptions(options: RoleOption[]) {
  if (props.operateType !== 'edit' || !props.rowData) return options;

  const optionMap = new Map(options.map(item => [item.value, item]));
  props.rowData.userRoles?.forEach((roleCode, index) => {
    if (!roleCode || optionMap.has(roleCode)) return;
    optionMap.set(roleCode, {
      label: props.rowData?.userRoleNames?.[index] || roleCode,
      value: roleCode,
      disabled: true
    });
  });

  return [...optionMap.values()];
}

async function getRoleOptions() {
  const { error, data } = await GetAllRoles();

  if (!error) {
    const options = data.map(item => ({
      label: item.roleName,
      value: item.roleCode
    }));

    roleOptions.value = appendCurrentUserRoleOptions(options);
  }
}

function handleInitModel() {
  Object.assign(model, createDefaultModel());

  if (props.operateType === 'edit' && props.rowData) {
    Object.assign(model, props.rowData);
  }
}

function closeModel() {
  visible.value = false;
}

async function handleSubmit() {
  if (props.operateType === 'add') {
    await validate();
    // request
    const addUserData: Api.Manage.AddUserRequest = {
      username: model.username,
      nickName: model.nickName,
      userPhone: model.userPhone,
      userEmail: model.userEmail,
      userRoles: model.userRoles,
      password: model.password,
      status: model.status
    };
    confirmStartLoding();
    const { error } = await AddUser(addUserData);
    if (!error) {
      window.$message?.success($t('common.addSuccess'));
      closeModel();
      emit('submitted');
    }
    confirmEndLoading();
  } else if (props.operateType === 'edit') {
    await validate();
    // request
    const editUserData: Api.Manage.EditUserRequest = {
      uuid: props.rowData?.uuid,
      username: model.username,
      nickName: model.nickName,
      userPhone: model.userPhone,
      userEmail: model.userEmail,
      userRoles: model.userRoles,
      status: model.status
    };
    confirmStartLoding();
    const { error } = await EditUser(editUserData);
    if (!error) {
      window.$message?.success($t('common.updateSuccess'));
      closeModel();
      emit('submitted');
    }
    confirmEndLoading();
  }
}

watch(visible, () => {
  if (visible.value) {
    handleInitModel();
    restoreValidation();
    getRoleOptions();
  }
});
</script>

<template>
  <NModal v-model:show="visible" :title="title" preset="card" class="w-600px">
    <NScrollbar class="h-408px pr-20px">
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="100">
        <NFormItem :label="$t('page.manage.user.username')" path="username">
          <NInput
            v-model:value="model.username"
            :placeholder="$t('page.manage.user.form.username')"
            :maxlength="20"
            show-count
            :disabled="props.operateType === 'edit'"
          />
        </NFormItem>
        <NFormItem :label="$t('page.manage.user.nickName')" path="nickName">
          <NInput v-model:value="model.nickName" :placeholder="$t('page.manage.user.form.nickName')" />
        </NFormItem>
        <NFormItem :label="$t('page.manage.user.userPhone')" path="userPhone">
          <NInput v-model:value="model.userPhone" :placeholder="$t('page.manage.user.form.userPhone')" />
        </NFormItem>
        <NFormItem :label="$t('page.manage.user.userEmail')" path="email">
          <NInput v-model:value="model.userEmail" :placeholder="$t('page.manage.user.form.userEmail')" />
        </NFormItem>
        <NFormItem :label="$t('page.manage.user.userStatus')" path="status">
          <NRadioGroup v-model:value="model.status">
            <NRadio v-for="item in enableStatusOptions" :key="item.value" :value="item.value" :label="$t(item.label)" />
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="$t('page.manage.user.userRole')" path="userRoles">
          <NSelect
            v-model:value="model.userRoles"
            multiple
            :options="roleOptions"
            :placeholder="$t('page.manage.user.form.userRole')"
          />
        </NFormItem>
        <NFormItem v-if="props.operateType === 'add'" :label="$t('page.manage.user.password')" path="password">
          <NInput v-model:value="model.password" :placeholder="$t('page.manage.user.form.password')" />
        </NFormItem>
      </NForm>
    </NScrollbar>
    <template #footer>
      <NSpace :size="16" justify="end">
        <NButton @click="closeModel">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="confirmLoading" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
