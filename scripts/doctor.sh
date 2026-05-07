#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/doctor.sh

作用:
  检查本地开发环境是否满足仓库脚本与门禁运行要求

检查项:
  - 必需命令: git / node / pnpm / go
  - 可选命令: gitleaks / shellcheck / golangci-lint / yamllint / shfmt / govulncheck
  - hooks 路径与关键脚本存在性
  - Node 版本与版本文件（.n-node-version/.node-version/.nvmrc）一致性（仅提示）
  - 初始化扫描脚本存在性（scripts/init-project.sh）
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	print_help
	exit 0
fi

if [[ $# -gt 0 ]]; then
	echo "[doctor] 不支持的参数: $*"
	print_help
	exit 1
fi

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

missing=0
warns=0

print_cmd_version() {
	case "$1" in
	git)
		git --version
		;;
	node)
		node -v
		;;
	pnpm)
		pnpm -v
		;;
	go)
		go version
		;;
	gitleaks)
		gitleaks version 2>/dev/null | head -n 1 || true
		;;
	shellcheck)
		shellcheck --version 2>/dev/null | head -n 1 || true
		;;
	golangci-lint)
		golangci-lint version 2>/dev/null | head -n 1 || true
		;;
	yamllint)
		yamllint --version 2>/dev/null | head -n 1 || true
		;;
	shfmt)
		shfmt --version 2>/dev/null || true
		;;
	govulncheck)
		govulncheck -version 2>/dev/null || true
		;;
	esac
}

echo "[doctor] 检查必需命令"
for cmd in git node pnpm go; do
	if command -v "$cmd" >/dev/null 2>&1; then
		printf "  - [OK] %s: " "$cmd"
		print_cmd_version "$cmd"
	else
		echo "  - [缺失] $cmd"
		missing=1
	fi
done

echo "[doctor] 检查可选命令"
for cmd in gitleaks shellcheck golangci-lint yamllint shfmt govulncheck; do
	if command -v "$cmd" >/dev/null 2>&1; then
		printf "  - [OK] %s: " "$cmd"
		print_cmd_version "$cmd"
	else
		echo "  - [可选缺失] $cmd"
		warns=1
	fi
done

hooks_path="$(git config --get core.hooksPath || true)"
if [[ "$hooks_path" == ".githooks" ]]; then
	echo "[doctor] hooksPath 正常：.githooks"
else
	echo "[doctor] hooksPath 当前为：${hooks_path:-<未设置>}（建议执行 scripts/setup-git-hooks.sh）"
	warns=1
fi

echo "[doctor] 检查关键脚本存在性"
required_files=(
	scripts/setup-git-hooks.sh
	scripts/bootstrap.sh
	scripts/init-project.sh
	scripts/qa/fast.sh
	scripts/qa/full.sh
	scripts/qa/strict.sh
	scripts/qa/db-guard.sh
	scripts/qa/secrets.sh
	scripts/qa/shellcheck.sh
	scripts/qa/go-vet.sh
	scripts/qa/golangci-lint.sh
	scripts/qa/yamllint.sh
	scripts/qa/shfmt.sh
	scripts/qa/govulncheck.sh
	scripts/git-hooks/pre-commit.sh
	scripts/git-hooks/pre-push.sh
	scripts/git-hooks/commit-msg.sh
	.githooks/pre-commit
	.githooks/pre-push
	.githooks/commit-msg
)

for f in "${required_files[@]}"; do
	if [[ -f "$f" ]]; then
		echo "  - [OK] $f"
	else
		echo "  - [缺失] $f"
		missing=1
	fi
done

if command -v node >/dev/null 2>&1; then
	version_file=""
	for f in .n-node-version .node-version .nvmrc; do
		if [[ -f "$f" ]]; then
			version_file="$f"
			break
		fi
	done

	if [[ -n "$version_file" ]]; then
		expected_node="$(tr -d ' \t\r\n' <"$version_file" | sed 's/^v//')"
		current_node="$(node -v | sed 's/^v//')"
		if [[ -n "$expected_node" ]] && [[ "$expected_node" != "$current_node" ]]; then
			echo "[doctor] Node 版本提示：当前 ${current_node}，${version_file} 期望 ${expected_node}"
			warns=1
		fi
	fi
fi

if [[ "$missing" -ne 0 ]]; then
	echo "[doctor] 存在缺失项，请先修复后再继续"
	exit 1
fi

if [[ "$warns" -ne 0 ]]; then
	echo "[doctor] 检查通过（含提示项）"
	exit 0
fi

echo "[doctor] 全部通过"
