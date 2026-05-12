## 归档索引
- 2026-05-10 之前历史流水：`docs/archive/progress-2026-05-10-pre-docker-cleanup-constraint.md`。
- 当前文件保留 2026-05-10 以来新增记录；归档文件只作追溯线索，不作为当前正式需求真源。


## 2026-05-12 线上 502 排查
- 完成：公网 `/healthz`、`/readyz` 均返回正常，最小 `/v1/responses` 非流式请求返回 `OK`，说明 Cloudflare、入口层、app-server 与 Codex 登录态主路径当前可用。
- 完成：通过管理员 JSON-RPC 查询线上 usage，当前上游策略为 `strategy=backend_with_cli_fallback`、`mode=codex_backend`、`fallback_enabled=true`；失败明细共 190 条，均为 `status_code=502`、`error_type=codex_backend_upstream_failed`，其中近 200 条成功请求仍主要走 `codex_backend`，说明不是服务整体不可用。
- 发现：失败请求集中在 `/v1/responses`、`reasoning_effort=high`，请求体大小从约 48KB 到 1.43MB，近期失败多为 300KB~470KB；带工具调用或工具历史的请求属于 backend-only，代码会禁止 fallback 到 `codex_cli`，因此即使后台打开 Backend + CLI 兜底，仍会直接返回 502。
- 验证通过：公网 `/v1/models` 正常；最小 `/v1/responses`、函数工具调用、带 `function_call` 历史的 Responses 请求均返回 `HTTP 200`，未复现旧的 function_call id 校验问题。
- 阻塞/风险：当前本机到 `8.218.4.199:22` SSH 超时，未能读取远端容器日志中的上游原始错误 body；下一步需要从可达网络或云控制台进入服务器执行 `docker logs --since ... openai-oauth-api-service-server`，重点看 `codex backend upstream failed before fallback` 的 status/body，再决定是上游 429/5xx、请求过大/上下文过长、工具历史格式，还是代理/出口抖动。

## 2026-05-12 API 凭据多选与备注同步改写部署
- 完成：修正 `api.key_update` 备注口径，编辑备注时保留原随机段并同步改写 `plain_key`、`key_hash`、`key_prefix`、`key_last4`；文档、前端提示和进度说明已从“编辑不改写 key”改为当前真源，避免继续沿用旧风险说明。
- 完成：API 凭据表保持单击行互斥单选，复选框支持累加多选；用量日志「调用凭据」筛选改为多选，前端统一向 usage 列表、每日模型、凭据统计、会话聚合、异常请求和导出路径传 `key_ids`，后端保留旧 `key_id` 兼容且多选优先。
- 验证通过：`cd server && go test ./internal/biz ./internal/data ./internal/server`、`cd web && pnpm test && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs && node --check scripts/styleL1.mjs`、`cd web && pnpm css && pnpm build`、`cd web && STYLE_L1_PORT=4324 NODE_USE_ENV_PROXY=0 pnpm style:l1`。
- 部署：本地构建镜像 `oauth-api-service-server:20260512T043120-bc28db59-local`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260512T043120-bc28db59-local/`；远端仅执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 线上验证：远端容器运行镜像为 `oauth-api-service-server:20260512T043120-bc28db59-local`，容器内 `GIT_SHA=bc28db59e8df9621fa7333fe21f5f98e2f207cd7-local`，`CODEX_UPSTREAM_MODE=codex_backend`、`CODEX_UPSTREAM_FALLBACK_ENABLED=false`；远端本机与公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`，公网 `/admin-login` 返回 `HTTP 200`；远端本机 RPC `usage_list` 携带 `key_ids:[1,2]` 返回 `code=0`、`total=69`；`opencode models oauth-api-service` 正常列出 `gpt-5.4/gpt-5.5`。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260511T230544-bc28db59-local`，回收 `347.6MB`；根分区从清理前 `47%` 回到 `46%`，未执行 volume prune。
- 阻塞/风险：本轮没有 schema 变更，不需要 Atlas migration；编辑备注会使旧完整 key 失效，使用方需同步替换为后台显示的新完整凭据。远端 release tar 包仍保留用于短期追溯。

## 2026-05-11 API 凭据备注前缀生成
- 完成：新建 API 凭据时，后端按 `ogw_<备注>_<随机串>` 生成明文 key；备注为空时先生成默认备注再写入明文前缀，历史凭据更正另见后续记录。
- 完成：备注校验收口到后端业务层，创建与编辑均只允许 ASCII 字母和数字；前端备注输入同步过滤非字母数字字符，凭据弹窗关闭浏览器原生校验以统一走页面中文错误提示，并更新新建弹窗提示、mock 与文档口径。
- 修正：空备注默认名称从 `key <last4>` 调整为 `key<last4>`，避免默认备注自身违反字母数字规则。
- 验证通过：`cd server && go test ./internal/biz ./internal/data`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs && node --check scripts/styleL1.mjs && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4324 NODE_USE_ENV_PROXY=0 pnpm style:l1`。
- 阻塞/风险：历史已有非字母数字备注会原样展示，但再次保存时需要改成新规则；后续编辑备注会改写 key 备注段并同步哈希，客户端需替换为新的完整凭据。

## 2026-05-11 API 凭据备注前缀部署与历史 key 更正
- 部署：本地构建镜像 `oauth-api-service-server:20260511T225044-bc28db59-local`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260511T225044-bc28db59-local/`；远端仅执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 完成：本地开发库 3 条、线上生产库 9 条历史 `gateway_api_keys` 均更正为 `ogw_<备注>_<原随机串>` 形式，并同步更新 `name`、`plain_key`、`key_hash`、`key_prefix`、`key_last4`；本地备份表为 `gateway_api_keys_backup_remark_prefix_local_20260511T2255`，线上备份表为 `gateway_api_keys_backup_remark_prefix_remote_20260511T2255`。
- 完成：本机 `~/.config/opencode/opencode.json` 与 `~/.codex/config.toml` 中命中的旧 key 已替换为新 key，备份分别为 `.bak-20260511T2255-before-key-remark-prefix`。
- 验证通过：线上本机与公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`，公网 `/admin-login` 返回 `HTTP 200`；本地脚本校验本地 3 条、线上 9 条均满足 `sha256(plain_key)=key_hash`、`key_prefix=plain_key[:12]`、`key_last4=plain_key[-4:]`；使用线上新 key 调用公网 `/v1/models` 返回 `HTTP 200` 和 6 个模型；`opencode models oauth-api-service` 返回 `gpt-5.4/gpt-5.5`，`codex exec ... model_provider="saurick-oauth" "只回复 OK"` 返回 `OK`。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260511T211545-bc28db59`，回收 `347.6MB`；根分区从清理前 `47%` 回到 `45%`，未执行 volume prune。
- 阻塞/风险：本轮没有 schema 变更，不需要 Atlas migration；旧 key 已失效，仍保存旧 key 的其他客户端需要同步替换。Codex 验证期间仍出现插件/模型刷新 warning，但请求最终完成并返回 `OK`。

## 2026-05-11 空备注 key 前缀补丁部署
- 修复：补齐空备注创建顺序问题，后端现在先确定默认备注 `key<last4>`，再生成 `ogw_key<last4>_<随机串>`；避免空备注新建仍只得到 `ogw_<随机串>`。
- 部署：本地构建镜像 `oauth-api-service-server:20260511T230544-bc28db59-local`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260511T230544-bc28db59-local/`；远端仅执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 验证通过：`cd server && go test ./internal/biz ./internal/data`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs && node --check scripts/styleL1.mjs && pnpm test`、`cd web && pnpm css && pnpm build`、`cd web && STYLE_L1_PORT=4324 NODE_USE_ENV_PROXY=0 pnpm style:l1`。
- 线上验证：远端本机 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；远端本机 RPC 创建空备注临时 key 返回 `created_name=key49sI`、`plain_key` 形如 `ogw_key49sI_...`，验证后已删除临时 key。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260511T225044-bc28db59-local`，回收 `347.6MB`；根分区从清理前 `47%` 回到 `45%`，未执行 volume prune。
- 阻塞/风险：公网脚本直连 `/rpc/auth` 被入口层返回 `403`，本轮 RPC 验证改走服务器本机 `127.0.0.1:8400`；浏览器后台页面不受该脚本入口限制影响。

## 2026-05-11 bc28db59 线上部署
- 完成：本地构建镜像 `oauth-api-service-server:20260511T211545-bc28db59`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260511T211545-bc28db59/`；远端仅执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 完成：远端当前运行镜像为 `oauth-api-service-server:20260511T211545-bc28db59`，容器内 `GIT_SHA=bc28db59e8df9621fa7333fe21f5f98e2f207cd7`。
- 验证通过：`cd server && go test -count=1 ./...`、`cd web && pnpm test -- --run`、`cd web && pnpm build`；远端本机与公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`，公网 `/admin-login` 返回 `HTTP 200`；`opencode models oauth-api-service` 正常列出 `gpt-5.4/gpt-5.5`，`opencode run --pure -m oauth-api-service/gpt-5.5 --variant low --format json '只回复 OK'` 返回 `OK`。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，仅删除未被容器使用的旧镜像 `20260511T201505-d530adb7-local`，回收 `347.6MB`；根分区从清理前 `46%` 回到 `45%`，未执行 volume prune。
- 阻塞/风险：本轮没有新增 schema migration，未执行 Atlas migration；远端 release tar 包仍保留用于短期追溯。

