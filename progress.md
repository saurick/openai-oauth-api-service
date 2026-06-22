## 归档索引

- 2026-06-04：旧 `progress.md` 已按超过 600 行阈值归档到 `docs/archive/progress-2026-06-04-before-govulncheck.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。

## 2026-06-20 管理端用量指标顺序

- 完成：`/admin-dashboard` 用量趋势指标按钮改为 `Token / 请求 / 服务错误 / 延迟 / 费用`，默认展示 Token；`/admin-usage` 摘要指标卡同步把总 Token 放在第一位，后续按请求、服务错误率、费用、上游和客户端分布排列。
- 完成：恢复 `/admin-dashboard` 最近调用表的“客户端 IP”列，保持与调用明细和既有回归口径一致。
- 验证：已执行 `pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminDashboard/index.jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`STYLE_L1_SCENARIOS=admin-dashboard-desktop,admin-dashboard-mobile,admin-usage-desktop,admin-usage-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`、`pnpm --dir web test`、`pnpm --dir web css` 和 `pnpm --dir web build`，均通过；`style:l1` 覆盖 dashboard / usage 桌面与移动端目标页、趋势按钮顺序与默认 Token、usage 摘要卡顺序、tooltip、暗色/浅色相关目标区域和表格盒模型。
- 部署：已提交并推送 `b9371bf fix: 调整管理端用量指标顺序`；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260620T151023-b9371bf0-usage-metric-order`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260620T151023-b9371bf0-usage-metric-order`。远端只执行 `docker load`、宿主机 Atlas status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604123931`、pending 0；远端本机和公网 `/healthz` / `/readyz` 通过，容器环境 `GIT_SHA_SHORT=b9371bf0`、`IMAGE_TAG=20260620T151023-b9371bf0-usage-metric-order`，公网静态 bundle 已包含新指标顺序和“客户端 IP”列，管理员 `admin/adminadmin` 登录、`api.summary` 与 `api.usage_list` smoke 通过。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包和 migration tar 包，执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260619T180249-eb054c3f-local`，回收 353.3MB，未清理 volume。清理后根分区使用率 30%，当前 app-server 运行镜像为 `oauth-api-service-server:20260620T151023-b9371bf0-usage-metric-order`。
- 阻塞/风险：本轮只改前端展示顺序、最近调用客户端 IP 展示和回归断言，不改后端 usage 真源、统计口径、schema 或部署配置。

## 2026-06-19 Codex 余额查询临时失败兜底

- 诊断：线上 `/admin-codex-balance` 一度显示“Codex 余额查询失败”，133 app-server 与 PostgreSQL 均健康，`/healthz` / `/readyz` 正常；后端日志显示 `account/rateLimits/read` 读取 `https://chatgpt.com/backend-api/wham/usage` 时出现 `error sending request`，随后同一路径又可正常返回 200，判断为 Codex app-server 到 ChatGPT usage 接口的临时上游 / 代理链路失败，不是 Codex 登录态整体失效。
- 完成：`/public/codex/balance` 保留上次成功结果；实时查询失败且已有成功缓存时返回 HTTP 200、原余额数据和 `stale=true` / `stale_reason=codex_balance_query_failed` / `last_error_at`，避免后台页直接清空余额并显示红色失败。首次启动且没有成功缓存时仍返回失败，不伪造余额。
- 完成：`/admin-codex-balance` 识别 stale 结果，接口状态显示“缓存结果”，并展示“实时查询暂时失败，当前显示上次成功读取的 Codex 余额。”提示；暗色主题沿既有 admin warning 变量覆盖。
- 文档：同步更新 `server/docs/api.md` 的公开余额查询说明，明确 stale 语义和首次无缓存仍失败的边界。
- 验证：已执行 `cd server && go test -count=1 ./internal/server -run 'TestCodexBalanceRoute'`、`cd server && go test -count=1 ./...`、`pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminCodexBalance/index.jsx scripts/styleL1.mjs`、`pnpm --dir web test`、`pnpm --dir web build`、`STYLE_L1_SCENARIOS=admin-codex-balance-desktop,admin-codex-balance-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`，均通过；`style:l1` 覆盖 Codex 余额桌面和移动端目标区域。线上 Playwright 登录 `https://oauth-api.saurick.me/admin-codex-balance` 后确认页面显示接口状态“正常”、Credits remaining 为 `0`，无红色失败提示、无 stale 提示、无横向溢出。
- 部署：本地构建 linux/amd64 镜像 `oauth-api-service-server:20260619T180249-eb054c3f-local`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260619T180249-eb054c3f-balance-stale`。远端只执行 `docker load`、宿主机 Atlas status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。首次 Atlas status 被 migration 目录中的 macOS `._*` 资源叉文件阻断，已清理资源叉文件后重跑通过；Atlas 当前版本 `20260604123931`、pending 0。远端本机和公网 `/healthz` / `/readyz` / `/public/codex/balance` 均通过，当前 `app-server` 运行镜像为 `oauth-api-service-server:20260619T180249-eb054c3f-local`，容器环境 `GIT_SHA_SHORT=eb054c3f-local`、`IMAGE_TAG=20260619T180249-eb054c3f-local`。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包，执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260614T175158-1a18472-context-anchor`，回收 353.3MB，未清理 volume。清理后根分区使用率 30%，当前 app-server 仍运行新镜像。
- 阻塞/风险：本轮兜底依赖 app-server 进程内缓存；容器刚重启且第一次查询就遇到上游失败时仍会显示失败。该修复不改变 Codex app-server、mihomo 节点或 ChatGPT usage 接口本身的稳定性。

## 2026-06-16 Codex 上游代理优先级切换

- 完成：调整 `scripts/ops/codex-upstream-proxy-failover.py` 的节点选择策略，从“当前节点的下一个”轮转改为每次触发时按 `CODEX_FAILOVER_NODES` 优先级选择第一个可用且不同于当前节点的目标；当前优先级仍为 `日本JP-HY2 -> 日本-优化3 -> 日本-优化2 -> 日本-优化`，因此低优先级线路失败会优先切回 HY2，HY2 失败时才切到下一优先级。
- 验证：已执行 `python3 -m py_compile scripts/ops/codex-upstream-proxy-failover.py`，并用 importlib 直接断言 HY2、低优先级、未知当前节点和无可切节点场景的选择结果。
- 部署：已同步到 `192.168.0.133` 的 `/usr/local/sbin/codex-upstream-proxy-failover.py`，执行远端 `python3 -m py_compile` 后重启宿主机级 `codex-upstream-proxy-failover.service`；服务为 `active`，`--check` 自检通过，当前 `节点选择=日本JP-HY2` 且 `ChatGPT=节点选择`。未重启 app-server、PostgreSQL，也未修改 Compose。
- 阻塞/风险：本轮只改触发后的节点选择顺序，不改变 180 秒冷却窗口，也不扩大错误日志匹配范围；`TLS handshake timeout` 和无 URL 的 `unexpected EOF` 是否触发切换仍需单独收口。

## 2026-06-15 21:50 CST

- 完成：新增 `scripts/deploy/production-preflight.sh` 和 `server/Makefile` 的 `production_preflight` 入口，作为 OAuth API Service 生产发布前门禁；检查运行时 `.env`、占位 secret、镜像 tag、Codex upstream fallback、Compose 禁止 `build:`、migration 文档边界和可选运行态 `/healthz` / `/readyz`。
- 完成：同步 `scripts/README.md` 与 `server/deploy/compose/prod/README.md` 的 preflight 入口说明；保留当前个人部署 `admin/adminadmin` 口径，不在脚本里擅自强制改密。
- 验证：`bash scripts/deploy/production-preflight.sh --example` 通过；`.env.example` 作为生产 env 被 placeholder 门禁阻断；临时非占位 env 通过静态 preflight；`make -n production_preflight`、`bash scripts/qa/secrets.sh`、`git diff --check` 通过。
- 下一步：真实发布前先替换 `server/deploy/compose/prod/.env` 并执行 `cd server && make production_preflight`；部署后追加 `--runtime`。
- 阻塞/风险：本项目当前没有独立 `migrate_online.sh`，preflight 检查的是 Compose 与部署文档中的宿主机 Atlas / flock 迁移边界；本轮未连接真实生产 `.env` 或运行中 Compose。

