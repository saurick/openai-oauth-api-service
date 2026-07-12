#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WARN_BYTES=$((16 * 1024))
BLOCK_BYTES=$((24 * 1024))
failed=0
while IFS= read -r -d '' file; do
	bytes="$(wc -c <"$file" | tr -d ' ')"
	relative="${file#"$ROOT_DIR"/}"
	if ((bytes > BLOCK_BYTES)); then
		echo "[qa:agents-size] 阻断: $relative 为 ${bytes} bytes，超过 24 KiB；请按 AGENTS 治理优先级精简后再继续"
		failed=1
	elif ((bytes >= WARN_BYTES)); then
		echo "[qa:agents-size] 预警: $relative 为 ${bytes} bytes，已达到 16 KiB；新增规则前先去重和迁移细节"
	else echo "[qa:agents-size] 通过: $relative 为 ${bytes} bytes"; fi
done < <(find "$ROOT_DIR" -name AGENTS.md -type f -not -path '*/.git/*' -not -path '*/node_modules/*' -not -path '*/vendor/*' -print0)
exit "$failed"
