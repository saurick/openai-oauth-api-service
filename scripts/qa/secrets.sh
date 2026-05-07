#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/secrets.sh

作用:
  对变更文件做密钥泄露扫描（依赖 gitleaks）

行为:
  未安装 gitleaks: 默认仅提示并跳过；SECRETS_STRICT=1 时阻断
  检测到疑似泄露: 默认提示不阻断；SECRETS_STRICT=1 时阻断

环境变量:
  SKIP_SECRETS_SCAN=1   跳过检查
  SECRETS_STRICT=1      命中或工具缺失时阻断
  SECRETS_STAGED_ONLY=1 仅扫描 staged 内容（用于 pre-commit）
  QA_BASE_RANGE=...     指定 diff 范围（例：origin/master...HEAD）
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ $# -gt 0 ]]; then
	echo "[qa:secrets] 不支持的参数: $*"
	print_help
	exit 1
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_SECRETS_SCAN:-0}" == "1" ]]; then
	echo "[qa:secrets] SKIP_SECRETS_SCAN=1，跳过"
	exit 0
fi

strict="${SECRETS_STRICT:-0}"
staged_only="${SECRETS_STAGED_ONLY:-0}"

if ! command -v gitleaks >/dev/null 2>&1; then
	echo "[qa:secrets] 未安装 gitleaks"
	if [[ "$strict" == "1" ]]; then
		echo "[qa:secrets] SECRETS_STRICT=1，阻断"
		exit 1
	fi
	echo "[qa:secrets] 跳过（建议安装后启用）"
	exit 0
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

files=()
if [[ "$staged_only" == "1" ]]; then
	while IFS= read -r f; do
		[[ -n "$f" ]] && files+=("$f")
	done < <(git diff --cached --name-only --diff-filter=ACMR)
else
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
fi

if [[ "${#files[@]}" -eq 0 ]]; then
	echo "[qa:secrets] 未检测到待扫描变更，跳过"
	exit 0
fi

while IFS= read -r f; do
	[[ -z "$f" ]] && continue

	case "$f" in
	.git/* | web/node_modules/* | server/bin/*)
		continue
		;;
	esac

	mkdir -p "$tmp_dir/$(dirname "$f")"

	if [[ "$staged_only" == "1" ]]; then
		if ! git cat-file -e ":$f" 2>/dev/null; then
			continue
		fi
		git show ":$f" >"$tmp_dir/$f" 2>/dev/null || true
	else
		[[ -f "$ROOT_DIR/$f" ]] || continue
		cp "$ROOT_DIR/$f" "$tmp_dir/$f"
	fi
done < <(printf "%s\n" "${files[@]}" | sort -u)

if [[ -z "$(find "$tmp_dir" -type f -print -quit)" ]]; then
	echo "[qa:secrets] 过滤后无可扫描文件，跳过"
	exit 0
fi

if gitleaks detect --source "$tmp_dir" --no-banner --redact >/dev/null 2>&1; then
	echo "[qa:secrets] 通过"
	exit 0
fi

if [[ "$strict" == "1" ]]; then
	echo "[qa:secrets] 检测到疑似密钥泄露（SECRETS_STRICT=1，阻断）"
	exit 1
fi

echo "[qa:secrets] 检测到疑似密钥泄露（默认仅提示，不阻断）"
exit 0
