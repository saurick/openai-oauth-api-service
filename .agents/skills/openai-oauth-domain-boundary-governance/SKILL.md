---
name: openai-oauth-domain-boundary-governance
description: Project-specific domain-boundary implementation governance for openai-oauth-api-service. Use when Codex implements or reviews openai-oauth-api-service feature work that may affect data ownership, domain models, workflows, facts, schemas, APIs, permissions, frontend/backend responsibility, customer-specific behavior, source-of-truth fields, stale/missing field values, or cross-module boundaries.
---

# OpenAI OAuth Domain Boundary Governance

Use this skill before implementing openai-oauth-api-service feature work that may change domain ownership, data truth, APIs, permissions, frontend/backend responsibility, or customer/template-specific behavior.

## Truth Chain

- Read project `AGENTS.md`, `README.md`, current-source docs, and nearest module docs/code/tests for the touched area.
- Treat existing code, schema/migrations, tests, and formal docs as stronger truth than chat plans or old reference notes.

## Project Rules

- 先区分 OAuth/auth、gateway/proxy、upstream provider、admin UI、usage/billing visibility 和 deploy host 边界。
- 不把 host-side failover、mihomo/systemd 逻辑写进业务 Docker image。
- 中途流式失败不能危险重放；要分类 pre-stream open failure 和 mid-stream interruption。

## Workflow

1. State the single domain outcome and the owning layer.
2. Identify source-of-truth fields, states, identifiers, permissions, and derived values.
3. Check whether an existing table/usecase/API/helper already owns the behavior.
4. Cover stale/missing value paths: defaults, edits, source switch/clear, list/detail/print/export/search, and historical fallback when relevant.
5. Keep UI from inventing backend facts and keep customer/template specifics out of generic core.
6. Choose tests by impact: unit/integration/contract/browser/migration as applicable.

## Output

Report ownership decisions, source truth, changed layers, intentionally untouched layers, stale/missing paths, validation, and residual risks.
