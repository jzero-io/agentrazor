package conversation

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
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
	threads, err := l.listOwnedThreadMetadata(owned)
	if err != nil {
		return nil, err
	}
	order := conversationOrder(owned)
	sort.SliceStable(threads, func(i, j int) bool {
		left, right := order[threads[i].ID], order[threads[j].ID]
		if left != nil && right != nil && !left.CreateTime.Equal(right.CreateTime) {
			return left.CreateTime.After(right.CreateTime)
		}
		return threads[i].ID > threads[j].ID
	})

	resp = &types.StatsResponse{}
	for _, thread := range threads {
		conversation := toConversation(thread)
		setConversationActiveRun(&conversation, l.svcCtx.AgentThreads, thread.ID)

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
	info, err := auth.Info(l.ctx)
	if err != nil {
		return false, err
	}
	if len(info.RoleUuids) == 0 {
		return false, nil
	}
	roles, err := l.svcCtx.Model.ManageRole.FindByCondition(l.ctx, nil, condition.NewChain().
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

func (l *Stats) listOwnedThreadMetadata(owned []*conversationmodel.Conversation) ([]agentdomain.StoredThread, error) {
	threads := make([]agentdomain.StoredThread, 0, len(owned))
	for _, row := range owned {
		thread, err := l.svcCtx.AgentThreads.Metadata(l.ctx, row.Id)
		if errors.Is(err, agentdomain.ErrThreadNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		threads = append(threads, thread)
	}
	return threads, nil
}

func (l *Stats) tokenUsageTotal(userUUID string, superAdmin bool) (int64, bool, error) {
	whereClause := "where user_uuid = $1"
	args := []any{userUUID}
	if superAdmin {
		whereClause = ""
		args = nil
	}

	var total int64
	err := l.svcCtx.SqlxConn.QueryRowCtx(l.ctx, &total, `
		select coalesce(sum(total_tokens), 0)
		from (
			select distinct on (conversation_id) conversation_id, total_tokens
			from conversation_token_usage_event
			`+whereClause+`
			order by conversation_id, id desc
		) latest
	`, args...)
	if err != nil {
		return 0, false, err
	}
	var count int64
	err = l.svcCtx.SqlxConn.QueryRowCtx(l.ctx, &count, `
		select count(1)
		from conversation_token_usage_event
		`+whereClause+`
	`, args...)
	if err != nil {
		return 0, false, err
	}
	return total, count > 0, nil
}
