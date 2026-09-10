package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type SaveProviderApiKey struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSaveProviderApiKey(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SaveProviderApiKey {
	return &SaveProviderApiKey{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SaveProviderApiKey) SaveProviderApiKey(req *types.SaveProviderApiKeyRequest) (resp *types.SaveProviderApiKeyResponse, err error) {
	if err := l.svcCtx.AgentService.SaveProviderAPIKey(l.ctx, req.ProviderId, req.ApiKey); err != nil {
		return nil, err
	}
	return &types.SaveProviderApiKeyResponse{
		HasApiKey: true,
	}, nil
}
