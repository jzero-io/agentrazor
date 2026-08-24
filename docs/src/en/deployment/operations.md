---
title: Operations
icon: solar:health-linear
order: 2
---

# Operations

## Logs and Data Directories

Docker Compose maps server data, logs, and configuration into the deployment directory for troubleshooting and persistence.

Common directories:

- `deploy/docker-compose/server/data`: server data and Codex home.
- `deploy/docker-compose/server/logs`: server logs.
- `deploy/docker-compose/server/etc`: deployment configuration.

## app-server Restart

The admin console can view app-server status and restart it. Before restarting, the server checks active tasks. If tasks are running, restart is rejected to avoid interrupting conversations.

## Config Changes

After changing `config.toml`, `models.json`, or `auth.json`, restart app-server manually to reload configuration.