## 2026-05-11 上游策略三态与无兜底部署
- 完成：后台 `/admin-upstream` 从旧「上游模式」改为「上游策略」，前端展示三种策略：Backend 直连、Backend + CLI 兜底、强制 CLI；当前线上已持久化为 `backend_only`，即 `mode=codex_backend`、`fallback_enabled=false`。
- 完成：`api.gateway_upstream_get` / `api.gateway_upstream_set` 返回并接受 `strategy`，同时保留旧 `mode + fallback_enabled` 入参兼容；服务端运行时读取同一套设置，避免 UI 只改文案但实际 fallback 口径不变。
- 完成：JSON-RPC 业务日志和 Kratos HTTP middleware `args` 均对 password、token、plain_key 等敏感参数脱敏，避免后台登录校验时把认证信息写入容器日志。
- 部署：本地构建最终镜像 `oauth-api-service-server:20260511T201505-d530adb7-local`，上传到远端 `/data/openai-oauth-api-service/releases/20260511T201505-d530adb7-local/`；远端仅执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 验证通过：`cd server && go test -count=1 ./api/jsonrpc/v1 ./internal/server ./internal/biz ./internal/data`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx src/pages/AdminDashboard/index.jsx src/common/components/layout/AdminFrame.jsx scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4325 NODE_USE_ENV_PROXY=0 pnpm style:l1`。
- 线上验证：`/healthz` 返回 `ok`、`/readyz` 返回 `ready`；`gateway_upstream_get` 返回 `strategy=backend_only`、`mode=codex_backend`、`fallback_enabled=false`、`options=3`；Playwright 真实打开线上 `/admin-upstream`，确认侧栏 / 面包屑 / 标题为「上游策略」，三项 tab 均存在且 Backend 直连已选中，不再显示旧「Backend 优先」。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被容器使用的旧 `oauth-api-service-server` 镜像，回收 `1.043GB`；根分区从清理前 `49%` 回到 `44%`，未执行 volume prune。
- 阻塞/风险：本轮没有 schema 变更，不需要 migration；远端 release tar 包仍保留用于短期追溯。日志脱敏不重写历史容器日志，已存在的旧日志记录仍按容器日志保留策略自然滚动。

## 2026-05-11 Codex function_call call_* id 502 修复
- 发现：线上近 2 小时 `502` 实际来自服务端将 Codex backend 上游 `400` 映射为 `codex_backend_upstream_failed`；上游错误为 `Invalid 'input[4].id': 'call_...'. Expected an ID that begins with 'fc'.`，说明 OpenAI/Codex 客户端工具历史中的 `tool_call.id=call_*` 被错误当作 Responses `function_call.id` 原样转发。
- 修复：direct backend adapter 对 Responses `function_call` item id 增加 `fc*` 前缀约束；空 id、非法 id、以及合法字符但非 `fc*` 的 `call_*` id 都统一按 `call_id` 生成 `fc_*`，同时保留原始 `call_id` 语义，避免工具结果关联丢失。
- 部署：本地构建镜像 `oauth-api-service-server:20260511T195435-d530adb-fc-id`，上传到远端 `/data/openai-oauth-api-service/releases/20260511T195435-d530adb-fc-id/`；远端仅执行 `docker load`、备份并更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 验证通过：`cd server && go test ./internal/server ./internal/biz ./internal/data`、`git diff --check -- server/internal/server/codex_backend_adapter.go server/internal/server/openai_gateway_handler_test.go progress.md`；线上 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`。
- 验证通过：公网真实 `/v1/chat/completions` 请求包含 `tool_calls.id=call_7DrSS117GxOq1ztYHVJCjrjZ` 和 tool result 历史，返回 `HTTP 200`、`finish_reason=stop`，确认不再出现 `Expected an ID that begins with 'fc'`。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，仅删除未被容器使用的旧镜像 `20260511T194715-d530adb7-local`，回收 `347.6MB`；根分区从清理前 `45%` 回到 `43%`，当前运行镜像和数据库 volume 保持正常。
- 阻塞/风险：本轮只修复 direct backend 工具历史 id 映射，不改变 fallback 策略；带工具调用、工具历史或文件输入的请求仍不会 fallback 到 CLI。远端 release tar 包仍保留用于短期追溯。

## 2026-05-11 Windows Codex function_call 空 id 修复
- 发现：Windows Codex 自定义 provider 发消息时，服务端日志持续出现 `codex backend upstream failed`，上游 400 原因为 `Invalid 'input[7].id': ''`；健康检查正常，说明不是 key、域名或服务进程不可用，而是 Responses 历史中的 `function_call.id` 空字符串被原样转发。
- 修复：direct backend adapter 在转换 `function_call` 历史时统一校验 item `id`，空值或非法字符会按 `call_id` 生成合法 `fc_*` id，避免触发 Codex backend 的 `input[n].id` 严格校验。
- 修复：`/v1/models` 的 Codex CLI 兼容 `models[]` 补齐 reasoning levels、shell type、context window、输入模态、availability nux 和 model messages 等模型元数据，减少自定义 provider 启动时的模型刷新解析报错。
- 部署：本地构建并部署最终镜像 `oauth-api-service-server:20260511T194715-d530adb7-local`；远端仅执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE` 和 `docker compose up -d app-server`，未在服务器构建。
- 验证通过：`cd server && go test ./internal/server ./internal/biz ./internal/data`；公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；Windows `codex exec --skip-git-repo-check --ephemeral --ignore-rules --json -s read-only -m gpt-5.5 -c model_provider="saurick-oauth" "只回复 OK"` 返回 `item.completed`，文本为 `OK`，且不再输出模型刷新解析 warning。
- 验证通过：远端日志窗口内未再出现 `input[n].id`、`failed to decode models response` 或 `codex backend upstream failed`。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，未执行 volume prune；根分区从清理前 `45%` 回到 `43%`，当前运行容器和镜像 `20260511T194715-d530adb7-local` 保持正常。
- 阻塞/风险：本轮没有 schema 变更，不需要 migration；远端 release tar 包仍保留用于短期回滚，后续可按发布归档策略清理旧 release 包。

## 2026-05-11 后台分页 trade-erp 风格
- 完成：后台共享 `TablePagination` 改为 trade-erp 风格，展示“共 N 条”、圆形数字页码、左右箭头和 `8 条/页` 下拉；支持直接点击数字页码，保留原分页状态、每页条数和后端 `limit/offset` 逻辑。
- 完成：分页样式收口到 `web/src/tailwind.css`，使用后台主题变量，浅色和暗色下当前页均为绿色描边圆形，移动端保持紧凑换行且不把每个按钮拉成整行。
- 验证通过：`cd web && pnpm exec eslint --fix --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && pnpm css`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4324 NODE_USE_ENV_PROXY=0 pnpm style:l1`。
- 验证补充：内置 Browser 登录真实本地 `/admin-usage` 并切到「调用明细」，确认分页 DOM 为 `共 47 条`、页码 `1 2 3 … 6`、`8 条/页`，点击下一页后当前页变为 `2`、上一页可用；Playwright 额外覆盖浅色 / 暗色 / 移动端截图、直接点击数字页码、页容量切换到 `10 条/页`、无横向溢出。
- 阻塞/风险：内置 Browser 截图和下拉输入接口本轮出现超时，截图留证使用本地 Playwright 输出到 `/tmp/openai-oauth-pagination-*.png`；本轮只改共享分页控件和样式，不改 usage 数据真源、筛选字段或导出逻辑。

## 2026-05-11 Usage Reasoning Effort 统计
- 完成：usage log 新增请求级 `reasoning_effort` 快照字段和 Atlas migration `20260511033926`；OpenAI-compatible 入参的 `reasoning_effort` / `reasoningEffort` / `reasoning.effort` 统一归一化为 `low`、`medium`、`high`、`xhigh` 后落库，历史旧数据保持空值。
- 完成：`api.usage_list`、管理端 usage CSV/JSON 导出、后台最近调用、调用明细、每日模型下钻和会话请求明细均展示 `reasoning_effort`；用量日志新增 Effort 筛选并透传到后端查询。
- 完成：补充后端测试覆盖 direct backend 对所有 effort 输出 OpenAI Responses `reasoning.effort`，Codex CLI 对所有 effort 输出 `model_reasoning_effort`，并验证 backend 真实 HTTP 请求体包含对应 effort。
- 下一步：默认聚合仍保持日期 / 模型主维度，不把每日模型表强拆成 `date + model + effort`；后续如需成本归因再新增独立 `day_model_effort` 聚合。
- 阻塞/风险：旧 usage 没有历史请求体，不能可靠回填 effort；本轮不根据 reasoning tokens 反推，避免污染统计口径。

## 2026-05-11 OpenCode 新会话标题修复
- 完成：排查 OpenCode 1.14.33 本地实现后确认标题生成走隐藏 `title` agent，并通过 `small_model` 解析为 `provider/model`；当前配置写成 `oauth-api-service/gpt-5.4/high`，会被解析成模型 ID `gpt-5.4/high`，本地 provider 不存在该模型，标题任务在发出网关请求前失败。
- 完成：已备份 `~/.config/opencode/opencode.json` 到 `~/.config/opencode/opencode.json.bak-20260511114022-before-small-model-titlefix`，并将 `small_model` 改为 `oauth-api-service/gpt-5.4`；主聊天 `model` 仍保持 `oauth-api-service/gpt-5.5/high`，不改变默认 agent variant。
- 验证通过：`opencode debug config` 显示 `small_model=oauth-api-service/gpt-5.4` 且 provider 存在 `gpt-5.4`；新建 `opencode run --pure` 会话时日志已出现 `agent=title small=true modelID=gpt-5.4`，线上 usage 同步出现 `gpt-5.4 200` 标题请求。
- 阻塞/风险：`opencode run --pure` 进程会在主回复完成后退出，标题异步任务虽已发出请求但未必有时间写回本地 session；桌面常驻应用需要重启以重新加载配置，之后新建会话应触发并写回标题。历史已创建的 `New session` 不会自动回填。

## 2026-05-11 凭据限额与统计回显线上部署
- 完成：提交并推送 `83b01be6`（完善凭据限额与统计回显），本地构建镜像 `oauth-api-service-server:20260511T112203-83b01be6`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260511T112203-83b01be6/`；远端只执行 `docker load`、Atlas migration、更新 Compose `.env` 的 `APP_IMAGE` 和 `docker compose up -d app-server`，未在服务器构建。
- 完成：线上 Atlas migration 从 `20260510141225` 升级到 `20260511024741`，为 `gateway_api_keys` 增加日 / 周输入 Token、输出 Token、非缓存输入 Token 限额列，旧数据默认 `0` 表示不限。
- 验证通过：远端当前运行镜像为 `oauth-api-service-server:20260511T112203-83b01be6`；容器内 `GIT_SHA=83b01be67e9269bbf5e403a9fdb947d021aa0aaa`、HTTP timeout `650s`、gRPC timeout `10s`；`/healthz` 返回 `ok`，`/readyz` 返回 `ready`，公网 `https://oauth-api.saurick.me/admin-login` 返回 `HTTP 200`。
- 验证通过：`admin_login` 与后台 `key_list` 返回 `code=0`，线上 `key_list` 已包含新增细分限额字段；Atlas 状态为 `OK`、当前版本 `20260511024741`、待执行迁移 `0`。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，未执行 volume prune；根分区从清理前 `39%` 回到 `37%`，当前运行容器和数据库 volume 保持不变。
- 阻塞/风险：本轮保留 release 镜像包和 Compose 备份用于短期回滚；如果后续继续频繁发布，需定期清理旧 release tar 包，不能删除数据库目录或运行中容器依赖镜像。

