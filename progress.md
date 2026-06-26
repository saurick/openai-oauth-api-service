## 归档索引

- 2026-06-04：旧 `progress.md` 已按超过 600 行阈值归档到 `docs/archive/progress-2026-06-04-before-govulncheck.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。
- 2026-06-25：旧 `progress.md` 已按超过 80KB 阈值归档到 `docs/archive/progress-2026-06-25-before-skill-scenario-matrix.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。

## 2026-06-26 单 key 30 并行会话大请求保护放宽

- 诊断：朋友反馈的 `429 Too Many Requests` 仍集中在单个下游 key 的大上下文请求；真实使用方式是每个人一个 key，但同一个 key 会同时开多路 Codex / OpenCode 会话，最新要求按 30 个并行会话承载。过去 24 小时该 key 的 `/v1/responses` / `/v1/chat/completions` 请求没有稳定 `session_id`，因此无法立即按真实会话分桶，只能继续按 key 级大请求并发和突发阈值放宽。
- 完成：将服务端默认、生产 Compose 默认、`.env.example` 和部署说明调整为 `GATEWAY_LARGE_REQUEST_MAX_INFLIGHT_PER_KEY=30`、`GATEWAY_LARGE_REQUEST_BURST_MAX_PER_KEY=120`、`GATEWAY_LARGE_REQUEST_BURST_WINDOW_SECONDS=60`、`GATEWAY_LARGE_REQUEST_MIN_BYTES=65536`；同步 config 文档，明确同一 key 可以承载更多并行会话，但仍保留异常循环保护。
- 部署：133 先前临时经历 `1 / 8 / 60s / 65536B` 与 `10 / 40 / 60s / 65536B` 两轮放宽；本轮备份 `.env` 为 `.env.bak.20260626T111203-large-guard-30`，本地构建 linux/amd64 镜像 `oauth-api-service-server:20260626T191039-3c31ee33-dirty-large-guard-30`，上传到 `/data/openai-oauth-api-service/releases/20260626T191039-3c31ee33-dirty-large-guard-30/app-server-image.tar.gz`，远端只执行 `docker load`、更新 `APP_IMAGE` 和重建 `app-server`，未在 133 构建。
- 验证：133 新镜像运行后 `healthz=ok`、`readyz=ready`、`GET /public/codex/balance` 返回 200，容器环境确认为 `30 / 120 / 60s / 65536B`，`IMAGE_TAG/GIT_SHA_SHORT/GIT_SHA` 与新镜像一致，近 2 分钟 app 日志未见 WARN/ERROR/PANIC/FATAL。根分区清理前约 59G 可用；验证通过后执行 `docker image prune -a -f` 和 `docker builder prune -f`，回收 706.7MB，清理后约 60G 可用，未清理 volume。
- 验证：本地已通过 `cd server && go test ./internal/server -run 'TestGatewayLargeRequest'`、`docker compose --env-file server/deploy/compose/prod/.env.example -f server/deploy/compose/prod/compose.yml config -q`、`bash scripts/qa/secrets.sh` 和 `git diff --check`。
- 阻塞/风险：本轮按用户要求放宽单 key 多会话能力，但不关闭保护；如果同一 key 后续在 60 秒内超过 120 个大请求，仍会返回 `gateway_large_request_burst`。Codex / OpenCode 当前没有传稳定 `session_id`，后台会话聚合和真正的会话级限流仍无法生效；后续若客户端能传 `X-Session-ID`、`session_id`、`conversation_id` 或 `thread_id`，再把保护细化到会话级会更精确。

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
