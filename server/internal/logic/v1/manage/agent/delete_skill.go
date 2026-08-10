package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type DeleteSkill struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewDeleteSkill(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *DeleteSkill {
	return &DeleteSkill{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *DeleteSkill) DeleteSkill(req *types.DeleteSkillRequest) (resp *types.DeleteSkillResponse, err error) {
	config := l.svcCtx.MustGetConfig()
	if err := deleteSkill(config.Agent.CodexHome, req.SkillName); err != nil {
		return nil, err
	}
	return &types.DeleteSkillResponse{}, nil
}
