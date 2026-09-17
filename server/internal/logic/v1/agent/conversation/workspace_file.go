package conversation

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/agent/conversation"
)

type WorkspaceFile struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
	w      http.ResponseWriter
}

// 读取会话工作区文件
func NewWorkspaceFile(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request, w http.ResponseWriter) *WorkspaceFile {
	return &WorkspaceFile{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
		w:      w,
	}
}

func (l *WorkspaceFile) WorkspaceFile(req *types.WorkspaceFileRequest) error {
	if _, err := requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return err
	}
	workspaceFile, err := l.svcCtx.AgentService.ReadWorkspaceFile(l.ctx, req.ConversationId, req.FilePath)
	if err != nil {
		return err
	}
	contentType := strings.TrimSpace(workspaceFile.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	l.w.Header().Set("Content-Type", contentType)
	l.w.Header().Set("Content-Length", fmt.Sprintf("%d", len(workspaceFile.Data)))
	l.w.Header().Set("Content-Disposition", "inline; filename*=UTF-8\x27\x27"+url.PathEscape(filepath.Base(req.FilePath)))
	l.w.Header().Set("Cache-Control", "private, no-store")
	_, err = l.w.Write(workspaceFile.Data)
	return err
}
