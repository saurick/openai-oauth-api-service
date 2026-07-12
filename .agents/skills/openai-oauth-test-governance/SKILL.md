---
name: openai-oauth-test-governance
description: 项目测试治理（openai-oauth-api-service）。Use when choosing, running, reviewing, or explaining Go, web, admin UI, auth/key/quota/usage, migration, secret, or deploy validation.
---

# OpenAI OAuth Test Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这份 skill 把 openai-oauth-api-service 的验证范围落到真实风险：鉴权、API key、额度、usage logging、Codex backend、上游失败、管理端页面、migration、低配部署和 secrets。

## OpenAI OAuth 测试质量门禁 Test Quality Gate

测试治理不是跑得越多越好，也不是用一个全量命令替代关键场景。

### 结构质量检查 Structure Quality Checks

- 边界清晰、合理严谨：说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 语义清晰：测试名称、fixture、断言和报告必须说明验证的业务语义、合同和证据环境，避免只证明命令跑过。
- 职业任务文案：涉及用户可见文案时，测试或审查要覆盖业务岗位语言和裸工程术语泄漏，不只断言字符串存在。
- 模块化：测试按单元、集成、契约、浏览器回归、发布验证等风险层分工，不用一个大命令掩盖关键断言缺失。
- 高内聚：同一业务规则的样本、fixture、helper 和断言尽量收口，避免不同测试文件维护近似但冲突的口径。
- 低耦合：测试不依赖脆弱顺序、真实外部服务或无关全局状态；需要真实环境时显式声明 target 和证据。
- 单一职责：每个测试说明它证明什么；验证报告区分已覆盖、未覆盖和不适合本轮覆盖的风险。

- 按改动影响面选择最小必要验证组合；docs/skill-only 不机械跑全量，业务真源、RBAC、migration、页面交互或发布链路必须升级验证。
- 测试要覆盖本轮最可能出错的合同、状态、权限、旧数据、边界值、浏览器状态或目标环境；不能只证明 happy path。
- 测试通过不能替代业务边界、字段真源、客户/模板差异、可维护性、可回滚性和文档同步判断。
- 最终必须说明验证层级、测试形态、证据环境、未跑项和剩余盲区，避免“已通过测试”被误读成全系统已验收。

## Workflow

1. 先判断改动触达 server、web、migration、deploy、proxy/upstream、Codex backend、管理端页面或文档。
2. 读取相关真源：`README.md`、`AGENTS.md`、`server/README.md`、`web/README.md`、`scripts/README.md`，部署任务再读 `server/deploy/README.md`。
3. 按风险选最小充分命令；不要把 live upstream 调用当稳定单元测试。
4. 涉及线上或低配服务器时，本地/CI 构建，远端只做加载制品、migration、启动、健康检查和 smoke。
5. 汇报命令、结果、未覆盖项；有正式改动时更新 `progress.md`。

## Test Shapes

| 类型 | 适用场景 | 常用命令 / 验证 |
| --- | --- | --- |
| Static / Guard | 任意改动、配置、脚本、密钥边界 | `git diff --check`、`bash scripts/qa/secrets.sh`、`shellcheck`、`shfmt` |
| Go Unit / Integration | OAuth、API key、quota、usage、gateway、Codex backend | `go test ./...` 或定向 server 包 |
| Web Unit | 管理端组件、错误码、auth 分类、表格交互 | `cd web && pnpm lint`、`pnpm test` |
| Admin UI Regression | 管理端登录、dashboard、usage、admins、layout/style | `cd web && pnpm style:l1`，必要时设置 `STYLE_L1_SCENARIOS=...` |
| Migration / DB | schema、migration、usage log、quota 表结构 | `make migrate_status`、`make migrate_apply` 前先确认目标 DB |
| Deploy Preflight | Compose、低配发布、运行时配置 | `bash scripts/qa/production-preflight.sh --runtime`，再做 health/ready |
| Full / Strict | 跨层改动、提交前、发布前 | `bash scripts/qa/fast.sh`、`bash scripts/qa/full.sh`、`bash scripts/qa/strict.sh` |

## Selection Rules

- 鉴权、API key、余额、quota、usage logging 或 error classification 改动必须覆盖正常路径、权限失败、额度不足、禁用/过期、上游失败和日志字段。
- Codex backend / direct API / CLI fallback 相关改动要区分可控 fake/local 测试与真实上游 smoke；真实上游不作为 deterministic 单元测试。
- 管理端指标、表格、筛选、详情页或登录态改动必须跑对应 web 测试和 `style:l1` 场景。
- Secrets、prompt、token、API key、session 和日志改动必须跑 secrets/grep 类检查，最终回复不得泄漏值。
- 低配部署验证必须包含 `/healthz`、`/readyz`、migration 状态、容器状态和必要业务 smoke；禁止在低配服务器直接重构建。
- 如果只改文档或 skill，做文档/skill 校验即可，不机械跑 live gateway 或部署 smoke。

## Reporting Standard

最终回复必须写清：

- 本轮覆盖了 server、web、migration、deploy、upstream 中哪些层。
- 实际命令与结果。
- 是否覆盖 auth/API-key/quota/usage/Codex/backend/admin UI/secrets 边界。
- 没跑真实上游、远端部署或 full/strict 时，要写清原因和剩余风险。
