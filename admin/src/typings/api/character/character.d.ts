declare namespace Api {
  namespace Manage {
    type AgentCharacterOwner = {
      userUuid: string;
      username: string;
      nickName: string;
      userEmail: string;
    };

    type AgentCharacter = {
      id: string;
      name: string;
      description: string;
      prompt: string;
      builtin: boolean;
      sort: number;
      owner?: AgentCharacterOwner;
      createdAt: string;
      updatedAt: string;
    };

    type AgentCharacterKind = 'builtin' | 'custom';

    type AgentCharacterSearchParams = Pick<Common.PaginatingCommonParams, 'current' | 'size'> & {
      kind: AgentCharacterKind;
      keyword?: string;
    };

    type AgentCharacterList = Common.PaginatingQueryRecord<AgentCharacter>;

    type SaveBuiltinAgentCharacterRequest = {
      name: string;
      description: string;
      prompt: string;
      sort: number;
    };
  }
}
