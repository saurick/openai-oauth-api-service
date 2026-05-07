#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

collect_staged_files() {
	STAGED_FILES=()
	while IFS= read -r -d '' file; do
		STAGED_FILES+=("$file")
	done < <(git diff --cached --name-only --diff-filter=ACMR -z)
}

build_web_targets() {
	PRETTIER_TARGETS_WEB=()
	PRETTIER_TARGETS_ROOT=()
	ESLINT_TARGETS_WEB=()
	ESLINT_TARGETS_ROOT=()

	for file in "${STAGED_FILES[@]}"; do
		[[ "$file" =~ ^web/ ]] || continue

		case "$file" in
		web/node_modules/* | web/build/*)
			continue
			;;
		esac

		case "$file" in
		*.js | *.jsx | *.ts | *.tsx | *.cjs | *.mjs | *.css | *.scss | *.sass | *.json | *.md | *.html | *.yml | *.yaml)
			PRETTIER_TARGETS_WEB+=("${file#web/}")
			PRETTIER_TARGETS_ROOT+=("$file")
			;;
		esac

		if [[ "$file" =~ ^web/src/.*\.(js|jsx)$ ]]; then
			ESLINT_TARGETS_WEB+=("${file#web/}")
			ESLINT_TARGETS_ROOT+=("$file")
		fi
	done
}

add_go_target() {
	local target="$1"
	local existing
	for existing in "${GO_TARGETS[@]:-}"; do
		[[ "$existing" == "$target" ]] && return
	done
	GO_TARGETS+=("$target")
}

detect_go_targets() {
	HAS_GO_CHANGES=0
	RUN_GO_ALL=0
	GO_TARGETS=()

	local file rel dir
	for file in "${STAGED_FILES[@]}"; do
		case "$file" in
		server/go.mod | server/go.sum | .golangci.yml | .golangci.yaml | .golangci.toml | .golangci.json)
			HAS_GO_CHANGES=1
			RUN_GO_ALL=1
			;;
		esac

		if [[ "$file" =~ ^server/.*\.go$ ]]; then
			HAS_GO_CHANGES=1
			rel="${file#server/}"
			dir="$(dirname "$rel")"
			if [[ "$dir" == "." ]]; then
				add_go_target "./"
			else
				add_go_target "./$dir"
			fi
		fi
	done

	GO_VET_ARGS=()
	GOLANGCI_ARGS=()
	if [[ "$HAS_GO_CHANGES" -eq 1 ]]; then
		if [[ "$RUN_GO_ALL" -eq 1 || "${#GO_TARGETS[@]}" -eq 0 ]]; then
			GO_VET_ARGS=(./...)
			GOLANGCI_ARGS=(./...)
		else
			GO_VET_ARGS=("${GO_TARGETS[@]}")
			GOLANGCI_ARGS=("${GO_TARGETS[@]}")
		fi
	fi
}

is_yaml_ignored() {
	local file="$1"
	case "$file" in
	.git/* | web/node_modules/* | web/build/* | server/bin/* | web/pnpm-lock.yaml | .playwright-cli/*)
		return 0
		;;
	*)
		return 1
		;;
	esac
}

detect_yaml_targets() {
	YAML_TARGETS=()
	local file
	for file in "${STAGED_FILES[@]}"; do
		case "$file" in
		*.yml | *.yaml)
			if is_yaml_ignored "$file"; then
				continue
			fi
			YAML_TARGETS+=("$file")
			;;
		esac
	done
}

detect_shell_targets() {
	SHFMT_TARGETS_ROOT=()
	local file
	for file in "${STAGED_FILES[@]}"; do
		case "$file" in
		scripts/* | .githooks/*) ;;
		*)
			continue
			;;
		esac

		case "$file" in
		*.sh | .githooks/pre-commit | .githooks/pre-push | .githooks/commit-msg)
			[[ -f "$file" ]] || continue
			SHFMT_TARGETS_ROOT+=("$file")
			;;
		esac
	done
}

collect_staged_files
if [[ "${#STAGED_FILES[@]}" -eq 0 ]]; then
	exit 0
fi

build_web_targets
if [[ "${#PRETTIER_TARGETS_WEB[@]}" -gt 0 || "${#ESLINT_TARGETS_WEB[@]}" -gt 0 ]]; then
	if ! command -v pnpm >/dev/null 2>&1; then
		echo "[pre-commit] 未找到 pnpm，请先安装 pnpm"
		exit 1
	fi
fi

if [[ "${#PRETTIER_TARGETS_WEB[@]}" -gt 0 ]]; then
	echo "[pre-commit] 运行 Prettier（仅暂存文件）"
	(
		cd "$ROOT_DIR/web"
		pnpm exec prettier --write "${PRETTIER_TARGETS_WEB[@]}"
	)
	git add -- "${PRETTIER_TARGETS_ROOT[@]}"
fi

if [[ "${#ESLINT_TARGETS_WEB[@]}" -gt 0 ]]; then
	echo "[pre-commit] 运行 ESLint --fix（仅暂存文件）"
	(
		cd "$ROOT_DIR/web"
		pnpm exec eslint --fix --ext .js --ext .jsx "${ESLINT_TARGETS_WEB[@]}"
	)
	git add -- "${ESLINT_TARGETS_ROOT[@]}"
fi

collect_staged_files
if [[ "${#STAGED_FILES[@]}" -eq 0 ]]; then
	exit 0
fi

detect_shell_targets
if [[ "${#SHFMT_TARGETS_ROOT[@]}" -gt 0 ]]; then
	echo "[pre-commit] 运行 shfmt（仅暂存脚本）"
	SHFMT_STRICT=1 bash "$ROOT_DIR/scripts/qa/shfmt.sh" "${SHFMT_TARGETS_ROOT[@]}"
	git add -- "${SHFMT_TARGETS_ROOT[@]}"
fi

collect_staged_files
if [[ "${#STAGED_FILES[@]}" -eq 0 ]]; then
	exit 0
fi

echo "[pre-commit] 运行 shellcheck"
SHELLCHECK_STRICT=1 bash "$ROOT_DIR/scripts/qa/shellcheck.sh"

if [[ -f "$ROOT_DIR/scripts/gen-error-codes.mjs" ]]; then
	if ! command -v node >/dev/null 2>&1; then
		echo "[pre-commit] 未找到 node，无法同步前端错误码生成产物"
		exit 1
	fi

	# 先同步生成产物，再做守卫检查，避免把目录真源与前端常量拆开提交。
	echo "[pre-commit] 同步前端错误码生成产物"
	node "$ROOT_DIR/scripts/gen-error-codes.mjs"
	if [[ -f "$ROOT_DIR/web/src/common/consts/errorCodes.generated.js" ]]; then
		git add -- "$ROOT_DIR/web/src/common/consts/errorCodes.generated.js"
	fi
fi

echo "[pre-commit] 运行错误码生成同步检查"
bash "$ROOT_DIR/scripts/qa/error-code-sync.sh"

echo "[pre-commit] 运行错误码魔法数字检查（仅暂存文件）"
ERROR_CODE_GUARD_STAGED_ONLY=1 bash "$ROOT_DIR/scripts/qa/error-codes.sh"

echo "[pre-commit] 运行 gitleaks（仅暂存文件）"
SECRETS_STRICT=1 SECRETS_STAGED_ONLY=1 bash "$ROOT_DIR/scripts/qa/secrets.sh"

detect_go_targets
if [[ "$HAS_GO_CHANGES" -eq 1 ]]; then
	echo "[pre-commit] 检测到 Go 相关改动，运行 go vet（仅改动包）"
	bash "$ROOT_DIR/scripts/qa/go-vet.sh" "${GO_VET_ARGS[@]}"

	echo "[pre-commit] 检测到 Go 相关改动，运行 golangci-lint（仅新增问题）"
	GOLANGCI_STRICT=1 GOLANGCI_ONLY_NEW=1 bash "$ROOT_DIR/scripts/qa/golangci-lint.sh" "${GOLANGCI_ARGS[@]}"
fi

detect_yaml_targets
if [[ "${#YAML_TARGETS[@]}" -gt 0 ]]; then
	echo "[pre-commit] 运行 yamllint（仅暂存 YAML）"
	YAMLLINT_STRICT=1 bash "$ROOT_DIR/scripts/qa/yamllint.sh" "${YAML_TARGETS[@]}"
fi

echo "[pre-commit] 完成"
