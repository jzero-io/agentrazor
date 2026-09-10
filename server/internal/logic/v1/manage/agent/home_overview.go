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
	store, err := readAgentSettingsStore(l.ctx, l.svcCtx.Codex)
	if err != nil {
		return nil, err
	}
	providers, err := store.providers()
	if err != nil {
		return nil, err
	}
	skills, err := l.svcCtx.Codex.ListSkills(l.ctx)
	if err != nil {
		return nil, err
	}
	summary, err := l.svcCtx.Model.ConversationTokenUsageEvent.TokenUsageSummary(l.ctx, "")
	if err != nil {
		return nil, err
	}

	providerID := store.activeProvider()
	modelID := store.model()
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

	running := false
	if l.svcCtx.AgentThreads != nil {
		running = l.svcCtx.AgentThreads.RuntimeStatus().Running
	}

	return &types.HomeOverviewResponse{
		AgentRunning:   running,
		ActiveProvider: providerID,
		Model:          modelID,
		ModelName:      modelName,
		TotalTokens:    summary.TotalTokens,
		SkillCount:     int64(len(skills)),
	}, nil
}
