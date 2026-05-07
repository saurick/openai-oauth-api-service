#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/error-code-sync.sh

作用:
  校验前端错误码生成文件是否与服务端错误码目录保持同步。

环境变量:
  SKIP_ERROR_CODE_SYNC=1 跳过检查
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_ERROR_CODE_SYNC:-0}" == "1" ]]; then
	echo "[qa:error-code-sync] SKIP_ERROR_CODE_SYNC=1，跳过"
	exit 0
fi

if ! command -v node >/dev/null 2>&1; then
	echo "[qa:error-code-sync] 未找到 node，请先安装 Node.js"
	exit 1
fi

# 统一走生成器的 check 模式，确保 CI / pre-push 可直接拦截漏同步。
node "$ROOT_DIR/scripts/gen-error-codes.mjs" --check

echo "[qa:error-code-sync] 通过"
