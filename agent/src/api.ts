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

export interface Message {
  id: string;
  runId: string;
  role: 'user' | 'assistant';
  content: string;
  status: string;
  createdAt: string;
}

export interface ConversationDetail {
  conversation: Conversation;
  sessionId?: string;
  messages: Message[];
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
  subscribe(id: string, onEvent: (event: StreamEvent) => void, onError: () => void) {
    const token = getToken();
    const authQuery = token ? `?token=${encodeURIComponent(token)}` : '';
    const source = new EventSource(`${apiBase}/api/v1/conversations/${encodeURIComponent(id)}/events${authQuery}`);
    source.onmessage = message => {
      try {
        const response = JSON.parse(message.data) as EventsResponse;
        if (response.event === 'stream.heartbeat') return;
        onEvent(JSON.parse(response.data) as StreamEvent);
      } catch {
        onError();
      }
    };
    source.onerror = onError;
    return () => source.close();
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
