#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/golangci-lint.sh [golangci-lint 包参数...]

作用:
  对 server 执行 golangci-lint。
  默认仅校验“相对 HEAD 的新增问题”，用于基线降噪。

环境变量:
  SKIP_GOLANGCI_LINT=1  跳过检查
  GOLANGCI_STRICT=1     未安装 golangci-lint 时阻断
  GOLANGCI_ONLY_NEW=1   仅拦截新增问题（默认 1）
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_GOLANGCI_LINT:-0}" == "1" ]]; then
	echo "[qa:golangci-lint] SKIP_GOLANGCI_LINT=1，跳过"
	exit 0
fi

strict="${GOLANGCI_STRICT:-0}"
if ! command -v golangci-lint >/dev/null 2>&1; then
	echo "[qa:golangci-lint] 未安装 golangci-lint"
	if [[ "$strict" == "1" ]]; then
		echo "[qa:golangci-lint] GOLANGCI_STRICT=1，阻断"
		exit 1
	fi
	echo "[qa:golangci-lint] 跳过"
	exit 0
fi

if [[ ! -d "$ROOT_DIR/server" ]]; then
	echo "[qa:golangci-lint] 未找到 server 目录，跳过"
	exit 0
fi

if [[ $# -gt 0 ]]; then
	targets=("$@")
else
	targets=(./...)
fi

run_args=(run)
if [[ "${GOLANGCI_ONLY_NEW:-1}" == "1" ]]; then
	run_args+=(--new-from-rev HEAD)
fi
run_args+=("${targets[@]}")

(
	cd "$ROOT_DIR/server"
	golangci-lint "${run_args[@]}"
)

echo "[qa:golangci-lint] 通过"
