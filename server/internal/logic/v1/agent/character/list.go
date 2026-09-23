package character

import (
	"context"
	"net/http"
	"sort"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/agent/character"
)

type List struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewList(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *List {
	return &List{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *List) List() (*types.ListResponse, error) {
	userUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	builtins, err := l.svcCtx.Model.AgentCharacter.FindByCondition(l.ctx, nil, condition.NewChain().
		Equal(agentcharactermodel.IsBuiltin, true).
		Build()...)
	if err != nil {
		return nil, err
	}
	custom, err := l.svcCtx.Model.AgentCharacter.FindByCondition(l.ctx, nil, condition.NewChain().
		Equal(agentcharactermodel.IsBuiltin, false).
		Equal(agentcharactermodel.UserUuid, userUUID).
		Build()...)
	if err != nil {
		return nil, err
	}
	rows := append(builtins, custom...)
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].IsBuiltin != rows[j].IsBuiltin {
			return rows[i].IsBuiltin
		}
		if rows[i].Sort != rows[j].Sort {
			return rows[i].Sort < rows[j].Sort
		}
		return rows[i].CreateTime.Before(rows[j].CreateTime)
	})
	result := &types.ListResponse{Characters: make([]types.Character, 0, len(rows))}
	for _, row := range rows {
		result.Characters = append(result.Characters, toCharacter(row))
	}
	return result, nil
}
