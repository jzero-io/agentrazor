package conversation

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

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
	runtimeEntries, err := l.svcCtx.Codex.ListWorkspaceFiles(l.ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	entries := make([]types.WorkspaceEntry, 0, len(runtimeEntries))
	for _, entry := range runtimeEntries {
		entries = append(entries, types.WorkspaceEntry{
			Name: entry.Name,
			Path: entry.Path,
			Type: entry.Type,
			Size: entry.Size,
		})
	}
	return &types.WorkspaceFilesResponse{Entries: entries}, nil
}
