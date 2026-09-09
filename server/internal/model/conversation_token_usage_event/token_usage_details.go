package conversation_token_usage_event

import (
	"context"
	"database/sql"
	"time"
)

type TokenUsageSummaryRow struct {
	InputTokens           int64 `db:"input_tokens"`
	CachedInputTokens     int64 `db:"cached_input_tokens"`
	CacheWriteInputTokens int64 `db:"cache_write_input_tokens"`
	OutputTokens          int64 `db:"output_tokens"`
	ReasoningOutputTokens int64 `db:"reasoning_output_tokens"`
	TotalTokens           int64 `db:"total_tokens"`
}

type TokenUsageAccountRow struct {
	UserUUID          string `db:"user_uuid"`
	Username          string `db:"username"`
	Nickname          string `db:"nickname"`
	TotalTokens       int64  `db:"total_tokens"`
	ConversationCount int64  `db:"conversation_count"`
	TurnCount         int64  `db:"turn_count"`
	PageTotal         int64  `db:"page_total"`
}

type TokenUsageConversationRow struct {
	ConversationID        string        `db:"conversation_id"`
	ConversationTotal     int64         `db:"conversation_total"`
	ConversationTurnCount int64         `db:"conversation_turn_count"`
	FirstUsedAt           time.Time     `db:"first_used_at"`
	LastUsedAt            time.Time     `db:"last_used_at"`
	TurnID                string        `db:"turn_id"`
	InputTokens           int64         `db:"input_tokens"`
	CachedInputTokens     int64         `db:"cached_input_tokens"`
	CacheWriteInputTokens int64         `db:"cache_write_input_tokens"`
	OutputTokens          int64         `db:"output_tokens"`
	ReasoningOutputTokens int64         `db:"reasoning_output_tokens"`
	TotalTokens           int64         `db:"total_tokens"`
	ModelContextWindow    sql.NullInt64 `db:"model_context_window"`
	StartedAt             time.Time     `db:"started_at"`
	UpdatedAt             time.Time     `db:"updated_at"`
	PageTotal             int64         `db:"page_total"`
}

const tokenUsageDeltaCTE = `
WITH ordered AS (
    SELECT e.*,
           lag(total_input_tokens) OVER conversation_order AS previous_input,
           lag(total_cached_input_tokens) OVER conversation_order AS previous_cached_input,
           lag(total_cache_write_input_tokens) OVER conversation_order AS previous_cache_write_input,
           lag(total_output_tokens) OVER conversation_order AS previous_output,
           lag(total_reasoning_output_tokens) OVER conversation_order AS previous_reasoning_output,
           lag(total_tokens) OVER conversation_order AS previous_total
    FROM conversation_token_usage_event e
    WHERE (NOT $1::boolean OR e.user_uuid = $2)
    WINDOW conversation_order AS (PARTITION BY user_uuid, conversation_id ORDER BY id)
), deltas AS (
    SELECT *,
           CASE WHEN previous_input IS NULL OR total_input_tokens < previous_input THEN total_input_tokens ELSE total_input_tokens - previous_input END AS input_delta,
           CASE WHEN previous_cached_input IS NULL OR total_cached_input_tokens < previous_cached_input THEN total_cached_input_tokens ELSE total_cached_input_tokens - previous_cached_input END AS cached_input_delta,
           CASE WHEN previous_cache_write_input IS NULL OR total_cache_write_input_tokens < previous_cache_write_input THEN total_cache_write_input_tokens ELSE total_cache_write_input_tokens - previous_cache_write_input END AS cache_write_input_delta,
           CASE WHEN previous_output IS NULL OR total_output_tokens < previous_output THEN total_output_tokens ELSE total_output_tokens - previous_output END AS output_delta,
           CASE WHEN previous_reasoning_output IS NULL OR total_reasoning_output_tokens < previous_reasoning_output THEN total_reasoning_output_tokens ELSE total_reasoning_output_tokens - previous_reasoning_output END AS reasoning_output_delta,
           CASE WHEN previous_total IS NULL OR total_tokens < previous_total THEN total_tokens ELSE total_tokens - previous_total END AS total_delta
    FROM ordered
)
`

const tokenUsageSummaryQuery = tokenUsageDeltaCTE + `
SELECT COALESCE(SUM(input_delta), 0)::bigint AS input_tokens,
       COALESCE(SUM(cached_input_delta), 0)::bigint AS cached_input_tokens,
       COALESCE(SUM(cache_write_input_delta), 0)::bigint AS cache_write_input_tokens,
       COALESCE(SUM(output_delta), 0)::bigint AS output_tokens,
       COALESCE(SUM(reasoning_output_delta), 0)::bigint AS reasoning_output_tokens,
       COALESCE(SUM(total_delta), 0)::bigint AS total_tokens
FROM deltas`

