#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if [[ "${SKIP_PRE_PUSH:-0}" == "1" ]]; then
	echo "[pre-push] SKIP_PRE_PUSH=1，跳过检查"
	exit 0
fi

if [ ! -x "$ROOT_DIR/scripts/qa/shellcheck.sh" ]; then
	echo "[pre-push] 缺少 scripts/qa/shellcheck.sh，请先执行 bootstrap/setup"
	exit 1
fi

if [ ! -x "$ROOT_DIR/scripts/qa/full.sh" ]; then
	echo "[pre-push] 缺少 scripts/qa/full.sh，请先执行 bootstrap/setup"
	exit 1
fi

if [ ! -x "$ROOT_DIR/scripts/qa/secrets.sh" ]; then
	echo "[pre-push] 缺少 scripts/qa/secrets.sh，请先执行 bootstrap/setup"
	exit 1
fi

# pre-push 强制脚本静态检查，缺失 shellcheck 视为失败。
SHELLCHECK_STRICT=1 bash "$ROOT_DIR/scripts/qa/shellcheck.sh"

# pre-push 强制密钥扫描阻断策略，避免疑似泄露被推送。
SECRETS_STRICT=1 bash "$ROOT_DIR/scripts/qa/full.sh"
