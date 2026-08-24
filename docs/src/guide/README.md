---
title: 使用指南
icon: solar:book-2-linear
order: 1
---

# 使用指南

AgentRazor 分为用户侧 Agent 页面和管理后台两部分。

## 用户侧 Agent 页面

用户侧页面位于 `agent` 应用，核心能力包括：

- 用户登录和会话认证。
- Access Token 过期后通过 Refresh Token 自动续期。
- 创建、切换、归档、删除、置顶和分组对话。
- 实时展示 Codex app-server 返回的思考、过程项、工具调用和最终回答。
- 支持 Mermaid 图表渲染和源码复制。
- 支持生成图片和工作台入口展示。

## 管理后台

管理后台位于 `web` 应用，面向系统管理员：

- 用户、角色、菜单和接口权限管理。
- Agent Skills 上传、浏览、编辑和删除。
- Codex 配置文件管理，包括 `config.toml`、`models.json` 和 `auth.json`。
- app-server 运行状态查看和重启。
- 首页 Agent 指标，包括会话数量、活跃任务、归档对话和 Token 消耗。

## 会话状态

服务端保存必要的会话元数据，并通过 Codex app-server 读取线程内容。运行中的 turn 通过 SSE 实时推送，前端负责把事件合并成稳定的对话展示。
