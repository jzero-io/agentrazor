package conversation

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type UploadAttachment struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 上传消息附件到会话工作区
func NewUploadAttachment(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UploadAttachment {
	return &UploadAttachment{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *UploadAttachment) UploadAttachment(req *types.UploadAttachmentRequest) (resp *types.UploadAttachmentResponse, err error) {
	if _, err := requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return nil, err
	}
	thread, err := l.svcCtx.AgentService.Metadata(l.ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	if thread.Archived {
		return nil, errors.New("cannot upload attachments to an archived conversation")
	}

	l.r.Body = http.MaxBytesReader(nil, l.r.Body, agentdomain.MaxAttachmentSize+(1<<20))
	if err := l.r.ParseMultipartForm(agentdomain.MaxAttachmentSize); err != nil {
		return nil, errors.New("invalid attachment upload")
	}
	if l.r.MultipartForm != nil {
		defer l.r.MultipartForm.RemoveAll()
	}
	file, header, err := l.r.FormFile("file")
	if err != nil || header == nil {
		return nil, errors.New("attachment file is required")
	}
	defer file.Close()
	if header.Size == 0 {
		return nil, errors.New("attachment is empty")
	}
	if header.Size > agentdomain.MaxAttachmentSize {
		return nil, errors.New("attachment is too large")
	}
	data, err := io.ReadAll(io.LimitReader(file, agentdomain.MaxAttachmentSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > agentdomain.MaxAttachmentSize {
		return nil, errors.New("attachment is too large")
	}
	attachment, err := l.svcCtx.AgentService.SaveWorkspaceAttachment(l.ctx, req.ConversationId, header.Filename, data)
	if err != nil {
		return nil, err
	}
	return &types.UploadAttachmentResponse{Attachment: types.MessageAttachment{
		Name: attachment.Name, Path: attachment.Path, ContentType: attachment.ContentType,
		Size: attachment.Size, Kind: attachment.Kind,
	}}, nil
}
