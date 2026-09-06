package conversation

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type TokenUsageDetails struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewTokenUsageDetails(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *TokenUsageDetails {
	return &TokenUsageDetails{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *TokenUsageDetails) TokenUsageDetails(req *types.TokenUsageDetailsRequest) (*types.TokenUsageDetailsResponse, error) {
	userUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	superAdmin, err := isSuperAdmin(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	if superAdmin {
		userUUID = ""
	}

	current, size := normalizeTokenUsagePage(req.Current, req.Size)
	summary, err := l.svcCtx.Model.ConversationTokenUsageEvent.TokenUsageSummary(l.ctx, userUUID)
	if err != nil {
		return nil, err
	}
	rows, err := l.svcCtx.Model.ConversationTokenUsageEvent.TokenUsageAccounts(l.ctx, userUUID, req.Username, current, size)
	if err != nil {
		return nil, err
	}

	accounts := make([]types.TokenUsageAccount, 0, len(rows))
	var total int64
	for _, row := range rows {
		total = row.PageTotal
		accounts = append(accounts, types.TokenUsageAccount{
			UserUuid:          row.UserUUID,
			Username:          row.Username,
			Nickname:          row.Nickname,
			TotalTokens:       row.TotalTokens,
			ConversationCount: row.ConversationCount,
			TurnCount:         row.TurnCount,
		})
	}

	return &types.TokenUsageDetailsResponse{
		PageResponse: types.PageResponse{Current: current, Size: size, Total: total},
		Summary: types.TokenUsageSummary{
			InputTokens:           summary.InputTokens,
			CachedInputTokens:     summary.CachedInputTokens,
			CacheWriteInputTokens: summary.CacheWriteInputTokens,
			OutputTokens:          summary.OutputTokens,
			ReasoningOutputTokens: summary.ReasoningOutputTokens,
			TotalTokens:           summary.TotalTokens,
		},
		Accounts: accounts,
	}, nil
}

func normalizeTokenUsagePage(current, size int) (int, int) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return current, size
}
