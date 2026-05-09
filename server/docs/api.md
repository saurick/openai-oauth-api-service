# JSON-RPC API 说明

当前项目保留 JSON-RPC 入口承载管理后台能力，并提供 OpenAI 兼容 `/v1/*` HTTP 转发入口。

## 统一入口

协议定义见：

- `server/api/jsonrpc/v1/jsonrpc.proto`

HTTP 路由：

- `GET /rpc/{url}`
- `POST /rpc/{url}`

其中：

- `{url}` 表示业务域，例如 `system`、`auth`、`user`、`api`
- `method` 表示具体动作，例如 `login`、`me`、`list`

## 当前默认保留的业务域

### `system`

- `ping`
- `version`

用途：无鉴权的基础联通性检查。

### `auth`

- `admin_login`
- `logout`
- `me`

用途：管理员登录、退出和当前登录态查询。普通用户 `/login` 与 `/register` 前端入口已收口到管理员登录。

### `user`

- `create`
- `list`
- `reset_password`
- `set_disabled`

用途：历史组织用户兼容能力。当前前端主路径不再展示账号目录，默认只通过管理员登录进入后台；如后续彻底移除，需要同步评估 `users` 表、key 归属字段和历史数据。

### `api`

- `summary`
- `key_list`
- `key_create`
- `key_update`
- `key_delete`
- `key_set_disabled`
- `usage_list`
- `usage_buckets`
- `usage_key_summaries`
- `model_list`
- `model_upsert`
- `model_delete`
- `model_set_enabled`
- `model_sync`
- `policy_list`
- `policy_upsert`
- `policy_delete`
- `price_list`
- `price_upsert`
- `price_delete`
- `alert_rule_list`
- `alert_rule_upsert`
- `alert_rule_delete`
- `alert_rule_set_enabled`
- `alert_event_list`
- `alert_event_ack`
- `user_key_list`
- `user_usage_summary`
- `user_usage_list`

用途：管理员管理下游 API key、组织用户归属、key+model 策略、模型列表、模型价格、站内告警、usage 汇总、按天聚合、按 key 聚合和最近请求。创建 key 时会随机生成完整 key；数据库保存 `plain_key` 用于后台展示，同时保存 `key_hash` 用于鉴权匹配，`key_prefix` 和 `key_last4` 用于 usage 归属与人工识别。

普通组织用户只允许调用 `api.user_key_list`、`api.user_usage_summary` 和 `api.user_usage_list`，并且后端按当前登录用户过滤 `owner_user_id`，不返回其他用户 key。

HTTP 管理导出：

- `GET /admin/exports/usage.csv`
- `GET /admin/exports/usage.json`

导出要求管理员登录态，筛选条件与 `api.usage_list` 保持一致：时间范围、key、模型、endpoint、success。导出会写审计日志。

## OpenAI 兼容入口

HTTP 路由：

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`

鉴权：

- 下游请求必须使用 `Authorization: Bearer ogw_...`
- 上游请求由服务端使用 `OPENAI_API_KEY` / `data.openai.apiKey` 注入
- 统一上游代理由 `UPSTREAM_PROXY_URL` / `data.openai.upstreamProxyUrl` 控制

usage 记录：

- 成功和失败请求都会记录 usage log
- 默认记录 endpoint、model、HTTP 状态、耗时、请求/响应字节数和 token usage
- 非流式 JSON 和 SSE completed event 都会尝试解析 usage
- 默认不保存 prompt、response body 或正文采样
- `/v1/chat/completions` 和 `/v1/responses` 转发前会检查 key 状态、模型权限、key 级 token 总额度、RPM、TPM、日/月请求配额和日/月 token 配额；超限返回 HTTP `429`
- token 计量以 OpenAI 响应里的实际 usage 为准，key 级 token 总额度、TPM 和 token 配额允许单次请求短暂越界，下一次请求开始拦截

## 管理员 OAuth 入口

管理员 OAuth 登录默认关闭，配置完整后开放以下 HTTP 路由：

- `GET /auth/oauth/config`：返回 `{ enabled, provider }`，前端据此决定是否显示 OAuth 登录按钮。
- `GET /auth/oauth/start`：发起授权。可传 `frontend_origin` 和 `next`，服务端会把当前前端 origin 和回跳路径写入 signed state。
- `GET /auth/oauth/callback`：OAuth provider 固定回调后端的地址。服务端完成 code exchange 和 userinfo 查询后，签发本系统管理员 JWT，并通过前端 `/oauth/callback` 的 URL fragment 回传登录态。

本地 Google Console 回调登记 `http://localhost:8400/auth/oauth/callback`，不登记 Vite 端口。生产环境登记后端 HTTPS 域名下的同一路径，并通过 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS` allowlist 前端后台 origin。

