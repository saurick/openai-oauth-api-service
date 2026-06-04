## 归档索引

- 2026-06-04：旧 `progress.md` 已按超过 600 行阈值归档到 `docs/archive/progress-2026-06-04-before-govulncheck.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。

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
