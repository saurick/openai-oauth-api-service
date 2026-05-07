# server/docs 文档索引

`server/docs` 只保留后端真正需要的默认说明，目标是让模板初始化后，开发、部署和排障都能有统一入口。

## 建议阅读顺序

1. `server/README.md`
2. `server/docs/runtime.md`
3. `server/docs/config.md`
4. `server/docs/api.md`
5. `server/docs/observability.md`
6. `server/docs/ent.md`
7. `server/docs/k8s.md`

## 文档说明

- `runtime.md`
  - 服务如何启动
  - 默认端口、静态资源和健康检查
  - 本地开发最常用命令
- `config.md`
  - `server/configs/*/config.yaml` 的字段说明
  - 初始化新项目后必须替换的配置项
- `api.md`
  - 当前项目默认暴露的 JSON-RPC 入口
  - 默认保留的方法和鉴权边界
- `observability.md`
  - 当前日志、trace、健康检查基线
  - 已知盲区和后续业务常见补法
- `ent.md`
  - Ent + Atlas 数据模型和迁移工作流
- `k8s.md`
  - Kubernetes 部署模板的目录、占位符和使用方式

## 相关入口

- 服务端总览：`server/README.md`
- 部署模板总览：`server/deploy/README.md`
- 数据库变更工作流：`server/internal/data/AI_DB_WORKFLOW.md`