## 2026-05-11 OpenCode PDF 附件支持
- 完成：服务端 OpenAI-compatible 请求解析新增 PDF 文件部件支持，兼容 `input_file` / `file`、`file_data` data URL，以及带 `mimeType=application/pdf` 的 raw base64；direct `codex_backend` 转为 Codex backend `input_file`，单次最多 4 个 PDF、单个最大 16 MiB。
- 完成：`codex_cli` 路径仍只支持图片 `--image`，遇到 PDF 会明确返回“仅 codex_backend 支持文件输入”，避免 fallback 后丢失 PDF 内容却伪装成功；图片 data URL 也在 direct backend 路径补充大小与格式校验。
- 完成：本机 OpenCode `oauth-api-service` 与 `oauth-api-service-local` 的 `gpt-5.5/gpt-5.4` 已将 `modalities.input` 扩为 `text/image/pdf`，配置备份为 `~/.config/opencode/opencode.json.bak-20260511113631-before-pdf-modalities`。
- 完成：补跑 Ent 生成并生成 migration `20260511033926_migrate.sql`，补齐既有 `gateway_usage_logs.reasoning_effort` schema 改动缺失的 Ent 生成代码；线上已通过本地 Atlas + SSH 隧道从 `20260511024741` 迁移到 `20260511033926`。
- 完成：本地构建镜像 `oauth-api-service-server:20260511T113835-dc1649c0-pdf-attachments`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260511T113835-dc1649c0-pdf-attachments/`；远端只执行 `docker load`、Atlas migration、更新 Compose `.env` 和 `docker compose up -d app-server`，未在服务器构建。
- 文档：同步更新 `README.md` 与 `server/docs/api.md`，说明 PDF 支持范围、CLI 限制和 Office 文件暂不声明为原生模态。
- 验证通过：`cd server && go test ./...`；线上 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`，当前运行镜像为 `oauth-api-service-server:20260511T113835-dc1649c0-pdf-attachments`；`opencode models oauth-api-service` 正常列出 `gpt-5.4/gpt-5.5`。
- 验证通过：默认线上 OpenCode provider 附加 PDF 回归返回 PDF 内固定文本 `PDF-CHECK-7429`；附加 txt 回归返回 `TXT-CHECK-1842`。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，未执行 volume prune；根分区从清理前 `39%` 回到 `37%`，当前运行容器和数据库 volume 保持不变。
- 下一步：如需支持 `doc` / `docx` / `xls` / `xlsx`，需要先增加服务端转换链路并确认 OpenCode 客户端如何发送这类附件；本轮不通过配置伪造原生 Office 模态。

## 2026-05-11 OpenCode 默认 provider 图片能力修复
- 完成：排查当前本机 OpenCode 配置后确认默认 `oauth-api-service/gpt-5.5`、`oauth-api-service/gpt-5.4` 未声明 `modalities.input=["text","image"]`，OpenCode 因此在客户端阶段判定模型不支持图片并拒绝读取附件；服务端图片转发能力不是本次根因。
- 完成：已备份 `~/.config/opencode/opencode.json` 到 `~/.config/opencode/opencode.json.bak-20260511112356-before-image-modalities`，并给默认线上 provider 的两个模型补齐 text/image 输入与 text 输出能力声明；未修改 baseURL、API key、reasoning variants 或服务端代码。
- 验证通过：`opencode run --pure -m oauth-api-service/gpt-5.5 --variant low --file /tmp/opencode-vision-smoke.png --format json '只回复：image-ok'` 返回 `image-ok`；红色 1x1 PNG 视觉回归返回 `red`，确认图片附件已进入模型链路。
- 下一步：暂无。
- 阻塞/风险：`api-ndev-me` provider 当前仍未声明图片能力，因其不是本仓库默认服务入口且未确认该上游是否支持图片，本轮不擅自修改。

## 2026-05-11 API 凭据暗色 hover 修复
- 完成：API 凭据表格选中行 hover 从浅色硬编码改为主题变量，暗夜模式下选中行 hover 保持深色背景和可读文字；表头、普通行 hover 和选中行背景继续复用后台主题变量。
- 完成：`style:l1` 增加暗夜主题下“选中行 + hover”的背景亮度、文字对比和盒模型断言，覆盖截图中浅色残留问题。
- 下一步：暂无。
- 阻塞/风险：本轮只修复后台表格 hover 样式，不改变 API 凭据业务字段、保存或展示映射链路。

## 2026-05-11 凭据 Token 细分限额
- 补充：修复 API 凭据限额弹窗暗色主题下新限额区块仍使用浅色背景的问题；暗色覆盖规则新增 `bg-[#f7fbf8]` 与 `border-[#e4ece6]`，项目级 `AGENTS.md` 已写入后台前端必须同时支持浅色 / 暗色主题及目标区域回归要求。
- 补充验证：`style:l1` 已新增暗色模式下打开 API 凭据新建弹窗的回归，覆盖总 Token 与细分 Token 两个限额区块、8 个限制输入框、所有日 / 周总量 / 输入 / 非缓存输入 / 输出字段、背景亮度、文字对比和盒模型。
- 完成：API 凭据新增日 / 周输入 Token、输出 Token、非缓存输入 Token 限额字段，Ent schema 与 Atlas migration `20260511024741` 已生成并应用到当前开发库；旧凭据默认 0 表示不限，不改变既有总 Token 日 / 周限额语义。
- 完成：转发前 quota 检查改为同一日 / 周窗口内同时判断总量、输入、输出和非缓存输入；非缓存输入统一按 `input_tokens - cached_tokens` 且下限为 0，不伪造缺失缓存值。
- 完成：后台 API 凭据创建 / 编辑弹窗可配置细分限额，列表同步展示已设置的总量、输入、输出和非缓存输入日 / 周额度；JSON-RPC 文档和前端说明已同步。
- 下一步：如需按模型级 policy 继续拆输入 / 输出 / 非缓存输入，需要单独扩展 `gateway_policies`，本轮保持模型策略仍按总 Token，避免两套限额同时膨胀。
- 阻塞/风险：限额仍以已落库 usage 为真源，单次请求可能在本次成功后使窗口越过额度，下一次请求开始拦截；历史 usage 的缓存缺值不会被回填。

## 2026-05-11 统计凭据备注回显
- 完成：后端 `usage_list` 和 `usage_session_summaries` 按当前 `gateway_api_keys.name` 回补 `api_key_name`，CSV/JSON usage 导出同步增加 API 凭据备注字段；凭据已删除或缺失时不伪造备注。
- 完成：后台最近调用、调用明细、每日模型详情和会话聚合表格统一展示“备注 + 前缀”，避免只看凭据前缀难以判断使用方；`style:l1` mock 与断言同步覆盖备注回显。
- 下一步：如需按历史调用时的备注保留快照，需要单独评估在 usage log 中新增快照字段及迁移口径；本轮保持当前 key 表为备注真源。
- 阻塞/风险：30 天趋势、Token 构成、模型 / 接口分布是跨凭据聚合图表，没有单个凭据备注可展示；凭据删除后的旧 usage 只能保留前缀。

## 2026-05-11 Compose Nginx 迁移入口
- 完成：新增可选 `server/deploy/compose/prod/compose.nginx.yml`，使用官方 `nginx:1.27-alpine` 镜像和配置挂载，不构建自定义 Nginx 镜像；默认 `compose.yml` 不启动 Nginx，避免和当前宿主机 Nginx 抢占 `80/443`。
- 完成：新增 `server/deploy/compose/prod/nginx/` 配置目录，收口 HTTP-01 challenge、主域 HTTPS 反代、旧域名跳转样本、proxy header、`proxy_read_timeout 700s` / `proxy_send_timeout 700s`，反代目标为 Compose 内部 `app-server:8400`。
- 完成：同步更新 `.env.example`、Compose 部署 README 和部署总览，说明证书目录、ACME webroot、非标准端口预验证、切换宿主机 Nginx 与回滚步骤。
- 验证通过：`docker compose -f compose.yml -f compose.nginx.yml --env-file .env.example config` 可解析；使用临时自签证书和 `nginx:1.27-alpine` 执行 `nginx -t` 通过；`git diff --check` 通过。
- 下一步：需要真正切线上入口时，先把当前宿主机证书复制到 `/data/openai-oauth-api-service/nginx/certs`，用 `8080/8443` 验证容器 Nginx 后，再停宿主机 Nginx 切 `80/443`。
- 阻塞/风险：本轮只增加迁移能力，不切当前线上入口；旧域名 HTTPS redirect server block 依赖旧域名证书，新机器如果不迁旧证书，应先删除或注释对应 server block。

## 2026-05-11 Codex 网关 10 秒 502 修复
- 完成：排查线上 usage 后确认近 2 小时 502 均为 `codex_backend_upstream_failed`，耗时集中在 `10000-10006ms`，与服务端 `server.http.timeout=10s` 完全吻合；外层 HTTP context 先取消，导致 Codex backend / CLI 600 秒上游超时配置没有生效。
- 完成：将 dev/prod `server.http.timeout` 调整为 `650s`，覆盖 `CODEX_BACKEND_TIMEOUT_SECONDS=600` 与 `CODEX_CLI_TIMEOUT_SECONDS=600` 的正常等待窗口；`server.grpc.timeout` 保持 `10s`，避免放大无关内部接口等待时间。
- 完成：本地构建镜像 `oauth-api-service-server:20260511T102030-469f082c-local` 并上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260511T102030-469f082c-local/`；远端只执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 验证通过：远端当前运行镜像为 `oauth-api-service-server:20260511T102030-469f082c-local`，容器内 `/app/configs/config.yaml` 显示 HTTP timeout `650s`、gRPC timeout `10s`；`/healthz` 返回 `ok`，`/readyz` 返回 `ready`，公网 `https://oauth-api.saurick.me/admin-login` 返回 `HTTP 200`。
- 验证通过：线上直连 `/v1/chat/completions` 使用现有下游 key 返回 `HTTP 200`、正文 `OK`，耗时约 `1.71s`；部署后 usage 新增记录为 `chat.completions 200`，截至复查没有新增 502。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，未执行 volume prune；根分区从清理前 `38%` 回到 `36%`，当前运行容器和数据库 volume 保持不变。
- 阻塞/风险：Cloudflare / Nginx 仍可能对极长非流式请求有更外层超时限制；本轮修复的是当前已确认的服务进程 10 秒外层超时。

