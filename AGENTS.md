# AGENTS.md

这个文件为在本仓库中工作的编码 Agent 提供项目上下文和协作规则。

## 项目概览

AgentRazor 主要由三个应用组成：

- `server`：Go + jzero REST 服务端，负责认证、会话元数据、Codex app-server 集成、SSE 事件分发、Token 用量记录、数据库迁移和插件能力。
- `agent`：面向用户的 Vue 3 对话界面。
- `web`：Vue 3 管理后台。

## 工作规则

- 服务端开发优先遵循现有 jzero 目录和代码生成模式，包括 `server/desc/api`、生成的 handler/types 和 logic 文件。
- 修改服务端 API 时，先更新 API 描述，再重新生成 jzero 代码，最后实现业务逻辑。
- PostgreSQL 迁移位于 `server/desc/sql_migration/pgx`，只维护 pgx 迁移集。
- 不提交生成的 swagger JSON；`server/desc/swagger/` 已加入忽略列表，并在构建时生成。
- Agent 用户端的会话状态主要在 `agent/src/App.vue` 中维护，过程展示映射在 `agent/src/processDisplay.ts` 中维护，需要和 Codex app-server 事件模型保持一致。
- 管理后台路由需要和菜单数据、elegant-router 生成声明保持一致。

## 验证方式

根据改动范围选择最小验证命令：

```shell
cd server && go test ./...
cd agent && npm run build
cd web && npm run build
```

## 部署方式

从 `deploy/docker-compose` 执行部署：

```shell
docker compose build server && docker compose up -d --no-deps server
docker compose build agent && docker compose up -d --no-deps agent
docker compose build web && docker compose up -d --no-deps web
```

## 规格文档

规划文档放在 `specs/` 下：

- `specs/features`
- `specs/product`
- `specs/research`
