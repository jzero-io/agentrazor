import { computed, onBeforeUnmount, onMounted, ref, type Ref } from 'vue';
import type { Conversation, Turn } from '../../service/api';
import { displayConversationTitle, formatConversationDate, messagePreview } from '../../utils/conversation';
import { renderMarkdown } from '../../utils/markdown';
import { userItemText } from './useConversationTurns';


function markdownPreview(content: string, maxLines = 6, maxChars = 220) {
  const normalized = content
    .replace(/\r\n?/g, '\n')
    .split('\n')
    .map(line => line.trimEnd())
    .filter((line, index, lines) => line.trim() || (index > 0 && index < lines.length - 1));

  let text = normalized.slice(0, maxLines).join('\n').trim();
  const lineLimited = normalized.length > maxLines;
  let charLimited = false;
  if (text.length > maxChars) {
    text = text.slice(0, maxChars).replace(/\s+\S*$/, '').trim();
    charLimited = true;
  }
  return lineLimited || charLimited ? `${text}...` : text;
}

export interface MessagePreview {
  id: string;
  title: string;
  finalAnswer: string;
  finalAnswerHtml: string;
}

export function useConversationPreview(options: {
  renderedTurns: Ref<Turn[]>;
}) {
  const conversationPreview = ref<Conversation | null>(null);
  const conversationPreviewTop = ref(0);
  const selectedPreviewMessageId = ref('');
  const hoveredPreviewMessageId = ref('');

  const messagePreviews = computed<MessagePreview[]>(() => options.renderedTurns.value.flatMap(turn => {
    const userItem = turn.items.find(item => item.type === 'userMessage');
    if (!userItem) return [];
    const finalAnswer = [...turn.items]
      .reverse()
      .find(item => item.type === 'agentMessage' && item.phase === 'final_answer' && item.text?.trim());
    const finalAnswerText = finalAnswer?.text?.trim() || '';
    const finalAnswerPreview = finalAnswerText ? markdownPreview(finalAnswerText) : '';
    return [{
      id: userItem.id,
      title: messagePreview(userItemText(userItem)),
      finalAnswer: finalAnswerText,
      finalAnswerHtml: finalAnswerPreview ? renderMarkdown(finalAnswerPreview) : ''
    }];
  }));

  const activePreviewMessageId = computed(() => hoveredPreviewMessageId.value || selectedPreviewMessageId.value);
  const activePreviewIndex = computed(() =>
    messagePreviews.value.findIndex(item => item.id === activePreviewMessageId.value)
  );

  function previewTickClasses(index: number) {
    if (activePreviewIndex.value < 0) return {};
    const distance = Math.abs(index - activePreviewIndex.value);
    return {
      'is-active': distance === 0,
      'is-near-1': distance === 1,
      'is-near-2': distance === 2,
      'is-near-3': distance === 3
    };
  }

  function showConversationPreview(item: Conversation, event: MouseEvent) {
    const target = event.currentTarget;
    if (!(target instanceof HTMLElement)) return;
    const rect = target.getBoundingClientRect();
    conversationPreview.value = item;
    conversationPreviewTop.value = Math.min(rect.top, window.innerHeight - 180);
  }

  function hideConversationPreview() {
    conversationPreview.value = null;
  }

  function clearMessagePreviewSelection() {
    selectedPreviewMessageId.value = '';
    hoveredPreviewMessageId.value = '';
  }

  function handleDocumentPointerOver(event: PointerEvent) {
    if (!conversationPreview.value) return;
    const target = event.target;
    if (!(target instanceof Element) || !target.closest('.previewable-conversation-row')) {
      hideConversationPreview();
    }
  }

  onMounted(() => {
    document.addEventListener('pointerover', handleDocumentPointerOver);
    window.addEventListener('blur', hideConversationPreview);
  });

  onBeforeUnmount(() => {
    document.removeEventListener('pointerover', handleDocumentPointerOver);
    window.removeEventListener('blur', hideConversationPreview);
  });

  return {
    conversationPreview,
    conversationPreviewTop,
    messagePreviews,
    selectedPreviewMessageId,
    hoveredPreviewMessageId,
    previewTickClasses,
    showConversationPreview,
    hideConversationPreview,
    clearMessagePreviewSelection,
    displayConversationTitle,
    formatConversationDate
  };
}
