## 归档索引
- 2026-05-10 之前历史流水：`docs/archive/progress-2026-05-10-pre-docker-cleanup-constraint.md`。
- 当前文件保留 2026-05-10 以来新增记录；归档文件只作追溯线索，不作为当前正式需求真源。

## 2026-05-26 登录态失效弹窗统一
- 完成：将全局 `AppModal` / `AlertDialog` 从旧深色通用面板改为复用后台 `admin-modal-*` 弹窗结构、标题、关闭按钮、确认按钮和浅色 / 暗色主题变量；登录态失效弹窗现在与 API key 确认、模型上下文、用量详情等后台弹窗保持一致。
- 完成：后台内容列增加页面级横向 overflow 收口，避免移动端宽表在自己的 `overflow-auto` 滚动容器外继续撑出 body 横向滚动；内部表格横向滚动能力保留。
- 完成：同步修复 `/admin-keys` 完整凭据列宽，把 key 列从 260px 扩到 300px，避免复制按钮把完整 key 文本压窄到逐字符竖排风险区间。
- 完成：`style:l1` 增加 `STYLE_L1_SCENARIOS` 局部场景过滤，并新增登录态失效弹窗桌面浅色与移动暗色回归，覆盖弹窗结构、主题颜色、按钮圆角、关闭按钮尺寸、确认后跳转登录页和移动端无页面级横向溢出。
- 文档：更新 `web/README.md`，补充登录态失效弹窗已纳入 `style:l1`，以及局部场景过滤用法。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm css`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/common/components/layout/AdminFrame.jsx src/common/components/modal/AppModal.jsx src/common/components/modal/AlertDialog.jsx scripts/styleL1.mjs`、`cd web && STYLE_L1_PORT=4367 NODE_USE_ENV_PROXY=0 STYLE_L1_SCENARIOS=admin-dashboard-mobile,admin-session-expired-modal-desktop,admin-session-expired-modal-mobile-dark pnpm style:l1`、`cd web && STYLE_L1_PORT=4370 NODE_USE_ENV_PROXY=0 STYLE_L1_SCENARIOS=admin-keys-desktop pnpm style:l1`、`cd web && STYLE_L1_PORT=4371 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`cd web && pnpm build`、`bash scripts/qa/secrets.sh`、`git diff --check`。
- 验证补充：in-app Browser 通过 OAuth callback 测试管理员态打开 `http://127.0.0.1:4368/admin-dashboard`，页面身份和首屏渲染正常，仅有 React Router v7 future warning；本地旧 `VITE_ENABLE_RPC_MOCK` 未覆盖当前 `admin_login` / `api` 主路径，因此登录态弹窗和 API key 列宽用 `style:l1` 网络 mock 做确定性回归。
- 下一步：提交推送后按低配服务器主路径部署，远端只执行镜像加载、Atlas 状态 / 迁移、重建、健康检查和清理。

## 2026-05-26 公开客户端配置生成页
- 完成：新增免登录 `/client-config` 公开入口，供非管理员朋友填写自己的 Base URL、API Key、客户端和系统后在浏览器本地生成 Codex / opencode 配置；现有 `/admin-client-config` 仍保留 `AuthGuard requireAdmin`，不放开后台路由。
- 完成：将原后台客户端模板页的表单、预览、复制和下载逻辑抽成 `ClientConfigBuilder` 共享组件；后台页继续使用 `AdminFrame` 和后台导航，公开页只使用轻量页头与主题切换，不展示后台导航、超级管理员标识或退出按钮。
- 文档：更新 `web/README.md`，明确 `/client-config` 不调用后端接口、不保存 API Key，`style:l1` 覆盖公开配置生成页。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm build`、`git diff --check`；in-app Browser 验证 `http://127.0.0.1:5177/client-config` 桌面默认态、填写 `https://proxy.example.test/v1/` 与 `ogw_demo_public_key` 后切换 opencode 的预览更新、无后台壳、无横向溢出，并验证 390x844 移动视口默认态无横向溢出。
- 阻塞/风险：`cd web && pnpm style:l1` 已跑过新增公开页场景，但全量随后停在既有 `admin-keys-desktop` 完整凭据列宽断言（`valueWidth=116 < 140`），本轮未改 API key 表格列宽；本轮仅完成本地代码和前端验证，未部署线上。

## 2026-05-26 Codex 自定义 provider 可见过程说明
- 完成：只读排查线上 `https://oauth-api.saurick.me/v1/responses`，确认生产容器健康且运行镜像 `oauth-api-service-server:20260525T122959-5d604ae3-local-codex-0.133.0`；无显式引导的工具调用请求只返回 `function_call` 事件，`response.output_text.delta=0`、`response.reasoning_summary_text.delta=0`，说明当前默认链路没有把中文过程说明引导出来。
- 完成：同一线上链路在请求 `instructions` 显式要求“调用任何工具之前先输出一句简短中文说明”时，会先返回 `phase=commentary` 的中文 `output_text.delta`，再返回 `function_call`，说明服务端和上游具备承载可见 commentary 的能力，缺口在默认 Codex backend instructions。
- 完成：`codexBackendInstructions` 追加服务端级可见过程说明规则，要求非平凡工具调用、读文件、命令、SSH、浏览器操作或外部请求前先输出一到两句简体中文可见 commentary；同时保留禁止输出隐藏思维链的边界，并保持对客户端显式 `instructions` 与压缩恢复规则的幂等追加。
- 验证通过：`cd server && go test ./internal/server -run 'TestCodexBackendRequestUsesDefaultInstructions|TestCodexBackendRequestAppendsResumeRuleToExplicitInstructions|TestCodexBackendInstructionsAreIdempotent|TestCodexBackendRequestPreservesReasoningSummary|TestCodexBackendRequestPassesAllReasoningEfforts'`、`cd server && go test ./internal/server`、`cd server && go test ./...`、`git diff --check -- server/internal/server/codex_backend_adapter.go server/internal/server/openai_gateway_handler_test.go progress.md`。
- 部署：本地构建 linux/amd64 镜像 `oauth-api-service-server:20260526T090202-5d604ae-local-visible-commentary`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260526T090202-5d604ae-local-visible-commentary/`；远端只执行 checksum、`docker load`、Atlas migration status、备份并更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 部署注意：首次 `docker load -i image.tar.gz` 期间宿主机出现重启，旧容器自动恢复；复查包 checksum / gzip 均正常，改用 `gunzip -c image.tar.gz | docker load` 后镜像加载成功，加载期间 `/healthz` 持续返回 `ok`。
- 验证通过：远端 Atlas `migrate status` 为 `Already at latest version`；容器运行镜像为 `oauth-api-service-server:20260526T090202-5d604ae-local-visible-commentary`，容器内 `codex --version` 为 `codex-cli 0.133.0`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`。
- 验证通过：线上临时 key 默认工具调用请求收到 `LOCAL_OUTPUT_TEXT_DELTAS=19`、`LOCAL_TEXT_BEFORE_FUNCTION=19`、`LOCAL_FUNCTION_EVENTS=4`，样例为“我将调用工作区列表工具，查看当前目录下可用的文件和结构。”；Windows 主机 `sauri@192.168.0.45` 侧请求收到 `WINDOWS_OUTPUT_TEXT_DELTAS=22`、`WINDOWS_TEXT_BEFORE_FUNCTION=22`、`WINDOWS_FUNCTION_EVENTS=4`，样例为“我将调用 `list_workspace` 查看当前工作区文件列表，以便了解可用文件和目录。”；两次临时 key 均已删除并确认 `key_list search total=0`。
- 文档：`server/docs/api.md` 已补充 direct backend 的服务端级 Codex 运行规则，明确可见过程说明是用户可见 commentary / process summary，不是隐藏 chain-of-thought；该规则对客户端显式 `instructions` 幂等追加，不依赖 Windows 端 `hide_agent_reasoning` 或全局 AGENTS。
- 清理：清理前远端 `/` 使用率 53%、Docker images 4.78GB；执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除上一版未使用 app 镜像 `20260525T122959-5d604ae3-local-codex-0.133.0`，回收 352.3MB；清理后 `/` 使用率 51%、Docker images 4.067GB，未执行 volume prune。
- 阻塞/风险：上线回归中有两次客户端侧测试超时 / 脚本中断导致服务日志出现 `client_canceled` WARN，后续同类请求已通过；该 WARN 属于本轮验证噪音，不代表当前服务启动失败。

## 2026-05-25 远端 Codex CLI 升级
- 完成：确认 npm `@openai/codex` 当前 `latest` 为 `0.133.0`；远端宿主机 `8.218.4.199` 原 `/usr/local/bin/codex` 为 `0.130.0`，线上 `openai-oauth-api-service-server` 容器内实际调用的镜像内 Codex CLI 为 `0.129.0`。
- 完成：已将宿主机全局 `@openai/codex` 升级到 `0.133.0`，并将 `server/Dockerfile` 中镜像内固定版本同步调整为 `0.133.0`。
- 部署：本地构建 linux/amd64 镜像 `oauth-api-service-server:20260525T122959-5d604ae3-local-codex-0.133.0`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260525T122959-5d604ae3-local-codex-0.133.0/`；远端仅执行 `docker load`、备份并更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 验证通过：远端宿主机 `codex --version` 为 `codex-cli 0.133.0`；容器内 `codex --version` 和 `npm list -g @openai/codex` 均为 `0.133.0`，`CODEX_HOME=/root/.codex codex login status` 显示 `Logged in using ChatGPT`。远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；公网 `/public/codex/balance` 返回 200；容器运行镜像为 `oauth-api-service-server:20260525T122959-5d604ae3-local-codex-0.133.0`，`GIT_SHA_SHORT=5d604ae3-local`。
- 清理：清理前远端 `/` 使用率 53%、Docker images 4.798GB；删除本轮 release 镜像 tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260522T185252-d88c2651-local-big-image-90m` 和未使用 `frpc` 镜像，回收 354.8MB；清理后 `/` 使用率 51%、Docker images 4.07GB，未执行 volume prune。

## 2026-05-24 后台分页箭头样式
- 完成：根据 Codex 网页标注定位 `/admin-models` 分页「下一页」按钮，确认共享分页箭头仍使用大号 `›` 字符，移动视口下按钮为 44x44 但箭头字形达 34px，视觉上偏大且不稳定。
- 完成：将共享 `.admin-page-button-arrow` 从可见字体箭头改为 CSS chevron，保留原 `aria-label` 与分页结构；箭头图形收口为 16px 居中盒 + 8px 线性箭头，避免依赖 `‹/›` 字形重心。
- 补充完成：根据 1024px 标注宽度继续压缩分页整体密度，分页字号从 22px 降到 16px，圆形页码 / 箭头从 48px 降到 36px，移动端为 34px；每页条数输入从 132px / 48px 降到 108px / 36px，避免页码和「N 条/页」区域挤压。
- 修复：每页条数控件首次压缩到 108px 后，最长 `100 条/页` 会被右侧 caret 预留区挤压；最终调整为 120px 宽、左右 padding 为 `10px / 26px`，仍明显小于旧 132px，同时保证最长文案完整显示。
- 验证补充：`style:l1` 分页断言新增箭头盒模型检查，覆盖箭头字体隐藏、16px 居中图标盒、8px chevron、2px 线宽、按钮中心偏移、页码按钮和每页条数输入的压缩后尺寸，并额外切换到 `100 条/页` 断言 `scrollWidth <= clientWidth`，防止最长选项再次被裁切。
- 验证通过：in-app Browser 在 `/admin-models` 设置 1024x768 视口，确认分页高度 48px、控件行高 36px、当前页 36x36、每页条数 120x36、可用文字区 80px、页码到每页条数间距 8px、选择 `100 条/页` 后无输入框滚动裁切，分页和 document 均无横向溢出；早前 772x994 和 1280x900 视口也确认箭头居中与无横向溢出。`cd web && pnpm css`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx scripts/styleL1.mjs`、`cd web && pnpm build`、`git diff --check -- web/src/tailwind.css web/scripts/styleL1.mjs progress.md` 均通过。
- 阻塞/风险：`cd web && STYLE_L1_PORT=4352 NODE_USE_ENV_PROXY=0 pnpm style:l1` 与换端口 `4353` 均停在既有 `admin-codex-balance-mobile` 暗色模式公开接口按钮对比度断言，未跑到后续分页场景；本轮已用 Browser 对标注目标做桌面 / 移动盒模型回归，但 `style:l1` 全量仍需后续先处理该既有断言。

## 2026-05-24 后台移动端左侧导航
- 完成：根据 Codex 网页标注定位 `/admin-models` 移动端后台导航，确认根布局只在 `lg` 以上启用左侧 grid，`lg` 以下把完整 `aside` 堆到顶部，导致导航占据首屏大块高度。
- 完成：后台根布局改为全断点左侧栏：极窄屏使用 118px 窄文字栏并保留 `aria-label/title`，`sm` 以上使用 220px 文字栏，`lg` 以上保留原 276px 桌面栏；顶部 header 和 main 始终从侧栏右侧开始。首次尝试纯图标栏会让移动端既有「用量日志」可见文本断言失败，已改回可见文字。
- 验证补充：`style:l1` 的 `assertAdminChrome` 改为断言后台侧边栏在所有断点都贴左，且不与 header/main 重叠，替代原先“移动端侧边栏必须在顶部”的旧口径。
- 验证通过：in-app Browser 复查 `/admin-models`，821px 标注宽度下侧栏宽 276px、390px 极窄宽度下侧栏宽 118px，`header/main` 均从侧栏右侧开始，「用量日志」文字可见，document/body 无横向溢出；`cd web && pnpm css`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/common/components/layout/AdminFrame.jsx scripts/styleL1.mjs`、`cd web && pnpm build`、`git diff --check -- web/src/common/components/layout/AdminFrame.jsx web/scripts/styleL1.mjs web/src/tailwind.css progress.md` 均通过。
- 阻塞/风险：`cd web && STYLE_L1_PORT=4357 NODE_USE_ENV_PROXY=0 pnpm style:l1` 已通过到 `admin-keys-desktop`，随后停在既有完整凭据列宽断言；本轮改的是后台框架左侧栏和分页样式，没有调整 API key 表格列宽。`4355` 曾因纯图标栏导致移动端可见文本断言失败，该问题已通过 118px 窄文字栏修正。

