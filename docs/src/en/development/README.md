---
title: Development
icon: solar:code-square-linear
order: 1
---

# Development

## Local Startup

Use Docker Compose for local development so the server, UIs, PostgreSQL, Redis, Codex home, and runtime configuration stay aligned.

```shell
cd deploy/docker-compose
docker compose up -d --build
```

## Rebuild One Service

After changing one service, rebuild and restart only that container:

```shell
docker compose build server && docker compose up -d --no-deps server
docker compose build agent && docker compose up -d --no-deps agent
docker compose build web && docker compose up -d --no-deps web
```

Server configuration, data, and logs live under `deploy/docker-compose/server`. Codex app-server reads `config.toml`, `models.json`, and `auth.json` from the Codex home in the server data directory.

## Server

The server follows jzero API description and code generation patterns. For new APIs, update `server/desc/api`, regenerate handlers/types/routes, then implement logic.

## Agent UI

Main state lives in `agent/src/App.vue`; process display mapping lives in `agent/src/processDisplay.ts`. When changing streaming behavior, verify refresh, conversation switching, and SSE reconnect paths.

## Admin Console

When adding admin pages, keep menu data, route declarations, API permissions, and API types in sync.

## Verification

Run the smallest relevant checks before submitting changes:

```shell
cd server && go test ./...
cd agent && npm run build
cd web && npm run build
```

## Generated Files

- Do not commit swagger JSON under `server/desc/swagger/`.
- Only maintain PostgreSQL migrations under `server/desc/sql_migration/pgx`.
- Use `.gitkeep` for empty directories.
