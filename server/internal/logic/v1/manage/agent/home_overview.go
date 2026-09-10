package agent

import (
	"context"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type HomeOverview struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewHomeOverview(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *HomeOverview {
	return &HomeOverview{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *HomeOverview) HomeOverview(req *types.HomeOverviewRequest) (resp *types.HomeOverviewResponse, err error) {
	settings, err := l.svcCtx.AgentService.Settings(l.ctx)
	if err != nil {
		return nil, err
	}
	providers, err := settings.Providers()
	if err != nil {
		return nil, err
	}
	skills, err := l.svcCtx.AgentService.ListSkills(l.ctx)
	if err != nil {
		return nil, err
	}
	summary, err := l.svcCtx.Model.ConversationTokenUsageEvent.TokenUsageSummary(l.ctx, "")
	if err != nil {
		return nil, err
	}

	providerID := settings.ActiveProvider()
	modelID := settings.Model()
	modelName := modelID
	for _, provider := range providers {
		if provider.Id != providerID {
			continue
		}
		for _, model := range provider.Models {
			if model.Id == modelID && strings.TrimSpace(model.Name) != "" {
				modelName = model.Name
				break
			}
		}
		break
	}

	return &types.HomeOverviewResponse{
		AgentRunning:   l.svcCtx.AgentService.Running(),
		ActiveProvider: providerID,
		Model:          modelID,
		ModelName:      modelName,
		TotalTokens:    summary.TotalTokens,
		SkillCount:     int64(len(skills)),
	}, nil
}