## 2026-05-22 大图片上传 413 排查
- 完成：使用公网 `https://oauth-api.saurick.me/v1/responses` 复现 data URL 大图片请求体 413；24 MiB 原始图片转 base64 后请求体约 32.00 MiB，线上返回 `HTTP 413`，响应体为 `{"code":"request_too_large","message":"request body too large"}`，说明命中 app-server 的总请求体限制，不是 Codex 上游拒绝。
- 完成：将 app-server OpenAI-compatible 总请求体上限从 32 MiB 调整为 90 MiB，并同步 Compose Nginx 样例 `client_max_body_size 90m`；保留图片 / PDF 单个附件 16 MiB、单次最多 4 个的业务限制，避免 data URL base64 膨胀后提前被总请求体限制误杀。
- 完成：更新 `README.md`、`server/docs/api.md`、`server/deploy/README.md` 与 `server/deploy/compose/prod/README.md` 的附件体积口径；新增单测校验 4 个最大图片 / PDF 附件的 base64 预算不会超过总请求体上限。
- 验证通过：线上旧版本边界复现已完成，临时测试 key 已删除；无效 key 探测约 70 MiB 请求当前会被宿主机 Nginx 返回 HTML 413，说明生产发布时也需要同步宿主机 Nginx `client_max_body_size`；本地 `cd server && go test ./internal/server -run 'TestGatewayRequestLimitCoversMaxBase64Attachments|TestCodexCLIPromptFromChatCompletionsPayloadMaterializesImages'`、`cd server && go test ./...`、`git diff --check` 均通过。
- 部署：本地构建 amd64 镜像 `oauth-api-service-server:20260522T185252-d88c2651-local-big-image-90m`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260522T185252-d88c2651-local-big-image-90m/`；远端仅执行 release 解包、宿主机 Atlas status、`docker load`、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，并备份后更新宿主机 Nginx snippet 为 `client_max_body_size 90m`、`nginx -t`、`systemctl reload nginx`；未在服务器构建，也未改管理员密码。Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`。
- 线上验证：远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；容器运行镜像为 `oauth-api-service-server:20260522T185252-d88c2651-local-big-image-90m`，容器环境 `GIT_SHA_SHORT=d88c2651-local`、`IMAGE_TAG=20260522T185252-d88c2651-local-big-image-90m`。公网 `/v1/responses` 34 MiB 请求不再 413，返回预期无效模型 `403 gateway_model_disabled`；绕过 Cloudflare 直连源站 HTTPS 的 65 MiB 请求也返回 `403 gateway_model_disabled`，确认宿主机 Nginx 64m 限制已解除；公网 70 MiB 请求不再 413，但触发 Cloudflare `524`，超大请求仍可能受边缘超时影响。最小 `/v1/responses` 非流式请求返回 `DEPLOY_BIG_IMAGE_90M_OK`；验证用临时 key 均已删除。
- 清理：清理前远端 `/` 使用率 53%、Docker images 4.721GB；删除本轮 release 镜像 tar 与 migration tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260522T175109-c0512d22-local-context-1m`，回收 348.5MB；清理后 `/` 使用率 50%、Docker images 4.017GB，未执行 volume prune。
- 下一步：若用户实际上传接近 70 MiB 请求体仍失败，优先排查 Cloudflare 524 / 上传耗时，而不是 app-server 或宿主机 Nginx 413。

## 2026-05-22 Codex 长会话上下文超限排查
- 完成：只读统计线上近 7 天 `gateway_usage_logs`，6210 次请求中失败 259 次，只有 2 次带 `session_id`，9 次触发网关压缩；失败不局限于 `zichun`，主要集中在无 session 的长流式 backend-only 请求，典型错误包括 `context_length_exceeded`、`response.incomplete max_output_tokens`、`unexpected EOF`、`client_canceled` 和历史孤立 `function_call_output` 400。
- 完成：定位根因之一是上下文预检 token 估算复用了 Codex CLI prompt 提取逻辑，只取最近 8 条 `user/assistant`，忽略较早 `tool`、`function_call_output`、`arguments`、`output` 等模型可见历史；线上 90 万字节以上 chat 请求诊断曾只估算数百到一千 tokens，导致没有提前压缩。
- 完成：`estimateGatewayRequestTokens` 改为从完整请求上下文提取 `content`、`text`、`arguments`、`output`、`tool_calls`、`summary` 和工具定义文本，避免长 session 工具历史被低估；内置 Codex 模型推荐 byte 压缩触发阈值先从 `compactTokens*4` 收紧并封顶到 `850000`，随后按最新口径调整为 `1M`，覆盖线上 92-96 万字节已出现上游 context 超限的区间。
- 完成：流式 backend 收到上游 `response.failed` / `response.incomplete` 时保留事件级错误分类；若事件正文包含 `context_length_exceeded`，usage 记录为明确 `context_length_exceeded`，不再被后续 EOF 收口覆盖成普通 `codex_backend_response_failed`。
- 补充完成：收窄服务端 backend 重试边界，只对上游 HTTP `429` / `5xx` 和连接类错误做有限重试；`context_length_exceeded`、上游终态 `response.failed` / `response.incomplete`、`client_canceled` 等不再服务端盲重试，避免和 Codex / OpenCode 客户端自身重试叠加放大请求。当前线上容器 Codex CLI 为 `0.129.0`，Windows 测试机 Codex CLI 为 `0.125.0`，npm `@openai/codex` latest 为 `0.133.0`。
- 验证通过：新增单元复现无上传文件的长 session 场景：旧工具输出占据 85-104 万字节时会在网关预检阶段提前压缩；旧 `tool` 内容会进入 token 估算；上游流式 `response.failed` 中的 context 超限会被准确分类；终态 `response.failed` / `response.incomplete` 不再被判定为可服务端重试。`cd server && go test ./internal/server ./internal/biz` 与 `cd server && go test ./...` 均通过。未用生产请求打上游复现，避免消耗线上额度和干扰真实用户。
- 补充验证：新增同一 session 多次压缩单测，模拟第一次压缩生成摘要、第二次继续压缩时携带前次摘要和最新“继续”指令；结果确认服务端在客户端稳定传 `session_id` 的前提下可以把摘要接回下一轮。该测试不代表 Codex App 会自动传 `session_id`，客户端是否传入仍需单独验证。
- 补充调整：内置 Codex 模型推荐与旧默认兜底的字节压缩触发阈值保留为服务端设置并调整到 `1M`，硬拦截仍为 `1.9M`；后台模型级配置和环境变量覆盖路径保持不变，不下放到 Codex / OpenCode 普通客户端模板。
- 部署：本地构建 amd64 镜像 `oauth-api-service-server:20260522T175109-c0512d22-local-context-1m`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260522T175109-c0512d22-local-context-1m/`；远端仅执行 release 解包、宿主机 Atlas status、`docker load`、更新 Compose `.env` 的 `APP_IMAGE` 和 `docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`。
- 线上验证：远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；容器运行镜像为 `oauth-api-service-server:20260522T175109-c0512d22-local-context-1m`，`GIT_SHA_SHORT=c0512d22-local`。管理员 RPC `model_list` 确认 `gpt-5.5` 生效上下文策略为 `400000 / 260000 / 380000 / 1000000 / 1900000 / 8`，数据库模型级字节覆盖仍为 `0 / 0`。
- 清理：清理前远端 `/` 使用率 53%、Docker images 4.731GB；删除本轮 release 镜像 tar 与 migration tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260522T162645-c0512d2-local-context-retry`，回收 348.5MB；清理后 `/` 使用率 51%、Docker images 4.026GB，未执行 volume prune。
- 下一步：观察近 24 小时 1M 字节以上请求的 `context_compacted`、`context_length_exceeded` 和 `codex_backend_response_incomplete` 占比；长期还应让 Codex/OpenCode 客户端稳定传 `session_id`，否则网关只能做单请求预检压缩，无法做到官方 App 那种跨线程 summary。

## 2026-05-20 Windows Codex 上下文压缩暂停排查
- 完成：通过 `ssh sauri@192.168.0.45` 检查 Windows Codex 现场，截图对应 session 为 `rollout-2026-05-20T16-47-14-019e4491-501b-7b92-afff-f82a3a2b85fd.jsonl`；压缩/恢复后模型产出了 `final_answer`，内容是“已加载这段历史上下文，请告诉我下一步”，因此表现为任务暂停。
- 完成：确认线上 `/v1/models` 当时声明 `supports_reasoning_summaries=true`、`default_reasoning_summary=auto`、`default_verbosity=medium`；问题 session 和 17:16 后续 session 的 `turn_context.summary` 仍为 `none`，说明当时 Codex Desktop 没有按 reasoning summary 模式发起恢复轮。
- 完成：继续检查 Windows VS Code storage、Codex Desktop 进程参数和 bundled `codex.exe`；同一份 0.131 app 内置 `codex.exe exec` 可读到 `reasoning summaries: detailed`，但 Desktop/VS Code app-server session 仍落为 `summary=none`，根因收敛为 Desktop/VS Code 会话创建路径覆盖/忽略 summary，而非 CLI、config.toml 或模型 catalog。
- 完成：direct backend 转发补齐 `reasoning.summary`：保留客户端 `auto/concise/detailed`，缺失或 `none` 时补 `detailed`；`/v1/models` 也改为声明 `default_reasoning_summary=detailed`，避免 Desktop/VSCode 自定义 provider 继续因 `summary=none/auto` 拿不到可用于压缩恢复的摘要。
- 完成：客户端配置模板补齐 Codex `model_reasoning_summary="detailed"`、`model_supports_reasoning_summaries=true` 和 `hide_agent_reasoning=false`，并覆盖默认配置、OpenAI profile、自定义 provider profile 与 fast/medium/high/deep profile，避免后续导出配置继续缺字段。
- 部署：本地构建 amd64 镜像 `oauth-api-service-server:20260520T175500-f0093787-local-summary-detailed`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T175500-f0093787-local-summary-detailed/`；远端仅执行 `docker load`、宿主机 Atlas status、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。第一次 `summary=auto` 版本验证后发现 Desktop session 仍收到空 summary，因此收紧为 `detailed` 后重新部署。
- 线上验证：远端 Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`；远端本机 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；Windows 侧 `/v1/models` 返回 `supports_reasoning_summaries=true`、`default_reasoning_summary=detailed`、`default_verbosity=medium`。Windows 流式 `/v1/responses` 显式传 `reasoning.summary=none` 时，服务端上游请求已改写为 `reasoning.summary=detailed`，HTTP 200、SSE `[DONE]`、最终文本包含 `SUMMARY_DETAILED_STREAM_OK`。
- 补充发现：真实 Codex Desktop 压缩线程继续执行时出现 502；生产日志显示上游拒绝 `function_call_output`，错误为 `No tool call found for function call output with call_id ...`。这说明 Desktop 压缩/重放后的历史里可能残留孤立工具输出，但对应 `function_call` 已不在本轮输入里。
- 补充完成：Codex backend 转发前统一过滤无对应前置 `function_call` 的 `function_call_output`，同时保留同一输入中合法的函数调用与工具输出配对；覆盖 `/v1/responses` 历史项和 `/v1/chat/completions` 的 `tool` 消息映射，避免压缩历史残值继续导致上游 400。
- 补充验证：`cd server && go test ./internal/server -run 'TestCodexBackendRequest|TestRunCodexBackendPostsResponsesAndParsesSSE'`、`cd server && go test ./...`、`git diff --check` 均通过。
- 补充部署：本地构建 amd64 镜像 `oauth-api-service-server:20260520T181855-f0093787-local-orphan-tool-output`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T181855-f0093787-local-orphan-tool-output/`；远端仅执行 `docker load`、宿主机 Atlas status、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 补充线上验证：远端 Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`；远端本机 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`，容器运行镜像为 `oauth-api-service-server:20260520T181855-f0093787-local-orphan-tool-output`。Windows 机器使用同一个 `saurick-oauth` provider 配置发送带孤立 `function_call_output` 的 `/v1/responses stream=true` 复现请求，返回 `HTTP=200`，文本包含 `ORPHAN_FILTER_OK`，生产日志未再出现 `No tool call found`、上游 `status=400` 或 `codex backend stream failed`。
- 补充完成：为 Codex backend `instructions` 统一追加压缩恢复行为规则：如果本轮是 compacted context、reasoning summary 或 history summary 恢复，应把摘要视为既有工作状态并继续未完成任务；用户说“继续 / 下一步”时必须执行下一步，禁止只机械回复“已读取压缩上下文，请告诉我下一步”，除非用户明确只要状态或存在真实阻塞。
- 恢复规则验证：`cd server && go test ./internal/server -run 'TestCodexBackendRequest'`、`cd server && go test ./...`、`cd web && node --test src/common/utils/clientConfigTemplates.test.mjs`、`git diff --check` 均通过。Windows 机器使用同一个 `saurick-oauth` provider 发送“压缩摘要 + 用户说下一步”的 `/v1/responses stream=true` 请求，最终 `output_text.done` 为 `RESUME_GUARD_OK_2`，未出现“已读取上下文 / 请告诉我下一步”式最终回答。
- 恢复规则部署：本地构建 amd64 镜像 `oauth-api-service-server:20260520T184137-ef3ec0e5-local-resume-guard`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T184137-ef3ec0e5-local-resume-guard/`；远端仅执行 `docker load`、宿主机 Atlas status、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。远端 Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`；远端本机 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`，容器运行镜像为 `oauth-api-service-server:20260520T184137-ef3ec0e5-local-resume-guard`。
- 恢复规则清理：清理前远端 `/` 使用率 53%、Docker images 4.731GB；删除 release 镜像 tar 与 migration tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260520T181855-f0093787-local-orphan-tool-output`，回收 348.5MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地 release 临时目录已移入废纸篓。
- 补充清理：清理前远端 `/` 使用率 53%、Docker images 4.731GB；删除 release 镜像 tar 与 migration tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260520T175500-f0093787-local-summary-detailed`，回收 348.5MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地 release 临时目录已移入废纸篓。
- 清理：清理前远端 `/` 使用率 54%、Docker images 5.435GB；删除本轮两个 release 镜像 tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260520T170126-2b731356-confirm-modal` 和中间验证镜像 `20260520T174642-f0093787-local-summary-backend`，回收 697MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune。
- 下一步：如仍要验证 Codex Desktop UI 的同线程续跑，需要先把 Windows App 里的 Codex 窗口恢复到可交互状态；当前 RDP 桌面只能看到进程和任务栏图标，窗口未正常弹出。本轮服务端已覆盖 Desktop 压缩后实际触发的两个失败条件：`reasoning.summary=none` 和孤立 `function_call_output`。

