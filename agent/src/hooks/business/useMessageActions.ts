import { onBeforeUnmount, ref } from 'vue';
import type { ThreadItem, Turn } from '../../service/api';
import { writeClipboardText } from '../../utils/clipboard';
import { formatMessageTime } from '../../utils/conversation';
import { userItemText } from './useConversationTurns';

interface UseMessageActionsOptions {
  parseAgentMarkdown: (content: string) => string;
  showError: (message: string) => void;
}

export function useMessageActions(options: UseMessageActionsOptions) {
  const copiedMessageId = ref('');
  let copiedMessageTimer: number | undefined;

  function messageCopyText(item: ThreadItem) {
    if (item.type === 'userMessage') return userItemText(item);
    if (typeof item.text === 'string') return options.parseAgentMarkdown(item.text) || item.text;
    if (typeof item.aggregatedOutput === 'string') return item.aggregatedOutput;
    return '';
  }

  async function copyMessage(item: ThreadItem) {
    const text = messageCopyText(item).trim();
    if (!text) return;
    try {
      await writeClipboardText(text);
      copiedMessageId.value = item.id;
      window.clearTimeout(copiedMessageTimer);
      copiedMessageTimer = window.setTimeout(() => {
        if (copiedMessageId.value === item.id) copiedMessageId.value = '';
      }, 1200);
    } catch (error) {
      options.showError(error instanceof Error ? error.message : '复制失败');
    }
  }

  function assistantMessageTime(turn: Turn) {
    return formatMessageTime(turn.completedAt || turn.startedAt);
  }

  function clearCopiedMessageTimer() {
    window.clearTimeout(copiedMessageTimer);
  }

  onBeforeUnmount(clearCopiedMessageTimer);

  return {
    copiedMessageId,
    messageCopyText,
    copyMessage,
    assistantMessageTime,
    clearCopiedMessageTimer
  };
}
