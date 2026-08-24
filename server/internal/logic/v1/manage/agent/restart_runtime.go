package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type RestartRuntime struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewRestartRuntime(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *RestartRuntime {
	return &RestartRuntime{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *RestartRuntime) RestartRuntime(req *types.RestartRuntimeRequest) (resp *types.RuntimeStatus, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, agentdomain.ErrServiceStopped
	}
	if err := l.svcCtx.AgentThreads.RestartRuntime(); err != nil {
		return nil, err
	}
	status := runtimeStatus(l.svcCtx.AgentThreads.RuntimeStatus())
	return &status, nil
}
