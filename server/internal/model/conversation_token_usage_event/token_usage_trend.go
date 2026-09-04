package conversation_token_usage_event

import "context"

type TokenUsageTrendPoint struct {
	Period string `db:"period"`
	Tokens int64  `db:"tokens"`
}

const tokenUsageTrendDayQuery = `
WITH ordered AS (
    SELECT create_time,
           total_tokens,
           lag(total_tokens) OVER (PARTITION BY conversation_id ORDER BY id) AS previous_total
    FROM conversation_token_usage_event
    WHERE (NOT $1::boolean OR user_uuid = $2)
), deltas AS (
    SELECT create_time,
           CASE
               WHEN previous_total IS NULL OR total_tokens < previous_total THEN total_tokens
               ELSE total_tokens - previous_total
           END AS tokens
    FROM ordered
), buckets AS (
    SELECT generate_series(
        date_trunc('day', CURRENT_TIMESTAMP) - interval '29 days',
        date_trunc('day', CURRENT_TIMESTAMP),
        interval '1 day'
    ) AS bucket
)
SELECT to_char(b.bucket, 'YYYY-MM-DD') AS period,
       COALESCE(SUM(d.tokens), 0)::bigint AS tokens
FROM buckets b
LEFT JOIN deltas d
       ON d.create_time >= b.bucket
      AND d.create_time < b.bucket + interval '1 day'
GROUP BY b.bucket
ORDER BY b.bucket`

const tokenUsageTrendMonthQuery = `
WITH ordered AS (
    SELECT create_time,
           total_tokens,
           lag(total_tokens) OVER (PARTITION BY conversation_id ORDER BY id) AS previous_total
    FROM conversation_token_usage_event
    WHERE (NOT $1::boolean OR user_uuid = $2)
), deltas AS (
    SELECT create_time,
           CASE
               WHEN previous_total IS NULL OR total_tokens < previous_total THEN total_tokens
               ELSE total_tokens - previous_total
           END AS tokens
    FROM ordered
), buckets AS (
    SELECT generate_series(
        date_trunc('month', CURRENT_TIMESTAMP) - interval '11 months',
        date_trunc('month', CURRENT_TIMESTAMP),
        interval '1 month'
    ) AS bucket
)
SELECT to_char(b.bucket, 'YYYY-MM') AS period,
       COALESCE(SUM(d.tokens), 0)::bigint AS tokens
FROM buckets b
LEFT JOIN deltas d
       ON d.create_time >= b.bucket
      AND d.create_time < b.bucket + interval '1 month'
GROUP BY b.bucket
ORDER BY b.bucket`

func (m *customConversationTokenUsageEventModel) TokenUsageTrend(
	ctx context.Context,
	userUUID string,
	dimension string,
) ([]TokenUsageTrendPoint, error) {
	query := tokenUsageTrendDayQuery
	if dimension == "month" {
		query = tokenUsageTrendMonthQuery
	}

	points := make([]TokenUsageTrendPoint, 0)
	filterUser := userUUID != ""
	if err := m.conn.QueryRowsCtx(ctx, &points, query, filterUser, userUUID); err != nil {
		return nil, err
	}
	return points, nil
}
