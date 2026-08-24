---
home: true
icon: solar:home-2-linear
title: 首页
heroText: AgentRazor
heroFullScreen: false
tagline: 插件化 AI Agent 平台，连接 Codex app-server、业务插件、Skills 和可运营的会话系统。
actions:
  - text: 快速开始
    icon: solar:rocket-linear
    link: /guide/
    type: primary
  - text: 查看架构
    icon: solar:diagram-up-linear
    link: /architecture/
features:
  - title: 用户侧 Agent 对话
    icon: solar:chat-round-line-linear
    details: 支持登录、Token 续期、流式回答、过程状态、Mermaid、图片展示、归档、置顶和分组。
  - title: Codex app-server 集成
    icon: solar:server-2-linear
    details: 服务端托管 app-server 生命周期，统一处理会话、SSE、错误、Token 用量和运行状态。
  - title: 插件化能力
    icon: solar:plug-circle-linear
    details: 插件可提供 CLI、Skills、迁移和运行时能力，让 Agent 能面向业务场景扩展。
  - title: 管理后台
    icon: solar:settings-linear
    details: 提供用户、角色、菜单、Skills、配置文件、运行时状态和指标管理。
---

## AgentRazor 是什么

AgentRazor 是一个围绕 Codex app-server 协议构建的 AI Agent 平台。它不是单一聊天页面，而是一套完整的 Agent 运行和运营体系：用户在 `agent` 中发起对话，`server` 负责鉴权、会话元数据、事件流和 app-server 调度，业务插件和 Skills 为 Agent 提供领域能力，`web` 管理后台负责配置、权限和运营管理。

## 适合的场景

- 希望把 Codex app-server 接入企业内部系统。
- 希望统一管理 Agent 会话、用户权限、Skills、模型配置和运行状态。
- 希望通过插件把业务 CLI、领域知识、迁移脚本和运行时能力接入 Agent。
- 希望同时拥有用户侧对话体验和管理后台。
