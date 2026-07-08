## 归档索引

- 2026-06-04：旧 `progress.md` 已按超过 600 行阈值归档到 `docs/archive/progress-2026-06-04-before-govulncheck.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。
- 2026-06-25：旧 `progress.md` 已按超过 80KB 阈值归档到 `docs/archive/progress-2026-06-25-before-skill-scenario-matrix.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。

## 2026-07-02 Codex 上下文压缩结构化状态包

- 完成：针对 2026-07-03 多轮压缩后 `stopped / must_not_do` 与“继续执行并允许工具调用”同时存在的问题，修正结构化状态包的冲突解析：`latest_user_instruction` 只从 user-eligible 片段提取，不再把 assistant 回复、工具输出、恢复规则说明或旧状态包字段当作最新用户消息。
- 完成：新增 `latest_uncompressed_user_instruction`、`current_effective_constraints` 和 `obsolete_or_superseded_constraints` 字段；当最新未压缩用户消息明确允许继续 / 工具调用时，旧的 `current_task_phase=stopped`、`must_not_do` 和 `requires_user_confirmation_before` 会进入 obsolete，不再阻塞当前任务。
- 完成：`pinned_raw_user_directives` 会按最新指令过滤冲突项，避免同时 pin “继续”和“停止”；`must_not_do` 过滤 `latest_user_instruction`、`must_not_do`、`requires_user_confirmation_before` 等恢复规则说明，避免把状态包字段说明反向污染为当前禁止事项。
- 完成：新增 `TestGatewayContextLatestUserAllowOverridesPreviousStoppedState` 和 `TestCompactGatewayContextTenRoundsKeepsLatestAllowEffective`，覆盖朋友提供的冲突状态包形态，以及 10 轮压缩后最新允许继续仍有效、不继承旧停止约束。
- 验证：已通过 `cd server && go test -count=1 ./internal/server -run 'TestCompactGatewayContext|TestGatewayContext|TestPrepareGatewayContext|TestEffectiveGatewayContextPolicy'`、`cd server && go test -count=1 ./...`、`bash scripts/qa/secrets.sh` 和 `git diff --check`。
- 部署：已按低配发布主路径在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260703T175800-14c7fae1-local`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260703T175800-14c7fae1-local-context-conflict-resolve-v2`；远端只执行 checksum、`docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260703T175800-context-conflict-resolve-v2`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：当前 `app-server` 运行镜像为 `oauth-api-service-server:20260703T175800-14c7fae1-local`，容器环境包含 `GIT_SHA_SHORT=14c7fae1-local` 和 `IMAGE_TAG=20260703T175800-14c7fae1-local`；本机与公网 `/healthz` / `/readyz` 均通过，`codex-runtime-health-check.py --auto-upgrade` 后所有 health checks 为 `ok`，Codex 为 `0.142.5`。
- Windows 10 轮回归：已在 `ssh sauri@192.168.0.45` 上确认 Codex `0.142.5`，且 `npm view @openai/codex version` 返回 `0.142.5`。在 `C:\Users\sauri\codex-compaction-10round-20260703-1805` 初始化 `.git` 目录后，通过 `saurick-oauth` provider 对同一 session `019f2781-35b0-7bb2-98e2-1f4875ab9419` 连续执行 10 轮 `codex exec resume <session_id>`，未使用 `resume --last`。每轮均输出对应 `ROUND*_ALLOW_CONTINUE` marker 和 `ACTION=NO_TOOL`，无 413、无 input too large、无“已停止 / 需要确认 / 无法继续”误停文本。
- 133 数据库证据：`gateway_usage_logs` 近 15 分钟记录 `context_compacted=true` 19 条、413 为 0；原始上下文最大 `10021191` bytes、压缩后最大 `112135` bytes。第 10 轮状态包为 `current_task_phase=executing`、`latest_user_instruction` 保留 `ROUND10_ALLOW_CONTINUE` 用户原话、`must_not_do=0`、`requires_user_confirmation_before=0`，`current_effective_constraints=["latest_user_instruction 明确允许继续执行和工具调用"]`。
- 清理：部署和 10 轮回归通过后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧镜像并回收约 `1.638GB`；未执行 volume prune。根分区从 42% 使用降到 39%，当前 app-server 仍运行新镜像。
- 完成：将网关预压缩摘要从自由 Markdown 摘要改为 `gateway_context_restore_state.v1` JSON 状态包，显式保留 `current_user_goal`、`latest_user_instruction`、`pinned_raw_user_directives`、`must_not_do`、`current_task_phase`、`next_action`、`restore_audit`、`historical_context_only` 和 `obsolete_or_superseded_goals`，避免恢复后从历史路径、日志或旧目标碎片里自行生成新目标。
- 完成：压缩插入提示改为要求先读取 schema 并执行 restore audit；当最新指令包含停止、暂停、只读、不要执行、不要 SSH、不要重启或需要确认等语义时，状态包会把下一步收口为 `no_op` 或 `ask_user`，并把 shell / file write / tool call / ssh / restart 标记为需确认动作。
- 完成：补充单测覆盖最近进度锚点结构化保存、停止后不得继续工具调用的状态表达、重复 compaction 后目标 A 不被历史 bug B/C 噪声污染，以及旧 summary 继续作为 `historical_context_only` 而非当前目标真源。
- 完成：Windows 显式 `resume <session_id>` 第二轮大上下文回归暴露出数组型 Responses 压缩只压旧段落、不压保留下来的超大最新 user item，导致压缩后仍 1.12MB 并返回 413；已改为对保留的最近 Responses items 做二次大内容压缩，让第二轮最新用户指令进入结构化状态包而不是原样撑爆请求。
- 完成：新增 `TestCompactGatewayContextResponsesArrayCompactsHugeLatestResumeInstruction`，覆盖“第一轮已有结构化摘要 + 第二轮 resume 末尾 1MB 级最新用户指令”的目标不漂移和压缩后低于 1MB。
- 完成：二次部署后的 Windows 真实 resume 请求仍返回 413，诊断显示真实请求形态里的第二轮大文本不在原 `responsesItem` 读取路径；已补通用第二道压缩，递归扫描 `input/content/text/arguments/output/instructions` 的超大字符串并跳过 tools / metadata，让非标准位置里的最新用户指令也进入同一个结构化状态包。
- 完成：新增 `TestCompactGatewayContextGenericSecondPassCompactsHugeInstructionString`，覆盖最新大文本落在 `instructions` 这类非标准字段时仍能压缩、保留 marker 并明显收缩请求体。
- 完成：同步 `server/docs/config.md` 的上下文压缩口径，明确网关能稳定传递状态包和禁止事项，但真正拒绝客户端本地工具调用仍取决于 Codex / OpenCode runtime 的工具层执行；本服务不代理本机 shell、文件或 SSH 工具。
- 验证：已通过 `cd server && go test -count=1 ./internal/server -run 'TestCompactGatewayContext|TestPrepareGatewayContext|TestEffectiveGatewayContextPolicy'`、`cd server && go test -count=1 ./...`、`bash scripts/qa/secrets.sh` 和 `git diff --check`。期间全量测试首次因本机 Codex balance 外部依赖瞬时 502 失败，单独重跑该用例和再次全量重跑均通过。后续还需重新构建部署到 133，并在 Windows 端用 `saurick-oauth` provider 重跑显式 resume 压缩恢复回归。
- 部署：已按低配发布主路径在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260702T232900-14c7fae1-local`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260702T232900-14c7fae1-local-context-generic-compact`；远端只执行 checksum、`docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260702T232900-context-generic-compact`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：当前 `app-server` 运行镜像为 `oauth-api-service-server:20260702T232900-14c7fae1-local`，容器环境包含 `GIT_SHA_SHORT=14c7fae1-local` 和 `IMAGE_TAG=20260702T232900-14c7fae1-local`；本机与公网 `/healthz` / `/readyz` 均通过，`codex-runtime-health-check.py --auto-upgrade` 后容器内 Codex 为 `0.142.5` 且所有 health checks 为 `ok`。近 3 分钟 app 日志未见 WARN / ERROR / PANIC / FATAL。
- Windows 回归：已在 `ssh sauri@192.168.0.45` 上确认 Codex `0.142.5`，在 `C:\Users\sauri\codex-compression-regression-20260702` 初始化 git 目录后通过 `saurick-oauth` provider 测试。第一轮大请求 session `019f236e-fb6a-7d02-a145-45cd42734cfe` 输出 `MARKER=WIN_COMPRESS_SCHEMA_V1C` / `ACTION=NO_TOOL`；显式 `codex exec resume 019f236e-fb6a-7d02-a145-45cd42734cfe` 第二轮输出 `MARKER=WIN_RESUME_COMPRESS_SCHEMA_V2C` / `ACTION=NO_TOOL`，未使用 `resume --last`。服务端 `gateway_usage_logs` 记录第二轮原始 `2187420` bytes，压缩后 `104793` bytes，`context_compacted=true` 且状态包包含第二轮 marker。
- 清理：部署和 Windows 回归通过后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧镜像并回收约 `1.06GB`；未执行 volume prune。根分区从约 `55G` 可用恢复到约 `57G` 可用，当前 app-server 仍运行新镜像。
- 下一步：后续若继续收口“停止 / 暂停必须由工具层硬拒绝”，应在 Codex / OpenCode runtime 增加外部状态机和 goal / permission guard；本服务端已把这些字段稳定写入结构化状态包，但不代理客户端本地工具执行。
- 阻塞/风险：本轮代码不改 schema、migration、auth、API key、quota、usage 真源、上游策略或 admin UI；页面只会显示新的结构化压缩摘要文本。客户端 runtime 若忽略状态包，仍可能绕过“停止/暂停”的硬权限语义，需通过 Windows 实测和上游 runtime 修复继续验证。

