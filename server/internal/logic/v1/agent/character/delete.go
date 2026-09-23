package character

import (
	"context"
	"net/http"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/agent/character"
)

type Delete struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewDelete(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Delete {
	return &Delete{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *Delete) Delete(req *types.PathRequest) (*types.DeleteResponse, error) {
	userUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	row, err := findOwnedCustomCharacter(l.ctx, l.svcCtx, userUUID, req.CharacterId)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Model.AgentCharacter.DeleteByCondition(l.ctx, nil, condition.NewChain().
		Equal(agentcharactermodel.Uuid, row.Uuid).
		Equal(agentcharactermodel.UserUuid, userUUID).
		Build()...); err != nil {
		return nil, err
	}
	return &types.DeleteResponse{}, nil
}
