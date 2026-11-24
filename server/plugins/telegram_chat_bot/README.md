# Telegram 群聊问答机器人

该插件把 Telegram 群聊消息接入 AgentRazor，只处理配置群聊中明确
`@机器人` 的文本消息。

## 行为

- 支持 Telegram Bot API 长轮询，不需要公网 Webhook。
- 仅接受群聊或超级群，并严格校验群 ID allowlist。
- 只有消息实体中存在当前 Bot 的 `@username` 时才触发问答。
- 每个 Telegram 群映射到一个独立 AgentRazor 会话。
- 回复按 Telegram 4096 UTF-16 单位限制自动拆分。
- 不调用 Lark，不创建人工接管状态，不执行超时转人工。
- Bot Token 与 AgentRazor API Key 加密保存在 PostgreSQL，不写入日志或静态配置。

## 后台配置

在 AgentRazor 管理后台的“插件管理 → Telegram 问答机器人”中配置：

- Telegram Bot Token
- 允许的群聊 ID，例如 `-1001234567890`
- AgentRazor 服务地址和 API Key
- 长轮询、请求和回答超时

保存后会立即重载 Telegram 网关。

## 验证

```shell
go test ./...
go vet ./...
```
