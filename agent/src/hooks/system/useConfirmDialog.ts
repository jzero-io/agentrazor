import { ref } from 'vue';

export function useConfirmDialog() {
  const visible = ref(false);
  const title = ref('');
  const content = ref('');
  const positiveText = ref('删除');
  const loading = ref(false);
  let action: (() => Promise<void>) | undefined;

  function open(nextTitle: string, nextContent: string, nextPositiveText: string, nextAction: () => Promise<void>) {
    title.value = nextTitle;
    content.value = nextContent;
    positiveText.value = nextPositiveText;
    action = nextAction;
    visible.value = true;
  }

  function close() {
    if (loading.value) return;
    visible.value = false;
    action = undefined;
  }

  async function submit() {
    if (!action || loading.value) return;
    loading.value = true;
    try {
      await action();
      visible.value = false;
      action = undefined;
    } finally {
      loading.value = false;
    }
  }

  return {
    visible,
    title,
    content,
    positiveText,
    loading,
    open,
    close,
    submit
  };
}
