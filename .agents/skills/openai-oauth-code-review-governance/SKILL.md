---
name: openai-oauth-code-review-governance
description: 项目代码审查治理（openai-oauth-api-service）。Use when Codex reviews openai-oauth-api-service code changes in any conversation, including side chats, new main chats, post-implementation reviews, pre-commit reviews, PR-style reviews, current worktree review, staged/unstaged diff review, commit review, or when the user mentions OAuth API service with code review, 审查代码, 查 bug, 独立审查, API key, OAuth, usage, gateway_usage_logs, upstream, Codex backend, CLI fallback, secrets, deployment, admin console, or 不要改只看.
---

# OpenAI OAuth 代码审查治理 Code Review Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这个 skill 审查 `/Users/simon/projects/openai-oauth-api-service` 的代码和正式文档改动。默认只审查，不改代码。

## OpenAI OAuth 工程质量门禁 Engineering Quality Gate

review 不能只找会不会报错。要把可维护性、可扩展性、复杂度预算和长期真源稳定性当成一等审查目标。

### 结构质量检查 Structure Quality Checks

- 边界清晰、合理严谨：说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 模块化：按真实业务/技术职责拆分；只有能降低理解、测试或变更成本时才拆，不做空壳转发或为拆而拆。
- 高内聚：同一业务规则、字段真源、错误/权限判断、数据转换或状态推进尽量收口到同一 usecase/helper/config/test source。
- 低耦合：页面不偷做后端事实逻辑，usecase 不管展示细节，repo 不承载业务决策；跨层依赖要有清楚方向和合同。
- 单一职责：一个模块不要同时处理展示、权限、数据派生、保存、副作用和兜底；如果必须临时承载，说明边界和退出路径。

- 新增 helper、组件、schema、migration、API、RBAC 权限、Workflow/业务规则、配置、QA 脚本或部署步骤时，检查现有能力是否可以承接。
- 警惕为通过当前页面或当前测试而加入局部 fallback、重复派生、页面私有真源、宽松校验、隐藏兼容分支或后处理补丁。
- 实现如果把一个可验证切片扩张成 schema、RBAC、runtime、docs、deployment 多层大改，要检查是否越界、是否能拆小、是否缺少中间验收。
- 质量问题不等于一律阻断；但必须明确当前复杂度和验证范围为什么恰当。

## 范围解析 Scope

1. 用户指定 commit、branch、文件、目录或 PR 时，只审指定范围。
2. side chat 或新会话未指定范围时，审当前仓库 `git status`、staged diff、unstaged diff 和最近相关提交。
3. 当前主会话里“实现后 review”时，审本轮相关改动；若工作区有多组无关改动，先按最近用户请求收窄。
4. 不依赖聊天记忆或实现者解释；以代码、测试、正式文档和当前 diff 为准。

## 必读真源 Truth Chain

先运行：

```bash
git -C /Users/simon/projects/openai-oauth-api-service status --short
git -C /Users/simon/projects/openai-oauth-api-service diff --stat
```

再按触达范围读：

- `AGENTS.md`
- `README.md`
- `docs/architecture.md`
- `docs/operations.md`
- 服务端任务读 `server/README.md`、`server/docs/README.md`、相关 service/biz/data/schema/migration 和测试。
- 前端/admin 任务读 `web/README.md`、相关页面、样式和测试。
- 部署任务读 `server/deploy/README.md`、`server/deploy/compose/prod/*`。
- 涉及 OpenAI 官方 API 行为时，必须另用官方文档能力核对，不靠旧记忆推断。

## 高风险检查 Risk Checklist

重点审这些问题：

- 项目边界：这是长期维护的 OAuth 登录、下游 API key、usage 统计和 OpenAI-compatible API 服务，不是简单转发脚本或一次性网关。
- Secrets：不能提交真实 token、JWT secret、数据库密码、OAuth secret、Codex 登录态路径中的敏感信息或生产 `.env`。
- 日志与存储：请求体、用户 prompt、模型输出和完整 token 默认不落库；usage 优先记录 key、模型、状态码、延迟、字节数、token 用量和错误类型。
- OAuth / admin：管理员登录、JWT、OAuth callback、允许前端 origin、默认 `admin/adminadmin` 口径和生产改密边界不能漂移。
- API key lifecycle：下游 key 签发、吊销、配额、usage 查询和权限边界必须由后端校验，前端隐藏不是安全边界。
- 上游策略：Codex backend、CLI fallback、工具调用、文件/图片/PDF 输入、reasoning override 和超时策略不能互相误降级。
- 余额公开接口：`/public/codex/balance` 只能返回安全字段；不得泄露账号邮箱、access token、refresh token 或请求正文。
- 数据库与 migration：schema 变更必须走 Ent + Atlas；发布前要考虑 pending migration。
- 部署：低配服务器只负责 load/up/migration/smoke，不构建；Atlas 是宿主机工具，不写进业务 Compose。
- 前端样式：后台表单、弹窗、表格、卡片、筛选器和限制配置要同时检查浅色与暗色可读性。
- 错误码：服务端目录、前端生成码表、消费层和测试要同步。
- legacy：`legacy-python-mvp/` 只能作历史参考，不能覆盖当前 Go/React 主路径。

## 验证建议 Validation

- 后端改动至少检查相关 `go test` 覆盖；API/usage/upstream 改动要看 JSON-RPC 或 HTTP 边界测试。
- 前端样式/交互默认考虑 `cd /Users/simon/projects/openai-oauth-api-service/web && pnpm lint && pnpm css && pnpm test && pnpm style:l1`。
- 文档/skill-only 改动至少运行 `git diff --check` 和对应 skill validator。
- 涉及 secrets、部署、OpenAI 官方行为、migration 时，要明确已验证和未验证范围。

## 输出要求 Output

1. Findings first，按严重度排序，带文件行号、影响和建议。
2. 无问题时明确写“未发现阻塞问题”。
3. 写清审查范围、已读真源、已跑或未跑的验证。
4. 单列剩余盲区，尤其是未查 secrets、未跑后端测试、未做暗色样式回归、未核对线上 deployment 或官方 OpenAI 文档的情况。
