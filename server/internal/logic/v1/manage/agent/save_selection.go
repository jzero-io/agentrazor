package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

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
	if err := l.svcCtx.AgentService.SaveSelection(l.ctx, req.ProviderId, req.Model, req.ReasoningEffort); err != nil {
		return nil, err
	}
	return &types.SaveSelectionResponse{}, nil
}
