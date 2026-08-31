package conversation_token_usage_event

import (
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
