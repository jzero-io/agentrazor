package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type ConfigFile struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewConfigFile(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *ConfigFile {
	return &ConfigFile{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *ConfigFile) ConfigFile(req *types.ConfigFileRequest) (resp *types.ConfigFileResponse, err error) {
	config := l.svcCtx.MustGetConfig()
	file, err := readAgentConfigFile(config.Agent.CodexHome, req.Name)
	if err != nil {
		return nil, err
	}
	return &file, nil
}
