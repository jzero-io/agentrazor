package conversation

import (
	"context"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type TokenUsageConversations struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewTokenUsageConversations(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *TokenUsageConversations {
	return &TokenUsageConversations{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *TokenUsageConversations) TokenUsageConversations(req *types.TokenUsageConversationsRequest) (resp *types.TokenUsageConversationsResponse, err error) {
	currentUserUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	superAdmin, err := isSuperAdmin(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}

	userUUID := req.UserUuid
	if !superAdmin {
		userUUID = currentUserUUID
	}
	current, size := normalizeTokenUsagePage(req.Current, req.Size)
	rows, err := l.svcCtx.Model.ConversationTokenUsageEvent.TokenUsageConversations(
		l.ctx, userUUID, req.ConversationId, current, size,
	)
	if err != nil {
		return nil, err
	}

	conversations := make([]types.TokenUsageConversation, 0)
	conversationIndexes := make(map[string]int)
	var total int64
	for _, row := range rows {
		total = row.PageTotal
		index, ok := conversationIndexes[row.ConversationID]
		if !ok {
			index = len(conversations)
			conversationIndexes[row.ConversationID] = index
			conversations = append(conversations, types.TokenUsageConversation{
				ConversationId: row.ConversationID,
				TotalTokens:    row.ConversationTotal,
				TurnCount:      row.ConversationTurnCount,
				FirstUsedAt:    row.FirstUsedAt.Format(time.RFC3339),
				LastUsedAt:     row.LastUsedAt.Format(time.RFC3339),
				Turns:          make([]types.TokenUsageTurn, 0, row.ConversationTurnCount),
			})
		}

		var modelContextWindow *int64
		if row.ModelContextWindow.Valid {
			value := row.ModelContextWindow.Int64
			modelContextWindow = &value
		}
		conversations[index].Turns = append(conversations[index].Turns, types.TokenUsageTurn{
			TurnId:                row.TurnID,
			InputTokens:           row.InputTokens,
			CachedInputTokens:     row.CachedInputTokens,
			CacheWriteInputTokens: row.CacheWriteInputTokens,
			OutputTokens:          row.OutputTokens,
			ReasoningOutputTokens: row.ReasoningOutputTokens,
			TotalTokens:           row.TotalTokens,
			ModelContextWindow:    modelContextWindow,
			StartedAt:             row.StartedAt.Format(time.RFC3339),
			UpdatedAt:             row.UpdatedAt.Format(time.RFC3339),
		})
	}

	return &types.TokenUsageConversationsResponse{
		PageResponse:  types.PageResponse{Current: current, Size: size, Total: total},
		Conversations: conversations,
	}, nil
}
