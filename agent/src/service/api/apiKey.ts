import { request } from './request';
import type { AgentApiKey, CreatedAgentApiKey } from './types';

export const apiKeyApi = {
  list() {
    return request<{ keys: AgentApiKey[] }>('/api/v1/agent/api-keys');
  },
  create() {
    return request<CreatedAgentApiKey>('/api/v1/agent/api-keys', { method: 'POST' });
  },
  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/agent/api-keys/${encodeURIComponent(id)}`, { method: 'DELETE' });
  }
};
