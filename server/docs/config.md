# 服务配置说明

本文档对应：

- `server/internal/conf/conf.proto`
- `server/configs/dev/config.yaml`
- `server/configs/prod/config.yaml`

## 顶层结构

当前配置分为 4 组：

- `server`
- `log`
- `trace`
- `data`

## `server`

用于定义监听地址和超时：

- `server.http.addr`
- `server.http.timeout`
- `server.grpc.addr`
- `server.grpc.timeout`

当前项目默认值：

- HTTP `0.0.0.0:8400`
- gRPC `0.0.0.0:9400`
- HTTP 超时 `28900s`，需要覆盖 Codex backend / CLI 最长 `28800s` 上游等待窗口，避免长上下文请求在外层 HTTP context 先被切断并记录为 502。
- gRPC 超时 `10s`，当前主要用于内部接口默认保护，不承载 OpenAI-compatible 长耗时转发。

## `log`

- `log.debug`
  - `true` 时更适合本地开发
  - `false` 时更适合生产环境

## `trace`

当前只保留 `jaeger` 这一组字段：

- `trace.jaeger.traceName`
- `trace.jaeger.endpoint`
- `trace.jaeger.ratio`

说明：

- `traceName` 为空时，会回退到 `cmd/server/main.go` 里的默认服务名。
- `endpoint` 为空时，服务仍能启动，只是使用本地无 exporter 的 tracer provider。
- 模板当前通过 OTLP HTTP exporter 发 trace，不要求一定叫 Jaeger；如果后续业务改用其他 OTLP 兼容后端，只需替换 endpoint 和服务名即可。

## `data.postgres`

- `data.postgres.dsn`
- `data.postgres.debug`

说明：

- 这是当前项目唯一真正运行时必需的数据依赖。
- `debug=true` 时会输出更多 SQL 调试信息，更适合开发环境。

## `data.etcd`

- `data.etcd.hosts`

说明：

- 当前配置骨架里保留了这一组字段，方便后续业务继续扩展。
- 但当前项目默认代码路径并未真正初始化 etcd 客户端，所以它只是扩展位，不是现阶段必填运行依赖。

## `data.auth`

- `data.auth.jwtSecret`
- `data.auth.jwtExpireSeconds`
- `data.auth.admin.username`
- `data.auth.admin.password`

说明：

- 这组字段决定用户 token 签名和默认管理员初始化逻辑。
- 初始化新项目后必须替换模板里的 JWT 密钥；当前个人部署的默认管理员账号保持 `admin/adminadmin`。不要在部署流程中擅自生成或替换管理员密码，如需改密应由维护者明确指定后再调整 `OAUTH_API_ADMIN_PASSWORD`。

## Codex 上游环境变量

服务端 `/v1` 上游默认通过 `CODEX_UPSTREAM_MODE` 指定启动初始模式；管理员在后台「上游策略」页保存过策略后，后续请求以数据库中的运行时设置为准，无需重启服务。全局推理档位开关同样保存在数据库设置中，默认关闭，不通过环境变量预设：

