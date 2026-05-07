#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/go-vet.sh [go vet 包参数...]

作用:
  对 server 执行 go vet（默认 ./...）

环境变量:
  SKIP_GO_VET=1  跳过检查
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_GO_VET:-0}" == "1" ]]; then
	echo "[qa:go-vet] SKIP_GO_VET=1，跳过"
	exit 0
fi

if ! command -v go >/dev/null 2>&1; then
	echo "[qa:go-vet] 未找到 go，请先安装 Go"
	exit 1
fi

if [[ ! -d "$ROOT_DIR/server" ]]; then
	echo "[qa:go-vet] 未找到 server 目录，跳过"
	exit 0
fi

if [[ $# -gt 0 ]]; then
	targets=("$@")
else
	targets=(./...)
fi

(
	cd "$ROOT_DIR/server"
	go vet "${targets[@]}"
)

echo "[qa:go-vet] 通过"
