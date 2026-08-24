---
title: 配置管理
icon: solar:settings-linear
order: 2
---

# 配置管理

AgentRazor 的运行配置分为服务端配置和 Codex 配置两类。

## 服务端配置

服务端配置位于 `server/etc/etc.yaml`。它负责数据库、Redis、鉴权、Agent runtime、Codex home、插件目录等基础运行参数。

常见配置项包括：

- Codex 二进制路径。
- Codex home 路径。
- Agent 工作目录。
- app-server 是否启用。
- 插件目录和插件同步策略。

## Codex 配置文件

Codex 配置位于配置的 Codex home 目录，管理后台支持直接编辑：

- `config.toml`：Codex app-server 主配置。
- `models.json`：模型和 provider 配置，可以为空。
- `auth.json`：ChatGPT 登录态或认证信息，可以为空。

保存配置后，如果需要让 app-server 重新读取配置，可以在管理后台手动重启 app-server。

## 模型切换

模型切换不单独做按钮，而是直接暴露底层配置文件。管理员可以修改 `config.toml` 和 `models.json`，再重启 app-server。这样 ChatGPT、DeepSeek 或后续 provider 都能按同一套机制接入。
