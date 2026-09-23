package character

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/character"
)

type Create struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewCreate(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Create {
	return &Create{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *Create) Create(req *types.CreateRequest) (*types.Character, error) {
	name, description, prompt, err := normalizeCharacterInput(req.Name, req.Description, req.Prompt)
	if err != nil {
		return nil, err
	}
	if err := ensureBuiltinNameUnique(l.ctx, l.svcCtx, name, ""); err != nil {
		return nil, err
	}
	row := &agentcharactermodel.AgentCharacter{
		Uuid: uuid.NewString(), UserUuid: builtinOwner(), Name: name, Description: description,
		Prompt: prompt, IsBuiltin: true, Sort: req.Sort,
	}
	if err := l.svcCtx.Model.AgentCharacter.InsertV2(l.ctx, nil, row); err != nil {
		return nil, normalizeWriteError(err)
	}
	row, err = l.svcCtx.Model.AgentCharacter.FindOne(l.ctx, nil, row.Uuid)
	if err != nil {
		return nil, err
	}
	result := toCharacter(row, nil)
	return &result, nil
}
