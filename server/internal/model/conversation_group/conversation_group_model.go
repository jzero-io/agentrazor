package conversation_group

import (
	"github.com/eddieowens/opts"
	"github.com/jzero-io/jzero/core/stores/modelx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ConversationGroupModel = (*customConversationGroupModel)(nil)

type (
	ConversationGroupModel interface {
		conversationGroupModel
	}

	customConversationGroupModel struct {
		*defaultConversationGroupModel
	}
)

func NewConversationGroupModel(conn sqlx.SqlConn, op ...opts.Opt[modelx.ModelOpts]) ConversationGroupModel {
	return &customConversationGroupModel{defaultConversationGroupModel: newConversationGroupModel(conn, op...)}
}
