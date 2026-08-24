package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type RuntimeStatus struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewRuntimeStatus(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *RuntimeStatus {
	return &RuntimeStatus{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *RuntimeStatus) RuntimeStatus(req *types.RuntimeStatusRequest) (resp *types.RuntimeStatus, err error) {
	if l.svcCtx.AgentThreads == nil {
		return &types.RuntimeStatus{}, nil
	}
	status := runtimeStatus(l.svcCtx.AgentThreads.RuntimeStatus())
	return &status, nil
}
