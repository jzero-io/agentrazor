import { conversationApi, type StreamEvent } from '../../service/api';

interface ConversationStreamsOptions {
  onEvent: (event: StreamEvent) => void | Promise<void>;
  onBufferedEvents: (events: StreamEvent[], dedupeSnapshotOverlap: boolean) => void | Promise<void>;
  onStreamError: (conversationId: string) => void | Promise<void>;
  onReconnect: (conversationId: string) => void | Promise<void>;
}

interface ManagedConversationStream {
  close: () => void;
  ready: Promise<void>;
  buffering: boolean;
  pendingEvents: StreamEvent[];
  synchronization?: Promise<void>;
}

export function useConversationStreams(options: ConversationStreamsOptions) {
  const streams = new Map<string, ManagedConversationStream>();

  function closeStream(id: string) {
    const stream = streams.get(id);
    if (!stream) return;
    streams.delete(id);
    stream.pendingEvents.length = 0;
    stream.close();
  }

  function closeAll() {
    for (const stream of streams.values()) stream.close();
    streams.clear();
  }

  function createStream(id: string, buffering = false): ManagedConversationStream {
    let managed!: ManagedConversationStream;
    const stream = conversationApi.subscribe(
      id,
      event => {
        if (managed.buffering) {
          managed.pendingEvents.push(event);
          return;
        }
        return options.onEvent(event);
      },
      () => {
        closeStream(id);
        void options.onStreamError(id);
      },
      () => synchronize(managed, () => options.onReconnect(id))
    );
    managed = {
      ...stream,
      buffering,
      pendingEvents: []
    };
    streams.set(id, managed);
    return managed;
  }

  function getOrCreateStream(id: string, buffering = false) {
    if (!id) throw new Error('conversation id is required');
    const existing = streams.get(id);
    if (existing) {
      if (buffering) existing.buffering = true;
      return existing;
    }
    return createStream(id, buffering);
  }

  function ensureStream(id: string): Promise<void> {
    try {
      return getOrCreateStream(id).ready;
    } catch (error) {
      return Promise.reject(error);
    }
  }

  function synchronize(stream: ManagedConversationStream, loadSnapshot: () => void | Promise<void>) {
    stream.buffering = true;
    const previous = stream.synchronization || Promise.resolve();
    const current = previous
      .catch(() => undefined)
      .then(async () => {
        await stream.ready;
        let snapshotError: unknown;
        let snapshotLoaded = false;
        try {
          await loadSnapshot();
          snapshotLoaded = true;
        } catch (error) {
          snapshotError = error;
        }
        let dedupeSnapshotOverlap = snapshotLoaded;
        while (stream.pendingEvents.length) {
          const events = stream.pendingEvents.splice(0);
          await options.onBufferedEvents(events, dedupeSnapshotOverlap);
          dedupeSnapshotOverlap = false;
        }
        if (snapshotError) throw snapshotError;
      });
    stream.synchronization = current;
    return current.finally(() => {
      if (stream.synchronization !== current) return;
      stream.synchronization = undefined;
      stream.buffering = false;
    });
  }

  function synchronizeStream(id: string, loadSnapshot: () => void | Promise<void>): Promise<void> {
    try {
      return synchronize(getOrCreateStream(id, true), loadSnapshot);
    } catch (error) {
      return Promise.reject(error);
    }
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
    synchronizeStream,
    closeInactive
  };
}
