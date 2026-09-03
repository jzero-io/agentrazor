import { apiBase, expireSession, getToken, isEnvelope, refreshAccessToken, refreshAccessTokenOrExpire, request, withAuthHeaders } from './request';
import type { Conversation, ConversationDetail, Envelope, EventsResponse, SendMessageResponse, StreamEvent, WorkspaceEntry, WorkspaceFileBlob, WorkspaceFileContent } from './types';

export const conversationApi = {
  async list(): Promise<Conversation[]> {
    const result = await request<{ conversations: Conversation[] }>('/api/v1/conversations');
    return result.conversations;
  },
  get(id: string) {
    return request<ConversationDetail>(`/api/v1/conversations/${encodeURIComponent(id)}`);
  },
  metadata(id: string) {
    return request<Conversation>(`/api/v1/conversations/${encodeURIComponent(id)}/metadata`);
  },
  update(id: string, changes: { title?: string; pinned?: boolean; archived?: boolean; groupId?: string }) {
    return request<Conversation>(`/api/v1/conversations/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: JSON.stringify(changes)
    });
  },
  remove(id: string) {
    return request<null>(`/api/v1/conversations/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },
  send(conversationId: string, content: string, groupId = '') {
    return request<SendMessageResponse>('/api/v1/conversations', {
      method: 'POST',
      body: JSON.stringify({ conversationId: conversationId || undefined, groupId: groupId || undefined, content })
    });
  },
  cancelTurn(id: string) {
    return request<null>(`/api/v1/conversations/${encodeURIComponent(id)}/turn/cancel`, { method: 'POST' });
  },
  async workspaceFiles(id: string): Promise<WorkspaceEntry[]> {
    const result = await request<{ entries: WorkspaceEntry[] }>(`/api/v1/conversations/${encodeURIComponent(id)}/workspace/files`);
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
  subscribe(id: string, afterId: number, onEvent: (event: StreamEvent) => void, onError: () => void, onReconnect?: () => void) {
    const controller = new AbortController();
    const url = `${apiBase}/api/v1/conversations/${encodeURIComponent(id)}/events`;
    let retried = false;
    let lastEventId = Math.max(0, Number(afterId) || 0);

    const reconnect = async (refresh = false): Promise<boolean> => {
      if (controller.signal.aborted || retried) return false;
      retried = true;
      await new Promise(resolve => setTimeout(resolve, 1000));
      if (refresh && !(await refreshAccessTokenOrExpire())) return false;
      onReconnect?.();
      await connect();
      return true;
    };

    const connect = async (): Promise<void> => {
      if (controller.signal.aborted) return;
      try {
        const token = getToken();
        const response = await fetch(url, {
          headers: {
            Accept: 'text/event-stream',
            ...(lastEventId > 0 ? { 'Last-Event-ID': String(lastEventId) } : {}),
            ...(token ? { Authorization: `Bearer ${token}` } : {})
          },
          cache: 'no-store',
          signal: controller.signal
        });
        const contentType = response.headers.get('content-type') || '';
        if (!response.ok || !response.body || !contentType.includes('text/event-stream')) {
          const body = await response.clone().json().catch(() => null) as Envelope<unknown> | null;
          const authFailed = response.status === 401 || (body !== null && isEnvelope<unknown>(body) && body.code === 40101);
          if (await reconnect(authFailed)) return;
          if (!controller.signal.aborted) onError();
          return;
        }
        retried = false;
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
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
            try {
              const eventResponse = JSON.parse(payload) as EventsResponse;
              if (eventResponse.event === 'stream.heartbeat') continue;
              if (eventResponse.id <= lastEventId) continue;
              lastEventId = eventResponse.id;
              onEvent(JSON.parse(eventResponse.data) as StreamEvent);
            } catch {
              onError();
            }
          }
        }
        if (await reconnect(false)) return;
        if (!controller.signal.aborted) onError();
      } catch (error) {
        if ((error as Error)?.name !== 'AbortError') {
          if (await reconnect(false)) return;
          if (!controller.signal.aborted) onError();
        }
      }
    };

    void connect();
    return () => controller.abort();
  }
};