## 2026-05-20 Codex reasoning summary 远端部署验证
- 完成：当前环境无可用本地 Docker / WSL Docker，无法按常规方式构建镜像 tar；本轮改为在本地 WSL 交叉编译 linux/amd64 `server` 二进制，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T150611-82397c53-local-reasoning-summary/`，用 `docker cp` 替换现有 `app-server` 容器内 `/app/server` 后通过 Compose 重启；未在服务器执行 `docker build` / `go build`，未改管理员密码，也无 schema migration。
- 完成：远端本机 `/healthz`、`/readyz` 和公网 `https://oauth-api.saurick.me/healthz`、`/readyz` 均返回正常；`/v1/models` 返回 `supports_reasoning_summaries=true`、`default_reasoning_summary=auto`。
- 完成：公网 `/v1/responses stream=true` 携带 `reasoning.summary=detailed` 验证通过，SSE 包含 `response.reasoning_summary_text.delta`、`response.completed.output` 中的 `type=reasoning` item，最终答案包含 `DEPLOY_REASONING_SUMMARY_CN_OK`；网关将英文 reasoning summary 兜底替换为中文摘要“正在分析用户请求，核对上下文与约束，并组织最终回答。”。
- 验证通过：`cd server && go test ./...`、远端健康检查、公网 Responses reasoning summary 流式调用。
- 清理：部署前远端 `/` 使用率 51%、Docker images 4.244GB；按规则执行 `docker image prune -a -f` 与 `docker builder prune -f`，无可回收镜像 / 构建缓存，清理后 `/` 仍 51%、Docker images 4.244GB；未执行 volume prune。
- 阻塞/风险：由于当前本机缺少 Docker，本轮线上容器镜像标签仍显示旧镜像 `oauth-api-service-server:20260520T131239-9bb58677`，实际运行的是替换后的新 `/app/server` 二进制；release 目录和容器内 `/app/server.prev*` 暂保留用于短期回滚。后续有可用 Docker 后建议补一次正式镜像构建发布，收口镜像标签与二进制版本口径。
## 2026-05-20 Codex reasoning summary 支持
- 完成：自定义 provider 的 `/v1/models` 元数据改为声明 `supports_reasoning_summaries=true`、默认 `auto`，便于本地 Codex 在 `wire_api="responses"` 下显示过程摘要能力。
- 完成：direct backend 请求默认补 `reasoning.summary=auto`，并解析上游 `response.reasoning_summary_text.delta/done`；网关合成 Responses SSE 时会输出 reasoning summary 事件，并在 `response.completed.output` 中保留 `reasoning` item。
- 完成：更新后端 API 文档，说明 reasoning summary 请求、stream 事件与模型元数据口径。
- 下一步：用本地 Codex 自定义 provider 实际发起一次需要 reasoning 的请求，确认 UI 可展示中文摘要；若需要真正实时逐 token 过程，再把 backend SSE 从“收完后合成”升级为边读边转发。
- 阻塞/风险：当前实现是最小改动，summary 会随最终结果一并下发，不是完全实时透传；UI 固定英文标签如 `Thinking` / `Running` 仍由 Codex 客户端决定。
## 2026-05-20 API key 重置与普通保存隔离
- 补充完成：后台 `/admin-keys` 将单个 / 批量重置 API key、单个 / 批量删除 API 凭据从浏览器原生 `confirm` 改为项目内确认弹窗；弹窗展示操作标题、影响说明、取消按钮和危险确认按钮，提交期间锁定避免重复操作。
- 确认弹窗补充验证通过：`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs src/common/utils/tableInteraction.js src/common/utils/tableInteraction.test.mjs`、`cd web && pnpm test`、`cd web && pnpm build`、`cd server && go test ./...`、`cd web && STYLE_L1_PORT=4347 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。`style:l1` 已覆盖单个重置确认弹窗、批量重置确认弹窗、批量重置执行、浅色 / 暗色目标区域盒模型，并断言不再触发浏览器原生确认框。
- 确认弹窗补充部署：已提交并推送 `2b73135`（优化 API key 确认弹窗）；本地构建 amd64 镜像 `oauth-api-service-server:20260520T170126-2b731356-confirm-modal`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T170126-2b731356-confirm-modal/`；远端仅执行 `docker load`、宿主机 Atlas status、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 确认弹窗线上验证：远端 Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；容器运行镜像 `oauth-api-service-server:20260520T170126-2b731356-confirm-modal`。公网 `/admin-keys` 使用真实管理员登录，确认批量重置 8 个凭据、单个重置和删除都会打开站内确认弹窗，取消后不执行操作；原生浏览器 dialog 数为 `0`，控制台错误为 `0`，批量重置弹窗宽 `520px`、高 `226px`，document 横向溢出为 `0`。
- 确认弹窗清理：本轮 migration tar 首次解包带入 macOS `._*` 扩展属性文件，已在远端 release 目录删除后重新通过 Atlas 校验；清理前远端 `/` 使用率 52%、Docker images 4.731GB；删除 release 镜像 tar 和 migration tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260520T164604-827d8e98-select-all`，回收 348.5MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地 release 临时目录已移入废纸篓。
- 补充完成：参考 `trade-erp` 表格选择交互，为 `/admin-keys` 凭据表头增加当前页全选复选框；行点击仍保持单选，行首复选框支持多选，表头复选框支持当前页全选 / 取消并在部分选中时显示半选态。
- 补充完成：后台 `/admin-keys` 的「当前操作」区新增批量「重置 API key」按钮，支持选择框多选后一键轮换多个 key；确认后逐个调用 `api.key_reset_secret`，完成后展示本次所有新完整 key，并提供逐条复制和「复制全部完整凭据」。
- 全选补充验证通过：`cd web && pnpm test`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs src/common/utils/tableInteraction.js src/common/utils/tableInteraction.test.mjs`、`cd web && pnpm build`、`cd server && go test ./...`、`cd web && STYLE_L1_PORT=4346 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。
- 全选补充部署：已提交并推送 `827d8e9`（支持 API key 表格全选）；本地构建 amd64 镜像 `oauth-api-service-server:20260520T164604-827d8e98-select-all`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T164604-827d8e98-select-all/`；远端仅执行 `docker load`、宿主机 Atlas status、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 全选线上验证：远端 Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；公网 `/admin-keys` 使用真实管理员 token 打开后确认表头全选框存在、当前页 8 行可一次全选、再次点击可清空、批量重置按钮存在、document 横向溢出为 `0`，控制台无错误。
- 全选清理：清理前远端 `/` 使用率 53%、Docker images 4.731GB；删除 release 镜像 tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260520T163145-62b3d2ab-batch-reset`，回收 348.5MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地 release 临时目录已移入废纸篓。
- 完成：新增 `api.key_reset_secret`，重置时只改写 `key_hash`、`plain_key`、`key_prefix`、`key_last4`，保留备注、归属、模型限制、上游策略、额度、启用状态和历史 usage 归属；`api.key_update` 类型层面移除 secret 输入，普通保存备注、额度、模型或上游策略不会重新生成 API key。
- 完成：后台编辑 API 凭据弹窗新增「重置 API key」按钮和泄密场景说明；新建弹窗不显示该危险操作，备注提示改为明确普通保存不会重新生成 key。重置成功后会关闭弹窗并展示新的完整 key，方便立即复制同步客户端。
- 完成：同步更新 `server/docs/api.md`、`web/README.md` 和 `style:l1` 断言，当前真源为“普通编辑不轮换，泄密时手动重置轮换”。
- 批量重置补充验证通过：`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm build`、`cd server && go test ./...`、`cd web && STYLE_L1_PORT=4345 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。首次 `STYLE_L1_PORT=4344 NODE_USE_ENV_PROXY=0 pnpm style:l1` 在无关 `admin-codex-balance-mobile` 暗色按钮对比度断言偶发失败，原命令换端口重跑 26 个场景通过。
- 批量重置补充部署：已提交并推送 `62b3d2a`（支持批量重置 API key）；本地构建 amd64 镜像 `oauth-api-service-server:20260520T163145-62b3d2ab-batch-reset`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T163145-62b3d2ab-batch-reset/`；远端仅执行 `docker load`、宿主机 Atlas status、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 批量重置线上验证：远端 Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；容器运行镜像 `oauth-api-service-server:20260520T163145-62b3d2ab-batch-reset`，容器环境 `GIT_SHA_SHORT=62b3d2ab`。管理员 RPC 临时创建 `batchreset1` 和 `batchreset2` 两个 key 后连续重置，确认两个新完整 key 都已变化、备注保留、前缀正确；验证用临时 key 已删除。
- 批量重置清理：清理前远端 `/` 使用率 53%、Docker images 4.731GB；删除 release 镜像 tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260520T160410-6f45781c-local-reset-key`，回收 348.5MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地 release 临时目录已移入废纸篓。
- 验证通过：`cd server && go test ./internal/biz ./internal/data ./internal/server`、`cd server && go test ./...`、`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4342 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。
- 浏览器回归：本地 Vite `127.0.0.1:4343` 使用 mock 管理接口打开 `/admin-keys`，暗色主题下双击 key 行打开编辑弹窗，确认重置区块宽 `598px`、高 `132px`、包含泄密说明和按钮、普通保存不重新生成的提示存在，document 横向溢出为 `0`；接受确认后调用 `key_reset_secret`，页面展示新的完整 key，控制台无错误。
- 部署：本地构建 amd64 镜像 `oauth-api-service-server:20260520T160410-6f45781c-local-reset-key`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T160410-6f45781c-local-reset-key/`；远端仅执行 `docker load`、宿主机 Atlas status、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 线上验证：远端 Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；容器运行镜像 `oauth-api-service-server:20260520T160410-6f45781c-local-reset-key`。管理员 RPC 临时创建 `resetdeploy` key 后，`key_update` 保存备注为 `resetdeploy2` 时完整 key 保持不变，随后 `key_reset_secret` 返回 `ogw_resetdeploy2_` 前缀的新完整 key 且旧完整 key 已变化；验证用临时 key 已删除。
- 清理：清理前远端 `/` 使用率 53%、Docker images 4.731GB；删除 release 镜像 tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260520T153735-6f45781c-local`，回收 348.5MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地 release 临时目录已移入废纸篓。
- 阻塞/风险：本轮提供旧 key 无法找回时的手动轮换入口；点击重置会让旧 key 立即失效，需要同步更新客户端配置。当前代码已部署，但尚未提交和推送，需用户确认后再执行 Git 提交 / 推送。

## 2026-05-20 API key 完整展示恢复
- 完成：恢复新建 API key 时持久化 `plain_key`，管理员 `api.key_list` / `api.key_update` 重新返回完整凭据；普通组织用户 `user_key_list` 仍不返回完整明文。
- 完成：后台 `/admin-keys` 将「凭据标识」改为「完整凭据」，优先展示完整 `plain_key` 并提供复制；若历史数据仍缺少 `plain_key`，继续按前缀 + 后四位降级展示，避免伪造不可恢复的完整 key。
- 完成：同步更新 `server/docs/api.md`、`web/README.md` 和 `style:l1` 断言，收口当前真源为“管理员后台可见完整凭据”。
- 验证通过：`cd server && go test ./...`、`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4339 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`cd web && STYLE_L1_PORT=4341 NODE_USE_ENV_PROXY=0 pnpm style:l1`。
- 浏览器回归：本地 Vite `127.0.0.1:4340` 登录 `/admin-keys` 后确认「完整凭据」列返回完整 `ogw_` key、复制按钮存在、桌面 key 文本 `white-space=nowrap` 且高度 16px；移动暗色模式下表格改由内部横向滚动承载长 key，document 无横向溢出，只有 React Router v7 future flag 既有 warning。
- 部署：本地构建 amd64 镜像 `oauth-api-service-server:20260520T153735-6f45781c-local`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T153735-6f45781c-local/`；远端仅执行 `docker load`、宿主机 Atlas status、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 线上验证：远端 Atlas 状态 `OK`、当前版本 `20260520090000`、待执行 `0`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；管理员登录成功；临时创建 `deployfullkey` 验证创建响应返回完整明文、随后 `key_list` 能返回同一完整 `plain_key`，验证后已删除临时 key。
- 清理：清理前远端 `/` 使用率 52%、Docker images 4.731GB；删除 release 镜像 tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260520T131239-9bb58677`，回收 348.5MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune。
- 阻塞/风险：生产主表当前 10 条 key 中只有 1 条仍有 `plain_key`；备份表 `gateway_api_keys_backup_remark_prefix_remote_20260511T2255` 有 9 条明文，但与当前主表 `id/hash/prefix/last4` 均不匹配，不能安全回填。旧 key 完整明文无法从 `key_hash` 反推；新建 key 已恢复可在管理员后台完整展示和复制。完整 key 会重新进入数据库和管理员接口响应，日志参数脱敏仍覆盖 `plain_key`。

## 2026-05-20 Codex provider 流式透传与 key 明文收敛
- 完成：下游 API key 完整明文改为只在 `api.key_create` 创建响应中返回一次；`api.key_list` / `api.key_update` 不再返回 `plain_key`，后台列表只展示前缀与后四位，普通编辑备注、额度、模型权限、上游策略或禁用状态时不再改写 key hash / 前缀 / 后四位。
- 完成：新增数据迁移 `20260520090000_migrate.sql`，部署迁移时清空历史 `gateway_api_keys.plain_key` 残留明文；鉴权继续使用 `key_hash`，人工识别和 usage 归属继续使用 `key_prefix` / `key_last4`。
- 完成：`/v1/responses stream=true` 在 Codex backend 模式下改为直连透传上游 Responses SSE `data:` 事件，执行过程、文本增量、完成事件和 usage 不再由网关先缓存再重组；网关旁路解析 usage / 错误后照常落库，并保留 SSE keepalive。
- 完成：Codex CLI 收到 `response.completed` 后立即关闭连接时，usage 不再误记为 `client_canceled`；只要已解析完成事件且未收到 `response.failed` / `response.incomplete`，按成功记录 token、字节数和 backend 模式。
- 完成：`/v1/models` 的 Codex catalog 元数据不再固定 `reasoning_summary=none` 和 `verbosity=low`，改为声明支持 reasoning summary，默认 `auto` / `medium`，以匹配 Codex 长期多人接入的目标体验。
- 验证通过：`cd server && go test ./...`、`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4336 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。
- Windows 验证：`ssh sauri@192.168.0.45` 确认 Codex CLI `0.125.0`、OpenCode `1.14.48`；Windows 能访问本机修复版服务 `http://192.168.0.26:8401/healthz`；用临时 provider `oauthlocal` 指向本机 `/v1` 后，`codex exec` 返回 `WIN_CODEX_LOCAL_PROVIDER_OK_2`，服务端 usage 记录 `/v1/responses`、`stream=true`、`status=200`、`success=true`、`upstream_mode=codex_backend`、`total_tokens=12072`。
- 清理：Windows 验证用临时 API key 已删除；本机 8401 临时服务已停止。
- 提交推送：已提交并推送 `9bb5867`（修复 Codex 流式透传与凭据明文收敛）到 `origin/main`。
- 部署：本地构建 amd64 镜像 `oauth-api-service-server:20260520T131239-9bb58677`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260520T131239-9bb58677/`；远端仅执行 `docker load`、宿主机 Atlas migration、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 迁移：远端先清理 release `migrate/._*` 资源叉文件；Atlas dry-run 仅包含 `UPDATE "gateway_api_keys" SET "plain_key" = '' WHERE "plain_key" <> '';`；正式 apply 后状态 `OK`、当前版本 `20260520090000`、待执行 `0`。
- 部署验证：远端容器运行镜像 `oauth-api-service-server:20260520T131239-9bb58677`，容器环境 `GIT_SHA_SHORT=9bb58677`、`IMAGE_TAG=20260520T131239-9bb58677`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；管理员 `admin/adminadmin` 登录成功；临时 key 创建后只在创建响应返回完整明文，`key_list` 不返回 `plain_key`；`/v1/models` 返回 `supports_reasoning_summaries=true`、`default_reasoning_summary=auto`、`default_verbosity=medium`；生产 `/v1/responses stream=true` 返回 `response.completed`、`[DONE]` 和 `DEPLOY_STREAM_OK`；验证用临时 key 已删除；生产库 `plain_key` 非空数量为 `0`。
- 清理：清理前远端 `/` 使用率 52%、Docker images 4.731GB；删除本轮远端 release 镜像 tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260519T231421-9052e9d0`，回收 348.4MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地镜像 tar 和 migration tar 已移入废纸篓。
- 下一步：观察多人 Codex 长会话下的 usage 分布和 `response.failed` / `response.incomplete` 占比，必要时再按真实失败类型调整 backend fallback 策略。
- 阻塞/风险：迁移已清空历史 `plain_key`，后台无法再找回旧完整 key；遗失的 key 需要新建替换。

