# 部署说明

当前项目只保留 Docker Compose 作为主部署路径。

部署构建边界：目标服务器配置较低，只负责导入已经构建好的镜像、启动 Compose、执行 migration 和部署后检查；不要在服务器上执行 `docker build`、`pnpm build`、`go build`、`make build_server` 等重构建步骤。镜像必须在本地开发机或 CI 构建完成后，再上传到服务器。

Atlas migration 在生产 / 低配服务器上统一使用宿主机 `/usr/local/bin/atlas`，不要拉起 `arigaio/atlas:*` 临时容器，也不要把 Atlas 增加到 Compose。迁移目录随 release 上传，执行时使用宿主机可达的 PostgreSQL DSN（当前 Compose 默认是 `127.0.0.1:5433`），并通过 `flock /tmp/atlas-migrate.lock` 避免并发迁移。

| 路径 | 说明 |
| --- | --- |
| `compose/prod/compose.yml` | PostgreSQL + 后端服务 |
| `compose/prod/compose.nginx.yml` | 可选容器化 Nginx 入口层，迁移或切入口时叠加启用 |
| `compose/prod/.env.example` | 生产环境变量示例 |
| `compose/prod/README.md` | Compose 运行说明 |

Kubernetes、dashboard、lab-ha 和远端 SSH 发布脚本已经从主路径裁剪。后续如果有明确集群、镜像仓库和域名，再按真实环境新增。

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

当前线上仍使用宿主机 Nginx；容器化 Nginx 是迁移/切入口能力，不是默认启动项。
