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

## `data.openai`

- `data.openai.apiKey`
- `data.openai.baseUrl`
- `data.openai.upstreamProxyUrl`
- `data.openai.requestTimeoutSeconds`

说明：

- 这是 OpenAI 兼容转发链路的上游配置。
- `apiKey` 用于配置 OpenAI 兼容上游鉴权。
- `baseUrl` 默认使用 `https://api.openai.com/v1`，兼容测试时可指向本地 mock upstream。
- `upstreamProxyUrl` 为空时直连上游；需要统一出口时可配置 HTTP 或 SOCKS5 代理。
- `requestTimeoutSeconds` 控制上游请求超时，流式请求同样受该超时约束。

## Codex CLI 上游环境变量

这组配置不进入 `conf.proto`，只用于个人部署的统一出口模式：

- `OAUTH_API_UPSTREAM_PROVIDER`
  - `openai_api`：默认值，使用 `data.openai.apiKey` / `OPENAI_API_KEY`。
  - `codex_cli`：使用容器内 Codex CLI 调用服务器 Codex 登录态。
- `CODEX_HOST_HOME`：宿主机 Codex 登录态目录，Compose 默认挂载到容器。
- `CODEX_CONTAINER_HOME`：容器内 `CODEX_HOME`，默认 `/root/.codex`。
- `CODEX_CLI_BIN`：Codex CLI 可执行文件，默认 `codex`。
- `CODEX_CLI_TIMEOUT_SECONDS`：单次 Codex CLI upstream 超时，默认 `600` 秒。

`codex_cli` 模式适合多台客户端统一走本服务出口：客户端只保存 `ogw_...` 下游 key，服务端统一使用服务器 Codex 登录态，并继续记录 usage。该模式会为每次 `/v1/chat/completions` 或 `/v1/responses` 启动一次 Codex CLI；服务端会串行执行 Codex CLI upstream，避免多个 CLI 进程同时争用 Codex 登录态和本地状态。低配服务器应提高 `APP_MEM_LIMIT`，客户端 provider timeout 建议不低于 600 秒。

## `data.api`

- `data.api.rateLimitEnabled`
- `data.api.exportMaxDays`
- `data.api.modelSyncTimeoutSeconds`
- `data.api.alertRetentionDays`

说明：

- `rateLimitEnabled=true` 时，`/v1/chat/completions` 与 `/v1/responses` 会在转发前检查 key+model 策略；关闭后仍保留 key 状态与模型权限校验。
- `exportMaxDays` 控制 `/admin/exports/usage.csv` 和 `/admin/exports/usage.json` 的最大导出时间范围。
- `modelSyncTimeoutSeconds` 只控制后台模型同步动作；它会调用配置的 OpenAI 兼容上游 `GET /models`，不会抓取网页价格。
- `alertRetentionDays` 是站内告警事件保留天数的生产配置口径，当前事件写入与确认已落库，清理任务后续接入时复用该字段。
- 模型价格默认为空，必须在后台价格表维护；价格缺失时 usage 费用估算返回 `null`，前端显示“未配置价格”，不硬编码美元单价。

## 初始化后必须改的字段

以下内容不应直接进入交付项目：

- `data.postgres.dsn`
- `data.auth.jwtSecret`
- `data.auth.admin.username`
- `data.auth.admin.password`
- `data.openai.apiKey`
- `data.openai.upstreamProxyUrl`
- `data.api.rateLimitEnabled`
- `data.api.exportMaxDays`
- `data.api.modelSyncTimeoutSeconds`
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
