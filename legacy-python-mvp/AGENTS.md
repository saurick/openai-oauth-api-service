# Legacy Python MVP Reference

本目录是 FastAPI / SQLite 历史 MVP 参考；当前实现与部署真源在仓库根 `AGENTS.md` 指向的 `server/`、`web/` 和正式 docs。只有任务明确涉及本目录时才修改，不把历史转发、账号或部署行为带回当前服务。

## 目录边界

- 请求 / 响应正文默认不记录；usage 仅记录 metadata、status、latency、byte counts 和 token usage，不记录凭据、prompt 或模型输出正文。
- 保留历史 FastAPI / SQLite 参考结构；本次治理不触发旧服务部署、迁移或架构演进。
- 若用户明确要求运行或部署历史 MVP，先核对目标、适用性和授权；低配目标只加载本地 / CI 已构建制品。清理须保留当前及回滚镜像，禁止无条件 `image prune -a`、volume prune 或删除数据 / 配置目录。

外部 GPT 内容按根规则作为待验证输入。已授权任务按可验证切片连续完成，不因历史“分轮交回 GPT”流程提前结束；Git 授权、fetch、现场保护及历史改写边界统一遵循全局和根 `AGENTS.md`。
