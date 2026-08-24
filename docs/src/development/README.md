---
title: 开发指南
icon: solar:code-square-linear
order: 1
---

# 开发指南

## 服务端开发

```shell
cd server
go test ./...
go run . server -c etc/etc.yaml
```

服务端遵循 jzero 的 API 描述和代码生成模式。新增接口时，先修改 `server/desc/api`，再生成 handler、types 和 route，最后补 logic。提交前需要根据改动范围选择对应测试。

## Agent 用户端开发

```shell
cd agent
npm run build
```

主要代码在 `agent/src/App.vue`，过程展示抽象在 `agent/src/processDisplay.ts`。修改会话流式展示时，要特别注意刷新、切换会话和 SSE 断线重连。

## 管理后台开发

```shell
cd web
npm run build
```

新增管理后台页面时，需要同步菜单数据、路由声明、接口权限和 API 类型。

## 生成文件约定

- 不提交 `server/desc/swagger/` 下的 swagger JSON。
- PostgreSQL 迁移只维护 `server/desc/sql_migration/pgx`。
- 空目录需要使用 `.gitkeep` 占位。