## 2026-05-19 模型上下文弹窗布局修复
- 完成：模型上下文策略弹窗改为专用宽度与响应式字段网格，字段内部增加 `min-width: 0`、输入框满宽、说明文字可换行，避免“填入当前值”、placeholder 和当前生效说明互相挤压、溢出或遮挡。
- 完成：`style:l1` 的模型管理桌面场景新增该弹窗浅色 / 暗色盒模型断言，覆盖面板横向溢出、字段数量、输入框边界、填值按钮边界和字段头部滚动宽度。
- 验证通过：`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && pnpm css`、`cd web && pnpm test`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4335 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check -- web/src/pages/AdminApi/index.jsx web/src/tailwind.css web/scripts/styleL1.mjs progress.md`。
- 浏览器回归：本地 Vite `127.0.0.1:4334` 登录 `/admin-models` 后打开模型上下文弹窗，桌面暗色 1280x720 量测面板宽 `980px`、字段宽 `296px`、面板与字段均无横向 scroll；移动浅色 390x844 量测面板宽 `358px`、字段宽 `290px`、body/document 均无横向溢出。
- 提交推送：已提交并推送 `9052e9d`（完善模型上下文策略与后台展示）到 `origin/main`。
- 部署：本地构建 amd64 镜像 `oauth-api-service-server:20260519T231421-9052e9d0`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260519T231421-9052e9d0/`；远端仅执行 `docker load`、宿主机 Atlas migration、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 迁移：首次 release migration tar 带出 macOS `._*` 资源叉文件导致 Atlas checksum mismatch；服务尚未重建，清理 release `migrate/._*` 后重跑 Atlas。最终状态 `OK`、当前版本 `20260519105354`、待执行 `0`。
- 部署验证：远端容器运行镜像 `oauth-api-service-server:20260519T231421-9052e9d0`，容器环境 `GIT_SHA_SHORT=9052e9d0`；远端本机 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；公网 `https://oauth-api.saurick.me/healthz`、`/readyz`、`/public/codex/balance` 均返回 200，管理员 `admin/adminadmin` 登录成功，`model_list` 返回 6 个 Codex 模型且首条生效阈值为 `400000 / 260000 / 380000 / 1040000 / 1900000 / 8`。
- 清理：清理前远端 `/` 使用率 52%、Docker images 4.73GB；删除本轮远端 release 镜像 tar 后执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260519T215756-5dc73aaf-local`，回收 348.4MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地镜像 tar 和 migration tar 已移入废纸篓。
- 阻塞/风险：本次只修复后台模型上下文弹窗并部署当前工作区，不改变管理员密码；release 目录保留 migration tar 和已清理资源叉后的 `migrate/` 目录用于追溯。

## 2026-05-19 Codex 余额公开接口入口
- 完成：`/admin-codex-balance` 顶部新增「打开公开接口」次级按钮，使用相对路径 `/public/codex/balance`、新标签页打开，并保留原「刷新」主操作；不在前端写死生产域名。
- 完成：`style:l1` 新增 Codex 余额桌面 / 移动端场景，mock 公开余额接口，覆盖按钮链接属性、余额卡片、进度条、暗色模式可读性和横向溢出检查。
- 验证通过：`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminCodexBalance/index.jsx scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && STYLE_L1_PORT=4332 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`cd web && pnpm build`、`git diff --check -- web/src/pages/AdminCodexBalance/index.jsx web/scripts/styleL1.mjs progress.md`。
- 浏览器回归：本地 Vite `127.0.0.1:4333` 通过 mock 后端打开 `/admin-codex-balance`，确认页面可见「打开公开接口」、链接 `href=/public/codex/balance`、`target=_blank`、`rel=noreferrer noopener`，点击刷新后仍显示「正常」且无横向溢出。
- 远端修复：为 `8.218.4.199` 的 `/etc/systemd/system/mihomo.service.d/docker-bridge.conf` 增加 `Wants=docker.service`、`After=network-online.target docker.service`，并在 `ExecStartPre` 中等待 `172.19.0.1/16` Docker bridge 出现后再启动；未切换代理节点或代理组。
- 远端验证：重启 `mihomo.service` 后 `172.19.0.1:7890` 已监听；宿主机经 `172.19.0.1:7890` 访问 `https://chatgpt.com/backend-api/wham/usage` 返回 `HTTP 405` 而非连接失败；容器内 Node fetch 同样返回 `status 405`；远端本机和公网 `/public/codex/balance` 均返回 `HTTP 200 status=ok`。
- 阻塞/风险：前端「打开公开接口」按钮代码已完成本地实现和验证，但本轮尚未构建并部署新的前端镜像到线上；生产后台是否能看到该按钮取决于后续部署。

## 2026-05-19 模型级上下文压缩策略
- 完成：为 `gateway_models` 新增上下文窗口、开始压缩 token / bytes、硬拦截 token / bytes 和压缩保留条数字段；后台模型页展示每个模型当前生效阈值，并提供“上下文”弹窗按模型保存显式覆盖，留空或 `0` 继续使用环境变量运维覆盖、内置模型推荐值或旧默认兜底。
- 完成：后台上下文阈值输入支持纯数字与 `K` / `M` 单位，例如 `260K`、`0.38M`；服务端 JSON-RPC 同步支持相同解析，`context_keep_items` 保持整数条数，避免把保留消息数误写成 token 单位。
- 完成：网关 `/v1/chat/completions` 与 `/v1/responses` 的压缩预检改为按请求模型读取实时策略；`/v1/models` 的 Codex 模型元数据使用当前生效 context window 和硬阈值比例；usage diagnostic 补充本次生效的窗口、压缩阈值、硬阈值和保留条数，便于排查。
- 完成：内置 Codex 模型推荐统一按 `400000` 上下文窗口生效，默认 `260000` 开始压缩、`380000` 硬拦截，字节阈值默认 `1040000` / `1900000`；大于 API 模型最大窗口的用法需要后台按模型显式覆盖，避免默认进入 long-context 高消耗区间。
- 顺手修复：默认 API key 备注不再取随机明文 key 的 last4，改用 key hash 前 8 位生成 `key<hex>`，避免 base64url 中 `_` / `-` 触发备注校验导致测试和创建偶发失败。
- 验证通过：`cd server && go test ./...`、`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4326 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。
- 客户端验证通过：本地开发库已应用 migration `20260519105354`；本地 OpenCode 使用 `oauth-api-service-local/gpt-5.5` 长输入触发压缩并返回 `LOCAL_OPENCODE_COMPRESS_OK`，usage 显示 `/v1/chat/completions` `compacted=true`，本次阈值按后台保存的 `2K` / `0.8M` 生效。
- 客户端验证通过：Windows `192.168.0.45` 上 Codex 0.125.0 通过临时 provider 指向本机开发服务，长输入返回 `WIN_CODEX_LOCAL_PROFILE_OK`，usage 显示 `/v1/responses` `compacted=true`；Windows OpenCode 1.14.48 临时切到本机开发服务后返回 `WIN_OPENCODE_COMPRESS_OK`，usage 显示 `/v1/chat/completions` `compacted=true`，测试后已恢复 Windows OpenCode 配置。
- 部署：本地构建镜像 `oauth-api-service-server:20260519T192827-5dc73aaf-local-model-context`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260519T192827-5dc73aaf-local-model-context/`；远端仅执行 `docker load`、宿主机 Atlas migration、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 迁移：远端重启后确认无残留 `atlas` / `flock` 进程，用服务器本机脚本读取 `.env` 并 URL-encode 数据库密码后执行 Atlas；远端 Atlas 从 `20260519093313` 应用到 `20260519105354`，新增 `gateway_models` 上下文字段；迁移后状态 `OK`、待执行 `0`。
- 部署验证：远端当前运行镜像为 `oauth-api-service-server:20260519T192827-5dc73aaf-local-model-context`，容器内 `GIT_SHA_SHORT=5dc73aaf-local`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；公网管理员 `admin/adminadmin` 登录返回成功，`model_list` 返回上下文配置字段和生效字段。
- 清理：清理前远端 `/` 使用率 52%、Docker images 4.73GB；执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260519T174003-48640da6`，回收 348.3MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune。
- 阻塞/风险：首次远端发布命令中 DSN 曾被本地 shell 提前展开为空密码，导致 Atlas 状态检查阶段卡住并需要重启服务器；已改为远端脚本本机读取 `.env` 构造 DSN。现有生产 `.env` 若仍设置 `GATEWAY_CONTEXT_*`，会作为未显式配置模型字段的运维覆盖继续生效，后台模型字段显式保存后则无需重启即可覆盖。
- 补充调整：确认 `2,000 / 800,000` 属于本地测试 K/M 输入留下的模型级显式覆盖，不是默认推荐；已清空本地开发库模型上下文覆盖字段。默认推荐改为 Codex 侧保守 `400K` 窗口、`260K` 开始压缩、`380K` 硬拦截、`1.04M` / `1.9M` 字节阈值；部署模板移除旧 `GATEWAY_CONTEXT_*` 默认值，避免环境变量继续压过后台推荐。
- 补充调整：模型表「上下文窗口 / 压缩阈值 / 字节阈值 / 保留」和上下文弹窗字段均增加问号说明，明确字段含义、`0` 的继承行为、`K` / `M` 单位、字节阈值不是计费 token。
- 补充验证：`cd server && go test ./...`、`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4327 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check` 通过。
- 补充部署：本地构建镜像 `oauth-api-service-server:20260519T202200-5dc73aaf-local` 并上传远端；远端同步新的 `compose.yml`，备份并更新 `.env`，删除旧 `GATEWAY_CONTEXT_*` 显式覆盖，`APP_IMAGE` 切到新镜像后重建 `app-server`；Atlas 状态仍为 `OK`、待执行 `0`。
- 补充部署验证：远端当前运行镜像 `oauth-api-service-server:20260519T202200-5dc73aaf-local`；本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；公网 `model_list` 回归确认 Codex 模型默认生效字段为 `400000 / 260000 / 380000 / 1040000 / 1900000 / 8`。
- 补充清理：清理前远端 `/` 使用率 52%、Docker images 4.73GB；执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260519T192827-5dc73aaf-local-model-context`，回收 348.4MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地镜像 tar 已移入废纸篓。
- 说明优化：模型上下文弹窗顶部、字段 placeholder 和每个输入项下方均明确“留空或 0 表示继承当前生效值，不是无限制”，并展示当前生效值，避免把数据库覆盖值 `0` 误解成关闭限制。
- 说明优化验证：`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4328 NODE_USE_ENV_PROXY=0 pnpm style:l1` 通过。
- 说明优化部署：本地构建镜像 `oauth-api-service-server:20260519T204053-5dc73aaf-local` 并上传远端；远端 `docker load` 后更新 `APP_IMAGE` 并重建 `app-server`，Atlas 状态 `OK`、待执行 `0`；公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`。
- 说明优化部署验证：远端 `model_list` 回归确认模型覆盖字段均为 `0`，生效字段仍为 `400000 / 260000 / 380000 / 1040000 / 1900000 / 8`；发现并清理了生产库 `gpt-5.2.context_keep_items=8` 的历史残值，使所有模型覆盖字段重新回到 `0`。
- 说明优化清理：清理前远端 `/` 使用率 52%、Docker images 4.73GB；执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260519T202200-5dc73aaf-local`，回收 348.4MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地镜像 tar 已移入废纸篓。
- 填值交互：模型上下文弹窗保持输入框默认空值表示继承，不自动写入表格生效值；每个字段新增“填入当前值”按钮，用户明确点击后才把当前生效值填进输入框，避免误把推荐值固化成数据库覆盖。
- 填值交互验证：`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4329 NODE_USE_ENV_PROXY=0 pnpm style:l1` 通过；首次 `style:l1` 在 `/register` 重定向处偶发失败，原命令重跑通过 26 个场景。
- 填值交互部署：本地 Docker / OrbStack 曾短暂不可用，重启 OrbStack 后本地构建镜像 `oauth-api-service-server:20260519T215756-5dc73aaf-local` 并上传远端；远端 `docker load`、更新 `APP_IMAGE`、重建 `app-server`，Atlas 状态 `OK`、待执行 `0`，公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`。
- 填值交互清理：清理前远端 `/` 使用率 52%、Docker images 4.73GB；执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260519T204053-5dc73aaf-local`，回收 348.4MB；清理后 `/` 使用率 50%、Docker images 4.026GB，未执行 volume prune；本地镜像 tar 已移入废纸篓。

