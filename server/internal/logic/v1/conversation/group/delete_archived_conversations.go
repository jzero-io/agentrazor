package group

import (
	"context"
	"errors"
	"net/http"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation/group"
)

type DeleteArchivedConversations struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewDeleteArchivedConversations(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *DeleteArchivedConversations {
	return &DeleteArchivedConversations{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

// 删除分组内所有已归档对话（批量，由后端统一处理）
func (l *DeleteArchivedConversations) DeleteArchivedConversations(req *types.PathRequest) (resp *types.DeleteArchivedConversationsResponse, err error) {
	user, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.Model.ConversationGroup.FindOne(l.ctx, nil, req.GroupId)
	if err != nil || row.UserUuid != user.Uuid {
		return nil, errors.New("conversation group not found")
	}
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	convs, err := l.svcCtx.Model.Conversation.FindByCondition(l.ctx, nil,
		condition.NewChain().
			Equal(conversationmodel.UserUuid, user.Uuid).
			Equal(conversationmodel.GroupUuid, req.GroupId).
			Build()...,
	)
	if err != nil {
		return nil, err
	}
	for _, conv := range convs {
		thread, err := l.svcCtx.AgentThreads.Get(l.ctx, conv.Id)
		if err != nil {
			// 线程已不存在（孤儿数据）：跳过 thread 侧，直接清理业务库记录
			if errors.Is(err, agentdomain.ErrThreadNotFound) {
				if dbErr := l.svcCtx.Model.Conversation.DeleteByCondition(l.ctx, nil,
					condition.NewChain().
						Equal(conversationmodel.Id, conv.Id).
						Equal(conversationmodel.UserUuid, user.Uuid).
						Build()...,
				); dbErr != nil {
					return nil, dbErr
				}
				continue
			}
			return nil, err
		}
		// 只删除已归档的对话，组内未归档的跳过，不影响本次批量删除
		if !thread.Archived {
			continue
		}
		if err := l.svcCtx.AgentThreads.Delete(l.ctx, conv.Id); err != nil {
			return nil, err
		}
		if err := l.svcCtx.Model.Conversation.DeleteByCondition(l.ctx, nil,
			condition.NewChain().
				Equal(conversationmodel.Id, conv.Id).
				Equal(conversationmodel.UserUuid, user.Uuid).
				Build()...,
		); err != nil {
			return nil, err
		}
	}
	return &types.DeleteArchivedConversationsResponse{}, nil
}
