---
name: openai-oauth-prompt-governance
description: Project-specific prompt governance for /Users/simon/projects/openai-oauth-api-service. Use when Codex writes, refines, evaluates, or converts an OpenAI OAuth API service request into an executable prompt for implementation, review, docs governance, admin page design, tests, deployment, handoff, side chat, main chat, or commit/push work; when prompts need auth/API-key/quota/usage logging/Codex backend/secrets/proxy/upstream/deploy boundaries; or when the user wants positive "要做什么" wording instead of broad "不要" lists.
---

# OpenAI OAuth Prompt Governance

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
```

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
