package conversation_token_usage_event

import (
	"context"
	"time"

	"github.com/eddieowens/opts"
	"github.com/jzero-io/jzero/core/stores/modelx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ConversationTokenUsageEventModel = (*customConversationTokenUsageEventModel)(nil)

type (
	// ConversationTokenUsageEventModel is an interface to be customized, add more methods here,
	// and implement the added methods in customConversationTokenUsageEventModel.
	ConversationTokenUsageEventModel interface {
		conversationTokenUsageEventModel
		TokenUsageTrend(ctx context.Context, userUUID, dimension string) ([]TokenUsageTrendPoint, error)
		TokenUsageSummary(ctx context.Context, userUUID string) (*TokenUsageSummaryRow, error)
		TokenUsageAccounts(ctx context.Context, userUUID, username string, current, size int) ([]TokenUsageAccountRow, error)
		TokenUsageConversations(ctx context.Context, userUUID, conversationID string, current, size int) ([]TokenUsageConversationRow, error)
		TokenQuotaWindowUsage(ctx context.Context, userUUID string, fiveHourStart, sevenDayStart time.Time) (*TokenQuotaWindowUsageRow, error)
	}

	customConversationTokenUsageEventModel struct {
		*defaultConversationTokenUsageEventModel
	}
)

// NewConversationTokenUsageEventModel returns a model for the database table.
func NewConversationTokenUsageEventModel(conn sqlx.SqlConn, op ...opts.Opt[modelx.ModelOpts]) ConversationTokenUsageEventModel {
	return &customConversationTokenUsageEventModel{
		defaultConversationTokenUsageEventModel: newConversationTokenUsageEventModel(conn, op...),
	}
}