## 2026-05-19 API key 级上游策略覆盖
- 完成：新增 `gateway_api_keys.upstream_strategy`，空值表示继承全局默认；可按单个 API key 覆盖为 `backend_only`、`backend_with_cli_fallback` 或 `codex_cli`。后端转发链路改为先取 key 级覆盖，再回退全局默认，上下文预检失败和 usage 记录沿用最终生效模式。
- 完成：`api.key_create` / `api.key_update` / `api.key_list` 接入 `upstream_strategy`；后台 `/admin-keys` 列表新增“上游策略”列，凭据新建 / 编辑弹窗新增“继承全局默认 / Backend 直连 / Backend + CLI 兜底 / 强制 CLI”选择器；`/admin-upstream` 继续只负责全局默认策略。
- 完成：同步更新 `server/docs/api.md`、`web/README.md` 和 `style:l1` mock/断言，覆盖 key 级策略列表展示、弹窗新建态、编辑态回显、暗色模式和桌面/移动端盒模型。
- 验证通过：`cd server && make data`、`cd server && go test ./internal/biz ./internal/data ./internal/server`、`cd server && go test ./...`、`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs`、`cd web && node --check scripts/styleL1.mjs`、`cd web && pnpm test`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4325 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。
- 提交：已提交并推送 `48640da6`（`支持 API key 级上游策略`）。
- 部署：本地构建镜像 `oauth-api-service-server:20260519T174003-48640da6`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260519T174003-48640da6/`；远端仅执行 `docker load`、宿主机 Atlas migration、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 迁移：远端 Atlas 从 `20260518092411` 应用到 `20260519093313`，新增 `gateway_api_keys.upstream_strategy`；迁移后状态 `OK`、待执行 `0`。
- 部署验证：远端当前运行镜像为 `oauth-api-service-server:20260519T174003-48640da6`，容器内 `GIT_SHA_SHORT=48640da6`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；公网管理员 `admin/adminadmin` 登录返回 `code=0`，`api.key_list` 返回 `code=0 total=10` 且 `items[].upstream_strategy` 字段存在；使用现有 key 调用公网 `/v1/models` 返回 `HTTP 200` 和 6 个模型。
- 清理：清理前远端 `/` 使用率 52%、Docker images 4.73GB；执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260519T172757-fb7d27d1`，回收 348.3MB；清理后 `/` 使用率 50%、Docker images 4.026GB；已删除远端本轮 release image tar 包，未执行 volume prune。
- 阻塞/风险：本轮没有改 usage schema 来单独记录“策略来源=全局/key”，排障时可从当前 key 配置和 usage 的最终 `upstream_configured_mode` / `upstream_mode` 判断；如后续需要审计策略来源，应单独扩展 usage 字段。首次远端重建时 shell 中旧 `APP_IMAGE` 覆盖 `.env`，容器仍使用旧镜像；已用显式 `APP_IMAGE=oauth-api-service-server:20260519T174003-48640da6` 重新重建并复核通过。

## 2026-05-19 FontAwesome npm token 暴露处理
- 完成：扫描当前工作区和 Git 历史，未发现 OpenAI / ChatGPT `refresh_token` 或 `auth.json` 明文入库；发现 FontAwesome npm auth token 曾以字面量写入 `web/.npmrc` 和 `web/.yarnrc.yml`。
- 完成：当前工作区已将 FontAwesome npm auth token 改为 `FONTAWESOME_NPM_AUTH_TOKEN` 环境变量引用，避免后续提交继续携带明文 token。
- 完成：项目级 `AGENTS.md` 已补充包管理器、CLI、SDK 和第三方服务配置不得写入字面量 token 的规则，要求相关文件只使用环境变量或示例占位符，并在提交前做当前文件树 secrets 扫描。
- 完成：已在临时 mirror clone 中改写历史并验证 `gitleaks detect --source <mirror> --redact` 无命中；当前主工作区和远端尚未切换到改写后的提交链。
- 下一步：按当前决策暂不处理远端历史，只保证当前文件树和后续提交不再携带明文 token；如旧 token 曾可用，仍建议在 FontAwesome / npm 侧撤销或轮换。
- 阻塞/风险：本轮未执行 force push；远端旧提交历史如已公开，仍可能保留旧明文，历史风险只能通过 token 失效来真正止血。

## 2026-05-19 用量日志视图顺序调整
- 完成：`/admin-usage` 用量日志标签按查看频率调整为「调用明细、异常请求、会话聚合、凭据统计、每日模型」，并将默认视图改为「调用明细」。
- 完成：同步更新前端 README 口径与 `style:l1` 回归断言，覆盖标签顺序、默认激活项、调用明细默认态、筛选、分页、异常请求、会话聚合、凭据统计和每日模型切换。
- 本地补充：排查截图中的失败后确认 `5176` 仍是旧 Vite 进程且开发库 migration 停在 `20260512130053`；已重启 `5176` 前端并执行 `make migrate_apply` 到 `20260518092411`，`usage_session_summaries` 恢复 `code=0`。
- 下一步：如后续增加新的 usage 视图，应继续按排障高频入口优先、聚合统计靠后的顺序维护。
- 阻塞/风险：本轮只调整前端入口顺序和默认激活视图，不改 usage 数据查询、导出、后端 DTO 或数据库字段。

## 2026-05-18 Codex 长上下文错误分类与摘要压缩
- 完成：排查线上 junnan key 的长会话 502，确认根因是上游 `context_length_exceeded` 被网关记成普通 backend 502，且 Codex CLI 对该错误重复重试；长流式请求本身未复现 stream 超时问题。
- 完成：新增 `context_length_exceeded` 错误分类，长上下文不再走 5 次无效重试；超过硬阈值的请求在网关提前拦截为 413，并在 usage 中记录明确错误类型与诊断。
- 完成：新增按 `session_id` 保存的上下文摘要压缩能力，长请求超过压缩阈值时生成工程摘要、保留系统指令和最近消息后再转发，并记录压缩次数、摘要、压缩前后 bytes / token 估算。
- 完成：后台用量明细和会话聚合展示上下文压缩诊断；同步更新 API / 配置文档、前端 README 与生产 `.env.example`。
- 补充：`compose.yml` 已透传 `GATEWAY_STREAM_HEARTBEAT_SECONDS` 和 `GATEWAY_CONTEXT_*` 配置，避免 `.env` 调参不进容器。
- 验证通过：`cd server && go test ./...`、`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`、`cd web && pnpm lint && pnpm css && pnpm test && pnpm build`、`cd web && node --check scripts/styleL1.mjs && pnpm style:l1`。
- 部署：本机构建镜像 `oauth-api-service-server:20260518T175802-ace0afcc-local`，上传到 `8.218.4.199` 后仅执行 `docker load`、宿主机 Atlas migration 和 `docker compose up -d app-server`，未在服务器构建；Atlas 从 `20260512130053` 应用到 `20260518092411`，状态 `OK`、待执行 `0`。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260518T175802-ace0afcc-local`；容器内和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；管理员登录、`api.summary` 和 `usage_session_summaries` 返回 `code=0`。
- 线上验证通过：Windows 机器 `192.168.0.45` 使用测试 key 跑 Codex CLI 自定义 provider，最小请求与 `resume --last` 均返回成功；1,036,105B 长 prompt 返回 `LONG_COMPRESS_OK`，usage 显示本次请求从 1,126,851B 压缩到 63,001B 后成功。
- 线上验证通过：Windows 机器直连 `/v1/responses` 带 `session_id=win-session-compaction-20260518180227` 的 1,103,270B 请求返回 `SESSION_COMPRESS_OK`；生产库 `gateway_context_summaries` 记录压缩 1 次、压缩后 14,800B；后台 `/admin-usage`「会话聚合」页面可见该 session、`1 次压缩`、`265,116 -> 3,513 tokens`、`1,103,270 -> 14,800B` 和摘要。
- 线上验证通过：本机 opencode 1.2.21 使用临时 XDG 配置和测试 key 跑 OpenAI-compatible provider，最小请求返回 `OPENCODE_OK`；`--continue` stdin 长上下文 1,104,061B 返回 `OPENCODE_STDIN_COMPRESS_OK`，usage 显示原始 1,153,490B 自动压缩到 58,016B 后成功。opencode `--file` 附件路径不会完整内联 1.13MB 文件，线上只看到 96KB 请求，因此附件场景不触发网关压缩。
- 清理：清理前远端 `/` 使用率 53%、Docker images 5.433GB；执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未使用旧 app 镜像 `20260518T174705-ace0afcc-local`、`20260513T201322-081f551e`，回收 695.9MB；清理后 `/` 使用率 49%、Docker images 4.025GB；已删除本轮远端 release tar、本地镜像 tar 和 Windows 临时测试目录。
- 剩余风险：当前摘要压缩是确定性工程摘要，不调用模型做语义总结，极端超大请求仍会被硬阈值拦截；Codex CLI 0.125.0 对单次输入有本地 1,048,576 字符限制，超过该限制时请求不会发到网关；Codex App 当前未做无侵入测试，避免切换登录态或 provider 导致当前会话中断。

## 2026-05-13 Compose 入口层与证书续签口径
- 完成：新增可选 `server/deploy/compose/prod/compose.certbot.yml`，提供项目级 one-shot Certbot 服务；服务带 `certbot` profile，不随主 `docker compose up -d` 自动启动，避免把证书续签混入业务主路径。
- 完成：更新 `server/deploy/README.md`、`server/deploy/compose/prod/README.md` 和 `.env.example`，明确主 `compose.yml` 只管业务服务与数据库，`compose.nginx.yml` / `compose.certbot.yml` 仅用于单项目交付、独占机器或入口迁移；测试服务器多项目共存只作为临时运行方式，不沉淀共享入口层真源。
- 下一步：后续触达其他同类项目时按该结构逐步统一，不批量机械改所有项目；若新增发布脚本，应把 nginx/certbot overlay 作为可选步骤而非默认主路径。
- 阻塞/风险：本轮只补项目级可选 Certbot overlay 和文档口径，未在真实服务器申请或续签证书，也未改变当前线上宿主机 Nginx / 续签脚本。

## 2026-05-13 Atlas 线上迁移规则收口
- 完成：将线上 / 低配服务器 Atlas migration 口径写入 `AGENTS.md`、`server/deploy/README.md` 和 `server/deploy/compose/prod/README.md`：统一使用宿主机 `/usr/local/bin/atlas` 与 `flock /tmp/atlas-migrate.lock`，migration 目录随 release 上传，禁止通过 `arigaio/atlas:*` 临时容器或 Compose 服务执行生产迁移。
- 下一步：后续若补自动发布脚本，应把 Atlas 预检、host DSN 和迁移锁做成脚本级门禁。
- 阻塞/风险：本轮只更新本仓库规则和 runbook；仓库当前仍没有正式远端发布脚本，因此脚本层尚未自动拦截 Atlas 容器用法。

## 2026-05-13 OpenCode stream terminate 排查
- 发现：截图中的 `request_id=435ee41334021f2bd599110ad1044d26` 是 `/v1/chat/completions`、`stream=true`、`reasoning_effort=high` 请求，usage 记录 `HTTP 502`、`duration_ms≈125s`、`request_bytes=167189`、`backend_only=true`，诊断为 `upstream_body=context canceled`；本机 OpenCode 日志同类长请求出现 `ECONNRESET` 和 Cloudflare `524 A timeout occurred`。
- 结论：最小 `chat.completions stream` 与 `/v1/responses stream` 线上均能返回 `HTTP 200` 和 `OK`，不是流式格式整体不可用；问题集中在大上下文 / 工具历史等 backend-only 长请求。当前服务只在流开始时发一次首包，随后等待 Codex backend 完整结束，长时间无输出会被 Cloudflare / 客户端 / 代理断开，导致上游 context 被取消，OpenCode UI 表现为 shell `terminated`。
- 修复：`stream=true` 请求等待上游期间新增周期性 SSE keepalive，默认 15 秒，可通过 `GATEWAY_STREAM_HEARTBEAT_SECONDS` 调整；补充 `chat.completions` 慢上游回归，确保结果返回前会持续发送 keepalive。
- 补充修复：Codex 自定义 provider 使用 `/v1/responses` 时可能忽略 SSE comment 级 keepalive；`/v1/responses` 等待上游期间改为发送标准 `response.in_progress` 保活事件，`chat.completions` 仍保留 comment keepalive。
- 补充修复：下游请求上下文取消时记录 `client_canceled`，usage 状态码记为 `499`，不再把客户端 / 入口代理主动断开误归类为 `codex_backend_upstream_failed` 或 Backend 502。
- 验证通过：线上最小 `/v1/chat/completions stream=true`、非流式 chat 和 `/v1/responses stream=true` 均返回 `HTTP 200` / `OK`；本地 `cd server && go test -count=1 ./...`、`git diff --check`、`cd server && make migrate_status` 通过。
- OpenCode 验证：本地服务恢复到当前代码并监听 `127.0.0.1:8400`；使用 `oauth-api-service-local/gpt-5.5`、`--pure`、`low` 执行受控长任务，实际运行约 94 秒，工具命令 `sleep 70 && printf OPENCODE_LONG_OK` 成功，最终输出 `OPENCODE_LONG_OK`，日志未出现 `terminated`、`ECONNRESET`、`AI_APICallError`、`server_is_overloaded` 或 `response.failed`。
- 环境处理：本地开发库应用待执行 migration `20260512130053` 后，usage 查询恢复可用；该 migration 是本轮前已有的 schema 变更，用于支持 `gateway_usage_logs.diagnostic`。
- 下一步：部署后复测长上下文 OpenCode 会话；若仍有 502，需要继续看新 usage 的 `upstream_error_type` / `diagnostic.upstream_body`，区分上游真实 `response.failed/incomplete`、账号限流、模型超上下文、网络断流和客户端主动取消。
- 阻塞/风险：本轮只修复客户端 / Cloudflare 空闲超时导致的断流，不改变 backend-only 请求不能 fallback 到 CLI 的策略；如果 Codex backend 自身在大工具历史上返回失败，仍会按真实上游错误记录为 502。若部署后仍出现 125s 左右取消，下一步应改为实时转发 / 转换 Codex backend 上游 SSE，而不是继续只在等待结果期间做合成保活。

## 2026-05-13 OpenCode stream 修复发布
- 完成：提交并推送 `081f551e`（`修复 OpenCode 长流式请求保活`）；本地构建镜像 `oauth-api-service-server:20260513T201322-081f551e`，构建过程包含前端生产构建和服务端 Linux 二进制构建。
- 验证通过：`cd server && go test -count=1 ./...`、`cd web && pnpm test -- --run`、`cd web && pnpm css`、`cd web && pnpm build`、`cd web && STYLE_L1_PORT=4324 NODE_USE_ENV_PROXY=0 pnpm style:l1`（24 个场景）、`git diff --check`。
- 部署进度：镜像包已上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260513T201322-081f551e/`，远端 `docker load` 已确认加载 `oauth-api-service-server:20260513T201322-081f551e`；migration 文件已上传到同一 release 的 `migrate/` 目录。
- 阻塞：执行 Atlas 容器迁移前检查时，远端 Docker / SSH 开始无响应；随后 SSH 卡在 banner exchange，公网 `/healthz` 与 `/readyz` 均超时。当前未确认完成 `migrate apply`、未更新 Compose `.env` 的 `APP_IMAGE`、未执行 `docker compose up -d app-server`，因此不能视为已完成线上切换。
- 下一步：等服务器 SSH 或云控制台恢复后，先检查 `docker ps`、Atlas 临时容器、`df -h /`、`docker system df` 和当前 app 镜像；确认数据库 migration 状态后再继续更新 `APP_IMAGE=oauth-api-service-server:20260513T201322-081f551e`、重启 `app-server`、验证 `/healthz` / `/readyz` / OpenCode 最小调用。禁止在确认现场前重启 Docker 或清理 volume。
- 补充排查：服务器重启后线上旧镜像 `oauth-api-service-server:20260512T212250-5eb24e4d-local` 自动恢复，公网 `/healthz` 与 `/readyz` 正常；`openai-oauth-api-service-server` 当前 `OOMKilled=false`，内存约 17MiB / 900MiB。今天 20:17 之后内核日志没有记录具体 OOM kill，但 `systemd-journald` 持续报告 `Under memory pressure`，时间点与拉取 / 运行 `arigaio/atlas:latest` 容器迁移检查重合；截图中的 `python3/gunicorn` OOM 记录来自 5 月 10 日历史 Docker 容器，不是本项目 Go 服务。
- 处理：已删除本次引入的 `arigaio/atlas:latest` 镜像，未清理 volume，保留已 load 的新业务镜像 `oauth-api-service-server:20260513T201322-081f551e`。后续迁移禁止在低配服务器上拉起 Atlas 容器，应改为上传轻量 `atlas` 二进制、使用已验证的轻量迁移方式，或在资源更充足环境执行后再远端只做加载 / 重启。
- 完成部署：按固定版本安装 `/usr/local/bin/atlas v1.1.0`，sha256 校验通过；宿主机使用 `127.0.0.1:5433` 连接数据库，`atlas migrate status` 显示当前版本 `20260512130053`、无 pending。已更新远端 `.env` 的 `APP_IMAGE=oauth-api-service-server:20260513T201322-081f551e` 并重启 `app-server`，线上容器版本为 `081f551e26a9a24bbabd06de2c0e7d72cf114ed4`。
- 部署验证：远端本机与公网 `/healthz`、`/readyz` 均返回 200；`opencode models oauth-api-service` 返回 `oauth-api-service/gpt-5.5`；重启 mihomo 后恢复 `172.19.0.1:7890` 监听，`opencode run --pure -m oauth-api-service/gpt-5.5 --variant low --format json '只回复 OK'` 返回 `OK`，未出现 `error`、`terminated`、`ECONNRESET` 或 `response.failed`。部署后 `openai-oauth-api-service-server` 约 42MiB / 900MiB，`OOMKilled=false`，根分区 48%，保留上一版镜像用于回滚，仅执行 `docker builder prune -f`（回收 0B），未执行 volume prune。
- 额外发现：重启后 mihomo 早于 Docker bridge 就绪启动，导致 `listen tcp 172.19.0.1:7890: bind: cannot assign requested address`，app-server 代理请求报 `connect: connection refused`；手动 `systemctl restart mihomo` 后恢复。后续应给 mihomo service 增加 Docker network 就绪依赖或启动前等待 `172.19.0.1`，避免服务器重启后代理入口缺失。


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
- 部署：本地构建镜像并上传到远端 release 目录；远端仅执行 `docker load`、备份并更新 Compose 镜像环境变量、`docker compose up -d app-server`，未在服务器构建。
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

