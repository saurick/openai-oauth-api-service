# 运维说明

## 本地开发

```bash
cd web
pnpm install
pnpm start
```

```bash
cd server
make init
make run
```

本地访问：

- 后端：`http://127.0.0.1:8400`
- 前端：Vite 默认监听 `http://localhost:5176`；该端口只用于本地开发，与生产 Compose 端口无关
- 管理登录：`/admin-login`
- API 运营控制台：`/admin-api`

## 配置

开发环境私有配置复制后填写：

```bash
cp server/configs/dev/config.local.example.yaml \
  server/configs/dev/config.local.yaml
```

生产 Compose 配置复制后填写：

```bash
cd server/deploy/compose/prod
cp .env.example .env
```

真实密钥不要提交到仓库。关键密钥包括：

- `CODEX_HOST_HOME` 指向的 Codex 登录态目录
- `OAUTH_API_JWT_SECRET`
- `POSTGRES_DSN`

API 上游统一使用服务器 Codex CLI 登录态：

- app-server 容器内调用 Codex CLI，并通过 `CODEX_HOST_HOME` 挂载服务器上的 Codex 登录态。
- 多台客户端仍只使用本系统签发的 `ogw_...` 下游 key。

当前个人部署的管理员账号默认保持 `admin/adminadmin`。不要在部署时擅自生成或替换 `OAUTH_API_ADMIN_PASSWORD`；只有维护者明确要求改密时才调整该变量并重启服务。

管理员 OAuth 登录默认关闭。启用 Google/OIDC 时，Google Console 的本地回调登记后端固定地址 `http://localhost:8400/auth/oauth/callback`，不要再登记 Vite 端口；服务端会把当前前端 origin 写入 signed state，并在授权完成后动态跳回当前前端端口。生产环境需额外设置 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS` 为管理后台 HTTPS 域名。

开发环境可在 `server/.env` 设置 `DB_URL`，Makefile 会自动映射到 `POSTGRES_DSN`。本地联调数据库名建议使用 `openai_oauth_api_service`，真实密码只保存在本地忽略文件中。

## 部署主路径

当前只保留 Docker Compose 主路径：

```bash
cd server/deploy/compose/prod
docker compose -f compose.yml up -d
```

Kubernetes、dashboard、lab-ha 和远端 SSH 发布脚本已从本项目主路径裁剪。后续需要时按真实环境新增，避免使用旧占位清单。

## 检查命令

```bash
bash scripts/doctor.sh
bash scripts/qa/fast.sh
bash scripts/qa/full.sh
```

## Codex runtime 健康检查

133 低配服务器上的 Codex runtime 更新由仓库内运维脚本 + systemd timer 管理。按当前个人部署要求，日常 timer 会检查 `@openai/codex` latest，发现新版本就安装 latest，然后执行健康检查。

安装：

```bash
bash scripts/ops/install-codex-runtime-health-check.sh
```

默认每天按 Asia/Shanghai 05:00 固定执行一次，不跟随服务器本地时区漂移：

```bash
/usr/local/sbin/codex-runtime-health-check.py --auto-upgrade
```

检查内容包括：

- `codex --version`；默认 `CODEX_RUNTIME_MODE=auto`，宿主机无 `codex` 时改查 app-server 容器内的 `codex`
- `/healthz`、`/readyz`
- `/public/codex/balance`，其中 `stale=true` 记为 warning
- `openai-oauth-api-service-server` 容器运行状态
- Codex 上游代理 failover 配置检查
- 根分区磁盘余量

结果写入：

```bash
/var/lib/codex-runtime-health/state.json
/var/log/codex-runtime-health.log
```

默认会按当前运行位置自动选择版本查询和升级方式：

```bash
docker exec openai-oauth-api-service-server npm view @openai/codex version
docker exec openai-oauth-api-service-server npm install -g @openai/codex@latest
```

如需覆盖安装方式，可配置自定义命令：

```bash
CODEX_RUNTIME_LATEST_VERSION_COMMAND='npm view @openai/codex version'
CODEX_RUNTIME_UPGRADE_COMMAND='npm install -g @openai/codex@latest'
```

不同安装方式的服务器迁移时，只需要调整 `CODEX_RUNTIME_MODE`、`CODEX_RUNTIME_BIN`、`CODEX_RUNTIME_LATEST_VERSION_COMMAND` 和 `CODEX_RUNTIME_UPGRADE_COMMAND`，不需要修改 app-server 代码。当前 133 Codex 随 app-server 容器运行，timer 会升级运行中容器内的 Codex；后续 app-server 镜像重建后，仍以镜像内版本为启动基线，再由 timer 拉到 latest。

后端：

```bash
cd server
go test ./...
```

前端：

```bash
cd web
pnpm lint
pnpm css
pnpm test
```
