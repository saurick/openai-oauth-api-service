---
name: openai-oauth-release-governance
description: Project-specific release, deployment, version, migration, rollback, and release-evidence governance for openai-oauth-api-service. Use when Codex plans, performs, reviews, or explains openai-oauth-api-service releases, deploys, version tags, image tags, migrations, release notes, changelog, rollback, production preflight, health checks, post-deploy verification, or target environment delivery.
---

# OpenAI OAuth Release Governance

Use this skill for openai-oauth-api-service release, deployment, and lightweight version governance. Version management is part of release evidence unless the project later needs a standalone customer-facing release program.

## Truth Chain

- Read project `AGENTS.md`, `README.md`, deployment docs, test strategy, and changed release scripts before action.
- Check worktree and upstream before commit/push/deploy.

## Project Rules

- 133 低配发布主路径是本地构建镜像、上传 tar、远端 docker load、Atlas/migration 检查、更新 `APP_IMAGE`、重建 app 容器。
- 已验证目标为 `root@192.168.0.133` 和 `/data/openai-oauth-api-service/compose`。
- 版本证据记录 commit、image tag、`GIT_SHA_SHORT`/`IMAGE_TAG`、health/ready/admin smoke 和回滚点。

## Workflow

1. Define scope: target branch, target host/environment, service/container, migration, config/env, and rollback point.
2. Bind version: commit hash, image/package tag, migration status, config/env version, and release note/changelog need.
3. Run local/CI validation appropriate to changed surfaces before touching a target environment.
4. Build artifacts off low-spec targets unless project docs explicitly allow target-side build.
5. Deploy using the documented path; confirm the target is running the new version from runtime evidence.
6. Check health/ready, logs, smoke/browser/API evidence, migration state, and disk/image cleanup boundaries.
7. Update progress/docs when release behavior, versioning, deployment, config, or operational truth changes.

## Output

Report commit/tag/image, target environment, migration status, commands, health/smoke evidence, rollback point, cleanup, docs/progress updates, and remaining blind spots.
