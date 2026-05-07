# Data

`data` 层负责数据库、外部依赖和 JSON-RPC 入口聚合，不承载业务决策本身。

当前项目默认保留：

- PostgreSQL 初始化与重试
- Ent ORM 访问
- 用户 / 管理员鉴权 repo
- 后台账号目录 repo
- JSON-RPC 入口聚合

数据库变更前，必须先读：

- [`AI_DB_WORKFLOW.md`](./AI_DB_WORKFLOW.md)

补充说明见：

- `server/docs/ent.md`
- `server/docs/config.md`
- `server/docs/observability.md`
