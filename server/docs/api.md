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
- `key_reset_secret`
- `key_delete`
- `key_set_disabled`
- `key_disable_all`
- `key_enable_all`
- `gateway_upstream_get`
- `gateway_upstream_set`
- `usage_list`
- `usage_buckets`
- `usage_key_summaries`
- `usage_session_summaries`
- `model_list`
- `official_model_price_list`
- `model_set_enabled`
- `model_context_update`
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

用途：管理员管理下游 API key、组织用户归属、key+model 策略、key 级上游策略覆盖、key 级默认推理档位、固定官方模型列表启停、模型级上下文压缩策略、模型价格、站内告警、usage 汇总、按天聚合、按 key 聚合和最近请求。创建 key 时会随机生成完整 key；若创建参数 `name` 非空，备注必须只包含字母和数字，并会生成 `ogw_<name>_<random>` 形式的明文 key。管理员 `api.key_list` / `api.key_update` 返回完整 `plain_key` 供后台展示和复制；普通组织用户接口不返回完整明文。`api.key_update` 只保存备注、额度、模型权限、上游策略、默认推理档位和禁用状态，不会重新生成 key；key 泄密或需要轮换时调用 `api.key_reset_secret` 单独重置。额度紧张或需要临时停服时，管理员可调用 `api.key_disable_all` 一次性禁用所有当前启用的下游 key；需要恢复调用时，可调用 `api.key_enable_all` 一次性启用所有当前禁用的下游 key；两类全站操作都不删除 key、不改历史 usage。鉴权继续使用 `key_hash`，`key_prefix` 和 `key_last4` 用于 usage 归属与人工识别。

模型目录以服务端代码中的官方 Codex 列表为真源，管理端只允许读取、启停和调整上下文压缩策略；`model_upsert` / `model_delete` 不作为正式管理接口开放。`api.model_context_update` 支持按模型保存 `context_window_tokens`、`context_compact_tokens`、`context_hard_tokens`、`context_compact_bytes`、`context_hard_bytes` 和 `context_keep_items`，阈值字段可传整数或 `260K` / `0.38M` 这类字符串，`K=1000`、`M=1000000`；`0` 表示使用服务端推荐 / 运维覆盖值；保存后仅影响后续请求，不改写历史 usage。内置推荐按 Codex 使用体验控制在 `400K` 上下文窗口内，默认 `260K` 开始压缩、`380K` 硬拦截，避免默认进入 API long-context 高消耗区间。

普通组织用户只允许调用 `api.user_key_list`、`api.user_usage_summary` 和 `api.user_usage_list`，并且后端按当前登录用户过滤 `owner_user_id`，不返回其他用户 key。

HTTP 管理导出：

- `GET /admin/exports/usage.csv`
- `GET /admin/exports/usage.json`