- `CODEX_HOST_HOME`：宿主机 Codex 登录态目录，Compose 默认挂载到容器。
- `CODEX_CONTAINER_HOME`：容器内 `CODEX_HOME`，默认 `/root/.codex`。
- `CODEX_UPSTREAM_MODE`：启动默认上游模式，默认 `codex_backend`；需要初始强制旧路径时设为 `codex_cli`。
- `CODEX_UPSTREAM_FALLBACK_ENABLED`：是否允许 `codex_backend` 失败后 fallback 到 `codex_cli`，默认 `false`；只建议临时救急打开，工具调用、工具历史和文件输入始终不会 fallback。
- `CODEX_CLI_BIN`：Codex CLI 可执行文件，默认 `codex`。
- `CODEX_APP_SERVER_BIN`：公开余额查询使用的 Codex app-server 可执行文件，默认复用 `CODEX_CLI_BIN`，再回退到 `codex`。
- `CODEX_BALANCE_TIMEOUT_SECONDS`：`GET /public/codex/balance` 启动 Codex app-server 并读取 `account/rateLimits/read` 的超时，默认 `15` 秒。
- `CODEX_BALANCE_CACHE_SECONDS`：公开余额查询的内存缓存时间，默认 `30` 秒；设为 `0` 可关闭缓存。
- `CODEX_CLI_TIMEOUT_SECONDS`：单次 Codex CLI upstream 超时，默认 `28800` 秒。
- `CODEX_BACKEND_BASE_URL`：direct backend 基础地址，默认 `https://chatgpt.com/backend-api/codex`。
- `CODEX_BACKEND_TIMEOUT_SECONDS`：direct backend 单次请求超时，默认 `28800` 秒。
- `CODEX_BACKEND_RETRY_ATTEMPTS`：direct backend 瞬时失败重试次数，默认 `2`；仅对 HTTP `429` / `5xx` 和连接类错误生效，`response.failed` / `response.incomplete` 这类上游终态事件不做服务端盲重试。
- `CODEX_BACKEND_USER_AGENT`：direct backend 请求 `User-Agent`，默认 `codex-cli`。
- `GATEWAY_TRUSTED_PROXY_CIDRS`：可选可信反代 CIDR 列表，多个值用逗号分隔；设置后只有匹配的直连来源才采信 `X-Forwarded-For` / `X-Real-IP`。留空时默认信任本机、内网和 link-local 直连来源，适配 Docker/Nginx 反代部署。
- `GATEWAY_STREAM_HEARTBEAT_SECONDS`：`stream=true` 请求等待上游期间的 SSE keepalive 间隔，默认 `15` 秒；backend Responses 流会在连接上游前先建立下游 SSE 并持续输出 keepalive，用于避免 Codex / OpenCode / Cloudflare / 代理在长请求无输出时断开连接。
- `GATEWAY_LARGE_REQUEST_MIN_BYTES`：大上下文保护的请求体阈值，默认 `65536` 字节；设为 `0` 可关闭该保护。
- `GATEWAY_LARGE_REQUEST_MAX_INFLIGHT_PER_KEY`：同一 API key 同时允许进入上游的大请求数量，默认 `30`；用于支持一个 key 下的多个 Codex / OpenCode 会话并行工作。超过后返回 HTTP `429` / `gateway_large_request_inflight`，并带 `Retry-After` 与 `error.retry_after_seconds`，避免客户端异常循环并发消耗 token。
- `GATEWAY_LARGE_REQUEST_BURST_MAX_PER_KEY`：同一 API key 在突发窗口内允许进入上游的大请求数量，默认 `120`；设为 `0` 可关闭突发保护。
- `GATEWAY_LARGE_REQUEST_BURST_WINDOW_SECONDS`：大请求突发保护窗口，默认 `60` 秒；超过窗口内上限后返回 HTTP `429` / `gateway_large_request_burst`，并用同一窗口秒数填充 `Retry-After` 与 `error.retry_after_seconds`。这是本地网关保护，不表示上游额度耗尽。
- `GATEWAY_CONTEXT_COMPACT_BYTES`：模型未配置字节阈值时的全局运维覆盖；留空时按模型推荐值生效。默认部署模板不再给旧固定值，避免覆盖后台模型推荐。
- `GATEWAY_CONTEXT_HARD_BYTES`：模型未配置硬字节阈值时的全局运维覆盖；留空时按模型推荐值生效。
- `GATEWAY_CONTEXT_COMPACT_TOKENS`：模型未配置 token 阈值时的全局运维覆盖；留空时按模型推荐值生效。
- `GATEWAY_CONTEXT_HARD_TOKENS`：模型未配置硬 token 阈值时的全局运维覆盖；留空时按模型推荐值生效。
- `GATEWAY_CONTEXT_KEEP_ITEMS`：模型未配置最近保留条数时的全局运维覆盖；留空时默认 `8`。工具输出和 function_call_output 会向前扩展边界，避免拆断最近工具闭环。
- `CODEX_AUTH_FILE`：可选，显式指定 Codex `auth.json`；默认读取 `CODEX_HOME/auth.json`。
- `CODEX_REFRESH_TOKEN_URL_OVERRIDE`：可选，覆盖 ChatGPT OAuth refresh token 端点，默认 `https://auth.openai.com/oauth/token`。
- `HTTP_PROXY` / `HTTPS_PROXY` / `WS_PROXY` / `WSS_PROXY` / `ALL_PROXY` 及对应小写变量：可选 Codex CLI 出站代理。
- `NO_PROXY` / `no_proxy`：可选代理排除列表，至少应包含 `localhost,127.0.0.1,::1,postgres,openai-oauth-api-service-postgres`。
- `NODE_USE_ENV_PROXY`：Node.js 代理环境开关；Codex CLI 需要跟随上述代理变量时设置为 `1`。

