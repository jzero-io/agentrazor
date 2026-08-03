# AgentRazor

[![服务端测试](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/server.yaml?branch=main&label=server&logo=go&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/server.yaml)
[![管理端测试](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/web.yaml?branch=main&label=admin%20web&logo=vuedotjs&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/web.yaml)
[![Agent 用户端测试](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/agent.yaml?branch=main&label=agent%20web&logo=vuedotjs&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/agent.yaml)

[English](README.md) | 简体中文

AgentRazor 是一个面向业务插件的通用 AI Agent 基座，仓库包含：

- `server`：基于 jzero 的服务端，负责 Codex app-server 生命周期、会话管理和 SSE。
- `agent`：面向普通用户的 Vue 3 对话页面。
- `web`：Vue 3 管理端。
- `core-engine`：管理端共享能力。

## 本地开发

### 服务端

```shell
cd server
go test ./...
go run . server -c etc/etc.yaml
```

Codex 运行时支持通过 `CODEX_BINARY_PATH`、`CODEX_HOME_PATH`、
`AGENT_WORKSPACE`、`AGENT_SANDBOX` 和 `AGENT_ENABLED` 调整。

### Agent 用户端

```shell
cd agent
pnpm install
pnpm typecheck
pnpm dev
```

开发地址为 `http://localhost:5174`，`/api` 默认代理到
`http://localhost:8001`。

### 管理端

```shell
cd web
pnpm install
pnpm typecheck
pnpm dev
```
