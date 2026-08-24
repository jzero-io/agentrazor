---
title: 开发指南
icon: solar:code-square-linear
order: 1
---

# 开发指南

## 本地启动

本地开发统一使用 Docker Compose，从部署目录启动完整环境：

```shell
cd deploy/docker-compose
docker compose up -d --build
```

这样可以保证 server、agent、web、PostgreSQL、Redis、Codex home 和运行配置保持一致。

## 单服务重建

修改某个服务后，可以只重建并重启对应容器：

```shell
docker compose build server && docker compose up -d --no-deps server
docker compose build agent && docker compose up -d --no-deps agent
docker compose build web && docker compose up -d --no-deps web
```

服务端运行配置、数据和日志位于 `deploy/docker-compose/server` 下。Codex app-server 会从服务端数据目录中的 Codex home 读取 `config.toml`、`models.json` 和 `auth.json`。

## 服务端开发

服务端遵循 jzero 的 API 描述和代码生成模式。新增接口时，先修改 `server/desc/api`，再生成 handler、types 和 route，最后补 logic。

## Agent 用户端开发

主要代码在 `agent/src/App.vue`，过程展示抽象在 `agent/src/processDisplay.ts`。修改会话流式展示时，要特别注意刷新、切换会话和 SSE 断线重连。

## 管理后台开发

新增管理后台页面时，需要同步菜单数据、路由声明、接口权限和 API 类型。

## 验证命令

提交前按改动范围选择验证命令：

```shell
cd server && go test ./...
cd agent && npm run build
cd web && npm run build
```

## 生成文件约定

- 不提交 `server/desc/swagger/` 下的 swagger JSON。
- PostgreSQL 迁移只维护 `server/desc/sql_migration/pgx`。
- 空目录需要使用 `.gitkeep` 占位。
