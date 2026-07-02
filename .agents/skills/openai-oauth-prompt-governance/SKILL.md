---
name: openai-oauth-prompt-governance
description: 项目提示词治理（openai-oauth-api-service）。Use when Codex writes, refines, evaluates, or converts an OpenAI OAuth API service request into an executable prompt for implementation, review, docs governance, admin page design, tests, deployment, handoff, side chat, main chat, or commit/push work; when a complete copyable final prompt, prompt length control, Codex input limit, engineering quality gate, maintainability, extensibility, simplicity, complexity budget, or prompt boundary conditions are needed; when prompts need auth/API-key/quota/usage logging/Codex backend/secrets/proxy/upstream/deploy boundaries; or when the user wants positive "要做什么" wording instead of broad "不要" lists.
---

# OpenAI OAuth Prompt Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

Use this skill to draft prompts for openai-oauth-api-service work. The prompt should identify the target layer, security-sensitive boundaries, validation, and closeout. Prefer positive instructions; reserve "do not" for secrets, auth, live upstream, deployment, and destructive Git risks.

## Prompt Principle

Write prompts around "要做什么":

- 要修正或评估哪个 auth/API key/quota/usage/admin/deploy behavior。
- 要先读 README、AGENTS、server/web/scripts/deploy docs 中哪些真源。
- 要允许改哪些 server/web/deploy/docs paths。
- 要覆盖哪些测试形态和运行环境。
- 要在最终回复说明验证命令、未覆盖真实上游或远端部署的原因。

Use "不要 / 禁止" only for expensive mistakes:

- 不泄漏 API key、tokens、session、prompt、customer data 或 secrets。
- 不把 live OpenAI/Codex upstream 调用当 deterministic unit test。
- 不把项目说成单纯 gateway；它还包含 OAuth、API key、token、usage 和管理端。
- 不在低配服务器构建；远端只加载制品、migration、启动、health/ready 和 smoke。
- 不跳过 auth、quota、usage logging、error classification、admin UI 或 deploy preflight 的边界说明。
- 不改 unrelated dirty worktree，不 reset/stash/force push。

## Complete Prompt Output

当任务是“写 / 改 / 转换提示词”时，必须输出一份完整可复制的 `最终提示词`，用 fenced Markdown 包起来；不要只给原则、片段或检查清单。

如果用户只是问“是否合理 / 为什么 / 怎么处理”，先短答，不强制展开成完整提示词。

长度治理：

- 最终提示词必须能放进目标 Codex / ChatGPT 输入窗口。目标限制未知时，默认压缩历史，保留真源、当前状态、决策、阻塞和验收。
- 如果仍可能超限，输出 `主提示词` + `补充上下文`，不要给一个无法粘贴执行的超长版本。
- 不凭空声称精确 token 余量；需要时只说明压缩和拆分策略。

完整 openai-oauth 提示词通常应包含：相关 `$openai-oauth-*` skills、目标、先读真源、允许修改、本轮不做、验收、progress.md 要求、真实上游 / secrets / deploy 边界和收口要求。微型提示词可省略明显无关段落。

## Engineering Quality Gate

Structure constraints to include when relevant:

- 边界清晰、合理严谨：说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 语义清晰：提示词要定义关键名词、目标输出、范围、非目标、验收和后果，避免泛称驱动无约束实现。
- 职业任务文案：当提示词要求生成页面、帮助、错误提示或业务文档时，必须先定义目标读者/岗位，并要求输出使用业务语言，不把开发术语写进非开发者会看到的文案。
- 模块化：提示词要要求按真实职责拆分，不做无意义拆文件，也不把多个阶段塞进一次大改。
- 高内聚：同一规则、字段真源、权限判断、错误处理或文档口径要收口到同一 usecase/helper/config/test source。
- 低耦合：要求页面、usecase、repo、schema、配置、部署和测试的依赖方向清楚，不跨层偷做逻辑。
- 单一职责：要求输出说明新增抽象、fallback、API/schema/config 的理由、收益、验证方式和退出边界。

openai-oauth 提示词必须把安全、可诊断和复杂度控制作为交付条件。不能为了“请求能通”而绕过 auth、quota、usage、error 或 deploy 边界：

