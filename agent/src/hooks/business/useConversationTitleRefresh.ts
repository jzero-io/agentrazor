import type { Conversation, ConversationMetadata } from '../../service/api';
import { conversationApi } from '../../service/api';

interface UseConversationTitleRefreshOptions {
  syncConversationMetadata: (conversation: ConversationMetadata) => void;
}

export function useConversationTitleRefresh(options: UseConversationTitleRefreshOptions) {
  interface TitleRefreshJob {
    remaining: number;
    timer?: number;
  }

  const titleRefreshJobs = new Map<string, TitleRefreshJob>();

  function stopConversationTitleRefresh(id: string) {
    const job = titleRefreshJobs.get(id);
    if (job?.timer !== undefined) window.clearTimeout(job.timer);
    titleRefreshJobs.delete(id);
  }

  function stopAllConversationTitleRefresh() {
    for (const job of titleRefreshJobs.values()) {
      if (job.timer !== undefined) window.clearTimeout(job.timer);
    }
    titleRefreshJobs.clear();
  }

  function shouldRefreshConversationTitle(item?: Pick<Conversation, 'title'> | null) {
    return !item?.title?.trim();
  }

  async function refreshConversationTitle(id: string, job: TitleRefreshJob) {
    try {
      const updated = await conversationApi.metadata(id);
      if (titleRefreshJobs.get(id) !== job) return;
      options.syncConversationMetadata(updated);
      if (!shouldRefreshConversationTitle(updated)) {
        stopConversationTitleRefresh(id);
        return;
      }
    } catch {
      // 标题刷新是体验增强，失败时保留当前会话流，下一轮继续轻量重试。
    }

    if (titleRefreshJobs.get(id) !== job || --job.remaining <= 0) {
      stopConversationTitleRefresh(id);
      return;
    }
    job.timer = window.setTimeout(() => void refreshConversationTitle(id, job), 1500);
  }

  function scheduleConversationTitleRefresh(id: string, item?: Pick<Conversation, 'title'> | null) {
    if (!id || !shouldRefreshConversationTitle(item) || titleRefreshJobs.has(id)) return;
    const job: TitleRefreshJob = { remaining: 20 };
    titleRefreshJobs.set(id, job);
    void refreshConversationTitle(id, job);
  }

  return {
    scheduleConversationTitleRefresh,
    stopConversationTitleRefresh,
    stopAllConversationTitleRefresh
  };
}
