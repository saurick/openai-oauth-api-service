# Compose 部署

本目录是 OpenAI OAuth API Service 当前主部署路径，包含 PostgreSQL 和后端服务。

## 文件

| 文件 | 说明 |
| --- | --- |
| `compose.yml` | PostgreSQL + app-server |
| `compose.nginx.yml` | 可选容器化 Nginx 入口层，迁移或切入口时叠加启用 |
| `compose.certbot.yml` | 可选项目级 Certbot，单项目交付或独占机器部署时按需执行 |
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
CODEX_CLI_TIMEOUT_SECONDS=28800
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

## 测试服务器共存

本项目的 Compose 交付结构按单项目机器设计，但当前测试服务器上多个项目只是临时共存，不作为项目部署真源。共存时优先让各项目只启动 `compose.yml`，通过不同 `APP_HTTP_PORT` 暴露服务，或由宿主机 Nginx 做临时反代；不要为了测试机共存把共享入口层、共享证书续签或端口抢占规则沉淀进项目主路径。

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
- `proxy_read_timeout 28900s` / `proxy_send_timeout 28900s`，给 app-server 与 Codex 上游 28800 秒等待窗口留余量。
- `client_max_body_size 90m`，与 app-server 的 OpenAI-compatible data URL 附件请求体上限保持一致。

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

## 可选项目级 Certbot

证书申请与续签默认不放入主 `compose.yml`。如果甲方机器已有宿主机 Nginx / Certbot 运维体系，优先沿用宿主机 Certbot，并把证书路径挂载给容器 Nginx 或直接由宿主机 Nginx 反代本项目。

如果是单项目交付、机器由本项目独占，且希望证书管理跟随项目目录，可以叠加 `compose.certbot.yml` 执行一次性 certbot 命令。该服务带 `certbot` profile，不会随普通 `docker compose up -d` 自动启动。

首次申请证书前，必须先有一个能响应 HTTP-01 challenge 的入口。若甲方机器已有宿主机 Nginx，推荐直接使用宿主机 Nginx 提供 `/.well-known/acme-challenge/`，证书签发后再决定是否复制给容器 Nginx。

如果要用本项目容器 Nginx 完成首次签发，需要先放入临时自签证书让当前 HTTPS server block 能启动；签发成功后再切到 Certbot 的真实证书目录。新机器如果不保留旧域名，先删除或注释旧域名 HTTPS redirect server block，可以少准备旧域名临时证书。

```bash
mkdir -p /data/openai-oauth-api-service/letsencrypt
mkdir -p /data/openai-oauth-api-service/certbot/work
mkdir -p /data/openai-oauth-api-service/certbot/logs
mkdir -p /data/openai-oauth-api-service/nginx/acme
for domain in oauth-api.saurick.me oauth-api.saurick.space openai.saurick.space; do
  mkdir -p "/data/openai-oauth-api-service/nginx/certs/$domain"
  openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
    -subj "/CN=$domain" \
    -keyout "/data/openai-oauth-api-service/nginx/certs/$domain/privkey.pem" \
    -out "/data/openai-oauth-api-service/nginx/certs/$domain/fullchain.pem"
done

docker compose -f compose.yml -f compose.nginx.yml --env-file .env up -d nginx

docker compose -f compose.certbot.yml --env-file .env --profile certbot run --rm certbot \
  certonly --webroot \
  -w /var/www/acme \
  -d oauth-api.saurick.me \
  --email admin@example.com \
  --agree-tos \
  --no-eff-email
```

Certbot 默认产物在 `${CERTBOT_CONFIG_DIR}/live/<domain>/`。如果让容器 Nginx 直接读取这套产物，应在 `.env` 中把证书目录调整为：

```bash
CERTBOT_CONFIG_DIR=/data/openai-oauth-api-service/letsencrypt
NGINX_CERTS_DIR=/data/openai-oauth-api-service/letsencrypt/live
```

调整后重建 Nginx，使其读取正式证书：

```bash
docker compose -f compose.yml -f compose.nginx.yml --env-file .env up -d nginx
```

续签时执行：

```bash
docker compose -f compose.certbot.yml --env-file .env --profile certbot run --rm certbot \
  renew --webroot -w /var/www/acme

docker exec openai-oauth-api-service-nginx nginx -s reload
```

如果入口层仍是宿主机 Nginx，续签后改为执行：

```bash
systemctl reload nginx
```

## 说明

- Compose 主路径不包含 Jaeger，`TRACE_ENDPOINT` 默认为空；后续需要 tracing 时接入外部 OTLP endpoint。
- 真实 `.env`、认证信息、代理配置和数据库密码不得提交到仓库。
- 当前不保留远端 SSH 发布脚本；发布流程确定后再补自动化脚本。
