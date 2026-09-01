import type { Conversation } from '../../service/api';
import { conversationApi } from '../../service/api';

interface UseConversationTitleRefreshOptions {
  syncConversationMetadata: (conversation: Conversation) => void;
}

export function useConversationTitleRefresh(options: UseConversationTitleRefreshOptions) {
  const titleRefreshTimers = new Map<string, number>();

  function stopConversationTitleRefresh(id: string) {
    const timer = titleRefreshTimers.get(id);
    if (timer === undefined) return;
    window.clearInterval(timer);
    titleRefreshTimers.delete(id);
  }

  function stopAllConversationTitleRefresh() {
    for (const timer of titleRefreshTimers.values()) window.clearInterval(timer);
    titleRefreshTimers.clear();
  }

  function shouldRefreshConversationTitle(item?: Pick<Conversation, 'title'> | null) {
    return !item?.title?.trim();
  }

  function refreshConversationTitle(id: string) {
    void (async () => {
      try {
        const updated = await conversationApi.metadata(id);
        options.syncConversationMetadata(updated);
        if (!shouldRefreshConversationTitle(updated)) stopConversationTitleRefresh(id);
      } catch {
        // 标题刷新是体验增强，失败时保留当前会话流，下一轮继续轻量重试。
      }
    })();
  }

  function scheduleConversationTitleRefresh(id: string, item?: Pick<Conversation, 'title'> | null) {
    if (!id || !shouldRefreshConversationTitle(item) || titleRefreshTimers.has(id)) return;
    let remaining = 20;
    refreshConversationTitle(id);
    const timer = window.setInterval(() => {
      remaining -= 1;
      if (remaining <= 0) {
        stopConversationTitleRefresh(id);
        return;
      }
      refreshConversationTitle(id);
    }, 1500);
    titleRefreshTimers.set(id, timer);
  }

  return {
    scheduleConversationTitleRefresh,
    stopConversationTitleRefresh,
    stopAllConversationTitleRefresh
  };
}
