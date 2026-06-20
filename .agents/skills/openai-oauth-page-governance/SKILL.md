---
name: openai-oauth-page-governance
description: Project-specific page design governance for /Users/simon/projects/openai-oauth-api-service. Use when Codex designs, reviews, simplifies, or implements this project's admin console pages, API operations dashboard, usage logs, upstream strategy page, API key tables and dialogs, OAuth/admin login, client-config generator, model/limit settings, public balance page, buttons, filters, tables, charts, empty/error states, light/dark theme, responsive layout, credential/key reset actions, accessibility, keyboard/focus behavior, or when the user mentions openai-oauth-api-service with 页面治理, 简洁易用, 心智负担, 信息密度, admin 页面, usage 可见性, key 管理, 上游策略, style:l1, 暗色模式, 表格, 弹窗, or asks whether admin-console guidance should become reusable.
---

# OpenAI OAuth Page Governance

Use this skill to keep `openai-oauth-api-service` admin pages useful, safe, and verifiable. This is project-specific admin-console guidance, not OpenAI official product documentation and not ERP page governance.

## Workflow

1. Establish scope and current truth.
   - Run `git -C /Users/simon/projects/openai-oauth-api-service status --short` before editing and protect unrelated dirty files.
   - Classify the task as page-only, page-adjacent, or behavior-changing. If schema, API, auth, key semantics, usage aggregation, upstream failover, deployment, or server behavior changes are needed, follow the corresponding project workflow too.
   - Read the relevant truth chain before making current-state claims: `AGENTS.md`, `README.md`, `docs/README.md`, `docs/architecture.md`, and `web/README.md`.
   - For server/API/data-backed page work, also read `server/README.md` and the relevant `server/docs/*` or implementation files.
   - For deployment-visible behavior, read `docs/operations.md` and `server/deploy/README.md`.
   - Inspect the real runtime page and existing components when changing layout, density, spacing, styles, interactions, responsive behavior, or visible page structure.

2. Define the page's primary job.
   - State who uses the page: usually a project administrator operating downstream API keys, usage, upstream mode, OAuth/login, or client configuration.
   - Keep the admin dashboard as an operational overview. Move deep diagnostics, wide tables, and request-level drilldown to the dedicated pages such as `/admin-usage` or `/admin-upstream` when that matches existing structure.
   - Classify visible elements as decision information, action entry, operational feedback/status, navigation/context, or auxiliary explanation.
   - Keep duplicate shortcuts only when they reduce operator risk or speed a frequent workflow. Otherwise merge, rename, or downgrade.

3. Protect credential, usage, and upstream semantics.
   - Treat API key creation, reset, copy, disable/enable all, delete, quota limits, model settings, and upstream strategy changes as high-risk actions.
   - Confirm what changes after the user acts: data mutation, key invalidation, runtime setting change, copied config, navigation, validation feedback, or a clear next step.
   - Do not imply that a setting is persisted, global, key-specific, backend-only, CLI fallback, or public unless the backend actually supports that scope.
   - Do not display or log secrets beyond the current project rule. Full generated downstream keys may be shown where the project explicitly supports copy/rotation; upstream tokens, JWT secrets, Codex login state, prompt bodies, and model outputs must not be surfaced.
   - Usage visibility should reflect recorded backend data such as `gateway_usage_logs`; do not fabricate missing request/session/client IP data in the frontend.
   - Distinguish pre-stream backend open failures from mid-stream interruptions when showing diagnostics; they have different retry and user expectation boundaries.

4. Reduce density by operational meaning.
   - Dashboard cards should answer "is the service healthy, used, expensive, failing, or rate-limited?" without becoming a second usage-log page.
   - Tables should support scanning: stable columns, clear status labels, readable timestamps, obvious filters, page-size control, and detail drawers/modals only when they add diagnostic value.
   - Empty, loading, failed, unauthorized, disabled-admin, no-data, no-permission, long key remark, long session id, wide IP, large token count, and high-error states must remain readable.
   - Avoid explanatory text that restates labels. Use microcopy for destructive actions, irreversible key reset, public exposure, fallback limitations, or configuration export boundaries.

5. Preserve project boundaries.
   - Do not change schema, migration, auth semantics, route truth, upstream mode behavior, key lifecycle, deployment defaults, or logging policy as a side effect of visual cleanup.
   - Do not restore old portal/user-account flows or Python MVP behavior unless the task explicitly asks for that product review.
   - Do not change the default personal-deploy admin password policy or production deploy path from page work.
   - If a page simplification requires hiding, renaming, or combining official admin routes, stop and treat it as a product/navigation review.

6. Implement with existing admin patterns.
   - Reuse current `web/src` admin components, auth/request helpers, table helpers, theme CSS variables, and error-message helpers.
   - Keep light, dark, and "follow system" themes readable. New backgrounds, borders, status blocks, inputs, tables, and dialogs need dark coverage.
   - Prefer in-app confirmation dialogs for destructive/admin actions instead of browser-native prompts when existing pages already follow that pattern.
   - Keep public `/client-config` separate from authenticated admin navigation. It should not save API keys or call backend admin APIs.
   - Preserve focus, Tab order, Escape/close behavior, disabled/loading states, accessible names, copy-button feedback, and focus return after dialog close.
   - Do not show raw `err.message`, English transport errors, or backend stack-like text to users; use the project's user-facing error helpers.
   - Prefer scoped styles and existing tokens. Do not add `!important` unless the source cannot be controlled; document the reason in the final response.

7. Validate as regression.
   - For admin page/style work, default to:
     ```bash
     cd /Users/simon/projects/openai-oauth-api-service/web && pnpm lint && pnpm css && pnpm test
     cd /Users/simon/projects/openai-oauth-api-service/web && pnpm style:l1
     ```
   - Use `STYLE_L1_SCENARIOS=... pnpm style:l1` only for narrow checks and name the covered scenarios and blind spots.
   - For layout-sensitive work, inspect DOM/box metrics: bounding boxes, overflow, scrollWidth/clientWidth, offsetHeight/clientHeight/scrollHeight, wrapping, and neighboring overlap.
   - For server/API-backed behavior, also run relevant backend tests such as `cd /Users/simon/projects/openai-oauth-api-service/server && go test ./...` or a narrower package test when justified.
   - If files in the repo changed, update `progress.md` according to `AGENTS.md`; do not overwrite unrelated existing progress or failover-script changes.
   - If page behavior or admin wording changed, check whether `README.md`, `web/README.md`, `docs/architecture.md`, `docs/operations.md`, or deploy docs need matching updates.

## Deliverable Standard

When answering after using this skill, report:

- What page meaning, feature semantics, key/usage/upstream behavior, or density changed.
- Which project truth docs and runtime surfaces were checked.
- Which credential, usage, session, client IP, retry/failover, accessibility, keyboard, light/dark, mobile, and adjacent-area states were verified or intentionally left out.
- Which files changed, including whether `progress.md` needed an update.
- Which automated/browser checks passed.
- What stayed intentionally out of scope, especially schema, migration, auth semantics, key lifecycle, upstream behavior, deployment, secrets, Python MVP compatibility, and broad docs reorganization.
