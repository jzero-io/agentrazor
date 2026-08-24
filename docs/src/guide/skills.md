---
title: Skills 管理
icon: solar:document-add-linear
order: 3
---

# Skills 管理

Skills 是 AgentRazor 扩展 Agent 行为的重要方式。一个 Skill 通常包含 `SKILL.md` 和相关参考文件，Codex 会根据 Skill 描述在合适场景中使用它。

## 管理能力

管理后台支持：

- 上传 zip 格式 Skill。
- 查看已安装 Skills。
- 浏览 Skill 文件树。
- 查看和编辑 Skill 文件内容。
- 删除 Skill。

## 插件 Skills

插件也可以携带 Skills。服务端启动或运行时同步时，会把插件中的 Skills 同步到 Codex home 的 Skills 目录，供 Codex app-server 使用。

## 设计原则

- 用户上传的是 zip，最终安装位置由系统决定。
- `.system` 内置 Skills 不展示给普通管理操作。
- 文件树展示真实文件结构，便于维护参考文档。
