export interface Conversation {
  id: string;
  title: string;
  status: 'active' | 'archived';
  pinnedAt?: string;
  archivedAt?: string;
  groupId?: string;
  running: boolean;
  runningStartedAt?: string;
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
  commandActions?: Array<
    | { type: 'read'; command: string; name: string; path: string }
    | { type: 'listFiles'; command: string; path: string | null }
    | { type: 'search'; command: string; query: string | null; path: string | null }
    | { type: 'unknown'; command: string }
  >;
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
  eventCursor: number;
  turns: Turn[];
}

export interface StreamEvent {
  id: number;
  type: string;
  conversationId: string;
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

export interface SendMessageResponse {
  conversationId: string;
  conversation: Conversation;
  run: {
    id: string;
    createdAt: string;
  };
}

const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

const TOKEN_KEY = 'agentrazor_token';
const REFRESH_TOKEN_KEY = 'agentrazor_refresh_token';
export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || '';
}
export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}
export function getRefreshToken() {
  return localStorage.getItem(REFRESH_TOKEN_KEY) || '';
}
export function setRefreshToken(refreshToken: string) {
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
}
export function clearRefreshToken() {
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

// authErrorHandler is invoked when a request returns 401, so the UI can reset
// to the logged-out state.
let authErrorHandler: (() => void) | null = null;
export function setAuthErrorHandler(fn: (() => void) | null) {
  authErrorHandler = fn;
}

type RefreshAccessTokenResult = 'refreshed' | 'expired' | 'failed';

// 刷新 token 的并发单飞：多个请求同时鉴权失败时只发起一次刷新，成功后一起重试。
let refreshTokenPromise: Promise<RefreshAccessTokenResult> | null = null;

async function refreshAccessToken(): Promise<RefreshAccessTokenResult> {
  if (!refreshTokenPromise) {
    refreshTokenPromise = (async () => {
      const refreshToken = getRefreshToken();
      if (!refreshToken) return 'expired';
      try {
        const response = await fetch(`${apiBase}/api/v1/auth/refreshToken`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refreshToken })
        });
        const body = (await response.json().catch(() => null)) as Envelope<LoginResponse> | LoginResponse | null;
        const envelope = body !== null && isEnvelope<LoginResponse>(body) ? body : null;
        if (response.status === 401 || envelope?.code === 40102) return 'expired';
        if (!response.ok) return 'failed';
        if (envelope && envelope.code !== 200) return envelope.code === 40102 ? 'expired' : 'failed';
        const data = envelope ? envelope.data : (body as LoginResponse | null);
        if (!data?.token || !data.refreshToken) return 'failed';
        setToken(data.token);
        setRefreshToken(data.refreshToken);
        return 'refreshed';
      } catch {
        return 'failed';
      }
    })().finally(() => {
      refreshTokenPromise = null;
    });
  }
  return refreshTokenPromise;
}

async function refreshAccessTokenOrExpire() {
  const result = await refreshAccessToken();
  if (result === 'expired') expireSession();
  return result === 'refreshed';
}

function expireSession() {
  clearToken();
  clearRefreshToken();
  authErrorHandler?.();
}

function withAuthHeaders(init?: RequestInit): Headers {
  const headers = new Headers(init?.headers);
  const hasBody = init?.body !== undefined && init.body !== null;
  if (hasBody && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const token = getToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  return headers;
}

async function request<T>(path: string, init?: RequestInit, retried = false): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: withAuthHeaders(init)
  });
  const body = (await response.json().catch(() => null)) as Envelope<T> | T | null;
  const envelope = body !== null && isEnvelope<T>(body) ? body : null;
  // 鉴权失败：HTTP 401（未来兼容）或业务码 40101（access token 过期，可刷新）
  const authFailed = response.status === 401 || envelope?.code === 40101;
  if (authFailed) {
    if (!retried) {
      const refreshResult = await refreshAccessToken();
      if (refreshResult === 'refreshed') return request<T>(path, init, true);
      if (refreshResult === 'failed') throw new Error('登录续期失败，请稍后重试');
    }
    expireSession();
    throw new Error('登录已过期，请重新登录');
  }
  if (!response.ok) {
    throw new Error(`请求失败（${response.status}）`);
  }
  if (envelope) {
    if (envelope.code !== 200) {
      // 40102 = refresh token 过期，登录态失效，直接登出
      if (envelope.code === 40102) expireSession();
      throw new Error(envelope.msg || '服务请求失败');
    }
    return envelope.data;
  }
  return body as T;
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
    return request<SendMessageResponse>(`/api/v1/conversations/${encodeURIComponent(id)}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content })
    });
  },
  cancelTurn(id: string) {
    return request<null>(`/api/v1/conversations/${encodeURIComponent(id)}/turn/cancel`, {
      method: 'POST'
    });
  },
  subscribe(id: string, afterId: number, onEvent: (event: StreamEvent) => void, onError: () => void, onReconnect?: () => void) {
    // EventSource 无法携带 Authorization 头，改用 fetch 流式读取 SSE，
    // 这样鉴权与其他接口保持一致（Bearer token 走 Authorization）。
    const controller = new AbortController();
    const url = `${apiBase}/api/v1/conversations/${encodeURIComponent(id)}/events`;
    // 断线重试标记：一次断开最多自动重连一次，避免死循环
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
        // 鉴权失败时服务端返回 HTTP 200 + JSON envelope（如 code 40101），
        // 用 Content-Type 区分：SSE 是 text/event-stream，失败响应是 JSON
        const contentType = response.headers.get('content-type') || '';
        if (!response.ok || !response.body || !contentType.includes('text/event-stream')) {
          const body = await response.clone().json().catch(() => null) as Envelope<unknown> | null;
          const authFailed = response.status === 401 || (body !== null && isEnvelope<unknown>(body) && body.code === 40101);
          if (await reconnect(authFailed)) return;
          if (!controller.signal.aborted) onError();
          return;
        }
        // 连接成功，重置重试标记：后续再次断开仍可自动重连
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
        // 流结束（服务端空闲回收/重启等）：自动重连保持事件不中断
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

export interface ConversationGroup {
  id: string;
  name: string;
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
  update(id: string, patch: { name?: string }) {
    return request<ConversationGroup>(`/api/v1/conversation-groups/${encodeURIComponent(id)}`, {
      method: 'PATCH', body: JSON.stringify(patch)
    });
  },
  archiveConversations(id: string) {
    return request<null>(`/api/v1/conversation-groups/${encodeURIComponent(id)}/archive-conversations`, {
      method: 'POST'
    });
  },
  deleteArchivedConversations(id: string) {
    return request<null>(`/api/v1/conversation-groups/${encodeURIComponent(id)}/delete-archived-conversations`, {
      method: 'POST'
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
