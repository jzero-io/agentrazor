package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type ListConfigFiles struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewListConfigFiles(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *ListConfigFiles {
	return &ListConfigFiles{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *ListConfigFiles) ListConfigFiles(req *types.ListConfigFilesRequest) (resp *types.ListConfigFilesResponse, err error) {
	return &types.ListConfigFilesResponse{Files: listAgentConfigFiles()}, nil
}
