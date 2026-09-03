import { conversationApi, type StreamEvent } from '../../service/api';

interface ConversationStreamsOptions {
  onEvent: (event: StreamEvent) => void | Promise<void>;
  onStreamError: (conversationId: string) => void | Promise<void>;
  onReconnect: (conversationId: string) => void | Promise<void>;
}

export function useConversationStreams(options: ConversationStreamsOptions) {
  const streams = new Map<string, { close: () => void; ready: Promise<void> }>();

  function closeStream(id: string) {
    const stream = streams.get(id);
    if (!stream) return;
    streams.delete(id);
    stream.close();
  }

  function closeAll() {
    for (const stream of streams.values()) stream.close();
    streams.clear();
  }

  function ensureStream(id: string, afterId = 0): Promise<void> {
    if (!id) return Promise.reject(new Error('conversation id is required'));
    const existing = streams.get(id);
    if (existing) return existing.ready;
    const stream = conversationApi.subscribe(
      id,
      afterId,
      event => options.onEvent(event),
      () => {
        closeStream(id);
        void options.onStreamError(id);
      },
      () => options.onReconnect(id)
    );
    streams.set(id, stream);
    return stream.ready;
  }

  function closeInactive(activeId = '') {
    for (const id of [...streams.keys()]) {
      if (id !== activeId) closeStream(id);
    }
  }

  return {
    closeStream,
    closeAll,
    ensureStream,
    closeInactive
  };
}
