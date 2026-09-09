import { request } from '../request';

export interface TokenUsageTurn {
  turnId: string;
  inputTokens: number;
  cachedInputTokens: number;
  cacheWriteInputTokens: number;
  outputTokens: number;
  reasoningOutputTokens: number;
  totalTokens: number;
  modelContextWindow?: number;
  startedAt: string;
  updatedAt: string;
}

export interface TokenUsageConversation {
  conversationId: string;
  totalTokens: number;
  turnCount: number;
  firstUsedAt: string;
  lastUsedAt: string;
  turns: TokenUsageTurn[];
}

export interface TokenUsageAccount {
  userUuid: string;
  username: string;
  nickname: string;
  totalTokens: number;
  conversationCount: number;
  turnCount: number;
}

export interface TokenUsageSummary {
  inputTokens: number;
  cachedInputTokens: number;
  cacheWriteInputTokens: number;
  outputTokens: number;
  reasoningOutputTokens: number;
  totalTokens: number;
}

export interface TokenUsageDetails {
  current: number;
  size: number;
  total: number;
  summary: TokenUsageSummary;
  accounts: TokenUsageAccount[];
}

export interface TokenUsageConversations {
  current: number;
  size: number;
  total: number;
  conversations: TokenUsageConversation[];
}

export interface TokenUsageDetailsParams {
  current: number;
  size: number;
  username?: string;
}

export interface TokenUsageConversationsParams {
  current: number;
  size: number;
  userUuid: string;
  conversationId?: string;
}

export interface TokenQuotaGlobal {
  fiveHourLimitTokens: number;
  sevenDayLimitTokens: number;
}

export interface TokenQuotaUser {
  userUuid: string;
  configured: boolean;
  enabled: boolean;
  fiveHourDisabled: boolean;
  fiveHourLimitTokens?: number;
  sevenDayLimitTokens?: number;
  effectiveFiveHourLimit?: number;
  effectiveSevenDayLimit: number;
  fiveHourUsedTokens: number;
  sevenDayUsedTokens: number;
  fiveHourResetAt?: string;
  sevenDayResetAt?: string;
  quotaResetAt?: string;
}

export interface SaveTokenQuotaUserRequest {
  enabled: boolean;
  fiveHourDisabled: boolean;
  fiveHourLimitTokens?: number;
  sevenDayLimitTokens?: number;
}

export function GetTokenUsageDetails(params: TokenUsageDetailsParams) {
  return request<TokenUsageDetails>({
    url: '/api/v1/conversation/token-usage-details',
    method: 'get',
    params
  });
}

export function GetTokenUsageConversations(params: TokenUsageConversationsParams) {
  return request<TokenUsageConversations>({
    url: '/api/v1/conversation/token-usage-conversations',
    method: 'get',
    params
  });
}

export function GetTokenQuotaGlobal() {
  return request<{ quota: TokenQuotaGlobal }>({
    url: '/api/v1/manage/agent/token-quota/global',
    method: 'get'
  });
}

export function SaveTokenQuotaGlobal(data: TokenQuotaGlobal) {
  return request<Record<string, never>>({
    url: '/api/v1/manage/agent/token-quota/global',
    method: 'post',
    data
  });
}

export function GetUserTokenQuota(userUuid: string) {
  return request<{ quota: TokenQuotaUser }>({
    url: `/api/v1/manage/agent/token-quota/users/${encodeURIComponent(userUuid)}`,
    method: 'get'
  });
}

export function SaveUserTokenQuota(userUuid: string, data: SaveTokenQuotaUserRequest) {
  return request<Record<string, never>>({
    url: `/api/v1/manage/agent/token-quota/users/${encodeURIComponent(userUuid)}`,
    method: 'post',
    data
  });
}

export function DeleteUserTokenQuota(userUuid: string) {
  return request<Record<string, never>>({
    url: `/api/v1/manage/agent/token-quota/users/${encodeURIComponent(userUuid)}`,
    method: 'delete'
  });
}

export function ResetUserTokenQuota(userUuid: string) {
  return request<Record<string, never>>({
    url: `/api/v1/manage/agent/token-quota/users/${encodeURIComponent(userUuid)}/reset`,
    method: 'post'
  });
}
