import { request } from '../request';

export interface ConversationStats {
  totalConversations: number;
  activeConversations: number;
  runningConversations: number;
  archivedConversations: number;
  totalTokens: number;
  tokenUsageAvailable: boolean;
}

export type ConversationTrendDimension = 'day' | 'month';

export interface ConversationTrendPoint {
  period: string;
  totalConversations: number;
  archivedConversations: number;
}

export interface ConversationTrend {
  dimension: ConversationTrendDimension;
  points: ConversationTrendPoint[];
}

export function GetConversationTrend(dimension: ConversationTrendDimension) {
  return request<ConversationTrend>({
    url: '/api/v1/conversation/trend',
    method: 'get',
    params: { dimension }
  });
}

export type TokenUsageDimension = 'day' | 'month';

export interface TokenUsageTrendPoint {
  period: string;
  tokens: number;
}

export interface TokenUsageTrend {
  dimension: TokenUsageDimension;
  points: TokenUsageTrendPoint[];
}

export function GetTokenUsageTrend(dimension: TokenUsageDimension) {
  return request<TokenUsageTrend>({
    url: '/api/v1/conversation/token-usage-trend',
    method: 'get',
    params: { dimension }
  });
}

export function GetConversationStats() {
  return request<ConversationStats>({
    url: '/api/v1/conversation/stats',
    method: 'get'
  });
}
