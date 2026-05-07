#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/shfmt.sh [shell 文件...]

作用:
  对 shell 脚本执行 shfmt 格式化。
  - 传入参数：仅处理指定文件
  - 不传参数：默认处理 scripts/ 与 .githooks/ 下脚本

环境变量:
  SKIP_SHFMT=1     跳过检查
  SHFMT_STRICT=1   未安装 shfmt 时阻断（默认仅提示）
  SHFMT_CHECK=1    仅检查格式，不改写文件（用于严格模式）
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_SHFMT:-0}" == "1" ]]; then
	echo "[qa:shfmt] SKIP_SHFMT=1，跳过"
	exit 0
fi

strict="${SHFMT_STRICT:-0}"
if ! command -v shfmt >/dev/null 2>&1; then
	echo "[qa:shfmt] 未安装 shfmt"
	if [[ "$strict" == "1" ]]; then
		echo "[qa:shfmt] SHFMT_STRICT=1，阻断"
		exit 1
	fi
	echo "[qa:shfmt] 跳过"
	exit 0
fi

targets=()
if [[ $# -gt 0 ]]; then
	for f in "$@"; do
		[[ -f "$f" ]] || continue
		targets+=("$f")
	done
else
	while IFS= read -r -d '' f; do
		targets+=("$f")
	done < <(find scripts .githooks -type f \( -name '*.sh' -o -name 'pre-commit' -o -name 'pre-push' -o -name 'commit-msg' \) -print0)
fi

if [[ "${#targets[@]}" -eq 0 ]]; then
	echo "[qa:shfmt] 未发现可处理脚本"
	exit 0
fi

if [[ "${SHFMT_CHECK:-0}" == "1" ]]; then
	if ! shfmt -d "${targets[@]}"; then
		echo "[qa:shfmt] 检测到未格式化脚本"
		exit 1
	fi
	echo "[qa:shfmt] 通过"
	exit 0
fi

shfmt -w "${targets[@]}"
echo "[qa:shfmt] 通过"
