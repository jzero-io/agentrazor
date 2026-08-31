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
	if strings.TrimSpace(thread.Name) == "" {
		if err := svcCtx.AgentThreads.SetName(ctx, conversationID, content); err != nil {
			return nil, err
		}
		thread.Name = content
	}
	run, err := svcCtx.AgentThreads.Send(conversationID, content)
	if err != nil {
		return nil, err
	}
	return &types.SendMessageResponse{
		ConversationId: conversationID,
		Conversation: func() types.Conversation {
			item := toConversation(thread)
			item.Running = true
			startedAt := formatTime(run.CreatedAt)
			item.RunningStartedAt = &startedAt
			return item
		}(),
		Run: toRun(run),
	}, nil
}
