package agent_token_quota

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (m *customAgentTokenQuotaModel) SaveGlobal(
	ctx context.Context,
	fiveHourLimitTokens, sevenDayLimitTokens int64,
) error {
	statement := fmt.Sprintf(`
INSERT INTO %s (
    scope, enabled, five_hour_disabled, five_hour_limit_tokens,
    seven_day_limit_tokens, update_time
) VALUES ('global', true, false, $1, $2, CURRENT_TIMESTAMP)
ON CONFLICT (scope) WHERE scope = 'global'
DO UPDATE SET
    five_hour_limit_tokens = EXCLUDED.five_hour_limit_tokens,
    seven_day_limit_tokens = EXCLUDED.seven_day_limit_tokens,
    update_time = CURRENT_TIMESTAMP`, m.table)
	_, err := m.conn.ExecCtx(ctx, statement, fiveHourLimitTokens, sevenDayLimitTokens)
	return err
}

func (m *customAgentTokenQuotaModel) SaveUser(
	ctx context.Context,
	userUUID string,
	enabled, fiveHourDisabled bool,
	fiveHourLimitTokens, sevenDayLimitTokens sql.NullInt64,
) error {
	statement := fmt.Sprintf(`
INSERT INTO %s (
    scope, user_uuid, enabled, five_hour_disabled,
    five_hour_limit_tokens, seven_day_limit_tokens, update_time
) VALUES ('user', $1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
ON CONFLICT (user_uuid) WHERE scope = 'user'
DO UPDATE SET
    enabled = EXCLUDED.enabled,
    five_hour_disabled = EXCLUDED.five_hour_disabled,
    five_hour_limit_tokens = EXCLUDED.five_hour_limit_tokens,
    seven_day_limit_tokens = EXCLUDED.seven_day_limit_tokens,
    update_time = CURRENT_TIMESTAMP`, m.table)
	_, err := m.conn.ExecCtx(
		ctx,
		statement,
		userUUID,
		enabled,
		fiveHourDisabled,
		fiveHourLimitTokens,
		sevenDayLimitTokens,
	)
	return err
}

func (m *customAgentTokenQuotaModel) ResetUser(
	ctx context.Context,
	userUUID string,
	resetAt time.Time,
) error {
	statement := fmt.Sprintf(`
INSERT INTO %s (
    scope, user_uuid, enabled, five_hour_disabled,
    five_hour_limit_tokens, seven_day_limit_tokens, quota_reset_at, update_time
) VALUES ('user', $1, true, false, NULL, NULL, $2, CURRENT_TIMESTAMP)
ON CONFLICT (user_uuid) WHERE scope = 'user'
DO UPDATE SET
    quota_reset_at = EXCLUDED.quota_reset_at,
    update_time = CURRENT_TIMESTAMP`, m.table)
	_, err := m.conn.ExecCtx(ctx, statement, userUUID, resetAt)
	return err
}

func (m *customAgentTokenQuotaModel) DeleteUser(ctx context.Context, userUUID string) error {
	statement := fmt.Sprintf(`DELETE FROM %s WHERE scope = 'user' AND user_uuid = $1`, m.table)
	_, err := m.conn.ExecCtx(ctx, statement, userUUID)
	return err
}
