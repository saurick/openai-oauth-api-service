#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/fast.sh

作用:
  执行开发期高频快速检查

检查内容:
  error-code-sync: 前端生成错误码同步检查
  error-codes: 统一错误码魔法数字检查
  web: pnpm lint -> pnpm css
  server: go test ./internal/... ./pkg/...（存在即测）

环境变量:
  SKIP_DB_GUARD=1  跳过 DB 迁移守卫
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ $# -gt 0 ]]; then
	echo "[qa:fast] 不支持的参数: $*"
	print_help
	exit 1
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if ! command -v pnpm >/dev/null 2>&1; then
	echo "[qa:fast] 未找到 pnpm，请先安装 pnpm"
	exit 1
fi

if ! command -v go >/dev/null 2>&1; then
	echo "[qa:fast] 未找到 go，请先安装 Go"
	exit 1
fi

if [ -x "$ROOT_DIR/scripts/qa/db-guard.sh" ]; then
	bash "$ROOT_DIR/scripts/qa/db-guard.sh"
fi

if [ -x "$ROOT_DIR/scripts/qa/error-code-sync.sh" ]; then
	bash "$ROOT_DIR/scripts/qa/error-code-sync.sh"
fi

if [ -x "$ROOT_DIR/scripts/qa/error-codes.sh" ]; then
	# 先拦截错误码魔法数字，避免在明显违规代码上继续跑后续检查。
	echo "[qa:fast] 运行错误码魔法数字检查"
	bash "$ROOT_DIR/scripts/qa/error-codes.sh"
fi

echo "[qa:fast] 运行 web 快速检查"
(
	cd "$ROOT_DIR/web"
	pnpm lint
	pnpm css
)

echo "[qa:fast] 运行 server 快速检查"
(
	cd "$ROOT_DIR/server"
	pkgs=()
	if [ -d internal ]; then
		pkgs+=("./internal/...")
	fi
	if [ -d pkg ]; then
		pkgs+=("./pkg/...")
	fi

	if [ "${#pkgs[@]}" -gt 0 ]; then
		go test "${pkgs[@]}"
	else
		echo "[qa:fast] 未发现 internal/pkg，跳过 Go 测试"
	fi
)

echo "[qa:fast] 完成"
