package agent_character

import (
	"github.com/eddieowens/opts"
	"github.com/jzero-io/jzero/core/stores/modelx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AgentCharacterModel = (*customAgentCharacterModel)(nil)

type (
	// AgentCharacterModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAgentCharacterModel.
	AgentCharacterModel interface {
		agentCharacterModel
	}

	customAgentCharacterModel struct {
		*defaultAgentCharacterModel
	}
)

// NewAgentCharacterModel returns a model for the database table.
func NewAgentCharacterModel(conn sqlx.SqlConn, op ...opts.Opt[modelx.ModelOpts]) AgentCharacterModel {
	return &customAgentCharacterModel{
		defaultAgentCharacterModel: newAgentCharacterModel(conn, op...),
	}
}
