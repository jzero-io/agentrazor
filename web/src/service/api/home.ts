import { request } from '../request';

export interface HomeOverview {
  agentRunning: boolean;
  activeProvider: string;
  model: string;
  modelName: string;
  totalTokens: number;
  skillCount: number;
}

export function GetHomeOverview() {
  return request<HomeOverview>({
    url: '/api/v1/home/overview',
    method: 'get'
  });
}
