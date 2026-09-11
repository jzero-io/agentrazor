import { apiBase, expireSession, getToken, isEnvelope, refreshAccessToken, request } from './request';
import type { Conversation, ConversationDetail, ConversationMetadata, Envelope, EventsResponse, StartedTurn, StreamEvent, TokenQuotaStatus, WorkspaceEntry, WorkspaceFileBlob, WorkspaceFileContent } from './types';
interface WorkspaceFilePayload {
  name: string;
  contentType: string;
  dataBase64: string;
}

function decodeBase64(value: string) {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
  return bytes;
}


export const conversationApi = {
  async list(): Promise<Conversation[]> {
    const result = await request<{ conversations: Conversation[] }>('/api/v1/conversation');
    return result.conversations;
  },
  get(id: string) {
    return request<ConversationDetail>(`/api/v1/conversation/${encodeURIComponent(id)}`);
  },
  metadata(id: string) {
    return request<ConversationMetadata>(`/api/v1/conversation/${encodeURIComponent(id)}/metadata`);
  },
  update(id: string, changes: { title?: string; pinned?: boolean; archived?: boolean; groupId?: string }) {
    return request<Conversation>(`/api/v1/conversation/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: JSON.stringify(changes)
    });
  },
  remove(id: string) {
    return request<null>(`/api/v1/conversation/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },
  create(groupId = '') {
    return request<Conversation>('/api/v1/conversation', {
      method: 'POST',
      body: JSON.stringify({ groupId: groupId || undefined })
    });
  },
  send(conversationId: string, content: string) {
    return request<StartedTurn>(`/api/v1/conversation/${encodeURIComponent(conversationId)}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content })
    });
  },
  tokenQuota() {
    return request<TokenQuotaStatus>('/api/v1/conversation/token-quota');
  },
  cancelTurn(id: string) {
    return request<null>(`/api/v1/conversation/${encodeURIComponent(id)}/turn/cancel`, { method: 'POST' });
  },
  async workspaceFiles(id: string): Promise<WorkspaceEntry[]> {
    const result = await request<{ files: WorkspaceEntry[] }>(`/api/v1/conversation/${encodeURIComponent(id)}/workspace/files`);
    return result.files;
  },
  async fetchWorkspacePayload(path: string): Promise<{ payload: WorkspaceFilePayload; pathname: string }> {
    const pathname = path.startsWith('/') ? path : '/' + path;
    const payload = await request<WorkspaceFilePayload>(pathname, { cache: 'no-store' });
    return { payload, pathname };
  },
  async fetchWorkspaceFile(path: string): Promise<WorkspaceFileContent> {
    const { payload, pathname } = await this.fetchWorkspacePayload(path);
    const content = new TextDecoder().decode(decodeBase64(payload.dataBase64));
    return { path: pathname, name: payload.name, content, contentType: payload.contentType };
  },
  async fetchWorkspaceBlob(path: string): Promise<WorkspaceFileBlob> {
    const { payload, pathname } = await this.fetchWorkspacePayload(path);
    const bytes = decodeBase64(payload.dataBase64);
    const data = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
    const blob = new Blob([data], { type: payload.contentType });
    return { path: pathname, name: payload.name, blob, contentType: payload.contentType || blob.type };
  },
  subscribe(
    id: string,
    onEvent: (event: StreamEvent) => void | Promise<void>,
    onError: () => void,
    onReconnect?: () => void | Promise<void>
  ) {
    const controller = new AbortController();
    const url = `${apiBase}/api/v1/conversation/${encodeURIComponent(id)}/events`;
    let reconnectAttempts = 0;
    let connected = false;
    let terminal = false;
    let readySettled = false;
    let resolveReady!: () => void;
    let rejectReady!: (error: Error) => void;
    const ready = new Promise<void>((resolve, reject) => {
      resolveReady = resolve;
      rejectReady = reject;
    });
    void ready.catch(() => undefined);

    const markReady = () => {
      if (readySettled) return;
      readySettled = true;
      resolveReady();
    };

    const fail = () => {
      if (terminal) return;
      terminal = true;
      if (!readySettled) {
        readySettled = true;
        rejectReady(new Error('conversation stream connection failed'));
      }
      onError();
    };

    const waitForReconnect = () => new Promise<boolean>(resolve => {
      if (controller.signal.aborted) {
        resolve(false);
        return;
      }
      const delay = Math.min(1000 * (2 ** Math.min(reconnectAttempts, 4)), 15000);
      reconnectAttempts += 1;
      const finish = (ready: boolean) => {
        window.clearTimeout(timer);
        controller.signal.removeEventListener('abort', abort);
        resolve(ready);
      };
      const abort = () => finish(false);
      const timer = window.setTimeout(() => finish(true), delay);
      controller.signal.addEventListener('abort', abort, { once: true });
    });

    const retryResponse = async (response: Response, body: Envelope<unknown> | null) => {
      const authFailed = response.status === 401 || (body !== null && isEnvelope<unknown>(body) && body.code === 40101);
      if (authFailed) {
        const refreshResult = await refreshAccessToken();
        if (refreshResult === 'refreshed') {
          reconnectAttempts = 0;
          return true;
        }
        if (refreshResult === 'expired') {
          expireSession();
          fail();
          return false;
        }
        return waitForReconnect();
      }
      if (response.status >= 400 && response.status < 500 && response.status !== 408 && response.status !== 429) {
        fail();
        return false;
      }
      return waitForReconnect();
    };

    const connect = async (): Promise<void> => {
      while (!controller.signal.aborted && !terminal) {
        let response: Response;
        try {
          const token = getToken();
          response = await fetch(url, {
            headers: {
              Accept: 'text/event-stream',
              ...(token ? { Authorization: `Bearer ${token}` } : {})
            },
            cache: 'no-store',
            signal: controller.signal
          });
        } catch (error) {
          if ((error as Error)?.name === 'AbortError') return;
          if (!(await waitForReconnect())) return;
          continue;
        }

        const contentType = response.headers.get('content-type') || '';
        if (!response.ok || !response.body || !contentType.includes('text/event-stream')) {
          const body = await response.clone().json().catch(() => null) as Envelope<unknown> | null;
          if (!(await retryResponse(response, body))) return;
          continue;
        }

        let reconcileAfterReady = connected;
        connected = true;
        reconnectAttempts = 0;
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        try {
          for (;;) {
            const { done, value } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            let sep: number;
            while ((sep = buffer.indexOf('\n\n')) >= 0 || (sep = buffer.indexOf('\r\n\r\n')) >= 0) {
              const sepLength = buffer.startsWith('\r\n', sep) ? 4 : 2;
              const raw = buffer.slice(0, sep);
              buffer = buffer.slice(sep + sepLength);
              const payload = raw
                .split(/\r?\n/)
                .filter(line => line.startsWith('data:'))
                .map(line => line.slice(5).trimStart())
                .join('\n')
                .trim();
              if (!payload) continue;
              const eventResponse = JSON.parse(payload) as EventsResponse;
              if (eventResponse.event === 'stream.ready') {
                markReady();
                if (reconcileAfterReady) {
                  reconcileAfterReady = false;
                  void Promise.resolve(onReconnect?.()).catch(() => undefined);
                }
                continue;
              }
              if (eventResponse.event === 'stream.heartbeat') continue;
              await onEvent(JSON.parse(eventResponse.data) as StreamEvent);
            }
          }
        } catch (error) {
          if ((error as Error)?.name === 'AbortError') return;
        }
        // A valid stream always sends stream.ready before it can end. The SSE
        // handler closes immediately for a missing or inaccessible conversation;
        // treating that empty response as reconnectable would leave `ready`
        // pending forever and keep the conversation page spinning.
        if (!readySettled) {
          fail();
          return;
        }
        if (!(await waitForReconnect())) return;
      }
    };

    void connect();
    return {
      ready,
      close: () => {
        controller.abort();
        if (!readySettled) {
          readySettled = true;
          rejectReady(new Error('conversation stream connection closed'));
        }
      }
    };
  }
};
