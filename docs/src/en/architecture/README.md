---
title: Architecture
icon: solar:diagram-up-linear
order: 1
---

# Architecture

AgentRazor consists of the Agent UI, admin console, server, Codex app-server, plugin system, and infrastructure.

```mermaid
flowchart LR
  User[User] --> AgentUI[agent UI]
  Admin[Admin] --> AdminUI[web Admin]

  AgentUI -->|REST / SSE| Server[server jzero REST]
  AdminUI -->|REST| Server

  Server -->|RPC / JSONL| Codex[Codex app-server]
  Server -->|read / write| PG[(PostgreSQL)]
  Server -->|cache / auxiliary state| Redis[(Redis)]
  Server -->|sync Skills / call CLI| Plugins[Business Plugins]

  Codex --> CodexHome[(Codex Home)]
  CodexHome --> Config[config.toml / models.json / auth.json]
  CodexHome --> Skills[Skills]

  Plugins --> PluginCLI[Plugin CLI]
  Plugins --> PluginSkills[Plugin Skills]
  Plugins --> PluginMigrations[Plugin Migrations]
```

## Core Flow

1. A user logs in and sends a message in `agent`.
2. `server` validates identity and creates or resumes a Codex thread.
3. `server` starts a turn through Codex app-server.
4. Codex app-server streams process events, tool calls, message deltas, and completion events.
5. `server` forwards events through SSE and records necessary token usage and conversation state.
6. `agent` merges events into stable process display and final answers.