- 优先复用现有 auth、API key、quota、usage logging、admin UI、proxy/upstream、deploy 和 health/ready 结构。
- 新增抽象、配置、fallback、upstream 策略、缓存、migration、admin 页面或部署步骤前，必须说明为什么现有能力不能承接，以及对安全、可观测性和运维的影响。
- 不把真实 upstream 抖动、密钥问题或 quota 边界用宽松 fallback 掩盖；需要 fallback 时写清触发条件、退出条件和日志证据。
- 如果任务同时牵涉后端逻辑、管理端、真实上游、部署和文档，先拆成可验证切片。
- 收口必须说明复用点、复杂度变化、安全 / 隐私边界、可观测性证据、未验证真实上游或远端环境的范围和剩余风险。

## Standard OpenAI OAuth Prompt

```markdown
$openai-oauth-prompt-governance
$relevant-openai-oauth-skill

目标：
请完成 <one concrete service/admin/deploy outcome>.

先读：
- /Users/simon/projects/openai-oauth-api-service/README.md
- /Users/simon/projects/openai-oauth-api-service/AGENTS.md
- <server/web/scripts/deploy docs relevant to this task>

允许修改：
- <exact paths/modules>

本轮不做：
- <only high-risk non-goals: secrets, live upstream, migration, deploy, low-spec build, etc.>

工程质量：
- 优先复用 openai-oauth 现有 auth、API key、quota、usage logging、admin UI、proxy/upstream、deploy 和 health/ready 结构。
- 新增抽象、配置、fallback、upstream 策略、缓存、migration 或部署步骤前，先说明复用不足、安全影响和运维影响。
- 收口时说明复杂度控制、复用点、安全 / 隐私边界、可观测性证据、未验证项和剩余风险。

验收：
- 先按影响面选择测试形态。
- 执行 <Go/web/style:l1/migration/preflight/full/strict as needed>.
- 有正式改动时更新 progress.md。

收口：
- 说明改动文件、验证命令、未覆盖项和剩余风险。
- 如用户要求提交/推送，只提交本轮相关文件，推送前 fetch 并确认不落后远端。
```

## Skill Pairing

| Task | Add these skills |
| --- | --- |
| 文档治理 / docs | `$openai-oauth-docs-governance` |
| 管理端页面 / 信息密度 | `$openai-oauth-page-governance` |
| 代码 review | `$openai-oauth-code-review-governance` |
| 测试选择 / 验证范围 | `$openai-oauth-test-governance` |
| 通用提示词整理 | `$prompt-governance` |
| 发布/部署/版本 | `$openai-oauth-release-governance` |
| 领域边界/实现前评估 | `$openai-oauth-domain-boundary-governance` |
| 运行故障诊断 | `$openai-oauth-runtime-diagnostics` |
| 可观测/错误提示 | `$openai-oauth-observability-error-governance` |
| 安全/隐私/权限 | `$openai-oauth-security-privacy-governance` |

## Prompt Patterns

### Server / Auth / Usage

```markdown
$openai-oauth-prompt-governance
$openai-oauth-test-governance

目标：
请修正 <auth/API key/quota/usage/Codex backend behavior>.

验收：
- 覆盖正常路径、权限失败、额度不足、禁用/过期、上游失败和日志字段。
- 真实上游只作为明确授权的 smoke；单元/集成测试优先用 fake/local path。
工程质量：
- 不用宽松 fallback 掩盖 auth、quota、usage logging 或 upstream 分类错误；需要 fallback 时写清日志和退出条件。
```

When asked to produce a prompt, deliver it as:

````markdown
最终提示词：

```markdown
$openai-oauth-prompt-governance
...
```
````

### Admin UI

```markdown
$openai-oauth-page-governance
$openai-oauth-prompt-governance

目标：
请优化或修复 <admin page>.
要求保留管理端操作效率和指标可读性，验证 web unit/static 和必要 `style:l1` 场景。
```

### Deploy

```markdown
$openai-oauth-prompt-governance

目标：
请准备或执行 <deploy/release task>.
要求先确认目标、制品、migration、health/ready、回滚和磁盘清理边界。
低配服务器不构建，只加载制品、启动、migration、健康检查和必要 smoke。
```

## Common Mistakes

- 只说 "接口有问题"，不说明是 OAuth、API key、quota、usage、Codex backend 还是 admin UI。
- 把密钥、token 或真实请求体粘进提示词。
- 要求 "完整测试" 但不说明是否包含真实上游、admin browser regression、migration 或远端 smoke。
- 把部署、业务修复、UI 重排和 docs 全塞进一个提示词。
- 只讲提示词原则但不给最终可复制版本，或把完整聊天历史塞进一个超长 prompt。
- 只要求“请求可用”，但不要求 auth/quota/usage/error/deploy 边界、复杂度预算和可观测证据。
