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

type ArchiveConversations struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewArchiveConversations(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *ArchiveConversations {
	return &ArchiveConversations{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

// 归档分组内所有对话（批量，由后端统一处理）
func (l *ArchiveConversations) ArchiveConversations(req *types.PathRequest) (resp *types.ArchiveConversationsResponse, err error) {
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
		// 线程已不存在（孤儿数据）：无需归档，跳过
		if _, err := l.svcCtx.AgentThreads.Get(l.ctx, conv.Id); err != nil {
			if errors.Is(err, agentdomain.ErrThreadNotFound) {
				continue
			}
			return nil, err
		}
		if err := l.svcCtx.AgentThreads.SetArchived(l.ctx, conv.Id, true); err != nil {
			return nil, err
		}
	}
	return &types.ArchiveConversationsResponse{}, nil
}
