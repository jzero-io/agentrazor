package config

import "github.com/jzero-io/agentrazor/core-engine/config"

type Config struct {
	config.Config
	Agent AgentConf
}

type AgentConf struct {
	BinaryPath          string   `json:",default=codex"`
	CodexHome           string   `json:",default=data/codex-home"`
	AgentrazorHome      string   `json:",default=data/agentrazor-home"`
	Sandbox             string   `json:",default=workspace-write,options=read-only|workspace-write|danger-full-access"`
	DisableApps         bool     `json:",default=true"`
	DisabledMCPServers  []string `json:",optional"`
	StartTimeoutSeconds int      `json:",default=15"`
	ModelProvider       string   `json:",optional"`
	Model               string   `json:",optional"`
	ReasoningEffort     string   `json:",optional"`
}
