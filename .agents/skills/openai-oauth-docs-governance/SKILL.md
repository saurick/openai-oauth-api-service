---
name: openai-oauth-docs-governance
description: 项目文档治理（openai-oauth-api-service）。Use to maintain service architecture, operations, deployment, README, AGENTS, and progress docs.
---

# OpenAI OAuth Docs Governance

维护本服务文档；OpenAI 官方产品 / API 问题使用当前可发现的官方文档技能，不把本仓库说明当官方结论。

## Read the Relevant Truth

- 先读 `AGENTS.md`、相关 README / `docs/README.md`，用 `GIT_OPTIONAL_LOCKS=0` 核对 scoped diff 并保留外部改动。
- OAuth、API key、usage、upstream、日志 / 保留策略和管理端语义看 `docs/architecture.md`；运行与部署看 `docs/operations.md`、`server/README.md`、`server/deploy/README.md` 和 Compose 真源。
- 页面文案 / 质量命令看 `web/README.md`、页面 / 路由代码和 `web/scripts/styleL1.mjs`；后端行为核对 service / biz / data、Ent schema、migration 和测试。
- `legacy-python-mvp/`、progress 和 archive 是历史 / 过程证据，不能覆盖当前实现。只读任务相关分支，已读且未变化的内容不重复加载。

## Preserve Boundaries

- 用户明确要求长期规则治理时可改 AGENTS，普通文档维护不改政策。必要行为改动在已有授权范围内进入领域 / operations 流程继续，新增范围才询问。
- 不写真实 tokens、JWT / OAuth secrets、DB 密码、含凭据登录路径或生产 `.env` 值。usage 描述基于 `gateway_usage_logs` 等真实记录，不宣称默认保存请求正文、prompt 或模型输出。
- 默认个人部署管理员密码口径、secret / logging 和低配发布约束遵循 AGENTS 与正式部署文档；低配目标只加载本地 / CI 制品并迁移 / smoke，不现场构建。

## Write and Sync

- 按读者区分开发、管理员操作、部署和排障入口；结论、范围、主路径和命令前置，具体章节可跳转。比较用表格、步骤用编号、配置用代码块，复杂关系才用 Mermaid。
- 同一合同集中维护；保留稳定英文文档路径，不套用其他项目中文命名或 inventory。metadata / frontmatter 只在实际消费者需要时增加。
- 文档增删 / 改名 / 职责调整时同步 `docs/README.md`、附近 README、锚点与引用；行为、命令、key / usage / upstream、OAuth callback 或部署口径变化时同步对应专题和消费者。
- usage 可见性变化按需核对 dashboard 与 `/admin-usage`。progress 仅按 AGENTS 触发条件维护，保留外部内容；纯全局 Skill 改动不更新项目记录。

## Validate and Report

运行 `git diff --check`、定向路径 / 命令 / 术语扫描；Skills 运行 validator 与元数据 / 引用检查。文档改变实际脚本或页面合同才运行对应测试，纯治理不运行 migration 或全量 QA。

说明关键修改、AGENTS 是否变化、必要同步、验证与盲区；不用未触达事项填满报告。
