package agent

import (
	"context"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type Logout struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewLogout(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Logout {
	return &Logout{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Logout) Logout(req *types.LogoutRequest) (resp *types.LogoutResponse, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is unavailable")
	}
	if err := l.svcCtx.AgentThreads.Logout(l.ctx); err != nil {
		return nil, err
	}
	return &types.LogoutResponse{}, nil
}