导出要求管理员登录态，筛选条件与 `api.usage_list` 保持一致：时间范围、key、模型、reasoning_effort、endpoint、success、status_code、upstream_mode、client_type、error_type。导出行包含 `api_key_name`、`reasoning_effort` 和 `client_type`，API 凭据备注按当前 key 表回补；凭据已删除时为空。导出会写审计日志。

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
- 默认记录 endpoint、model、最终生效的 reasoning_effort、客户端类型、可选 session_id、HTTP 状态、耗时、请求/响应字节数和 token usage；请求未传且后台没有覆盖档位、或历史旧数据保持空值，不按 token 反推
- 客户端类型字段为 `client_type`，只归类为 `codex`、`opencode` 和 `other`。网关优先读取 `X-Client-Type`、`X-Client-Name`、`X-App-Name`，再从 `User-Agent` 识别 Codex / OpenCode；无法识别或历史旧数据默认归 `other`。
- 上游策略可在管理后台「上游策略」页通过 `api.gateway_upstream_get` / `api.gateway_upstream_set` 读取和切换全局默认；同页也可设置全局推理档位开关，默认关闭。单个 API key 可通过 `upstream_strategy` 覆盖全局上游策略，空值表示继承全局；也可通过 `default_reasoning_effort` 继承全局、关闭覆盖或覆盖为 `low`、`medium`、`high`、`xhigh`。未保存运行时设置时，默认 Backend 直连且不覆盖 reasoning effort。`codex_backend` 会直接请求 Codex backend `/responses`，backend 失败时默认直接返回上游错误；只有显式选择 Backend + CLI 兜底策略时，纯文本 / 图片请求才允许 fallback 到 `codex_cli`。带工具调用、工具历史或文件输入的 backend-only 请求始终不会 fallback 到 CLI；显式切到强制 CLI 时只走 CLI。
- `codex_cli` 模式 token 优先读取 Codex JSON 事件里的 usage，没有事件时才退回字符数估算；`codex_backend` 模式优先读取 Responses SSE `response.completed.usage`
- usage log 会记录 `upstream_configured_mode`、`upstream_mode`、`upstream_fallback`、细分 `upstream_error_type` 和统一 `error_type`，用于区分配置模式、实际执行模式、fallback 情况和最终失败类型；后台表格按 `error_type` 保留原始错误码并展示简短中文说明，完整含义见下方“错误类型”。聚合统计中的 `failed_requests` 表示服务 / 上游 / 网关错误数，默认不包含 `client_canceled`；客户端主动断开以 `client_canceled_requests` 单独返回。
- OpenAI-compatible 请求体支持 `reasoning_effort`、`reasoningEffort` 和 `reasoning.effort`，可选值为 `low`、`medium`、`high`、`xhigh`；最终值按“key 覆盖档位 > 全局覆盖档位 > 客户端请求档位”决定，key 级 `none` 表示强制不使用全局覆盖。direct backend 会转为 OpenAI Responses 口径的 `reasoning.effort`，并默认补 `reasoning.summary=detailed` 以便自定义 Codex provider 展示 reasoning summary；客户端显式传 `auto`、`concise` 或 `detailed` 时会保留，缺失、`none` 或非法值会回补为 `detailed`；CLI 模式会转为 Codex CLI `model_reasoning_effort`
- direct backend 模式会把 `system` / `developer` 消息合并为 `instructions`；若请求没有这类消息，会补一个最小默认 instructions，因为 Codex backend 要求该字段非空；同时会追加服务端级 Codex 运行规则：
  - 可见过程说明：非平凡工具调用、读文件、shell 命令、SSH、浏览器操作或外部请求前，先输出一到两句简体中文用户可见 commentary / process summary，说明即将做什么和为什么。这是执行过程摘要，不是隐藏 chain-of-thought；不要输出完整私有思考链。
  - 压缩恢复续跑：模型在收到 compacted context、reasoning summary 或 history summary 恢复时继续未完成任务，避免只机械回复“已读取上下文，请告诉我下一步”。
  - 追加规则对客户端显式 `instructions` 幂等生效，不依赖 Windows 端 `hide_agent_reasoning` 或全局 AGENTS。这个服务端规则是自定义 Codex provider 新会话工具调用前能看到中文过程说明的主路径。
