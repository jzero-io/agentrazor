---
title: Guide
icon: solar:book-2-linear
order: 1
---

# Guide

AgentRazor has two user-facing surfaces: the Agent chat UI and the admin console.

## Agent Chat UI

The `agent` application provides:

- User login and authenticated sessions.
- Access Token renewal through Refresh Token.
- Conversation create, switch, archive, delete, pin, and group operations.
- Real-time display of thinking, process items, tool calls, and final answers from Codex app-server.
- Mermaid rendering and source copying.
- Generated image and workspace entry display.

## Admin Console

The `web` application provides:

- User, role, menu, and API permission management.
- Agent Skills upload, browse, edit, and delete.
- Codex config file management for `config.toml`, `models.json`, and `auth.json`.
- app-server runtime status and restart.
- Homepage metrics for conversations, active tasks, archives, and token usage.
