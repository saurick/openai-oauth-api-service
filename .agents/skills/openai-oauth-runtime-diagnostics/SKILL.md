---
name: openai-oauth-runtime-diagnostics
description: Project-specific runtime diagnostics workflow for openai-oauth-api-service. Use when Codex diagnoses openai-oauth-api-service page errors, API/RPC failures, backend read/write failures, migration drift, database mismatch, deployment mismatch, browser/runtime issues, logs, request IDs, configuration drift, environment confusion, or production/test/local differences before changing code.
---

# OpenAI OAuth Runtime Diagnostics

Use this skill to diagnose openai-oauth-api-service runtime failures from evidence before editing code.

## Truth Chain

- Check actual environment, branch/commit/image, config/env, DB/migration state, logs, request IDs, browser network/console, and recent deploys.
- Do not infer runtime truth from static code alone when live behavior is available.

## Project Rules

- 生产 502 / balance / usage 问题先查 `gateway_usage_logs`、request_id/session_id、容器日志、上游响应和 host 网络。
- 管理端页面问题要同时核对 API payload、浏览器渲染、缓存/stale fallback 和部署版本。
- 先定位服务是否真的属于当前 repo，再改代码。

## Workflow

1. Capture exact symptom, route/API, user/role, timestamp, environment, and last known good version.
2. Classify the failing layer: browser/UI, route/menu, API/RPC, service/usecase, DB/migration, auth/RBAC, config/env, deploy/container, network/upstream.
3. Reproduce narrowly with one command/request/browser action.
4. Compare runtime evidence against code/docs; distinguish local/test/prod and mock/real paths.
5. Fix the owning layer, avoiding page-local or fallback patches unless they are documented and bounded.
6. Rerun the failing path and adjacent regression checks.

## Output

Report root cause, evidence, environment, commands/requests, fix scope, validation, and unverified paths.
