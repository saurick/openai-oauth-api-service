# Codex 项目 Skills / Project Skills

本目录保存 openai-oauth-api-service 的项目专属 Codex skills，是仓库内 canonical 版本。全局 `~/.codex/skills` 只放通用范式；涉及本项目时优先用这里的 `$openai-oauth-*` skills。

## 使用入口 / Usage

| Skill | 主要用途 |
| --- | --- |
| `$openai-oauth-docs-governance` | 文档真源、architecture / operations / deploy、admin 可见性、low-spec 和进度记录 |
| `$openai-oauth-page-governance` | 管理后台、usage、upstream strategy、API key、登录、暗色模式和 L1 回归 |
| `$openai-oauth-code-review-governance` | 独立代码审查、OAuth、API key、usage、gateway、Codex backend、secrets 和部署 |
| `$openai-oauth-test-governance` | Go/web/admin UI、migration、auth/API-key/quota/usage、secrets 和 deploy preflight |
| `$openai-oauth-prompt-governance` | 新会话、side chat、review、实现、测试、部署和提交推送提示词 |
| `$openai-oauth-release-governance` | 133 低配发布、本地构建、上传 tar、`APP_IMAGE`、health/ready/admin smoke 和 rollback |
| `$openai-oauth-domain-boundary-governance` | OAuth/auth、gateway/proxy、upstream provider、admin API、usage logging 和 persisted config |
| `$openai-oauth-runtime-diagnostics` | 502、balance、usage、`gateway_usage_logs`、request_id/session_id、container logs 和 stale fallback |
| `$openai-oauth-observability-error-governance` | request logs、upstream error、latency、`stale=true`、dashboard 字段顺序和排障证据 |
| `$openai-oauth-security-privacy-governance` | API keys、OAuth tokens、upstream credentials、admin access、request logs 和脱敏 |

## 常用组合 / Pairings

| 场景 | 建议同时使用 |
| --- | --- |
| 文档改动会影响管理端页面、admin 可见性或 low-spec 说明 | `$openai-oauth-docs-governance` + `$openai-oauth-page-governance` |
| 管理端页面改动涉及 auth、API key、quota、usage、gateway 或 persisted config | `$openai-oauth-page-governance` + `$openai-oauth-domain-boundary-governance` |
| 实现完成后做独立 review 或提交前自查 | `$openai-oauth-code-review-governance` + `$openai-oauth-test-governance` |
| 502、balance、usage、container 或部署故障排查后准备发布 / 回滚 | `$openai-oauth-runtime-diagnostics` + `$openai-oauth-release-governance` |
| key、token、admin access、request logs 或脱敏边界相关 | `$openai-oauth-security-privacy-governance` + `$openai-oauth-observability-error-governance` |

## 使用规则 / Rules

- 在 Codex 会话里直接写 `$skill-name` 即可触发，例如 `$openai-oauth-docs-governance`；一次任务经常跨边界时，可以在同一条消息里同时写多个 skill。
- 先选最贴近本轮主任务的 skill，再按影响面补相邻 skill：文档 + 管理端页面用 docs/page，页面 + 服务边界用 page/domain，发布故障用 release/runtime，涉及 key、token 或 admin 权限再加 security。
- 涉及 openai-oauth-api-service 时优先使用本目录 `$openai-oauth-*` 项目版；只有缺少项目专属能力，或任务明确跨项目通用，才退回 `~/.codex/skills` 的通用版。
- 本 README 只负责选型和导航；真正执行前必须读对应 skill 的 `SKILL.md`，不要只按 README 摘要执行。
- 修改 skill 本身时同步检查 `SKILL.md`、`agents/openai.yaml`、触发名和 UI 摘要；只改目录 README 不代表更新了任何 skill workflow。

## 维护规则 / Maintenance

- 单个 skill 的入口必须是它自己的 `SKILL.md`；不要在每个 skill 子目录再加 README、quick reference 或 changelog。
- 新增或修改 skill 时保持 `name`、目录名和 UI `display_name` 英文稳定；`description`、正文、`short_description` 和 `default_prompt` 使用中文主体 + English anchors。
- 本项目没有项目专属 seed/import skill；导入类临时任务使用通用 `$seed-import-governance`，避免把服务误判为 ERP 导入系统。
- 只改 skill/docs 时默认跑 skill validator、YAML 解析、`git diff --check` 和必要引用扫描，不机械运行真实上游或远端部署 smoke。
- 修改本目录后按项目约定更新 `/Users/simon/projects/openai-oauth-api-service/progress.md`。
