#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/ops/install-codex-runtime-health-check.sh

作用:
  在当前服务器安装并启用 Codex runtime 每日健康检查 timer。

安装内容:
  - /usr/local/sbin/codex-runtime-health-check.py
  - /etc/systemd/system/codex-runtime-health-check.service
  - /etc/systemd/system/codex-runtime-health-check.timer

默认行为:
  - 每天只执行 --check，不自动升级 Codex
  - 不重启 app-server
  - 不重启 Docker、mihomo 或 Nginx
  - 写入 /var/lib/codex-runtime-health/state.json
  - 追加 /var/log/codex-runtime-health.log

可通过环境变量覆盖:
  OAUTH_API_COMPOSE_DIR=/data/openai-oauth-api-service/compose
  OAUTH_API_BASE_URL=http://127.0.0.1:8400
  OAUTH_API_CONTAINER=openai-oauth-api-service-server
  CODEX_RUNTIME_BIN=codex
  CODEX_RUNTIME_LATEST_VERSION_COMMAND='npm view @openai/codex version'
  CODEX_RUNTIME_UPGRADE_COMMAND='npm install -g @openai/codex@latest'

手动升级:
  CODEX_RUNTIME_UPGRADE_COMMAND='...' /usr/local/sbin/codex-runtime-health-check.py --upgrade
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ $# -gt 0 ]]; then
	echo "[codex-runtime-health-install] 不支持的参数: $*"
	print_help
	exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SOURCE_SCRIPT="$ROOT_DIR/scripts/ops/codex-runtime-health-check.py"
SOURCE_SERVICE="$ROOT_DIR/server/deploy/systemd/codex-runtime-health-check.service"
SOURCE_TIMER="$ROOT_DIR/server/deploy/systemd/codex-runtime-health-check.timer"
TARGET_SCRIPT="/usr/local/sbin/codex-runtime-health-check.py"
TARGET_SERVICE="/etc/systemd/system/codex-runtime-health-check.service"
TARGET_TIMER="/etc/systemd/system/codex-runtime-health-check.timer"

if [[ ! -f "$SOURCE_SCRIPT" ]]; then
	echo "[codex-runtime-health-install] 缺少源脚本: $SOURCE_SCRIPT"
	exit 1
fi

if [[ ! -f "$SOURCE_SERVICE" ]]; then
	echo "[codex-runtime-health-install] 缺少 systemd service: $SOURCE_SERVICE"
	exit 1
fi

if [[ ! -f "$SOURCE_TIMER" ]]; then
	echo "[codex-runtime-health-install] 缺少 systemd timer: $SOURCE_TIMER"
	exit 1
fi

for cmd in python3 systemctl install; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		echo "[codex-runtime-health-install] 缺少命令: $cmd"
		exit 1
	fi
done

sudo_cmd=()
if [[ "$(id -u)" -ne 0 ]]; then
	if ! command -v sudo >/dev/null 2>&1; then
		echo "[codex-runtime-health-install] 当前不是 root，且未找到 sudo"
		exit 1
	fi
	sudo_cmd=(sudo)
fi

echo "[codex-runtime-health-install] 语法检查"
python3 -m py_compile "$SOURCE_SCRIPT"

echo "[codex-runtime-health-install] 安装脚本和 systemd unit"
"${sudo_cmd[@]}" install -m 0755 "$SOURCE_SCRIPT" "$TARGET_SCRIPT"
"${sudo_cmd[@]}" install -m 0644 "$SOURCE_SERVICE" "$TARGET_SERVICE"
"${sudo_cmd[@]}" install -m 0644 "$SOURCE_TIMER" "$TARGET_TIMER"

echo "[codex-runtime-health-install] 重载 systemd 并启用 timer"
"${sudo_cmd[@]}" systemctl daemon-reload
"${sudo_cmd[@]}" systemctl enable --now codex-runtime-health-check.timer

echo "[codex-runtime-health-install] 执行一次健康检查"
"${sudo_cmd[@]}" "$TARGET_SCRIPT" --check

echo "[codex-runtime-health-install] timer 状态"
"${sudo_cmd[@]}" systemctl --no-pager --lines=12 status codex-runtime-health-check.timer

echo "[codex-runtime-health-install] 完成"
