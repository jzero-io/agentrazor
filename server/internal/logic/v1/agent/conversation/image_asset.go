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

type ImageAsset struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
	w      http.ResponseWriter
}

// 读取会话生成的图片资产
func NewImageAsset(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request, w http.ResponseWriter) *ImageAsset {
	return &ImageAsset{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
		w:      w,
	}
}

func (l *ImageAsset) ImageAsset(req *types.ImageAssetRequest) error {
	if _, err := requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return err
	}
	image, err := l.svcCtx.AgentService.ReadGeneratedImageAsset(l.ctx, req.ConversationId, req.Name)
	if err != nil {
		return err
	}
	contentType := strings.TrimSpace(image.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	l.w.Header().Set("Content-Type", contentType)
	l.w.Header().Set("Content-Length", fmt.Sprintf("%d", len(image.Data)))
	l.w.Header().Set("Content-Disposition", "inline; filename*=UTF-8\x27\x27"+url.PathEscape(filepath.Base(req.Name)))
	l.w.Header().Set("Cache-Control", "private, no-store")
	_, err = l.w.Write(image.Data)
	return err
}
