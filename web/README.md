# web 前端说明

## 目录结构（简版）

| 路径          | 职责                                        |
| ------------- | ------------------------------------------- |
| `src/common/` | 通用认证、组件、hooks、状态、常量与工具函数 |
| `src/pages/`  | 管理员登录与 API 运营控制台                 |
| `src/mocks/`  | 本地 mock 与前端基线测试辅助                |
| `src/assets/` | 图标等静态资源                              |
| `public/`     | 静态公开资源                                |
| `scripts/`    | 最小浏览器级样式回归等前端侧脚本            |
| `build/`      | 构建产物，不作为日常开发真源                |

日常开发入口优先关注 `src/`、`scripts/` 与 `public/`；`build/`、`output/` 更偏本地产物，不建议当成业务实现入口。

## 启动与构建

```bash
cd web
pnpm install
pnpm start
```

默认地址：`http://127.0.0.1:5176`；开发服务器会把 `http://localhost:5176` 自动规范到同一 IPv4 地址。

```bash
cd web
pnpm lint
pnpm css
pnpm test
pnpm playwright:install
pnpm style:l1
pnpm build
```

- `pnpm style:l1` 是当前仓库最小浏览器级样式回归，会自动拉起本地 Vite 并覆盖根路径、历史 `/login`、`/register`、`/oauth-login`、`/portal` 到管理员登录的收口，管理员登录、未登录访问 `/admin-menu` 的重定向，公开 `/client-config` 配置生成页，以及注入测试管理员态后的 `/admin-dashboard` 桌面与移动端页面、登录态失效弹窗、`/admin-upstream`、`/admin-accounts` 和 `/admin-oauth` 到看板的收口。局部排查可用 `STYLE_L1_SCENARIOS=场景名1,场景名2 pnpm style:l1` 只跑指定场景。
- API 运营后台路径为 `/admin-dashboard`，生产运行依赖真实后端 `/rpc/api` 数据；兼容保留 `/admin-api`、`/admin-keys`、`/admin-models`、`/admin-analytics`、`/admin-upstream`、`/admin-client-config` 和 `/admin-usage` 入口，其中 `/admin-analytics` 会回跳到合并后的 `/admin-usage`。公开 `/client-config` 只提供免登录客户端配置生成器，不复用后台导航，不调用后端接口，不保存 API Key；`style:l1` 使用 mock 数据覆盖登录后的基础样式、调用趋势可视化面板、上游策略与全局默认推理档位切换、API 凭据弹窗和盒模型回归。
- 业务看板保留今日 Token、今日请求、错误率、响应耗时、当前 RPM/TPM、Backend / CLI 上游分布、API 凭据、用量趋势、Token 构成、模型 / 接口分布和最近调用样本；用量趋势复用「用量日志」的 `今天/24h/7 天/30 天/90 天/180 天/1 年/2 年/3 年/5 年` 时间窗口，默认 30 天，支持柱状 / 折线切换，并支持 hover / focus 查看日期与指标明细；凭据宽表、调用状态细分和按天明细进入统一的 `/admin-usage`「用量日志」，上游策略切换进入 `/admin-upstream`「上游策略」，避免首页信息过载。
- API 运营后台表格默认每页 8 条，并支持 `8/10/20/50/100` 切换；用量日志支持按 `今天/24h/7 天/30 天/90 天/180 天/1 年/2 年/3 年/5 年` 时间窗口分页查询，默认选择「今天」，并按常用查看频率提供调用明细、异常请求、会话聚合、凭据统计、每日模型、上游模式、客户端类型和上游错误类型筛选视图；调用明细、顶部最近请求表和会话请求级明细都会以独立列展示后端记录的客户端 IP，调用明细支持按完整客户端 IP 筛选，异常请求明细会展示请求 / 响应大小、backend-only、fallback 阻断、上下文压缩状态和脱敏上游摘要等诊断字段。每日模型默认使用 30 天窗口，手动改时间范围后遵循当前筛选；它按日期 + 模型聚合请求、Token、费用、错误率、Backend / CLI / fallback 以及 Codex / OpenCode / 其他客户端统计，汇总表支持本地分页，点击详情弹窗下钻当天该模型的请求级明细，详情分页同样使用后台表格分页控件；会话聚合只展示后端已记录 `session_id` 的调用，并可在详情弹窗展开请求级明细，同时展示该会话的上下文压缩次数、摘要、压缩前后体积 / token 粗估和客户端类型分布。统计表格中涉及单个 API 凭据的行会同时展示备注和前缀，凭据统计表按 `今天/24h/7 天/30 天/180 天/360 天/1 年/3 年/5 年` 固定窗口展示每个凭据的 Token 汇总，并优先按今天 Token 排序；今天窗口全 0 时自动降级到 24h，再按 7 天、30 天和更长窗口依次降级，便于定位近期真实使用方。API 凭据表支持单击行单选、行首选择框多选、表头选择框全选 / 取消当前页、双击行打开编辑弹窗，备注输入只允许字母和数字；新建时后端会生成 `ogw_<备注>_<随机串>` 形式的凭据，管理员后台凭据列表展示完整凭据并提供复制；保存备注、额度、模型、上游策略或默认推理档位不会重新生成 API key，选中一个或多个凭据后可在「当前操作」批量重置 API key，也可在编辑弹窗里重置单个 key。重置、删除、全站禁用全部 key 和全站启用全部 key 统一使用后台内确认弹窗，避免浏览器原生提示打断操作流。重置只用于 key 泄密或主动轮换，会让旧 key 立即失效，并展示新完整 key 供逐条复制或复制全部；全站禁用全部 key 用于额度紧张或临时停用下游调用，全站启用全部 key 用于恢复下游调用，两者都只改禁用状态，不删除 key 和历史 usage。凭据级上游策略默认继承全局，也可单独覆盖为 Backend 直连、Backend + CLI 兜底或强制 CLI；凭据级默认推理档位可继承全局、关闭覆盖或覆盖为 Fast / Medium / High / Deep，`/admin-upstream` 的全局推理档位默认关闭，开启后会覆盖客户端传入的 reasoning effort；凭据级 Token 限制按每日和每周两个窗口配置，并可分别限制总量、输入、输出和非缓存输入；模型表使用代码内固定官方目录，支持启停并可按模型调整上下文窗口、开始压缩阈值、硬拦截阈值、字节兜底和压缩保留条数，默认按 Codex 体验使用 400K 窗口、260K/380K token 阈值，阈值输入支持整数、`K` 和 `M` 单位，相关列和弹窗字段带问号说明；客户端配置生成器可导出 macOS / Windows 的 Codex 与 opencode 最小配置，复制或下载前要求填写 API Key。
- API 凭据列表用独立列展示每个 key 的最近使用时间；后端没有记录时显示 `-`。
- 凭据统计表支持点击任意 Token 窗口表头切换该列升序 / 降序；未手动排序时仍按今天优先、空窗口自动降级到 24h、7 天及更长窗口。
- 管理员登录页和 API 运营后台支持「跟系统 / 浅色 / 暗夜」三种主题模式，默认跟随系统偏好，并通过浏览器 `localStorage` 在刷新后保持手动选择。
- `pnpm test` 当前只负责验证错误码常量与登录态错误分类这类最小前端基线；它不替代浏览器里的样式 / box 模型验收。

## 环境变量

- `VITE_BASE_URL`：前端部署基础路径
- `VITE_APP_TITLE`：页面标题
- `VITE_ENABLE_RPC_MOCK`：是否启用本地 RPC mock
- `VITE_API_PROXY_TARGET`：本地 Vite 代理的后端地址，默认 `http://127.0.0.1:8400`
  说明：前端只保存本系统管理员登录返回的 JWT。下游 `ogw_` key 由管理员在 `/admin-keys` 页面生成和维护，OpenAI 兼容客户端使用本服务 `/v1` Base URL 和该 key 调用。

管理员 OAuth 登录按钮只在后端 `/auth/oauth/config` 返回启用时显示。授权完成后前端 `/oauth/callback` 从 URL fragment 读取管理员 JWT 并写入 `admin_access_token`，随后跳转到后台页面；本地 Vite 端口变化不需要改前端配置。

环境文件：

- `web/.env.development`
- `web/.env.production`

说明：当前可执行 `cd web && pnpm test` 验证错误码常量与鉴权分类基线，执行 `pnpm style:l1` 验证后台登录与后台页面的最小浏览器级样式回归；若任务涉及更复杂页面，仍应继续补页面级回归。