## 2026-05-11 首页最近调用字段线上部署
- 完成：按低配服务器发布边界在本地构建镜像 `oauth-api-service-server:20260511T002238-469f082c-local`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260511T002238-469f082c-local/`；远端只执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 完成：远端当前运行镜像为 `oauth-api-service-server:20260511T002238-469f082c-local`；`/healthz` 返回 `ok`，`/readyz` 返回 `ready`，公网 `https://oauth-api.saurick.me/admin-login` 返回 `HTTP 200`。
- 验证通过：Browser 插件通过线上 `https://oauth-api.saurick.me/oauth/callback` 注入本轮登录 token 后进入 `/admin-dashboard`，确认「最近调用」已包含请求、Session、缓存输入 / 推理输出、字节、Backend 优先和强制 CLI 字段；未发现新的页面错误，浏览器日志仅保留既有 React Router v7 future flag warning。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`，未执行 volume prune；根分区从清理前 `38%` 回到 `36%`，当前运行容器和数据库 volume 保持不变。
- 阻塞/风险：本轮没有数据库 schema 变更，未执行 migration；远端 release 目录保留本轮上传镜像包，便于短期追溯，后续磁盘紧张时可单独清理 release tar 包。

## 2026-05-11 首页最近调用字段对齐
- 完成：将 `/admin-dashboard`「最近调用」表格字段对齐 `/admin-usage`「调用明细」口径，补齐请求 ID、Session、请求方法、上游模式、缓存输入 / 推理输出、请求 / 响应字节和同款表头说明，避免首页样本与明细页字段继续漂移。
- 完成：`style:l1` 增加首页最近调用字段断言，要求请求 ID、Session、缓存输入 / 推理输出、字节和表头 tooltip 与明细页保持一致。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm lint`、`cd web && pnpm style:l1`（22 个场景）均通过；Browser 插件通过真实本地后端登录 `http://127.0.0.1:5176/admin-dashboard`，确认首页最近调用 DOM 已包含新增明细字段，控制台仅有既有 React Router v7 future flag warning。
- 阻塞/风险：本轮只调整首页最近调用展示和前端回归断言，未修改后端 usage DTO、数据库字段、导出和 `/admin-usage` 明细页真源。

## 2026-05-11 线上证书自动续签口径收口
- 完成：线上 `8.218.4.199` 的证书续签入口已从旧 `openai.saurick.space` 专用脚本收口为 `/usr/local/sbin/renew-openai-oauth-api-certs.sh`，root crontab 现只保留 `23 3 * * * /usr/local/sbin/renew-openai-oauth-api-certs.sh`。
- 完成：`acme.sh` 自动续签列表已移除 `openai.saurick.space` 与 `oauth-api.saurick.space`，当前只保留主域 `oauth-api.saurick.me`；旧脚本、旧 acme 状态目录和旧 Cloudflare env 已移到 `/root/ops-backups/acme-renewal-20260510T160712Z/` 下归档，不再位于活跃路径。
- 验证通过：手动执行新脚本后，日志只扫描 `oauth-api.saurick.me`，按 Let's Encrypt ARI 跳过未到期证书，下一续签时间为 `2026-07-09T14:08:47Z`；`nginx -t` 与 reload 成功，源站证书仍为 Let's Encrypt `E7`，有效期到 `2026-08-08 13:59:44 GMT`。
- 验证通过：公网 `https://oauth-api.saurick.me/admin-login` 返回 `HTTP 200`，源站直连 `8.218.4.199:443` 使用 SNI `oauth-api.saurick.me` 的证书主机名校验通过。
- 阻塞/风险：本轮只收口续签自动化和 acme 活跃状态；Nginx 中旧 `saurick.space` 跳转 server block 仍保留但不再自动续签，后续若确认旧域名彻底废弃，可单独清理对应 Nginx 配置与旧证书文件。

## 2026-05-10 上游模式线上部署与 OpenCode 验证
- 完成：按低配服务器发布边界在本地构建镜像 `oauth-api-service-server:20260510T152408-469f082c-local3`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260510T152408-469f082c-local3/`；远端只执行 `docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- 完成：远端 Compose 当前运行镜像为 `oauth-api-service-server:20260510T152408-469f082c-local3`；`/readyz` 返回 `ready`，`https://oauth-api.saurick.me/admin-upstream` 返回新版前端资源，Cloudflare 代理解析正常，本轮未修改 Cloudflare DNS 记录。
- 验证通过：本地 `opencode models oauth-api-service` 返回 `gpt-5.4/gpt-5.5`；`opencode run -m oauth-api-service/gpt-5.5 --variant high '只回复 pong，不要解释。'` 返回 `pong`，截图中的 `API key 无效` 未复现。
- Token 对比：生产 usage 记录显示 Backend 单轮 `29 in / 17 out / 46 total / 1.69s`，Backend 带历史 `50 in / 17 out / 67 total / 1.25s`；CLI 单轮 `13765 in / 5 out / 13770 total / 5.40s`；CLI 带历史重试成功为 `13780 in / 17 out / 13797 total / 5.54s` 和 `13775 in / 19 out / 13794 total / 5.99s`。另有一次 CLI 带历史返回 `502 codex_cli_upstream_failed / 10.95s`，最终已切回 `codex_backend`。
- 清理：部署后执行 `docker image prune -a -f` 与 `docker builder prune -f`；根分区从 `20G used / 52%` 降到 `14G used / 36%`，未执行 volume prune，所有运行中容器保持 up。
- 阻塞/风险：CLI 路径仍会显著放大输入 token，并存在偶发 `codex_cli_upstream_failed`；高频 OpenCode 默认应保持 `codex_backend`，`codex_cli` 只作为兼容兜底或临时排障模式。

## 2026-05-10 Codex 上游模式开关与统计
- 完成：新增 `gateway_settings` 运行时设置表，后台 `api.gateway_upstream_get` / `api.gateway_upstream_set` 可在 `codex_backend` 与 `codex_cli` 间切换；默认仍为 `codex_backend`，未保存后台设置时才读取 `CODEX_UPSTREAM_MODE` 作为启动默认值。
- 完成：usage log 新增 `upstream_configured_mode`、`upstream_mode`、`upstream_fallback`、`upstream_error_type`；后台 summary、每日模型、凭据统计、会话聚合、请求明细和导出均补充 Backend / CLI / fallback 统计或字段，并支持 `upstream_mode` 筛选。
- 完成：后台新增独立 `/admin-upstream`「上游模式」菜单，仅负责 Codex 上游开关读写；「用量日志」页保留上游模式筛选、统计列和导出字段；业务看板补充上游分布卡片、趋势 tooltip 上游分布和最近调用上游列。
- 验证通过：`cd server && make data` 生成 migration 与 Ent 代码；`cd server && make print_db_url && make migrate_apply` 已应用本地开发库到 `20260510141225`；`cd server && go test ./...`、`cd web && pnpm test`、`cd web && pnpm style:l1`（22 个场景）、`cd web && pnpm build` 均通过。
- 本地真实请求对比：经后台开关分别调用 `/v1/chat/completions`，Backend 单轮 `input=25/output=17/total=42/2.45s`，Backend 带历史消息 `input=40/output=5/total=45/1.72s`；CLI 单轮 `input=23593/output=5/total=23598/9.35s`，CLI 带历史消息 `input=23604/output=5/total=23609/7.31s`。四条 usage 均正确记录 configured/actual/fallback 字段，测试后已切回 `codex_backend`。
- 前端回归：Browser 插件在 `http://127.0.0.1:5176/admin-usage` 验证登录、用量统计桌面与 390px 移动视口页面非空、无框架错误覆盖；`/admin-upstream` 验证独立菜单只展示模式开关，开关切到 CLI 再切回 Backend；控制台仅有既有 React Router v7 future flag warning。
- 阻塞/风险：本轮只验证本地开发库与本机 Codex 登录态；线上服务器仍需确认能访问 ChatGPT Codex backend，否则即使后台切换为 backend 也会 fallback 或失败。

## 2026-05-10 线上域名切换到 oauth-api.saurick.me
- 完成：新购 `saurick.me` 已在 Cloudflare 生效，`oauth-api.saurick.me` 公网解析到 Cloudflare 代理地址；域名状态无 `serverHold`。
- 完成：为 `oauth-api.saurick.me` 配置 Nginx HTTP-01 challenge，通过 Let's Encrypt 签发并安装正式证书，有效期到 `2026-08-08 13:59:44 GMT`；Nginx 443 主入口已切到 `oauth-api.saurick.me` 并反代 `127.0.0.1:8400`。
- 完成：旧源站入口 `oauth-api.saurick.space` 与 `openai.saurick.space` 已在 Nginx 配置为 308 跳转到 `https://oauth-api.saurick.me$request_uri`；由于 `saurick.space` 仍是 `serverHold`，这些旧域名公网解析本身仍不可用。
- 完成：线上 Compose `.env` 的 `OAUTH_API_OAUTH_ALLOWED_FRONTEND_ORIGINS` 已改为 `https://oauth-api.saurick.me` 并重建 `app-server`；本机 OpenCode `oauth-api-service` provider 已改为 `https://oauth-api.saurick.me/v1`，配置备份为 `~/.config/opencode/opencode.json.bak-20260510225902-saurick-me`。
- 文档：同步更新 `README.md`、`server/docs/api.md`、`server/docs/config.md`、`server/deploy/compose/prod/README.md` 中的当前个人部署域名。
- 验证通过：`https://oauth-api.saurick.me/admin-login` 返回 `HTTP 200` 且标题为 `Saurick API Console`；`http://oauth-api.saurick.me/admin-login` 返回 301 到 HTTPS；使用本地下游 key 请求 `https://oauth-api.saurick.me/v1/models` 返回模型列表；`opencode models oauth-api-service` 返回 `gpt-5.4/gpt-5.5`；Google Safe Browsing status 对 `oauth-api.saurick.me` 返回 `6` / no available data，未命中 unsafe。
- 阻塞/风险：本轮只切换域名和运行配置，未重建应用镜像。旧 `saurick.space` 仍被注册局 hold，无法作为公网跳转入口依赖；后续如需释放旧域名或彻底删除旧证书/配置，需要等 `serverHold` 解除或确认不再使用。

