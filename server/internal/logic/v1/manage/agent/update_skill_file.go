package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type UpdateSkillFile struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewUpdateSkillFile(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UpdateSkillFile {
	return &UpdateSkillFile{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *UpdateSkillFile) UpdateSkillFile(req *types.UpdateSkillFileRequest) (resp *types.UpdateSkillFileResponse, err error) {
	config := l.svcCtx.MustGetConfig()
	if err := updateSkillFile(config.Agent.CodexHome, req.SkillName, req.File, req.Content); err != nil {
		return nil, err
	}
	return &types.UpdateSkillFileResponse{}, nil
}
