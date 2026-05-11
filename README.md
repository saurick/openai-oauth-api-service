# OpenAI OAuth API Service

OpenAI OAuth API Service 是一个长期维护的 OpenAI 兼容 API 转发与 token/usage 统计管理项目。系统通过后台账号登录管理 API 凭据，向下游签发独立 API key，并通过服务器 Codex 登录态统一执行上游调用，记录 usage、状态码、延迟、字节数和 token 用量。

## 边界

| 类型 | 说明 |
| --- | --- |
| 支持 | 管理员账号登录、本系统 JWT 签发与后台权限控制 |
| 支持 | 下游 API key 签发、吊销、配额、usage 监控 |
| 支持 | OpenAI 兼容 API 转发，例如 `/v1/responses`、`/v1/chat/completions` |
| 支持 | Codex backend / Codex CLI 上游出口、结构化日志、健康检查、Compose 部署 |

## 技术栈

| 路径 | 技术栈 | 说明 |
| --- | --- | --- |
| `server/` | Go + Kratos + Ent + Atlas + PostgreSQL | 长期主后端 |
| `web/` | Vite + React | 管理后台 |
| `server/deploy/compose/prod/` | Docker Compose | 当前主部署路径 |
| `legacy-python-mvp/` | FastAPI + SQLite | 首轮 MVP 参考实现，不作为长期主路径 |

## 快速开始

### 前端

```bash
cd web
pnpm install
pnpm start
```

默认地址：`http://localhost:5176`

### 后端

```bash
cd server
make init
make run
```

### 数据迁移

```bash
cd server
make data
make migrate_apply
```

执行迁移前可先确认当前命中的数据库：

```bash
cd server
make print_db_url
make migrate_status
```

## 配置

开发环境配置入口：

- `server/configs/dev/config.yaml`
- `server/configs/dev/config.local.example.yaml`
- `server/.env.example`

生产 Compose 配置入口：

- `server/deploy/compose/prod/.env.example`
- `server/deploy/compose/prod/compose.yml`

关键环境变量：

```bash
OAUTH_API_JWT_SECRET=change-this-secret
OAUTH_API_ADMIN_USERNAME=admin
OAUTH_API_ADMIN_PASSWORD=adminadmin
POSTGRES_DSN=postgres://postgres:change-this-password@postgres:5432/openai_oauth_api_service?sslmode=disable
TRACE_ENDPOINT=
```

可选管理员 OAuth 登录：

```bash
OAUTH_API_OAUTH_PROVIDER=google
OAUTH_API_OAUTH_CLIENT_ID=...
OAUTH_API_OAUTH_CLIENT_SECRET=...
OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS=https://oauth-api.saurick.me
```

本地 Google OAuth Client 只需要登记后端固定回调 `http://localhost:8400/auth/oauth/callback`。前端当前端口会通过 signed state 自动回跳，例如 `http://localhost:5176/oauth/callback`；生产环境继续登记线上 HTTPS 后端回调，并用 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS` 明确允许前端后台域名。

API 转发统一使用服务器 Codex 登录态，默认走 direct Codex backend，并在 backend 失败时自动用 Codex CLI 兜底；管理后台「上游模式」页可以运行时切换 `codex_backend` / `codex_cli`，`CODEX_UPSTREAM_MODE` 只作为未保存后台设置时的启动默认值：

```bash
CODEX_HOST_HOME=/root/.codex
CODEX_CONTAINER_HOME=/root/.codex
CODEX_UPSTREAM_MODE=codex_backend
CODEX_CLI_BIN=codex
CODEX_CLI_TIMEOUT_SECONDS=600
CODEX_BACKEND_BASE_URL=https://chatgpt.com/backend-api/codex
CODEX_BACKEND_TIMEOUT_SECONDS=600
# 启动默认值改为旧路径；后台保存过模式后，以后台设置为准：
# CODEX_UPSTREAM_MODE=codex_cli
```

管理后台入口：

- 管理登录：`/admin-login`
- API 运营控制台：`/admin-api`

开发与当前个人部署默认会初始化管理员账号 `admin/adminadmin`。不要在部署时擅自生成或替换管理员密码；如需改密，应由维护者明确指定后再调整 `OAUTH_API_ADMIN_PASSWORD` 并重启服务。JWT secret、数据库密码和 Codex 登录态路径仍必须通过私有环境变量配置。

## 下游调用 API

当前主路径由管理员在后台生成下游凭据，再交给客户端调用：

1. 打开 `/admin-login`，使用管理员账号登录。
2. 进入 `/admin-keys`，生成或复用一个 `ogw_` key。
3. OpenAI 兼容客户端使用本服务的 `/v1` 作为 Base URL，并把 `ogw_` key 作为 `OPENAI_API_KEY`。

本地示例：

```bash
export OPENAI_BASE_URL=http://localhost:8400/v1
export OPENAI_API_KEY=ogw_xxx
```

兼容请求可通过 `reasoning_effort` 传递推理强度，当前支持 `low`、`medium`、`high`、`xhigh`；服务端会把该值作为请求级 usage 快照记录，并分别转为 direct backend 的 `reasoning.effort` 或 Codex CLI 的 `model_reasoning_effort`。图片输入支持 OpenAI-compatible 的 data URL 形式 `image_url` / `input_image`；PDF 输入支持 `input_file` / `file` 的 `application/pdf` data URL 或带 `mimeType=application/pdf` 的 base64 文件数据。图片在 CLI 模式会转为 `codex exec --image` 附件，PDF 仅支持 direct backend 模式；文本类附件由客户端读取成文本后按普通 `text` 输入转发。

生产环境把 `OPENAI_BASE_URL` 换成部署域名下的 `/v1`，当前个人部署为：

```bash
export OPENAI_BASE_URL=https://oauth-api.saurick.me/v1
```

这里的 `ogw_` key 是本系统下游 key；上游调用由服务端通过服务器 Codex 登录态统一执行。

## 常用质量命令

```bash
bash scripts/doctor.sh
bash scripts/qa/fast.sh
bash scripts/qa/full.sh
```

前端样式或交互改动时额外执行：

```bash
cd web
pnpm lint
pnpm css
pnpm test
pnpm style:l1
```

后端改动至少执行：

```bash
cd server
go test ./...
```

## 文档索引

- 架构说明：`docs/architecture.md`
- 运维说明：`docs/operations.md`
- 后端说明：`server/README.md`
- 前端说明：`web/README.md`
- 部署说明：`server/deploy/README.md`
- 进度记录：`progress.md`
