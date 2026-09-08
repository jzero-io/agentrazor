package svc

import (
	"time"

	"github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/config"
)

func (sc *ServiceContext) AgentOptionsFromConfig(agentConfig config.AgentConf) agent.CodexAppServerOptions {
	return agent.CodexAppServerOptions{
		CodexHome:          agentConfig.CodexHome,
		AgentrazorHome:     agentConfig.AgentrazorHome,
		DisabledMCPServers: agentConfig.DisabledMCPServers,
		StartTimeout:       time.Duration(agentConfig.StartTimeoutSeconds) * time.Second,
		ModelProvider:      agentConfig.ModelProvider,
		Model:              agentConfig.Model,
		ReasoningEffort:    agentConfig.ReasoningEffort,
	}
}