## 2026-05-10 线上域名切换到 oauth-api.saurick.space
- 完成：前端公网可见标题、登录页品牌、后台壳子和 HTML meta 从显眼 `OpenAI OAuth API Service` 收口为 `Saurick API Console` / `API 管理后台`，降低被识别为 OpenAI 官方登录页的风险；生产示例域名同步改为 `https://oauth-api.saurick.space/v1`。
- 完成：补齐既有后端改动缺失的 Ent 生成物与 Atlas migration，新增 `gateway_settings` 表和 upstream mode 统计字段迁移；线上通过 Atlas 容器执行到 `20260510141225`。
- 完成：本地构建镜像 `oauth-api-service-server:20260510T221315-ffa4cbb6-local`，上传到 `8.218.4.199` 后 `docker load` 并切换 Compose `APP_IMAGE`；当前 `app-server` 已运行该镜像，`/healthz` 返回 `ok`，`/readyz` 返回 `ready`。
- 完成：Cloudflare DNS 中已设置 `A oauth-api.saurick.space -> 8.218.4.199` 且 `proxied=true`；Nginx 已新增 `oauth-api.saurick.space` 反代到 `127.0.0.1:8400`，旧 `openai.saurick.space` 在源站 Nginx 返回 308 到新域名。本机 OpenCode 线上 provider 已改为 `https://oauth-api.saurick.space/v1`。
- 验证通过：`cd web && pnpm lint && pnpm css && pnpm test && pnpm build && pnpm style:l1`（`style:l1` 第一次 `admin-keys-mobile` 行选择恢复态偶发失败，复跑 21 场景通过）；`cd server && go test ./internal/server ./internal/biz ./internal/data`；`git diff --check`；源站直连 `https://oauth-api.saurick.space/admin-login --resolve ... -k` 返回新标题，`/v1/models` 使用下游 key 返回模型列表。
- 运维清理：发布前记录 `/` 使用率 51%、Docker build cache 约 971.8MB；发布后删除本轮上传 tar 包，执行 `docker builder prune -f` 和 dangling `docker image prune -f`，`/` 使用率 50%、build cache 降至 186.4MB。因公网 DNS 尚不可用，保留未运行旧镜像作为回滚余地，未执行 `docker image prune -a -f`。
- 阻塞/风险：`whois saurick.space` 当前仍显示 `serverHold`，公共 DNS 对 `saurick.space` / `oauth-api.saurick.space` 返回 `NXDOMAIN`；Cloudflare 记录已存在但注册局未解析，Let's Encrypt DNS-01 也因此无法签发公网证书。当前 Nginx 新域名源站证书为临时自签，仅用于源站就绪；必须先在阿里云/注册商解除 `serverHold`，再重新签发正式证书并做公网 HTTPS 回归。

## 2026-05-10 OpenCode direct Codex backend adapter
- 完成：新增 `CODEX_UPSTREAM_MODE`，默认切为 `codex_backend`，需要强制旧路径时可设为 `codex_cli`；direct backend 模式由 app-server 进程直接请求 `https://chatgpt.com/backend-api/codex/responses`，避免每次请求启动 `codex exec` 子进程和注入 Codex CLI agent 上下文。
- 完成：默认 backend 模式增加 Codex CLI fallback，backend 请求失败时自动尝试 `codex exec`；显式 `CODEX_UPSTREAM_MODE=codex_cli` 时只走 CLI。
- 完成：direct backend 模式读取 Codex `auth.json` 的 access token / account_id，请求时带 `Authorization: Bearer ...` 与 `ChatGPT-Account-Id`；access token 过期或上游返回 401 时使用 refresh token 调 `https://auth.openai.com/oauth/token` 刷新，并写回 `auth.json`。
- 完成：OpenAI-compatible `/v1/chat/completions` 与 `/v1/responses` 会被转换为 Codex backend Responses 请求；支持 `reasoning_effort` 和 data URL 图片输入，Responses SSE 的 `response.output_text.delta` / `response.completed.usage` 会回填为当前兼容响应和 usage 记录。
- 修复：Codex backend 要求 `instructions` 非空；direct backend adapter 现在优先使用请求里的 `system` / `developer` 消息，缺失时补最小默认 instructions，避免单轮无 system 请求先 400 再落到 CLI。
- 文档：同步更新 `README.md`、`server/docs/api.md`、`server/docs/config.md`、Compose `.env.example`、`compose.yml` 与部署 README，说明 `codex_cli` / `codex_backend` 的适用边界、配置项和回退方式。
- 验证通过：`cd server && go test ./internal/server ./internal/biz ./internal/data`、`git diff --check -- ...`、`cd server && make dev_build`；新增 mock backend 测试覆盖默认 backend 模式、backend 失败 CLI fallback、请求头、Responses 请求体、SSE 文本和 usage 解析，以及 access token 过期后的 refresh 与 `auth.json` 写回。
- 本地 token 对比：同一 `/v1/chat/completions`、`gpt-5.5`、`reasoning_effort=low`，backend 单轮 `prompt_tokens=25/total=42/2.82s`，backend 带历史续会话 `prompt_tokens=40/total=57/1.90s`；强制 CLI 单轮 `prompt_tokens=23593/total=23598/5.97s`，强制 CLI 带历史续会话 `prompt_tokens=23604/total=23609/8.01s`。
- 阻塞/风险：本轮尚未部署线上镜像；若 ChatGPT Codex backend 协议变化，默认会先 fallback 到 CLI，仍可显式 `CODEX_UPSTREAM_MODE=codex_cli` 固定旧路径。本机已用 detached `screen` 启动默认 backend 模式 dev 服务，session 名为 `oauth-api-dev`。

## 2026-05-10 每日模型汇总详情
- 完成：用量日志默认视图从单纯按天汇总调整为「每日模型汇总」；后端 `api.usage_buckets` 新增 `group_by=day_model`，按日期 + 模型聚合请求数、输入 / 输出 / 缓存 / reasoning tokens、费用估算和错误率。
- 完成：每日模型汇总的「详情」按钮新增弹窗下钻，按当天 + 模型复用 `api.usage_list` 拉取请求级明细，展示时间、输入 / 输出 / 缓存 / reasoning tokens、总 tokens、费用和成功状态，并提供上一页 / 下一页分页。
- 阻塞/风险：本地已构建镜像 `oauth-api-service-server:20260510T003654-ffa4cbb6-usage-detail` 并上传到 `8.218.4.199:/data/openai-oauth-api-service/`；远端执行 `docker load -i oauth-api-service-server-20260510T003654-ffa4cbb6-usage-detail.tar.gz` 后服务器用户态服务无响应，SSH 卡在 banner、`/healthz` 超时，已断开本地 SSH 部署会话。线上是否完成镜像导入未知，Compose `.env` 更新与 `app-server` 重启尚未确认。
- 阻塞/风险：2026-05-10 继续部署时复查，ICMP、22 和 8400 TCP 端口可达，但 SSH 60 秒仍卡在 banner exchange，`/healthz` 继续超时；本机未发现可用 `aliyun` CLI 配置，无法从当前终端恢复服务器。继续发布前需要先通过云控制台重启或恢复 8.218.4.199，再检查 Docker 镜像导入和 Compose 状态。
- 完成：服务器扩容恢复后，确认新镜像已加载，已将 `/data/openai-oauth-api-service/compose/.env` 的 `APP_IMAGE` 切换到 `oauth-api-service-server:20260510T003654-ffa4cbb6-usage-detail` 并重建 `app-server`；`/healthz` 返回 200，线上 `api.usage_buckets group_by=day_model` 与按日期 + 模型调用 `api.usage_list` 验证通过，Playwright 线上回归通过登录、每日模型表格、详情弹窗打开和下一页分页。
- 阻塞/风险：2026-05-10 继续部署最新工作区代码时，本地构建镜像 `oauth-api-service-server:20260510T122824-ffa4cbb6-latest` 成功并上传到 `8.218.4.199:/data/openai-oauth-api-service/`；远端 `docker load` 成功，Compose `.env` 已切到该镜像，`docker compose up -d app-server` 已打印 `app-server Started`，但随后服务器用户态再次无响应，SSH 卡在 banner、`/healthz` 超时。当前需通过云控制台重启或恢复 8.218.4.199 后，再确认新镜像是否正常运行。
- 完成：补齐「调用明细」行详情弹窗的上一页 / 下一页导航；当前页内按相邻请求切换，到页边界时按当前筛选条件拉取上一页 / 下一页 usage 数据再切换，避免只有「每日模型」详情有分页而「调用明细」详情无分页的交互不一致。已更新 `style:l1` 断言覆盖调用详情弹窗分页按钮和下一条切换。
- 完成：将「调用明细」行详情明确为单次请求排障面板，并新增 `Session ID` 字段；空值显示“未传入”，用于解释为什么某些请求不会出现在「会话聚合」。已更新 `style:l1` 断言覆盖说明文案和 Session ID 展示。
- 完成：根据最新交互口径移除「调用明细」表格的详情按钮和单条详情弹窗；请求 ID、Session、缓存 / Reasoning Token、请求 / 响应字节等排障字段直接展示在明细表格中。已更新 `style:l1` 断言调用明细不再出现详情按钮，并覆盖新增表格字段。
- 完成：优化调用明细表格的组合数字展示，Token 列改为“总 / 输入 / 输出”带标签，缓存 / Reasoning 与请求 / 响应字节也增加中文标签，避免只显示 `23,570 / 5` 这类难以理解的裸数字。
- 完成：给调用明细中容易混淆的统计表头增加问号 hover/focus 说明，覆盖 Token、缓存输入 / 推理输出、费用估算、耗时和字节；同时把缓存 / Reasoning 文案统一为缓存输入 / 推理输出。
- 阻塞/风险：每日模型详情仍不展示请求 / 响应正文，费用字段沿用现有模型价格估算；若模型没有价格配置，主表和详情继续显示“未配置价格”。

## 2026-05-10 00:30
- 完成：补充 `AGENTS.md` 的多项目低配 Docker 宿主机发布后清理约束，明确发布完成、健康检查和必要回归通过后，只清理未被任何容器使用的旧镜像与构建缓存，优先使用 `docker image prune -a -f` 与 `docker builder prune -f`，并禁止清理 volume、数据库目录、compose `.env`、上传目录或运行中容器依赖镜像；同步给 `legacy-python-mvp/AGENTS.md` 加入轻量版同类约束。更新前因 `progress.md` 超过归档阈值，已归档旧流水。
- 下一步：如后续继续完善发布脚本，可把该约束落为脚本级 post-deploy cleanup，并在执行前后输出磁盘与容器状态。
- 阻塞/风险：本轮只更新协作与部署约束文档，未修改运行代码、Compose 配置或线上服务；旧镜像清理仍需在发布脚本中显式实现。

## 2026-05-10 API 凭据表格操作列精简
- 完成：`/admin-keys` API 凭据表删除行内“操作”列，编辑、启用 / 禁用、删除继续收口到表格上方“当前操作”区域；状态 badge 统一加 `whitespace-nowrap`，避免窄列下“启用 / 禁用”拆行。
- 验证补充：`style:l1` 的 API 凭据页断言新增“无行内操作列”“状态列无按钮”“状态 badge computed white-space 为 nowrap”，覆盖桌面和移动端 mock 数据。
- 阻塞/风险：本轮只调整 API 凭据表展示与前端回归断言，不修改后端 key 创建、编辑、启停、删除接口和数据库真源；行选择、双击编辑和顶部“当前操作”仍沿用原有交互。

