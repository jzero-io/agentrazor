package conversation

import (
	"context"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type SendMessage struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 发送消息
func NewSendMessage(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SendMessage {
	return &SendMessage{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SendMessage) SendMessage(req *types.SendMessageRequest) (resp *types.SendMessageResponse, err error) {
	return sendMessage(l.ctx, l.svcCtx, req.ConversationId, req.Content)
}

func sendMessage(ctx context.Context, svcCtx *svc.ServiceContext, conversationID, content string) (*types.SendMessageResponse, error) {
	if svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("message content is required")
	}
	if _, err := requireOwner(ctx, svcCtx, conversationID); err != nil {
		return nil, err
	}
	thread, err := svcCtx.AgentThreads.Get(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if thread.Archived {
		return nil, agentdomain.ErrThreadArchived
	}
	run, err := svcCtx.AgentThreads.Send(conversationID, content)
	if err != nil {
		return nil, err
	}
	return &types.SendMessageResponse{
		ConversationId: conversationID,
		SessionId:      conversationID,
		Run:            toRun(run),
	}, nil
}
