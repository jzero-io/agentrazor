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
	skills, err := l.svcCtx.AgentService.ListSkills(l.ctx)
	if err != nil {
		return nil, err
	}
	result := make([]types.Skill, 0, len(skills))
	for _, skill := range skills {
		result = append(result, types.Skill{Name: skill.Name})
	}
	return &types.ListSkillsResponse{Skills: result}, nil
}
