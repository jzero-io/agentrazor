package agent_api_key

import (
	"github.com/eddieowens/opts"
	"github.com/jzero-io/jzero/core/stores/modelx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AgentApiKeyModel = (*customAgentApiKeyModel)(nil)

type (
	// AgentApiKeyModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAgentApiKeyModel.
	AgentApiKeyModel interface {
		agentApiKeyModel
	}

	customAgentApiKeyModel struct {
		*defaultAgentApiKeyModel
	}
)

// NewAgentApiKeyModel returns a model for the database table.
func NewAgentApiKeyModel(conn sqlx.SqlConn, op ...opts.Opt[modelx.ModelOpts]) AgentApiKeyModel {
	return &customAgentApiKeyModel{
		defaultAgentApiKeyModel: newAgentApiKeyModel(conn, op...),
	}
}
