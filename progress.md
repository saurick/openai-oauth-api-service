## 归档索引

- 2026-06-04：旧 `progress.md` 已按超过 600 行阈值归档到 `docs/archive/progress-2026-06-04-before-govulncheck.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。
- 2026-06-25：旧 `progress.md` 已按超过 80KB 阈值归档到 `docs/archive/progress-2026-06-25-before-skill-scenario-matrix.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。

## 2026-07-11 GPT-5.5 / GPT-5.6 四模型收口

- 完成：固定模型目录收口为 `gpt-5.6-sol`、`gpt-5.6-terra`、`gpt-5.6-luna`、`gpt-5.5`，默认仍为 Sol；启动同步沿用现有主路径，删除目录外模型及其价格、模型策略和 key `allowed_models` 残值，不改写历史 usage。
- 完成：四模型官方 context window 统一为 1,050,000 tokens；Luna Standard 单价补为输入 `$1`、缓存输入 `$0.1`、输出 `$6`，前端固定目录、上下文帮助、模型管理 mock、API/config/web 文档同步收口。超过 272K 输入的长上下文加价、cache write、Batch、Flex、Priority 与区域处理仍不进入现有三字段估算。
- 本地工具：本机 Codex CLI 与 npm stable latest 均为 `0.144.1`；使用本机 ChatGPT 登录态直接调用 `gpt-5.6-luna` 已返回指定 marker `LUNA_DIRECT_56_OK`，确认当前账号 rollout 已不再是上一轮 404。
- 验证：`bash scripts/qa/full.sh` 全部通过，包含 secrets、错误码同步、govulncheck（可达漏洞 0）、前端 lint/css/test/build、全量 Go test/build；`style:l1` 的 `admin-models-desktop` / `admin-models-mobile` 通过，覆盖浅色、暗色、上下文弹窗、桌面和移动盒模型。
- 下一步：以本提交构建 linux/amd64 镜像并按低配发布路径部署 133；部署后核对数据库仅剩四模型及关联残值清理，再分别验证四模型真实请求，并运行多组同 session 连续压缩、跨 session 隔离与数据库摘要事实 / 当前目标对齐回归。
- 阻塞/风险：本轮不改 schema、migration、auth、API key 生命周期、quota、usage 历史、上游重试或代理策略。生产数据库若保留旧的模型级 context override，需要在部署后按当前官方窗口复核并清除过期覆盖值；压缩阈值仍维持 260K / 380K 与 1.0MB / 1.9MB 的既有安全口径。

## 2026-07-11 GPT-5.6 模型目录同步

