---
title: 事件流
icon: solar:bolt-linear
order: 2
---

# 事件流

AgentRazor 使用 SSE 将服务端事件实时推送到用户侧页面。事件分为两类：

- `run.*`：AgentRazor 服务端生命周期事件，例如 `run.started`、`run.completed`、`run.failed`。
- `codex.*`：Codex app-server 原始事件的转发，例如 `codex.turn.started`、`codex.item.started`、`codex.item.agentMessage.delta`、`codex.item.completed`。

## 展示原则

用户侧页面不会简单堆叠所有事件，而是按 turn 合并为三类展示：

- 正在思考：turn 已开始，但还没有可展示的过程项或最终答案。
- 正在处理：出现命令、Skills、网页搜索、工具调用等过程项后展示耗时和过程列表。
- 已处理：turn 完成后折叠过程项，并展示最终答案或错误。

## 错误处理

当 app-server 返回错误，例如模型未配置或认证失败，服务端会发布 `run.failed`。用户侧页面会立即把错误合并到当前 turn，而不是等刷新后再从历史详情中读取。