## 2026-05-10 本地 OpenCode 转发验证
- 完成：本机 OpenCode 配置新增 `oauth-api-service-local` provider，指向 `http://localhost:8400/v1`，复用本地开发库中启用的 `ogw_` 下游 key；原线上 `oauth-api-service` provider 保持不变，配置备份为 `~/.config/opencode/opencode.json.bak-20260510131820-local-provider`。
- 修复：后端 Codex CLI JSON 解析兼容当前 `item.completed` / `turn.completed` 事件格式，避免把 usage 事件 JSON 误当成 OpenAI 兼容响应正文；旧的 `event_msg` / `response_item` 格式继续保留。
- 验证通过：`cd server && go test ./internal/server`；本地 `/v1/chat/completions` 返回 `HTTP 200` 且正文 `OK`；`opencode models oauth-api-service-local` 返回 `gpt-5.4/gpt-5.5`；`opencode run --pure -m oauth-api-service-local/gpt-5.5 --variant high --format json "只回复 OK"` 返回 `OK`。
- 阻塞/风险：本地后端需要带 `server/.env` 中的 `DB_URL` 作为 `POSTGRES_DSN` 启动；直接运行二进制只读默认 `config.yaml` 会连到占位库并认证失败。当前本地测试服务以前台会话方式运行在 `:8400`。

## 2026-05-10 OpenCode reasoning effort 支持
- 完成：本地 `oauth-api-service-local` provider 的 `gpt-5.5/gpt-5.4` 已声明 `low`、`medium`、`high`、`xhigh` variants；OpenCode UI 可显示并切换对应 effort。
- 完成：后端 OpenAI-compatible 请求体新增 `reasoning_effort` 支持，兼容 `reasoningEffort` 和 `reasoning.effort` 输入，只允许 `low/medium/high/xhigh`，并映射为 Codex CLI `model_reasoning_effort`。
- 文档：同步更新 `README.md` 与 `server/docs/api.md`，说明兼容入口的 `reasoning_effort` 取值和上游映射。
- 验证通过：`cd server && go test ./internal/server`、`git diff --check -- README.md server/docs/api.md server/internal/server/openai_gateway_handler.go server/internal/server/openai_gateway_handler_test.go progress.md`；本地重建 `server-dev` 后，`opencode run --pure -m oauth-api-service-local/gpt-5.5 --variant low/medium/high/xhigh --format json "只回复 OK"` 均返回 `OK`，usage 最近 4 条均为 `chat.completions 200 success=true`；非法 `reasoning_effort=extreme` 返回 `HTTP 400 gateway_reasoning_effort_invalid`。
- 阻塞/风险：线上 `oauth-api-service` provider 的 variants 已先从本机 OpenCode 配置撤回，避免默认线上继续因 effort 下拉触发失败；配置备份为 `~/.config/opencode/opencode.json.bak-20260510142213-disable-online-effort`。线上当前不带 effort 直连也返回 `HTTP 502 codex_cli_upstream_failed`，根因是服务器容器内 Codex CLI 调用 `https://chatgpt.com/backend-api/codex/responses` 被 Cloudflare 拦截，不是本地 provider effort 解析本身。线上要恢复后才能重新开放线上 provider effort。

## 2026-05-10 OpenCode 图片输入排查
- 发现：本地 OpenCode 自定义 provider 的模型未声明 `modalities.input=["text","image"]`，OpenCode 会认为模型不支持图片输入；服务端 `contentTextValue` 也只抽取文本，未把 `image_url` / `input_image` 转给 Codex CLI，因此即使客户端发图也会在网关层丢失。
- 修复：后端 OpenAI-compatible 请求体支持 data URL 形式图片输入，单次最多 4 张、单张最大 16 MiB，临时落盘后通过 `codex exec --image` 附加到本次请求，请求结束清理临时文件；本机 `oauth-api-service-local` 的 `gpt-5.5/gpt-5.4` 已补 `modalities`，配置备份为 `~/.config/opencode/opencode.json.bak-20260510193130-before-modalities`。
- 文档：同步更新 `README.md` 与 `server/docs/api.md`，说明图片输入支持范围和 data URL 限制。
- 验证通过：`cd server && go test ./internal/server`，新增 fake Codex CLI 测试确认图片文件会作为 `--image` 传入且请求结束后清理；`opencode debug config` 确认本地 provider 两个模型均声明 text/image 输入。当前运行中的 `:8400` 本地服务仍是旧二进制，需要重建 / 重启后图片链路才会实际生效。
- 阻塞/风险：当前只支持请求体内 data URL 图片，不支持让服务端主动拉取远程 `http(s)` 图片 URL，避免把网关变成任意 URL 抓取入口；线上 provider 尚未部署本轮改动。

## 2026-05-10 OpenCode 三 provider 部署回归
- 完成：本地构建镜像 `oauth-api-service-server:20260510T151520-ffa4cbb6-local`，上传到 `8.218.4.199` 后通过 `docker load` 切换 `/data/openai-oauth-api-service/compose/.env` 的 `APP_IMAGE` 并重建 `app-server`；远端 `/healthz` 返回 `ok`，`/readyz` 返回 `ready`，当前运行镜像为 `oauth-api-service-server:20260510T151520-ffa4cbb6-local`。
- 完成：本机 OpenCode 三个 provider `oauth-api-service`、`oauth-api-service-local`、`api-ndev-me` 均已配置 `gpt-5.5/gpt-5.4` 的 `low/medium/high/xhigh` variants；`opencode models` 三组均能列出目标模型。
- 验证通过：`/v1/models` 三组 URL 均返回 `HTTP 200`；`oauth-api-service-local` 和 `api-ndev-me` 直连 `reasoning_effort=high` 返回 `OK`，对应 `opencode run --pure -m <provider>/gpt-5.5 --variant high --format json "只回复 OK"` 也返回 `OK`。
- 阻塞/风险：线上 `oauth-api-service` 域名和直连 `8.218.4.199:8400` 的真实 chat 仍返回 `HTTP 502`；直连后端错误码为 `codex_cli_upstream_failed`，错误内容显示服务器容器内 Codex CLI 访问 `https://chatgpt.com/backend-api/codex/responses` 被 Cloudflare 拦截。不带 `reasoning_effort` 也同样 502，因此当前剩余问题是线上服务器到 ChatGPT Codex 上游的网络 / 风控链路，不是本轮 `reasoning_effort` 参数解析。因线上 chat 回归未通过，本轮只删除上传 tar 包，未执行远端旧镜像 prune，以保留回滚余地。

## 2026-05-10 线上 Codex 上游故障排查
- 发现：`8.218.4.199` 宿主机直接访问 `https://chatgpt.com/` 返回 Cloudflare `HTTP 403`，`/v1/chat/completions` 直连后端错误也指向 `https://chatgpt.com/backend-api/codex/responses` 被 Cloudflare 阻断；这是线上 provider 502 的主因。
- 修复：`runCodexCLI` 改为继承请求 `context` 创建 Codex CLI 子进程，避免客户端断开、OpenCode 超时或重试后，服务端子进程仍继续跑到 `CODEX_CLI_TIMEOUT_SECONDS=600` 才退出，放大低配服务器资源占用。
- 验证通过：新增假 Codex CLI 取消测试；`cd server && go test ./internal/server ./internal/biz ./internal/data` 通过。本地已构建待部署镜像 `oauth-api-service-server:20260510T153810-ffa4cbb6-local` / `oauth-api-service-server:deploy-cancel-20260510`。
- 阻塞/风险：排查过程中线上机器再次进入用户态无响应，`healthz`、`readyz` 和 SSH banner 均持续超时；当前无法上传并部署取消修复。若机器不自行恢复，需要先通过云控制台重启或恢复 `8.218.4.199`，再部署新镜像并配置可用的上游出口 / 代理。
- 完成：服务器续费恢复后，已先部署取消修复镜像 `oauth-api-service-server:20260510T153810-ffa4cbb6-local`，随后发现 Codex CLI 超时后容器内出现 `[codex]` 僵尸进程；进一步将运行镜像加入 `tini` 作为 PID 1，并部署 `oauth-api-service-server:20260510T163723-ffa4cbb6-local`。
- 验证通过：远端当前 `app-server` 命令为 `/sbin/tini -- /app/server -conf /app/configs`，`/healthz` 返回 `ok`，`/readyz` 返回 `ready`；线上 `/v1/models` 返回 `HTTP 200`；OpenCode 线上 provider `opencode models oauth-api-service` 能列出 `gpt-5.4/gpt-5.5`。短超时 chat 与 OpenCode 超时后，容器内无残留 `codex/node` 进程，服务健康保持正常。
- 阻塞/风险：`8.218.4.199` 宿主机访问 `https://chatgpt.com/` 仍返回 Cloudflare `HTTP 403`，线上 `oauth-api-service` 真实 chat 仍失败：直连 `8.218.4.199:8400` 返回 `HTTP 502 codex_cli_upstream_failed` / `codex cli upstream timed out after 10m0s`，`opencode run --pure -m oauth-api-service/gpt-5.5 --variant high --format json "只回复 OK"` 在 60 秒外层超时。当前线上服务已不再被失败请求拖住，但要让线上 provider 真正可用，仍需配置可访问 ChatGPT Codex backend 的服务器出口 / 代理。因真实 chat 未通过，本轮只删除上传 tar 包，未执行远端旧镜像 prune，以保留回滚余地。
- 完成：确认宿主机 `codex` 能通是因为交互 shell 从 `/root/.zshrc` / `/root/.bashrc` 注入了 mihomo 代理环境；app-server 容器此前没有代理环境，且宿主机 mihomo 只监听 `127.0.0.1:7890`，容器无法访问。
- 修复：备份远端 mihomo、root shell 和 compose 配置后，将 mihomo `allow-lan` 改为 `true`、`bind-address` 改为 app-server Docker bridge 网关 `172.19.0.1`，并同步把 root shell 与 app-server compose `.env` 的 `HTTP_PROXY` / `HTTPS_PROXY` / `WS_PROXY` / `WSS_PROXY` / `ALL_PROXY` 及小写变量设置为 `http://172.19.0.1:7890`，`NODE_USE_ENV_PROXY=1`；未启用 TUN，未切换代理节点。
- 文档：同步更新 `server/deploy/compose/prod/compose.yml`、`.env.example`、`server/docs/config.md` 和 `server/deploy/compose/prod/README.md`，明确 Codex CLI 可选代理环境和 mihomo / Clash 推荐接法。
- 验证通过：容器内同款 `codex exec --skip-git-repo-check --ephemeral --ignore-user-config --ignore-rules --json -s read-only -m gpt-5.5 -c model_reasoning_effort="high" -` 返回 `OK`；直连 `http://8.218.4.199:8400/v1/chat/completions` 与域名 `https://openai.saurick.space/v1/chat/completions` 带 `reasoning_effort=high` 均返回 `HTTP 200 content=OK`；本机 `opencode run --pure -m oauth-api-service/gpt-5.5 --variant high --format json "只回复 OK"` 返回 `OK`；远端 `/healthz`、`/readyz` 保持正常，容器内无残留 `codex/node` 进程。
- 优化：OpenAI-compatible usage 响应新增 `prompt_tokens_details.cached_tokens`、`completion_tokens_details.reasoning_tokens` 以及 Responses 对应 details 字段，避免客户端只看到总 input tokens 而误判缓存未命中；已部署镜像 `oauth-api-service-server:20260510T194246-ffa4cbb6-local`。
- 验证通过：`cd server && go test ./internal/server ./internal/biz ./internal/data`；线上重复请求返回 `HTTP 200`，usage 示例为 `prompt_tokens=13761`、`prompt_tokens_details.cached_tokens=13696`，另一次为 `cached_tokens=7552`；域名 chat 耗时约 `6.8s-10.3s`。本轮已删除上传 tar 包，未清理旧镜像以保留回滚余地。

