package conversation

import (
	"context"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type Metadata struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取会话元信息
func NewMetadata(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Metadata {
	return &Metadata{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Metadata) Metadata(req *types.PathRequest) (resp *types.Conversation, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	uuid, err := requireOwner(l.ctx, l.svcCtx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	thread, err := l.svcCtx.AgentThreads.Metadata(l.ctx, req.ConversationId)
	if err != nil {
		if errors.Is(err, agentdomain.ErrThreadNotFound) {
			return nil, errConversationNotOwned
		}
		return nil, err
	}
	conversation := toConversation(thread)
	setConversationActiveRun(&conversation, l.svcCtx.AgentThreads, req.ConversationId)
	assignments, err := groupAssignments(l.ctx, l.svcCtx, uuid)
	if err != nil {
		return nil, err
	}
	if groupID := assignments[req.ConversationId]; groupID != "" {
		conversation.GroupId = &groupID
	}
	return &conversation, nil
}
