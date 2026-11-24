# BlockRazor 服务目录

该目录用于纠正已确认的产品事实，并非全量产品清单。未收录的产品或能力不能据此判定为不存在。

| 链 | 服务 | 已知情况 |
| --- | --- | --- |
| Ethereum | Private Mempool | BlockRazor 目前不提供 Ethereum Private Mempool。 |
| Solana | 交易发送 | 提供 Solana transaction sending 服务。 |
| Solana | Send Bundle | 提供 Solana Send Bundle；开通资格和权限仍需以官方文档及实际账户为准。 |
| Robinhood Chain | Sequencer Feed | 提供 Robinhood Sequencer Feed。 |
| Robinhood Chain | Transaction Sending RPC | 可通过官方 Portal 自助订阅。 |
| Fast Submit | 订阅服务 | 官方端点为 `http://fast-submit.blockrazor.io/`，保留 HTTP 协议，不改写为 HTTPS。 |
| Gas Sponsor | 服务 | 支持范围、开放对象和接入条件以官方资料为准，不从名称推断。 |

## 已确认边界

- Private Mempool 不包含发往 builder 的任何 bundle 或隐私交易。
- 服务目录缺项、搜索无结果或抓取失败，都不能证明产品不存在。
- 接口能力、参数、认证、地域覆盖和计费必须查看对应官方文档。
- 用户拥有某个服务的密钥不自动证明其拥有其他服务权限。
