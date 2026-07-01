---
name: openai-oauth-runtime-diagnostics
description: 项目运行时故障诊断（openai-oauth-api-service）。Use when Codex diagnoses openai-oauth-api-service page errors, API/RPC failures, backend read/write failures, migration drift, database mismatch, deployment mismatch, browser/runtime issues, logs, request IDs, configuration drift, environment confusion, or production/test/local differences before changing code.
---

# OpenAI OAuth 运行时诊断 Runtime Diagnostics

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这个 skill 在修改 `openai-oauth-api-service` 代码前先用 runtime evidence 分层定位故障，避免把环境、数据、migration、部署或浏览器问题误修成代码补丁。

## 真源链 Truth Chain

- 核对 actual environment、branch/commit/image、config/env、DB/migration state、logs、request IDs、browser network/console、recent deploys。
- live behavior 可取得时，不只靠 static code 推断 runtime truth。

## 项目规则 Project Rules

- 生产 502 / balance / usage 问题先查 `gateway_usage_logs`、request_id/session_id、container logs、upstream response 和 host network。
- 管理端页面问题要同时核对 API payload、browser render、cache/stale fallback 和部署版本。
- 先确认故障服务确实属于当前 repo，再改代码。

## 结构质量门禁 Structure Quality Gate

- 边界清晰、合理严谨：说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 模块化：诊断先按 browser、API、service、DB/migration、auth/RBAC、config/deploy、external dependency 分层，不把所有问题混成代码补丁。
- 高内聚：证据、复现步骤、日志/request_id、环境差异和 root cause 归到同一故障链路里，避免散落成不可复查的截图或口头结论。
- 低耦合：先证明失败层，再决定改代码、数据、配置、migration 或部署；不要用 UI fallback 掩盖后端/环境真因。
- 单一职责：临时探针、脚本或日志只服务本次定位；若要长期保留，必须说明归属、触发条件和验证价值。

## 工作流 Workflow

1. 捕获 symptom：route/API、user/role、timestamp、environment、last known good version。
2. 分层：browser/UI、route/menu、API/RPC、service/usecase、DB/migration、auth/RBAC、config/env、deploy/container、network/upstream。
3. 用一个最小 command/request/browser action 复现。
4. 对比 runtime evidence 与 code/docs，区分 local/test/prod 和 mock/real path。
5. 先给出 root cause 或 narrowed suspects，再决定改代码、改数据、改配置、补 migration 或回滚部署。

## 输出 Output

汇报 symptom、evidence、failing layer、reproduction、root cause/suspects、fix path、validation 和 remaining blind spots。