- 完成：按 OpenAI 官方模型文档与真实 ChatGPT Codex backend 回归，将默认模型从 `gpt-5.5` 更新为 `gpt-5.6-sol`，新增已真实验证的 `gpt-5.6-terra`，两档官方 context window 均为 1,050,000 tokens；无后缀 `gpt-5.6` 虽是 OpenAI API 的 Sol alias，但 Codex backend 会返回“不支持 ChatGPT account”，因此不进入本项目模型目录。官方 `gpt-5.6-luna` 在当前账号仍返回 404 `Model not found`，暂缓接入。保留 `gpt-5.5` 和既有旧模型，避免现有 key 的 `allowed_models`、模型策略与历史 usage 被启动清理。
- 完成：服务端内置价格表同步 GPT-5.6 Sol / Terra 的 Standard 短上下文输入、缓存输入、输出单价；前端固定目录、模型管理 mock、Codex/opencode 客户端配置模板和 API/config 文档同步更新。长上下文加价、cache write、Batch、Flex、Priority 与区域处理仍不进入当前费用估算，避免把未建模计费维度混入现有三字段价格真源。
- 完成：生产页面回归发现模型表已显示 1,050,000 tokens，但问号说明仍残留“默认 400K”旧口径；已改为按模型目录继承，并明确 GPT-5.6 为 1.05M、旧模型按各自目录值，避免可见数据与帮助文案冲突。
- 本地工具：本机 Codex CLI 已从 `0.143.0` 更新到 npm stable latest `0.144.1`，并安装官方 OpenAI Developer Docs MCP；仓库镜像真源 `server/Dockerfile` 已是 `@openai/codex@0.144.1`，本轮无需重复改版本。
- 验证：已通过 `go test -count=1 ./...`、`/usr/local/bin/pnpm --dir web lint`、`css`、`test`、`build`；`style:l1` 通过模型管理、后台客户端配置、公开客户端配置的桌面 / 移动 6 个场景，覆盖模型上下文弹窗浅色 / 暗色、保存 / 恢复态、表格盒模型与 `gpt-5.6-sol` 配置生成。
- 部署：最终提交 `8944b19679398b75a30bb8056d35aca6f28fca80` 在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260711T120343-8944b196-gpt-5.6-available`；远端校验镜像与 migration 包 SHA-256 后执行 `docker load`、宿主机 Atlas status、备份 `.env`、更新 `APP_IMAGE` 和仅重建 `app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0，容器内 Codex CLI 为 `0.144.1`。
- 线上验证：远端本机与公网 `/healthz` / `/readyz` 均通过，`/public/codex/balance` 与 Codex runtime health 均为 `status=ok`，runtime 记录 `before/latest=0.144.1`、`action=already_latest`。本机 Codex CLI `0.144.1` 经 `saurick-oauth` provider 使用 `gpt-5.6-sol` 新建 thread `019f4f4c-7d9b-7df0-86cd-11a900b20111` 后返回指定 marker，并两次显式 `codex exec resume <thread_id>` 成功；`gpt-5.6-terra` 独立 marker 请求也成功。usage 落库确认 Sol / Terra 为 200；Luna 的 404 证据用于阻止未开放模型进入正式目录。生产 Playwright 登录 `/admin-models` 后在暗色模式确认可用模型、价格、1,050,000 context 和新帮助文案均正确。
- 清理与回滚：多轮真实回归收口后执行 `docker image prune -a -f` 与 `docker builder prune -f`，累计回收约 1.91GB，根分区可用空间回升到约 52GB；未执行 volume prune，当前容器仍运行最终镜像。最终 release 包与 `.env` 备份保留，回滚时可重新 `docker load` 上一 release 并恢复 `APP_IMAGE`。
- 阻塞/风险：本轮不改 schema、migration、auth、API key 生命周期、quota、usage 历史、上游重试或代理策略。GPT-5.6 超过 272K 输入的长上下文计费与 cache write 尚未进入现有费用估算字段，后台估算不覆盖这两类附加费用。本机旧 `[profiles.saurick]` 配置仍可正常加载，但 Codex CLI `0.144.1` 的新 `-p` profile v2 参数会要求独立 `~/.codex/saurick.config.toml`；本轮为避免扩大个人配置迁移范围，真实回归使用显式 `model_provider="saurick-oauth"` 覆盖。

## 2026-07-10 Codex 余额瞬时上游失败恢复与 runtime 更新

