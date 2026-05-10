# 架构说明

## 目标

本项目的长期目标是提供一个生产可维护的 OpenAI 兼容 API 转发与 token/usage 统计服务：

- 服务端集中管理上游连接配置。
- 管理员通过后台账号登录后使用本系统签发的 JWT；可选启用 Google/OIDC 管理员授权登录。
- 下游调用方使用本系统签发的 API key。
- API 转发链路统一做鉴权、配额、限流、转发、usage 记录和审计。
- 管理后台用于查看 key、用量、错误、延迟和运行状态。
- 上游出口可统一走代理，便于控制网络路径。

当前对外管理路径使用 `/admin-api` 和 `/rpc/api`。数据库表名和部分内部 Go 类型仍沿用早期模块名，后续如需彻底清理应单独走 Ent/Atlas migration，避免破坏现有数据。

## 模块划分

| 模块 | 职责 |
| --- | --- |
| `web/` | 管理后台，承载后台登录、key 管理、模型管理、usage 看板和运行状态 |
| `server/internal/server` | HTTP / JSON-RPC 接入、健康检查、中间件和观测包装 |
| `server/internal/service` | 请求 DTO 转换与接口编排 |
| `server/internal/biz` | 管理员登录、下游 key、配额、usage、API 转发策略等业务规则 |
| `server/internal/data` | PostgreSQL、usage 写入与查询、模型缓存和管理端查询 |
| `server/deploy/compose/prod` | 当前部署主路径 |
| `legacy-python-mvp/` | FastAPI + SQLite MVP 参考实现 |

## 数据口径

默认不保存请求体和响应体正文。usage 记录优先保存：

- 下游 key id / key 前缀
- endpoint、method、model
- 上游状态码、错误类型
- 可选 session_id，用于客户端显式传入会话标识后的会话聚合
- 请求字节数、响应字节数
- input / output / total tokens
- 延迟、创建时间

如后续需要排障采样，应单独设计脱敏采样策略、TTL 和开关，不能把 prompt / output 正文默认落库。

## 演进路线

1. 已完成长期仓库初始化：模板收口、Compose 主路径、文档和质量门禁。
2. 已迁入 Go 后端主路径：管理员登录、下游 key、模型缓存、usage log、OpenAI 兼容转发和 Codex CLI 统一上游出口。
3. 已补管理后台主入口：key 创建/启停、模型启停、usage 汇总和最近请求。
4. 管理员授权登录入口作为可选能力保留：Google/OIDC 回调固定走后端 `/auth/oauth/callback`，前端 origin 通过 signed state 动态回跳，避免本地 Vite 端口顺延后反复修改 OAuth Client 回调。
5. 账号目录和普通用户门户不作为当前前端主路径；后端历史 `users` 相关能力仅作为兼容实现保留，后续如需彻底移除需单独评估数据迁移与 key 归属字段。
6. 已增加 key+model 细粒度策略、站内告警、审计日志、usage CSV/JSON 导出和本地模型价格表费用估算。
