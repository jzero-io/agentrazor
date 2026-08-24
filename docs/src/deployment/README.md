---
title: 部署
icon: solar:server-2-linear
order: 1
---

# 部署

推荐使用 `deploy/docker-compose` 部署完整环境。

```shell
cd deploy/docker-compose
docker compose up -d --build
```

## 服务组成

- `server`：jzero REST 服务端。
- `agent`：用户侧 Agent 页面。
- `web`：管理后台。
- `postgres`：主数据库。
- `redis`：缓存和辅助状态。

## 单服务部署

只更新服务端：

```shell
docker compose build server
docker compose up -d --no-deps server
```

只更新用户侧页面：

```shell
docker compose build agent
docker compose up -d --no-deps agent
```

只更新管理后台：

```shell
docker compose build web
docker compose up -d --no-deps web
```
