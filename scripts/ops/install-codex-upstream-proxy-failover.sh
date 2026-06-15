#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/ops/install-codex-upstream-proxy-failover.sh

作用:
  在当前服务器安装并启用 Codex 上游代理自动切换守护服务。

安装内容:
  - /usr/local/sbin/codex-upstream-proxy-failover.py
  - /etc/systemd/system/codex-upstream-proxy-failover.service

默认行为:
  - 不重启 app-server
  - 不重启 mihomo
  - 只重启 codex-upstream-proxy-failover 自身以加载最新脚本
  - 不修改 mihomo 订阅配置
  - 通过 mihomo controller 检查选择器和候选节点

可通过环境变量覆盖:
  CODEX_FAILOVER_CONTAINER=openai-oauth-api-service-server
  MIHOMO_CONFIG=/etc/mihomo/config.yaml
  MIHOMO_API=http://127.0.0.1:9090
  MIHOMO_SELECTOR=节点选择
  MIHOMO_INHERIT_GROUPS=ChatGPT
  CODEX_FAILOVER_NODES=日本JP-HY2,日本-优化3,日本-优化2,日本-优化
  CODEX_FAILOVER_COOLDOWN_SECONDS=180
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ $# -gt 0 ]]; then
	echo "[codex-failover-install] 不支持的参数: $*"
	print_help
	exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SOURCE_SCRIPT="$ROOT_DIR/scripts/ops/codex-upstream-proxy-failover.py"
SOURCE_UNIT="$ROOT_DIR/server/deploy/systemd/codex-upstream-proxy-failover.service"
TARGET_SCRIPT="/usr/local/sbin/codex-upstream-proxy-failover.py"
TARGET_UNIT="/etc/systemd/system/codex-upstream-proxy-failover.service"

if [[ ! -f "$SOURCE_SCRIPT" ]]; then
	echo "[codex-failover-install] 缺少源脚本: $SOURCE_SCRIPT"
	exit 1
fi

if [[ ! -f "$SOURCE_UNIT" ]]; then
	echo "[codex-failover-install] 缺少 systemd unit: $SOURCE_UNIT"
	exit 1
fi

for cmd in python3 systemctl docker; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		echo "[codex-failover-install] 缺少命令: $cmd"
		exit 1
	fi
done

sudo_cmd=()
if [[ "$(id -u)" -ne 0 ]]; then
	if ! command -v sudo >/dev/null 2>&1; then
		echo "[codex-failover-install] 当前不是 root，且未找到 sudo"
		exit 1
	fi
	sudo_cmd=(sudo)
fi

echo "[codex-failover-install] 语法检查"
python3 -m py_compile "$SOURCE_SCRIPT"

echo "[codex-failover-install] 安装脚本和 systemd unit"
"${sudo_cmd[@]}" install -m 0755 "$SOURCE_SCRIPT" "$TARGET_SCRIPT"
"${sudo_cmd[@]}" install -m 0644 "$SOURCE_UNIT" "$TARGET_UNIT"

echo "[codex-failover-install] 重载 systemd 并启用服务"
"${sudo_cmd[@]}" systemctl daemon-reload
"${sudo_cmd[@]}" systemctl enable codex-upstream-proxy-failover.service
"${sudo_cmd[@]}" systemctl restart codex-upstream-proxy-failover.service

echo "[codex-failover-install] 检查 mihomo 选择器配置"
"${sudo_cmd[@]}" "$TARGET_SCRIPT" --check

echo "[codex-failover-install] 服务状态"
"${sudo_cmd[@]}" systemctl --no-pager --lines=12 status codex-upstream-proxy-failover.service

echo "[codex-failover-install] 完成"
