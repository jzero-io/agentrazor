package conversation

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

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
	owned, err := conversationIDsByUser(l.ctx, l.svcCtx, uuid)
	if err != nil {
		return nil, err
	}
	ownedSet := make(map[string]struct{}, len(owned))
	for _, id := range owned {
		ownedSet[id] = struct{}{}
	}
	conversations, err := l.svcCtx.AgentThreads.List(l.ctx)
	if err != nil {
		return nil, err
	}
	assignments, err := groupAssignments(l.ctx, l.svcCtx, uuid)
	if err != nil {
		return nil, err
	}
	response := &types.ListResponse{
		Conversations: make([]types.Conversation, 0, len(owned)),
	}
	sort.SliceStable(conversations, func(i, j int) bool {
		if conversations[i].IsPinned != conversations[j].IsPinned {
			return conversations[i].IsPinned
		}
		return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
	})
	for _, current := range conversations {
		if _, ok := ownedSet[current.ID]; !ok {
			continue
		}
		mapped := toConversation(current)
		if groupID := assignments[current.ID]; groupID != "" {
			mapped.GroupId = &groupID
		}
		response.Conversations = append(response.Conversations, mapped)
	}
	return response, nil
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

func conversationIDsByUser(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string) ([]string, error) {
	rows, err := svcCtx.Model.Conversation.FindFieldsByCondition(ctx, nil, []condition.Field{conversationmodel.Id},
		condition.NewChain().Equal(conversationmodel.UserUuid, userUUID).Build()...)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Id)
	}
	return ids, nil
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
		if rows[i].PinnedAt.Valid != rows[j].PinnedAt.Valid {
			return rows[i].PinnedAt.Valid
		}
		if rows[i].PinnedAt.Valid {
			return rows[i].PinnedAt.Time.After(rows[j].PinnedAt.Time)
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
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
