package conversation

import (
	"github.com/eddieowens/opts"
	"github.com/jzero-io/jzero/core/stores/modelx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ConversationModel = (*customConversationModel)(nil)

type (
	ConversationModel interface {
		conversationModel
	}

	customConversationModel struct {
		*defaultConversationModel
	}
)

func NewConversationModel(conn sqlx.SqlConn, op ...opts.Opt[modelx.ModelOpts]) ConversationModel {
	return &customConversationModel{defaultConversationModel: newConversationModel(conn, op...)}
}
