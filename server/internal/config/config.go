package config

import "github.com/jzero-io/agentrazor/core-engine/config"

type Config struct {
	config.Config
	Agent AgentConf
}

type AgentConf struct {
	CodexHome      string `json:",default=data"`
	AgentrazorHome string `json:",default=data/agentrazor-home"`
	CodexSocket    string `json:",default=/run/codex/app-server.sock"`
}
