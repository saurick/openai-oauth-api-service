## 归档索引

- 2026-06-04：旧 `progress.md` 已按超过 600 行阈值归档到 `docs/archive/progress-2026-06-04-before-govulncheck.md`。归档内容只作历史追溯线索，不替代当前代码、README、docs 或部署真源。

## 2026-06-04 Go 漏洞依赖升级

- 完成：修复 `govulncheck` 可达漏洞告警，升级 `server/go.mod` 的 Go patch 指令到 `1.25.11` 并显式固定 `toolchain go1.26.4`，同步升级 `go.opentelemetry.io/otel/*` 到 `v1.43.0`、`golang.org/x/net` 到 `v0.53.0` 及相关间接依赖。未改 OAuth、网关转发、usage 统计、schema、迁移、前端页面、部署脚本或生产配置。
- 验证：已执行 `bash scripts/qa/govulncheck.sh`，结果为 0 个可达漏洞；已执行 `cd server && go test ./...`，通过。第一次全量测试中 `TestCodexBalanceRouteReturnsRateLimitsWithoutAuth` 曾返回 502，单测重跑通过后全量重跑也通过，判断为本机瞬时测试状态，不是本轮依赖升级后的稳定失败。
- 下一步：后续如要把公开 Codex 余额查询继续产品化，应单独补更稳定的 app-server fake / cache 隔离测试，避免本机状态影响全量测试判断。
- 阻塞/风险：本轮只处理服务端 Go 依赖安全更新和 `progress.md` 归档；未部署、未构建 Docker 镜像、未修改线上配置。
