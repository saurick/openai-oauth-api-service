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
- 前端：Vite 默认监听 `http://localhost:5175`；如果端口被占用，按终端输出的新端口访问
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

- `OPENAI_API_KEY`
- `CODEX_HOST_HOME` 指向的 Codex 登录态目录
- `OAUTH_API_JWT_SECRET`
- `POSTGRES_DSN`
- 代理认证信息

上游有两种模式：

- `OAUTH_API_UPSTREAM_PROVIDER=openai_api`：默认模式，服务端使用 `OPENAI_API_KEY` 调用 OpenAI 兼容上游。
- `OAUTH_API_UPSTREAM_PROVIDER=codex_cli`：个人统一出口模式，app-server 容器内调用 Codex CLI，并通过 `CODEX_HOST_HOME` 挂载服务器上的 Codex 登录态。该模式下多台客户端仍只使用本系统签发的 `ogw_...` 下游 key。

当前个人部署的管理员账号默认保持 `admin/adminadmin`。不要在部署时擅自生成或替换 `OAUTH_API_ADMIN_PASSWORD`；只有维护者明确要求改密时才调整该变量并重启服务。

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
