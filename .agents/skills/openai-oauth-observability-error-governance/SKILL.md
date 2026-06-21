---
name: openai-oauth-observability-error-governance
description: Project-specific observability and error-governance workflow for openai-oauth-api-service. Use when Codex designs, reviews, or changes openai-oauth-api-service structured logs, request IDs, trace IDs, metrics, audit evidence, error codes, error classification, retries, fallbacks, alerts, dashboards, user-facing error messages, or debugging evidence.
---

# OpenAI OAuth Observability Error Governance

Use this skill when openai-oauth-api-service logs, traces, metrics, audit evidence, error codes, fallbacks, dashboards, or user-facing errors change.

## Truth Chain

- Read project error/logging helpers, API contracts, frontend error handling, observability docs, and tests for touched paths.
- Check whether the signal must support local debugging, production operations, user support, audit, or product metrics.

## Project Rules

- `gateway_usage_logs` 是请求诊断主表；新字段要能支持 request_id/session_id、客户端 IP、上游错误和耗时分类。
- fallback 要诚实标记 `stale=true`、原因和时间，不把失败伪装成实时成功。
- 管理端指标顺序和最近调用字段是可见性合同，改动后用测试/浏览器回归保护。

## Workflow

1. Define which operator/user question the signal answers.
2. Include stable request/job/session/domain identifiers and sanitized classifications.
3. Separate technical logs from user-facing messages.
4. Mark degraded/stale/fallback behavior honestly.
5. Redact secrets and sensitive customer/user data.
6. Validate at least one success and relevant failure path when feasible.

## Output

Report changed signals, identifiers, redaction, user-facing messages, failure paths checked, and remaining diagnostic gaps.
