---
title: 插件化能力
icon: solar:plug-circle-linear
order: 3
---

# 插件化能力

插件是 AgentRazor 面向业务扩展的核心机制。一个插件可以提供可执行文件、配置、Skills、迁移脚本和领域能力。

## 插件可以提供什么

- CLI 工具：让 Codex 在会话中执行确定性的业务命令。
- Skills：向 Codex 注入领域知识、工作流程和约束。
- 数据库迁移：为插件自己的数据结构提供初始化和升级能力。
- 运行时能力：通过服务端集成，把插件能力暴露给 Agent 或管理后台。

## 与 Codex 的关系

Codex app-server 负责模型推理和工具执行，AgentRazor 负责把插件能力放到 Codex 能使用的位置：

- 插件 CLI 被构建到容器并加入可执行路径。
- 插件 Skills 同步到 Codex home。
- 插件配置随服务端部署和运行时读取。

## 前端页面

插件可以携带两个独立构建的静态微前端；它们由插件自身的
`serverless.HandlerFunc` 注册，并随服务端二进制一起构建：

- 工作台：`/plugins/<plugin_id>/workbench/`，由 Agent 会话中的
  `workspace` 项在右侧 iframe 中打开。
- 后台配置：`/plugins/<plugin_id>/admin/`，由管理后台的
  `view.plugin-management` 在既有后台布局的内容区 iframe 中打开。

例如 crypto-tracing 的工作台入口为
`/plugins/crypto_tracing/workbench/`。其 serverless 模块把
`assets/dist/index.html` 和静态资源嵌入服务端，并注册该路径。

管理后台和 Agent Nginx 都会把 `/plugins/` 反向代理到 server；插件被构建进
镜像时，`jzero serverless build` 会发现并加载其 serverless 模块，未构建的
插件没有静态入口、菜单或接口。

`插件管理` 是普通的后台一级菜单组；每个插件是它的一个二级菜单，结构与
`系统管理 → 用户管理` 一致。二级菜单使用内嵌页面组件，因此点击菜单只在
既有 Admin 基础布局中切换路由，并在内容区嵌入插件静态页，不会打开外部页面。
iframe 不接触登录令牌：它通过受限的 `postMessage` 请求宿主，宿主再用核心 Admin
的统一请求层调用 `/api/v1/manage/plugin/<plugin_id>/...`。

简单插件可直接在 `插件管理` 下登记一个 `menu_type = 2` 的页面。若插件有多个
后台功能，则插件自身是 `menu_type = 1` 目录，下面再放各个 `menu_type = 2` 页面，
例如“运营管理”和“配置管理”；页面继续使用 `view.iframe-page`，因此仍承载在
`layout.base` 内。

页面的 `href` 指向插件静态入口（可用 `?page=` 选择插件内部子页），操作码则声明
为该页面的 `menu_type = 3` 按钮。这样角色菜单授权、子菜单可见性和服务端 Casbin
校验都以同一份菜单权限为准。密钥类接口必须分别校验读取和保存权限，采用加密持久化
并只返回“已配置”状态，不能回传明文。

## 适合沉淀到插件的内容

- 某个业务域的命令行工具。
- 某类项目的代码生成规范。
- 内部平台 API 的操作手册。
- 固定的数据分析或诊断流程。
