---
name: openai-oauth-release-governance
description: openai-oauth-api-service 项目发布、部署、版本与回滚治理。Use when Codex plans, performs, reviews, or explains openai-oauth-api-service releases, deploys, image tags, migrations, changelog, rollback, health checks, post-deploy verification, or target environment delivery.
---

# OpenAI OAuth 发布治理 Release Governance

阅读口径：正文默认中文主线 + English anchors；`name` / `display_name` 保持英文，`Workflow / Fact / RBAC / API / migration / runtime` 等术语按需保留，方便触发、检索和跨工具引用。

用这个 skill 处理 `openai-oauth-api-service` 的 release、deploy、version、migration、rollback 和 release evidence。版本管理默认并入发布证据，不另起重流程。

## 真源链 Truth Chain

- 先读 `AGENTS.md`、`README.md`、`docs/architecture.md`、`docs/operations.md`、server/web/deploy docs 和相关 tests。
- 执行前检查 `git status -sb`、upstream state、unrelated dirty files。

## 项目规则 Project Rules

- 133 低配发布主路径是本地构建镜像、上传 tar、远端 `docker load`、Atlas/migration 检查、更新 `APP_IMAGE`、重建 app 容器。
- 已验证目标是 `root@192.168.0.133` 和 `/data/openai-oauth-api-service/compose`。
- 版本证据记录 commit、image tag、`GIT_SHA_SHORT`/`IMAGE_TAG`、health/ready/admin smoke 和 rollback point。

## 工作流 Workflow

1. 定义 scope：branch、host/environment、service/container、migration、config/env、rollback point。
2. 绑定 version：commit hash、image/package tag、migration status、config/env version、release note/changelog need。
3. 先跑本地/CI validation，再触碰目标环境。
4. 低配目标默认不构建，只加载 artifacts、执行 migration、启动服务、做 health/smoke。
5. 从目标 runtime evidence 确认新版本已运行，而不是从本地预期推断。
6. 检查 health/ready、logs、smoke/browser/API、migration state、disk/image cleanup boundary。
7. 发布行为、版本、部署、配置或 operational truth 改变时，同步 docs/progress。

## 输出 Output

汇报 commit/tag/image、target environment、migration status、commands、health/smoke evidence、rollback point、cleanup、docs/progress updates 和 remaining blind spots。
