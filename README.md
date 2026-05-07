# OpenAI OAuth API Service

OpenAI OAuth API Service 是一个长期维护的 OAuth 登录、OpenAI 兼容 API 转发与 token/usage 统计管理项目。系统负责接入合规 OAuth/OIDC 登录，用官方 OpenAI API key 作为上游凭据，向下游签发独立 API key，并统一记录 usage、状态码、延迟、字节数和 token 用量。

## 边界

| 类型 | 说明 |
| --- | --- |
| 支持 | 官方 OpenAI API key、Project API key、Service Account key |
| 支持 | OAuth/OIDC 登录接入、本系统 JWT 签发与账号管理 |
| 支持 | 下游 API key 签发、吊销、配额、usage 监控 |
| 支持 | OpenAI 兼容 API 转发，例如 `/v1/responses`、`/v1/chat/completions` |
| 支持 | 统一上游代理、结构化日志、健康检查、Compose 部署 |
| 不支持 | 抓取或分享 Codex / ChatGPT 登录态、Cookie、设备码、个人账号 token |
| 不支持 | 把个人订阅账号包装成多人共享 API |

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

默认地址：`http://localhost:5173`

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

API 转发的上游 OpenAI 凭据应使用环境变量或 Secret 注入，例如：

```bash
OPENAI_API_KEY=sk-proj-...
OPENAI_BASE_URL=https://api.openai.com/v1
UPSTREAM_PROXY_URL=socks5://127.0.0.1:7890
```

管理后台入口：

- 管理登录：`/admin-login`
- API 运营控制台：`/admin-api`

开发配置默认会初始化管理员账号 `admin/adminadmin`；共享或部署前必须替换默认管理员密码和 JWT secret。

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
