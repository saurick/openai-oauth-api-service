---
name: openai-oauth-security-privacy-governance
description: Project-specific security and privacy governance for openai-oauth-api-service. Use when Codex works on openai-oauth-api-service authentication, authorization, RBAC, permissions, secrets, credentials, API keys, tokens, production access, customer data, PII, data export, logs containing sensitive data, privacy boundaries, or security-sensitive deployment/configuration changes.
---

# OpenAI OAuth Security Privacy Governance

Use this skill when openai-oauth-api-service changes touch authentication, authorization, secrets, credentials, production access, customer/user data, sensitive logs, exports, or privacy boundaries.

## Truth Chain

- Read project `AGENTS.md`, auth/RBAC docs, deploy/config docs, secret/preflight scripts, and touched code/tests.
- Treat production/test envs, tokens, credentials, customer data, logs, screenshots, and exports as sensitive by default.

## Project Rules

- API keys、OAuth tokens、upstream credentials、admin access、balance payload 和 request logs 默认敏感。
- 日志和 UI 不暴露完整密钥、Authorization header、用户隐私或可复用 token。
- 生产 env/compose/secret 改动必须有 preflight、最小权限、回滚和脱敏证据。

## Workflow

1. Identify assets, actors, permissions, secrets, and sensitive data involved.
2. Confirm backend/API authorization; UI hiding is not a security boundary.
3. Avoid logging/committing/exposing real secrets, tokens, PII, customer files, or reusable credentials.
4. Use least privilege and explicit target environment for risky operations.
5. Validate unauthorized/disabled/no-permission/wrong-role/secret-placeholder/data-leak paths as relevant.
6. Update docs/progress when security, privacy, deploy, or permission behavior changes.

## Output

Report assets touched, permission model, secret/privacy handling, checks run, residual risk, and any rotation or follow-up needed.