## 2026-05-11 OpenCode API key 工具调用排查
- 发现：本机 SSH 到 `sauri@100.72.19.6` 超时；更正目标为 `sauri@192.168.0.44` 后可登录 Windows PowerShell 7，`cmd.exe` / `powershell.exe` / `pwsh.exe` 均在 PATH，但 `opencode` 不在该用户 PATH，全局 npm 仅安装 `@openai/codex@0.125.0` 和 `yarn`。
- 修复：OpenAI-compatible direct backend adapter 不再固定 `tools: []`，改为透传 `tools` / `tool_choice` / `parallel_tool_calls`，并转换 Chat Completions 与 Responses 的 `tool_calls`、`function_call`、`function_call_output` 历史；Codex backend 返回 function call 时会映射回 Chat Completions `tool_calls` 或 Responses `function_call`，让 OpenCode API key 模式能继续触发本机 shell / 文件 / 截图工具。
- 修复：Windows `sauri` 用户已通过 npm 全局安装 OpenCode，`opencode` 入口为 `C:\Users\sauri\AppData\Roaming\npm\opencode.ps1`，版本 `1.14.48`；`cmd.exe`、`pwsh.exe` 和 `powershell.exe` 基础 shell 探针均通过。
- 部署：本地构建 `oauth-api-service-server:20260511T150703-d530adb7-local`，上传到远端 `/data/openai-oauth-api-service/releases/`，远端仅执行 `docker load` 和 `docker compose up -d app-server`，未在服务器构建。
- 文档：同步更新 `README.md` 与 `server/docs/api.md`，说明工具调用只在 direct backend 模式支持，Codex CLI fallback 仍只返回纯文本。
- 验证通过：`cd server && go test ./internal/server ./internal/biz ./internal/data`；新增测试覆盖工具定义透传、tool result 历史转换、Codex backend SSE function call 解析和兼容响应回填；远端源站和域名 `/healthz`、`/readyz` 均返回正常；Windows OpenCode 使用 `oauth-api-service/gpt-5.5` 成功触发本地 `bash` 工具，执行 `Write-Output OPENCODE_SHELL_OK` 并返回 `exit=0`。
- 下一步：如需原生截图能力，需要确认 Windows OpenCode 是否加载了截图插件或自定义工具；当前 stock OpenCode 日志只注册了 `bash/read/glob/grep/task/webfetch/todowrite/skill/apply_patch` 等工具，没有名为 `screenshot` 的独立工具。
- 运维清理：发布后记录远端 `/` 使用率 40%、Docker images 4.71GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，仅删除未被容器使用的旧 `oauth-api-service-server:20260511T115416-d530adb7` 镜像，回收 347.5MB；清理后 `/` 使用率 38%，所有运行中容器和当前镜像保持正常。
- 阻塞/风险：工具调用依赖 Codex direct backend 协议继续兼容 Responses function_call 事件；若上游协议变化，仍会按既有策略 fallback 到 CLI，但 fallback 不具备工具调用能力。远端上传的本轮镜像 tar 包尚未清理。

## 2026-05-11 Windows Codex 自定义 key 发消息排查
- 发现：Windows `~/.codex/config.toml` 已配置 `model_provider = "saurick-oauth"`、`base_url = "https://oauth-api.saurick.me/v1"`、`wire_api = "responses"` 和 `experimental_bearer_token`；当前 SSH 环境变量里 `OPENAI_API_KEY` / `OPENAI_BASE_URL` 为空，但 Codex 自定义 provider 走的是 config token，不依赖这些环境变量。
- 发现：`codex exec` 失败主因是服务端 `/v1/responses` 在 `stream=true` 时返回了 Chat Completions SSE chunk，Codex CLI 等不到 Responses SSE 的 `response.completed`，报 `stream closed before response.completed`。
- 修复：`/v1/responses stream=true` 改为输出 Responses SSE 事件序列，补齐 `response.output_item.added`、`response.content_part.added`、`response.output_text.delta`、`response.output_text.done`、`response.content_part.done`、`response.output_item.done` 和 `response.completed`；`/v1/models` 额外返回 Codex CLI 读取的 `models[].slug`、`display_name` 等兼容字段，同时保留 OpenAI 标准 `data` 字段。
- 部署：本地构建并依次部署 `oauth-api-service-server:20260511T151900-d530adb7-local`、`20260511T152500-d530adb7-local` 和最终 `20260511T152900-d530adb7-local`；远端仅执行 `docker load` 与 `docker compose up -d app-server`，未在服务器构建。
- 验证通过：`cd server && go test ./internal/server ./internal/biz ./internal/data`；线上 `/healthz` 与 `/readyz` 正常；Windows 执行 `codex exec --skip-git-repo-check --ephemeral --ignore-rules --json -s read-only -m gpt-5.5 -c model_provider="saurick-oauth" "只回复 OK"` 已返回 `item.completed`，文本为 `OK`。
- 运维清理：发布后记录远端 `/` 使用率 45%、Docker images 6.116GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的本轮中间镜像 `20260511T150703/151900/152500-*`，回收 1.043GB；清理后 `/` 使用率 39%，当前运行镜像 `20260511T152900-d530adb7-local` 与所有容器保持正常。
- 阻塞/风险：Codex 模型刷新仍会提示 `models[].supported_reasoning_levels` 等官方模型元数据字段缺失，但不影响发消息主链路；如需消除 warning，可继续按 `models_cache.json` 的字段结构补齐完整模型目录。远端上传的本轮镜像 tar 包尚未清理。

## 2026-05-11 工具请求禁用 CLI fallback
- 发现：OpenCode / Codex 这类客户端的工具请求依赖 direct `codex_backend` 返回 `tool_calls` 后由客户端本机执行；若 backend 失败后自动 fallback 到 `codex_cli`，服务端会启动 `codex exec`，读取系统信息或文件时可能命中服务器容器的 `bwrap` / user namespace 限制。
- 修复：`codex_backend` 失败时先检查请求是否包含 `tools`、强制 `tool_choice`、assistant `tool_calls`、tool result、Responses `function_call(_output)` 或文件输入；这些 backend-only 请求直接返回 backend 上游错误，不再降级到服务端 CLI。
- 文档：同步更新 `README.md`、`server/docs/api.md`、`server/docs/config.md` 和 Compose 生产 README，明确只有 CLI 能忠实处理的纯文本/图片请求才允许 fallback，工具和文件请求不 fallback。
- 验证通过：新增测试覆盖带工具请求在 backend 失败时不会调用 fake Codex CLI fallback，避免再次出现服务端 `bwrap` 沙箱错误被当成客户端工具结果。
- 阻塞/风险：如果 direct Codex backend 本身不可用，OpenCode 工具请求会返回 502，而不是降级出一个不完整的纯文本回答；需要通过 usage 的 `upstream_mode/upstream_fallback/upstream_error_type` 或服务日志继续排查 backend 连接问题。

## 2026-05-11 Backend fallback 降低与线上部署
- 发现：线上近 6 小时 usage 中有 `42` 次 `codex_backend -> codex_cli` fallback、`103` 次 backend 直接成功和 `7` 次 backend 502；容器内用同一份 `auth.json` 直接请求 Codex backend 的纯文本、工具声明和工具结果回合均返回 `HTTP 200`，说明代理、登录态和基础工具格式不是全局失效，fallback 更像上游瞬时失败或特定请求触发的 backend error。
- 修复：direct backend 增加有限重试，默认 `CODEX_BACKEND_RETRY_ATTEMPTS=2`，仅对 HTTP `429` / `5xx`、上游 `response.failed` / `response.incomplete` 和连接类错误生效；backend 失败准备 fallback 前会写 warning 日志，后续能看到真实 backend 错误内容。
- 部署：本地构建镜像 `oauth-api-service-server:20260511T191924-d530adb7-local`，上传到 `/data/openai-oauth-api-service/releases/20260511T191924-d530adb7-local/`；远端只执行 `docker load`、备份并更新 compose / `.env`、`docker compose up -d app-server`，未在服务器构建。
- 验证通过：远端当前运行镜像为 `oauth-api-service-server:20260511T191924-d530adb7-local`，`CODEX_UPSTREAM_MODE=codex_backend`、`CODEX_BACKEND_RETRY_ATTEMPTS=2`；容器内和公网 `/healthz`、`/readyz` 均正常。
- 验证通过：公网真实工具请求两轮 `/v1/chat/completions` 均返回 `HTTP 200` 和 `tool_calls`；usage 记录 `id=407/408` 均为 `upstream_mode=codex_backend`、`upstream_fallback=false`、`status_code=200`，确认没有再降级到服务端 CLI。
- 清理：部署前 `/` 使用率 40%、Docker images 4.007GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `20260511T152900-d530adb7-local`，回收 347.6MB；清理后 `/` 仍为 40%，当前运行容器与当前镜像保持正常，未执行 volume prune。
- 阻塞/风险：本轮没有 schema 变更，不需要 migration；旧 release tar 包仍保留用于短期回滚。截图中暴露过的 `ogw_0OkfLIrV...` key 建议后台轮换，避免继续被第三方复用。

