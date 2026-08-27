package svc

import (
	"time"

	"github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/config"
)

func (sc *ServiceContext) AgentOptionsFromConfig(agentConfig config.AgentConf) agent.CodexAppServerOptions {
	return agent.CodexAppServerOptions{
		Binary:             agentConfig.BinaryPath,
		CodexHome:          agentConfig.CodexHome,
		AgentrazorHome:     agentConfig.AgentrazorHome,
		DisableApps:        agentConfig.DisableApps,
		DisabledMCPServers: agentConfig.DisabledMCPServers,
		StartTimeout:       time.Duration(agentConfig.StartTimeoutSeconds) * time.Second,
		ModelProvider:      agentConfig.ModelProvider,
		Model:              agentConfig.Model,
		ReasoningEffort:    agentConfig.ReasoningEffort,
	}
}