## 2026-05-12 客户端模板与用量筛选提交部署
- 提交：已提交并推送 `8957746 完善客户端模板与用量筛选` 到 `origin/main`。
- 验证通过：提交前执行 `cd server && go test ./internal/server ./internal/biz ./internal/data`、`cd web && pnpm test && pnpm style:l1 && pnpm build`。
- 部署：按低配服务器发布约定在本机构建镜像 `oauth-api-service-server:20260512T184202-8957746e`，上传到 `8.218.4.199` 后仅执行 `docker load` 和 `docker compose up -d app-server`，未在服务器构建。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T184202-8957746e`；容器内与公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；`/admin-client-config` 已返回新前端资源 `index.DYXQUfGa.js`；管理员 `admin/adminadmin` 登录、`summary` 和 `usage_list` 携带 `key_ids` 请求均返回 `code=0`。
- 清理：部署前记录远端 `/` 使用率 48%、Docker images 4.71GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260512T143300-bc28db5-local`，回收 347.6MB；清理后 `/` 使用率 47%、Docker images 4.007GB；已删除远端和本地本轮镜像 tar 包。

## 2026-05-12 Codex stream 首字节与 summary 能力声明收敛
- 完成：`/v1/responses` 的 `stream=true` 请求现在进入上游 Codex backend/CLI 前先返回并 flush `response.created`，避免 Cloudflare / 客户端在长时间首字节空窗中误判上游失败；首字节后上游失败时改为在 SSE 内返回 `response.failed` 和 `[DONE]`。
- 完成：`/v1/chat/completions` 的流式请求也会先发送 SSE comment keepalive，降低纯聊天流的首字节等待风险；后续仍保持原有最终 chunk 输出方式。
- 完成：模型元数据将 `supports_reasoning_summaries` 收敛为 `false`，明确当前自定义 provider 不承诺 Codex reasoning summary 过程事件透传，避免客户端误判能力。
- 文档：同步更新 `server/docs/api.md`，说明 stream 首字节、SSE 内错误和 reasoning summary 能力边界。
- 验证通过：`cd server && go test ./internal/server ./internal/biz`、`cd server && go test ./internal/server ./internal/biz ./internal/data`、`git diff --check`。本轮不做完整 reasoning summary 事件透传，过程展示仍不是目标能力。


## 2026-05-12 Codex 上游错误类型与日志补充
- 完成：上游失败时按 backend / CLI 细分 `upstream_error_type`，覆盖 backend 鉴权、限流、HTTP 5xx、超时、response failed / incomplete、流错误，以及 CLI 超时、二进制缺失、空回复等场景，失败 usage 不再只落 `codex_backend_upstream_failed` / `codex_cli_upstream_failed` 粗粒度类型。
- 完成：网关在上游失败时写入包含 request_id、mode、endpoint、model、error_type 的 warn 日志，便于直接关联 usage 与服务日志；stream 已发首字节后仍在 SSE 内返回细分错误码。
- 文档：同步更新 `server/docs/api.md` 的 usage 记录说明，列出常见上游错误类型。
- 验证通过：`cd server && go test ./internal/server ./internal/biz ./internal/data`、`git diff --check`。

## 2026-05-12 Codex stream 与错误类型部署验证
- 补充修复：发现自定义 HTTP 观测包装器 `statusCapturingResponseWriter` 未透传 `http.Flusher`，会导致 `/v1/responses` 虽然写了 `Flush()` 但到 app-server 实际响应仍带 `Content-Length` 并被缓冲；已为 wrapper 增加 `Flush()` 透传并补测试，确保 SSE handler 能拿到 flusher。
- 验证通过：本地执行 `cd server && go test ./internal/server ./internal/biz ./internal/data`、`cd web && pnpm test`、`cd web && pnpm style:l1`、`cd web && pnpm build`、`git diff --check`。
- 部署：按低配服务器发布约定，本机构建镜像 `oauth-api-service-server:20260512T201859-5eb24e4d-local`，上传到 `8.218.4.199` 后仅执行 `docker load` 与 `docker compose up -d app-server`，未在服务器构建。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T201859-5eb24e4d-local`；公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；管理员 `admin/adminadmin` 登录与 `api.summary` 返回 `code=0`；`/v1/models` 的所有 Codex 模型均返回 `supports_reasoning_summaries=false`。
- 线上验证通过：直连 app-server `http://8.218.4.199:8400/v1/responses stream=true` 首行 56ms 返回 `response.created`，响应头无 `Content-Length`，随后包含 `response.completed`、`[DONE]` 和 `OK`；公网 Cloudflare 入口也返回 `response.created` / `response.completed` / `[DONE]` / `OK`。
- 日志检查：部署后 5 分钟内 `docker logs` 未发现 `ERROR` / `WARN` / `panic` / `fatal`。
- 清理：部署前记录远端 `/` 使用率 51%、Docker images 5.413GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未被容器使用的旧镜像 `20260512T184202-8957746e` 与中间测试镜像 `20260512T195841-5eb24e4d-local`，回收 695.3MB；清理后 `/` 使用率 47%、Docker images 4.007GB；已删除远端 release image tar 包。

## 2026-05-12 usage 错误类型说明补充
- 完成：后台最近请求、用量明细和会话请求明细的“错误”列保留原始 `error_type`，并对已知 Codex backend / CLI 错误码展示简短中文说明；表头新增说明，提示该字段来自 usage 错误类型。
- 完成：新增前端共享 `gatewayErrorTypes` 说明映射，避免多个表格分散维护错误码文案。
- 文档：`server/docs/api.md` 新增“上游错误类型”表，列出 backend 鉴权、限流、5xx、超时、response failed / incomplete、流中断，以及 CLI 超时、二进制缺失、空输入、空回复等错误码含义。
- 验证通过：`cd web && pnpm test`、`cd web && pnpm style:l1`、`cd web && pnpm build`。`git diff --check` 待最终差异整理后执行。

## 2026-05-12 上游失败诊断与异常筛选
- 完成：为失败 usage 增加 `diagnostic` 元数据，记录请求 / 响应字节、backend-only、fallback enabled / blocked、reasoning effort、上游 HTTP 状态和脱敏上游错误摘要；不保存 prompt、response body 或模型输出正文。
- 完成：`api.usage_list` 支持 `upstream_error_type` 筛选并返回 `diagnostic` / `diagnostic_summary`；`gateway_usage_logs` 新增 `diagnostic` JSONB 字段和 `upstream_error_type + created_at` 索引，迁移文件为 `20260512130053_migrate.sql`。
- 完成：后台「用量日志」增加「上游错误类型」筛选和「诊断」列，异常请求页可直接查看 backend-only / fallback 阻断 / 上游 HTTP 状态 / 错误摘要，减少排查时必须 SSH 翻日志的次数。
- 文档：同步更新 `server/docs/api.md` 与 `web/README.md`，明确诊断字段边界与筛选参数。
- 下一步：部署前需执行 Atlas migration；当前实现仍只保存脱敏摘要，若要保存更详细采样必须单独设计开关、TTL 与脱敏策略。
- 验证通过：`cd server && go test ./internal/server ./internal/biz ./internal/data`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminApi/index.jsx scripts/styleL1.mjs src/common/utils/gatewayErrorTypes.js && node --check scripts/styleL1.mjs && pnpm test -- --run`、`cd web && pnpm css && pnpm build && STYLE_L1_PORT=4324 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。

## 2026-05-12 usage 错误说明与诊断部署
- 修复：部署前补齐 `diagnostic` RPC 映射为普通 map，避免 `structpb.NewStruct` 不能直接序列化 Go struct，`cd server && go test ./internal/server ./internal/biz ./internal/data` 已通过。
- 验证通过：本地执行 `cd server && go test ./internal/server ./internal/biz ./internal/data`、`cd web && pnpm test`、`cd web && pnpm style:l1`、`cd web && pnpm build`、`git diff --check`。
- 部署：按低配服务器发布约定在本机构建镜像 `oauth-api-service-server:20260512T211049-5eb24e4d-local`，上传到 `8.218.4.199` 后仅执行 `docker load`、Atlas migration 和 `docker compose up -d app-server`，未在服务器构建。
- 迁移：远端 Atlas 从 `20260511033926` 应用到 `20260512130053`，新增 `gateway_usage_logs.diagnostic` JSONB 字段与 `upstream_error_type, created_at` 索引；迁移后状态 `OK`、待执行 `0`。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T211049-5eb24e4d-local`；容器内与公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；`/admin-api` 返回新前端资源 `index.Br-LtUh6.js`；管理员 `admin/adminadmin` 登录、`api.summary` 和带 `upstream_error_type` 的 `api.usage_list` 均返回 `code=0`。
- 日志检查：部署后 5 分钟内 `docker logs` 未发现 `ERROR` / `WARN` / `panic` / `fatal`；仅有本轮验证用 JSON-RPC info 日志。
- 清理：部署前记录远端 `/` 使用率 47%、Docker images 4.007GB；迁移拉取临时 Atlas 镜像后清理前 `/` 使用率 49%、Docker images 4.885GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未使用旧 app 镜像 `20260512T201859-5eb24e4d-local` 与临时 `arigaio/atlas:latest`，回收 386.1MB；清理后 `/` 使用率 46%、Docker images 4.007GB；已删除远端和本地本轮镜像 / migration tar 包。

## 2026-05-12 上游失败诊断部署验证
- 部署：按低配服务器发布约定，本机构建镜像 `oauth-api-service-server:20260512T212250-5eb24e4d-local` 并上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260512T212250-5eb24e4d-local/`；远端只执行 `docker load`、Atlas migration、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d app-server`，未在服务器构建。
- migration：已同步本地 `server/internal/data/model/migrate/` 到远端 `/data/openai-oauth-api-service/migrate/` 并通过 Atlas 容器执行，状态为 `Current Version: 20260512130053`、`Pending Files: 0`；生产库 `gateway_usage_logs.diagnostic` 字段存在且类型为 `jsonb`。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260512T212250-5eb24e4d-local`，启动日志 service.version 为 `5eb24e4de4e3210a227a1f2109c56024de0dfa4e-local`；远端本机与公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；公网 `/admin-login` 返回 `HTTP 200` 且前端资源为 `assets/index.CE7ihMBu.js`。
- 线上验证通过：管理员 `admin/adminadmin` 本机 RPC 登录成功；`api.usage_list` 携带 `upstream_error_type=codex_backend_http_5xx` 返回 `code=0`；真实创建临时 key `diagnostictmp` 后调用 `/v1/chat/completions` 返回 `HTTP 200`，随后 `usage_list key_id=13` 返回 `diagnostic` 与 `diagnostic_summary` 字段，临时 key 已删除。
- 清理：部署前记录远端 `/` 使用率 48%、Docker images 4.885GB；已删除远端 release tar 包，执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260512T211049-5eb24e4d-local` 和临时 Atlas 镜像，回收 386.2MB；清理后 `/` 使用率 46%、Docker images 4.007GB，未执行 volume prune。
- 日志检查：最终健康检查后 2 分钟内未发现 `ERROR` / `WARN` / `panic` / `fatal`。较早验证期间存在一次手工传参错误产生的 `bad param` WARN，已确认不是新镜像启动或主链路异常。

## 2026-05-19 长上下文压缩与用量入口部署
- 完成：提交并推送 `86bd2f6 修复长上下文压缩与用量日志入口`；历史中旧 FontAwesome token、示例 downstream key 和误报镜像标签已重写为占位符，`gitleaks detect --redact --source .` 检查通过。
- 验证通过：本地执行 `cd server && go test ./...`、`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/ scripts/styleL1.mjs && pnpm css && pnpm test`、`cd web && pnpm build`、`STYLE_L1_PORT=4324 NODE_USE_ENV_PROXY=0 pnpm style:l1`、`git diff --check`。
- 本地回归：本地应用数据库已应用 `20260518092411` migration，`/admin-usage` 不再出现 API 操作失败；用量日志页默认入口和标签顺序为「调用明细、异常请求、会话聚合、凭据统计、每日模型」。
- 部署：按低配服务器发布约定，本机构建镜像 `oauth-api-service-server:20260519T081406-86bd2f65-local`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260519T081406-86bd2f65-local/`；远端只执行 `docker load`、宿主机 Atlas migration、更新 Compose `.env` 的 `APP_IMAGE`、`docker compose up -d --no-deps --force-recreate app-server`，未在服务器构建，也未改管理员密码。
- 迁移：远端 Atlas 状态为 `Current Version: 20260518092411`、`Pending Files: 0`，无待执行 migration。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260519T081406-86bd2f65-local`，容器环境 `GIT_SHA=86bd2f65c86c4f93bcf3820587997018ec0fe059`；远端本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；公网 `/admin-usage` 返回 `HTTP 200`；管理员 `admin/adminadmin` 登录后，`api.usage_list` 返回 `code=0`、`total=28`、`items=8`，`api.usage_session_summaries` 返回 `code=0`、`total=1`、`items=1`。
- 清理：部署前记录远端 `/` 使用率 51%、Docker images 4.73GB；执行 `docker image prune -a -f` 与 `docker builder prune -f`，删除未被容器使用的旧镜像 `oauth-api-service-server:20260518T175802-ace0afcc-local`，回收 348.2MB；清理后 `/` 使用率 49%、Docker images 4.025GB，未执行 volume prune。
- 下一步：远端 release 目录仍保留本轮镜像 tar 和 migration 文件，便于短期回溯；如确认无需保留，可后续只删除该 release 下的 tar 包。

## 2026-05-19 Codex 余额公开查询接口
- 完成：新增 `GET /public/codex/balance`，不要求管理员登录或下游 `ogw_` key；服务端按请求启动 Codex app-server，调用 `account/rateLimits/read` 后只返回限额窗口、剩余百分比、重置时间、plan type 和 credits，不返回账号邮箱或 token。
- 完成：新增 `CODEX_APP_SERVER_BIN`、`CODEX_BALANCE_TIMEOUT_SECONDS` 与 `CODEX_BALANCE_CACHE_SECONDS` 配置；公开接口默认 30 秒内存缓存，避免无登录查询反复拉起 Codex app-server 子进程。Compose `.env.example` / `compose.yml`、README、`server/docs/api.md` 和 `server/docs/config.md` 已同步说明公开查询边界。
- 下一步：如线上不希望任何人看到余额 / 限额百分比，应在 Nginx / Cloudflare 层给 `/public/codex/balance` 加 IP allowlist 或独立查询 token；本轮按需求保持应用层免登录。
- 阻塞/风险：该接口依赖服务器 Codex CLI 的 `app-server` 子命令和服务器 Codex 登录态；如果容器内未安装支持 app-server 的 Codex CLI，或 `auth.json` 失效，接口会返回 `codex_balance_query_failed`。

## 2026-05-19 Codex 余额公开查询部署中断
- 已完成：本地构建镜像 `oauth-api-service-server:20260519T164255-6f72a8ec-balance-local`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260519T164255-6f72a8ec-balance-local/`，远端执行 `docker load`、宿主机 Atlas status 和 compose 配置更新；Atlas 状态为 `Current Version: 20260518092411`、`Pending Files: 0`，无 schema 变更。
- 已发现并处理：首次 `docker compose up -d --no-deps --force-recreate app-server` 后容器仍运行旧镜像 `oauth-api-service-server:20260519T081406-86bd2f65-local`，导致 `/public/codex/balance` 返回前端 HTML；随后重新上传镜像包、重新 `docker load`，删除旧 `app-server` 容器并由 compose 重新创建，新容器曾通过远端本机 `/readyz`。
- 当前阻塞：重新创建新容器后，SSH 与 HTTP 端口均能建立 TCP 连接，但 SSH 在 banner 前超时，`http://8.218.4.199:8400/healthz` 连接后无响应；因此未能完成运行镜像确认、公开 `/public/codex/balance` 验证、部署后日志检查和最终镜像清理。
- 下一步：恢复 SSH 后优先检查宿主机负载、Docker daemon、`openai-oauth-api-service-server` 容器状态和日志；若新容器异常，应回滚到 `oauth-api-service-server:20260519T081406-86bd2f65-local` 后再重新发布。