- 诊断：截图对应的红色失败不是 `account/rateLimits/read` 协议字段变更。官方 Codex App Server 当前仍将该方法定义为 ChatGPT 限额读取入口；133 日志记录的是调用 `https://chatgpt.com/backend-api/wham/usage` 时的 `error sending request`。同一生产接口随后恢复为 HTTP 200，且 payload 已包含新 `codex_bengalfox`（`GPT-5.3-Codex-Spark`）限额分组；后台现有动态卡片会正常展示它。
- 完成：`readCodexRateLimits` 对 `error sending request`、连接重置/拒绝、`unexpected EOF` 与 TLS handshake timeout 等短暂网络错误，在同一个 15 秒总预算内延迟 250ms 后只重试一次；认证、协议和其他错误不重试。最终失败日志补充 `attempts`，缓存与 `stale=true` 语义不变，不把失败伪装成实时成功。
- 完成：镜像内固定 Codex CLI 从 `0.143.0` 提升到 npm latest `0.144.1`，避免容器重建后再依赖健康脚本的临时升级。
- 验证：已通过 `cd server && go test -count=1 ./...`、`cd server && go test -count=1 ./internal/server -run 'TestCodexBalanceRoute'`、`bash scripts/qa/secrets.sh`、`git diff --check`；新增 fake app-server 回归覆盖“首读网络发送失败、二次读取成功”返回 200。已用本地 linux/amd64 Docker 构建并在成品中确认 `codex-cli 0.144.1`。首次构建被本机代理传入 Docker 后导致 Alpine TLS EOF，清除本次构建命令的代理环境后构建通过，未改动系统代理。
- 部署：已提交功能版本 `b5ba554be431423b7c83b57d65f209c8d47883c0`，本地构建 linux/amd64 镜像 `oauth-api-service-server:20260710T215258-b5ba554-codex-balance-retry` 并上传 133。远端校验镜像与迁移压缩包 SHA-256 后执行 `docker load`、宿主机 Atlas status、备份 `.env` 为 `.env.bak.20260710T215258-b5ba554-codex-balance-retry`、更新 `APP_IMAGE` 和仅重建 `app-server`，未在 133 构建；Atlas 当前版本 `20260604123931`、pending 0。首次重建被远端 shell 继承的旧 `APP_IMAGE` 覆盖，已立即使用显式新镜像变量重建修正，未影响 PostgreSQL。
- 线上验证：当前 app-server 运行 `oauth-api-service-server:20260710T215258-b5ba554-codex-balance-retry`，容器环境 `GIT_SHA=b5ba554be431423b7c83b57d65f209c8d47883c0`、`GIT_SHA_SHORT=b5ba554`、`IMAGE_TAG=20260710T215258-b5ba554-codex-balance-retry`，镜像内 `codex-cli 0.144.1`。远端与公网 `/healthz` / `/readyz`、`/public/codex/balance` 均通过且为实时 `status=ok`、`stale=false`，返回 `codex` 与 `codex_bengalfox` 分组。生产 Playwright 登录 `/admin-codex-balance` 后接口状态为“正常”，`GPT-5.3-Codex-Spark` 卡片和重置券均可见，无红色失败提示。
- 清理与回滚：发布后执行 `docker image prune -a -f` 与 `docker builder prune -f`，回收 `689.8MB`，根分区可用空间由约 `51G` 回升至约 `53G`；未执行 volume prune。当前 release 目录与 `.env` 备份保留，回滚时需先重新导入上一 release 镜像，再恢复对应 `APP_IMAGE`，不能假定已被 prune 的旧镜像仍在本机。

## 2026-07-10 govulncheck 可达漏洞收敛