- direct backend 模式会透传 OpenAI-compatible `tools` / `tool_choice`，并在 chat messages 与 Responses input 之间转换 assistant `tool_calls`、`function_call` 和 `function_call_output`，上游返回 function call 时再映射回 Chat Completions `tool_calls` 或 Responses `function_call`；Codex CLI fallback 默认关闭，打开后也只支持纯文本响应，不支持工具调用回传，因此这类请求在 backend 失败时会直接返回上游错误，避免错误转为服务端 `codex exec`
- direct backend 会在转发 Responses `function_call` 历史前规范化 item `id`，避免客户端回传空字符串或非法字符时触发上游 `input[n].id` 校验错误；若压缩或重放历史中残留了没有对应前置 `function_call` 的孤立 `function_call_output`，转发前会丢弃这类残值，避免上游因 `No tool call found` 拒绝整轮请求
- `/v1/chat/completions` 和 `/v1/responses` 在请求体接近上下文窗口时会先做网关侧压缩预检：阈值按模型级配置、环境变量运维覆盖、内置模型推荐值和旧默认兜底依次决定；压缩会保留系统 / developer 指令、最近消息和最近完整工具闭环，较早历史压缩为工程摘要；单个超长 `input` / `content` / `function_call_output` 会额外保留最近进度、验证、部署、下一步、阻塞或风险附近的交接文本，多消息数组压缩时也会插入显式恢复锚点并优先保留最近用户请求，避免只保留末尾噪声或系统规则导致恢复后丢掉最新执行状态；如果客户端传入 `session_id`，摘要会按 session 保存并在后台会话聚合展示压缩次数、摘要、压缩前后体积、粗估 token 和本次生效阈值。
- 上下文压缩后仍超过硬阈值，或 Codex backend 明确返回 `context_length_exceeded` 时，usage 会记录 `upstream_error_type=context_length_exceeded`，避免继续显示成普通 `codex_backend_response_failed` / 502；非流式预检拦截返回 HTTP `413`，流式已开始后会在 SSE 内返回 `response.failed`。
- 服务端只对可判定为瞬时的 backend 错误做有限重试，包括上游 HTTP `429` / `5xx` 和连接类错误；`context_length_exceeded`、上游终态 `response.failed`、`response.incomplete`、下游 `client_canceled`、鉴权失败、参数校验失败和 backend-only 请求无法 fallback 的情况不做服务端盲重试，避免和 Codex / OpenCode 客户端自身重试叠加放大请求。
- `/v1/chat/completions` 和 `/v1/responses` 的 `stream=true` 会按 `GATEWAY_STREAM_HEARTBEAT_SECONDS` 输出保活事件，避免 Codex / OpenCode / Cloudflare / 代理在长请求无输出时断开连接。`/v1/responses` 在 Codex backend 模式下会先建立下游 SSE 并输出 keepalive，再直连透传上游 Responses SSE `data:` 事件，包括 `response.reasoning_summary_text.delta/done`、执行过程、文本增量、完成事件和 usage；网关只旁路解析 usage / 错误用于落库。CLI 模式和 Chat Completions 兼容入口仍会按 OpenAI SSE 口径合成下游事件，并在有 reasoning summary 时输出 reasoning item / summary 事件。上游在流内失败时返回 `response.failed` 与 `[DONE]`；下游客户端主动断开或下游 SSE 写失败会按 `client_canceled` 记录并取消上游请求，不再归类为 Backend 上游 502。若上游已经返回部分 SSE 事件，但还没出现 `response.completed` 或 `[DONE]` 就断开，会按 `codex_backend_stream_interrupted` 记录，避免和首个有效事件前的连接错误混在一起。
- `/v1/models` 除 OpenAI 标准 `data` 外还返回 Codex CLI 读取的 `models` 元数据，包括 reasoning levels、shell type、context window、reasoning summary、verbosity 和输入模态等字段，用于兼容自定义 provider 的模型刷新；context window 使用当前模型的生效上下文窗口，`effective_context_window_percent` 使用硬拦截阈值占窗口比例；默认按 Codex 体验声明 `supports_reasoning_summaries=true`、`default_reasoning_summary=detailed`、`default_verbosity=medium`
- OpenAI-compatible 图片输入支持 data URL 形式的 `image_url` / `input_image`；CLI 模式会临时落盘并通过 Codex CLI `--image` 附加到本次请求，direct backend 模式会直接传入 `/responses` 内容；单次最多 4 张、单张最大 16 MiB，网关总请求体上限 90 MiB，用于覆盖 data URL 的 base64 膨胀。
- OpenAI-compatible PDF 输入支持 `input_file` / `file` 的 `application/pdf` data URL，或带 `mimeType=application/pdf` / `media_type=application/pdf` 的 base64 文件数据；PDF 仅支持 direct backend 模式，单次最多 4 个、单个最大 16 MiB，网关总请求体上限 90 MiB。`txt` / `md` / 代码等文本类附件由客户端读取成文本后按普通 `text` 输入转发；`doc` / `docx` / `xls` / `xlsx` 暂不声明为原生模态，后续如需支持应先增加明确的服务端转换链路。
- 默认不保存 prompt、response body 或正文采样
- `/v1/chat/completions` 和 `/v1/responses` 转发前会检查 key 状态、模型权限、key 级总 / 输入 / 输出 / 非缓存输入 token 日周额度、RPM、TPM、日/月请求配额和日/月 token 配额；超限返回 HTTP `429`。默认还会对同一 API key 的大请求做并发和突发频率保护：请求体达到 `GATEWAY_LARGE_REQUEST_MIN_BYTES` 后，同一 key 同时最多允许 `GATEWAY_LARGE_REQUEST_MAX_INFLIGHT_PER_KEY` 个上游请求，并且每 `GATEWAY_LARGE_REQUEST_BURST_WINDOW_SECONDS` 秒最多允许 `GATEWAY_LARGE_REQUEST_BURST_MAX_PER_KEY` 个大请求；超过分别返回 `gateway_large_request_inflight` 或 `gateway_large_request_burst`。
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

