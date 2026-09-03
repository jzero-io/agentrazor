import { request } from '../request';

export interface ConversationStats {
  totalConversations: number;
  activeConversations: number;
  runningConversations: number;
  archivedConversations: number;
  totalTokens: number;
  tokenUsageAvailable: boolean;
}

export function GetConversationStats() {
  return request<ConversationStats>({
    url: '/api/v1/conversation/stats',
    method: 'get'
  });
}