## 2026-06-29 凭据统计 Token 表头点击排序

- 完成：参考 `trade-erp` 表头排序的“当前排序列 + 升降序状态 + 稳定兜底”模式，为 `/admin-usage` 的「凭据统计」Token 窗口表头增加点击排序；未手动排序时继续沿用今天优先、空窗口自动降级到 24h / 7 天 / 更长窗口的默认降序规则。
- 完成：Token 窗口表头改为可聚焦按钮，显示当前升 / 降序箭头并写入 `aria-sort`；点击任意窗口表头会切换该列升序 / 降序，并把凭据统计分页重置到第一页，避免排序后停留在旧分页看不到结果。
- 完成：同步 `web/README.md` 的凭据统计排序说明，并把 `web/scripts/styleL1.mjs` 的 `admin-usage` 桌面 / 移动场景扩展为真实点击 30 天 Token 表头，验证默认降级排序、降序 / 升序切换、`aria-sort` 和表头宽度不溢出。
- 验证：已通过 `/usr/local/bin/pnpm --dir web lint`、`/usr/local/bin/pnpm --dir web css`、`/usr/local/bin/pnpm --dir web test`、`/usr/local/bin/pnpm --dir web build`、`STYLE_L1_SCENARIOS=admin-usage-desktop,admin-usage-mobile NODE_USE_ENV_PROXY=0 /usr/local/bin/pnpm --dir web style:l1`、`bash scripts/qa/secrets.sh` 和 `git diff --check`。默认 `pnpm` 命中 Codex runtime `pnpm 11.7.0` 时仍会触发本仓库已知 `ERR_PNPM_IGNORED_BUILDS`，已改用稳定 `/usr/local/bin/pnpm 10.13.1` 重跑通过，并把失败过程生成的临时 `web/pnpm-workspace.yaml` 移到废纸篓。
- 部署：已基于提交 `38c0783` 在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260629T231200-38c07837-key-stats-header-sort`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260629T231200-38c07837-key-stats-header-sort`；远端只执行 checksum、`docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260629T231200-38c07837-key-stats-header-sort`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：当前 `app-server` 运行镜像为 `oauth-api-service-server:20260629T231200-38c07837-key-stats-header-sort`，容器环境包含 `GIT_SHA=38c078372d8689b1866229788ff23ea1107b043d`、`GIT_SHA_SHORT=38c07837` 和 `IMAGE_TAG=20260629T231200-38c07837-key-stats-header-sort`；远端本机与公网 `/healthz` / `/readyz` 均通过，`/public/codex/balance` 返回可解析 payload，近 2 分钟 app 日志未见 WARN/ERROR/PANIC/FATAL。生产 Playwright 登录 `https://oauth-api.saurick.me/admin-usage` 后切到「凭据统计」，点击 30 天 Token 表头确认降序 / 升序首行切换、`aria-sort` 为 `descending/ascending`、表头无横向溢出、浏览器控制台无错误。
- 清理：部署验证后删除远端本轮 release 的镜像 / migration 压缩包与校验文件，执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未使用旧镜像 `oauth-api-service-server:20260627T222513-e402593a-dirty-reset-credits`，回收 353.4MB；未执行 volume prune。根分区从约 57G 可用恢复到约 58G 可用，当前 app-server 仍运行新镜像。
- 阻塞/风险：本轮只改管理端前端派生排序和文档，不改 usage 真源、schema、后端 API、鉴权、key 生命周期、上游策略、quota 或 migration。

