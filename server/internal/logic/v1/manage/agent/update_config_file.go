package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type UpdateConfigFile struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewUpdateConfigFile(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UpdateConfigFile {
	return &UpdateConfigFile{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *UpdateConfigFile) UpdateConfigFile(req *types.UpdateConfigFileRequest) (resp *types.UpdateConfigFileResponse, err error) {
	config := l.svcCtx.MustGetConfig()
	if err := writeAgentConfigFile(config.Agent.CodexHome, req.Name, req.Content); err != nil {
		return nil, err
	}
	if l.svcCtx.AgentThreads == nil {
		if req.Restart {
			return nil, agentdomain.ErrServiceStopped
		}
		return &types.UpdateConfigFileResponse{}, nil
	}
	if req.Restart {
		if err := l.svcCtx.AgentThreads.RestartRuntime(); err != nil {
			return nil, err
		}
	}
	return &types.UpdateConfigFileResponse{Runtime: runtimeStatus(l.svcCtx.AgentThreads.RuntimeStatus())}, nil
}
