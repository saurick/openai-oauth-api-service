#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/yamllint.sh [YAML 文件...]

作用:
  检查 YAML 语法与风格。
  - 传入参数：仅检查指定 YAML 文件
  - 不传参数：默认只检查变更 YAML（基线模式）
  - 设置 YAMLLINT_ALL=1：全量扫描仓库 YAML

环境变量:
  SKIP_YAMLLINT=1   跳过检查
  YAMLLINT_STRICT=1 未安装 yamllint 时阻断
  YAMLLINT_ALL=1    全量扫描仓库 YAML
  QA_BASE_RANGE=... 指定 diff 范围（例：origin/master...HEAD）
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_YAMLLINT:-0}" == "1" ]]; then
	echo "[qa:yamllint] SKIP_YAMLLINT=1，跳过"
	exit 0
fi

strict="${YAMLLINT_STRICT:-0}"
if ! command -v yamllint >/dev/null 2>&1; then
	echo "[qa:yamllint] 未安装 yamllint"
	if [[ "$strict" == "1" ]]; then
		echo "[qa:yamllint] YAMLLINT_STRICT=1，阻断"
		exit 1
	fi
	echo "[qa:yamllint] 跳过"
	exit 0
fi

is_yaml_ignored() {
	local file="$1"
	case "$file" in
	.git/* | web/node_modules/* | web/build/* | server/bin/* | web/pnpm-lock.yaml | .playwright-cli/*)
		return 0
		;;
	*)
		return 1
		;;
	esac
}

targets=()
if [[ $# -gt 0 ]]; then
	for f in "$@"; do
		[[ -f "$f" ]] || continue
		if is_yaml_ignored "$f"; then
			continue
		fi
		targets+=("$f")
	done
elif [[ "${YAMLLINT_ALL:-0}" == "1" ]]; then
	while IFS= read -r -d '' f; do
		rel="${f#./}"
		if is_yaml_ignored "$rel"; then
			continue
		fi
		targets+=("$rel")
	done < <(find . -type f \( -name '*.yml' -o -name '*.yaml' \) -print0)
else
	files=()
	range="${QA_BASE_RANGE:-}"
	if [[ -z "$range" ]] && git rev-parse --abbrev-ref --symbolic-full-name "@{upstream}" >/dev/null 2>&1; then
		upstream="$(git rev-parse --abbrev-ref --symbolic-full-name "@{upstream}")"
		range="${upstream}...HEAD"
	fi

	if [[ -n "$range" ]]; then
		while IFS= read -r f; do
			[[ -n "$f" ]] && files+=("$f")
		done < <(git diff --name-only "$range")
	fi

	while IFS= read -r f; do
		[[ -n "$f" ]] && files+=("$f")
	done < <(git diff --name-only)

	while IFS= read -r f; do
		[[ -n "$f" ]] && files+=("$f")
	done < <(git diff --name-only --cached)

	while IFS= read -r f; do
		[[ "$f" == *.yml || "$f" == *.yaml ]] || continue
		[[ -f "$f" ]] || continue
		if is_yaml_ignored "$f"; then
			continue
		fi
		targets+=("$f")
	done < <(printf "%s\n" "${files[@]}" | sort -u)
fi

if [[ "${#targets[@]}" -eq 0 ]]; then
	echo "[qa:yamllint] 未发现可检查 YAML，跳过"
	exit 0
fi

yamllint_args=(-s)
if [[ -f "$ROOT_DIR/.yamllint" ]]; then
	yamllint_args+=(-c "$ROOT_DIR/.yamllint")
fi

yamllint "${yamllint_args[@]}" "${targets[@]}"

echo "[qa:yamllint] 通过"