## 2026-06-15 Codex 上游代理自动切换

- 完成：为 `192.168.0.133` 增加 mihomo 自动切换守护脚本，实时跟随 `openai-oauth-api-service-server` 日志；当 Codex backend 到 `https://chatgpt.com/backend-api/codex/responses` 出现 `EOF`、`INTERNAL_ERROR`、`stream disconnected`、`error sending request` 或 `connection reset` 时，自动把 mihomo `节点选择` 按 `日本JP-HY2 -> 日本-优化3 -> 日本-优化2 -> 日本-优化` 切到下一个，并保持 `ChatGPT` 组继承 `节点选择`。默认冷却时间 180 秒，避免同一批错误瞬间连跳。
- 完成：新增 `scripts/ops/install-codex-upstream-proxy-failover.sh` 作为迁移服务器时的一键安装入口；守护脚本新增 `--check` 自检模式，安装时会检查 mihomo controller、`节点选择`、继承组和候选节点，而不是把宿主机运维逻辑塞进业务 Docker 镜像。
- 部署：远端仅新增宿主机级 systemd 服务 `codex-upstream-proxy-failover.service` 和脚本 `/usr/local/sbin/codex-upstream-proxy-failover.py`，未重启 app-server，未修改 Compose、镜像、数据库、管理员密码或 mihomo 订阅配置。
- 验证：手动切换后确认 `ChatGPT=节点选择`，当前 `节点选择=日本-优化3`；`日本-优化3`、`日本-优化2`、`日本-优化` 均可完成 mihomo 延迟测试；经代理访问 `https://chatgpt.com/backend-api/wham/usage` 返回 401，app-server `healthz/readyz` 和 `/public/codex/balance` 正常。已用临时安装包在 133 执行 `bash scripts/ops/install-codex-upstream-proxy-failover.sh`，安装脚本完成语法检查、systemd reload、服务重启和 `--check` 自检，服务保持 `active`。
- 阻塞/风险：该守护只按日志中的 Codex backend 断流信号切换节点，不主动重启 mihomo 或业务容器；若所有候选节点都被上游风控或断流，服务会在冷却后继续按列表循环，仍需人工排查订阅、出口或 ChatGPT 上游状态。

## 2026-06-14 Codex 上下文压缩进度锚点保留

- 完成：补充项目级 `AGENTS.md`，明确 Codex 多轮上下文压缩、恢复或 Windows 端回归测试时，不能把 `codex exec resume --last` 作为可靠验收依据；必须从本轮 JSONL 读取 `thread_id` 并显式 `codex exec resume <thread_id>`，避免误捡 Windows 用户全局旧会话。
- 诊断：当前网关已能在大上下文前置压缩，但单个超长 Responses `input` / 消息 `content` / `function_call_output` 的裁剪主路径只保留自动摘要和固定短尾部；如果“最新会话进度 / 验证 / 下一步 / 部署”等交接文本后面还有大量工具 schema、环境文本或其他尾部噪声，压缩后可能丢失最新执行状态，恢复时更容易回到几小时前的旧讨论。Windows Codex 实测还暴露了数组压缩路径的第二个问题：被裁掉的前半段里同时包含 Codex 系统规则和用户当前请求时，正向扫描会优先抓到系统规则里的 `next step / blocked` 词，用户当前请求仍可能被弱化。
- 完成：上下文压缩新增最近进度锚点提取，自动摘要中单列“最近进度与交接锚点”；单个超长文本裁剪时改为保留“压缩恢复执行锚点”、原文开头、最近进度锚点和原文末尾，不再只依赖固定 2000 字尾部；多消息数组压缩时插入显式恢复锚点，并从被裁掉片段尾部反向提取，优先保留最近用户当前请求。本轮不新增 schema、不改压缩阈值、不改变请求体/响应体默认不落库口径。
- 文档：同步更新 `server/docs/config.md` 和 `server/docs/api.md`，说明压缩会保留最新进度、验证、部署、下一步、阻塞或风险附近的交接文本。
- 验证：已执行 `cd server && go test -count=1 ./internal/server -run 'TestCompactGatewayContext|TestPrepareGatewayContext'` 与 `cd server && go test -count=1 ./...`，均通过；新增回归覆盖“最新进度在尾部噪声之前，旧 2000 字末尾裁剪会丢失交接文本”、Responses 数组被压缩时保留当前请求、Chat 多消息数组保留当前请求、系统规则也含进度词时仍优先最近用户请求。
- 验证：本地强制低阈值直连 `/v1/responses` 返回 `MARKER=LOCAL-ANCHOR-V2`；`ssh sauri@192.168.0.45` 到 Windows Codex 0.133.0，临时 `CODEX_HOME` 指向本机网关，强制压缩阈值下新会话 `D-SMALL-T1`、`E-SMALL-T1` 通过，显式 `thread_id=019ec4f4-bbd7-7270-9f24-09e025444083` 恢复同一会话 `D-EXPLICIT-T2`、`D-EXPLICIT-T3` 通过。测试中发现 `codex exec resume --last` 会捡到 Windows 用户全局旧会话，必须显式 resume thread id；大流量测试遇到上游 429，已改用小流量多轮验证。
- 部署：Windows Codex 多会话与同会话多次压缩验证通过后，先本地构建临时 linux/amd64 镜像 `oauth-api-service-server:20260614T152137-ec3a01fe-context-anchor` 并部署到 133 完成验证；随后提交并推送 `1a18472 fix: 保留上下文压缩恢复锚点`，再次本地构建 clean SHA 镜像 `oauth-api-service-server:20260614T175158-1a18472-context-anchor`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260614T175158-1a18472-context-anchor`。远端只执行 `docker load`、宿主机 `/usr/local/bin/atlas migrate status`、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604123931`、pending 0；本机与公网 `/healthz` / `/readyz` 通过，管理员 `admin/adminadmin` 登录、`summary` 与 `model_list` smoke 通过。首次临时重建时 shell 旧 `APP_IMAGE` 覆盖 `.env` 导致仍跑旧镜像，已立即用显式 `APP_IMAGE=...` 重建修正；最终当前容器环境 `GIT_SHA_SHORT=1a184720`、`IMAGE_TAG=20260614T175158-1a18472-context-anchor`。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包，执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260614T152137-ec3a01fe-context-anchor`，回收 353.3MB，未清理 volume。清理后根分区使用率 30%，当前 app-server 运行镜像为 `oauth-api-service-server:20260614T175158-1a18472-context-anchor`。
- 阻塞/风险：本轮修的是网关侧压缩保留策略；Windows Codex 客户端的 `resume --last` 选择全局旧会话属于客户端/脚本用法边界，不能作为网关压缩失败判断。上游 429 会影响重压测稳定性，后续大批量压测需降频或换隔离测试 key。

## 2026-06-10 Codex 大上下文循环止血

- 诊断：线上 `gateway_usage_logs` 显示 `ogw_junnan_G` 在 2026-06-10 22:00-23:05（Asia/Shanghai）之间对 `gpt-5.5` 发起 75 次 `/v1/responses`，累计约 713.6 万 token；多数请求为上游 `response.completed` 后 200 成功，少量为 `client_canceled`，未见服务端 5xx、超时或资源瓶颈。现象更接近客户端异常循环重放大上下文，但服务端缺少同 key 大请求并发保护，且 backend SSE 写下游失败时没有立即把写错误作为取消信号。
- 诊断：首次发布并发保护后继续观察到顺序型大请求循环，`junnan` 10 分钟 32 次约 80.2 万 token，`xin2` 10 分钟 23 次约 254.4 万 token；仅靠 in-flight 保护无法挡住一前一后的客户端循环。
- 完成：`/v1/chat/completions` 与 `/v1/responses` 新增大请求 in-flight 与突发频率保护，默认请求体达到 `GATEWAY_LARGE_REQUEST_MIN_BYTES=65536` 后，同一 API key 同时最多允许 `GATEWAY_LARGE_REQUEST_MAX_INFLIGHT_PER_KEY=1` 个上游请求，并且每 `GATEWAY_LARGE_REQUEST_BURST_WINDOW_SECONDS=60` 秒最多允许 `GATEWAY_LARGE_REQUEST_BURST_MAX_PER_KEY=4` 个大请求；后续大请求快速返回 HTTP `429` / `gateway_large_request_inflight` 或 `gateway_large_request_burst` 并记录 usage 诊断，避免客户端异常循环继续烧 token。
- 完成：Codex backend `/v1/responses` streaming 主路径改为检查下游 SSE 写错误；下游断开或写失败时按 `client_canceled` 收口并取消上游请求，避免继续消费上游流。
- 文档：同步更新 `server/deploy/compose/prod/compose.yml`、`server/deploy/compose/prod/.env.example`、`server/deploy/compose/prod/README.md`、`server/docs/config.md`、`server/docs/api.md`；顺手修正 `.env.example` 中“默认自动 fallback”的旧注释。
- 验证：已执行 `cd server && go test -count=1 ./internal/server`、`cd server && go test -count=1 ./...`，均通过；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260610T232946-46f5fec-large-request-burst`，上传到 133 后执行 `docker load`、Atlas migration status、`docker compose up -d --no-deps --force-recreate app-server`，远端 `healthz/readyz`、公开域名 `healthz/readyz`、管理员 RPC summary、临时 key `/v1/models` smoke 均通过，临时 key 已删除。
- 部署：133 当前运行 `oauth-api-service-server:20260610T232946-46f5fec-large-request-burst`；容器环境已确认 `GATEWAY_LARGE_REQUEST_MIN_BYTES=65536`、`GATEWAY_LARGE_REQUEST_MAX_INFLIGHT_PER_KEY=1`、`GATEWAY_LARGE_REQUEST_BURST_MAX_PER_KEY=4`、`GATEWAY_LARGE_REQUEST_BURST_WINDOW_SECONDS=60`。部署后执行 `docker image prune -a -f` 和 `docker builder prune -f`，旧镜像清理释放 353.3MB，根分区回到 23G used / 71G available。
- 阻塞/风险：该保护是服务端止血，不修复 Windows/Codex 客户端自身为何反复编辑同一文件；新版本启动后暂未再观察到真实客户端触发 `gateway_large_request_burst`，若用户继续复现，应同时排查客户端版本、工作区状态和会话恢复逻辑。