## 2026-06-27 Codex rate limit reset credits 可见性

- 完成：`GET /public/codex/balance` 在保留原 Codex app-server `account/rateLimits/read` 余额 / 限额主路径的基础上，使用同一服务器 Codex 登录态只读获取 `rate-limit-reset-credits`，并裁剪为 `rate_limit_reset_credits` 摘要；只返回 `reset_type`、`status`、`granted_at`、`expires_at`、`title`、可用数量和累计数量，不返回上游内部 credit id、头像 URL、profile user id、账号邮箱或 token。
- 完成：后台 `/admin-codex-balance` 增加“可用重置券”概览和 reset credits 表格，按北京时间展示获得 / 过期时间；如果重置券读取失败，余额和限额窗口仍正常展示，并在表格区显示暂不可用提示。
- 完成：同步 `README.md`、`server/docs/api.md`、`server/docs/config.md`、生产 Compose 示例和 `compose.yml` 的公开余额接口口径与可选 `CODEX_RATE_LIMIT_RESET_CREDITS_URL` 配置。
- 验证：已通过 `go test -count=1 ./internal/server -run 'TestCodexBalanceRoute'`、`go test -count=1 ./...`、`/usr/local/bin/pnpm --dir web lint`、`/usr/local/bin/pnpm --dir web css`、`/usr/local/bin/pnpm --dir web test`、`/usr/local/bin/pnpm --dir web build`、`STYLE_L1_SCENARIOS=admin-codex-balance-desktop,admin-codex-balance-mobile NODE_USE_ENV_PROXY=0 /usr/local/bin/pnpm --dir web style:l1`、`bash scripts/qa/secrets.sh`、`docker compose --env-file server/deploy/compose/prod/.env.example -f server/deploy/compose/prod/compose.yml config -q` 和 `git diff --check`。Codex runtime 自带 `pnpm 11.7.0` 首次触发 `ERR_PNPM_IGNORED_BUILDS`，已按本仓库稳定路径改用 `/usr/local/bin/pnpm 10.13.1` 重跑通过，并删除生成的临时 `web/pnpm-workspace.yaml`。
- 部署：本地构建 linux/amd64 镜像 `oauth-api-service-server:20260627T222513-e402593a-dirty-reset-credits`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260627T222513-e402593a-dirty-reset-credits`；同步远端 compose `CODEX_RATE_LIMIT_RESET_CREDITS_URL` 环境入口，远端只执行 `docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260627T222513-e402593a-dirty-reset-credits`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0；首次重建被 shell 中旧 `APP_IMAGE` 覆盖，已立即用显式 `APP_IMAGE=oauth-api-service-server:20260627T222513-e402593a-dirty-reset-credits` 重建修正。
- 线上验证：当前 `app-server` 运行镜像为 `oauth-api-service-server:20260627T222513-e402593a-dirty-reset-credits`，容器环境含 `GIT_SHA=e402593ae9bbbeb4f399103b73a6e187dff38c84`、`GIT_SHA_SHORT=e402593a-dirty`、`IMAGE_TAG=20260627T222513-e402593a-dirty-reset-credits` 和 `CODEX_RATE_LIMIT_RESET_CREDITS_URL=https://chatgpt.com/backend-api/wham/rate-limit-reset-credits`；远端本机和公网 `/healthz` / `/readyz` 均通过，`/public/codex/balance` 返回 `rate_limit_reset_credits.status=ok`、`available_count=3`，单条字段只含 `expires_at/granted_at/reset_type/status/title`。生产 Playwright 登录 `/admin-codex-balance` 后确认页面显示 3 条 `Full reset (Weekly + 5 hr)`、无 `RateLimitResetCredit_` / `Codex Team` 泄漏、无页面级横向溢出、控制台无错误。
- 清理：部署验证后删除远端本轮 release tar 包，执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧镜像 `oauth-api-service-server:20260626T191039-3c31ee33-dirty-large-guard-30`，回收 353.3MB；未执行 volume prune。根分区从约 59G 可用恢复到约 60G 可用，当前 app-server 仍运行新镜像。
- 阻塞/风险：该信息仍来自当前服务器 Codex 登录态和 ChatGPT 后端只读接口；如上游接口字段或权限变化，页面会显示重置券暂不可用，但不会影响原余额 / 限额窗口展示。

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