## Codex 余额公开查询

HTTP 路由：

- `GET /public/codex/balance`

鉴权：

- 不要求管理员登录，也不要求下游 `ogw_` key。
- 服务端使用服务器自己的 Codex 登录态，通过 Codex app-server 的 `account/rateLimits/read` 读取限额和 credits。
- 默认使用 30 秒内存缓存，避免公开查询频繁启动 Codex app-server 子进程；可通过 `CODEX_BALANCE_CACHE_SECONDS` 调整。
- 如果实时读取 Codex app-server 或 ChatGPT usage 接口临时失败，但进程内已有上次成功结果，接口会返回 HTTP 200 和上次成功结果，并带 `stale=true`、`stale_reason=codex_balance_query_failed` 与 `last_error_at`；首次启动且没有成功缓存时仍返回 `codex_balance_query_failed`。

返回字段：

- `credits.balance`：Codex workspace credits 余额字符串；无 credits 时通常为 `"0"`。
- `rate_limits.primary` / `secondary`：主窗口与次窗口用量，包含 `used_percent`、`remaining_percent`、`window_duration_mins`、`resets_at` 和 UTC `resets_at_time`。
- `rate_limits_by_limit_id`：按 Codex limit id 返回的多桶视图，例如默认 `codex` 和特定模型桶。
- `stale`：可选布尔值；为 `true` 时表示当前展示的是上次成功查询结果，不是本次实时读取结果。

该接口不会返回账号邮箱、access token、refresh token、请求正文或模型输出正文。若生产环境不希望任何人看到余额 / 限额百分比，应在反代或边缘层增加 IP allowlist、独立查询 token 或直接屏蔽该路径。

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

创建参数 `name` 可留空；非空时只允许 ASCII 字母和数字。留空时后端使用 `key<hash>` 形式的默认备注，并同样写入新 key 明文前缀。`api.key_create` / `api.key_update` 可传 `default_reasoning_effort`：空字符串表示继承全局覆盖，`none` 表示关闭该 key 的覆盖，`low`、`medium`、`high`、`xhigh` 表示该 key 覆盖档位。`api.key_update` 更新备注时沿用同一限制；普通编辑只更新备注、额度、模型权限、上游策略、默认推理档位和禁用状态，不改写 `key_hash`、`plain_key`、`key_prefix` 或 `key_last4`。管理员列表和编辑响应会返回完整 `plain_key`。

返回创建后的 key 元数据和完整明文：

- `id`
- `name`
- `key_prefix`
- `key_last4`
- `plain_key`
- `allowed_models`
- `upstream_strategy`：空值表示继承全局默认；可选 `backend_only`、`backend_with_cli_fallback`、`codex_cli`
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

### `api.key_reset_secret`

参数：

- `key_id`

用途：在确认某个下游 key 泄密或需要主动轮换时，单独重置该 key 的完整明文和鉴权 hash。重置会立即让旧 key 失效，同时保留备注、归属、模型限制、上游策略、额度、启用状态和历史 usage 归属。

返回重置后的 key 元数据和新的完整明文：

- `id`
- `name`
- `key_prefix`
- `key_last4`
- `plain_key`
- `allowed_models`
- `upstream_strategy`
- `quota_daily_tokens`
- `quota_weekly_tokens`
- `disabled`

### 上游错误类型

`usage.error_type` / `items[].upstream_error_type` 保留机器可检索的原始错误码，便于和服务日志里的 `request_id`、`error_type` 对齐。后台表格只展示短说明；排障时仍以原始错误码和服务日志为准。

