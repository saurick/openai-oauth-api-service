---
name: openai-oauth-operations-governance
description: 项目运行与发布治理（openai-oauth-api-service）。Use when Codex diagnoses 502, quota, balance, usage, gateway, upstream or container failures; changes logs, stale fallback or error handling; handles keys and tokens; or plans low-spec releases, migrations, health checks, smoke tests, and rollback.
---

# OpenAI OAuth 运行与发布治理 / Operations Governance

## Truth Chain / 必读真源

- 先读 `AGENTS.md`、`README.md`、`server/README.md`、`server/deploy/README.md` 和相关 operations 文档。
- 核对当前 commit/image、container、config、DB/migration、gateway/upstream 响应、`gateway_usage_logs` 和 request/session evidence。

## Project Rules / 项目边界

- 502、429、balance、usage 和 stale 结果先区分 gateway、upstream、credential、quota、DB、container 和 deploy 层。
- API keys、OAuth tokens、upstream credentials、admin access 和 request logs 默认敏感，禁止在日志、文档或回复中泄露。
- `stale=true`、cache、fallback 和 degraded 必须诚实标记来源、时间和原因，不伪装实时成功。
- 133 低配发布在本地构建镜像并上传 tar；远端只 load、migration、启动、health/ready/admin smoke。
- 发布证据绑定 commit、`APP_IMAGE`、migration、目标容器、health/ready、真实主路径和 rollback point。

## Workflow / 工作流

1. 明确 diagnose、observe、secure、release 或 rollback 目标和环境。
2. 保存 request/session id、响应分类、latency、upstream、容器日志、DB 和版本证据并脱敏。
3. 最小复现并定位失败层；只在根因属于代码时修改代码。
4. 发布前检查 worktree/upstream、测试、镜像、migration、env 和回滚点；Git 收口搭配 `$git-closeout-coordination`。
5. 目标机 load 制品、apply migration、启动并验证 health/ready、admin、balance/usage 或本轮真实链路。
6. 同步 operations/deploy 文档和 `progress.md`，记录未验证的真实上游或生产盲区。

## Validation / 验证要求

- 诊断保留脱敏的请求、日志、数据库或容器证据。
- 错误与观测性覆盖 upstream failure、latency、stale/fallback 和用户可见状态。
- 发布记录 image、migration、health/ready、业务 smoke、rollback 和磁盘/镜像边界。

## Output / 输出要求

汇报失败层或发布结果、证据、脱敏策略、目标版本、验证、回滚点和剩余风险。
