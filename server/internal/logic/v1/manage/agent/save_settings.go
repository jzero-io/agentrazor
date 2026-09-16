package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type SaveSettings struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSaveSettings(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SaveSettings {
	return &SaveSettings{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SaveSettings) SaveSettings(req *types.SaveSettingsRequest) (resp *types.SaveSettingsResponse, err error) {
	if err := l.svcCtx.AgentService.SaveSettings(l.ctx, req.ProviderId, req.Model, req.ReasoningEffort, req.ProviderApiKey, req.DefaultSystemPrompt); err != nil {
		return nil, err
	}
	return &types.SaveSettingsResponse{}, nil
}
