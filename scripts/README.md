# 脚本说明

本目录提供本地质量门禁、初始化检查和 Git hooks。

## 常用命令

| 命令 | 作用 |
| --- | --- |
| `bash scripts/bootstrap.sh` | 安装依赖、启用 hooks、运行快速自检 |
| `bash scripts/doctor.sh` | 检查本机依赖、hooks 与关键脚本状态 |
| `bash scripts/ops/install-codex-upstream-proxy-failover.sh` | 在服务器安装 Codex 上游代理自动切换 systemd 守护 |
| `bash scripts/ops/install-codex-runtime-health-check.sh` | 在服务器安装 Codex runtime 每日健康检查 systemd timer |
| `bash scripts/init-project.sh --project --strict` | 检查是否仍有模板残留或默认配置 |
| `bash scripts/deploy/production-preflight.sh` | 生产发布前门禁，检查运行时 env、Compose、Codex upstream、migration 文档边界和部署后 healthz/readyz |
| `bash scripts/qa/fast.sh` | 开发期快速检查 |
| `bash scripts/qa/full.sh` | 提交/推送前全量检查 |
| `bash scripts/qa/strict.sh` | 发版前严格检查 |
| `python3 scripts/qa/live-context-compaction.py` | 手动线上多轮上下文压缩回归；必须显式提供 `GATEWAY_BASE_URL` 和 `GATEWAY_API_KEY` |

## 质量脚本

| 脚本 | 说明 |
| --- | --- |
| `scripts/qa/db-guard.sh` | Ent schema / ent 变更必须配套 migration |
| `scripts/qa/agents-size.sh` | 扫描全部 AGENTS.md；16 KiB 预警、超过 24 KiB 阻断，不自动改写 |
| `scripts/qa/skill-health.mjs` | 检查项目 Skill frontmatter、目录名、metadata、README 索引和相对引用；不依赖 PyYAML，并由 fast/full/strict 调用 |
| `scripts/qa/error-code-sync.sh` | 前端生成错误码必须与服务端目录同步 |
| `scripts/qa/error-codes.sh` | 业务代码禁止裸写已注册错误码 |
| `scripts/qa/secrets.sh` | 扫描疑似密钥泄露 |
| `scripts/qa/shellcheck.sh` | Shell 静态检查 |
| `scripts/qa/shfmt.sh` | Shell 格式化检查 |
| `scripts/qa/go-vet.sh` | Go vet |
| `scripts/qa/golangci-lint.sh` | Go lint |
| `scripts/qa/govulncheck.sh` | Go 漏洞扫描 |
| `scripts/qa/yamllint.sh` | YAML 检查 |

## Live 回归

`scripts/qa/live-context-compaction.py` 会向真实 gateway `/v1/responses` 连续发送大上下文请求，验证同一 `session_id` 多次压缩后，最终官方回答仍能回忆早期 `durable_facts` 自然语言事实。它会消耗真实上游额度，也可能触发线上大请求突发保护，因此不纳入 `fast.sh` / `full.sh` / `strict.sh`。

最小运行方式：

```bash
GATEWAY_BASE_URL=https://example.com \
GATEWAY_API_KEY="$GATEWAY_API_KEY" \
python3 scripts/qa/live-context-compaction.py
```

常用调参：

```bash
GATEWAY_COMPACTION_ROUNDS=4 \
GATEWAY_COMPACTION_SLEEP_SECONDS=20 \
python3 scripts/qa/live-context-compaction.py
```

前端样式或布局改动时，`fast/full` 不能替代浏览器级回归；还需要执行：

```bash
cd web
pnpm style:l1
```

## Hooks

```bash
bash scripts/setup-git-hooks.sh
```

- `pre-commit`：增量格式化、shellcheck、错误码同步、密钥扫描、Go/YAML 检查。
- `pre-push`：严格 shellcheck + `SECRETS_STRICT=1 scripts/qa/full.sh`。
- `commit-msg`：提交信息格式检查。
