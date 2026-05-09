## 归档索引
- 2026-05-10 之前历史流水：`docs/archive/progress-2026-05-10-pre-docker-cleanup-constraint.md`。
- 当前文件保留 2026-05-10 以来新增记录；归档文件只作追溯线索，不作为当前正式需求真源。

## 2026-05-10 00:30
- 完成：补充 `AGENTS.md` 的多项目低配 Docker 宿主机发布后清理约束，明确发布完成、健康检查和必要回归通过后，只清理未被任何容器使用的旧镜像与构建缓存，优先使用 `docker image prune -a -f` 与 `docker builder prune -f`，并禁止清理 volume、数据库目录、compose `.env`、上传目录或运行中容器依赖镜像；同步给 `legacy-python-mvp/AGENTS.md` 加入轻量版同类约束。更新前因 `progress.md` 超过归档阈值，已归档旧流水。
- 下一步：如后续继续完善发布脚本，可把该约束落为脚本级 post-deploy cleanup，并在执行前后输出磁盘与容器状态。
- 阻塞/风险：本轮只更新协作与部署约束文档，未修改运行代码、Compose 配置或线上服务；旧镜像清理仍需在发布脚本中显式实现。
