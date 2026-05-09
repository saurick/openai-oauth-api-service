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
CODEX_CLI_BIN=codex
CODEX_CLI_TIMEOUT_SECONDS=600
APP_MEM_LIMIT=900m
APP_MEM_RESERVATION=256m
```

`codex_cli` 模式要求服务器先完成 `codex login`，app-server 容器会挂载 `CODEX_HOST_HOME` 并在容器内调用 Codex CLI。客户端仍只配置本系统签发的 `ogw_...` 下游 key。

管理员账号默认保持 `admin/adminadmin`。只有维护者明确要求改密时，才设置 `OAUTH_API_ADMIN_PASSWORD` 并重启 `app-server`；部署过程不要擅自生成随机管理员密码。

如启用管理员 OAuth 登录，OAuth provider 回调固定登记后端 `/auth/oauth/callback`。本地为 `http://localhost:8400/auth/oauth/callback`；生产为后端 HTTPS 域名下的同一路径。前端后台域名通过 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS` allowlist 控制，避免授权完成后跳到未登记来源。

## 说明

- Compose 主路径不包含 Jaeger，`TRACE_ENDPOINT` 默认为空；后续需要 tracing 时接入外部 OTLP endpoint。
- 真实 `.env`、认证信息、代理配置和数据库密码不得提交到仓库。
- 当前不保留远端 SSH 发布脚本；发布流程确定后再补自动化脚本。