`codex_backend` 模式复用同一个 app-server 进程直接请求 `https://chatgpt.com/backend-api/codex/responses`，从 `auth.json` 读取 access token，并在 access token 过期或上游返回 401 时用 refresh token 刷新后写回 `auth.json`；该模式适合高频 OpenCode 调用。默认策略是 Backend 直连，backend 请求失败时直接返回上游错误，避免把客户端本机工具错误降级为服务端 `codex exec`。确需临时救急时可在后台选择 Backend + CLI 兜底，或设置 `CODEX_UPSTREAM_FALLBACK_ENABLED=true` 作为初始环境口径，仅允许 CLI 能忠实处理的纯文本 / 图片请求 fallback；带工具调用、工具历史或文件输入的请求始终只允许走 backend。全局推理档位可在后台「上游策略」页切换，单个 API key 可继承全局、关闭覆盖或覆盖为 Fast / Medium / High / Deep；后台设置会覆盖客户端传入的 `reasoning_effort`，用于约束 Codex / OpenCode 这类会自动带默认 effort 的客户端。`codex_cli` 模式会为每次 `/v1/chat/completions` 或 `/v1/responses` 启动一次 Codex CLI，并串行执行上游请求，稳定但首包和单次延迟较高。usage 记录会同时保存配置模式、实际执行模式、fallback 状态、最终生效的 reasoning effort、上下文压缩前后体积和压缩摘要，后台统计表可按上游模式筛选并展示两种模式的请求数。

网关上下文压缩只保存工程摘要，不保存完整 prompt 或模型输出正文。压缩会尽量保留系统 / developer 指令、最近若干轮未压缩消息和最近完整工具闭环；较早历史会收口为带文件路径、错误线索、命令 / 测试线索、最近进度与交接锚点、片段摘要的工程摘要。单个超长 `input`、`content` 或 `function_call_output` 被裁剪时，不只保留固定短尾部，还会优先提取“最新会话进度 / 当前状态 / 下一步 / 验证 / 部署 / 阻塞 / 风险”等附近文本；多消息数组压缩时也会插入显式恢复锚点，并从被裁掉片段尾部反向提取最近用户请求，避免 Codex 压缩恢复后被系统规则或末尾噪声带偏。压缩阈值按“模型级配置 > 环境变量运维覆盖 > 内置模型推荐值 > 旧默认兜底”的顺序生效；模型级配置在后台模型管理页保存后立即影响后续请求，不需要重启服务。内置推荐按 Codex 使用体验控制在 `400K` 上下文窗口内，默认 `260K` 开始压缩、`380K` 硬拦截，字节阈值默认 `1M` / `1.9M`；如果确实要使用 API 模型更大的长上下文窗口，应在后台按模型显式覆盖，并接受更高 token 消耗和 long-context 价格档。后台阈值输入支持整数、`K` 和 `M` 单位，例如 `260K` 表示 260000，`0.38M` 表示 380000；保留条数只接受普通整数。若压缩后仍超过硬阈值，网关会在转发前返回 `context_length_exceeded`，避免 Codex CLI 对同一超长请求连续重试。

两种模式下客户端都只保存 `ogw_...` 下游 key，服务端统一使用服务器 Codex 登录态，并继续记录 usage。direct backend 模式不会启动 `codex exec`，因此也不会注入 Codex CLI 自身的大量 agent 上下文；token usage 更接近客户端实际请求体。

公开余额查询路径 `GET /public/codex/balance` 不走 `/v1` 下游 key，也不要求管理员登录。它会按请求启动 Codex app-server，读取 `account/rateLimits/read`，再返回裁剪后的限额窗口、剩余百分比、重置时间和 credits 余额。该路径不会返回账号邮箱或 token；若线上不希望公开展示余额 / 限额百分比，应在 Nginx / Cloudflare 层限制访问。

