package svc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jzero-io/agentrazor/server/internal/agent"
	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	conversationtokenusageeventmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_token_usage_event"
)

func (sc *ServiceContext) installAgentTokenUsageRecorder() {
	if sc.AgentThreads == nil {
		return
	}
	sc.AgentThreads.SetTokenUsageRecorder(sc.recordAgentTokenUsage)
}

func (sc *ServiceContext) recordAgentTokenUsage(ctx context.Context, event agent.TokenUsageEvent) error {
	conversation, err := sc.Model.Conversation.FindOne(ctx, nil, event.ConversationID)
	if errors.Is(err, conversationmodel.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	var modelContextWindow sql.NullInt64
	if event.ModelContextWindow != nil {
		modelContextWindow = sql.NullInt64{Int64: *event.ModelContextWindow, Valid: true}
	}
	return sc.Model.ConversationTokenUsageEvent.InsertV2(ctx, nil, &conversationtokenusageeventmodel.ConversationTokenUsageEvent{
		ConversationId:             event.ConversationID,
		UserUuid:                   conversation.UserUuid,
		TurnId:                     event.TurnID,
		LastInputTokens:            event.Last.InputTokens,
		LastCachedInputTokens:      event.Last.CachedInputTokens,
		LastCacheWriteInputTokens:  event.Last.CacheWriteInputTokens,
		LastOutputTokens:           event.Last.OutputTokens,
		LastReasoningOutputTokens:  event.Last.ReasoningOutputTokens,
		LastTotalTokens:            event.Last.TotalTokens,
		TotalInputTokens:           event.Total.InputTokens,
		TotalCachedInputTokens:     event.Total.CachedInputTokens,
		TotalCacheWriteInputTokens: event.Total.CacheWriteInputTokens,
		TotalOutputTokens:          event.Total.OutputTokens,
		TotalReasoningOutputTokens: event.Total.ReasoningOutputTokens,
		TotalTokens:                event.Total.TotalTokens,
		ModelContextWindow:         modelContextWindow,
	})
}
