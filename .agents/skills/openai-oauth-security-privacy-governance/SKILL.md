---
name: openai-oauth-security-privacy-governance
description: openai-oauth-api-service 项目安全与隐私治理。Use when Codex works on openai-oauth-api-service authentication, authorization, RBAC, permissions, secrets, credentials, API keys, tokens, production access, customer data, PII, data export, logs containing sensitive data, privacy boundaries, or security-sensitive deployment/configuration changes.
---

# OpenAI OAuth 安全与隐私治理 Security Privacy Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这个 skill 处理 `openai-oauth-api-service` authentication、authorization、RBAC、secrets、production access、customer/user data、sensitive logs、exports 和 privacy boundaries。

## 真源链 Truth Chain

- 先读 `AGENTS.md`、`README.md`、`docs/architecture.md`、`docs/operations.md`、server/web/deploy docs 和相关 tests。
- auth/RBAC docs、deploy/config docs、secret/preflight scripts、touched code/tests 是当前判断依据。

## 项目规则 Project Rules

- API keys、OAuth tokens、upstream credentials、admin access、balance payload 和 request logs 默认敏感。
- 日志和 UI 不暴露完整密钥、Authorization header、用户隐私或可复用 token。
- 生产 env/compose/secret 改动必须有 preflight、least privilege、rollback 和脱敏证据。

## 工作流 Workflow

1. 识别 assets、actors、permissions、secrets、sensitive data。
2. 确认 backend/API authorization；UI hiding 不是 security boundary。
3. 不记录、不提交、不展示真实 secrets、tokens、PII、customer files、reusable credentials。
4. 高风险操作使用 least privilege、explicit target environment、backup/rollback。
5. 用 tests、secret scan、log redaction check、preflight 或 deployment evidence 验证。

## 输出 Output

汇报 assets、permission boundary、secret/privacy handling、logs/export choices、validation commands 和 residual risks。