## 2026-05-11 默认关闭 CLI fallback
- 调整：新增 `CODEX_UPSTREAM_FALLBACK_ENABLED`，默认 `false`；`codex_backend` 失败时默认直接返回上游错误，不再自动降级到 `codex_cli`。确需临时救急时可设为 `true`，但工具调用、工具历史和文件输入仍始终不 fallback。
- 目的：避免隐藏 backend 退化，避免纯文本 fallback 掩盖上游问题，并防止服务端 `codex exec` 在低配服务器上引入额外延迟、token 放大和沙箱语义差异。
- 文档：同步更新 `README.md`、`server/docs/api.md`、`server/docs/config.md`、Compose `.env.example`、`compose.yml` 与生产 Compose README。
- 部署：本地构建镜像 `oauth-api-service-server:20260511T194408-d530adb7-local`，上传到 `/data/openai-oauth-api-service/releases/20260511T194408-d530adb7-local/`；远端备份并更新 `.env` / `compose.yml`，设置 `CODEX_UPSTREAM_FALLBACK_ENABLED=false` 后重启 `app-server`，未在服务器构建。
- 验证通过：`cd server && go test -count=1 ./internal/server ./internal/biz ./internal/data`、`git diff --check`；新增测试覆盖默认不 fallback、显式打开开关才允许纯文本 fallback，以及工具请求不 fallback。
- 验证通过：远端当前运行镜像为 `oauth-api-service-server:20260511T194408-d530adb7-local`，环境变量包含 `CODEX_UPSTREAM_MODE=codex_backend`、`CODEX_UPSTREAM_FALLBACK_ENABLED=false`；容器内和公网 `/healthz`、`/readyz` 均正常。
- 验证通过：公网真实工具请求返回 `HTTP 200` 和 `tool_calls`；usage 最新记录 `id=512` 为 `upstream_mode=codex_backend`、`upstream_fallback=false`、`status_code=200`。

## 2026-05-11 API 凭据多选
- 完成：API 凭据表保持单击行互斥单选，复选框改为可累加多选；用量日志「调用凭据」筛选改为多选，前端向 usage 列表、每日模型、凭据统计、会话聚合、异常请求等统一传 `key_ids`。
- 完成：后端 `GatewayUsageFilter` 增加 `KeyIDs`，JSON-RPC 与 usage 导出均支持 `key_ids`，并保留旧 `key_id` 单选兼容；多选时优先按 `key_ids` 过滤。
- 文档：同步更新 `server/docs/api.md`，说明 `api.usage_list` 及相关聚合接口支持 `key_ids` 多凭据过滤。
- 验证通过：`cd web && pnpm test`、`cd server && go test ./internal/biz ./internal/data ./internal/server`；`pnpm style:l1` 新增覆盖凭据多选筛选并通过。
- 下一步：无阻塞；未涉及 schema 变更和部署配置，不需要 migration。

## 2026-05-12 客户端配置模板页面
- 完成：新增后台「客户端模板」菜单和 `/admin-client-config` 页面，支持 macOS / Windows 的 Codex 与 opencode 配置模板导出；模板只保留 Base URL、API Key、Codex profile、模型变体和必要运行选项，不导出 auth.json、历史会话、projects 信任记录、opencode secrets 或本机绝对路径。
- 完成：按用户要求通过 `ssh sauri@192.168.0.44` 查看 Windows 配置口径，确认 Windows Codex 需要 `[windows] sandbox = "elevated"`，Windows opencode 配置主路径为 `%USERPROFILE%\.config\opencode\opencode.json`，Codex 为 `%USERPROFILE%\.codex\config.toml`。
- 完成：页面提供上传配置文件入口，只替换显式占位符 `{{BASE_URL}}`、`{{API_KEY}}`、`{{PROFILE}}`，不做启发式替换，避免误改个人字段或隐藏状态。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm style:l1`、`cd web && pnpm build`；`style:l1` 覆盖客户端模板页桌面 / 移动、浅色 / 暗色、上传占位符替换、无横向溢出和后台 chrome 回归。
- 阻塞/风险：本轮是前端静态模板导出功能，没有后端持久化上传文件，也没有把真实 key 落库；如果后续要做共享模板管理，需要单独设计密钥脱敏、权限和审计。

## 2026-05-12 客户端配置模板部署
- 部署：本地完成 `cd server && go test ./internal/server ./internal/biz ./internal/data`、`cd web && pnpm test`、`cd web && pnpm style:l1` 后，按低配服务器发布约定在本机构建镜像 `oauth-api-service-server:20260512T130207-bc28db5-local`，上传到 `8.218.4.199` 后仅执行 `docker load` 和 `docker compose up -d app-server`，未在服务器构建。
- 验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T130207-bc28db5-local`；容器内与公网 `/healthz` 均返回 `ok`，`/readyz` 均返回 `ready`；`https://oauth-api.saurick.me/admin-client-config` 返回前端 HTML；管理员 JSON-RPC 登录与 `api.summary` 查询返回 `code=0`。
- 清理：部署前记录远端 `/` 使用率 48%、Docker images 4.71GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260512T043120-bc28db59-local`，回收 347.6MB；清理后 `/` 使用率 46%、Docker images 4.007GB，所有运行中容器保持正常；已删除本轮上传 tar 包与本地临时 tar 包。
- 阻塞/风险：本轮没有 schema 变更，不需要 migration；当前部署包含工作区内尚未提交的多处服务端与前端改动，后续提交时需按路径核对，不要误以为只有客户端模板页改动。

## 2026-05-12 客户端配置模板上传入口移除
- 完成：移除 `/admin-client-config` 页面里的“上传已有模板并替换占位符”入口，页面只保留内置 Codex / opencode 模板的参数填写、预览、复制和下载，减少不必要功能复杂度。
- 完成：删除前端 `replaceClientConfigPlaceholders` helper 及对应测试，更新 README 与 `style:l1` 断言，避免继续维护无实际收益的上传分支。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm style:l1`；客户端模板页桌面 / 移动、浅色 / 暗色、无上传入口、无横向溢出和后台 chrome 回归通过。
- 下一步：如需上线，需要重新构建镜像并按低配服务器发布流程部署；当前仅完成本地代码修改与验证。

## 2026-05-12 客户端配置下载文件名修正与部署
- 完成：客户端模板下载文件名改为真实配置文件名：Codex 固定 `config.toml`，opencode 固定 `opencode.json`，不再使用 `codex-config.windows.toml` / `opencode.windows.json` 这类自定义文件名，方便直接替换目标配置文件。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm style:l1`、`cd web && pnpm build`；新增测试确认 macOS / Windows 下载文件名均与真实配置文件名一致。
- 部署：本地构建镜像 `oauth-api-service-server:20260512T132600-bc28db5-local`，上传到 `8.218.4.199` 后仅执行 `docker load` 和 `docker compose up -d app-server`，未在服务器构建。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T132600-bc28db5-local`；容器内与公网 `/healthz` 返回 `ok`，`/readyz` 返回 `ready`；`https://oauth-api.saurick.me/admin-client-config` 已返回新前端资源 `index.BMklAv9U.js`。
- 清理：部署前记录远端 `/` 使用率 48%、Docker images 4.71GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260512T130207-bc28db5-local`，回收 347.6MB；清理后 `/` 使用率 46%、Docker images 4.007GB；已删除本轮上传 tar 包与本地临时 tar 包。

## 2026-05-12 客户端配置教程文案收口
- 完成：客户端模板页安装教程第 3 步改为按当前选择显示单一客户端名称，选择 Codex 时显示“安装 Codex”，选择 opencode 时显示“安装 opencode”，不再写“Codex 或 opencode”。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm style:l1`。
- 下一步：如需上线，需要重新构建镜像并按低配服务器发布流程部署；当前仅完成本地代码修改与验证。

## 2026-05-12 客户端配置教程文案部署
- 部署：本地构建镜像 `oauth-api-service-server:20260512T133600-bc28db5-local`，上传到 `8.218.4.199` 后仅执行 `docker load` 和 `docker compose up -d app-server`，未在服务器构建。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T133600-bc28db5-local`；容器内与公网 `/healthz` 返回 `ok`，`/readyz` 返回 `ready`；`https://oauth-api.saurick.me/admin-client-config` 已返回新前端资源 `index.UclIJ6sg.js`。
- 清理：部署前记录远端 `/` 使用率 48%、Docker images 4.71GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260512T132600-bc28db5-local`，回收 347.6MB；清理后 `/` 使用率 46%、Docker images 4.007GB；已删除本轮上传 tar 包与本地临时 tar 包。

## 2026-05-12 opencode 教程文案部署
- 完成：客户端模板页教程第 1 步改为按当前客户端显示，选择 opencode 时只提示填写 Base URL 和 API Key，不再出现 Codex profile 文案；选择 Codex 时仍提示确认 profile。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm style:l1`。
- 部署：本地构建镜像 `oauth-api-service-server:20260512T141400-bc28db5-local`，上传到 `8.218.4.199` 后仅执行 `docker load` 和 `docker compose up -d app-server`，未在服务器构建。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T141400-bc28db5-local`；容器内与公网 `/healthz` 返回 `ok`，`/readyz` 返回 `ready`；`https://oauth-api.saurick.me/admin-client-config` 已返回新前端资源 `index.RpJgmBD_.js`。
- 清理：部署前记录远端 `/` 使用率 48%、Docker images 4.71GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260512T133600-bc28db5-local`，回收 347.6MB；清理后 `/` 使用率 46%、Docker images 4.007GB；已删除本轮上传 tar 包与本地临时 tar 包。

## 2026-05-12 客户端模板默认 medium 部署
- 完成：Codex 模板默认 `model_reasoning_effort` 改为 `medium`；opencode 模板默认 agent `variant` 改为 `medium`，模型默认 `reasoningEffort` 改为 `medium`；opencode 模板移除 `gpt-5.4`，只保留 `gpt-5.5`。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm style:l1`；测试覆盖 Codex 默认 medium、opencode build/plan medium、opencode 只包含 `gpt-5.5`。
- 部署：本地构建镜像 `oauth-api-service-server:20260512T143300-bc28db5-local`，上传到 `8.218.4.199` 后仅执行 `docker load` 和 `docker compose up -d app-server`，未在服务器构建。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T143300-bc28db5-local`；容器内与公网 `/healthz` 返回 `ok`，`/readyz` 返回 `ready`；`https://oauth-api.saurick.me/admin-client-config` 已返回新前端资源 `index.Cu2SXAFD.js`。
- 清理：部署前记录远端 `/` 使用率 48%、Docker images 4.71GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260512T141400-bc28db5-local`，回收 347.6MB；清理后 `/` 使用率 46%、Docker images 4.007GB；已删除本轮上传 tar 包与本地临时 tar 包。

## 2026-05-12 客户端模板代码预览对比度修复
- 完成：修复 `/admin-client-config` 配置预览在浅色模式下代码文字被后台主题全局 `text-slate-100` 覆盖为深色导致看不清的问题；预览区改用 `admin-code-preview` 专用样式，固定深色代码背景与高对比浅色代码文本，不使用 `!important`。
- 回归：`style:l1` 的客户端模板页断言新增浅色 / 暗色代码预览对比度检查，继续覆盖桌面、移动端、无横向溢出和后台 chrome。
- 验证通过：`cd web && pnpm css`、`cd web && pnpm test`、`cd web && pnpm style:l1`。
- 下一步：当前仅完成本地前端修复；如需线上生效，需要重新构建镜像并按低配服务器发布流程部署。
