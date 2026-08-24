package svc

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlc"

	"github.com/jzero-io/agentrazor/server/internal/agent"
)

func (sc *ServiceContext) installAgentTokenUsageRecorder() {
	if sc.AgentThreads == nil {
		return
	}
	sc.AgentThreads.SetTokenUsageRecorder(sc.recordAgentTokenUsage)
}

func (sc *ServiceContext) recordAgentTokenUsage(ctx context.Context, event agent.TokenUsageEvent) error {
	var userUUID string
	err := sc.SqlxConn.QueryRowCtx(ctx, &userUUID, "select user_uuid from conversation where id = $1", event.ConversationID)
	if err == sqlc.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	var modelContextWindow any
	if event.ModelContextWindow != nil {
		modelContextWindow = *event.ModelContextWindow
	}
	_, err = sc.SqlxConn.ExecCtx(ctx, `
		insert into conversation_token_usage_event (
			conversation_id,
			user_uuid,
			turn_id,
			last_input_tokens,
			last_cached_input_tokens,
			last_cache_write_input_tokens,
			last_output_tokens,
			last_reasoning_output_tokens,
			last_total_tokens,
			total_input_tokens,
			total_cached_input_tokens,
			total_cache_write_input_tokens,
			total_output_tokens,
			total_reasoning_output_tokens,
			total_tokens,
			model_context_window
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`,
		event.ConversationID,
		userUUID,
		event.TurnID,
		event.Last.InputTokens,
		event.Last.CachedInputTokens,
		event.Last.CacheWriteInputTokens,
		event.Last.OutputTokens,
		event.Last.ReasoningOutputTokens,
		event.Last.TotalTokens,
		event.Total.InputTokens,
		event.Total.CachedInputTokens,
		event.Total.CacheWriteInputTokens,
		event.Total.OutputTokens,
		event.Total.ReasoningOutputTokens,
		event.Total.TotalTokens,
		modelContextWindow,
	)
	return err
}

func (sc *ServiceContext) DeleteConversationTokenUsageEvents(ctx context.Context, conversationID, userUUID string) error {
	_, err := sc.SqlxConn.ExecCtx(ctx, `
		delete from conversation_token_usage_event
		where conversation_id = $1 and user_uuid = $2
	`, conversationID, userUUID)
	return err
}