## 2026-05-19 本地公开接口代理补齐
- 完成：Vite 本地开发代理新增 `/public` 到后端 `apiProxyTarget`，避免访问 `http://localhost:5176/public/codex/balance` 时被前端 dev server 当作静态 / SPA 路径处理。
- 验证：本地后端 `http://localhost:8400/public/codex/balance` 已返回 Codex 余额 JSON；当前 5176 端口未监听，需要启动前端 dev server 后再通过 5176 代理验证。

## 2026-05-19 Codex 余额后台可视化
- 完成：新增后台页面 `/admin-codex-balance` 和侧边栏「用量统计 / Codex 余额」入口，登录管理员可查看接口状态、credits、更新时间、Codex 与 GPT-5.3-Codex-Spark 的 5 小时 / 每周剩余额度进度条，并支持手动刷新。
- 完成：页面直接读取 `/public/codex/balance`，只展示服务端裁剪后的余额与限额信息；不新增后台 RPC 和数据库字段。
- 验证通过：`pnpm exec eslint --ext .js --ext .jsx src/pages/AdminCodexBalance/index.jsx src/App.jsx src/common/components/layout/AdminFrame.jsx`、`pnpm test`、`pnpm build`、`git diff --check`。
- 浏览器回归：本地 `http://localhost:5176/admin-codex-balance` 通过 Playwright 验证桌面 1440x900、移动 390x844、暗色模式；页面渲染正常、4 条进度条可见、无横向溢出。控制台仅有项目既存 React Router v7 future warning。

## 2026-05-19 Codex 余额正式部署
- 提交：已提交并推送 `fb7d27d 新增 Codex 余额查询与后台展示` 到 `origin/main`；未把本地无关 `AGENTS.md` 改动纳入提交。
- 验证通过：提交前执行 `cd server && go test ./...`、`cd web && pnpm exec eslint --ext .js --ext .jsx src/pages/AdminCodexBalance/index.jsx src/App.jsx src/common/components/layout/AdminFrame.jsx && pnpm test && pnpm build`、`git diff --check`。
- 部署：按低配服务器发布约定在干净 worktree 本机构建镜像 `oauth-api-service-server:20260519T172757-fb7d27d1`，上传到 `8.218.4.199:/data/openai-oauth-api-service/releases/20260519T172757-fb7d27d1/`；远端只执行 `docker load`、宿主机 Atlas status、更新 Compose `.env`、重建 `app-server`，未在服务器构建，也未改管理员密码。
- 迁移：远端 Atlas 状态为 `Current Version: 20260518092411`、`Pending Files: 0`，本轮无 schema 变更。
- 线上验证通过：远端当前 `app-server` 运行镜像为 `oauth-api-service-server:20260519T172757-fb7d27d1`，容器环境 `GIT_SHA=fb7d27d19cdf177dc210cd90095c437e30f3f2a2`；本机和公网 `/healthz` 返回 `ok`、`/readyz` 返回 `ready`；本机和公网 `/public/codex/balance` 均返回 `status=ok`，credits 为 `0`，公开页面 `/admin-codex-balance` 返回 `HTTP 200`。
- 浏览器回归：公网 `https://oauth-api.saurick.me/admin-codex-balance` 通过 Playwright 登录管理员后验证，页面可见接口状态「正常」、Credits remaining、Codex 与 GPT-5.3-Codex-Spark 两张卡和 4 条额度进度条；桌面 1440x900 无横向溢出，控制台无错误。
- 清理：部署前远端 `/` 使用率 53%、Docker images 5.434GB；执行 `docker image prune -a -f` 和 `docker builder prune -f`，删除未使用旧镜像 `20260519T081406-86bd2f65-local` 与 `20260519T164255-6f72a8ec-balance-local`，回收 696.5MB；清理后 `/` 使用率 50%、Docker images 4.026GB；已删除远端本轮 release image tar 包，未执行 volume prune。

## 2026-05-20 Windows API 凭据表格样式修复
- 完成：修复 `/admin-keys` API 凭据表格在 Windows 下“完整凭据”列被压窄后逐字符竖排的问题；该表改为固定列宽，并为完整凭据列使用 `overflow-wrap:anywhere` + `word-break:normal`，避免 `break-all` 在窄列中产生单字符换行。
- 完成：`style:l1` 的 API 凭据 mock 数据补充超长连续 key，并新增表格布局、完整凭据列宽/高度、复制按钮尺寸和换行策略断言，覆盖 Windows 字体/滚动条差异导致的回归。
- 下一步：执行前端 lint/css/test/build 与 `style:l1` 回归；重点确认浅色、暗色、桌面和移动视口下 API 凭据表默认态、选择态、弹窗和分页交互不受影响。
- 阻塞/风险：本地 PowerShell 环境无 `pnpm`，需通过 WSL 执行前端命令；当前仓库 `.npmrc` 读取时会提示 FontAwesome token 环境变量占位符未注入，若依赖未完整安装可能需要先配置环境变量或安装依赖。

## 2026-05-20 本机 dev_restart 与 pnpm start 修复
- 完成：本机后端  失败根因为公共 dev 配置命中  共享 PG 且密码失效；已在本机 PostgreSQL 18 的  创建/复用  库和 ，写入未跟踪的 ，并执行 Atlas migration 到 。
- 完成：前端  ERR_PNPM_NO_IMPORTER_MANIFEST_FOUND  No package.json (or package.yaml, or package.json5) was found in "/root/projects/openai-oauth-api-service". 启动时会因  /  引用未注入的  产生本机告警；当前项目没有  依赖，已移除 FontAwesome 私有 registry 配置，避免无关 token 占位符影响本机启动。
- 验证通过：postgres://test_user:***@127.0.0.1:5433/openai_oauth_api_service?sslmode=disable 输出本机脱敏 DSN，using DB_URL=postgres://test_user:***@127.0.0.1:5433/openai_oauth_api_service?sslmode=disable
No migration files to execute 成功应用 15 个 migration；>>> stopping dev listeners on: 8400 9400
>>> kill port 8400 via lsof: 4126959
>>> building ./bin/server-dev from ./cmd/server
>>> running: ./bin/server-dev
using conf path: ./configs/dev/config.yaml
{"caller":"server/main.go:240","level":"INFO","mode":"local","msg":"tracer provider initialized without remote exporter","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_ratio":0,"trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"server/main.go:116","level":"INFO","msg":"postgres dsn overridden from env","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"server/main.go:125","level":"INFO","msg":"jwt secret overridden from env","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"server/main.go:133","level":"INFO","msg":"admin username overridden from env","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"server/main.go:137","level":"INFO","msg":"admin password overridden from env","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"data/data.go:108","level":"INFO","logger.name":"data","msg":"init postgres(otelsql) start...","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"data/data.go:150","level":"INFO","logger.name":"data","msg":"init postgres(otelsql) done","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"data/admin_user_init.go:63","level":"INFO","logger.name":"data","msg":"sync admin_users admin success","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[gpt-5.5]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Query","query":"SELECT \"gateway_models\".\"id\" FROM \"gateway_models\" WHERE \"gateway_models\".\"model_id\" = $1 LIMIT 1","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[gpt-5.4]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Query","query":"SELECT \"gateway_models\".\"id\" FROM \"gateway_models\" WHERE \"gateway_models\".\"model_id\" = $1 LIMIT 1","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[gpt-5.4-mini]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Query","query":"SELECT \"gateway_models\".\"id\" FROM \"gateway_models\" WHERE \"gateway_models\".\"model_id\" = $1 LIMIT 1","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[gpt-5.3-codex]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Query","query":"SELECT \"gateway_models\".\"id\" FROM \"gateway_models\" WHERE \"gateway_models\".\"model_id\" = $1 LIMIT 1","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[gpt-5.3-codex-spark]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Query","query":"SELECT \"gateway_models\".\"id\" FROM \"gateway_models\" WHERE \"gateway_models\".\"model_id\" = $1 LIMIT 1","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[gpt-5.2]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Query","query":"SELECT \"gateway_models\".\"id\" FROM \"gateway_models\" WHERE \"gateway_models\".\"model_id\" = $1 LIMIT 1","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Query","query":"SELECT \"gateway_models\".\"id\", \"gateway_models\".\"model_id\", \"gateway_models\".\"owned_by\", \"gateway_models\".\"created_unix\", \"gateway_models\".\"enabled\", \"gateway_models\".\"source\", \"gateway_models\".\"context_window_tokens\", \"gateway_models\".\"context_compact_tokens\", \"gateway_models\".\"context_hard_tokens\", \"gateway_models\".\"context_compact_bytes\", \"gateway_models\".\"context_hard_bytes\", \"gateway_models\".\"context_keep_items\", \"gateway_models\".\"last_seen_at\", \"gateway_models\".\"created_at\", \"gateway_models\".\"updated_at\" FROM \"gateway_models\"","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[gpt-5.5 gpt-5.4 gpt-5.4-mini gpt-5.3-codex gpt-5.3-codex-spark gpt-5.2]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Exec","query":"DELETE FROM \"gateway_model_prices\" WHERE \"gateway_model_prices\".\"model_id\" NOT IN ($1, $2, $3, $4, $5, $6)","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[* gpt-5.5 gpt-5.4 gpt-5.4-mini gpt-5.3-codex gpt-5.3-codex-spark gpt-5.2]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Exec","query":"DELETE FROM \"gateway_policies\" WHERE \"gateway_policies\".\"model_id\" \u003c\u003e $1 AND \"gateway_policies\".\"model_id\" NOT IN ($2, $3, $4, $5, $6, $7)","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"args":"[]","caller":"dialect/dialect.go:79","level":"DEBUG","msg":"driver.Query","query":"SELECT \"gateway_api_keys\".\"id\", \"gateway_api_keys\".\"owner_user_id\", \"gateway_api_keys\".\"name\", \"gateway_api_keys\".\"key_hash\", \"gateway_api_keys\".\"plain_key\", \"gateway_api_keys\".\"key_prefix\", \"gateway_api_keys\".\"key_last4\", \"gateway_api_keys\".\"disabled\", \"gateway_api_keys\".\"upstream_strategy\", \"gateway_api_keys\".\"quota_requests\", \"gateway_api_keys\".\"quota_total_tokens\", \"gateway_api_keys\".\"quota_daily_tokens\", \"gateway_api_keys\".\"quota_weekly_tokens\", \"gateway_api_keys\".\"quota_daily_input_tokens\", \"gateway_api_keys\".\"quota_weekly_input_tokens\", \"gateway_api_keys\".\"quota_daily_output_tokens\", \"gateway_api_keys\".\"quota_weekly_output_tokens\", \"gateway_api_keys\".\"quota_daily_billable_input_tokens\", \"gateway_api_keys\".\"quota_weekly_billable_input_tokens\", \"gateway_api_keys\".\"allowed_models\", \"gateway_api_keys\".\"last_used_at\", \"gateway_api_keys\".\"created_at\", \"gateway_api_keys\".\"updated_at\" FROM \"gateway_api_keys\"","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"data/token.go:37","level":"INFO","module":"data.token","msg":"token generator init ok, expire=168h0m0s","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"data/admin_token.go:34","level":"INFO","module":"data.admin_token","msg":"admin token generator init ok, expire=168h0m0s","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"data/jsonrpc.go:83","level":"INFO","module":"data.jsonrpc","msg":"JsonrpcData created (auth/admin auth/user admin usecases constructed inside)","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"server/http_custom_handlers.go:248","level":"INFO","msg":"http static dir not found or not dir: /app/public, skip static handler","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:32+08:00"}
{"caller":"grpc/server.go:212","level":"INFO","msg":"[gRPC] server listening on: [::]:9400","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:33+08:00"}
{"caller":"http/server.go:330","level":"INFO","msg":"[HTTP] server listening on: [::]:8400","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:02:33+08:00"}
{"caller":"grpc/server.go:224","level":"INFO","msg":"[gRPC] server stopping","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:21:47+08:00"}
{"caller":"http/server.go:345","level":"INFO","msg":"[HTTP] server stopping","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:21:47+08:00"}
{"caller":"taskgroup/taskgroup.go:304","component":"taskgroup","level":"INFO","msg":"taskgroup stop begin","request_id":"","running_count":0,"service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","timeout_ms":30000,"trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:21:47+08:00","wait":true}
{"caller":"taskgroup/taskgroup.go:304","component":"taskgroup","level":"INFO","msg":"taskgroup stop finished before timeout","request_id":"","running_count":0,"service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","timeout_ms":30000,"trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:21:47+08:00","wait":true}
{"caller":"taskgroup/taskgroup.go:304","canceled_count":0,"component":"taskgroup","level":"INFO","msg":"taskgroup dispatched cancellation to remaining tasks","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:21:47+08:00","wait":true}
{"caller":"config/config.go:67","level":"INFO","msg":"watcher's ctx cancel : context canceled","request_id":"","service.id":"simon","service.name":"openai-oauth-api-service-server","service.version":"7b0f361","span.id":"","task.id":"","trace.id":"","trace_link_id":"","trace_sampled":false,"ts":"2026-05-20T14:21:47+08:00"} 可启动并监听 ，管理员  登录和  返回成功。
- 验证通过：
> openai-oauth-api-service-web@1.0.0 start /root/projects/openai-oauth-api-service/web
> vite -- --host 127.0.0.1 --no-open

