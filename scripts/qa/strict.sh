#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/qa/strict.sh

作用:
  执行严格质量检查（warning 也视为失败）

检查内容:
  1) db-guard + secrets
  2) shellcheck + shfmt（可选）
  3) govulncheck（可选）
  4) web: eslint --max-warnings=0 + stylelint --max-warnings=0 + (可选 test) + build
  5) server: go test ./... + make build

环境变量:
  SKIP_DB_GUARD=1           跳过 DB 守卫
  SKIP_SECRETS_SCAN=1       跳过密钥扫描
  SECRETS_STRICT=1          secrets 命中时阻断
  STRICT_SKIP_SHELLCHECK=1  跳过 shellcheck
  STRICT_SKIP_SHFMT=1       跳过 shfmt
  STRICT_SKIP_GOVULNCHECK=1 跳过 govulncheck
  QA_BASE_RANGE=...         指定 diff 范围供 db-guard/secrets 使用
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ $# -gt 0 ]]; then
	echo "[qa:strict] 不支持的参数: $*"
	print_help
	exit 1
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

if ! command -v pnpm >/dev/null 2>&1; then
	echo "[qa:strict] 未找到 pnpm，请先安装 pnpm"
	exit 1
fi

if ! command -v go >/dev/null 2>&1; then
	echo "[qa:strict] 未找到 go，请先安装 Go"
	exit 1
fi

if [ -x "$ROOT_DIR/scripts/qa/db-guard.sh" ]; then
	bash "$ROOT_DIR/scripts/qa/db-guard.sh"
fi

if [ -x "$ROOT_DIR/scripts/qa/secrets.sh" ]; then
	bash "$ROOT_DIR/scripts/qa/secrets.sh"
fi

if [[ "${STRICT_SKIP_SHELLCHECK:-0}" != "1" ]] && [ -x "$ROOT_DIR/scripts/qa/shellcheck.sh" ]; then
	SHELLCHECK_STRICT=1 bash "$ROOT_DIR/scripts/qa/shellcheck.sh"
fi

if [[ "${STRICT_SKIP_SHFMT:-0}" != "1" ]] && [ -x "$ROOT_DIR/scripts/qa/shfmt.sh" ]; then
	SHFMT_STRICT=1 SHFMT_CHECK=1 bash "$ROOT_DIR/scripts/qa/shfmt.sh"
fi

if [[ "${STRICT_SKIP_GOVULNCHECK:-0}" != "1" ]] && [ -x "$ROOT_DIR/scripts/qa/govulncheck.sh" ]; then
	GOVULNCHECK_STRICT=1 bash "$ROOT_DIR/scripts/qa/govulncheck.sh"
fi

echo "[qa:strict] 运行 web 严格检查"
(
	cd "$ROOT_DIR/web"
	pnpm exec eslint --max-warnings=0 --ext .js --ext .jsx src/
	pnpm exec stylelint "src/**/*.{css,scss,sass}" --max-warnings=0

	# 兼容模板差异：只有定义了 test 脚本才执行前端测试。
	if node -e "const fs=require('fs');const pkg=JSON.parse(fs.readFileSync('package.json','utf8'));process.exit(pkg.scripts&&pkg.scripts.test?0:1)"; then
		pnpm test
	else
		echo "[qa:strict] web/package.json 未定义 test，跳过前端测试"
	fi

	pnpm build
)

echo "[qa:strict] 运行 server 严格检查"
(
	cd "$ROOT_DIR/server"
	go test ./...
	make build
)

echo "[qa:strict] 全部通过"
