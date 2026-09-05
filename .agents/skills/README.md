# Codex 项目 Skills / Project Skills

本目录只保存 openai-oauth-api-service 的专项 SOP。长期规则在 `AGENTS.md`，服务事实在正式 docs、代码、migration、测试和运行证据；通用工作流使用 `~/.codex/skills`。

| Skill | 适用范围 |
| --- | --- |
| `$openai-oauth-code-review-governance` | review OAuth、API key、gateway、usage、secrets 和部署改动 |
| `$openai-oauth-docs-governance` | architecture/operations/deploy、low-spec 和 `progress.md` |
| `$openai-oauth-domain-boundary-governance` | OAuth/auth、gateway/upstream、admin API、usage 与 persisted config |
| `$openai-oauth-page-governance` | 管理后台、usage、API key、登录、暗色和 L1 回归 |
| `$openai-oauth-test-governance` | Go/web/admin UI、auth/key/quota/usage、migration 和 preflight |
| `$openai-oauth-operations-governance` | 502/balance/usage 诊断、stale/日志、安全、低配发布和回滚 |

## 选择规则

从当前任务的 checkout / Worktree 内用 `git rev-parse --show-toplevel` 核对仓库根；下文命令以该根目录为工作目录，仓库内文件引用也相对它解析。

只执行当前目标相关的分支，切换 Skill 后继续已授权工作；入口和元数据保持一致，未变化的内容不重复加载。

- 简单任务只选一个主 skill；跨页面、服务边界或运行环境时再组合。
- 提示词整理使用全局显式 `$prompt-governance`；临时 fixture/import 使用通用 `$seed-import-governance`；当前任务已授权 commit / push 且现场复杂时使用 `$git-closeout-coordination`。
- 项目 skill 只保留服务真源、判断流程、命令和验收，不复制通用工程常识。
- 修改 skill 后同步 metadata，运行 validator、`node scripts/qa/skill-health.mjs` 与 `git diff --check`，仅在命中 `AGENTS.md` 的过程记录条件时更新 `progress.md`。