## 2026-07-01 prompt skill 工程质量门禁

- 完成：补强 `openai-oauth-prompt-governance`，要求生成实现 / 管理端 / 文档 / 测试 / 部署 / review 提示词时显式包含 Engineering Quality Gate：复用现有 auth、API key、quota、usage logging、admin UI、proxy/upstream、deploy 和 health/ready 结构；新增抽象 / 配置 / fallback / upstream 策略 / 缓存 / migration / 部署步骤前说明复用不足、安全影响和运维影响。同步 UI metadata 加入工程质量门禁和复杂度预算。
- 验证：追加前 `progress.md` 为 51 行、13092 字节，未达到归档阈值；本组已执行 YAML 解析、等价 skill metadata 校验和限定 `git diff --check`。
- 下一步：后续 openai-oauth 提示词把“请求可用 / 做完整 / 稳定”落成 auth/quota/usage/error/deploy 边界、复杂度预算、可观测证据和验证命令，不用宽松 fallback 掩盖真实上游或密钥问题。
- 阻塞/风险：本组只改 skill 文档、UI metadata 和过程记录，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署脚本、监控系统、真实上游验证或 133 环境。

## 2026-07-01 项目治理 skills 质量门禁同步

- 完成：同步 `openai-oauth-*` 项目治理 skills 的质量门禁。docs/page/domain/release/test/code-review 正文补齐质量门禁；runtime/observability/security 等默认提示词补齐根因、可观测、安全质量锚点，触发 `$openai-oauth-*` 时默认关注 OAuth/API key/usage/upstream 真源、secrets、低配发布证据、测试可信度和管理端可读性。
- 下一步：若后续新增 seed/import 类项目 skill，再按本项目真实数据导入边界单独设计，不从 ERP 项目复制。
- 阻塞/风险：本组只改 `.agents/skills` 和 `progress.md`；不改 runtime、schema、auth、API key、usage、上游策略、部署或生产配置。

