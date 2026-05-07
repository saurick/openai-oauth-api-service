# web 前端说明

## 目录结构（简版）

| 路径          | 职责                                               |
| ------------- | -------------------------------------------------- |
| `src/common/` | 通用认证、组件、hooks、状态、常量与工具函数        |
| `src/pages/`  | 管理员登录、OAuth 回调、后台账号目录、OAuth 配置与 API 运营控制台 |
| `src/mocks/`  | 本地 mock 与前端基线测试辅助                       |
| `src/assets/` | 图标等静态资源                                     |
| `public/`     | 静态公开资源                                       |
| `scripts/`    | 最小浏览器级样式回归等前端侧脚本                   |
| `build/`      | 构建产物，不作为日常开发真源                       |

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

- `pnpm style:l1` 是当前仓库最小浏览器级样式回归，会自动拉起本地 Vite 并覆盖根路径、历史 `/login`、`/register`、`/oauth-login`、`/portal` 到管理员登录的收口，管理员登录、未登录访问 `/admin-menu` 的重定向，以及注入测试管理员态后的 `/admin-dashboard` 桌面与移动端页面和 `/admin-oauth` 后台配置页。
- API 运营后台路径为 `/admin-dashboard`，生产运行依赖真实后端 `/rpc/api` 数据；兼容保留 `/admin-api`、`/admin-keys`、`/admin-models` 和 `/admin-usage` 入口。`style:l1` 使用 mock 数据覆盖登录后的基础样式、usage 可视化面板和盒模型回归。
- `pnpm test` 当前只负责验证错误码常量与登录态错误分类这类最小前端基线；它不替代浏览器里的样式 / box 模型验收。

## 环境变量

- `VITE_BASE_URL`：前端部署基础路径
- `VITE_APP_TITLE`：页面标题
- `VITE_ENABLE_RPC_MOCK`：是否启用本地 RPC mock
- `VITE_API_PROXY_TARGET`：本地 Vite 代理的后端地址，默认 `http://localhost:8200`；如果你实际后端跑在 `8400`，可设为 `http://localhost:8400`

说明：OAuth 配置页会从后端 `GET /auth/oauth/config` 读取 OAuth 开关；前端不保存第三方 OAuth token，只保存后端回调签发的本系统管理员 JWT。

环境文件：

- `web/.env.development`
- `web/.env.production`

说明：当前可执行 `cd web && pnpm test` 验证错误码常量与鉴权分类基线，执行 `pnpm style:l1` 验证后台登录与后台页面的最小浏览器级样式回归；若任务涉及更复杂页面，仍应继续补页面级回归。
