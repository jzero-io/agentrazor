---
title: Plugins
icon: solar:plug-circle-linear
order: 3
---

# Plugins

Plugins are the core business extension mechanism in AgentRazor. A plugin can provide executables, configuration, Skills, migrations, and domain capabilities.

## What Plugins Provide

- CLI tools for deterministic business commands.
- Skills for domain knowledge, workflows, and constraints.
- Database migrations for plugin-owned data structures.
- Runtime capabilities exposed through the server to Agent or admin features.

## Relationship with Codex

Codex app-server handles model reasoning and tool execution. AgentRazor makes plugin capabilities available to Codex:

- Plugin CLI binaries are built into the container and exposed in PATH.
- Plugin Skills are synced into Codex home.
- Plugin configuration is deployed and read by the server runtime.
