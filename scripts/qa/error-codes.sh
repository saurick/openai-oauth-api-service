#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/error-codes.sh
  bash scripts/qa/error-codes.sh <file> [file ...]

作用:
  检查业务代码中是否直接裸写已注册的错误码魔法数字，强制回到统一错误码目录/常量。

环境变量:
  SKIP_ERROR_CODE_GUARD=1        跳过检查
  ERROR_CODE_GUARD_STAGED_ONLY=1 仅检查 staged 文件（pre-commit 默认使用）
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_ERROR_CODE_GUARD:-0}" == "1" ]]; then
	echo "[qa:error-codes] SKIP_ERROR_CODE_GUARD=1，跳过"
	exit 0
fi

if ! command -v rg >/dev/null 2>&1; then
	echo "[qa:error-codes] 未找到 rg，请先安装 ripgrep"
	exit 1
fi

is_supported_file() {
	local file="$1"
	[[ "$file" == server/* || "$file" == web/src/* ]] || return 1
	[[ "$file" =~ \.(go|js|jsx|ts|tsx|mjs)$ ]]
}

is_ignored_file() {
	local file="$1"
	case "$file" in
	server/internal/errcode/* | web/src/common/consts/errorCodes*.js | docs/* | README.md | AGENTS.md | progress.md)
		return 0
		;;
	server/third_party/* | server/internal/data/model/ent/* | server/api/* | server/internal/conf/*.pb.go | *.pb.go)
		return 0
		;;
	*_test.go | *.test.js | *.test.jsx | *.test.mjs | *.spec.js | *.spec.jsx)
		return 0
		;;
	*)
		return 1
		;;
	esac
}

normalize_target_path() {
	local file="$1"
	if [[ "$file" == "$ROOT_DIR"/* ]]; then
		file="${file#"$ROOT_DIR"/}"
	fi
	printf '%s\n' "$file"
}

collect_targets() {
	local -a raw_files=()
	if [[ $# -gt 0 ]]; then
		raw_files=("$@")
	elif [[ "${ERROR_CODE_GUARD_STAGED_ONLY:-0}" == "1" ]]; then
		while IFS= read -r -d '' file; do
			raw_files+=("$file")
		done < <(git diff --cached --name-only --diff-filter=ACMR -z)
	else
		while IFS= read -r file; do
			raw_files+=("$file")
		done < <(git ls-files server web)
	fi

	TARGET_FILES=()
	if [[ "${#raw_files[@]}" -eq 0 ]]; then
		return
	fi

	local file
	for file in "${raw_files[@]}"; do
		file="$(normalize_target_path "$file")"
		[[ -f "$file" ]] || continue
		is_supported_file "$file" || continue
		is_ignored_file "$file" && continue
		TARGET_FILES+=("$file")
	done
}

ERROR_CODES=()
while IFS= read -r code; do
	[[ -n "$code" ]] || continue
	[[ "$code" == "0" ]] && continue
	ERROR_CODES+=("$code")
done < <(rg -o 'Code:[[:space:]]*[0-9]+' server/internal/errcode/catalog.go | sed -E 's/.*Code:[[:space:]]*([0-9]+)/\1/' | sort -u)
if [[ "${#ERROR_CODES[@]}" -eq 0 ]]; then
	echo "[qa:error-codes] 未从 server/internal/errcode/catalog.go 提取到错误码"
	exit 1
fi

collect_targets "$@"
if [[ "${#TARGET_FILES[@]}" -eq 0 ]]; then
	echo "[qa:error-codes] 未发现需要检查的业务代码文件"
	exit 0
fi

CODE_REGEX="$(
	IFS='|'
	echo "${ERROR_CODES[*]}"
)"

MATCH_REGEX="Code:[[:space:]]*(${CODE_REGEX})\\b|\\.Code[[:space:]]*(==|!=|=)[[:space:]]*(${CODE_REGEX})\\b|\\.[A-Za-z_]*[Cc]ode[[:space:]]*(===|!==|==|!=|=)[[:space:]]*(${CODE_REGEX})\\b|\\b[A-Za-z_][A-Za-z0-9_]*[Cc]ode[[:space:]]*(===|!==|==|!=|=|:)[[:space:]]*(${CODE_REGEX})\\b|\\bcase[[:space:]]+(${CODE_REGEX})\\b"

if MATCHES="$(rg -n --color never -e "$MATCH_REGEX" -- "${TARGET_FILES[@]}" || true)" && [[ -n "$MATCHES" ]]; then
	echo "[qa:error-codes] 发现业务代码直接写错误码，请改为引用统一定义："
	echo "$MATCHES"
	exit 1
fi
echo "[qa:error-codes] 通过"
