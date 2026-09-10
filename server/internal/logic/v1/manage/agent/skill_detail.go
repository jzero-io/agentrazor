package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type SkillDetail struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSkillDetail(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SkillDetail {
	return &SkillDetail{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *SkillDetail) SkillDetail(req *types.SkillDetailRequest) (resp *types.SkillDetailResponse, err error) {
	detail, err := l.svcCtx.Codex.SkillDetail(l.ctx, req.SkillName, req.File)
	if err != nil {
		return nil, err
	}
	return manageSkillDetail(detail), nil
}
