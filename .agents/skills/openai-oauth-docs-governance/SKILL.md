---
name: openai-oauth-docs-governance
description: 项目文档治理（openai-oauth-api-service）。Use when creating, reviewing, renaming, reorganizing, or updating service architecture, operations, deployment, README, AGENTS, or progress docs.
---

# OpenAI OAuth Docs Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

Use this skill to keep `openai-oauth-api-service` docs concise, source-grounded, and operationally safe. This is local project documentation governance; use `openai-docs` separately for official OpenAI API/Product documentation questions.

## OpenAI OAuth 文档质量门禁 Docs Quality Gate

文档治理不能只追求写得多或排版整齐。要保护当前真源、降低心智负担、避免文档漂移，并控制文档体系复杂度。

### 结构质量检查 Structure Quality Checks

- 边界清晰、合理严谨：说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 语义清晰：标题、术语、表格、图、链接和锚点必须指向明确真源；读者能知道它是什么、适用哪里、下一步去哪。
- 职业任务文案：面向业务、运营、管理或客户阅读的文档，用读者岗位语言说明任务、影响和下一步；工程术语只在开发、诊断、部署、接口或规则文档中使用，并补业务释义。
- 模块化：按读者任务和真源边界组织文档；能在一页讲清就不拆，长文才用索引、摘要表和专题页。
- 高内聚：同一口径、命令、状态、链接和跳转锚点尽量收口到一个真源或索引，不在多处复制近似说明。
- 低耦合：文档引用代码、脚本、页面和正式真源位置，不复写易漂移的实现细节；过程记录不覆盖正式文档。
- 单一职责：README 管导航，专题文档管业务/操作，progress/changelog 管过程；图表和表格只在降低理解成本时使用。

- 先确认代码、migration、测试、README、正式 docs 和 AGENTS 的优先级，不让过程记录覆盖当前真源。
- 结论、适用范围、主路径、验收方式和风险边界前置；表格、Mermaid、链接和摘要只在减少查找成本时使用。
- 行为、入口、配置、测试或部署口径变化时，同步相关索引、README 和 progress；只改措辞时不机械扩大同步面。
- 不为普通说明引入重模板、重复负面清单或并行 metadata；能由现有脚本、索引或文档承接的规则，不再造一套真源。

## Workflow

1. Snapshot scope and classify the task.
   - Run `git -C /Users/simon/projects/openai-oauth-api-service status --short` before editing and protect unrelated dirty files.
   - Classify the task as docs-only, docs-adjacent, or behavior-changing.
   - If runtime, schema, API, auth, key lifecycle, upstream strategy, deployment, migration, or frontend regression behavior changes are required, stop treating it as docs-only and follow the relevant project workflow too.

2. Read the project truth chain.
   - Always read `AGENTS.md` for repository rules. Treat it as protected project-level governance.
   - Read `README.md` and `docs/README.md` before changing docs structure, reader paths, or current-state claims.
   - For architecture, OAuth/API key, usage, upstream, logging, data retention, and admin-console behavior, read `docs/architecture.md`.
   - For local run, configuration, operations, deploy defaults, and low-spec boundaries, read `docs/operations.md`, `server/README.md`, and `server/deploy/README.md`.
   - For frontend/admin page wording or testing commands, read `web/README.md`.
   - Treat `legacy-python-mvp/` as historical reference only, not current implementation truth.
   - Treat `progress.md` and `docs/archive/**` as process/history evidence, not current formal truth.

3. Protect governance and secrets boundaries.
   - Ordinary docs cleanup should read `AGENTS.md`, not edit it.
   - Edit `AGENTS.md` only when the user explicitly asks to change long-term rules, prohibited actions, required workflows, or repository-wide policy.
   - Keep secrets guidance, logging rules, deployment build boundaries, and default admin-password policy aligned with `AGENTS.md`; do not dilute them in ordinary docs.
   - Never add real tokens, JWT secrets, database passwords, OAuth secrets, Codex login paths with credentials, or production private `.env` values to docs.
   - In the final response, explicitly say whether `AGENTS.md` was read only or changed.