## 2026-06-08 API 凭据最近使用时间独立列

- 完成：按线上反馈把 `/admin-keys` 中“最近使用时间”从备注下方的二级文本提升为独立列，列头为“最近使用时间”；无记录时仍显示 `-`，不改后端 `last_used_at` 真源和写入口径。
- 完成：同步调整 API 凭据表最小宽度、列宽、空态 `colSpan`、`style:l1` 表格列索引和状态列断言，避免新增列后完整凭据、状态、复制按钮等相邻列错位。
- 文档：同步更新 `web/README.md`，明确 API 凭据列表用独立列展示最近使用时间。
- 验证：已执行 `node --check web/scripts/styleL1.mjs`、`pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`、`STYLE_L1_BASE_URL=http://127.0.0.1:4393 STYLE_L1_SCENARIOS=admin-keys-desktop,admin-keys-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`，均通过；`style:l1` 已确认 `/admin-keys` 桌面和移动端存在独立“最近使用时间”列，且列内同时覆盖有值和 `-`。
- 部署：待执行。
- 阻塞/风险：本轮只改前端表格展示、文档和回归断言，不改后端 `last_used_at` 更新口径、schema、历史数据或 usage 真源。

## 2026-06-08 API 凭据最近使用时间回归

- 完成：确认 API 凭据最近使用时间沿既有 `gateway_api_keys.last_used_at` 主路径维护，网关统一 usage 写入后调用 `TouchAPIKeyUsed` 更新，`key_list` 已返回 `last_used_at`，`/admin-keys` 在备注下方展示“最近使用”；本轮不新增 schema、不从前端伪造时间，也不改 usage 记录口径。
- 完成：`style:l1` 的 `/admin-keys` 桌面 / 移动端回归新增最近使用时间断言，覆盖有使用记录和无使用记录显示 `-` 两种状态。
- 文档：同步更新 `web/README.md`，明确 API 凭据列表展示完整凭据和最近使用时间。
- 验证：已执行 `node --check web/scripts/styleL1.mjs`、`pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`、`STYLE_L1_BASE_URL=http://127.0.0.1:4392 STYLE_L1_SCENARIOS=admin-keys-desktop,admin-keys-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`，均通过；`style:l1` 使用本地生产预览验证 `/admin-keys` 桌面和移动端列表、盒模型、完整凭据复制按钮、最近使用时间有值 / 无值展示。
- 阻塞/风险：直接由 `style:l1` 拉起 Vite dev server 时，本机 Playwright 对 HMR WebSocket 报 `net::ERR_ADDRESS_INVALID` 并被脚本当作 console error 拦截；本轮改用已构建产物的 `vite preview` 完成无 HMR 回归。未部署到远端，生产环境需发布后才能看到本轮新增的回归保护和文档更新；现有功能代码本身已在当前 `main`。

## 2026-06-08 用量日志客户端 IP 显示强化

- 完成：`/admin-usage` 调用明细表把 `client_ip` 从“请求”单元格内的附属信息提升为独立“客户端 IP”列；顶部最近请求紧凑表也展示该列，避免只能在完整表格或会话详情里找到 IP。
- 完成：用量日志筛选区新增“客户端 IP”输入框，按完整 IP 传递 `client_ip` 到后端 `usage_list`，便于直接定位某个来源。
- 文档：同步更新 `web/README.md`，明确顶部最近请求、调用明细和会话请求级明细均展示客户端 IP。
- 验证：已执行 `pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`、`STYLE_L1_SCENARIOS=admin-usage-desktop,admin-usage-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1` 和 `git diff --check`，均通过；`style:l1` 已新增断言，确认“客户端 IP”独立列表头、mock IP 值和 IP 筛选输入存在。另用 in-app Browser 打开本地 `http://127.0.0.1:5189/admin-usage`，未登录状态按预期回到 `/admin-login`，页面非空、标题为“API 管理后台”且横向溢出为 0；未启动后端，因此登录页的 `/auth/oauth/config` 代理请求出现预期连接失败。
- 部署：已提交并推送 `8312987 fix: 显示用量日志客户端 IP`；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260608T135652-83129878`，上传到 `/data/openai-oauth-api-service/releases/20260608T135652-83129878-usage-ip-ui`。远端只执行 `docker load`、宿主机 Atlas status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604123931`、pending 0。
- 线上验证：远端本机和公网 `/healthz` / `/readyz` 均通过；当前 `app-server` 运行镜像为 `oauth-api-service-server:20260608T135652-83129878`，容器环境 `GIT_SHA_SHORT=83129878`、`IMAGE_TAG=20260608T135652-83129878`。公网 `/admin-usage` 返回 200 并加载新前端资源 `index.DoEg5sfX.js`，bundle 已包含“客户端 IP”和“输入完整 IP”；管理员 RPC `admin_login`、`summary` 和 `usage_list` smoke 通过，近 5 天 usage 返回 `client_ip` 示例 `120.234.136.227`。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包和 migration tar 包，执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被任何容器使用的旧镜像并回收 353.3MB，未清理 volume。清理后根分区使用率 21%，当前 app-server 仍运行新镜像。
- 阻塞/风险：本轮只改管理端展示和筛选，不改后端 IP 记录口径、schema、导出、部署配置或历史 usage 数据。

## 2026-06-08 全站启用全部 API key