## 2026-07-01 governance skills 结构质量门禁

- 完成：补强 `openai-oauth-*` 治理 skills 的结构质量检查，明确模块化、高内聚、低耦合、单一职责；管理端页面、OAuth/API key/usage/upstream、运行时诊断、可观测性、安全、发布和测试分别保留项目语义。
- 完成：同步 `agents/openai.yaml` 默认提示词，让 `$openai-oauth-*` 默认把质量门禁理解为包含模块化、高内聚、低耦合和单一职责。
- 验证：追加前 `progress.md` 为 64 行、15017 字节，未达到归档阈值；Ruby YAML 解析通过 88 个 `agents/openai.yaml`；结构/frontmatter 扫描通过 54 个目标 skill；`quick_validate.py` 因当前 Python 环境缺 `yaml`/PyYAML 失败，已按依赖缺口记录。
- 下一步：后续 openai-oauth skill 继续围绕 auth/quota/usage/error/deploy 边界补充，不从 ERP 或模板项目复制业务事实。
- 阻塞/风险：本组只改 `.agents/skills` 和 `progress.md`；不改 runtime、schema、auth、API key、usage、上游策略、部署或生产配置。

## 2026-07-01 governance skills 边界清晰与合理严谨门禁

- 完成：在 `openai-oauth-*` 项目治理 skills 的结构质量检查中补入一条短门禁：边界清晰、合理严谨；要求说明本轮管什么、不管什么、依赖哪个真源，以及为什么当前拆分、抽象和验证足够但不过度。
- 完成：同步 `agents/openai.yaml` 默认提示词，让 `$openai-oauth-*` 的质量门禁显式包含边界清晰、合理严谨、模块化、高内聚、低耦合、单一职责。
- 下一步：后续 openai-oauth skill 继续围绕 auth、API key、quota、usage、upstream、deploy 和 secrets 边界补充，不复制 ERP 或模板项目事实。
- 阻塞/风险：追加前 `progress.md` 为 72 行、16097 字节，未达到归档阈值。本组只改 `.agents/skills` 和 `progress.md`；不改 runtime、schema、auth、API key、usage、上游策略、部署或生产配置。