如果宿主机通过 mihomo / Clash 提供代理，优先让代理只监听 Docker bridge 网关地址，并在 Compose `.env` 中显式注入代理环境变量。例如 app-server 所在网络网关为 `172.19.0.1` 时，可使用 `http://172.19.0.1:7890`。不要为了 app-server 默认启用全局 TUN，除非已经确认整机路由、Docker bridge 和回滚方式都可控。

## 可选管理员 OAuth 环境变量

管理员 OAuth 登录默认关闭，配置完整后才启用：

- `OAUTH_API_OAUTH_PROVIDER`：默认 `google`。
- `OAUTH_API_OAUTH_CLIENT_ID`
- `OAUTH_API_OAUTH_CLIENT_SECRET`
- `OAUTH_API_OAUTH_AUTH_URL`：Google 默认 `https://accounts.google.com/o/oauth2/v2/auth`。
- `OAUTH_API_OAUTH_TOKEN_URL`：Google 默认 `https://oauth2.googleapis.com/token`。
- `OAUTH_API_OAUTH_USERINFO_URL`：Google 默认 `https://openidconnect.googleapis.com/v1/userinfo`。
- `OAUTH_API_OAUTH_SCOPES`：默认 `openid email profile`。
- `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS`：生产前端后台 origin allowlist，多个值用逗号或空格分隔。

OAuth provider 的回调地址固定登记后端 `/auth/oauth/callback`，例如本地 `http://localhost:8400/auth/oauth/callback`；当前个人部署为 `https://oauth-api.saurick.me/auth/oauth/callback`。前端当前 origin 通过 signed state 动态回跳，`localhost / 127.0.0.1 / ::1` 自动允许任意本地端口；生产前端域名必须显式写入 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS`。授权邮箱必须匹配现有管理员用户名，或已绑定在 `admin_users.oauth_provider/oauth_subject`，服务端不会自动创建管理员。

## `data.api`

- `data.api.rateLimitEnabled`
- `data.api.exportMaxDays`
- `data.api.alertRetentionDays`

说明：

- `rateLimitEnabled=true` 时，`/v1/chat/completions` 与 `/v1/responses` 会在转发前检查 key+model 策略；关闭后仍保留 key 状态与模型权限校验。
- `exportMaxDays` 控制 `/admin/exports/usage.csv` 和 `/admin/exports/usage.json` 的最大导出时间范围。
- `alertRetentionDays` 是站内告警事件保留天数的生产配置口径，当前事件写入与确认已落库，清理任务后续接入时复用该字段。
- usage 费用估算优先使用数据库模型价格表覆盖值；未配置覆盖值时，回落到服务端内置 Codex 客户端可用模型中已定价模型的价格表。当前模型候选集合为 `gpt-5.5`、`gpt-5.4`、`gpt-5.4-mini`、`gpt-5.3-codex`、`gpt-5.3-codex-spark`、`gpt-5.2`；其中 `gpt-5.3-codex-spark` 为 research preview，价格未定，不进入费用估算单价表，长上下文、Batch、Flex、Priority 和区域处理加价暂不纳入当前估算口径。

## 初始化后必须改的字段

以下内容不应直接进入交付项目：

- `data.postgres.dsn`
- `data.auth.jwtSecret`
- `data.auth.admin.username`
- `data.auth.admin.password`
- `data.api.rateLimitEnabled`
- `data.api.exportMaxDays`
- `data.api.alertRetentionDays`
- `trace.jaeger.traceName`
- `trace.jaeger.endpoint`

## 配置选择建议

- 本地开发：
  - `log.debug=true`
  - `data.postgres.debug=true`
  - `trace.jaeger.ratio=1`
- 生产环境：
  - `log.debug=false`
  - `data.postgres.debug=false`
  - `trace.jaeger.ratio` 按观测成本控制

## 额外建议

- 生产环境不要把最终密钥长期写死在仓库中的 YAML 文件里。
- 如果项目会长期运行，建议把敏感配置迁移到 `.env`、密钥管理服务、K8s Secret 或其他外部注入方式。