- 完成：新增管理员 RPC `api.key_enable_all`，沿既有 `server -> service -> biz -> data` 主路径批量把当前禁用的 `gateway_api_keys.disabled` 改为 `false`；不改 schema、不删除 key、不改历史 usage。
- 完成：`/admin-keys` 增加「启用全部 key」按钮，复用后台内确认弹窗，明确操作是全站范围且不限于当前页或当前筛选；确认后清空当前选择并刷新列表。
- 文档：同步更新 `server/docs/api.md` 和 `web/README.md`，把全站禁用 / 启用都记录为只切换禁用状态的管理操作。
- 验证：已执行 `go test -count=1 ./internal/biz ./internal/data`、`go test -count=1 ./...`、`pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`、`STYLE_L1_SCENARIOS=admin-keys-desktop,admin-keys-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1` 和 `git diff --check`，均通过；`style:l1` 已覆盖新增启用全部按钮、确认弹窗内容、非浏览器原生 confirm、盒模型和 `key_enable_all` RPC 调用。另用 in-app Browser 打开本地 `http://127.0.0.1:5188/admin-keys`，未登录状态按预期回到 `/admin-login`，页面非空、标题正确且横向溢出为 0；未启动后端，因此登录页的 `/auth/oauth/config` 代理请求出现预期连接失败。
- 部署：已按低配服务器路径部署到 `192.168.0.133`。本地构建 linux/amd64 镜像 `oauth-api-service-server:20260608T051344-34295339-enable-all-keys`，上传到 `/data/openai-oauth-api-service/releases/20260608T051344-34295339-enable-all-keys`；远端只执行 `docker load`、宿主机 Atlas status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604123931`、pending 0；远端本机和公网 `/healthz` / `/readyz` 通过，当前容器镜像为新 tag，容器环境 `GIT_SHA_SHORT=34295339-dirty`、`IMAGE_TAG=20260608T051344-34295339-enable-all-keys`。
- 线上操作：已用管理员 RPC 调用 `api.key_enable_all`，执行前加载到 11 个 key 且 11 个均为禁用，执行返回 `updated=11`，执行后禁用数为 0；`api.summary` smoke 通过，近 5 分钟容器日志未见 `panic` / `fatal`。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包和 migration tar 包，执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被任何容器使用的旧镜像并回收 353.2MB，未清理 volume。清理后根分区使用率 21%，当前 app-server 仍运行新镜像。
- 阻塞/风险：本轮只加全站恢复开关，不改单 key 启停、批量删除、key 重置、usage 真源、部署脚本或生产配置。

## 2026-06-04 用量日志客户端 IP 记录

- 完成：`gateway_usage_logs` 新增 `client_ip` 字段和 `client_ip + created_at` 索引，OpenAI-compatible `/v1` 网关请求在统一 usage 写入点记录客户端 IP；默认只在直连来源为本机、内网或 link-local 时采信 `X-Forwarded-For` / `X-Real-IP`，也支持用 `GATEWAY_TRUSTED_PROXY_CIDRS` 显式收紧可信反代 CIDR。
- 完成：管理端 `usage_list` RPC、CSV/JSON 导出和后台调用明细 / 会话请求级明细已带出 `client_ip`；本轮不记录请求体、响应体、prompt、模型输出正文或完整认证信息。
- 完成：`server/Makefile` 的 `GO_BUILDER_IMAGE` 默认值对齐 `server/Dockerfile` 的 `golang:1.26.4`，避免 `make build_server` 覆盖回旧 Go builder 导致部署构建失败。
- 文档：同步更新 `docs/architecture.md`、`server/docs/config.md`、`server/deploy/compose/prod/.env.example` 和 `web/README.md`，明确 usage IP 口径和可信反代配置。
- 验证：已执行 `make data`、`atlas migrate validate --dir "file://internal/data/model/migrate"`、`go test ./internal/server ./internal/data ./internal/biz -count=1`、`pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`、`STYLE_L1_SCENARIOS=admin-usage-desktop,admin-usage-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`、`gitleaks detect --redact --no-git --source .`、`git diff --check`，均通过。第一次后端窄测遇到本机 `httptest` 临时端口 `can't assign requested address`，单独重跑失败相关用例和全包重跑均通过；`gitleaks detect --redact --source .` 默认扫描 Git 历史时命中 5 个历史记录，非本轮工作树新增。
- 浏览器：本地 `web/` Vite 服务在 `127.0.0.1:5179` 启动后，HTTP 访问 `/admin-usage` 可返回前端应用；内置 Browser 未登录访问 `/admin-usage` 按既有鉴权回跳 `/admin-login`，页面非空且无横向溢出。完整管理态和 IP 展示以 `style:l1` mock RPC 回归为准。
- 部署：已提交并推送 `9f9822e feat: 记录网关请求客户端 IP`；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260604T205535-9f9822ec`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260604T205535-9f9822ec`。远端只执行 `docker load`、宿主机 Atlas migration、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 从 `20260604051355` 应用到 `20260604123931`，pending 0；解压 migration tar 时清理了 macOS `._*` 资源叉文件后再执行 Atlas。
- 线上验证：远端本机和公网 `/healthz` / `/readyz` 均通过；当前 `app-server` 运行镜像为 `oauth-api-service-server:20260604T205535-9f9822ec`，容器环境 `GIT_SHA_SHORT=9f9822ec`、`IMAGE_TAG=20260604T205535-9f9822ec`。管理员 `admin/adminadmin` 登录和 `api.summary` smoke 通过；临时 key 调用远端本机 `/v1/models` 返回 6 个模型，随后 `api.usage_list key_id=<temp>` 返回 `endpoint=models`、`status=200`、`client_ip=172.20.0.1`，临时 key 已删除。公网 `/admin-usage` 返回 200 并加载前端资源 `index.DrOxgU7U.js`。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包和 migration tar 包，执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被任何容器使用的旧镜像并回收 392.8MB，未清理 volume。清理后根分区使用率 21%，当前 app-server 仍运行新镜像。
- 阻塞/风险：本轮只完成 schema / usage 写入 / 管理端展示与导出，不做 IP 风控、GeoIP、黑名单或保留期清理；如后续需要按公网真实来源 IP 归因，应继续核对宿主机 Nginx / frp / Docker bridge 的反代头链路和 `GATEWAY_TRUSTED_PROXY_CIDRS` 配置。

## 2026-06-09 用量日志默认今天窗口

- 完成：将用量日志共享默认时间窗口从滚动 `24h` 调整为「今天」；`today` 仍按本地自然日 00:00 到当前时间计算，`24h` 仍保留为可手动选择的滚动 24 小时窗口。每日模型保持未手动选择时默认 30 天，不改看板趋势的 30 天默认、不改后端 usage 接口、schema 或迁移。
- 文档：同步更新 `web/README.md`，明确用量日志默认选择「今天」。
- 验证：已执行 `pnpm --dir web exec eslint --ext .js --ext .jsx src/common/utils/usageTimeRange.js scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`、`STYLE_L1_PORT=4355 STYLE_L1_SCENARIOS=admin-analytics-redirect,admin-usage-desktop,admin-usage-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`、`git diff --check -- web/src/common/utils/usageTimeRange.js web/scripts/styleL1.mjs web/README.md progress.md`，均通过。线上 Playwright 登录 `https://oauth-api.saurick.me/admin-usage` 后确认默认摘要显示「今天 范围内第」，不再显示「24h 范围内第」，时间范围 combobox 当前值为「今天」，控制台无 error。
- 部署：本地构建 linux/amd64 镜像 `oauth-api-service-server:20260609T103505-48598b49-local-default-today`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260609T103505-48598b49-local-default-today`。远端只执行 `docker load`、宿主机 Atlas status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建；Atlas 当前版本 `20260604123931`、pending 0。远端本机与公网 `/healthz` / `/readyz` 通过，管理员 `admin/adminadmin` 登录、`api.summary` 与今天窗口 `api.usage_list` smoke 通过。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包，执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260608T170813-48598b49-key-last-used-column`，回收 353.3MB，未清理 volume。清理后根分区使用率 23%，当前 app-server 运行镜像为 `oauth-api-service-server:20260609T103505-48598b49-local-default-today`。
- 阻塞/风险：本轮只改前端默认时间窗口和文档/回归断言；镜像因按“先部署后提交”流程构建，tag 带有 `48598b49-local` 标记，但运行代码已包含本轮改动。

