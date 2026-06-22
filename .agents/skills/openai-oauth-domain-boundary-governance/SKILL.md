---
name: openai-oauth-domain-boundary-governance
description: openai-oauth-api-service 项目业务边界与数据真源治理。Use when Codex implements or reviews openai-oauth-api-service feature work that may affect data ownership, domain models, workflows, facts, schemas, APIs, permissions, frontend/backend responsibility, customer-specific behavior, source-of-truth fields, stale/missing field values, or cross-module boundaries.
---

# OpenAI OAuth 业务边界治理 Domain Boundary Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这个 skill 在实现 `openai-oauth-api-service` 功能前收敛 domain ownership、source of truth、API/RBAC、frontend/backend responsibility 和 customer/template-specific boundary。

它是后端/API/auth/usage/upstream 变更的主治理入口。管理端页面治理可以发现 UI 暗示了新能力；一旦涉及 OAuth/API key、quota、usage logging、gateway/proxy、upstream failover、admin API、schema/migration、transaction、error code 或 persisted config，就先回到本 skill。

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
