---
title: Development
icon: solar:code-square-linear
order: 1
---

# Development

## Server

```shell
cd server
go test ./...
go run . server -c etc/etc.yaml
```

The server follows jzero API description and code generation patterns. For new APIs, update `server/desc/api`, regenerate handlers/types/routes, then implement logic.

## Agent UI

```shell
cd agent
npm run build
```

Main state lives in `agent/src/App.vue`; process display mapping lives in `agent/src/processDisplay.ts`. When changing streaming behavior, verify refresh, conversation switching, and SSE reconnect paths.

## Admin Console

```shell
cd web
npm run build
```

When adding admin pages, keep menu data, route declarations, API permissions, and API types in sync.

## Generated Files

- Do not commit swagger JSON under `server/desc/swagger/`.
- Only maintain PostgreSQL migrations under `server/desc/sql_migration/pgx`.
- Use `.gitkeep` for empty directories.
