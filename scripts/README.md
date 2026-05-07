# 脚本说明

本目录提供本地质量门禁、初始化检查和 Git hooks。

## 常用命令

| 命令 | 作用 |
| --- | --- |
| `bash scripts/bootstrap.sh` | 安装依赖、启用 hooks、运行快速自检 |
| `bash scripts/doctor.sh` | 检查本机依赖、hooks 与关键脚本状态 |
| `bash scripts/init-project.sh --project --strict` | 检查是否仍有模板残留或默认配置 |
| `bash scripts/qa/fast.sh` | 开发期快速检查 |
| `bash scripts/qa/full.sh` | 提交/推送前全量检查 |
| `bash scripts/qa/strict.sh` | 发版前严格检查 |

## 质量脚本

| 脚本 | 说明 |
| --- | --- |
| `scripts/qa/db-guard.sh` | Ent schema / ent 变更必须配套 migration |
| `scripts/qa/error-code-sync.sh` | 前端生成错误码必须与服务端目录同步 |
| `scripts/qa/error-codes.sh` | 业务代码禁止裸写已注册错误码 |
| `scripts/qa/secrets.sh` | 扫描疑似密钥泄露 |
| `scripts/qa/shellcheck.sh` | Shell 静态检查 |
| `scripts/qa/shfmt.sh` | Shell 格式化检查 |
| `scripts/qa/go-vet.sh` | Go vet |
| `scripts/qa/golangci-lint.sh` | Go lint |
| `scripts/qa/govulncheck.sh` | Go 漏洞扫描 |
| `scripts/qa/yamllint.sh` | YAML 检查 |

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
