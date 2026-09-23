package character

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/agent/character"
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
	userUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	row, err := findOwnedCustomCharacter(l.ctx, l.svcCtx, userUUID, req.CharacterId)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]any)
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("角色名称不能为空")
		}
		if err := ensureCharacterNameUnique(l.ctx, l.svcCtx, userUUID, name, row.Uuid); err != nil {
			return nil, err
		}
		fields[string(agentcharactermodel.Name)] = name
	}
	if req.Description != nil {
		fields[string(agentcharactermodel.Description)] = strings.TrimSpace(*req.Description)
	}
	if req.Prompt != nil {
		prompt := strings.TrimSpace(*req.Prompt)
		if prompt == "" {
			return nil, errors.New("角色提示词不能为空")
		}
		fields[string(agentcharactermodel.Prompt)] = prompt
	}
	if len(fields) > 0 {
		err = l.svcCtx.Model.AgentCharacter.UpdateFieldsByCondition(l.ctx, nil, fields, condition.NewChain().
			Equal(agentcharactermodel.Uuid, row.Uuid).
			Equal(agentcharactermodel.UserUuid, userUUID).
			Build()...)
		if err != nil {
			return nil, normalizeCharacterWriteError(err)
		}
	}
	row, err = l.svcCtx.Model.AgentCharacter.FindOne(l.ctx, nil, row.Uuid)
	if err != nil {
		return nil, err
	}
	result := toCharacter(row)
	return &result, nil
}
