package agent

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type UploadSkill struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewUploadSkill(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UploadSkill {
	return &UploadSkill{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *UploadSkill) UploadSkill(req *types.UploadSkillRequest) (resp *types.UploadSkillResponse, err error) {
	if err := l.r.ParseMultipartForm(64 << 20); err != nil {
		return nil, err
	}
	file, header, err := l.r.FormFile("file")
	if err != nil {
		return nil, errors.New("skill zip file is required")
	}
	defer file.Close()
	name := strings.TrimSpace(l.r.FormValue("name"))
	if name == "" && header != nil {
		name = header.Filename
	}
	config := l.svcCtx.MustGetConfig()
	installed, err := installSkillZip(config.Agent.CodexHome, name, file, header.Size)
	if err != nil {
		return nil, err
	}
	return &installed, nil
}
