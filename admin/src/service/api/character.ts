import { request } from '../request';

export function GetAgentCharacters(params: Api.Manage.AgentCharacterSearchParams) {
  return request<Api.Manage.AgentCharacterList>({
    url: '/api/v1/manage/agent/characters',
    method: 'get',
    params
  });
}

export function CreateBuiltinAgentCharacter(data: Api.Manage.SaveBuiltinAgentCharacterRequest) {
  return request<Api.Manage.AgentCharacter>({
    url: '/api/v1/manage/agent/characters/builtin',
    method: 'post',
    data
  });
}

export function UpdateBuiltinAgentCharacter(id: string, data: Api.Manage.SaveBuiltinAgentCharacterRequest) {
  return request<Api.Manage.AgentCharacter>({
    url: `/api/v1/manage/agent/characters/builtin/${encodeURIComponent(id)}`,
    method: 'patch',
    data
  });
}

export function DeleteBuiltinAgentCharacter(id: string) {
  return request<Record<string, never>>({
    url: `/api/v1/manage/agent/characters/builtin/${encodeURIComponent(id)}`,
    method: 'delete'
  });
}
