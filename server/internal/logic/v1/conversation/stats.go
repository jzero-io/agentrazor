package conversation

import (
	"context"
	"errors"
	"net/http"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	conversationtokenusageeventmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_token_usage_event"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type Stats struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewStats(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Stats {
	return &Stats{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Stats) Stats() (resp *types.StatsResponse, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	uuid, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	superAdmin, err := l.isSuperAdmin()
	if err != nil {
		return nil, err
	}
	owned, err := l.conversationRows(uuid, superAdmin)
	if err != nil {
		return nil, err
	}
	threads, err := l.listOwnedThreads(owned)
	if err != nil {
		return nil, err
	}

	resp = &types.StatsResponse{}
	for _, thread := range threads {
		conversation := toConversation(thread)
		setConversationActiveTurn(&conversation, l.svcCtx.AgentThreads, thread.ID)

		resp.TotalConversations++
		if conversation.Status == "archived" {
			resp.ArchivedConversations++
		} else {
			resp.ActiveConversations++
		}
		if conversation.Running {
			resp.RunningConversations++
		}
	}

	tokenTotal, tokenAvailable, err := l.tokenUsageTotal(uuid, superAdmin)
	if err != nil {
		return nil, err
	}
	resp.TotalTokens = tokenTotal
	resp.TokenUsageAvailable = tokenAvailable
	return resp, nil
}

func (l *Stats) isSuperAdmin() (bool, error) {
	return isSuperAdmin(l.ctx, l.svcCtx)
}

func isSuperAdmin(ctx context.Context, svcCtx *svc.ServiceContext) (bool, error) {
	info, err := auth.Info(ctx)
	if err != nil {
		return false, err
	}
	if len(info.RoleUuids) == 0 {
		return false, nil
	}
	roles, err := svcCtx.Model.ManageRole.FindByCondition(ctx, nil, condition.NewChain().
		In(manage_role.Uuid, info.RoleUuids).
		Build()...)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role.Code == "R_SUPER" {
			return true, nil
		}
	}
	return false, nil
}

func (l *Stats) conversationRows(userUUID string, superAdmin bool) ([]*conversationmodel.Conversation, error) {
	fields := []condition.Field{conversationmodel.Id, conversationmodel.CreateTime, conversationmodel.UpdateTime}
	if superAdmin {
		return l.svcCtx.Model.Conversation.FindFieldsByCondition(l.ctx, nil, fields)
	}
	return l.svcCtx.Model.Conversation.FindFieldsByCondition(l.ctx, nil, fields,
		condition.NewChain().Equal(conversationmodel.UserUuid, userUUID).Build()...)
}

func (l *Stats) listOwnedThreads(owned []*conversationmodel.Conversation) ([]agentdomain.StoredThread, error) {
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

	threads := make([]agentdomain.StoredThread, 0, len(byID))
	for _, row := range owned {
		if thread, ok := byID[row.Id]; ok {
			threads = append(threads, thread)
		}
	}
	return threads, nil
}

func (l *Stats) tokenUsageTotal(userUUID string, superAdmin bool) (int64, bool, error) {
	chain := condition.NewChain().OrderByDesc(conversationtokenusageeventmodel.Id)
	if !superAdmin {
		chain = chain.Equal(conversationtokenusageeventmodel.UserUuid, userUUID)
	}
	rows, err := l.svcCtx.Model.ConversationTokenUsageEvent.FindFieldsByCondition(
		l.ctx,
		nil,
		[]condition.Field{
			conversationtokenusageeventmodel.ConversationId,
			conversationtokenusageeventmodel.TotalTokens,
		},
		chain.Build()...,
	)
	if err != nil {
		return 0, false, err
	}

	seen := make(map[string]struct{}, len(rows))
	var total int64
	for _, row := range rows {
		if _, ok := seen[row.ConversationId]; ok {
			continue
		}
		seen[row.ConversationId] = struct{}{}
		total += row.TotalTokens
	}
	return total, len(rows) > 0, nil
}
