---
title: 运维说明
icon: solar:health-linear
order: 2
---

# 运维说明

## 日志和数据目录

Docker Compose 部署会把服务端数据、日志和配置映射到部署目录下，便于排查问题和持久化运行状态。

常见目录：

- `deploy/docker-compose/server/data`：服务端数据和 Codex home。
- `deploy/docker-compose/server/logs`：服务端日志。
- `deploy/docker-compose/server/etc`：部署配置。

## app-server 重启

管理后台支持查看 app-server 状态并重启。重启前服务端会检查活跃任务数量；如果存在正在运行的任务，会拒绝重启，避免中断用户会话。

## 配置变更

修改 `config.toml`、`models.json` 或 `auth.json` 后，需要手动重启 app-server 才能让新配置生效。
