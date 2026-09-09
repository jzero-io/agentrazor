package conversation_token_usage_event

import (
	"context"
	"database/sql"
	"time"
)

type TokenQuotaWindowUsageRow struct {
	FiveHourTokens  int64        `db:"five_hour_tokens"`
	SevenDayTokens  int64        `db:"seven_day_tokens"`
	FiveHourResetAt sql.NullTime `db:"five_hour_reset_at"`
	SevenDayResetAt sql.NullTime `db:"seven_day_reset_at"`
}

const tokenQuotaWindowUsageQuery = `
WITH recent AS (
    SELECT id, conversation_id, create_time, total_tokens
    FROM conversation_token_usage_event
    WHERE user_uuid = $1
      AND create_time >= $3
), relevant_conversations AS (
    SELECT DISTINCT conversation_id
    FROM recent
), boundary AS (
    SELECT previous.id,
           previous.conversation_id,
           previous.create_time,
           previous.total_tokens
    FROM relevant_conversations conversation
    CROSS JOIN LATERAL (
        SELECT event.id,
               event.conversation_id,
               event.create_time,
               event.total_tokens
        FROM conversation_token_usage_event event
        WHERE event.user_uuid = $1
          AND event.conversation_id = conversation.conversation_id
          AND event.create_time < $3
        ORDER BY event.id DESC
        LIMIT 1
    ) previous
), selected AS (
    SELECT * FROM boundary
    UNION ALL
    SELECT * FROM recent
), ordered AS (
    SELECT id,
           conversation_id,
           create_time,
           total_tokens,
           lag(total_tokens) OVER (
               PARTITION BY conversation_id
               ORDER BY id
           ) AS previous_total
    FROM selected
), deltas AS (
    SELECT create_time,
           CASE
               WHEN previous_total IS NULL OR total_tokens < previous_total THEN total_tokens
               ELSE total_tokens - previous_total
           END AS token_delta
    FROM ordered
)
SELECT COALESCE(
           SUM(token_delta) FILTER (WHERE create_time >= $2),
           0
       )::bigint AS five_hour_tokens,
       COALESCE(
           SUM(token_delta) FILTER (WHERE create_time >= $3),
           0
       )::bigint AS seven_day_tokens,
       MIN(create_time) FILTER (
           WHERE create_time >= $2 AND token_delta > 0
       ) + INTERVAL '5 hours' AS five_hour_reset_at,
       MIN(create_time) FILTER (
           WHERE create_time >= $3 AND token_delta > 0
       ) + INTERVAL '7 days' AS seven_day_reset_at
FROM deltas`

func (m *customConversationTokenUsageEventModel) TokenQuotaWindowUsage(
	ctx context.Context,
	userUUID string,
	fiveHourStart, sevenDayStart time.Time,
) (*TokenQuotaWindowUsageRow, error) {
	var row TokenQuotaWindowUsageRow
	if err := m.conn.QueryRowCtx(
		ctx,
		&row,
		tokenQuotaWindowUsageQuery,
		userUUID,
		fiveHourStart,
		sevenDayStart,
	); err != nil {
		return nil, err
	}
	return &row, nil
}
