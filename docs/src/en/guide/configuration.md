---
title: Configuration
icon: solar:settings-linear
order: 2
---

# Configuration

AgentRazor has server configuration and Codex configuration.

## Server Configuration

Server configuration lives in `server/etc/etc.yaml`. It controls database, Redis, auth, Agent runtime, Codex home, plugin directories, and other runtime parameters.

## Codex Config Files

The admin console can edit files in Codex home:

- `config.toml`: main Codex app-server configuration.
- `models.json`: model and provider configuration. It can be empty.
- `auth.json`: ChatGPT login state or authentication information. It can be empty.

After changing these files, restart app-server from the admin console when the runtime needs to reload configuration.

## Model Switching

Model switching is handled by editing the underlying config files instead of a separate switch button. Administrators update `config.toml` and `models.json`, then restart app-server. This keeps ChatGPT, DeepSeek, and future providers under one mechanism.