- 完成：将 `server/go.mod` 声明工具链从 `go1.26.4` 升级到 `go1.26.5`，将 `github.com/jackc/pgx/v5` 从 `v5.9.0` 升级到 `v5.9.2`，并同步 `server/Dockerfile` / `server/Makefile` 默认 `GO_BUILDER_IMAGE=golang:1.26.5`，避免本地扫描和发布镜像构建工具链分叉。
- 完成：全量 QA 首次暴露本机临时端口压力下 `TestCodexBackendRefreshesExpiredAccessToken` 偶发 `Can't assign requested address`；已将该测试从两个 `httptest` server 收敛为同一 server 下的 `/oauth/token` 和 `/responses` 两条路径，并在 cleanup 关闭 idle connection，降低全量测试端口压力，不改变生产刷新逻辑。
- 验证：已通过 `bash scripts/qa/govulncheck.sh`，可达漏洞从 Go 标准库 `GO-2026-5856` 和 `pgx` `GO-2026-5004` 收敛为 0；已通过 `cd server && go test -count=1 ./...`、`cd server && go test -count=1 ./internal/server -run TestCodexBackendRefreshesExpiredAccessToken` 和 `PATH="/usr/local/bin:$PATH" bash scripts/qa/full.sh`。本轮不改 OAuth、网关压缩、usage 统计、schema、migration、admin UI 或生产配置语义。
- 部署：已提交并推送 `1ec2e44cec382286fe46d697af1e89aa4ee7b031`；按低配发布主路径在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260710T002309-1ec2e44c-govulncheck-clean`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260710T002309-1ec2e44c-govulncheck-clean`。远端只执行 `docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260710T002309-1ec2e44c-govulncheck-clean`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：当前 app-server 运行镜像为 `oauth-api-service-server:20260710T002309-1ec2e44c-govulncheck-clean`，容器环境 `GIT_SHA=1ec2e44cec382286fe46d697af1e89aa4ee7b031`、`GIT_SHA_SHORT=1ec2e44c`、`IMAGE_TAG=20260710T002309-1ec2e44c-govulncheck-clean`；远端本机 `/healthz` / `/readyz` 均通过，`/v1/models` 使用测试 key smoke 返回 6 个模型。启动日志显示 `service.version=1ec2e44cec382286fe46d697af1e89aa4ee7b031`。
- 清理：部署验证后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260709T233619-41b03fa4-natural-facts`，回收 `353.5MB`；未执行 volume prune。根分区从 42% 回到 41%，当前 app-server 仍运行新镜像。
- 下一步：后续若开启 `GOVULNCHECK_STRICT=1`，当前可达漏洞已满足阻断要求；仍可择机处理 govulncheck 报告中“依赖树存在但当前代码不可达”的非阻断项。
- 阻塞/风险：本轮只处理可达漏洞和构建工具链一致性；没有升级 Alpine runtime、Node/Codex CLI runtime、OpenTelemetry 或其他未调用漏洞来源。

## 2026-07-10 133 最新代码部署与 Codex runtime 持久同步

- 完成：先确认本地 `main` 与 `origin/main` 同步后，将最新 HEAD `79c20c83768b3f2419f17a54a9221b4db52768b7` 打包部署到 133；部署后发现运行容器可通过健康脚本临时升级到 `@openai/codex@0.143.0`，但镜像真源仍固定 `0.133.0`，因此将 `server/Dockerfile` 的镜像内 Codex CLI 固定版本同步提升到当前 npm latest `0.143.0`，避免后续容器重建回退。
- 线上压缩验证：在 133 新镜像上通过 `scripts/qa/live-context-compaction.py` 连续跑 3 个 `rounds=3` 隔离 session 和 1 个 `rounds=4` session；每轮登记均返回 `ACK_LIVE_CONTEXT_ROUND_*`，最终轮正文不再提供客户代号，只要求从同 session 压缩摘要回忆。前三个 session 均正确返回第 1 / 第 3 轮事实，`rounds=4` session 正确返回第 1 / 第 2 / 第 4 轮事实。
- DB 证据：4 个线上 session 的 `gateway_context_summaries.compaction_count` 分别为 `4/4/4/5`，`durable_facts` 分别保留各自随机 token 的自然语言事实；`gateway_usage_logs` 按 session 聚合均为全 200 成功，原始约 `1.007MB` 的请求经网关压缩后入库 request_bytes 约 `19.7KB..21.8KB`，未观察到跨 session 串事实或目标漂移。
- 部署验证：远端只执行 `docker load`、宿主机 Atlas status、更新 Compose `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`；Atlas 当前版本 `20260604123931`、pending 0。远端 `/healthz` / `/readyz` 通过，`/v1/models` smoke 返回 6 个模型，启动日志 `service.version` 与镜像内 `GIT_SHA` 对齐。
- 下一步：后续若 npm latest 再变化，优先先判断是否需要同步镜像固定版本；运行时健康脚本的 auto-upgrade 只作为兜底，不替代镜像构建真源。
- 阻塞/风险：线上 live 压缩回归覆盖的是明确登记的自然语言事实和 session 级 durable_facts，不承诺任意无结构长文本细节 100% 永久保留；Codex runtime 健康脚本仍可作为兜底自动升级，但镜像固定版本才是可复现发布真源。

## 2026-07-10 压缩 live 回归脚本化

- 完成：新增 `scripts/qa/live-context-compaction.py`，用于手动线上多轮上下文压缩回归。脚本必须显式提供 `GATEWAY_BASE_URL` 和 `GATEWAY_API_KEY`，默认生成独立 `session_id`，连续登记自然语言事实并在最终轮不提供客户代号值，只验证官方回答能从同 session 压缩摘要 / `durable_facts` 回忆早期事实。
- 完成：更新 `scripts/README.md`，明确该脚本会消耗真实上游额度、可能触发大请求突发保护，因此不纳入 `fast.sh` / `full.sh` / `strict.sh`，只作为发布后或问题复现时的 live 手动回归入口。
- 线上验证：本机直连 `192.168.0.133:8400` 首次 run `live-context-compaction-20260710-000804-9b352e6e` 前两轮返回官方 ACK，第三轮在本机 Python 建连时触发 `Can't assign requested address`，判断为本机网络栈 / 临时端口抖动而非网关压缩失败。随后将同一脚本放到 133 并通过 `127.0.0.1:8400` 跑完整 run `live-context-compaction-20260710-000916-0b5c59b4`：3 轮登记均返回 `ACK_LIVE_CONTEXT_ROUND_*`，最终轮正文不提供客户代号，官方回答正确返回第 1 / 第 3 轮事实。
- DB 证据：`live-context-compaction-20260710-000916-0b5c59b4` 在 `gateway_usage_logs` 有 4 条记录，均 `status_code=200`、`success=true`、`context_compacted=true`，`context_compaction_count=1..4`；原始请求约 `1.107MB`，压缩后约 `19.7KB..21.8KB`。最终 `gateway_context_summaries.summary.durable_facts` 包含第 1/2/3 轮自然语言事实，`current_user_goal` 指向当前“从同一个 session 的压缩摘要 durable_facts 中回忆指定轮次事实”请求，没有漂到旧目标。
- 清理：远端 `/tmp/live-context-compaction.py` 已移动到 `/tmp/openai-oauth-qa-used/live-context-compaction-20260710-000916.py` 作为临时证据文件；本轮不重新部署服务端，线上仍运行已验证镜像 `oauth-api-service-server:20260709T233619-41b03fa4-natural-facts`。
- 下一步：如后续继续出现“上下文没错乱但事实不关联”，优先用该脚本复现并记录 `session_id`，再结合线上 `gateway_usage_logs` / `gateway_context_summaries` 核对 `context_compacted`、`context_compaction_count` 和最终 `durable_facts`。
- 阻塞/风险：脚本只覆盖明确登记的自然语言事实契约，不声称任意无结构长文本都能 100% 保留；真实运行需要有效 gateway key，默认本地质量门禁不执行。

