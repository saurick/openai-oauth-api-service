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

## Codex CLI 上游环境变量

服务端只保留 Codex CLI 作为 `/v1` 上游：

- `CODEX_HOST_HOME`：宿主机 Codex 登录态目录，Compose 默认挂载到容器。
- `CODEX_CONTAINER_HOME`：容器内 `CODEX_HOME`，默认 `/root/.codex`。
- `CODEX_CLI_BIN`：Codex CLI 可执行文件，默认 `codex`。
- `CODEX_CLI_TIMEOUT_SECONDS`：单次 Codex CLI upstream 超时，默认 `600` 秒。

该模式适合多台客户端统一走本服务出口：客户端只保存 `ogw_...` 下游 key，服务端统一使用服务器 Codex 登录态，并继续记录 usage。服务端会为每次 `/v1/chat/completions` 或 `/v1/responses` 启动一次 Codex CLI，并串行执行 Codex CLI upstream，避免多个 CLI 进程同时争用 Codex 登录态和本地状态。低配服务器应提高 `APP_MEM_LIMIT`，客户端 provider timeout 建议不低于 600 秒。

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

OAuth provider 的回调地址固定登记后端 `/auth/oauth/callback`，例如本地 `http://localhost:8400/auth/oauth/callback`。前端当前 origin 通过 signed state 动态回跳，`localhost / 127.0.0.1 / ::1` 自动允许任意本地端口；生产前端域名必须显式写入 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS`。授权邮箱必须匹配现有管理员用户名，或已绑定在 `admin_users.oauth_provider/oauth_subject`，服务端不会自动创建管理员。

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
