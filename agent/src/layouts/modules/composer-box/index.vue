<script setup lang="ts">
import { Icon } from '@iconify/vue';
import { NButton, NInput } from 'naive-ui';

defineProps<{
  modelValue: string;
  pending: boolean;
  running: boolean;
  disabled: boolean;
  label: string;
  icon: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
  keydown: [event: KeyboardEvent];
  action: [];
}>();
</script>

<template>
  <footer class="composer-wrap">
    <div class="composer">
      <n-input
        :value="modelValue"
        type="textarea"
        autosize
        :maxlength="12000"
        placeholder="给 AgentRazor 发送消息"
        @update:value="value => emit('update:modelValue', value)"
        @keydown="event => emit('keydown', event)"
      />
      <div class="composer-footer">
        <n-button
          type="primary"
          circle
          class="composer-action-button"
          :class="{ 'is-pending': pending, 'is-running': running }"
          :disabled="disabled"
          :aria-label="label"
          :title="label"
          @click="emit('action')"
        >
          <template #icon>
            <Transition name="composer-action-icon" mode="out-in">
              <Icon :key="icon" :icon="icon" />
            </Transition>
          </template>
        </n-button>
      </div>
    </div>
  </footer>
</template>
