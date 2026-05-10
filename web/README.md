# web 前端说明

## 目录结构（简版）

| 路径          | 职责                                              |
| ------------- | ------------------------------------------------- |
| `src/common/` | 通用认证、组件、hooks、状态、常量与工具函数       |
| `src/pages/`  | 管理员登录与 API 运营控制台 |
| `src/mocks/`  | 本地 mock 与前端基线测试辅助                      |
| `src/assets/` | 图标等静态资源                                    |
| `public/`     | 静态公开资源                                      |
| `scripts/`    | 最小浏览器级样式回归等前端侧脚本                  |
| `build/`      | 构建产物，不作为日常开发真源                      |

日常开发入口优先关注 `src/`、`scripts/` 与 `public/`；`build/`、`output/` 更偏本地产物，不建议当成业务实现入口。

## 启动与构建

```bash
cd web
pnpm install
pnpm start
```

```bash
cd web
pnpm lint
pnpm css
pnpm test
pnpm playwright:install
pnpm style:l1
pnpm build
```

- `pnpm style:l1` 是当前仓库最小浏览器级样式回归，会自动拉起本地 Vite 并覆盖根路径、历史 `/login`、`/register`、`/oauth-login`、`/portal` 到管理员登录的收口，管理员登录、未登录访问 `/admin-menu` 的重定向，以及注入测试管理员态后的 `/admin-dashboard` 桌面与移动端页面、`/admin-accounts` 和 `/admin-oauth` 到看板的收口。
- API 运营后台路径为 `/admin-dashboard`，生产运行依赖真实后端 `/rpc/api` 数据；兼容保留 `/admin-api`、`/admin-keys`、`/admin-models`、`/admin-analytics` 和 `/admin-usage` 入口，其中 `/admin-analytics` 会回跳到合并后的 `/admin-usage`。`style:l1` 使用 mock 数据覆盖登录后的基础样式、调用趋势可视化面板和盒模型回归。
- 业务看板保留今日消费、今日请求、错误率、响应耗时、当前 RPM/TPM、API 凭据、30 天趋势、Token 构成、模型 / 接口分布和最近调用样本；30 天趋势支持柱状 / 折线切换，并支持 hover / focus 查看日期与指标明细；凭据宽表、调用状态细分和按天明细进入统一的 `/admin-usage`「用量日志」，避免首页信息过载。
- API 运营后台表格默认每页 8 条，并支持 `8/10/20/50/100` 切换；用量日志支持按 `24h/7 天/30 天/90 天/180 天/1 年/2 年/3 年/5 年` 时间窗口分页查询，并在同一入口提供每日模型、凭据统计、会话聚合、调用明细和异常请求视图。每日模型按日期 + 模型聚合请求、Token、费用和错误率，点击详情弹窗下钻当天该模型的请求级明细；会话聚合只展示后端已记录 `session_id` 的调用，并可在详情弹窗展开请求级明细。API 凭据表支持单击行单选、双击行打开编辑弹窗，凭据级 Token 限制按每日和每周两个窗口配置；模型表使用代码内固定官方目录，只保留启停操作。
- 管理员登录页和 API 运营后台支持「跟系统 / 浅色 / 暗夜」三种主题模式，默认跟随系统偏好，并通过浏览器 `localStorage` 在刷新后保持手动选择。
- `pnpm test` 当前只负责验证错误码常量与登录态错误分类这类最小前端基线；它不替代浏览器里的样式 / box 模型验收。

## 环境变量

- `VITE_BASE_URL`：前端部署基础路径
- `VITE_APP_TITLE`：页面标题
- `VITE_ENABLE_RPC_MOCK`：是否启用本地 RPC mock
- `VITE_API_PROXY_TARGET`：本地 Vite 代理的后端地址，默认 `http://localhost:8400`
说明：前端只保存本系统管理员登录返回的 JWT。下游 `ogw_` key 由管理员在 `/admin-keys` 页面生成和维护，OpenAI 兼容客户端使用本服务 `/v1` Base URL 和该 key 调用。

管理员 OAuth 登录按钮只在后端 `/auth/oauth/config` 返回启用时显示。授权完成后前端 `/oauth/callback` 从 URL fragment 读取管理员 JWT 并写入 `admin_access_token`，随后跳转到后台页面；本地 Vite 端口变化不需要改前端配置。

环境文件：

- `web/.env.development`
- `web/.env.production`

说明：当前可执行 `cd web && pnpm test` 验证错误码常量与鉴权分类基线，执行 `pnpm style:l1` 验证后台登录与后台页面的最小浏览器级样式回归；若任务涉及更复杂页面，仍应继续补页面级回归。
