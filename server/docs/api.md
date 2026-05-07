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

用途：管理员创建组织账号、查看账号目录、启用/禁用用户以及重置组织用户密码。公开注册已关闭，普通组织用户只能用管理员创建的账号登录。

### `api`

- `summary`
- `key_list`
- `key_create`
- `key_set_disabled`
- `usage_list`
- `usage_buckets`
- `model_list`
- `model_upsert`
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
- `alert_rule_set_enabled`
- `alert_event_list`
- `alert_event_ack`
- `user_key_list`
- `user_usage_summary`
- `user_usage_list`

用途：管理员管理下游 API key、组织用户归属、key+model 策略、模型列表、模型价格、站内告警、usage 汇总、按天聚合和最近请求。创建 key 时，明文 key 只在 `key_create` 响应中返回一次，数据库仅保存 hash、prefix 和 last4。

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
- `/v1/chat/completions` 和 `/v1/responses` 转发前会检查 key 状态、模型权限、RPM、TPM、日/月请求配额和日/月 token 配额；超限返回 HTTP `429`
- token 计量以 OpenAI 响应里的实际 usage 为准，TPM 和 token 配额允许单次请求短暂越界，下一次请求开始拦截

## OAuth 登录入口

HTTP 路由：

- `GET /auth/oauth/config`：返回当前是否启用管理员 OAuth 登录，以及前端按钮展示名。
- `GET /auth/oauth/start?redirect=/admin-menu`：创建 state cookie 并跳转到配置的身份提供方授权页。
- `GET /auth/oauth/callback`：接收身份提供方回调，换取 userinfo 后匹配或绑定已有管理员账号，并跳转到前端 `/oauth/callback` 写入本系统管理员 JWT。

说明：

- 服务端不会把第三方 `access_token`、`refresh_token` 或 `id_token` 返回给前端。
- 前端最终只持有本系统签发的管理员 `access_token`。
- OAuth 不自动创建管理员；首次绑定要求 IdP 返回的 `email` 或 `preferred_username` 能匹配既有管理员账号。

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

返回后台账号目录所需的最小字段：

- `id`
- `username`
- `disabled`
- `created_at`
- `last_login_at`

### `api.key_create`

返回创建后的 key 元数据和一次性明文：

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

## 不再属于模板主干的业务能力

以下能力已经从当前项目默认主干移除，不应再假定存在：

- 积分
- 订阅
- 邀请码
- 管理员层级
- 任何行业特定业务域

如果后续业务需要这些能力，应按真实需求重新定义 schema、错误码、接口和前端消费层，而不是把旧模板逻辑直接加回主干。
