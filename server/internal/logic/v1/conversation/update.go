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

type Update struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 修改会话
func NewUpdate(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Update {
	return &Update{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Update) Update(req *types.UpdateRequest) (resp *types.Conversation, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	uuid, err := requireOwner(l.ctx, l.svcCtx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	if req.Title != nil {
		if err := l.svcCtx.AgentThreads.SetName(l.ctx, req.ConversationId, *req.Title); err != nil {
			return nil, err
		}
	}
	if req.Pinned != nil {
		if err := l.svcCtx.AgentThreads.SetPinned(l.ctx, req.ConversationId, *req.Pinned); err != nil {
			return nil, err
		}
	}
	if req.Archived != nil {
		if err := l.svcCtx.AgentThreads.SetArchived(l.ctx, req.ConversationId, *req.Archived); err != nil {
			return nil, err
		}
	}
	if req.GroupId != nil {
		if *req.GroupId != "" {
			owner, exists, err := groupUser(l.ctx, l.svcCtx, *req.GroupId)
			if err != nil {
				return nil, err
			}
			if !exists || owner != uuid {
				return nil, errors.New("conversation group not found")
			}
		}
		groupUUID := any(nil)
		if *req.GroupId != "" {
			groupUUID = *req.GroupId
		}
		if err := l.svcCtx.Model.Conversation.UpdateFieldsByCondition(l.ctx, nil,
			map[string]any{string(conversationmodel.GroupUuid): groupUUID},
			condition.NewChain().Equal(conversationmodel.Id, req.ConversationId).Equal(conversationmodel.UserUuid, uuid).Build()...); err != nil {
			return nil, err
		}
	}
	updated, err := l.svcCtx.AgentThreads.Get(l.ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	if req.Archived != nil {
		updated.Archived = *req.Archived
	}
	response := toConversation(updated)
	assignments, err := groupAssignments(l.ctx, l.svcCtx, uuid)
	if err != nil {
		return nil, err
	}
	if groupID := assignments[req.ConversationId]; groupID != "" {
		response.GroupId = &groupID
	}
	return &response, nil
}
