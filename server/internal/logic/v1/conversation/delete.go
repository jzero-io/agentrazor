package conversation

import (
	"context"
	"errors"
	"net/http"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type Delete struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 删除会话
func NewDelete(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Delete {
	return &Delete{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Delete) Delete(req *types.PathRequest) (resp *types.DeleteResponse, err error) {
	uuid, err := requireOwner(l.ctx, l.svcCtx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	if err := l.svcCtx.AgentThreads.Delete(l.ctx, req.ConversationId); err != nil {
		return nil, err
	}
	if err := l.svcCtx.Model.Conversation.DeleteByCondition(l.ctx, nil,
		condition.NewChain().
			Equal(conversationmodel.Id, req.ConversationId).
			Equal(conversationmodel.UserUuid, uuid).
			Build()...,
	); err != nil {
		return nil, err
	}
	return nil, nil
}
