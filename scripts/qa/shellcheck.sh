#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/shellcheck.sh

作用:
  对 scripts 与 .githooks 下的 shell 脚本执行 shellcheck 静态检查

环境变量:
  SKIP_SHELLCHECK=1     跳过检查
  SHELLCHECK_STRICT=1   shellcheck 未安装时阻断（默认仅提示）
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ $# -gt 0 ]]; then
	echo "[qa:shellcheck] 不支持的参数: $*"
	print_help
	exit 1
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_SHELLCHECK:-0}" == "1" ]]; then
	echo "[qa:shellcheck] SKIP_SHELLCHECK=1，跳过"
	exit 0
fi

strict="${SHELLCHECK_STRICT:-0}"
if ! command -v shellcheck >/dev/null 2>&1; then
	echo "[qa:shellcheck] 未安装 shellcheck，跳过（建议安装后启用）"
	if [[ "$strict" == "1" ]]; then
		exit 1
	fi
	exit 0
fi

files=()
while IFS= read -r -d '' f; do
	files+=("$f")
done < <(find scripts .githooks -type f \( -name '*.sh' -o -name 'pre-commit' -o -name 'pre-push' -o -name 'commit-msg' \) -print0)

if [[ "${#files[@]}" -eq 0 ]]; then
	echo "[qa:shellcheck] 未发现可检查脚本"
	exit 0
fi

shellcheck "${files[@]}"

echo "[qa:shellcheck] 通过"
