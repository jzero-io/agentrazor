package conversation

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type WorkspaceFiles struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取会话工作区文件
func NewWorkspaceFiles(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *WorkspaceFiles {
	return &WorkspaceFiles{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *WorkspaceFiles) WorkspaceFiles(req *types.PathRequest) (resp *types.WorkspaceFilesResponse, err error) {
	if _, err = requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return nil, err
	}
	workspaceEntries, err := l.svcCtx.AgentService.ListWorkspaceFiles(l.ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	return &types.WorkspaceFilesResponse{Files: workspaceFiles(workspaceEntries)}, nil
}

func workspaceFiles(values []agentdomain.WorkspaceEntry) []types.WorkspaceEntry {
	result := make([]types.WorkspaceEntry, 0, len(values))
	for _, value := range values {
		result = append(result, types.WorkspaceEntry{
			Name: value.Name, Path: value.Path, Type: value.Type, Children: workspaceFiles(value.Children),
		})
	}
	return result
}
