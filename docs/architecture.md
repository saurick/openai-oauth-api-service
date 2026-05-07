# 架构说明

## 目标

本项目的长期目标是提供一个生产可维护的 OAuth 登录、OpenAI 兼容 API 转发与 token/usage 统计服务：

- 服务端集中持有官方 OpenAI API key。
- 管理员通过合规 OAuth/OIDC SSO 或后台账号登录后使用本系统签发的 JWT。
- 下游调用方使用本系统签发的 API key。
- API 转发链路统一做鉴权、配额、限流、转发、usage 记录和审计。
- 管理后台用于查看 key、用量、错误、延迟和运行状态。
- 上游出口可统一走代理，便于控制网络路径。

当前对外管理路径使用 `/admin-api` 和 `/rpc/api`。数据库表名和部分内部 Go 类型仍沿用早期模块名，后续如需彻底清理应单独走 Ent/Atlas migration，避免破坏现有数据。

## 合规边界

系统只接入 OpenAI 官方 API。禁止把 Codex / ChatGPT 客户端登录态、Cookie、设备码、个人账号 token 或浏览器会话包装成可共享 API。

## 模块划分

| 模块 | 职责 |
| --- | --- |
| `web/` | 管理后台，承载后台登录、账号管理、OAuth 配置、key 管理、模型管理、usage 看板和运行状态 |
| `server/internal/server` | HTTP / JSON-RPC 接入、健康检查、中间件和观测包装 |
| `server/internal/service` | 请求 DTO 转换与接口编排 |
| `server/internal/biz` | OAuth 登录、下游 key、配额、usage、API 转发策略等业务规则 |
| `server/internal/data` | PostgreSQL、usage 写入与查询、模型缓存和管理端查询 |
| `server/deploy/compose/prod` | 当前部署主路径 |
| `legacy-python-mvp/` | FastAPI + SQLite MVP 参考实现 |

## 数据口径

默认不保存请求体和响应体正文。usage 记录优先保存：

- 下游 key id / key 前缀
- endpoint、method、model
- 上游状态码、错误类型
- 请求字节数、响应字节数
- input / output / total tokens
- 延迟、创建时间

如后续需要排障采样，应单独设计脱敏采样策略、TTL 和开关，不能把 prompt / output 正文默认落库。

## 演进路线

1. 已完成长期仓库初始化：模板收口、Compose 主路径、文档和质量门禁。
2. 已迁入 Go 后端主路径：OAuth 登录、下游 key、模型缓存、usage log、OpenAI 兼容转发和统一上游代理。
3. 已补管理后台主入口：key 创建/启停、模型启停、usage 汇总和最近请求。
4. 已补管理员授权登录能力：仅接入合规 OAuth/OIDC SSO 或后台账号，不接入 Codex / ChatGPT 登录态。
5. 已接入组织账号主路径：复用 `users` 作为组织用户真源，关闭公开注册，管理员创建/禁用/重置用户密码，key 可绑定 `owner_user_id`。
6. 已增加 key+model 细粒度策略、站内告警、审计日志、usage CSV/JSON 导出、上游模型同步和本地模型价格表费用估算。
