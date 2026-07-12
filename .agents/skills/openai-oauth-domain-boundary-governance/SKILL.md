---
name: openai-oauth-domain-boundary-governance
description: 项目服务边界治理（openai-oauth-api-service）。Use when work may change OAuth/auth, gateway/proxy, upstream providers, admin API, usage logging, persistence, or configuration truth.
---

# OpenAI OAuth 业务边界治理 Domain Boundary Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这个 skill 在实现 `openai-oauth-api-service` 功能前收敛 domain ownership、source of truth、API/RBAC、frontend/backend responsibility 和 customer/template-specific boundary。

它是后端/API/auth/usage/upstream 变更的主治理入口。管理端页面治理可以发现 UI 暗示了新能力；一旦涉及 OAuth/API key、quota、usage logging、gateway/proxy、upstream failover、admin API、schema/migration、transaction、error code 或 persisted config，就先回到本 skill。

## OpenAI OAuth 工程质量门禁 Engineering Quality Gate

业务边界治理必须守住最小必要复杂度和单一真源。

### 结构质量检查 Structure Quality Checks

- 边界清晰、合理严谨：说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 语义清晰：字段、状态、权限、错误、生命周期和真源命名必须说明它是什么、不是什么、由谁负责、会触发什么后果。
- 模块化：按真实业务/技术职责拆分；只有能降低理解、测试或变更成本时才拆，不做空壳转发或为拆而拆。
- 高内聚：同一业务规则、字段真源、错误/权限判断、数据转换或状态推进尽量收口到同一 usecase/helper/config/test source。
- 低耦合：页面不偷做后端事实逻辑，usecase 不管展示细节，repo 不承载业务决策；跨层依赖要有清楚方向和合同。
- 单一职责：一个模块不要同时处理展示、权限、数据派生、保存、副作用和兜底；如果必须临时承载，说明边界和退出路径。

- 新增 schema、migration、repo、usecase、API、RBAC 权限、状态、字段或配置前，先证明现有真源不能承接，并说明新增复杂度的收益和退出边界。
- 优先主路径修复，不用页面私有逻辑、脚本补写、兼容 fallback、重复派生字段或宽松校验掩盖后端合同缺口。
- 字段残值/缺值、幂等、事务、权限和客户/模板差异必须可测试、可解释、可回滚；不能只让当前 happy path 通过。
- 若任务跨太多层，先收窄成一个可验证切片；不在一轮里无约束扩张到 schema、RBAC、UI、docs、deploy 全链路。

## 真源链 Truth Chain

- 先读 `AGENTS.md`、`README.md`、`docs/architecture.md`、`docs/operations.md`、server/web/deploy docs 和相关 tests。
- 代码、schema/migrations、tests、formal docs 强于聊天规划或旧 reference notes。

## 项目规则 Project Rules

- 先区分 OAuth/auth、gateway/proxy、upstream provider、admin UI、usage/billing visibility 和 deploy host 边界。
- 后端责任要落到 server/API/auth/usage/upstream 真源，不把 admin 页面状态、缓存 fallback 或旧 Python MVP 口径当成当前实现。
- 不把 host-side failover、mihomo/systemd 逻辑写进业务 Docker image。
- 中途流式失败不能危险重放；要区分 `pre-stream open failure` 和 `mid-stream interruption`。

## 工作流 Workflow

1. 写出 single domain outcome 和 owning layer。
2. 找到 source-of-truth fields、states、identifiers、permissions、derived values。
3. 检查现有 table/usecase/API/helper 是否已经拥有该行为。
4. 覆盖 stale/missing value paths：defaults、edits、source switch/clear、list/detail/print/export/search、historical fallback。
5. UI 不补造 backend facts；客户/模板特例不污染 generic core。
6. 按影响面选择 unit、integration、contract、browser、migration validation。

## 输出 Output

汇报 ownership decisions、source truth、changed layers、intentionally untouched layers、stale/missing paths、validation 和 residual risks。
