package character

import (
	"context"
	"net/http"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/character"
)

type Update struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewUpdate(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Update {
	return &Update{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *Update) Update(req *types.UpdateRequest) (*types.Character, error) {
	row, err := findBuiltinCharacter(l.ctx, l.svcCtx, req.CharacterId)
	if err != nil {
		return nil, err
	}
	name, description, prompt, err := normalizeCharacterInput(req.Name, req.Description, req.Prompt)
	if err != nil {
		return nil, err
	}
	if err := ensureBuiltinNameUnique(l.ctx, l.svcCtx, name, row.Uuid); err != nil {
		return nil, err
	}
	if err := l.svcCtx.Model.AgentCharacter.UpdateFieldsByCondition(l.ctx, nil, map[string]any{
		string(agentcharactermodel.Name): name, string(agentcharactermodel.Description): description,
		string(agentcharactermodel.Prompt): prompt, string(agentcharactermodel.Sort): req.Sort,
	}, condition.NewChain().Equal(agentcharactermodel.Uuid, row.Uuid).Equal(agentcharactermodel.IsBuiltin, true).Build()...); err != nil {
		return nil, normalizeWriteError(err)
	}
	row, err = l.svcCtx.Model.AgentCharacter.FindOne(l.ctx, nil, row.Uuid)
	if err != nil {
		return nil, err
	}
	result := toCharacter(row, nil)
	return &result, nil
}
