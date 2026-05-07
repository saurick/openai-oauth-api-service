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
- `data.auth.oauth.enabled`
- `data.auth.oauth.providerName`
- `data.auth.oauth.clientId`
- `data.auth.oauth.clientSecret`
- `data.auth.oauth.authUrl`
- `data.auth.oauth.tokenUrl`
- `data.auth.oauth.userInfoUrl`
- `data.auth.oauth.redirectUrl`
- `data.auth.oauth.scopes`

说明：

- 这组字段决定用户 token 签名和默认管理员初始化逻辑。
- 初始化新项目后，必须替换模板里的默认密钥和管理员密码。
- `oauth.enabled=false` 时不展示管理员授权登录入口；开启后，服务端通过 OAuth2/OIDC code flow 换取第三方身份，再签发本系统管理员 JWT。
- `oauth.providerName` 同时作为页面展示名称和本地 OAuth 身份提供方标识；已上线后不要随意改名，否则已绑定管理员不会自动匹配。
- `oauth.redirectUrl` 必须与身份提供方后台登记的回调地址一致，路径为 `/auth/oauth/callback`。

## `data.openai`

- `data.openai.apiKey`
- `data.openai.baseUrl`
- `data.openai.upstreamProxyUrl`
- `data.openai.requestTimeoutSeconds`

说明：

- 这是 OpenAI 兼容转发链路的上游配置。
- `apiKey` 必须是官方 OpenAI API key、Project API key 或 Service Account key，不能使用 Codex / ChatGPT 登录态、Cookie、设备码或个人账号 token。
- `baseUrl` 默认使用 `https://api.openai.com/v1`，兼容测试时可指向本地 mock upstream。
- `upstreamProxyUrl` 为空时直连上游；需要统一出口时可配置 HTTP 或 SOCKS5 代理。
- `requestTimeoutSeconds` 控制上游请求超时，流式请求同样受该超时约束。

## `data.api`

- `data.api.rateLimitEnabled`
- `data.api.exportMaxDays`
- `data.api.modelSyncTimeoutSeconds`
- `data.api.alertRetentionDays`

说明：

- `rateLimitEnabled=true` 时，`/v1/chat/completions` 与 `/v1/responses` 会在转发前检查 key+model 策略；关闭后仍保留 key 状态与模型权限校验。
- `exportMaxDays` 控制 `/admin/exports/usage.csv` 和 `/admin/exports/usage.json` 的最大导出时间范围。
- `modelSyncTimeoutSeconds` 只控制后台模型同步动作；它会调用配置的官方 OpenAI 兼容上游 `GET /models`，不会抓取网页价格。
- `alertRetentionDays` 是站内告警事件保留天数的生产配置口径，当前事件写入与确认已落库，清理任务后续接入时复用该字段。
- 模型价格默认为空，必须在后台价格表维护；价格缺失时 usage 费用估算返回 `null`，前端显示“未配置价格”，不硬编码美元单价。

## 初始化后必须改的字段

以下内容不应直接进入交付项目：

- `data.postgres.dsn`
- `data.auth.jwtSecret`
- `data.auth.admin.username`
- `data.auth.admin.password`
- `data.auth.oauth.clientId`
- `data.auth.oauth.clientSecret`
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