const tokenUsageAccountsQuery = tokenUsageDeltaCTE + `
, account_usage AS (
    SELECT d.user_uuid,
           SUM(d.total_delta)::bigint AS total_tokens,
           COUNT(DISTINCT d.conversation_id)::bigint AS conversation_count,
           COUNT(DISTINCT (d.conversation_id, d.turn_id))::bigint AS turn_count
    FROM deltas d
    GROUP BY d.user_uuid
), all_accounts AS (
    SELECT u.uuid AS user_uuid,
           u.username,
           u.nickname,
           COALESCE(a.total_tokens, 0)::bigint AS total_tokens,
           COALESCE(a.conversation_count, 0)::bigint AS conversation_count,
           COALESCE(a.turn_count, 0)::bigint AS turn_count
    FROM manage_user u
    LEFT JOIN account_usage a ON a.user_uuid = u.uuid
    WHERE NOT $1::boolean OR u.uuid = $2

    UNION ALL

    SELECT a.user_uuid,
           '' AS username,
           '' AS nickname,
           a.total_tokens,
           a.conversation_count,
           a.turn_count
    FROM account_usage a
    LEFT JOIN manage_user u ON u.uuid = a.user_uuid
    WHERE u.uuid IS NULL
), filtered AS (
    SELECT *, COUNT(*) OVER()::bigint AS page_total
    FROM all_accounts
    WHERE $3 = '' OR username ILIKE '%' || $3 || '%' OR nickname ILIKE '%' || $3 || '%'
)
SELECT user_uuid, username, nickname, total_tokens, conversation_count, turn_count, page_total
FROM filtered
ORDER BY total_tokens DESC, username
LIMIT $4 OFFSET $5`

const tokenUsageConversationsQuery = tokenUsageDeltaCTE + `
, turn_usage AS (
    SELECT conversation_id,
           turn_id,
           SUM(input_delta)::bigint AS input_tokens,
           SUM(cached_input_delta)::bigint AS cached_input_tokens,
           SUM(cache_write_input_delta)::bigint AS cache_write_input_tokens,
           SUM(output_delta)::bigint AS output_tokens,
           SUM(reasoning_output_delta)::bigint AS reasoning_output_tokens,
           SUM(total_delta)::bigint AS total_tokens,
           MAX(model_context_window) AS model_context_window,
           MIN(create_time) AS started_at,
           MAX(create_time) AS updated_at
    FROM deltas
    GROUP BY conversation_id, turn_id
), conversation_usage AS (
    SELECT conversation_id,
           SUM(total_tokens)::bigint AS conversation_total,
           COUNT(*)::bigint AS conversation_turn_count,
           MIN(started_at) AS first_used_at,
           MAX(updated_at) AS last_used_at
    FROM turn_usage
    GROUP BY conversation_id
), filtered AS (
    SELECT *, COUNT(*) OVER()::bigint AS page_total
    FROM conversation_usage
    WHERE $3 = '' OR conversation_id ILIKE '%' || $3 || '%'
), paged AS (
    SELECT *
    FROM filtered
    ORDER BY last_used_at DESC, conversation_id
    LIMIT $4 OFFSET $5
)
SELECT p.conversation_id,
       p.conversation_total,
       p.conversation_turn_count,
       p.first_used_at,
       p.last_used_at,
       t.turn_id,
       t.input_tokens,
       t.cached_input_tokens,
       t.cache_write_input_tokens,
       t.output_tokens,
       t.reasoning_output_tokens,
       t.total_tokens,
       t.model_context_window,
       t.started_at,
       t.updated_at,
       p.page_total
FROM paged p
JOIN turn_usage t ON t.conversation_id = p.conversation_id
ORDER BY p.last_used_at DESC, p.conversation_id, t.updated_at DESC, t.turn_id`

func (m *customConversationTokenUsageEventModel) TokenUsageSummary(
	ctx context.Context,
	userUUID string,
) (*TokenUsageSummaryRow, error) {
	var row TokenUsageSummaryRow
	filterUser := userUUID != ""
	if err := m.conn.QueryRowCtx(ctx, &row, tokenUsageSummaryQuery, filterUser, userUUID); err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *customConversationTokenUsageEventModel) TokenUsageAccounts(
	ctx context.Context,
	userUUID, username string,
	current, size int,
) ([]TokenUsageAccountRow, error) {
	rows := make([]TokenUsageAccountRow, 0)
	filterUser := userUUID != ""
	offset := (current - 1) * size
	if err := m.conn.QueryRowsCtx(ctx, &rows, tokenUsageAccountsQuery, filterUser, userUUID, username, size, offset); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *customConversationTokenUsageEventModel) TokenUsageConversations(
	ctx context.Context,
	userUUID, conversationID string,
	current, size int,
) ([]TokenUsageConversationRow, error) {
	rows := make([]TokenUsageConversationRow, 0)
	offset := (current - 1) * size
	if err := m.conn.QueryRowsCtx(ctx, &rows, tokenUsageConversationsQuery, true, userUUID, conversationID, size, offset); err != nil {
		return nil, err
	}
	return rows, nil
}
