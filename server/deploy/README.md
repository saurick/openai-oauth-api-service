# 部署说明

当前项目只保留 Docker Compose 作为主部署路径。

部署构建边界：目标服务器配置较低，只负责导入已经构建好的镜像、启动 Compose、执行 migration 和部署后检查；不要在服务器上执行 `docker build`、`pnpm build`、`go build`、`make build_server` 等重构建步骤。镜像必须在本地开发机或 CI 构建完成后，再上传到服务器。

Atlas migration 在生产 / 低配服务器上统一使用宿主机 `/usr/local/bin/atlas`，不要拉起 `arigaio/atlas:*` 临时容器，也不要把 Atlas 增加到 Compose。迁移目录随 release 上传，执行时使用宿主机可达的 PostgreSQL DSN（当前 Compose 默认是 `127.0.0.1:5433`），并通过 `flock /tmp/atlas-migrate.lock` 避免并发迁移。

| 路径 | 说明 |
| --- | --- |
| `compose/prod/compose.yml` | PostgreSQL + 后端服务 |
| `compose/prod/compose.nginx.yml` | 可选容器化 Nginx 入口层，迁移或切入口时叠加启用 |
| `compose/prod/compose.certbot.yml` | 可选项目级 Certbot，单项目交付或独占机器部署时按需执行 |
| `compose/prod/.env.example` | 生产环境变量示例 |
| `compose/prod/README.md` | Compose 运行说明 |

Kubernetes、dashboard、lab-ha 和远端 SSH 发布脚本已经从主路径裁剪。后续如果有明确集群、镜像仓库和域名，再按真实环境新增。

## 宿主机 Codex runtime 检查

Codex runtime 属于宿主机运维依赖，不由 app-server 业务进程负责升级。仓库提供 systemd timer 安装脚本，迁移服务器时随 deploy 文件一起复制即可：

```bash
bash scripts/ops/install-codex-runtime-health-check.sh
```

默认 timer 每天 Asia/Shanghai 05:00 固定执行 `/usr/local/sbin/codex-runtime-health-check.py --check`，检查 `codex --version`、`/healthz`、`/readyz`、`/public/codex/balance`、容器状态、failover 配置和磁盘余量，并把结果写入 `/var/lib/codex-runtime-health/state.json`。`CODEX_RUNTIME_MODE=auto` 会先查宿主机 `codex`，宿主机没有时改查 app-server 容器内的 `codex`。

升级不自动执行；如确需升级，先按当前服务器的 Codex 安装方式配置 `CODEX_RUNTIME_UPGRADE_COMMAND`，再手动运行。若 Codex 随 app-server 镜像运行，升级应走镜像发布，不要在容器内临时升级后当作持久变更：

```bash
CODEX_RUNTIME_UPGRADE_COMMAND='npm install -g @openai/codex@latest' \
  /usr/local/sbin/codex-runtime-health-check.py --upgrade
```

## 快速启动

```bash
cd server/deploy/compose/prod
cp .env.example .env
# 编辑 .env，至少替换数据库密码和 JWT 密钥。
# API 上游统一使用服务器 Codex CLI 登录态，部署时挂载 CODEX_HOST_HOME。
# 管理员账号默认保持 admin/adminadmin；不要在部署时擅自生成或替换管理员密码。
docker compose -f compose.yml up -d
```

如需让 Nginx 也跟随 Compose 迁移，可在准备好证书目录和 ACME webroot 后叠加启用：

```bash
docker compose -f compose.yml -f compose.nginx.yml --env-file .env up -d nginx
```

当前线上仍使用宿主机 Nginx；容器化 Nginx 是迁移/切入口能力，不是默认启动项。若继续使用宿主机 Nginx，入口层的 `client_max_body_size` 必须与 app-server 和 `compose/prod/nginx/` 样例保持一致，否则大图片 / PDF data URL 请求会先在入口层返回 413。

证书申请与续签不放入主 `compose.yml`。测试服务器上多个项目只是临时共存时，优先使用不同端口或宿主机 Nginx 临时转发；交付到单项目机器时，可选择宿主机 Certbot，也可叠加项目级 `compose.certbot.yml` 管理证书。
