---
title: Skills
icon: solar:document-add-linear
order: 3
---

# Skills

Skills extend Agent behavior. A Skill usually contains `SKILL.md` and related reference files. Codex uses Skills when their descriptions match the task.

## Management

The admin console supports:

- Uploading Skill zip files.
- Viewing installed Skills.
- Browsing the Skill file tree.
- Viewing and editing Skill files.
- Deleting Skills.

## Plugin Skills

Plugins may also ship Skills. The server syncs plugin Skills into Codex home so Codex app-server can use them.

## Design Principles

- Users upload zip files; the system decides the installation location.
- Built-in `.system` Skills are not exposed for regular management.
- The file tree shows the real file structure for maintainability.
