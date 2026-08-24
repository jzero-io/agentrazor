# AgentRazor

[![Server](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/server.yaml?branch=main&label=server&logo=go&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/server.yaml)
[![Admin Web](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/web.yaml?branch=main&label=admin%20web&logo=vuedotjs&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/web.yaml)
[![Agent Web](https://img.shields.io/github/actions/workflow/status/jzero-io/agentrazor/agent.yaml?branch=main&label=agent%20web&logo=vuedotjs&style=flat-square)](https://github.com/jzero-io/agentrazor/actions/workflows/agent.yaml)

简体中文 | [English](README.en.md)

AgentRazor 是一个围绕 Codex app-server 协议构建的插件化 AI Agent 平台。项目提供用户对话端、管理后台、Skills 管理、app-server 配置管理、会话生命周期管理和 Token 用量指标。

## 功能特性

- 用户侧 Agent 对话页面：支持用户登录、Access Token/Refresh Token 续期、流式回答、过程状态展示、Mermaid 渲染、生成图片展示、归档、置顶、分组和会话状态恢复。
- jzero REST 服务端：负责认证、会话元数据、SSE 事件分发、Token 用量记录、Codex app-server 生命周期集成和插件能力。
- 插件化 Agent 能力：服务端可加载独立插件二进制和插件配置，插件可以提供 CLI 工具、Skills、迁移脚本和运行时能力；Codex 运行时会同步插件 Skills，并在 Agent 会话中按需调用。
- 管理后台：支持用户、角色、菜单、Agent Skills、Codex 配置文件、app-server 重启和首页 Agent 指标。
- Skills 管理：支持上传、浏览、编辑和删除 Codex Skills。
- Docker Compose 部署：包含 server、agent 用户端、web 管理端、PostgreSQL 和 Redis。

## 仓库结构

- `server`：jzero REST 服务端，包含 Codex app-server 生命周期管理、会话、SSE、迁移、插件加载和 Skills 同步。
- `agent`：面向用户的 Vue 3 对话应用，包含登录态维护、Token 续期、对话流式展示和会话操作。
- `web`：Vue 3 管理后台。
- `core-engine`：管理后台使用的共享基础引擎。
- `deploy/docker-compose`：Docker Compose 部署配置。
- `docs`：项目文档。
- `specs`：产品、功能和技术研究文档。

## 本地开发

### 服务端

```shell
cd server
go test ./...
go run . server -c etc/etc.yaml
```

主要运行配置位于 `server/etc/etc.yaml` 和配置的 Codex home 目录中。Codex app-server 会从 Codex home 中读取 `config.toml`、`models.json` 和 `auth.json`。

### Agent 用户端

```shell
cd agent
pnpm install
pnpm typecheck
pnpm dev
```

开发服务默认监听 `http://localhost:5174`，并将 `/api` 代理到 `http://localhost:8001`。

### 管理后台

```shell
cd web
pnpm install
pnpm typecheck
pnpm dev
```

## 部署

默认部署配置位于 `deploy/docker-compose`。

```shell
cd deploy/docker-compose
docker compose up -d --build
```

常用单服务部署命令：

```shell
docker compose build server && docker compose up -d --no-deps server
docker compose build agent && docker compose up -d --no-deps agent
docker compose build web && docker compose up -d --no-deps web
```

## 规格文档

`specs/` 用于沉淀规划和设计材料：

- `specs/features`：功能规格和实现说明。
- `specs/product`：产品需求和体验决策。
- `specs/research`：技术调研、协议分析和方案记录。

## 开源协议

AgentRazor 基于 [Apache-2.0 License](LICENSE) 开源。
