# Compose 部署

本目录是 OpenAI OAuth API Service 当前主部署路径，包含 PostgreSQL 和后端服务。

## 文件

| 文件 | 说明 |
| --- | --- |
| `compose.yml` | PostgreSQL + app-server |
| `compose.nginx.yml` | 可选容器化 Nginx 入口层，迁移或切入口时叠加启用 |
| `.env.example` | 环境变量示例，复制为 `.env` 后填写真实值 |
| `nginx/` | 容器化 Nginx 配置、反代 header、timeout 和域名跳转样本 |

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
CODEX_UPSTREAM_FALLBACK_ENABLED=false
CODEX_CLI_BIN=codex
CODEX_CLI_TIMEOUT_SECONDS=600
CODEX_BACKEND_RETRY_ATTEMPTS=2
APP_MEM_LIMIT=900m
APP_MEM_RESERVATION=256m
```

默认 `codex_backend` 模式要求服务器先完成 `codex login`，app-server 容器会挂载 `CODEX_HOST_HOME` 并直接读取容器内 Codex `auth.json` 的 access token 请求 Codex backend `/responses`。access token 过期或上游返回 401 时会用 refresh token 刷新并写回 `auth.json`。客户端仍只配置本系统签发的 `ogw_...` 下游 key。

如果需要强制旧路径，可切换 Codex CLI 模式：

```bash
CODEX_UPSTREAM_MODE=codex_cli
```

`codex_backend` 不会为每次请求启动 `codex exec` 子进程，也不会注入 Codex CLI 自身的大量 agent 上下文，适合高频低延迟调用。默认策略是 Backend 直连，backend 失败时直接返回上游错误；确需临时救急时可在后台「上游策略」选择 Backend + CLI 兜底，或把 `CODEX_UPSTREAM_FALLBACK_ENABLED` 设为 `true` 作为初始环境口径，仅允许纯文本 / 图片请求 fallback 到 `codex_cli`。带工具调用、工具历史或文件输入的请求不会 fallback 到 CLI，避免把客户端本机工具错误改成服务端 `codex exec`。若想完全避开 backend 协议变化风险，可把 `CODEX_UPSTREAM_MODE` 固定为 `codex_cli`。

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

## 线上迁移

低配服务器上 Atlas 作为宿主机运维工具预装在 `/usr/local/bin/atlas`，不要用 `arigaio/atlas:*` 临时容器执行 migration，也不要把 Atlas 服务写进 `compose.yml`。

```bash
cd /data/openai-oauth-api-service/releases/<release>

atlas version
flock /tmp/atlas-migrate.lock \
  /usr/local/bin/atlas migrate status \
  --dir "file://$PWD/migrate" \
  --url 'postgres://postgres:***@127.0.0.1:5433/openai_oauth_api_service?sslmode=disable'
```

正式 apply 前先执行 `migrate status` 和 dry-run；如果 release 里没有 schema 变更，记录 status 即可，不需要拉取额外镜像。

如启用管理员 OAuth 登录，OAuth provider 回调固定登记后端 `/auth/oauth/callback`。本地为 `http://localhost:8400/auth/oauth/callback`；当前个人部署为 `https://oauth-api.saurick.me/auth/oauth/callback`。前端后台域名通过 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS` allowlist 控制，避免授权完成后跳到未登记来源。

## 可选容器化 Nginx

默认 `compose.yml` 不启动 Nginx，避免和当前宿主机 Nginx 抢占 `80/443`。迁移到新机器或决定把入口层切进 Compose 时，再叠加 `compose.nginx.yml`：

```bash
cd server/deploy/compose/prod
docker compose -f compose.yml -f compose.nginx.yml --env-file .env up -d nginx
```

容器化 Nginx 使用官方 `nginx:1.27-alpine` 镜像，不构建自定义镜像；配置通过 `nginx/` 目录挂载，便于迁移时跟随仓库复制。当前配置包含：

- `oauth-api.saurick.me` HTTPS 主入口，反代到 Compose 内部 `app-server:8400`。
- `/.well-known/acme-challenge/` HTTP-01 challenge webroot。
- 旧域名 `oauth-api.saurick.space`、`openai.saurick.space` 到 `oauth-api.saurick.me` 的跳转样本。
- `proxy_read_timeout 700s` / `proxy_send_timeout 700s`，给 app-server 与 Codex 上游 600 秒等待窗口留余量。

启用前必须准备证书和 ACME webroot 目录：

```bash
mkdir -p /data/openai-oauth-api-service/nginx/certs
mkdir -p /data/openai-oauth-api-service/nginx/acme
```

证书目录默认按域名放置：

```text
/data/openai-oauth-api-service/nginx/certs/
├── oauth-api.saurick.me/
│   ├── fullchain.pem
│   └── privkey.pem
├── oauth-api.saurick.space/
│   ├── fullchain.pem
│   └── privkey.pem
└── openai.saurick.space/
    ├── fullchain.pem
    └── privkey.pem
```

如果新机器只保留当前主域，可以先删除或注释 `nginx/conf.d/oauth-api.conf` 里的旧域名 HTTPS redirect server block，避免缺少旧域名证书导致 Nginx 启动失败。HTTP 旧域名跳转不依赖证书。

从宿主机 Nginx 切到容器 Nginx 时建议按顺序执行：

```bash
# 1. 先用非标准端口验证容器 Nginx，不影响当前 80/443
NGINX_HTTP_PORT=8080 NGINX_HTTPS_PORT=8443 \
  docker compose -f compose.yml -f compose.nginx.yml --env-file .env up -d nginx

# 2. 验证容器配置和内部反代
docker exec openai-oauth-api-service-nginx nginx -t
curl -kI https://oauth-api.saurick.me:8443/healthz --resolve oauth-api.saurick.me:8443:127.0.0.1

# 3. 通过后再停宿主机 Nginx，并用 80/443 重新拉起容器入口
systemctl stop nginx
docker compose -f compose.yml -f compose.nginx.yml --env-file .env up -d nginx
```

若需要回滚，停止容器 Nginx 后恢复宿主机 Nginx：

```bash
docker compose -f compose.yml -f compose.nginx.yml --env-file .env stop nginx
systemctl start nginx
```

## 说明

- Compose 主路径不包含 Jaeger，`TRACE_ENDPOINT` 默认为空；后续需要 tracing 时接入外部 OTLP endpoint。
- 真实 `.env`、认证信息、代理配置和数据库密码不得提交到仓库。
- 当前不保留远端 SSH 发布脚本；发布流程确定后再补自动化脚本。
