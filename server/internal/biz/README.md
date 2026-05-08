# Biz

`biz` 层承载业务规约和 usecase，不直接关心 HTTP / gRPC / JSON-RPC 传输细节，也不直接操作数据库驱动。

当前项目默认保留的通用能力：

- 管理员账号密码登录
- 当前登录态查询
- 后台账号目录
- 启用 / 禁用账号

职责边界：

- 输入：来自 `service` 层整理好的参数
- 输出：面向业务的结果与错误
- 依赖：通过接口依赖 `data` 层 repo

补充说明见：

- `server/docs/api.md`
- `server/docs/observability.md`