| 错误码 | 后台短说明 | 常见含义 / 排查方向 |
| --- | --- | --- |
| `codex_backend_auth_failed` | Backend 鉴权失败 | 服务器 Codex 登录态无效、`auth.json` / refresh token 失效，或上游返回 401 / 403。 |
| `codex_backend_rate_limited` | Backend 限流 | 上游返回 429，可能是账号、模型或组织维度被限流。 |
| `codex_backend_http_5xx` | Backend 5xx | Codex backend 或其上游服务返回 5xx。 |
| `codex_backend_overloaded` | Backend 容量繁忙 | Codex backend 终态事件返回 `server_is_overloaded` 或模型容量繁忙；通常是上游短时容量问题，不等同于本地上下文超限。 |
| `codex_backend_timeout` | Backend 超时 | Codex backend 调用超过超时时间；常见于上游慢、网络慢或 `CODEX_BACKEND_TIMEOUT_SECONDS` 到期。默认生产模板为 `28800` 秒。 |
| `codex_backend_response_failed` | Backend response failed | 上游 SSE 返回 `response.failed`，表示本次 response 执行失败。 |
| `context_length_exceeded` | 上下文超限 | 请求历史超过模型上下文窗口；网关会先尝试压缩可压缩历史，仍超限时直接拦截，避免客户端反复重试。 |
| `codex_backend_response_incomplete` | Backend response incomplete | 上游 SSE 返回 `response.incomplete`，可能因长度、上下文、策略、工具或内部中断。 |
| `codex_backend_stream_error` | Backend 流中断 | SSE 流在首个有效上游事件前连接 reset、unexpected EOF、代理或网络断流。 |
| `codex_backend_stream_interrupted` | Backend 流中途断开 | 上游 SSE 已返回部分事件，但尚未返回 `response.completed` / `[DONE]` 就断开；网关不会自动重试已开始输出的流。 |
| `codex_backend_http_error` | Backend HTTP 错误 | backend 返回其他非 2xx HTTP 状态，且不属于鉴权、限流或 5xx。 |
| `codex_backend_upstream_failed` | Backend 未分类失败 | backend 兜底错误，需要结合服务日志里的 `err` 查看。 |
| `gateway_large_request_inflight` | 大请求并发保护 | 同一 API key 已有大上下文 `/v1/chat/completions` 或 `/v1/responses` 请求在运行，后续大请求会返回 429，避免客户端异常循环并发消耗 token。 |
| `gateway_large_request_burst` | 大请求突发保护 | 同一 API key 在保护窗口内的大上下文请求过于频繁，后续大请求会返回 429，避免客户端异常循环串行消耗 token。 |
| `client_canceled` | 客户端取消 | 下游客户端或入口代理主动断开请求；通常应排查客户端超时、网络中断或流式保活是否被识别。 |
| `codex_cli_timeout` | CLI 超时 | Codex CLI 执行超过 `CODEX_CLI_TIMEOUT_SECONDS`。默认生产模板为 `28800` 秒。 |
| `codex_cli_not_found` | CLI 不存在 | 容器内找不到 `codex` 二进制，或 `CODEX_CLI_BIN` / PATH 配错。 |
| `codex_cli_empty_prompt` | CLI 空输入 | 请求体没有有效 user input，或请求转换后 prompt 为空。 |
| `codex_cli_empty_answer` | CLI 空回复 | CLI 正常退出但未解析到最终回答，可能输出格式变化或模型无最终回答。 |
| `codex_cli_upstream_failed` | CLI 未分类失败 | CLI 兜底错误，需要结合服务日志里的命令错误和输出摘要查看。 |

### `api.gateway_upstream_get` / `api.gateway_upstream_set`

读取或切换 Codex 上游策略，用于高频 OpenCode 场景在 direct backend、CLI 临时兜底和强制 CLI 兼容路径之间切换：

- `strategy`：当前运行时策略，`backend_only`、`backend_with_cli_fallback` 或 `codex_cli`
- `mode`：当前运行时模式，`codex_backend` 或 `codex_cli`
- `fallback_enabled`：当前策略是否允许 backend 失败后 CLI 兜底；工具调用、工具历史和文件输入始终不会兜底
- `default_strategy` / `default_mode`：未保存设置时的默认策略与模式
- `options[]`：前端开关可展示的策略列表
- `default_reasoning_effort`：当前全局推理档位覆盖值，空字符串表示关闭
- `reasoning_effort_options[]`：前端开关可展示的全局档位列表

`api.gateway_upstream_set` 参数优先使用 `strategy`。为兼容旧前端，仍接受旧的 `mode` 与 `fallback_enabled` 参数并转换为对应策略。可同时传 `default_reasoning_effort` 保存全局推理档位覆盖，支持空字符串、`low`、`medium`、`high`、`xhigh`；空字符串表示关闭。保存后立即影响未设置 key 级关闭或覆盖的后续 `/v1/chat/completions` 与 `/v1/responses` 请求；历史 usage 不会改写。

### `api.usage_list`

返回最近 usage 列表和汇总：