## 鉴权规则

- `system.*` 默认是公开方法
- 其他业务域默认要求已登录
- `user.*` 额外要求管理员登录态
- `api.*` 默认要求管理员登录态；`api.user_*` 只要求普通组织用户登录态，并按当前用户过滤数据

说明：管理员鉴权依赖 token 里的角色信息，而不是前端页面路径。

## 默认返回结构

所有 JSON-RPC 响应统一返回：

- `jsonrpc`
- `id`
- `result.code`
- `result.message`
- `result.data`
- `error`

其中：

- `result.code=0` 表示成功
- 其他错误码统一来源于 `server/internal/errcode/catalog.go`

## 当前项目默认保留的数据字段

### `auth.login` / `auth.admin_login`

返回最小登录态信息：

- `user_id`
- `username`
- `access_token`
- `expires_at`
- `token_type`
- `issued_at`

### `auth.register`

公开注册已关闭。该方法保留兼容入口，但会返回业务错误并提示联系管理员创建组织账号。

### `auth.me`

返回当前用户或当前管理员的最小信息，用于前端恢复登录态。

### `user.list`

返回历史组织用户列表所需的最小字段：

- `id`
- `username`
- `disabled`
- `created_at`
- `last_login_at`

### `api.key_create`

返回创建后的 key 元数据和完整明文：

- `id`
- `name`
- `key_prefix`
- `key_last4`
- `plain_key`
- `allowed_models`
- `quota_requests`
- `quota_total_tokens`
- `disabled`
- `owner_user_id`

### `api.usage_list`

返回最近 usage 列表和汇总：

- `items`
- `total`
- `summary.total_requests`
- `summary.success_requests`
- `summary.failed_requests`
- `summary.total_tokens`
- `summary.average_duration_ms`
- `summary.estimated_cost_usd`

### `api.usage_buckets`

返回 usage 按天聚合，用于后台趋势图和按天统计表：

- `items`
- `group_by`
- `items[].bucket_start`
- `items[].total_requests`
- `items[].success_requests`
- `items[].failed_requests`
- `items[].input_tokens`
- `items[].cached_tokens`
- `items[].output_tokens`
- `items[].reasoning_tokens`
- `items[].total_tokens`
- `items[].estimated_cost_usd`

当前只支持 `group_by=day`。费用估算从模型价格表读取本地数据库价格真源；价格缺失时返回 `null`，前端显示“未配置价格”。reasoning tokens 作为输出 tokens 子集展示，不重复计费。

### `api.usage_key_summaries`

返回 usage 按下游 key 聚合，用于后台查看每个 key 的消耗：

- `items`
- `items[].api_key_id`
- `items[].api_key_prefix`
- `items[].api_key_name`
- `items[].disabled`
- `items[].total_requests`
- `items[].success_requests`
- `items[].failed_requests`
- `items[].input_tokens`
- `items[].cached_tokens`
- `items[].output_tokens`
- `items[].reasoning_tokens`
- `items[].total_tokens`
- `items[].average_duration_ms`
- `items[].estimated_cost_usd`

筛选条件与 `api.usage_list` 保持一致。费用估算仍以本地模型价格表为真源；窗口内存在未配置价格的模型时，对应 key 的 `estimated_cost_usd` 返回 `null`。

## 不再属于模板主干的业务能力

以下能力已经从当前项目默认主干移除，不应再假定存在：

- 积分
- 订阅
- 邀请码
- 管理员层级
- 任何行业特定业务域

如果后续业务需要这些能力，应按真实需求重新定义 schema、错误码、接口和前端消费层，而不是把旧模板逻辑直接加回主干。