## 2026-07-02 governance skills 语义清晰门禁

- 完成：在 `openai-oauth-*` 项目治理 skills 的结构质量检查中补入类型化短门禁：语义清晰；覆盖文档、管理端页面、业务边界、代码审查、测试、提示词、运行时诊断、可观测错误、安全和发布，不改变 skill 名称、职责或触发边界。
- 完成：同步 `agents/openai.yaml` 默认提示词，让 `$openai-oauth-*` 的质量门禁显式包含语义清晰，避免 auth、API key、quota、usage、upstream、错误、日志、发布证据或管理端页面含义被泛称掩盖。
- 验证：追加前 `progress.md` 为 79 行、17021 字节，未达到归档阈值；已执行 54 个目标 skill 的语义门禁/metadata 扫描和 54 个 `agents/openai.yaml` Ruby YAML 解析，均通过。
- 下一步：后续 openai-oauth skill 继续围绕 auth、API key、quota、usage、upstream、deploy 和 secrets 语义补充，不复制 ERP 或模板项目事实。
- 阻塞/风险：本组只改 `.agents/skills` 和 `progress.md`；不改 runtime、schema、auth、API key、usage、上游策略、部署或生产配置。

## 2026-07-02 governance skills 职业任务文案门禁

- 完成：在 `openai-oauth-*` 相关治理 skills 中补入“职业任务文案”门禁，覆盖管理端页面、文档、提示词、代码审查、测试和可观测/错误提示；要求用户可见页面、帮助、错误提示和管理端说明用目标角色能理解的业务语言，不把内部实现细节直接暴露给非开发读者。
- 下一步：后续管理端、错误提示、帮助文档或提示词生成时，区分管理员/运维/开发读者；内部 error code、request_id、上游细节和 SQL/API 证据留给日志、诊断和开发文档。
- 阻塞/风险：追加前 `progress.md` 为 87 行、18165 字节，未达到归档阈值。本组只改 `.agents/skills` 和 `progress.md`；不改 runtime、schema、auth、API key、usage、上游策略、部署或生产配置。

## 2026-07-07 本地 Vite HMR / proxy IPv4 固定

- 完成：排查 `/Users/simon/projects` 下同类 Vite dev runtime 风险后，将 `web/vite.config.mjs` 的 HMR 目标固定为 `127.0.0.1:5176`，并把本地 API proxy 默认目标从 `localhost:8400` 收口到 `127.0.0.1:8400`。
- 下一步：后续如通过 `VITE_API_PROXY_TARGET` 指向非本机后端，可继续显式覆盖；本轮只改变未配置时的本地默认值。
- 阻塞/风险：追加前 `progress.md` 为 121 行、28557 字节，未达到 600 行或 80KB 归档阈值。本轮只改本地开发 Vite 配置，不改 OAuth/API key/usage 业务逻辑、schema、生产部署或正式文档。

## 2026-07-08 本地 Vite 开发入口 IPv4 统一

- 完成：继续收口本地 Vite dev origin：`web/vite.config.mjs` 保留 `host: 0.0.0.0` 和局域网 `Network` 地址，但将自动打开地址、终端 `Local:` 打印和 `localhost:5176` 页面访问统一规范到 `http://127.0.0.1:5176`；同步更新 `web/README.md` 默认本地地址和 proxy 默认值说明。
- 下一步：后续若改前端端口或通过 `VITE_API_PROXY_TARGET` 指向其他后端，继续保持本机默认入口使用明确 IPv4 loopback。
- 阻塞/风险：追加前 `progress.md` 为 127 行、29229 字节，未达到 600 行或 80KB 归档阈值。本轮只改本地开发 Vite 配置和前端 README，不改 OAuth、API key、usage、上游策略、schema、生产部署或真实密钥。
