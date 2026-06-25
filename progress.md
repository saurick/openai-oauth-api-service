## 归档索引

- 2026-06-04：旧 `progress.md` 已按超过 600 行阈值归档到 `docs/archive/progress-2026-06-04-before-govulncheck.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。
- 2026-06-25：旧 `progress.md` 已按超过 80KB 阈值归档到 `docs/archive/progress-2026-06-25-before-skill-scenario-matrix.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。

## 2026-06-25 Codex skills 使用场景速查补充

- 完成：补充根 `README.md` 的 `.agents/skills/` 导航，并完善 `.agents/skills/README.md` 的“按问题选 Skill / Scenario Matrix”，把选中文本分析、提示词、runtime 诊断、测试范围、代码 review、文档治理、管理端页面、服务边界、发布、通用 seed/import、可观测错误和安全隐私按常见提问方式映射到对应 skill。
- 完成：保留本项目没有专属 seed/import skill 的边界，导入 / fixture / cleanup 类临时任务继续指向通用 `$seed-import-governance`，避免把 openai-oauth-api-service 误判为 ERP 导入系统。
- 验证：本轮开始前 `progress.md` 为 373 行、86874 字节，已先归档再新建当前记录；本轮只改根 README、skill 目录 README、progress 归档和过程记录，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署脚本、监控系统或安全策略。
- 下一步：后续 openai-oauth 任务先按当前问题选择一个主 skill；涉及 gateway / upstream / usage / deploy / security 边界时，再同时 `$` 相邻 skill。
- 阻塞/风险：README 只负责选型导航，不替代各 skill 的 `SKILL.md`、项目 `AGENTS.md`、正式 docs、代码、runtime 证据或自动化校验。

## 2026-06-25 Git closeout coordination skill 接入

- 完成：新增全局 `/Users/simon/.codex/skills/git-closeout-coordination/`，用于提交推送、多会话同时收口、hook/lint/test 反复失败时先判定 owner、冻结范围、upstream/dirty 状态和停止条件。
- 完成：在 `.agents/skills/README.md` 增加 `$git-closeout-coordination` + `$openai-oauth-release-governance` 场景入口；`openai-oauth-release-governance` 增加提交推送前先走全局协调、hook/generator/formatter 改写后重查 `git status -sb` 的项目差异规则。
- 验证：追加前 `progress.md` 为 12 行、1793 字节，未达到归档阈值；已执行全局 skill 与 `openai-oauth-release-governance` 的 `quick_validate.py`、`agents/openai.yaml` Ruby YAML 解析、TODO 扫描和限定 `git diff --check`，均通过。
- 下一步：后续 openai-oauth 提交推送相关 / 所有代码，尤其多会话、脏工作区、hook 反复失败或 133 发布前收口时，先 `$git-closeout-coordination`，再按 `$openai-oauth-release-governance` 和 `$openai-oauth-test-governance` 选择项目命令。
- 阻塞/风险：本轮只改全局 skill、项目 skill README、release skill 和过程记录，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署脚本、监控系统、真实上游验证或 133 环境。
