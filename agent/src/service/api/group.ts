import { request } from './request';
import type { ConversationGroup } from './types';

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
    return request<null>(`/api/v1/conversation-groups/${encodeURIComponent(id)}/archive-conversations`, { method: 'POST' });
  },
  deleteArchivedConversations(id: string) {
    return request<null>(`/api/v1/conversation-groups/${encodeURIComponent(id)}/delete-archived-conversations`, { method: 'POST' });
  },
  remove(id: string) {
    return request<null>(`/api/v1/conversation-groups/${encodeURIComponent(id)}`, { method: 'DELETE' });
  }
};
