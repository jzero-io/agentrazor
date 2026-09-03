package conversation

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	conversationgroupmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_group"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type List struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取会话列表
func NewList(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *List {
	return &List{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *List) List() (resp *types.ListResponse, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	uuid, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	owned, err := conversationsByUser(l.ctx, l.svcCtx, uuid)
	if err != nil {
		return nil, err
	}
	assignments, err := groupAssignments(l.ctx, l.svcCtx, uuid)
	if err != nil {
		return nil, err
	}

	threads, err := l.listOwnedThreads(owned)
	if err != nil {
		return nil, err
	}
	order := conversationOrder(owned)
	sort.SliceStable(threads, func(i, j int) bool {
		if threads[i].IsPinned != threads[j].IsPinned {
			return threads[i].IsPinned
		}
		if !threads[i].UpdatedAt.Equal(threads[j].UpdatedAt) {
			return threads[i].UpdatedAt.After(threads[j].UpdatedAt)
		}
		left, right := order[threads[i].ID], order[threads[j].ID]
		if left != nil && right != nil && !left.CreateTime.Equal(right.CreateTime) {
			return left.CreateTime.After(right.CreateTime)
		}
		return threads[i].ID > threads[j].ID
	})

	response := &types.ListResponse{Conversations: make([]types.Conversation, 0, len(threads))}
	for _, thread := range threads {
		mapped := toConversation(thread)
		setConversationActiveTurn(&mapped, l.svcCtx.AgentThreads, thread.ID)
		if groupID := assignments[thread.ID]; groupID != "" {
			mapped.GroupId = &groupID
		}
		response.Conversations = append(response.Conversations, mapped)
	}
	return response, nil
}

func (l *List) listOwnedThreads(owned []*conversationmodel.Conversation) ([]agentdomain.StoredThread, error) {
	listed, err := l.svcCtx.AgentThreads.List(l.ctx)
	if err != nil {
		return nil, err
	}
	ownedSet := make(map[string]struct{}, len(owned))
	for _, row := range owned {
		ownedSet[row.Id] = struct{}{}
	}
	byID := make(map[string]agentdomain.StoredThread, len(listed))
	for _, thread := range listed {
		if _, ok := ownedSet[thread.ID]; ok {
			byID[thread.ID] = thread
		}
	}
	threads := make([]agentdomain.StoredThread, 0, len(owned))
	for _, row := range owned {
		id := row.Id
		thread, ok := byID[id]
		if !ok {
			var err error
			thread, err = l.svcCtx.AgentThreads.Metadata(l.ctx, id)
			if errors.Is(err, agentdomain.ErrThreadNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
		}
		threads = append(threads, thread)
	}
	return threads, nil
}

func conversationUser(ctx context.Context, svcCtx *svc.ServiceContext, conversationID string) (string, bool, error) {
	row, err := svcCtx.Model.Conversation.FindOne(ctx, nil, conversationID)
	if errors.Is(err, conversationmodel.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.UserUuid, true, nil
}

func conversationsByUser(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string) ([]*conversationmodel.Conversation, error) {
	return svcCtx.Model.Conversation.FindFieldsByCondition(ctx, nil,
		[]condition.Field{conversationmodel.Id, conversationmodel.CreateTime, conversationmodel.UpdateTime},
		condition.NewChain().Equal(conversationmodel.UserUuid, userUUID).Build()...)
}

func conversationOrder(rows []*conversationmodel.Conversation) map[string]*conversationmodel.Conversation {
	result := make(map[string]*conversationmodel.Conversation, len(rows))
	for _, row := range rows {
		result[row.Id] = row
	}
	return result
}

func groupAssignments(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string) (map[string]string, error) {
	rows, err := svcCtx.Model.Conversation.FindFieldsByCondition(ctx, nil,
		[]condition.Field{conversationmodel.Id, conversationmodel.GroupUuid},
		condition.NewChain().Equal(conversationmodel.UserUuid, userUUID).Build()...)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, row := range rows {
		if row.GroupUuid.Valid {
			result[row.Id] = row.GroupUuid.String
		}
	}
	return result, nil
}

func groupsByUser(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string) ([]*conversationgroupmodel.ConversationGroup, error) {
	rows, err := svcCtx.Model.ConversationGroup.FindByCondition(ctx, nil,
		condition.NewChain().Equal(conversationgroupmodel.UserUuid, userUUID).Build()...)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].CreateTime.Before(rows[j].CreateTime)
	})
	return rows, nil
}

func groupUser(ctx context.Context, svcCtx *svc.ServiceContext, groupUUID string) (string, bool, error) {
	row, err := svcCtx.Model.ConversationGroup.FindOne(ctx, nil, groupUUID)
	if errors.Is(err, conversationgroupmodel.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.UserUuid, true, nil
}