## 2026-07-09 Agent passthrough 上下文压缩边界

- 完成：Codex / OpenCode 客户端按 `client_type=codex/opencode` 默认进入 Agent passthrough；其他 Agent 可通过 `X-Gateway-Agent-Passthrough: true`、`X-Agent-Passthrough: true`、`X-Raw-OpenAI-Compatible: true`，或请求体 / `metadata` 中的 `passthrough`、`raw_openai_compatible`、`disable_context_compression` 等布尔开关显式启用。`X-Gateway-Agent-Passthrough: false` 可覆盖自动判定，临时回到旧网关压缩路径。
- 完成：passthrough 请求继续走 auth、API key、quota、限流、路由、模型映射、reasoning effort 策略和 usage 记录，但跳过网关上下文预压缩、`gateway_context_restore_state.v1` 注入、context summary 持久化、默认工程助手 prompt、visible process prompt 和 resume prompt；普通非 passthrough 请求保留原结构化压缩主路径。
- 完成：usage diagnostic 增加 `agent_passthrough` 和 `agent_passthrough_reason`，并在 DB JSON、RPC 输出和 summary 中保留，便于区分“网关透明转发后上游返回 context_length_exceeded”和“网关预压缩失败”。
- 完成：同步 `README.md` 与 `server/docs/config.md`，明确普通聊天压缩和 Agent 透明转发的边界。
- 验证：已通过 `cd server && go test -count=1 ./internal/server -run 'TestGatewayRequestOptions|TestPrepareGatewayContext|TestCodexBackendRequest|TestGatewayUsageDiagnostic'`、`cd server && go test -count=1 ./...`、`bash scripts/qa/secrets.sh` 和 `git diff --check`。本机 PATH Codex CLI 已从 `0.142.2` 更新到 `0.143.0`；用 `saurick` provider 新建 thread `019f472b-99f4-7f12-aa5f-ae9911e6be55` 输出 `MARKER=AGENT_PASSTHROUGH_CLI_0143_ROUND1`，再显式 `codex exec resume 019f472b-99f4-7f12-aa5f-ae9911e6be55` 输出 `MARKER=AGENT_PASSTHROUGH_CLI_0143_RESUME`，未使用 `resume --last`。
- 阻塞/风险：本轮不改 schema、migration、auth、API key 生命周期、quota 语义、上游策略和 admin UI；passthrough 模式不再替 Agent 客户端“救”超长上下文，真实超限会由上游 / 客户端自身 compact 机制处理并在 usage 中记录错误分类。当前本机 `local` provider 真实请求回归被 dev Postgres 认证失败挡住，`8400` 旧 dev 进程已停止；上述 Codex CLI 回归验证的是 `0.143.0` 显式 resume 流程和线上 provider 可用性，不等同于本轮新服务端代码已部署到线上。

## 2026-07-09 Agent passthrough 线上部署与压缩复核

