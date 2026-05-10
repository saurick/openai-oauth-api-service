# Compose 部署

本目录是 OpenAI OAuth API Service 当前主部署路径，包含 PostgreSQL 和后端服务。

## 文件

| 文件 | 说明 |
| --- | --- |
| `compose.yml` | PostgreSQL + app-server |
| `.env.example` | 环境变量示例，复制为 `.env` 后填写真实值 |

## 启动

目标服务器配置较低，部署时不要在服务器上构建镜像或前后端产物。先在本地或 CI 完成 `docker build` / 前端构建 / Go 构建并导出镜像包，再把镜像包上传到服务器，由服务器执行 `docker load` 与 `docker compose up`。

```bash
cd server/deploy/compose/prod
cp .env.example .env
docker compose -f compose.yml up -d
```

至少替换以下值：

```bash
POSTGRES_PASSWORD=...
POSTGRES_DSN=...
OAUTH_API_JWT_SECRET=...
```

API 上游统一使用服务器 Codex 登录态：

```bash
CODEX_HOST_HOME=/root/.codex
CODEX_CONTAINER_HOME=/root/.codex
CODEX_UPSTREAM_MODE=codex_backend
CODEX_CLI_BIN=codex
CODEX_CLI_TIMEOUT_SECONDS=600
APP_MEM_LIMIT=900m
APP_MEM_RESERVATION=256m
```

默认 `codex_backend` 模式要求服务器先完成 `codex login`，app-server 容器会挂载 `CODEX_HOST_HOME` 并直接读取容器内 Codex `auth.json` 的 access token 请求 Codex backend `/responses`。access token 过期或上游返回 401 时会用 refresh token 刷新并写回 `auth.json`。客户端仍只配置本系统签发的 `ogw_...` 下游 key。

如果需要强制旧路径，可切换 Codex CLI 模式：

```bash
CODEX_UPSTREAM_MODE=codex_cli
```

`codex_backend` 不会为每次请求启动 `codex exec` 子进程，也不会注入 Codex CLI 自身的大量 agent 上下文，适合高频低延迟调用。默认策略是 backend 失败时自动 fallback 到 `codex_cli`；若想完全避开 backend 协议变化风险，可把 `CODEX_UPSTREAM_MODE` 固定为 `codex_cli`。

如果服务器 Codex CLI 需要走宿主机 mihomo / Clash，优先使用显式代理环境变量，而不是直接启用全局 TUN。推荐让代理只监听 app-server 所在 Docker bridge 网关，例如 `172.19.0.1:7890`，并在 `.env` 中配置：

```bash
HTTP_PROXY=http://172.19.0.1:7890
HTTPS_PROXY=http://172.19.0.1:7890
WS_PROXY=http://172.19.0.1:7890
WSS_PROXY=http://172.19.0.1:7890
ALL_PROXY=http://172.19.0.1:7890
NO_PROXY=localhost,127.0.0.1,::1,postgres,openai-oauth-api-service-postgres
http_proxy=http://172.19.0.1:7890
https_proxy=http://172.19.0.1:7890
ws_proxy=http://172.19.0.1:7890
wss_proxy=http://172.19.0.1:7890
all_proxy=http://172.19.0.1:7890
no_proxy=localhost,127.0.0.1,::1,postgres,openai-oauth-api-service-postgres
NODE_USE_ENV_PROXY=1
```

管理员账号默认保持 `admin/adminadmin`。只有维护者明确要求改密时，才设置 `OAUTH_API_ADMIN_PASSWORD` 并重启 `app-server`；部署过程不要擅自生成随机管理员密码。

如启用管理员 OAuth 登录，OAuth provider 回调固定登记后端 `/auth/oauth/callback`。本地为 `http://localhost:8400/auth/oauth/callback`；当前个人部署为 `https://oauth-api.saurick.me/auth/oauth/callback`。前端后台域名通过 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS` allowlist 控制，避免授权完成后跳到未登记来源。

## 说明

- Compose 主路径不包含 Jaeger，`TRACE_ENDPOINT` 默认为空；后续需要 tracing 时接入外部 OTLP endpoint。
- 真实 `.env`、认证信息、代理配置和数据库密码不得提交到仓库。
- 当前不保留远端 SSH 发布脚本；发布流程确定后再补自动化脚本。
