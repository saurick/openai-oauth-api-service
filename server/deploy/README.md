# 部署说明

当前项目只保留 Docker Compose 作为主部署路径。

部署构建边界：目标服务器配置较低，只负责导入已经构建好的镜像、启动 Compose、执行 migration 和部署后检查；不要在服务器上执行 `docker build`、`pnpm build`、`go build`、`make build_server` 等重构建步骤。镜像必须在本地开发机或 CI 构建完成后，再上传到服务器。

| 路径 | 说明 |
| --- | --- |
| `compose/prod/compose.yml` | PostgreSQL + 后端服务 |
| `compose/prod/.env.example` | 生产环境变量示例 |
| `compose/prod/README.md` | Compose 运行说明 |

Kubernetes、dashboard、lab-ha 和远端 SSH 发布脚本已经从主路径裁剪。后续如果有明确集群、镜像仓库和域名，再按真实环境新增。

## 快速启动

```bash
cd server/deploy/compose/prod
cp .env.example .env
# 编辑 .env，至少替换数据库密码、JWT 密钥、管理员密码和 OPENAI_API_KEY。
docker compose -f compose.yml up -d
```