- 完成：提交并推送 `18d76221124b11b3ca6a0b311cf7427eafa5fa9b` 后，按低配发布主路径本机构建镜像 `oauth-api-service-server:20260709T221549-18d76221-agent-passthrough` 并部署到 133；首轮线上压缩回归发现 `current_user_goal` 和 pinned directives 正确，但 `latest_user_instruction` 可能被尾部“不能覆盖最新目标”类历史噪声抢走。
- 完成：追加修复 `ed8102f0fa13e5dd5ae99662dd004e88bd226494`，让 `latest_user_instruction` 和 `current_user_goal` 优先取行首或行首附近的“当前请求 / 最新用户 / 当前目标”强信号；没有单独 current goal 时用当前段最新指令兜底。新增 `TestCompactGatewayContextLatestInstructionPrefersCurrentRequestOverRestrictionNoise` 覆盖这次线上发现的噪声形态。
- 验证：已通过 `cd server && go test -count=1 ./internal/server -run 'TestCompactGatewayContext|TestPrepareGatewayContext|TestGatewayContext'`、无代理环境 `cd server && go test -count=1 ./...`、`bash scripts/qa/secrets.sh`、`git diff --check` 和 `PATH="/usr/local/bin:$PATH" env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u NO_PROXY -u http_proxy -u https_proxy -u all_proxy -u no_proxy bash scripts/qa/full.sh`。`qa:full` 仍提示既有 Go 1.26.4 与 `pgx/v5@5.9.0` govulncheck 风险，按当前脚本仅提示不阻断。
- 部署：基于 `ed8102f0` 在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260709T222744-ed8102f0-context-latest-instruction`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260709T222744-ed8102f0-context-latest-instruction`；远端只执行 `docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260709T222744-ed8102f0-context-latest-instruction`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：当前 `app-server` 运行镜像为 `oauth-api-service-server:20260709T222744-ed8102f0-context-latest-instruction`，日志 `service.version=ed8102f0fa13e5dd5ae99662dd004e88bd226494`；远端本机和公网 `/healthz` / `/readyz` 均通过。线上 run `gw-compress-fix-20260709-0989d316` 中 1 条 Codex-like passthrough 请求返回 `MARKER=AGENT_PASSTHROUGH_FIX_R0`，usage 记录 `agent_passthrough=true`、`context_compaction_reason=agent_passthrough`；3 条非 Agent 大上下文请求均返回官方 marker，原始 `1050777` bytes 压缩到 `23551/25042` bytes，`context_compaction_count=1/2/3`，最终 summary 的 `current_user_goal`、`latest_user_instruction`、`latest_uncompressed_user_instruction` 均指向 R3 当前请求而非历史噪声。
- CLI 验证：本机 PATH Codex CLI 确认为 `0.143.0` 且 `npm view @openai/codex version` 也是 `0.143.0`；临时 git 目录中通过 `saurick-oauth` provider 新建 thread `019f474b-83e1-7d43-84a0-524fcabf319e` 输出 `MARKER=CLI_0143_AFTER_DEPLOY_ROUND1_223238`，再显式 `codex exec resume 019f474b-83e1-7d43-84a0-524fcabf319e` 输出 `MARKER=CLI_0143_AFTER_DEPLOY_RESUME_223306`，未使用 `resume --last`。运行期间本机 MCP 到 `developers.openai.com` 出现 `Can't assign requested address` 警告，但主 provider 请求返回正确 marker。
- 清理：部署和回归通过后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧镜像 `20260709T221549-18d76221-agent-passthrough` 和 `20260709T161200-b1c83f1b-pagination-50`，回收 `706.9MB`；未执行 volume prune。根分区从 42% 回到 41%，当前 app-server 仍运行 `ed8102f0` 新镜像。
- 阻塞/风险：本轮不改 schema、migration、auth、API key 生命周期、quota 语义、上游策略和 admin UI；Codex CLI 验证中本机 TCP `TIME_WAIT` 一度达到约 7700，造成局部 SSH/curl/MCP 连接抖动，已通过放慢连接和禁用测试脚本代理规避。passthrough 模式仍按设计把真实超长上下文交给 Agent 客户端自己的 compact / summarization 处理。

## 2026-07-09 压缩事实链 durable_facts 回归

