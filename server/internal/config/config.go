package config

import "github.com/jzero-io/jzero-admin/core-engine/config"

type Config struct {
	config.Config
	Agent AgentConf
}

type AgentConf struct {
	Enabled             bool     `json:",default=true"`
	BinaryPath          string   `json:",default=codex"`
	CodexHome           string   `json:",default=data/codex-home"`
	Workspace           string   `json:",default=.."`
	Sandbox             string   `json:",default=read-only,options=read-only|workspace-write|danger-full-access"`
	ServiceName         string   `json:",default=agentrazor"`
	DisableApps         bool     `json:",default=true"`
	DisabledMCPServers  []string `json:",optional"`
	StartTimeoutSeconds int      `json:",default=15"`
	RunTimeoutSeconds   int      `json:",default=600"`
}
