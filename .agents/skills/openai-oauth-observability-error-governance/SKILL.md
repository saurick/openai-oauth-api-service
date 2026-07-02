---
name: openai-oauth-observability-error-governance
description: 项目可观测性与错误治理（openai-oauth-api-service）。Use when Codex designs, reviews, or changes openai-oauth-api-service structured logs, request IDs, trace IDs, metrics, audit evidence, error codes, error classification, retries, fallbacks, alerts, dashboards, user-facing error messages, or debugging evidence.
---

# OpenAI OAuth 可观测性与错误治理 Observability Error Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这个 skill 处理 `openai-oauth-api-service` logs、traces、metrics、audit evidence、error codes、fallbacks、dashboards 和 user-facing errors，让问题能被定位、解释和复现。

## 真源链 Truth Chain

- 先读 error/logging helpers、API contracts、frontend error handling、observability docs 和相关 tests。
- 明确 signal 是给 local debugging、production operations、user support、audit 还是 product metrics 使用。

## 项目规则 Project Rules

- `gateway_usage_logs` 是请求诊断主表；新字段要支持 request_id/session_id、客户端 IP、upstream error、latency classification。
- fallback 要诚实标记 `stale=true`、原因和时间，不把失败伪装成实时成功。
- 管理端指标顺序和最近调用字段是 visibility contract，改动后用测试/浏览器回归保护。

## 结构质量门禁 Structure Quality Gate

- 边界清晰、合理严谨：说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 语义清晰：错误码、日志字段、request_id、用户提示和排障证据必须表达明确含义，不用模糊文案掩盖责任边界。
- 模块化：日志、指标、错误码、审计、用户提示和告警各自回答明确问题，不把所有信号堆进一个字段或一个通用错误。
- 高内聚：同一错误分类、request_id、domain id、fallback 状态和用户提示收口到共享 helper/码表/日志结构。
- 低耦合：用户提示不泄漏内部实现，内部日志不依赖页面文案；观测信号不反向改变业务事实。
- 单一职责：fallback、retry、cache、stale/degraded 只表达真实状态，不能同时承担成功路径、错误吞噬和指标美化。

## 工作流 Workflow

1. 定义 signal 要回答哪个 operator/user question。
2. 包含稳定 identifiers：request/job/session/domain ids、status、latency、dependency、sanitized classification。
3. 区分 technical logs 和 user-facing messages。
4. fallback/degraded/stale 行为要明确标记原因、时间和证据来源。
5. 用测试、日志样本、浏览器/API evidence 证明 signal 可用。

## 输出 Output

汇报 changed signals、fields、error classifications、user messages、redaction choices、validation 和 remaining observability gaps。