- 诊断：在 `ed8102f0` 线上版本补跑事实链回归时，9 轮登记事实后第 10 轮不携带事实值、只要求从同 session 压缩摘要回忆，官方回答只能恢复 R9，R1/R3/R6 丢失。根因是旧状态包只把上一轮完整 JSON 放入 `historical_context_only` 并截断到 1000 字，长期事实没有单独结构化字段，早期事实会被后续摘要覆盖。
- 完成：新增 `durable_facts` 字段，压缩时从上一轮状态和当前 user-eligible 片段合并 `FACT_x=value` / “已登记事实”类稳定事实，最多保留 64 条，避免靠截断历史摘要保留长期关联。旧 summary 没有该字段仍可正常解析，不涉及 DB schema 或 migration。
- 验证：已通过 `cd server && go test -count=1 ./internal/server -run 'TestCompactGatewayContextCarriesDurableFactsAcrossRepeatedCompactions|TestCompactGatewayContextLatestInstructionPrefersCurrentRequestOverRestrictionNoise|TestCompactGatewayContext'`、无代理环境 `cd server && go test -count=1 ./...`、`bash scripts/qa/secrets.sh`、`git diff --check` 和 `PATH="/usr/local/bin:$PATH" env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u NO_PROXY -u http_proxy -u https_proxy -u all_proxy -u no_proxy bash scripts/qa/full.sh`。`qa:full` 仍提示既有 Go 1.26.4 与 `pgx/v5@5.9.0` govulncheck 风险，按当前脚本仅提示不阻断。
- 部署：基于 `b427e0d92fb0cb53e6d0b27e97a21749ff8848bc` 在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260709T231533-b427e0d9-durable-facts`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260709T231533-b427e0d9-durable-facts`；远端只执行 `docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260709T231533-b427e0d9-durable-facts`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：当前 `app-server` 运行镜像为 `oauth-api-service-server:20260709T231533-b427e0d9-durable-facts`，日志 `service.version=b427e0d92fb0cb53e6d0b27e97a21749ff8848bc`；远端本机 `/healthz` / `/readyz` 通过。线上 run `gw-fact-durable-20260709-16533985` 连续 10 轮均 `status=200`、`success=true`、`context_compacted=true`，`context_compaction_count=1..10`；前 9 轮登记 `FACT_R1..FACT_R9`，第 10 轮正文不提供事实值，只要求从同 session 压缩摘要回忆，官方回答正确返回 R1/R3/R6/R9。DB 最终 summary 的 `durable_facts` 保留 R1-R9 全部事实。
- 清理：部署和回归通过后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧镜像 `20260709T222744-ed8102f0-context-latest-instruction`，回收 `353.4MB`；未执行 volume prune。根分区维持约 41%，当前 app-server 仍运行 `b427e0d9` 新镜像。
- 阻塞/风险：本轮保证的是可识别稳定事实锚点（如 `FACT_x=value` / “已登记事实”）跨压缩保留；普通长篇自然语言事实如果没有明确锚点，仍可能只按当前摘要启发式保留，不应承诺任意细节 100% 永久保留。

## 2026-07-09 自然语言事实链 durable_facts 回归

- 诊断：在 `b427e0d9` 线上版本补跑自然语言事实链时，6 轮登记“客户代号是 NATURAL_x”后第 7 轮只靠同 session 压缩摘要回忆，官方回答只恢复第 6 轮，第 1/3 轮丢失。DB 显示 7/7 轮已压缩，但 `durable_facts` 为空，原因是上一轮只提取 `FACT_x=value` 形式，没有收录“已登记事实：...”自然语言事实行。
- 完成：将 durable facts 提取扩展到明确事实前缀：`已登记事实：`、`已登记事实:`、`durable fact:`、`registered fact:`，并支持“当前最新用户请求：登记自然语言事实“...””中的引号内容；仍不把任意包含“事实”的噪声行纳入，避免过度抓取。
- 验证：已通过 `cd server && go test -count=1 ./internal/server -run 'TestCompactGatewayContextCarriesNaturalLanguageDurableFacts|TestCompactGatewayContextCarriesDurableFactsAcrossRepeatedCompactions|TestCompactGatewayContext'`、无代理环境 `cd server && go test -count=1 ./...`、`bash scripts/qa/secrets.sh`、`git diff --check` 和 `PATH="/usr/local/bin:$PATH" env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u NO_PROXY -u http_proxy -u https_proxy -u all_proxy -u no_proxy bash scripts/qa/full.sh`。`qa:full` 仍提示既有 Go 1.26.4 与 `pgx/v5@5.9.0` govulncheck 风险，按当前脚本仅提示不阻断。
- 部署：基于 `41b03fa49c9181c51a95600f9f17be7a78f3843c` 在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260709T233619-41b03fa4-natural-facts`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260709T233619-41b03fa4-natural-facts`；远端只执行 `docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260709T233619-41b03fa4-natural-facts`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：当前 `app-server` 运行镜像为 `oauth-api-service-server:20260709T233619-41b03fa4-natural-facts`，日志 `service.version=41b03fa49c9181c51a95600f9f17be7a78f3843c`；远端本机 `/healthz` / `/readyz` 通过。线上 run `gw-natural-fixed-20260709-5d8963ff` 共 7 轮均 `status=200`、`success=true`、`context_compacted=true`，`context_compaction_count=1..7`；前 6 轮登记自然语言事实，第 7 轮正文不提供事实值，只要求从同 session 压缩摘要回忆，官方回答正确返回第 1/3/6 轮事实。DB 最终 summary 的 `durable_facts` 保留第 1-6 轮全部自然语言事实。
- 清理：部署和回归通过后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧镜像 `20260709T231533-b427e0d9-durable-facts`，回收 `353.4MB`；未执行 volume prune。根分区从 42% 回到 41%，当前 app-server 仍运行 `41b03fa4` 新镜像。
- 阻塞/风险：当前保证的是明确登记的事实行跨压缩保留；完全无结构、无“已登记事实”前缀的普通段落仍只按摘要启发式处理，不承诺任意自然语言细节永久保留。