## 2026-06-04 看板与用量日志指标说明

- 完成：`/admin-dashboard` 顶部核心指标卡和 `/admin-usage` 用量日志摘要指标卡新增问号说明，复用现有后台 tooltip 交互与暗色主题变量；说明范围覆盖今日 Token / 请求数、服务错误率、响应耗时、RPM/TPM、上游分布、客户端分布、费用估算和 API 凭据状态。
- 验证：已执行 `pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminDashboard/index.jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`pnpm --dir web lint`、`pnpm --dir web css`、`pnpm --dir web test`、`pnpm --dir web build`、`STYLE_L1_SCENARIOS=admin-dashboard-desktop,admin-usage-desktop NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`、`STYLE_L1_SCENARIOS=admin-dashboard-mobile,admin-usage-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`、`git diff --check -- web/src/pages/AdminDashboard/index.jsx web/src/pages/AdminApi/index.jsx web/src/tailwind.css web/scripts/styleL1.mjs progress.md`，均通过。`style:l1` 已覆盖 dashboard / usage 桌面与移动端目标页、tooltip hover 可见态、usage 暗色主题对比和既有表格盒模型；内置 Browser 已连接本地 Vite，但普通 dev mock 不覆盖 `/rpc/api` 管理数据且页面脚本沙箱不能直接写入管理 token，完整管理态以 `style:l1` mock RPC 回归为准。
- 阻塞/风险：本轮只改前端说明与回归断言，不改后端 summary / usage_list / usage_buckets 真源、schema、历史 usage 数据或部署配置。

## 2026-06-04 凭据统计今天 Token 列

- 完成：`/admin-usage` 的「凭据统计」固定窗口表新增「今天 Token」列；今天窗口同样按本地当天 00:00 到当前时间计算，区别于滚动 `24h`。对应 `usage_key_summaries` 请求新增今天窗口，不改后端接口、usage 真源、schema 或迁移。
- 完成：凭据统计表最小宽度从 1040px 调整为 1240px，新增列后在窄屏继续走横向滚动，不强行压缩列内容。
- 文档：同步更新 `web/README.md`，明确凭据统计固定窗口包含 `今天/24h/7 天/30 天/180 天/360 天/1 年/3 年/5 年`。
- 验证：已执行 `pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`git diff --check -- web/src/pages/AdminApi/index.jsx web/scripts/styleL1.mjs web/README.md progress.md`、`STYLE_L1_PORT=4347 STYLE_L1_SCENARIOS=admin-usage-desktop,admin-usage-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`，均通过。
- 部署：已提交并推送 `6195576 fix: 补充凭据统计今天窗口`；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260604T195047-61955762-key-token-today`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260604T195047-61955762-key-token-today`。远端只执行 `docker load`、宿主机 Atlas status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604051355`、pending 0；远端本机与公网 `/healthz` / `/readyz` 通过，管理员 `admin/adminadmin` 登录、`api.summary` 与 `api.usage_key_summaries` smoke 通过。
- 验证：线上 `/admin-usage` 通过 Playwright 登录后台并切到「凭据统计」后，DOM 表头包含「今天 Token / 24h Token / 7 天 Token / 30 天 Token / 180 天 Token / 360 天 Token / 1 年 Token / 3 年 Token / 5 年 Token」，页面发出的 `usage_key_summaries` 请求包含今天窗口。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包，执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260604T192636-bbdc18cf-usage-today-range`，回收 353.2MB，未清理 volume。清理后根分区使用率 21%，当前 app-server 运行镜像为 `oauth-api-service-server:20260604T195047-61955762-key-token-today`。
- 阻塞/风险：本轮只补凭据统计固定窗口表，不改顶部筛选时间范围、每日模型、会话聚合或后端聚合口径。

## 2026-06-04 Usage 时间范围今天选项

- 完成：`/admin-dashboard` 趋势时间范围与 `/admin-usage` 用量日志时间范围新增「今天」选项；今天窗口按本地当天 00:00 到当前时间计算，区别于滚动 `24h`。
- 完成：新增共享 `getUsageTimeWindow` / `startOfLocalDayUnix` 作为前端 usage 时间窗口计算入口，避免 dashboard 与 usage 页各自写日期分支；不改后端 `usage_list` / `usage_buckets` 接口、usage 真源、schema 或迁移。
- 文档：同步更新 `web/README.md` 的可选时间窗口列表。
- 验证：已执行 `pnpm --dir web exec eslint --ext .js --ext .jsx src/common/utils/usageTimeRange.js src/pages/AdminDashboard/index.jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`、`git diff --check -- web/src/common/utils/usageTimeRange.js web/src/pages/AdminDashboard/index.jsx web/src/pages/AdminApi/index.jsx web/scripts/styleL1.mjs web/README.md progress.md`、`STYLE_L1_PORT=4346 STYLE_L1_SCENARIOS=admin-dashboard-desktop,admin-dashboard-mobile,admin-usage-desktop,admin-usage-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`，均通过；第一次同组 `style:l1` 在 `admin-usage-mobile` 等待后台壳标题时超时，单跑该场景和随后同组重跑均通过，判断为加载偶发。内置 Browser 在 `http://127.0.0.1:5177/admin-dashboard` 登录后确认趋势时间范围包含「今天」，选择后文案变为「当前 今天 窗口」，select 宽度 128px、页面无横向溢出；`/admin-usage` 页面可达且时间范围 combobox 存在，Browser 对该自定义输入控件的 `fill` 受虚拟剪贴板限制，具体下拉交互以 `style:l1` mock RPC 回归为准。Browser 控制台仅有 React Router v7 future flag 既有 warning。
- 部署：已提交并推送 `bbdc18c feat: 增加今天用量时间范围`；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260604T192636-bbdc18cf-usage-today-range`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260604T192636-bbdc18cf-usage-today-range`。远端只执行 `docker load`、宿主机 Atlas status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604051355`、pending 0；第一次远端重建时 release 变量被 `.env` 覆盖导致仍跑旧镜像，已立即用独立 `NEW_APP_IMAGE` 显式修正并重建。远端本机和公网 `/healthz` / `/readyz` 通过，容器环境 `GIT_SHA_SHORT=bbdc18cf`、`IMAGE_TAG=20260604T192636-bbdc18cf-usage-today-range`，公网 `/admin-dashboard` 静态产物包含「今天」/ `today`，管理员 `admin/adminadmin` 登录、`api.summary` 与 `api.usage_buckets` smoke 通过。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包，执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260604T082133-eda95418-dashboard-trend-readable`，回收 353.2MB，未清理 volume。清理后根分区使用率 21%，当前 app-server 运行镜像为 `oauth-api-service-server:20260604T192636-bbdc18cf-usage-today-range`。
- 阻塞/风险：本轮先只给可手动选择的 usage 时间范围补「今天」；凭据 Token 统计表的固定窗口列已在后续「凭据统计今天 Token 列」中补齐。

## 2026-06-04 业务看板今日 Token 指标

- 完成：将 `/admin-dashboard` 顶部核心卡片从「今日消费」改为「今日 Token」，主值使用本地今日起点的 `summary.total_tokens`，副标题保留过去 24h 的 `summary.total_tokens` 对照；不改后端 summary 接口、usage 真源、schema 或迁移。
- 文档：同步更新 `web/README.md` 的业务看板指标说明，避免继续把首页首卡描述为费用口径。
- 验证：已随本轮时间范围变更一起执行前端 lint、`pnpm --dir web test`、`pnpm --dir web css`、`pnpm --dir web build`、目标页面 `style:l1` 和 `git diff --check`，均通过；`style:l1` 已断言首页核心卡片存在「今日 Token」且不再显示「今日消费」。内置 Browser 在 `http://127.0.0.1:5177/admin-dashboard` 登录后确认页面标题正确、内容非空、首卡显示「今日 Token」、页面无横向溢出；控制台仅有 React Router v7 future flag 既有 warning。
- 阻塞/风险：本轮只改首页展示口径；费用估算仍保留在用量趋势「费用」指标、最近调用和用量日志等既有入口。

