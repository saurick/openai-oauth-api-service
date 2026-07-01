---
name: openai-oauth-release-governance
description: 项目发布、部署、版本与回滚治理（openai-oauth-api-service）。Use when Codex plans, performs, reviews, or explains openai-oauth-api-service releases, deploys, image tags, migrations, changelog, rollback, health checks, post-deploy verification, or target environment delivery.
---

# OpenAI OAuth 发布治理 Release Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这个 skill 处理 `openai-oauth-api-service` 的 release、deploy、version、migration、rollback 和 release evidence。版本管理默认并入发布证据，不另起重流程。

## OpenAI OAuth 发布质量门禁 Release Quality Gate

发布治理的质量不是把版本推上去，而是可复现、可回滚、可证明。

### 结构质量检查 Structure Quality Checks

- 边界清晰、合理严谨：说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 模块化：发布脚本、migration、镜像构建、健康检查、回滚、清理和证据归档各有清楚入口，不把手工命令散成隐藏流程。
- 高内聚：同一版本、镜像、migration、env、smoke 和 rollback evidence 绑定在同一 release truth，不在聊天、远端现场和文档里各存一份。
- 低耦合：构建机、目标机、数据库、反代和外部依赖的责任分开；低配目标机不承担本地/CI 应做的重构建。
- 单一职责：发布变更只发布已验证范围；若顺手修代码、改配置或清理现场，必须说明边界和回滚路径。

- 只发布已提交、已验证、已绑定版本证据的范围；不要把未归属 dirty worktree、临时脚本或手工远端改动混进 release truth。
- 低配服务器默认不构建；本地/CI 构建，远端只加载制品、执行 migration、启动、health/ready/smoke 和必要清理。
- 涉及 migration、配置、镜像或数据状态的发布，要说明 rollback point、不可逆风险、前向修复路径和保留证据。
- 运行版本必须用目标 runtime evidence 证明，包括 commit/tag、image、migration 状态、服务健康、日志和业务 smoke。

## 真源链 Truth Chain

- 先读 `AGENTS.md`、`README.md`、`docs/architecture.md`、`docs/operations.md`、server/web/deploy docs 和相关 tests。
- 执行前检查 `git status -sb`、upstream state、unrelated dirty files。

## 项目规则 Project Rules

- 提交推送、hook 重试或多会话同时收口时，先用全局 `$git-closeout-coordination` 判定 `owner`、冻结范围、upstream 和 dirty state，再回到本 skill 执行 openai-oauth release / closeout 证据。
- 133 低配发布主路径是本地构建镜像、上传 tar、远端 `docker load`、Atlas/migration 检查、更新 `APP_IMAGE`、重建 app 容器。
- 已验证目标是 `root@192.168.0.133` 和 `/data/openai-oauth-api-service/compose`。
- 版本证据记录 commit、image tag、`GIT_SHA_SHORT`/`IMAGE_TAG`、health/ready/admin smoke 和 rollback point。

## 工作流 Workflow

1. 定义 scope：branch、host/environment、service/container、migration、config/env、rollback point。
2. 绑定 version：commit hash、image/package tag、migration status、config/env version、release note/changelog need。
3. 提交推送前按 `$git-closeout-coordination` 检查 staged/unstaged/untracked、远端 ahead/behind、hook 改写和并行会话风险。
4. 先跑本地/CI validation，再触碰目标环境；hook、generator 或 formatter 改写文件后必须重新检查 `git status -sb`。
5. 低配目标默认不构建，只加载 artifacts、执行 migration、启动服务、做 health/smoke。
6. 从目标 runtime evidence 确认新版本已运行，而不是从本地预期推断。
7. 检查 health/ready、logs、smoke/browser/API、migration state、disk/image cleanup boundary。
8. 发布行为、版本、部署、配置或 operational truth 改变时，同步 docs/progress。

## 输出 Output

汇报 commit/tag/image、target environment、migration status、commands、health/smoke evidence、rollback point、cleanup、docs/progress updates 和 remaining blind spots。
