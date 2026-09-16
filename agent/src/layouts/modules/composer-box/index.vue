<script setup lang="ts">
import { ref } from 'vue';
import { Icon } from '@iconify/vue';
import { NButton, NInput } from 'naive-ui';

defineProps<{
  modelValue: string;
  pending: boolean;
  running: boolean;
  disabled: boolean;
  label: string;
  icon: string;
  attachments: Array<{
    id: string;
    name: string;
    size: number;
    kind: 'image' | 'file';
    previewUrl?: string;
  }>;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
  keydown: [event: KeyboardEvent];
  action: [];
  addAttachments: [files: File[]];
  removeAttachment: [id: string];
}>();

const fileInput = ref<HTMLInputElement | null>(null);
const dragActive = ref(false);

function chooseFiles() {
  fileInput.value?.click();
}

function addFileList(files: FileList | null) {
  if (!files?.length) return;
  emit('addAttachments', Array.from(files));
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  addFileList(input.files);
  input.value = '';
}

function handlePaste(event: ClipboardEvent) {
  const files = Array.from(event.clipboardData?.files || []);
  if (!files.length) return;
  event.preventDefault();
  emit('addAttachments', files);
}

function handleDrop(event: DragEvent) {
  dragActive.value = false;
  const files = Array.from(event.dataTransfer?.files || []);
  if (files.length) emit('addAttachments', files);
}

function formatAttachmentSize(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${Math.ceil(size / 1024)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}
</script>

<template>
  <footer class="composer-wrap">
    <div
      class="composer"
      :class="{ 'is-dragging': dragActive }"
      @dragenter.prevent="dragActive = true"
      @dragover.prevent="dragActive = true"
      @dragleave.self="dragActive = false"
      @drop.prevent="handleDrop"
    >
      <input ref="fileInput" class="composer-file-input" type="file" multiple @change="handleFileChange" />
      <div v-if="attachments.length" class="composer-attachments">
        <article v-for="attachment in attachments" :key="attachment.id" class="composer-attachment">
          <img v-if="attachment.kind === 'image' && attachment.previewUrl" :src="attachment.previewUrl" alt="" />
          <span v-else class="composer-attachment-icon"><Icon icon="solar:file-linear" /></span>
          <span class="composer-attachment-copy">
            <strong>{{ attachment.name }}</strong>
            <small>{{ formatAttachmentSize(attachment.size) }}</small>
          </span>
          <button
            type="button"
            class="composer-attachment-remove"
            :aria-label="`移除 ${attachment.name}`"
            :title="`移除 ${attachment.name}`"
            @click="emit('removeAttachment', attachment.id)"
          >
            <Icon icon="solar:close-circle-linear" />
          </button>
        </article>
      </div>
      <n-input
        :value="modelValue"
        type="textarea"
        autosize
        :maxlength="12000"
        placeholder="给 AgentRazor 发送消息"
        @update:value="value => emit('update:modelValue', value)"
        @keydown="event => emit('keydown', event)"
        @paste="handlePaste"
      />
      <div class="composer-footer">
        <n-button
          quaternary
          circle
          class="composer-attach-button"
          aria-label="添加图片或文件"
          title="添加图片或文件（单个最大 10 MB）"
          :disabled="pending || running"
          @click="chooseFiles"
        >
          <template #icon><Icon icon="solar:paperclip-2-linear" /></template>
        </n-button>
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