env = {
  VITE_BASE_URL: '/',
  VITE_ENABLE_RPC_MOCK: 'false',
  VITE_APP_TITLE: 'OpenAI OAuth API Service',
  HTTPS_PROXY: 'http://127.0.0.1:7897',
  no_proxy: '172.31.*,172.30.*,172.29.*,172.28.*,172.27.*,172.26.*,172.25.*,172.24.*,172.23.*,172.22.*,172.21.*,172.20.*,172.19.*,172.18.*,172.17.*,172.16.*,10.*,192.168.*,127.*,localhost,<local>',
  USER: 'root',
  npm_config_user_agent: 'pnpm/10.33.2 npm/? node/v24.14.0 linux x64',
  npm_node_execpath: '/usr/local/lib/nodejs/node-v24.14.0-linux-x64/bin/node',
  SHLVL: '1',
  HOME: '/root',
  OLDPWD: '/root/projects/openai-oauth-api-service',
  npm_config_force: '',
  NO_PROXY: '172.31.*,172.30.*,172.29.*,172.28.*,172.27.*,172.26.*,172.25.*,172.24.*,172.23.*,172.22.*,172.21.*,172.20.*,172.19.*,172.18.*,172.17.*,172.16.*,10.*,192.168.*,127.*,localhost,<local>',
  npm_package_json: '/root/projects/openai-oauth-api-service/web/package.json',
  COREPACK_ROOT: '/usr/local/lib/nodejs/node-v24.14.0-linux-x64/lib/node_modules/corepack',
  DBUS_SESSION_BUS_ADDRESS: 'unix:path=/run/user/0/bus',
  WSL_DISTRO_NAME: 'Ubuntu-26.04',
  npm_config_progress: 'true',
  https_proxy: 'http://127.0.0.1:7897',
  WAYLAND_DISPLAY: 'wayland-0',
  COREPACK_ENABLE_DOWNLOAD_PROMPT: '1',
  LOGNAME: 'root',
  pnpm_config_verify_deps_before_run: 'false',
  http_proxy: 'http://127.0.0.1:7897',
  PULSE_SERVER: 'unix:/mnt/wslg/PulseServer',
  WSL_INTEROP: '/run/WSL/4128427_interop',
  NAME: 'simon',
  _: '/usr/local/bin/pnpm',
  npm_config_package_import_method: 'hardlink',
  npm_config_registry: 'http://registry.npm.taobao.org/',
  npm_config_node_linker: 'hoisted',
  TERM: 'xterm-256color',
  npm_config_node_gyp: '/root/.cache/node/corepack/v1/pnpm/10.33.2/dist/node_modules/node-gyp/bin/node-gyp.js',
  PATH: '/root/projects/openai-oauth-api-service/web/node_modules/.bin:/root/.cache/node/corepack/v1/pnpm/10.33.2/dist/node-gyp-bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/usr/lib/wsl/lib:/mnt/c/Program Files/PowerShell/7:/mnt/c/Users/sauri/.codex/tmp/arg0/codex-arg01LdnQ8:/mnt/c/Program Files/OpenSSH/:/mnt/c/Program Files/Common Files/Oracle/Java/javapath:/mnt/c/Windows/system32:/mnt/c/Windows:/mnt/c/Windows/System32/Wbem:/mnt/c/Windows/System32/WindowsPowerShell/v1.0/:/mnt/c/Windows/System32/OpenSSH/:/mnt/c/Program Files (x86)/NVIDIA Corporation/PhysX/Common:/mnt/c/Program Files/Git/cmd:/mnt/c/Program Files/Microsoft SQL Server/150/Tools/Binn/:/mnt/c/Program Files/dotnet/:/mnt/c/Program Files (x86)/GtkSharp/2.12/bin:/mnt/c/Program Files/nodejs/:/mnt/c/Program Files/Go/bin:/mnt/c/Program Files/NVIDIA Corporation/NVIDIA app/NvDLISR:/mnt/c/Users/sauri/AppData/Local/Programs/Cursor/resources/app/bin:/mnt/c/WINDOWS/system32:/mnt/c/WINDOWS:/mnt/c/WINDOWS/System32/Wbem:/mnt/c/WINDOWS/System32/WindowsPowerShell/v1.0/:/mnt/c/WINDOWS/System32/OpenSSH/:/mnt/c/Program Files/Tailscale/:/mnt/c/Program Files/Sunshine:/mnt/c/Program Files/Sunshine/tools:/mnt/c/Program Files/PowerShell/7/:/mnt/c/Windows/system32/config/systemprofile/AppData/Local/Microsoft/WindowsApps:/mnt/c/Windows/system32/config/systemprofile/go/bin:/mnt/c/Users/sauri/AppData/Local/Programs/Microsoft VS Code/bin:/mnt/c/Program Files/Multipass/bin:/mnt/c/Windows/system32/config/systemprofile/.dotnet/tools:/mnt/c/Users/sauri/AppData/Roaming/npm:/mnt/c/Users/sauri/AppData/Local/Programs/Lens/resources/cli/bin:/mnt/c/Users/sauri/AppData/Local/GitHubDesktop/bin:/mnt/c/Program Files/mitmproxy/bin:/mnt/c/Users/sauri/go/bin:/mnt/c/Program Files/JetBrains/GoLand 2024.3.5/bin:/mnt/c/Users/sauri/AppData/Local/Microsoft/WindowsApps:/mnt/c/Users/sauri/AppData/Local/Programs/Ollama:/mnt/c/Users/sauri/AppData/Local/Microsoft/WinGet/Links:/mnt/c/Users/sauri/AppData/Local/Microsoft/WinGet/Packages/ar51an.iPerf3_Microsoft.Winget.Source_8wekyb3d8bbwe:/mnt/c/Users/sauri/AppData/Local/OpenAI/Codex/bin/ada252862d154cdd:/mnt/c/Program Files/WindowsApps/OpenAI.Codex_26.513.4821.0_x64__2p2nqsd0c76g0/app/resources',
  npm_package_name: 'openai-oauth-api-service-web',
  npm_config_prefer_offline: 'true',
  NODE: '/usr/local/lib/nodejs/node-v24.14.0-linux-x64/bin/node',
  XDG_RUNTIME_DIR: '/run/user/0/',
  npm_config_frozen_lockfile: '',
  DISPLAY: ':0',
  LANG: 'C.UTF-8',
  npm_lifecycle_script: 'vite -- --host 127.0.0.1 --no-open',
  SHELL: '/usr/bin/zsh',
  npm_package_version: '1.0.0',
  npm_lifecycle_event: 'start',
  npm_config_verify_deps_before_run: 'false',
  npm_config_strict_peer_dependencies: '',
  npm_config_npm_globalconfig: '/usr/local/lib/nodejs/node-v24.14.0-linux-x64/etc/npmrc',
  npm_config_globalconfig: '/root/.config/pnpm/rc',
  PWD: '/root/projects/openai-oauth-api-service/web',
  npm_execpath: '/root/.cache/node/corepack/v1/pnpm/10.33.2/bin/pnpm.cjs',
  HTTP_PROXY: 'http://127.0.0.1:7897',
  npm_config__jsr_registry: 'https://npm.jsr.io/',
  npm_command: 'run-script',
  PNPM_SCRIPT_SRC_DIR: '/root/projects/openai-oauth-api-service/web',
  HOSTTYPE: 'x86_64',
  WSL2_GUI_APPS_ENABLED: '1',
  npm_config_shamefully_hoist: 'true',
  WSLENV: '',
  INIT_CWD: '/root/projects/openai-oauth-api-service/web',
  NODE_ENV: 'development'
}
command = serve
mode = development

  VITE v5.4.21  ready in 181 ms

  ➜  Local:   http://localhost:5176/
  ➜  Network: http://10.255.255.254:5176/
  ➜  Network: http://192.168.0.45:5176/
2:23:55 PM [vite] hmr update /src/tailwind.css, /src/pages/AdminLogin/index.jsx, /src/pages/AdminDashboard/index.jsx, /src/pages/AdminApi/index.jsx, /src/common/components/layout/AdminFrame.jsx
2:23:56 PM [vite] hmr update /src/tailwind.css, /src/pages/AdminLogin/index.jsx, /src/pages/AdminDashboard/index.jsx, /src/pages/AdminApi/index.jsx, /src/common/components/layout/AdminFrame.jsx
5:03:50 PM [vite] hmr update /src/tailwind.css, /src/pages/AdminApi/index.jsx
5:03:50 PM [vite] hmr update /src/pages/AdminApi/index.jsx, /src/tailwind.css
5:03:50 PM [vite] hmr update /src/tailwind.css
 ELIFECYCLE  Command failed with exit code 1. 不再出现 FontAwesome token 告警，可启动 Vite 5176；在后端运行期间未再出现  代理连接拒绝。
- 下一步：如换机器或重置 WSL，需要先确保 PostgreSQL 18 在 5433 运行，并恢复本机  私有 DSN； 仍保持 gitignore，不提交真实本机凭据。

## 2026-05-22 Codex 长 session 上下文压缩与 backend 重试收敛
- 发现：线上长 session 报错不只影响 `zichun`，近 7 天失败主要来自无 `session_id` 的长 streaming backend 请求；历史诊断里 90 万字节级请求的 `context_original_estimated_tokens` 仍只有几百到一千多，说明旧估算只抽取了尾部对话，漏掉了较早的工具调用、`function_call_output`、`arguments` 和 `output`。
- 修复：网关上下文估算改为从完整 JSON 请求提取 `instructions`、`input/messages`、`content/text`、工具调用、函数参数、输出和 `tools`；Codex 推荐模型的自动压缩字节阈值收紧到 850000，硬上限仍按 1900000 字节；backend SSE 的 `response.failed/incomplete` 保留事件级错误类型，`context_length_exceeded` 不再被记成泛化 incomplete。
- 修复：direct Codex backend 重试边界收窄为 HTTP 429/5xx 与连接类错误；`response.failed`、`response.incomplete` 这类模型或请求语义终态不再由服务端重试，避免重复消耗和错误放大。客户端看到网络断开、超时或 429/5xx 可自行重试；已收到模型终态失败时应让用户修改输入或触发压缩后再发。
- 结论：本轮不升级服务器 Codex CLI。官方 `@openai/codex` 当前 npm latest 为 0.133.0，线上容器仍是 `codex-cli 0.129.0`；本次问题根因在网关 direct backend 请求估算/压缩和重试分类，不是 CLI 版本主因。
- 验证通过：`cd server && go test ./internal/server ./internal/biz`、`cd server && go test ./...`、`git diff --check`；新增测试覆盖旧工具上下文计入估算、850KB 前置压缩、SSE context length 分类和 terminal backend event 不重试。
- 部署完成：本地构建并上传 `oauth-api-service-server:20260522T162645-c0512d2-local-context-retry` 到 `root@8.218.4.199`，远端只执行 release 解包、Atlas status、`docker load`、`docker compose up`，未在低配服务器构建。Atlas migration status 为 OK，当前版本 `20260520090000`，pending 0。
- 线上验证通过：内网与公网 `/healthz` 返回 `ok`，`/readyz` 返回 `ready`；容器运行镜像为 `oauth-api-service-server:20260522T162645-c0512d2-local-context-retry`，`GIT_SHA_SHORT=c0512d2-local`，`IMAGE_TAG=20260522T162645-c0512d2-local-context-retry`；管理员 `admin/adminadmin` 登录、`api.summary`、`api.key_list` 均返回 `code=0`。
- 兼容入口验证：创建纯字母数字备注的临时 key，调用 `/v1/models` 返回 6 个模型且包含 `gpt-5.5`，随后删除临时 key 成功；部署后 15 分钟窗口已有 47 次请求且失败 0 次，最近 `ogw_zhonglia` streaming responses 请求 43 万到 59 万字节均成功。
- 清理：按发布约定执行 `docker image prune -a -f` 与 `docker builder prune -f`，未执行 volume prune；旧 app 镜像 `oauth-api-service-server:20260520T184137-ef3ec0e5-local-resume-guard` 已清理，根分区从 52% 回到 51%。
- 阻塞/风险：尚未等待到真实超过 850000 字节的新线上请求来验证生产压缩诊断字段；后续需要继续观察 `gateway_usage_logs.diagnostic` 中的 `context_compacted`、`context_original_estimated_tokens`、`context_final_estimated_tokens` 以及 `context_length_exceeded` 是否下降。
- 补充调整：按最新口径将 Codex 推荐模型和旧默认兜底的自动压缩字节阈值从 `850000` 调整为 `1000000`，硬上限保持 `1900000`；本地构建并部署 `oauth-api-service-server:20260522T175109-c0512d22-local-context-1m` 后，公网健康检查通过，管理员 `model_list` 确认 `gpt-5.5` 生效阈值为 `400000 / 260000 / 380000 / 1000000 / 1900000 / 8`。后续观察口径改为 1M 字节以上请求。