- `items`
- `total`
- `items[].upstream_configured_mode`
- `items[].api_key_id`
- `items[].api_key_name`
- `items[].api_key_prefix`
- `items[].client_type`
- `items[].reasoning_effort`
- `items[].upstream_mode`
- `items[].upstream_fallback`
- `items[].upstream_error_type`
- `items[].error_type`
- `items[].diagnostic`
- `items[].diagnostic_summary`
- `items[].diagnostic.upstream_stream_started`
- `items[].diagnostic.upstream_stream_completed`
- `items[].diagnostic.upstream_stream_done_seen`
- `items[].diagnostic.upstream_stream_events`
- `summary.total_requests`
- `summary.success_requests`
- `summary.failed_requests`
- `summary.client_canceled_requests`
- `summary.total_tokens`
- `summary.backend_requests`
- `summary.cli_requests`
- `summary.fallback_requests`
- `summary.codex_requests`
- `summary.opencode_requests`
- `summary.other_client_requests`
- `summary.average_duration_ms`
- `summary.estimated_cost_usd`

筛选条件支持 `key_id` 单凭据兼容参数、`key_ids` 多凭据数组或逗号分隔值、`reasoning_effort=low|medium|high|xhigh`、`status_code`、`upstream_mode=codex_backend|codex_cli`、`client_type=codex|opencode|other`、`error_type` 和 `exclude_error_type`。未传时不按对应维度过滤；同时传 `key_ids` 与 `key_id` 时优先按 `key_ids` 过滤。`upstream_error_type` 仍保留兼容，但前端“错误 / 中断类型”筛选使用 `error_type`，可覆盖客户端取消、上下文超限、网关预检和上游失败等统一错误码；异常请求视图默认传 `exclude_error_type=client_canceled`，避免把 HTTP 499 计入服务错误排查。

### `api.usage_buckets`

返回 usage 聚合，用于后台趋势图、每日模型汇总和按天统计表：

- `items`
- `group_by`
- `items[].bucket_start`
- `items[].model`
- `items[].total_requests`
- `items[].success_requests`
- `items[].failed_requests`
- `items[].client_canceled_requests`
- `items[].input_tokens`
- `items[].cached_tokens`
- `items[].output_tokens`
- `items[].reasoning_tokens`
- `items[].total_tokens`
- `items[].backend_requests`
- `items[].cli_requests`
- `items[].fallback_requests`
- `items[].codex_requests`
- `items[].opencode_requests`
- `items[].other_client_requests`
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
- `items[].client_canceled_requests`
- `items[].input_tokens`
- `items[].cached_tokens`
- `items[].output_tokens`
- `items[].reasoning_tokens`
- `items[].total_tokens`
- `items[].backend_requests`
- `items[].cli_requests`
- `items[].fallback_requests`
- `items[].codex_requests`
- `items[].opencode_requests`
- `items[].other_client_requests`
- `items[].average_duration_ms`
- `items[].estimated_cost_usd`

筛选条件与 `api.usage_list` 保持一致，包括 `key_ids` 多凭据过滤和 `client_type` 客户端过滤。费用估算优先使用数据库价格覆盖值，再回落到内置官方价格表；窗口内存在未配置价格的模型时，对应 key 的 `estimated_cost_usd` 返回 `null`。

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
- `items[].client_canceled_requests`
- `items[].input_tokens`
- `items[].cached_tokens`
- `items[].output_tokens`
- `items[].reasoning_tokens`
- `items[].total_tokens`
- `items[].backend_requests`
- `items[].cli_requests`
- `items[].fallback_requests`
- `items[].codex_requests`
- `items[].opencode_requests`
- `items[].other_client_requests`
- `items[].average_duration_ms`
- `items[].first_seen_at`
- `items[].last_seen_at`
- `items[].estimated_cost_usd`
- `items[].context_compaction_count`
- `items[].context_summary`
- `items[].context_original_bytes`
- `items[].context_compacted_bytes`
- `items[].context_original_tokens`
- `items[].context_compacted_tokens`
- `items[].context_compacted_at`

筛选条件与 `api.usage_list` 保持一致，包括 `key_ids` 多凭据过滤和 `client_type` 客户端过滤，并支持用 `session_id` 继续下钻请求级明细。`session_id` 来自客户端请求头 `X-Session-ID` / `X-Conversation-ID` / `X-Thread-ID`，或请求 JSON 顶层及 `metadata` 里的 `session_id` / `conversation_id` / `thread_id`。没有会话标识的历史记录不会伪造成会话聚合行。

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
