package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type SaveSelection struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSaveSelection(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SaveSelection {
	return &SaveSelection{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SaveSelection) SaveSelection(req *types.SaveSelectionRequest) (resp *types.SaveSelectionResponse, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, agentdomain.ErrServiceStopped
	}
	if err := updateAgentSettingsStore(l.ctx, l.svcCtx.Codex, func(store *agentSettingsStore) error {
		return store.saveSelection(req.ProviderId, req.Model, req.ReasoningEffort)
	}); err != nil {
		return nil, err
	}
	if err := l.svcCtx.Codex.Restart(l.ctx); err != nil {
		return nil, err
	}
	return &types.SaveSelectionResponse{Runtime: runtimeStatus(l.svcCtx.AgentThreads.RuntimeStatus())}, nil
}
