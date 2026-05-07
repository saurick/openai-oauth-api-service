# server 后端说明

## 技术栈

- Kratos
- Ent + Atlas
- PostgreSQL
- OpenTelemetry（可选）

## 架构分层

执行链路：`server -> service -> biz -> data`

- `server`：协议接入层（HTTP/gRPC/JSON-RPC）
- `service`：接口适配层（DTO 转换与调用编排）
- `biz`：业务规约与 UseCase
- `data`：数据库/缓存/外部依赖访问

## 快速开始

```bash
cd server
make init
make run
```

## 常用命令

```bash
# 代码生成
make api
make all

# 数据模型与迁移
make data
make migrate_apply
make print_db_url
make migrate_status

# 测试与构建
go test ./...
make build
```

## 数据库迁移说明

- `make migrate_apply` 默认优先读取 `server/configs/dev/config.yaml`，并允许 `config.local.yaml` 覆盖私有 DSN。
- 本地 `server/.env` 可设置 `DB_URL`，Makefile 会自动映射为 `POSTGRES_DSN`，用于运行和迁移命令。
- 可先执行 `make print_db_url` 确认当前真正命中的开发库；该命令默认只输出脱敏 DSN。
- `server/cmd/dburl` 只是迁移辅助命令，用来统一解析当前仓库默认 DSN，不属于服务运行时入口。

## 目录结构（简版）

```text
server/
├── api/
├── cmd/
├── configs/
├── internal/
│   ├── biz/
│   ├── data/
│   ├── server/
│   └── service/
├── pkg/
└── Makefile
```

| 路径 | 职责 |
| --- | --- |
| `api/` | 协议定义与生成入口，目前包含 JSON-RPC 相关接口描述 |
| `cmd/` | 服务启动、迁移辅助、排障与运维命令入口 |
| `configs/` | 按环境拆分的配置文件 |
| `internal/server/` | HTTP/gRPC/JSON-RPC 接入、中间件与路由装配 |
| `internal/service/` | 接口适配层，负责 DTO 转换与调用编排 |
| `internal/biz/` | 业务规约与 UseCase 真源 |
| `internal/data/` | 数据访问、外部依赖与持久化实现 |
| `internal/conf/` | 配置结构定义与加载相关代码 |
| `internal/errcode/` | 服务端错误码目录真源 |
| `pkg/` | 可复用基础设施组件，如日志、JWT、任务编排与 Telegram 辅助 |
| `deploy/` | Compose 部署配置 |
| `docs/` | 后端专题文档索引与 runbook |
| `third_party/` | 第三方 proto / OpenAPI 依赖 |

## 文档索引

- 文档索引：`server/docs/README.md`
- 部署模板：`server/deploy/README.md`
- 运行说明：`server/docs/runtime.md`
- 配置说明：`server/docs/config.md`
- API 说明：`server/docs/api.md`
- 可观测性：`server/docs/observability.md`
- Ent / Atlas：`server/docs/ent.md`
- DB 工作流：`server/internal/data/AI_DB_WORKFLOW.md`
- 业务层说明：`server/internal/biz/README.md`
- 数据层说明：`server/internal/data/README.md`
- 服务层说明：`server/internal/service/README.md`

## 部署

- Docker Compose 主路径：`server/deploy/compose/prod`
- Kubernetes 与远端 SSH 发布脚本当前不在主路径；需要时按真实环境新增。
- 部署说明优先看 `server/deploy/README.md`
