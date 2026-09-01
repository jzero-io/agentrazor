import { nextTick, ref, type ComponentPublicInstance, type Ref } from 'vue';

function isMessagePaneNearBottom(pane: HTMLElement, threshold = 96) {
  return pane.scrollHeight - pane.scrollTop - pane.clientHeight <= threshold;
}

export function useMessagePaneScroll(options: {
  selectedPreviewMessageId: Ref<string>;
}) {
  const messagePane = ref<HTMLElement>();
  const autoScrollEnabled = ref(true);

  function setMessagePane(el: Element | ComponentPublicInstance | null) {
    messagePane.value = el instanceof HTMLElement ? el : undefined;
  }

  function handleMessageScroll(event: Event) {
    const pane = event.currentTarget;
    if (pane instanceof HTMLElement) {
      autoScrollEnabled.value = isMessagePaneNearBottom(pane);
    }
  }

  async function scrollToBottom(options: { force?: boolean } = {}) {
    await nextTick();
    await new Promise(resolve => requestAnimationFrame(resolve));
    let pane = messagePane.value || document.querySelector<HTMLElement>('.message-pane');
    for (let i = 0; i < 80 && !pane; i++) {
      await new Promise(resolve => setTimeout(resolve, 50));
      pane = messagePane.value || document.querySelector<HTMLElement>('.message-pane');
    }
    if (!pane) return;
    if (!options.force && !autoScrollEnabled.value && !isMessagePaneNearBottom(pane)) return;
    pane.scrollTo({ top: pane.scrollHeight, behavior: 'auto' });
    autoScrollEnabled.value = true;
  }

  function jumpToMessage(id: string) {
    options.selectedPreviewMessageId.value = id;
    const target = document.getElementById(`message-${id}`);
    if (!target) return;
    target.scrollIntoView({ behavior: 'smooth', block: 'start' });
    target.focus({ preventScroll: true });
  }

  function enableAutoScroll() {
    autoScrollEnabled.value = true;
  }

  return {
    messagePane,
    autoScrollEnabled,
    setMessagePane,
    handleMessageScroll,
    scrollToBottom,
    jumpToMessage,
    enableAutoScroll
  };
}
