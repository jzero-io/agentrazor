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
	convs, err := l.svcCtx.Model.Conversation.FindByCondition(l.ctx, nil,
		condition.NewChain().
			Equal(conversationmodel.UserUuid, user.Uuid).
			Equal(conversationmodel.GroupUuid, req.GroupId).
			Build()...,
	)
	if err != nil {
		return nil, err
	}
	// thread/read (used by Metadata) does not reliably include the archived
	// flag. Build the archive set from thread/list, which is the authoritative
	// source for archive state.
	threads, err := l.svcCtx.AgentService.List(l.ctx)
	if err != nil {
		return nil, err
	}
	archived := archivedThreadIDs(threads)
	for _, conv := range convs {
		_, err := l.svcCtx.AgentService.Metadata(l.ctx, conv.Id)
		if err != nil {
			// 线程已不存在（孤儿数据）：跳过 thread 侧，直接清理业务库记录
			if errors.Is(err, agentdomain.ErrThreadNotFound) {
				if dbErr := l.svcCtx.AgentService.DeleteConversationHome(conv.Id); dbErr != nil {
					return nil, dbErr
				}
				if dbErr := l.svcCtx.Model.Conversation.DeleteByCondition(l.ctx, nil,
					condition.NewChain().
						Equal(conversationmodel.Id, conv.Id).
						Equal(conversationmodel.UserUuid, user.Uuid).
						Build()...,
				); dbErr != nil {
					return nil, dbErr
				}
				// Token usage is lifetime accounting data and must survive conversation deletion.
				continue
			}
			return nil, err
		}
		// 只删除已归档的对话，组内未归档的跳过，不影响本次批量删除
		if _, ok := archived[conv.Id]; !ok {
			continue
		}
		if err := l.svcCtx.AgentService.Delete(l.ctx, conv.Id); err != nil {
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
		// Keep token usage events so the historical total remains stable.
	}
	return &types.DeleteArchivedConversationsResponse{}, nil
}

func archivedThreadIDs(threads []agentdomain.StoredThread) map[string]struct{} {
	result := make(map[string]struct{})
	for _, thread := range threads {
		if thread.Archived {
			result[thread.ID] = struct{}{}
		}
	}
	return result
}
