#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/git-hooks/commit-msg.sh <commit-msg-file>

作用:
  校验提交标题是否符合 Conventional Commits

允许类型:
  feat|fix|chore|docs|refactor|test|ci|build|perf|style

环境变量:
  SKIP_COMMIT_MSG=1  跳过检查
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ "${SKIP_COMMIT_MSG:-0}" == "1" ]]; then
	echo "[commit-msg] SKIP_COMMIT_MSG=1，跳过"
	exit 0
fi

msg_file="${1:-}"
if [ -z "$msg_file" ] || [ ! -f "$msg_file" ]; then
	echo "[commit-msg] 未找到提交信息文件"
	print_help
	exit 1
fi

subject="$(head -n 1 "$msg_file" | tr -d "\r")"

if [[ "$subject" =~ ^(Merge|Revert|fixup!|squash!) ]]; then
	exit 0
fi

pattern='^(feat|fix|chore|docs|refactor|test|ci|build|perf|style)(\([a-zA-Z0-9._/-]+\))?: .+'
if [[ ! "$subject" =~ $pattern ]]; then
	echo "[commit-msg] 提交信息不符合规范：$subject"
	echo "[commit-msg] 参考：chore(hooks): 接入本地质量门禁"
	echo "[commit-msg] 允许类型：feat|fix|chore|docs|refactor|test|ci|build|perf|style"
	exit 1
fi