## 2026-06-04 业务看板长窗口趋势图可读性

- 完成：修复 `/admin-dashboard` 用量趋势在 1 年以上时间范围下按天渲染过密的问题；长窗口仍按完整 `usage_buckets group_by=day` 请求取数，但前端图表会把相邻日期聚合成最多 72 个可交互展示桶，tooltip 展示聚合日期范围与汇总指标。
- 完成：趋势图底部日期刻度改为独立的少量刻度行，状态文案单独居中显示，避免长窗口下出现“6/5 按请求展示 6/4”挤在一起看不清。
- 完成：部署构建时发现 `server/Dockerfile` 的 Go builder 仍固定在 `golang:1.25.9`，与当前 `go.mod` 的 Go patch / toolchain 要求不一致，已同步到 `golang:1.26.4`，避免本地发布镜像构建失败。
- 验证：已执行 `pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminDashboard/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`git diff --check -- web/src/pages/AdminDashboard/index.jsx web/scripts/styleL1.mjs progress.md`、`STYLE_L1_PORT=4343 STYLE_L1_SCENARIOS=admin-dashboard-desktop,admin-dashboard-narrow-desktop,admin-dashboard-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`、`pnpm --dir web build`，均通过。内置 Browser 在用户当前 `http://localhost:5176/admin-dashboard` 页面刷新验证，切换到 3 年窗口后渲染 69 个展示桶、5 个日期刻度，图表无横向溢出；控制台仅有 React Router v7 future flag 既有 warning。
- 部署：已提交并推送 `ca90d53 fix: 优化看板长窗口趋势图` 与 `eda9541 fix: 对齐 Docker Go 构建镜像`；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260604T082133-eda95418-dashboard-trend-readable`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260604T082133-eda95418-dashboard-trend-readable`。远端只执行 `docker load`、宿主机 Atlas migration status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604051355`、pending 0；远端本机和公网 `/healthz` / `/readyz` 通过，容器环境 `GIT_SHA_SHORT=eda95418`、`IMAGE_TAG=20260604T082133-eda95418-dashboard-trend-readable`，公网 `/admin-dashboard` 返回 200，管理员 `admin/adminadmin` 登录、`api.summary` 与 3 年 `api.usage_buckets group_by=day` smoke 通过。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；删除远端 release 镜像 tar 包，执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260604T075618-80667e0f-dashboard-range-style`，回收 353.2MB，未清理 volume。清理后根分区使用率 21%，当前 app-server 运行镜像为 `oauth-api-service-server:20260604T082133-eda95418-dashboard-trend-readable`。
- 阻塞/风险：本轮只改前端展示层与浏览器级回归脚本，不改后端 `usage_buckets` 聚合 SQL、usage 真源、schema、迁移、部署脚本或生产配置。

## 2026-06-04 Go 漏洞依赖升级

- 完成：修复 `govulncheck` 可达漏洞告警，升级 `server/go.mod` 的 Go patch 指令到 `1.25.11` 并显式固定 `toolchain go1.26.4`，同步升级 `go.opentelemetry.io/otel/*` 到 `v1.43.0`、`golang.org/x/net` 到 `v0.53.0` 及相关间接依赖。未改 OAuth、网关转发、usage 统计、schema、迁移、前端页面、部署脚本或生产配置。
- 验证：已执行 `bash scripts/qa/govulncheck.sh`，结果为 0 个可达漏洞；已执行 `cd server && go test ./...`，通过。第一次全量测试中 `TestCodexBalanceRouteReturnsRateLimitsWithoutAuth` 曾返回 502，单测重跑通过后全量重跑也通过，判断为本机瞬时测试状态，不是本轮依赖升级后的稳定失败。
- 下一步：后续如要把公开 Codex 余额查询继续产品化，应单独补更稳定的 app-server fake / cache 隔离测试，避免本机状态影响全量测试判断。
- 阻塞/风险：本轮只处理服务端 Go 依赖安全更新和 `progress.md` 归档；未部署、未构建 Docker 镜像、未修改线上配置。

## 2026-06-04 Usage 错误类型状态码提示

- 完成：`/admin-usage` 的“错误 / 中断类型”下拉选项补充 HTTP 状态提示，稳定网关错误显示固定 HTTP 码，动态上游错误按当前网关落库口径显示“网关 502 / 上游码”或常见码；筛选值仍使用原 `error_type`，不改变后端查询、usage 真源或状态码独立筛选。
- 验证：已执行 `cd web && pnpm lint`、`cd web && pnpm css`、`cd web && pnpm test`、`cd web && pnpm build`、`cd web && STYLE_L1_SCENARIOS=admin-usage-desktop STYLE_L1_PORT=4199 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check -- web/src/common/utils/gatewayErrorTypes.js web/src/pages/AdminApi/index.jsx web/scripts/styleL1.mjs progress.md`，均通过；Browser 临时 mock 页面确认错误类型下拉可显示 `413 / SSE 502 · 上下文超限`，菜单无横向溢出。
- 下一步：如后续要在表格错误类型列也展示状态码，应单独评估信息密度，避免和已有状态码列重复。
- 阻塞/风险：本轮只改前端展示层与回归脚本；不改服务端错误归因、数据库 schema、历史 usage 或部署配置。

## 2026-06-04 业务看板趋势时间范围

- 完成：修复窄桌面宽度下「趋势时间范围」原生下拉被右侧控制组压缩的问题；为时间范围 label/select 设置稳定最小宽度，并在 `style:l1` 新增 `admin-dashboard-narrow-desktop` 视口与 select 盒模型断言。
- 完成：`/admin-dashboard` 的「30 天趋势」调整为「用量趋势」，新增时间范围下拉，复用「用量日志」的 `24h/7 天/30 天/90 天/180 天/1 年/2 年/3 年/5 年` 选项，默认仍为 30 天；趋势图和 Token 构成都跟随当前窗口，`usage_buckets group_by=day` 请求同步传入对应 `start_time/end_time`。
- 完成：新增 `web/src/common/utils/usageTimeRange.js` 作为前端 usage 时间窗口单一真源，`/admin-usage` 与 `/admin-dashboard` 共同引用，避免两处选项漂移；长窗口趋势图改为自适应列宽，避免 1 年以上按天聚合撑出横向滚动。
- 文档：同步更新 `web/README.md`，明确业务看板用量趋势复用用量日志时间窗口。
- 验证：本轮窄桌面样式修复后已执行 `pnpm --dir web exec eslint --ext .js --ext .jsx src/pages/AdminDashboard/index.jsx scripts/styleL1.mjs`、`node --check web/scripts/styleL1.mjs`、`STYLE_L1_PORT=4342 STYLE_L1_SCENARIOS=admin-dashboard-desktop,admin-dashboard-narrow-desktop,admin-dashboard-mobile NODE_USE_ENV_PROXY=0 pnpm --dir web style:l1`、`pnpm --dir web build`，均通过。内置 Browser 在用户当前 `http://127.0.0.1:5176/admin-dashboard` 页面刷新验证，默认 30 天和切换 7 天时 select 宽度均为 128px，图表与 Token 文案跟随窗口变化，页面无横向溢出；控制台仅有 React Router v7 future flag 既有 warning。
- 验证：已执行 `cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminDashboard/index.jsx src/pages/AdminApi/index.jsx src/common/utils/usageTimeRange.js scripts/styleL1.mjs`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4341 STYLE_L1_SCENARIOS=admin-dashboard-desktop,admin-dashboard-mobile NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check -- web/src/pages/AdminDashboard/index.jsx web/src/pages/AdminApi/index.jsx web/src/common/utils/usageTimeRange.js web/scripts/styleL1.mjs web/README.md progress.md`，均通过。内置 Browser 使用本地 mock RPC 打开 `/admin-dashboard`，确认页面标题和内容非空，默认选中 30 天，切换 7 天后趋势说明、Token 构成说明和图表列数跟随变化，页面无横向溢出；控制台仅有 React Router v7 future flag 既有 warning。
- 部署：已提交并推送 `30be6cf feat: 完善后台用量统计交互`；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260604T070806-30be6cff-dashboard-trend-range`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260604T070806-30be6cff-dashboard-trend-range`。远端只执行 `docker load`、宿主机 Atlas migration status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604051355`、pending 0；远端本机和公网 `/healthz` / `/readyz` 通过，容器环境 `GIT_SHA_SHORT=30be6cff`、`IMAGE_TAG=20260604T070806-30be6cff-dashboard-trend-range`，公网 `/admin-dashboard` 产物包含「用量趋势」和「趋势时间范围」，管理员 `admin/adminadmin` 登录、`api.summary` 与 `api.usage_buckets group_by=day` smoke 通过。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260604T062400-926470fb-usage-client-type`，回收 352.4MB，未清理 volume。清理后根分区使用率 20%，当前 app-server 运行镜像为 `oauth-api-service-server:20260604T070806-30be6cff-dashboard-trend-range`。
- 部署：已提交并推送 `80667e0 fix: 修复看板趋势时间选择样式`；本地构建 linux/amd64 镜像 `oauth-api-service-server:20260604T075618-80667e0f-dashboard-range-style`，上传到 `192.168.0.133:/data/openai-oauth-api-service/releases/20260604T075618-80667e0f-dashboard-range-style`。远端只执行 `docker load`、宿主机 Atlas migration status、更新 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建。Atlas 当前版本 `20260604051355`、pending 0；远端本机和公网 `/healthz` / `/readyz` 通过，容器环境 `GIT_SHA_SHORT=80667e0f`、`IMAGE_TAG=20260604T075618-80667e0f-dashboard-range-style`，公网 `/admin-dashboard` 产物包含「用量趋势」和「趋势时间范围」，管理员 `admin/adminadmin` 登录、`summary` 与 7 天 `usage_buckets group_by=day` smoke 通过。
- 清理：部署成功后记录远端 `/` 使用率、`docker system df` 与运行容器；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260604T070806-30be6cff-dashboard-trend-range`，回收 353.2MB，未清理 volume。清理后根分区使用率 21%，当前 app-server 运行镜像为 `oauth-api-service-server:20260604T075618-80667e0f-dashboard-range-style`。
- 阻塞/风险：本轮只改前端筛选与展示层，不改后端 `usage_buckets` 聚合 SQL、usage 真源、schema、迁移或部署配置。

