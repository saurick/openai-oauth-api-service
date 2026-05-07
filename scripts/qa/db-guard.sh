#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/db-guard.sh

作用:
  防止 Ent schema/ent 变更遗漏 migration 文件

触发规则:
  变更包含 server/internal/data/model/schema/* 或 server/internal/data/model/ent/*
  但不包含 server/internal/data/model/migrate/* 时阻断

环境变量:
  SKIP_DB_GUARD=1    跳过检查
  QA_BASE_RANGE=...  指定 diff 范围（例：origin/master...HEAD）
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ $# -gt 0 ]]; then
	echo "[qa:db-guard] 不支持的参数: $*"
	print_help
	exit 1
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_DB_GUARD:-0}" == "1" ]]; then
	echo "[qa:db-guard] SKIP_DB_GUARD=1，跳过"
	exit 0
fi

if [ ! -d "$ROOT_DIR/server/internal/data/model" ]; then
	echo "[qa:db-guard] 未发现 server/internal/data/model，跳过"
	exit 0
fi

range="${QA_BASE_RANGE:-}"
if [ -z "$range" ]; then
	if git rev-parse --abbrev-ref --symbolic-full-name "@{upstream}" >/dev/null 2>&1; then
		upstream="$(git rev-parse --abbrev-ref --symbolic-full-name "@{upstream}")"
		range="${upstream}...HEAD"
	elif git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
		range="HEAD~1...HEAD"
	fi
fi

changed_files=()
if [ -n "$range" ]; then
	while IFS= read -r f; do
		[ -n "$f" ] && changed_files+=("$f")
	done < <(git diff --name-only "$range")
fi

while IFS= read -r f; do
	[ -n "$f" ] && changed_files+=("$f")
done < <(git diff --name-only)

while IFS= read -r f; do
	[ -n "$f" ] && changed_files+=("$f")
done < <(git diff --name-only --cached)

if [ "${#changed_files[@]}" -eq 0 ]; then
	echo "[qa:db-guard] 未检测到变更，跳过"
	exit 0
fi

uniq_files=()
while IFS= read -r f; do
	[ -n "$f" ] && uniq_files+=("$f")
done < <(printf "%s\n" "${changed_files[@]}" | sort -u)

need_migration=0
has_migration_file=0
for f in "${uniq_files[@]}"; do
	case "$f" in
	server/internal/data/model/schema/* | server/internal/data/model/ent/*)
		need_migration=1
		;;
	esac

	case "$f" in
	server/internal/data/model/migrate/*)
		has_migration_file=1
		;;
	esac
done

if [ "$need_migration" -eq 1 ] && [ "$has_migration_file" -eq 0 ]; then
	echo "[qa:db-guard] 检测到 schema/ent 变更但未发现 migration 变更"
	echo "[qa:db-guard] 请先在 /server 执行 make data，并提交生成的迁移文件"
	exit 1
fi

echo "[qa:db-guard] 通过"
