package config

import "github.com/jzero-io/agentrazor/core-engine/config"

type Config struct {
	config.Config
	Agent AgentConf
}

type AgentConf struct {
	CodexHome      string `json:",default=data/codex-home"`
	AgentrazorHome string `json:",default=data/agentrazor-home"`
}
