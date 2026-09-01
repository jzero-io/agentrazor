import { conversationApi, type StreamEvent } from '../../service/api';

interface ConversationStreamsOptions {
  onEvent: (event: StreamEvent) => void | Promise<void>;
  onStreamError: (conversationId: string) => void | Promise<void>;
  onReconnect: (conversationId: string) => void | Promise<void>;
  isProcessing: (conversationId: string) => boolean;
}

export function useConversationStreams(options: ConversationStreamsOptions) {
  const closers = new Map<string, () => void>();

  function closeStream(id: string) {
    const close = closers.get(id);
    if (!close) return;
    closers.delete(id);
    close();
  }

  function closeAll() {
    for (const close of closers.values()) close();
    closers.clear();
  }

  function ensureStream(id: string, afterId = 0) {
    if (!id || closers.has(id)) return;
    const close = conversationApi.subscribe(
      id,
      afterId,
      event => void options.onEvent(event),
      () => {
        closeStream(id);
        void options.onStreamError(id);
      },
      () => void options.onReconnect(id)
    );
    closers.set(id, close);
  }

  function closeIdle(activeId = '') {
    for (const id of [...closers.keys()]) {
      if (id !== activeId && !options.isProcessing(id)) closeStream(id);
    }
  }

  return {
    closeStream,
    closeAll,
    ensureStream,
    closeIdle
  };
}
