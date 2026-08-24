---
title: Deployment
icon: solar:server-2-linear
order: 1
---

# Deployment

Use `deploy/docker-compose` for the default deployment.

```shell
cd deploy/docker-compose
docker compose up -d --build
```

## Services

- `server`: jzero REST server.
- `agent`: Agent chat UI.
- `web`: admin console.
- `postgres`: primary database.
- `redis`: cache and auxiliary state.

## Service-only Deployment

Server only:

```shell
docker compose build server
docker compose up -d --no-deps server
```

Agent UI only:

```shell
docker compose build agent
docker compose up -d --no-deps agent
```

Admin console only:

```shell
docker compose build web
docker compose up -d --no-deps web
```