## 2026-07-09 后台分页页容量统一

- 完成：后台共享表格分页默认页容量从 8 条改为 50 条，页容量选项统一为 `50/100/200/500/1000`；同步放宽服务端管理端 list limit 上限到 1000，避免前端 500 / 1000 选项与真实返回数量不一致。
- 完成：分页控件宽度和 `style:l1` 回归同步调整到最长 `1000 条/页` 不裁切；mock 数据扩到超过 50 条，继续覆盖下一页请求、表头全选当前页和每日模型详情分页。修正 `style:l1` 启动非默认端口时 Vite HMR 仍固定指向 5176 的误报，不改变生产构建端口。
- 验证：本地已通过 `go test -count=1 ./...`、`/usr/local/bin/pnpm lint`、`/usr/local/bin/pnpm css`、`/usr/local/bin/pnpm test`、`/usr/local/bin/pnpm build`、无代理环境全量 `pnpm style:l1`、`bash scripts/qa/secrets.sh` 和 `git diff --check`。提交推送时 pre-push `qa:full` 也通过；`govulncheck` 仍提示 Go 1.26.4 标准库和 `pgx/v5@v5.9.0` 的已知漏洞，当前脚本按既有规则提示但不阻断。
- 部署：已基于提交 `b1c83f1b3cc488bb5c4ae3b07c76d550960b96e7` 在本地构建 linux/amd64 镜像 `oauth-api-service-server:20260709T161200-b1c83f1b-pagination-50`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260709T161200-b1c83f1b-pagination-50`；远端只执行 checksum、`docker load`、宿主机 `/usr/local/bin/atlas migrate status`、备份 `.env` 为 `.env.bak.20260709T161200-pagination-50`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在 133 构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：当前 `app-server` 运行镜像为 `oauth-api-service-server:20260709T161200-b1c83f1b-pagination-50`，容器环境包含 `GIT_SHA=b1c83f1b3cc488bb5c4ae3b07c76d550960b96e7`、`GIT_SHA_SHORT=b1c83f1b` 和 `IMAGE_TAG=20260709T161200-b1c83f1b-pagination-50`；远端本机和公网 `/healthz` / `/readyz` 均通过，`/public/codex/balance` 首次因 ChatGPT `wham/usage` 上游读取失败返回 502，重试后返回 200。生产 Playwright 登录 `/admin-keys` 与 `/admin-usage` 后确认分页默认 `50 条/页`，下拉只包含 `50/100/200/500/1000 条/页`，`1000 条/页` 不裁切，浏览器控制台无错误。
- 清理：部署验证后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧镜像 `oauth-api-service-server:20260703T175800-14c7fae1-local`，回收 353.4MB；未执行 volume prune。根分区从 41% 降到 40%，当前 app-server 仍运行新镜像，近 1 分钟日志未见 WARN / ERROR / PANIC / FATAL。
- 阻塞/风险：本轮不改 schema、migration、auth、API key 生命周期、usage 真源、上游策略或 quota 语义；首页最近调用样本仍是固定摘要数量，不属于可切换分页控件。

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
