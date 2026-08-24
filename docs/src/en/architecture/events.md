---
title: Event Stream
icon: solar:bolt-linear
order: 2
---

# Event Stream

AgentRazor uses SSE to push server events to the Agent UI. Events are grouped into two categories:

- `run.*`: AgentRazor lifecycle events such as `run.started`, `run.completed`, and `run.failed`.
- `codex.*`: forwarded Codex app-server events such as `codex.turn.started`, `codex.item.started`, `codex.item.agentMessage.delta`, and `codex.item.completed`.

## Display Rules

The Agent UI does not simply append every event. It merges events into turn-level states:

- Thinking: a turn has started, but no visible process item or final answer exists yet.
- Processing: commands, Skills, web search, tool calls, or other process items are visible.
- Completed: the turn has finished, process items are collapsed, and the final answer or error is displayed.

## Error Handling

When app-server returns an error, such as missing model configuration or auth failure, the server publishes `run.failed`. The Agent UI merges the error into the current turn immediately instead of waiting for a page refresh.
