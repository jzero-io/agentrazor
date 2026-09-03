package apikey

import (
	"context"
	"net/http"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	agentapikeymodel "github.com/jzero-io/agentrazor/server/internal/model/agent_api_key"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth/apikey"
)

type Delete struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 删除 Agent API 密钥
func NewDelete(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Delete {
	return &Delete{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Delete) Delete(req *types.DeleteRequest) (resp *types.DeleteResponse, err error) {
	user, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}
	conditions := condition.NewChain().
		Equal(agentapikeymodel.Uuid, req.Id).
		Equal(agentapikeymodel.UserUuid, user.Uuid).
		Build()
	if _, err := l.svcCtx.Model.AgentApiKey.FindOneByCondition(l.ctx, nil, conditions...); err != nil {
		if errors.Is(err, agentapikeymodel.ErrNotFound) {
			return nil, errors.New("密钥不存在")
		}
		return nil, err
	}
	if err := l.svcCtx.Model.AgentApiKey.DeleteByCondition(l.ctx, nil, conditions...); err != nil {
		return nil, err
	}
	return &types.DeleteResponse{}, nil
}
