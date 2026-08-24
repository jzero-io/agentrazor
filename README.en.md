# AgentRazor

[![Server](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/server.yaml?branch=main&label=server&logo=go&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/server.yaml)
[![Admin Web](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/web.yaml?branch=main&label=admin%20web&logo=vuedotjs&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/web.yaml)
[![Agent Web](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/agent.yaml?branch=main&label=agent%20web&logo=vuedotjs&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/agent.yaml)

[简体中文](README.md) | English

AgentRazor is a plugin-oriented AI agent platform built around the Codex app-server protocol. It provides a user chat application, an administration console, Skills management, app-server configuration management, conversation lifecycle management, and token usage metrics.

## Features

- User-facing Agent chat UI with login, Access Token/Refresh Token renewal, streaming answers, process status display, Mermaid rendering, generated image display, archives, pins, groups, and resumable conversation state.
- jzero REST server for authentication, conversation metadata, SSE event fan-out, token usage recording, Codex app-server lifecycle integration, and plugin capabilities.
- Plugin-oriented Agent capabilities: the server can load standalone plugin binaries and plugin configuration; plugins can provide CLI tools, Skills, migrations, and runtime capabilities. Codex runtime syncs plugin Skills and invokes them from Agent conversations when needed.
- Admin console for users, roles, menus, Agent Skills, Codex configuration files, app-server restart, and homepage Agent metrics.
- Skills management for uploading, browsing, editing, and deleting Codex Skills.
- Docker Compose deployment for server, agent UI, admin UI, PostgreSQL, and Redis.

## Repository Layout

- `server`: jzero REST server with Codex app-server lifecycle management, conversations, SSE, migrations, plugin loading, and Skills synchronization.
- `agent`: user-facing Vue 3 chat application with login state, token renewal, streaming conversation display, and conversation actions.
- `web`: Vue 3 administration application.
- `core-engine`: shared base engine used by the admin web application.
- `deploy/docker-compose`: Docker Compose deployment configuration.
- `docs`: project documentation.
- `specs`: product, feature, and technical research documents.

## Local Development

### Server

```shell
cd server
go test ./...
go run . server -c etc/etc.yaml
```

Main runtime configuration lives in `server/etc/etc.yaml` and the configured Codex home directory. The Codex app-server reads `config.toml`, `models.json`, and `auth.json` from Codex home.

### Agent UI

```shell
cd agent
pnpm install
pnpm typecheck
pnpm dev
```

The development server listens on `http://localhost:5174` and proxies `/api` to `http://localhost:8001`.

### Admin Console

```shell
cd web
pnpm install
pnpm typecheck
pnpm dev
```

## Deployment

The default deployment configuration is in `deploy/docker-compose`.

```shell
cd deploy/docker-compose
docker compose up -d --build
```

Common service-only deployment commands:

```shell
docker compose build server && docker compose up -d --no-deps server
docker compose build agent && docker compose up -d --no-deps agent
docker compose build web && docker compose up -d --no-deps web
```

## Specifications

Use `specs/` to keep planning and design material:

- `specs/features`: feature specifications and implementation notes.
- `specs/product`: product requirements and UX decisions.
- `specs/research`: technical research, protocol analysis, and design records.

## License

AgentRazor is released under the [Apache-2.0 License](LICENSE).
