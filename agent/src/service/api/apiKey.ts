import { request } from './request';
import type { AgentApiKey, CreatedAgentApiKey } from './types';

export const apiKeyApi = {
  list() {
    return request<{ keys: AgentApiKey[] }>('/api/v1/auth/api-keys');
  },
  create() {
    return request<CreatedAgentApiKey>('/api/v1/auth/api-keys', { method: 'POST' });
  },
  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/auth/api-keys/${encodeURIComponent(id)}`, { method: 'DELETE' });
  }
};
