import { apiBase, expireSession, getToken, isEnvelope, refreshAccessToken, request, withAuthHeaders } from './request';
import type { Conversation, ConversationDetail, ConversationMetadata, Envelope, EventsResponse, StartedTurn, StreamEvent, WorkspaceEntry, WorkspaceFileBlob, WorkspaceFileContent } from './types';

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
  cancelTurn(id: string) {
    return request<null>(`/api/v1/conversation/${encodeURIComponent(id)}/turn/cancel`, { method: 'POST' });
  },
  async workspaceFiles(id: string): Promise<WorkspaceEntry[]> {
    const result = await request<{ entries: WorkspaceEntry[] }>(`/api/v1/conversation/${encodeURIComponent(id)}/workspace/files`);
    return result.entries;
  },
  async fetchWorkspaceResponse(path: string, retried = false): Promise<{ response: Response; pathname: string; name: string }> {
    const normalizedPath = path.startsWith('/') ? path : `/${path}`;
    const response = await fetch(`${apiBase}${normalizedPath}`, {
      headers: withAuthHeaders(),
      cache: 'no-store'
    });
    if (response.status === 401 && !retried) {
      const refreshResult = await refreshAccessToken();
      if (refreshResult === 'refreshed') return this.fetchWorkspaceResponse(path, true);
      if (refreshResult === 'expired') expireSession();
    }
    if (!response.ok) throw new Error(response.status === 404 ? '文件不存在或无权访问' : `读取文件失败（${response.status}）`);
    const pathname = decodeURIComponent(normalizedPath.split('?')[0] || '');
    const name = pathname.split('/').filter(Boolean).pop() || '文件';
    return { response, pathname, name };
  },
  async fetchWorkspaceFile(path: string): Promise<WorkspaceFileContent> {
    const { response, pathname, name } = await this.fetchWorkspaceResponse(path);
    const content = await response.text();
    return { path: pathname, name, content, contentType: response.headers.get('content-type') || '' };
  },
  async fetchWorkspaceBlob(path: string): Promise<WorkspaceFileBlob> {
    const { response, pathname, name } = await this.fetchWorkspaceResponse(path);
    const blob = await response.blob();
    return { path: pathname, name, blob, contentType: response.headers.get('content-type') || blob.type || '' };
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