## 2026-06-04 每日模型分页

- 完成：`/admin-usage` 的「每日模型」汇总表新增独立分页，默认每页 8 条并复用后台表格分页控件；筛选、重置或切换 usage 分段时每日模型分页回到第一页。每日模型详情弹窗仍按当天该模型的请求级明细分页，并已统一为同一套后台表格分页控件。
- 文档：同步更新 `web/README.md`，明确每日模型汇总表和详情弹窗支持统一分页。
- 验证：已执行 `cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4340 STYLE_L1_SCENARIOS=admin-usage-desktop,admin-usage-mobile NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check -- web/src/pages/AdminApi/index.jsx web/scripts/styleL1.mjs web/src/tailwind.css web/README.md progress.md`，均通过。内置 Browser 已打开本地 dev server，确认未登录访问 `/admin-usage` 按现有鉴权回跳 `/admin-login`，控制台仅有 React Router v7 future flag 既有 warning；普通 dev mock 不覆盖 `/rpc/api`，每日模型详情交互以 `style:l1` mock RPC 回归为准。
- 下一步：如生产数据里 30 天窗口的日期 + 模型组合继续增长，可再评估是否把 `usage_buckets group_by=day_model` 扩展为后端分页接口；本轮不改后端聚合口径。
- 阻塞/风险：本轮只做每日模型汇总表和详情弹窗分页样式统一，不改后端 `usage_buckets group_by=day_model` 聚合接口、schema、usage 真源或详情弹窗请求级分页口径。

## 2026-06-20 Codex 项目 skills 迁入

- 完成：将 openai-oauth 项目专属 `openai-oauth-docs-governance`、`openai-oauth-page-governance` 从个人 `~/.codex/skills` 迁入 `.agents/skills/`，作为仓库内 canonical，避免长期依赖本机副本。
- 完成：同步更新根 `README.md` 技术栈 / 路径表，说明 `.agents/skills/` 只承载 Codex 项目专属文档治理和页面治理 workflow；本轮未更新 `docs/README.md`，因为没有新增、删除或重命名 `docs/` 文档，也未改变 architecture / operations / deploy 文档分层。
- 验证：追加前 `progress.md` 为 210 行、55108 字节，未达到归档阈值；项目内两份 skill 已执行 `quick_validate.py` 均通过；对应 `SKILL.md` 已通过 Ruby YAML 解析；`.agents` 未被 gitignore 忽略。
- 下一步：后续修改 openai-oauth 项目专属 skill 时以 `.agents/skills/` 为真源；个人全局同名 skill 只可作为临时入口，不再单独维护。
- 阻塞/风险：本轮不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署配置、密钥、历史 Python MVP 或正式运维口径。

## 2026-06-20 Codex 代码审查 skill 补充

- 完成：新增 `.agents/skills/openai-oauth-code-review-governance/`，作为 openai-oauth 项目独立代码审查入口，覆盖任意会话中的 worktree / staged diff / commit review；审查重点收口到 OAuth、下游 API key、usage、上游策略、Codex backend / CLI fallback、secrets、公开余额接口、部署和 legacy Python 边界。
- 完成：同步根 `README.md` 技术栈 / 路径表中 `.agents/skills/` 职责，从文档治理 / 页面治理扩展为文档治理、页面治理和代码审查；本轮未更新 `docs/README.md`，因为没有新增、删除或重命名 `docs/` 文档，也未改变 architecture / operations / deploy 文档分层。
- 验证：追加前 `progress.md` 为 218 行、56262 字节，未达到归档阈值；已执行 `quick_validate.py`（通过临时 PyYAML 路径）验证 `code-review-governance` 与 `openai-oauth-code-review-governance` 均通过；已执行 Ruby YAML 解析、TODO / 默认提示扫描、`git diff --check -- .agents/skills/openai-oauth-code-review-governance README.md progress.md`，通过。
- 下一步：后续 review 可直接在独立会话或当前会话使用 `$openai-oauth-code-review-governance`；涉及官方 OpenAI API 当前行为时仍需另行核对官方文档。
- 阻塞/风险：本轮只新增 Codex skill 和入口说明，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署配置、密钥、历史 Python MVP 或正式运维口径。

## 2026-06-20 Codex skill UI 名称英文化

- 完成：将 `.agents/skills/openai-oauth-code-review-governance/agents/openai.yaml` 的 `display_name` 改为英文 `OpenAI OAuth Code Review Governance`；项目内 docs/page governance 的 `display_name` 已是英文，无需改动。
- 验证：追加前 `progress.md` 为 226 行、57793 字节，未达到归档阈值；已扫描相关 skills 的 `display_name`，确认无中文命中；后续以 skill 正文保持中英结合，UI chip 名称保持英文。
- 下一步：如 Codex UI 仍显示旧名称，重新打开会话或等待 skill metadata 刷新。
- 阻塞/风险：本轮只改 skill UI metadata，不改 `SKILL.md` 规则正文、运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署配置或密钥。

## 2026-06-21 Codex 测试治理 skill 补充

- 完成：新增 `.agents/skills/openai-oauth-test-governance/`，作为 openai-oauth 项目专属测试治理入口，覆盖 Go / web / admin UI / migration / auth / API key / quota / usage logging / Codex backend / secrets / deploy preflight 验证选择；同步根 `README.md` 中 `.agents/skills/` 职责为文档治理、页面治理、代码审查和测试治理。
- 完成：同步新增通用 `~/.codex/skills/test-governance/`，用于跨项目测试分类和验证范围选择；项目内仍以 `.agents/skills/openai-oauth-test-governance/` 承载 openai-oauth 专属命令与边界。
- 验证：追加前 `progress.md` 为 233 行、58594 字节，未达到归档阈值；已执行 `quick_validate.py` 验证通用 `test-governance` 与项目 `openai-oauth-test-governance` 均通过；已执行 Ruby YAML 解析、TODO 扫描、中文 `display_name` 扫描、默认提示扫描和 `git diff --check`，均通过。
- 下一步：后续涉及测试选择、auth/API-key/quota/usage/Codex backend/admin UI/deploy 验证或“是否测试充分”时优先使用 `$openai-oauth-test-governance`；只需要通用测试分类时可用 `$test-governance`。
- 阻塞/风险：本轮只新增 Codex skill、README 入口和过程记录，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署配置、密钥或真实测试脚本；因此未运行 Go/web/full/strict、真实上游 smoke 或远端部署验证。

