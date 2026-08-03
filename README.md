# AgentRazor

[![Server](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/server.yaml?branch=main&label=server&logo=go&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/server.yaml)
[![Admin Web](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/web.yaml?branch=main&label=admin%20web&logo=vuedotjs&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/web.yaml)
[![Agent Web](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/agent.yaml?branch=main&label=agent%20web&logo=vuedotjs&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/agent.yaml)

English | [简体中文](README.zh-CN.md)

AgentRazor is a plugin-oriented AI agent runtime. The repository contains:

- `server`: jzero REST server, Codex app-server lifecycle management, conversations and SSE.
- `agent`: user-facing Vue 3 chat application.
- `web`: Vue 3 administration application.
- `core-engine`: shared administration engine.

## Local development

### Server

```shell
cd server
go test ./...
go run . server -c etc/etc.yaml
```

The Codex runtime can be configured with `CODEX_BINARY_PATH`, `CODEX_HOME_PATH`,
`AGENT_WORKSPACE`, `AGENT_SANDBOX`, and `AGENT_ENABLED`.

### Agent web

```shell
cd agent
pnpm install
pnpm typecheck
pnpm dev
```

The development server listens on `http://localhost:5174` and proxies `/api` to
`http://localhost:8001`.

### Admin web

```shell
cd web
pnpm install
pnpm typecheck
pnpm dev
```
