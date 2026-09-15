package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type SetSkillStatus struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSetSkillStatus(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SetSkillStatus {
	return &SetSkillStatus{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SetSkillStatus) SetSkillStatus(req *types.SetSkillStatusRequest) (resp *types.SetSkillStatusResponse, err error) {
	if err := l.svcCtx.AgentService.SetSkillStatus(l.ctx, req.SkillName, req.Enabled); err != nil {
		return nil, err
	}
	return &types.SetSkillStatusResponse{}, nil
}
