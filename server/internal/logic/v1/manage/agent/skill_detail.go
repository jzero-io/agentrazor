package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
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
	detail, err := l.svcCtx.AgentService.SkillDetail(l.ctx, req.SkillName, req.File)
	if err != nil {
		return nil, err
	}
	return &types.SkillDetailResponse{
		Skill:       types.Skill{Name: detail.Skill.Name},
		Files:       manageSkillFiles(detail.Files),
		CurrentFile: detail.CurrentFile,
		Content:     detail.Content,
	}, nil
}

func manageSkillFiles(values []agentdomain.SkillFile) []types.SkillFile {
	result := make([]types.SkillFile, 0, len(values))
	for _, value := range values {
		result = append(result, types.SkillFile{
			Name: value.Name, Path: value.Path, Type: value.Type, Children: manageSkillFiles(value.Children),
		})
	}
	return result
}
