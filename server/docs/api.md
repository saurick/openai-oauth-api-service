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
- `gateway_upstream_get`
- `gateway_upstream_set`
- `usage_list`
- `usage_buckets`
- `usage_key_summaries`
- `usage_session_summaries`
- `model_list`
- `official_model_price_list`
- `model_set_enabled`
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

用途：管理员管理下游 API key、组织用户归属、key+model 策略、固定官方模型列表启停、模型价格、站内告警、usage 汇总、按天聚合、按 key 聚合和最近请求。创建 key 时会随机生成完整 key；数据库保存 `plain_key` 用于后台展示，同时保存 `key_hash` 用于鉴权匹配，`key_prefix` 和 `key_last4` 用于 usage 归属与人工识别。

模型目录以服务端代码中的官方 Codex 列表为真源，管理端只允许读取和启停；`model_upsert` / `model_delete` 不作为正式管理接口开放。

普通组织用户只允许调用 `api.user_key_list`、`api.user_usage_summary` 和 `api.user_usage_list`，并且后端按当前登录用户过滤 `owner_user_id`，不返回其他用户 key。

HTTP 管理导出：

- `GET /admin/exports/usage.csv`
- `GET /admin/exports/usage.json`

导出要求管理员登录态，筛选条件与 `api.usage_list` 保持一致：时间范围、key、模型、endpoint、success、upstream_mode。导出行包含 `api_key_name`，按当前 API 凭据备注回补；凭据已删除时为空。导出会写审计日志。

## OpenAI 兼容入口

HTTP 路由：

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`

鉴权：

- 下游请求必须使用 `Authorization: Bearer ogw_...`
- 上游请求由服务端通过服务器 Codex 登录态执行，客户端不接触上游凭据

usage 记录：

- 成功和失败请求都会记录 usage log
- 默认记录 endpoint、model、可选 session_id、HTTP 状态、耗时、请求/响应字节数和 token usage
- 上游模式可在管理后台「上游模式」页通过 `api.gateway_upstream_get` / `api.gateway_upstream_set` 读取和切换；未保存运行时设置时，默认 `codex_backend`，也可用 `CODEX_UPSTREAM_MODE` 作为启动时默认值。`codex_backend` 会直接请求 Codex backend `/responses`，backend 失败时自动 fallback 到 `codex_cli`；显式切到 `codex_cli` 时只走 CLI。
- `codex_cli` 模式 token 优先读取 Codex JSON 事件里的 usage，没有事件时才退回字符数估算；`codex_backend` 模式优先读取 Responses SSE `response.completed.usage`
- usage log 会记录 `upstream_configured_mode`、`upstream_mode`、`upstream_fallback` 和 `upstream_error_type`，用于区分配置模式、实际执行模式和 fallback 情况。
- OpenAI-compatible 请求体支持 `reasoning_effort`，可选值为 `low`、`medium`、`high`、`xhigh`
- direct backend 模式会把 `system` / `developer` 消息合并为 `instructions`；若请求没有这类消息，会补一个最小默认 instructions，因为 Codex backend 要求该字段非空
- OpenAI-compatible 图片输入支持 data URL 形式的 `image_url` / `input_image`；CLI 模式会临时落盘并通过 Codex CLI `--image` 附加到本次请求，direct backend 模式会直接传入 `/responses` 内容；单次最多 4 张、单张最大 16 MiB
- 默认不保存 prompt、response body 或正文采样
- `/v1/chat/completions` 和 `/v1/responses` 转发前会检查 key 状态、模型权限、key 级总 / 输入 / 输出 / 非缓存输入 token 日周额度、RPM、TPM、日/月请求配额和日/月 token 配额；超限返回 HTTP `429`
- token 配额以本系统落库 usage 为准，key 级 token 总额度、TPM 和 token 配额允许单次请求短暂越界，下一次请求开始拦截

## 管理员 OAuth 入口

管理员 OAuth 登录默认关闭，配置完整后开放以下 HTTP 路由：

- `GET /auth/oauth/config`：返回 `{ enabled, provider }`，前端据此决定是否显示 OAuth 登录按钮。
- `GET /auth/oauth/start`：发起授权。可传 `frontend_origin` 和 `next`，服务端会把当前前端 origin 和回跳路径写入 signed state。
- `GET /auth/oauth/callback`：OAuth provider 固定回调后端的地址。服务端完成 code exchange 和 userinfo 查询后，签发本系统管理员 JWT，并通过前端 `/oauth/callback` 的 URL fragment 回传登录态。

本地 Google Console 回调登记 `http://localhost:8400/auth/oauth/callback`，不登记 Vite 端口。生产环境登记后端 HTTPS 域名下的同一路径；当前个人部署为 `https://oauth-api.saurick.me/auth/oauth/callback`，并通过 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS` allowlist 前端后台 origin。

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
- `quota_daily_tokens`
- `quota_weekly_tokens`
- `quota_daily_input_tokens`
- `quota_weekly_input_tokens`
- `quota_daily_output_tokens`
- `quota_weekly_output_tokens`
- `quota_daily_billable_input_tokens`
- `quota_weekly_billable_input_tokens`
- `disabled`
- `owner_user_id`

