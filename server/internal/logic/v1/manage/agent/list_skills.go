package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type ListSkills struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewListSkills(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *ListSkills {
	return &ListSkills{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *ListSkills) ListSkills(req *types.ListSkillsRequest) (resp *types.ListSkillsResponse, err error) {
	config := l.svcCtx.MustGetConfig()
	skills, err := listSkills(config.Agent.CodexHome)
	if err != nil {
		return nil, err
	}
	return &types.ListSkillsResponse{Skills: skills}, nil
}