4. Maintain source-of-truth boundaries.
   - Architecture truth: `docs/architecture.md`.
   - Operations and deployment truth: `docs/operations.md`, `server/deploy/README.md`, and `server/deploy/compose/prod/*` when relevant.
   - Frontend/admin truth: `web/README.md`, route/page code, and `web/scripts/styleL1.mjs`.
   - Backend/API truth: `server/README.md`, `server/docs/*`, service/biz/data code, Ent schema, migrations, and tests.
   - Usage diagnostics truth: backend recorded data such as `gateway_usage_logs`; docs must not claim request bodies, prompts, or outputs are stored by default.
   - Public/OpenAI official API behavior should be checked with `openai-docs` when current external docs matter; do not invent official claims in this project doc skill.

5. Design docs for readers.
   - Put current conclusion, scope, main path, commands, and risk boundary before history or detailed evidence.
   - Give readers a path near the top: local development, admin operation, deployment, debugging, or contribution.
   - Choose the expression shape by the information type, not by decoration:
     - Use tables for short comparable facts, architecture/operations status, endpoint or route comparisons, environment variables, key/usage/upstream behavior matrices, command catalogs, acceptance criteria, risk registers, and docs classification.
     - Use numbered lists for local run steps, deployment/runbooks, troubleshooting paths, migration sequences, and verification steps.
     - Use code blocks for commands, env/config snippets, API examples, SQL, and minimal reproducible snippets.
     - Use short paragraphs under clear headings for rules, rationale, boundaries, and caveats.
     - Use nearby links and section anchors when readers need to jump from a summary to an exact architecture section, operation runbook, deploy command, admin behavior, usage diagnostic, acceptance section, or risk boundary.
     - Use Mermaid or simple diagrams only when a visual structure makes request flow, OAuth redirect, admin/key lifecycle, usage logging, upstream fallback, deployment, source-of-truth chains, or decision trees easier to understand than prose.
   - Make commands copyable and context-specific: include `cd /Users/simon/projects/openai-oauth-api-service/...` when useful and name the expected success signal.
   - For non-trivial diagrams, add a short lead-in or follow-up sentence, keep diagrams compact, and use stable human-readable labels.
   - Do not stack tables, diagrams, and links for visual polish alone. Each structure should answer a reader question or reduce lookup cost.
   - Do not add Markdown frontmatter or metadata by default. First identify a real consumer such as a docs viewer, generator, search index, or build script.
   - Do not force plush's Chinese-filename or docs-inventory rules onto this project. This repo currently uses stable English doc filenames and `docs/README.md` as the docs index.

6. Keep docs synchronized with behavior.
   - If admin page behavior, route names, key management, usage fields, upstream strategy, model limits, public balance, OAuth callback, deploy steps, migration commands, or quality commands change, check related README/docs in the same round.
   - If deployment docs change, preserve the low-spec boundary: build locally/CI, upload/load remotely, run migration/smoke remotely, and do not build on the low-spec server.
   - If usage/admin visibility changes, check both compact dashboard wording and detailed `/admin-usage` behavior where relevant.
   - If any file in the repo changed, update `progress.md` according to `AGENTS.md`; do not overwrite unrelated existing `progress.md` content.
   - If only global skill files changed outside the repo, do not update project `progress.md`.

7. Validate with targeted scans.
   - Run `git -C /Users/simon/projects/openai-oauth-api-service diff --check` for repo changes.
   - Use targeted `rg` for old paths, route names, environment variables, stale headings, stale anchors, old deployment claims, and changed terminology.
   - For Mermaid changes, scan fenced blocks and surrounding references for syntax shape and label consistency.
   - For docs surfaced through frontend or scripts, run the relevant frontend/backend checks named by `README.md` or `web/README.md`.
   - For docs-only changes, do not run migrations or unrelated heavy runtime tests unless the touched docs/scripts require them.

## Deliverable Standard

When answering after using this skill, report:

- Verdict if the user asked whether a docs direction is reasonable.
- Whether `AGENTS.md` was read only or changed, and why.
- What docs were created, renamed, deleted, simplified, split, re-linked, or intentionally left untouched.
- What diagrams or metadata/frontmatter were added, changed, or intentionally skipped.
- Whether `README.md`, `docs/README.md`, `web/README.md`, `server/README.md`, deploy docs, anchors, references, and `progress.md` needed updates.
- Which scans or validation commands passed.
- What remains intentionally out of scope, especially runtime behavior, schema, auth/key semantics, upstream failover behavior, deployment execution, secrets, legacy Python MVP rewriting, archive rewriting, and broad directory reorganization.
