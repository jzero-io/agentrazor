export interface Conversation {
  id: string;
  title: string;
  status: 'active' | 'archived';
  pinnedAt?: string;
  archivedAt?: string;
  groupId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ThreadItem {
  id: string;
  type: string;
  text?: string;
  phase?: string | null;
  content?: Array<{ type?: string; text?: string; url?: string; path?: string }> | string[];
  summary?: string[];
  command?: string;
  aggregatedOutput?: string | null;
  cwd?: string;
  exitCode?: number | null;
  status?: string;
  tool?: string;
  server?: string;
  pluginId?: string | null;
  arguments?: unknown;
  result?: unknown;
  query?: string;
  changes?: Array<Record<string, unknown>>;
  durationMs?: number | null;
  dataUrl?: string;
  alt?: string;
  savedPath?: string | null;
  [key: string]: unknown;
}

export interface Turn {
  id: string;
  status: string;
  startedAt?: string;
  completedAt?: string;
  durationMs?: number;
  error?: string;
  items: ThreadItem[];
}

export interface GeneratedImage {
  id: string;
  dataUrl: string;
  alt: string;
}

export interface ConversationDetail {
  conversation: Conversation;
  sessionId?: string;
  turns: Turn[];
}

export interface StreamEvent {
  id: number;
  type: string;
  sessionId: string;
  runId?: string;
  data?: unknown;
  createdAt: string;
}

interface Envelope<T> {
  code: number;
  data: T;
  msg: string;
}

interface EventsResponse {
  id: number;
  event: string;
  data: string;
  createdAt: string;
}

const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

const TOKEN_KEY = 'agentrazor_token';
export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || '';
}
export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

// authErrorHandler is invoked when a request returns 401, so the UI can reset
// to the logged-out state.
let authErrorHandler: (() => void) | null = null;
export function setAuthErrorHandler(fn: (() => void) | null) {
  authErrorHandler = fn;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken();
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers
    }
  });
  if (response.status === 401) {
    clearToken();
    authErrorHandler?.();
    throw new Error('登录已过期，请重新登录');
  }
  const body = (await response.json()) as Envelope<T> | T;
  if (!response.ok) {
    throw new Error(`请求失败（${response.status}）`);
  }
  if (isEnvelope<T>(body)) {
    if (body.code !== 200) {
      throw new Error(body.msg || '服务请求失败');
    }
    return body.data;
  }
  return body;
}

function isEnvelope<T>(body: Envelope<T> | T): body is Envelope<T> {
  return typeof body === 'object' && body !== null && 'code' in body && 'data' in body;
}

export const conversationApi = {
  async list(): Promise<Conversation[]> {
    const result = await request<{ conversations: Conversation[] }>('/api/v1/conversations');
    return result.conversations;
  },
  create(title = '') {
    return request<Conversation>('/api/v1/conversations', {
      method: 'POST',
      body: JSON.stringify({ title })
    });
  },
  get(id: string) {
    return request<ConversationDetail>(`/api/v1/conversations/${encodeURIComponent(id)}`);
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
  send(id: string, content: string) {
    return request(`/api/v1/conversations/${encodeURIComponent(id)}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content })
    });
  },
  cancelTurn(id: string) {
    return request<null>(`/api/v1/conversations/${encodeURIComponent(id)}/turn/cancel`, {
      method: 'POST'
    });
  },
  subscribe(id: string, onEvent: (event: StreamEvent) => void, onError: () => void) {
    const token = getToken();
    // EventSource 无法携带 Authorization 头，改用 fetch 流式读取 SSE，
    // 这样鉴权与其他接口保持一致（Bearer token 走 Authorization）。
    const controller = new AbortController();
    const url = `${apiBase}/api/v1/conversations/${encodeURIComponent(id)}/events`;

    void (async () => {
      try {
        const response = await fetch(url, {
          headers: {
            Accept: 'text/event-stream',
            ...(token ? { Authorization: `Bearer ${token}` } : {})
          },
          cache: 'no-store',
          signal: controller.signal
        });
        if (!response.ok || !response.body) {
          onError();
          return;
        }
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
            const dataLine = raw.split('\n').find(line => line.startsWith('data:'));
            if (!dataLine) continue;
            const payload = dataLine.slice(5).trim();
            if (!payload) continue;
            try {
              const eventResponse = JSON.parse(payload) as EventsResponse;
              if (eventResponse.event === 'stream.heartbeat') continue;
              onEvent(JSON.parse(eventResponse.data) as StreamEvent);
            } catch {
              onError();
            }
          }
        }
      } catch (error) {
        if ((error as Error)?.name !== 'AbortError') onError();
      }
    })();

    return () => controller.abort();
  }
};

export interface ConversationGroup {
  id: string;
  name: string;
  pinnedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export const conversationGroupApi = {
  async list(): Promise<ConversationGroup[]> {
    const result = await request<{ groups: ConversationGroup[] }>('/api/v1/conversation-groups');
    return result.groups;
  },
  create(name: string) {
    return request<ConversationGroup>('/api/v1/conversation-groups', {
      method: 'POST', body: JSON.stringify({ name })
    });
  },
  update(id: string, patch: { name?: string; pinned?: boolean }) {
    return request<ConversationGroup>(`/api/v1/conversation-groups/${encodeURIComponent(id)}`, {
      method: 'PATCH', body: JSON.stringify(patch)
    });
  },
  remove(id: string) {
    return request<null>(`/api/v1/conversation-groups/${encodeURIComponent(id)}`, { method: 'DELETE' });
  }
};

export interface LoginResponse {
  token: string;
  refreshToken: string;
}

export interface UserInfo {
  userUuid: string;
  username: string;
}

export const authApi = {
  pwdLogin(username: string, password: string) {
    return request<LoginResponse>('/api/v1/auth/pwd-login', {
      method: 'POST',
      body: JSON.stringify({ username, password })
    });
  },
  getUserInfo() {
    return request<UserInfo>('/api/v1/auth/getUserInfo');
  }
};
