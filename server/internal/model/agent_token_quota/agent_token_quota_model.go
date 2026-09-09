package agent_token_quota

import (
	"context"
	"database/sql"
	"time"

	"github.com/eddieowens/opts"
	"github.com/jzero-io/jzero/core/stores/modelx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AgentTokenQuotaModel = (*customAgentTokenQuotaModel)(nil)

type (
	// AgentTokenQuotaModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAgentTokenQuotaModel.
	AgentTokenQuotaModel interface {
		agentTokenQuotaModel
		SaveGlobal(ctx context.Context, fiveHourLimitTokens, sevenDayLimitTokens int64) error
		SaveUser(ctx context.Context, userUUID string, enabled, fiveHourDisabled bool, fiveHourLimitTokens, sevenDayLimitTokens sql.NullInt64) error
		ResetUser(ctx context.Context, userUUID string, resetAt time.Time) error
		DeleteUser(ctx context.Context, userUUID string) error
	}

	customAgentTokenQuotaModel struct {
		*defaultAgentTokenQuotaModel
	}
)

// NewAgentTokenQuotaModel returns a model for the database table.
func NewAgentTokenQuotaModel(conn sqlx.SqlConn, op ...opts.Opt[modelx.ModelOpts]) AgentTokenQuotaModel {
	return &customAgentTokenQuotaModel{
		defaultAgentTokenQuotaModel: newAgentTokenQuotaModel(conn, op...),
	}
}