## 2026-06-21 Codex 提示词治理 skill 补充

- 完成：新增 `.agents/skills/openai-oauth-prompt-governance/`，作为 openai-oauth 项目专属提示词治理入口，覆盖 auth、API key、quota、usage logging、Codex backend、secrets、proxy/upstream、admin UI、deploy preflight、提交推送和交接提示词；同步根 `README.md` 中 `.agents/skills/` 职责为文档治理、页面治理、代码审查、测试治理和提示词治理。
- 完成：通用 `~/.codex/skills/prompt-governance/` 已存在，用于跨项目提示词治理；项目内仍以 `.agents/skills/openai-oauth-prompt-governance/` 承载 openai-oauth 专属边界。
- 验证：追加前 `progress.md` 为 241 行、60098 字节，未达到归档阈值；已执行项目 `openai-oauth-prompt-governance` 和通用 `prompt-governance` 的 `quick_validate.py`、Ruby YAML 解析、TODO 扫描、中文 `display_name` 扫描、默认提示扫描和 `git diff --check`，均通过。
- 下一步：后续新开主会话、side chat、review 会话或需要把 openai-oauth 需求整理成可执行任务时，优先使用 `$openai-oauth-prompt-governance`；跨项目通用提示词整理使用 `$prompt-governance`。
- 阻塞/风险：本轮只新增 Codex skill、README 入口和过程记录，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署配置、密钥、真实测试脚本或远端部署；因此不运行 Go/web/full/strict、真实上游 smoke 或远端部署验证。

## 2026-06-21 Codex 高风险治理 skills 补充

- 完成：新增项目专属 `.agents/skills/openai-oauth-release-governance/`、`openai-oauth-domain-boundary-governance/`、`openai-oauth-runtime-diagnostics/`、`openai-oauth-observability-error-governance/`、`openai-oauth-security-privacy-governance/`；未新增 seed/import 专属版，因为该服务不是数据导入型 ERP，避免误触发。
- 完成：同步根 `README.md` 中 `.agents/skills/` 职责，并补充项目 prompt-governance 的 skill pairing 表，方便后续一次提示词带出相关治理 skill。
- 验证：追加前 `progress.md` 为 249 行、61610 字节，未达到归档阈值；本轮只改 skill / README / progress，不改运行时代码、schema、migration、RBAC、部署脚本或生产配置；验证命令见本轮最终回复。
- 下一步：后续涉及发布/部署/版本、运行报错、业务边界、可观测错误或安全隐私任务时优先使用对应项目 skill；跨项目通用 seed/import 任务可用全局 skill。
- 阻塞/风险：新 skill 是执行治理入口，不等于已经修改 release 脚本、监控系统、安全策略或真实导入流程；如需自动守卫仍需后续落到脚本、测试或 CI/hook。

## 2026-06-21 Codex 高风险治理 skills 中英可读性修正

- 完成：将项目专属 `openai-oauth-release-governance`、`openai-oauth-domain-boundary-governance`、`openai-oauth-runtime-diagnostics`、`openai-oauth-observability-error-governance`、`openai-oauth-security-privacy-governance` 的 `SKILL.md` 改为中文主线 + English anchors；`name` 和 UI `display_name` 保持英文，`description` / `default_prompt` 改为中英结合。
- 完成：同步更新通用 `~/.codex/skills/` 中 6 个同类高风险治理 skill 的中英可读性；openai-oauth 仍不新增项目专属 seed/import skill，避免误触发。
- 验证：追加前 `progress.md` 为 257 行、62858 字节，未达到归档阈值；已执行 29 个相关 skill 目录的 `quick_validate.py`，均通过。
- 下一步：后续如继续发现旧治理 skill 正文过度英文，可按同一口径逐个补中文主线，不改 `$skill-name`。
- 阻塞/风险：本轮只改 Codex skill 文本和 metadata，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署脚本、监控系统或安全策略。

## 2026-06-22 Codex 项目 skills metadata 中英化补全

- 完成：统一修正项目内全部 `.agents/skills/*` 的 `SKILL.md` frontmatter `description`、`agents/openai.yaml` 的 `short_description` 和 `default_prompt`，避免 UI 摘要继续显示英文-only；`name`、目录名和 `display_name` 仍保持英文，方便 `$skill-name` 触发。
- 完成：给项目和通用治理 skill 正文顶部补充中文主线 + English anchors 的阅读口径，并在 `/Users/simon/.codex/AGENTS.md` 写入全局规则，后续创建或维护项目相关 skill 时默认遵守同一口径。
- 验证：追加前 `progress.md` 为 265 行、63988 字节，未达到归档阈值；已执行 54 个治理 skill 目录的 `quick_validate.py`，54 个 `agents/openai.yaml` Ruby YAML 解析通过；扫描确认 description 中文开头、`short_description` 含中文、`display_name` 无中文、`default_prompt` 包含 `$skill`。
- 下一步：如 Codex UI 仍显示旧摘要，重新打开会话或等待 skill metadata 刷新；后续新增 skill 应先按全局 AGENTS 的中英规则写 metadata。
- 阻塞/风险：本轮只改 Codex skill 文本、metadata 和全局 AGENTS 规则，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署脚本、监控系统或安全策略。

## 2026-06-22 项目 AGENTS skill 维护规则补充

- 完成：在项目级 `AGENTS.md` 增加“项目专属 Skill 维护约定”，明确 `.agents/skills/<skill-name>/` 随项目 git 管理、全局 `~/.codex/skills/` 只放通用 skill、项目版 skill 需包含 Truth Chain / Project Rules / Workflow / Output / Validation 等约束。
- 完成：同步写清 skill 命名与 metadata 口径：`name`、目录名、`display_name` 保持英文；`description`、正文、`short_description`、`default_prompt` 使用中文主体 + English anchors。
- 验证：追加前 `progress.md` 为 273 行、65310 字节，未达到归档阈值；本轮只改项目级 AGENTS / progress，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、页面或部署脚本；已执行 `git diff --check -- AGENTS.md progress.md`。
- 下一步：后续新增或维护项目 skill 时，按项目 AGENTS 和全局 AGENTS 的一致规则执行；如只改 skill 正文且职责不变，通常不需要改 `docs/README.md`。
- 阻塞/风险：本轮规则只约束后续 skill 维护，不代表已经修改任何自动 hook、CI、监控系统、安全策略或真实业务流程。

## 2026-06-22 页面治理与后端边界 skill 说明收口

- 完成：补充 `openai-oauth-page-governance` 与 `openai-oauth-domain-boundary-governance` 的边界说明，明确 admin 页面如果涉及 auth/API key/quota/usage logging/gateway/upstream/admin API/schema/migration/error code，应先回到 domain skill 定义后端/API/auth/usage/upstream 边界。
- 完成：同步修正通用 `~/.codex/skills/page-design-governance` 与 `domain-boundary-governance` 的页面 / 后端边界说明，避免服务版继续从通用版漂移。
- 验证：追加前 `progress.md` 为 281 行、66510 字节，未达到归档阈值；已执行相关 skill validator、YAML 解析和 diff 检查。
- 下一步：openai-oauth 管理端页面任务若需要 auth/API/usage/upstream 能力，先切到 `$openai-oauth-domain-boundary-governance`，再回到页面 skill 做 UI 回归。
- 阻塞/风险：本轮只改 skill 文本和过程记录，不新增 backend skill，不改运行时代码、schema、auth/key 语义、usage 真源、上游策略、部署脚本、监控系统或测试实现。
