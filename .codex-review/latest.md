# 默认推理档位开关审查报告

## 范围

- 后端新增全局默认推理档位设置，默认关闭。
- API key 新增 `default_reasoning_effort` 字段，支持继承全局、关闭默认或覆盖 `low` / `medium` / `high` / `xhigh`。
- 请求处理优先级为：key 覆盖档位 > 全局覆盖档位 > 客户端请求档位。
- 后台「上游策略」页增加全局默认开关，API 凭据表格和新建 / 编辑弹窗增加 key 级默认档位。

## 修改文件

- `server/internal/biz/gateway.go`
- `server/internal/data/gateway_repo.go`
- `server/internal/data/jsonrpc_gateway.go`
- `server/internal/server/openai_gateway_handler.go`
- `server/internal/data/model/schema/gateway_api_key.go`
- `server/internal/data/model/migrate/20260531143157_migrate.sql`
- `web/src/pages/AdminApi/index.jsx`
- `web/src/tailwind.css`
- `web/scripts/styleL1.mjs`
- `README.md`
- `server/docs/api.md`
- `server/docs/config.md`
- `web/README.md`
- `progress.md`

## 行为口径

- 全局默认档位默认关闭，保存在 gateway settings。
- key 级空值表示继承全局；`none` 表示关闭后台覆盖并保留客户端原始档位；其他合法值为 `low`、`medium`、`high`、`xhigh`。
- 后台 key / 全局档位会覆盖客户端请求里的 `reasoning_effort`，用于约束 Codex / OpenCode 这类会自动带默认 effort 的客户端。
- usage 继续记录最终生效的 `reasoning_effort`；没有显式值且没有默认值时保持空值。

## 验证

- 已通过：`cd server && go test ./internal/biz ./internal/data ./internal/server`
- 已通过：`cd server && go test ./...`
- 已通过：`cd server && make build`
- 已通过：`cd server && atlas migrate validate --dir "file://internal/data/model/migrate"`
- 已通过：`cd web && pnpm test`
- 已通过：`cd web && pnpm build`
- 已通过：`cd web && pnpm style:l1`，共 30 个场景
- 已通过：`git diff --check`
- 已通过：in-app Browser 打开本地 `/admin-upstream` 和 `/admin-keys`，确认全局默认档位可切 Fast 再关回、凭据表格展示默认 Effort、新建弹窗包含默认推理档位。
- 已通过：本机 Codex 临时配置 + 临时 key 返回 `LOCAL_CODEX_FAST_OVERRIDE_OK`，usage 为 `200 / low`。
- 已通过：Windows `sauri@192.168.0.45` 上 Codex 临时配置 + 临时 key 返回 `WINDOWS_CODEX_FAST_OVERRIDE_OK`，usage 为 `200 / low`。

## 备注

- Browser 控制台仅看到 React Router v7 future flag 既有 warning，未见本轮新增错误。

## 未做

- 本报告记录部署前代码验证结果；提交、推送和远端部署结果以本轮最终回复为准。