### `api.gateway_upstream_get` / `api.gateway_upstream_set`

读取或切换 Codex 上游模式，用于高频 OpenCode 场景在 direct backend 与 CLI 兼容路径之间切换：

- `mode`：当前运行时模式，`codex_backend` 或 `codex_cli`
- `default_mode`：未保存设置时的默认模式
- `options[]`：前端开关可展示的模式列表

`api.gateway_upstream_set` 参数为 `mode`。保存后立即影响后续 `/v1/chat/completions` 与 `/v1/responses` 请求；历史 usage 不会改写。

### `api.usage_list`

返回最近 usage 列表和汇总：

- `items`
- `total`
- `items[].upstream_configured_mode`
- `items[].api_key_id`
- `items[].api_key_name`
- `items[].api_key_prefix`
- `items[].upstream_mode`
- `items[].upstream_fallback`
- `items[].upstream_error_type`
- `summary.total_requests`
- `summary.success_requests`
- `summary.failed_requests`
- `summary.total_tokens`
- `summary.backend_requests`
- `summary.cli_requests`
- `summary.fallback_requests`
- `summary.average_duration_ms`
- `summary.estimated_cost_usd`

筛选条件支持 `upstream_mode=codex_backend|codex_cli`。未传时不按上游模式过滤。

### `api.usage_buckets`

返回 usage 聚合，用于后台趋势图、每日模型汇总和按天统计表：

- `items`
- `group_by`
- `items[].bucket_start`
- `items[].model`
- `items[].total_requests`
- `items[].success_requests`
- `items[].failed_requests`
- `items[].input_tokens`
- `items[].cached_tokens`
- `items[].output_tokens`
- `items[].reasoning_tokens`
- `items[].total_tokens`
- `items[].backend_requests`
- `items[].cli_requests`
- `items[].fallback_requests`
- `items[].estimated_cost_usd`

当前支持 `group_by=day` 和 `group_by=day_model`。`day_model` 会按日期 + 模型拆分聚合，前端可点击详情后用同一天的 `start_time/end_time` 和 `model` 调用 `api.usage_list` 下钻请求明细。费用估算优先读取数据库模型价格覆盖值，未配置时回落到服务端内置 OpenAI Codex 模型价格表；仍无价格时返回 `null`，前端显示“未配置价格”。reasoning tokens 作为输出 tokens 子集展示，不重复计费。

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
- `items[].backend_requests`
- `items[].cli_requests`
- `items[].fallback_requests`
- `items[].average_duration_ms`
- `items[].estimated_cost_usd`

筛选条件与 `api.usage_list` 保持一致。费用估算优先使用数据库价格覆盖值，再回落到内置官方价格表；窗口内存在未配置价格的模型时，对应 key 的 `estimated_cost_usd` 返回 `null`。

### `api.usage_session_summaries`

返回 usage 按 `session_id` 聚合，用于后台把同一个客户端会话合并展示：

- `items`
- `total`
- `items[].session_id`
- `items[].api_key_id`
- `items[].api_key_prefix`
- `items[].api_key_name`
- `items[].total_requests`
- `items[].success_requests`
- `items[].failed_requests`
- `items[].input_tokens`
- `items[].cached_tokens`
- `items[].output_tokens`
- `items[].reasoning_tokens`
- `items[].total_tokens`
- `items[].backend_requests`
- `items[].cli_requests`
- `items[].fallback_requests`
- `items[].average_duration_ms`
- `items[].first_seen_at`
- `items[].last_seen_at`
- `items[].estimated_cost_usd`

筛选条件与 `api.usage_list` 保持一致，并支持用 `session_id` 继续下钻请求级明细。`session_id` 来自客户端请求头 `X-Session-ID` / `X-Conversation-ID` / `X-Thread-ID`，或请求 JSON 顶层及 `metadata` 里的 `session_id` / `conversation_id` / `thread_id`。没有会话标识的历史记录不会伪造成会话聚合行。

### `api.official_model_price_list`

返回当前代码内置的 Codex 客户端可用模型中已定价模型的价格表，用于前端费用展示和费用估算兜底；当前已定价模型为 `gpt-5.5`、`gpt-5.4`、`gpt-5.4-mini`、`gpt-5.3-codex`、`gpt-5.2`：

- `items[].model_id`
- `items[].input_usd_per_million`
- `items[].cached_input_usd_per_million`
- `items[].output_usd_per_million`

说明：价格单位为 USD / 1M tokens。`gpt-5.3-codex-spark` 保留在 Codex 客户端可用模型候选中，但仍是 research preview，价格未定，不进入费用估算单价表；长上下文、Batch、Flex、Priority 和区域处理加价需要新增口径字段后再接入。

## 不再属于模板主干的业务能力

以下能力已经从当前项目默认主干移除，不应再假定存在：

- 积分
- 订阅
- 邀请码
- 管理员层级
- 任何行业特定业务域

如果后续业务需要这些能力，应按真实需求重新定义 schema、错误码、接口和前端消费层，而不是把旧模板逻辑直接加回主干。
