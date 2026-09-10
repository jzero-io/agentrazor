package conversation

import (
	"context"
	"encoding/base64"
	"net/http"
	"path"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type WorkspaceFile struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 读取会话工作区文件
func NewWorkspaceFile(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *WorkspaceFile {
	return &WorkspaceFile{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *WorkspaceFile) WorkspaceFile(req *types.WorkspaceFileRequest) (resp *types.WorkspaceFileResponse, err error) {
	if _, err = requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return nil, err
	}
	workspaceFile, err := l.svcCtx.AgentService.ReadWorkspaceFile(l.ctx, req.ConversationId, req.FilePath)
	if err != nil {
		return nil, err
	}
	contentType := strings.TrimSpace(workspaceFile.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &types.WorkspaceFileResponse{
		Name:        path.Base(strings.ReplaceAll(req.FilePath, "\\", "/")),
		ContentType: contentType,
		DataBase64:  base64.StdEncoding.EncodeToString(workspaceFile.Data),
	}, nil
}
