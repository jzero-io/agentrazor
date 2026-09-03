package apikey

import (
	"context"
	"net/http"
	"sort"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentapikeymodel "github.com/jzero-io/agentrazor/server/internal/model/agent_api_key"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth/apikey"
)

type List struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 列出当前账户的 Agent API 密钥
func NewList(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *List {
	return &List{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *List) List() (resp *types.ListResponse, err error) {
	user, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}
	rows, err := l.svcCtx.Model.AgentApiKey.FindByCondition(l.ctx, nil, condition.NewChain().
		Equal(agentapikeymodel.UserUuid, user.Uuid).
		Build()...)
	if err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreateTime.After(rows[j].CreateTime) })

	keys := make([]types.ApiKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, toAPIKey(row))
	}
	return &types.ListResponse{Keys: keys}, nil
}
