#!/usr/bin/env bash
set -euo pipefail

print_help() {
	cat <<'USAGE'
用法:
  bash scripts/init-project.sh [--strict] [--project|--template-source]

作用:
  扫描“由模板初始化后的新项目”里仍需处理的模板残留、默认配置与模块裁剪点。

模式:
  --project          按派生项目模式执行；命中“必须处理项”时，--strict 会返回非 0
  --template-source  按模板源仓库模式执行；仍会展示派生项目必改项，但默认不阻断
  默认 auto         根据仓库目录名 / origin remote 自动判断

参数:
  --strict           在派生项目模式下，命中“必须处理项”时返回非 0
  -h, --help         显示帮助

建议流程:
  1) 新项目初始化后先执行: bash scripts/init-project.sh
  2) 完成项目名 / 配置 / 部署方式 / 文档收口
  3) 再执行: bash scripts/init-project.sh --project --strict
USAGE
}

STRICT=0
MODE="auto"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--strict)
		STRICT=1
		;;
	--project)
		MODE="project"
		;;
	--template-source)
		MODE="template-source"
		;;
	-h | --help)
		print_help
		exit 0
		;;
	*)
		echo "[init-project] 不支持的参数: $1" >&2
		print_help
		exit 1
		;;
	esac
	shift
done

ROOT_DIR="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT_DIR"

if command -v rg >/dev/null 2>&1; then
	SEARCH_CMD="rg"
else
	SEARCH_CMD="grep"
fi

detect_mode() {
	if [[ "$MODE" != "auto" ]]; then
		printf '%s' "$MODE"
		return
	fi

	repo_name="$(basename "$ROOT_DIR")"
	origin_url="$(git config --get remote.origin.url 2>/dev/null || true)"

	if [[ "$repo_name" == "webapp-template" ]] || printf '%s' "$origin_url" | grep -Eq '(^|[:/])webapp-template(\.git)?$'; then
		printf '%s' "template-source"
		return
	fi

	printf '%s' "project"
}

EFFECTIVE_MODE="$(detect_mode)"
REQUIRED_COUNT=0
ADVISORY_COUNT=0

scan_pattern() {
	local pattern="$1"
	shift

	if [[ $# -eq 0 ]]; then
		return 0
	fi

	if [[ "$SEARCH_CMD" == "rg" ]]; then
		rg -n --color never --hidden \
			--glob '!**/.git/**' \
			--glob '!**/node_modules/**' \
			--glob '!**/build/**' \
			--glob '!**/dist/**' \
			--glob '!**/coverage/**' \
			--glob '!**/bin/**' \
			-e "$pattern" "$@" 2>/dev/null || true
		return
	fi

	grep -R -n -E "$pattern" "$@" 2>/dev/null || true
}

print_hits() {
	local hits="$1"
	if [[ -z "$hits" ]]; then
		return
	fi

	local total
	total="$(printf '%s\n' "$hits" | awk 'NF{n++} END{print n+0}')"
	printf '%s\n' "$hits" | sed -n '1,12p' | sed 's/^/    - /'
	if [[ "$total" -gt 12 ]]; then
		echo "    ...（其余 $((total - 12)) 条省略）"
	fi
}

report_required() {
	local title="$1"
	local action="$2"
	local hits="$3"
	if [[ -z "$hits" ]]; then
		return
	fi

	if [[ "$EFFECTIVE_MODE" == "project" ]]; then
		REQUIRED_COUNT=$((REQUIRED_COUNT + 1))
		echo "[必须处理] $title"
	else
		echo "[派生项目必改] $title"
	fi
	echo "  处理建议: $action"
	print_hits "$hits"
	echo
}

report_advisory() {
	local title="$1"
	local action="$2"
	local hits="$3"
	if [[ -z "$hits" ]]; then
		return
	fi

	ADVISORY_COUNT=$((ADVISORY_COUNT + 1))
	echo "[建议确认] $title"
	echo "  处理建议: $action"
	print_hits "$hits"
	echo
}

report_existing_paths() {
	local title="$1"
	local action="$2"
	shift 2
	local found=()
	local path
	for path in "$@"; do
		if [[ -e "$path" ]]; then
			found+=("$path")
		fi
	done

	if [[ "${#found[@]}" -eq 0 ]]; then
		return
	fi

	ADVISORY_COUNT=$((ADVISORY_COUNT + 1))
	echo "[建议确认] $title"
	echo "  处理建议: $action"
	for path in "${found[@]}"; do
		echo "    - $path"
	done
	echo
}

echo "[init-project] 仓库根目录: $ROOT_DIR"
echo "[init-project] 当前模式: $EFFECTIVE_MODE"
if [[ "$EFFECTIVE_MODE" == "template-source" ]]; then
	echo "[init-project] 提示: 当前仍是模板源仓库，下面的'派生项目必改'命中属于预期模板位。"
fi
echo

IDENTITY_HITS="$(
	scan_pattern 'webapp-template|react-webapp-template|template-server|compose\.webapp-template|deploy_webapp_template|Project Workspace|Starter Workspace|new-task|your-project' \
		README.md \
		AGENTS.md \
		docs/README.md \
		docs/project-init.md \
		server/README.md \
		server/docs/k8s.md \
		web/package.json \
		web/index.html \
		web/public/index.html \
		web/.env.development \
		web/.env.production \
		web/src/App.jsx \
		web/src/pages/Home/index.jsx \
		server/cmd/server/main.go \
		server/Makefile \
		server/configs/dev/config.yaml \
		server/configs/prod/config.yaml \
		server/deploy
)"
report_required \
	"项目标识 / 服务名 / 页面标题仍保留模板占位" \
	"统一替换项目名、服务名、镜像名、页面标题、Compose / K8s 清单里的占位符，避免新仓库继续带 webapp-template / your-project 语义。" \
	"$IDENTITY_HITS"

SECRET_HITS="$(
	scan_pattern 'YP\*H%k%a7xK1\*q|2@&0kq%qFafA4d|eB6Cc5Mz/OB/WrHyKJMQLnmj160ropjq3j167pkIGUI=|replace-me' \
		server/configs/dev/config.yaml \
		server/configs/prod/config.yaml \
		server/deploy \
		server/cmd/dbcheck/main.go \
		server/cmd/gen-password/main.go
)"
report_required \
	"默认密钥 / 管理员密码仍是模板值" \
	"派生项目必须替换 JWT 密钥、数据库密码、默认管理员密码、镜像仓库凭据与任何示例凭据；这些值只适合模板演示，不应直接进入交付项目。" \
	"$SECRET_HITS"

DEPLOY_HITS="$(
	scan_pattern '47\.84\.12\.211|registry\.xxxx|test_database_atlas|webapp-template-pro|registry\.example\.com|deploy\.example\.com|dashboard\.example\.local|otel-collector\.observability\.svc\.cluster\.local|prometheus:9090' \
		README.md \
		web/index.html \
		web/public/index.html \
		web/.env.production \
		server/configs/dev/config.yaml \
		server/configs/prod/config.yaml \
		server/deploy \
		server/docs/k8s.md
)"
report_required \
	"部署主机 / 网络地址 / 数据库名等仍是模板默认值" \
	"按当前项目的环境改掉远端主机、镜像仓库、K8s 域名、观测地址、数据库名与 base path；不要把模板默认网络参数带到新仓库。" \
	"$DEPLOY_HITS"

DOC_HITS="$(
	scan_pattern '本模板|派生项目|模板默认|初始化后建议替换' \
		AGENTS.md \
		README.md \
		docs/README.md \
		docs/project-init.md \
		scripts/README.md \
		server/README.md \
		server/deploy/README.md \
		server/deploy/compose/prod/README.md \
		server/deploy/dashboard/README.md \
		server/docs/k8s.md \
		web/src/pages/Home/index.jsx
)"
report_required \
	"文档与首页仍保留模板语义" \
	"新项目初始化后，应把 README / AGENTS / 部署文档 / 首页占位文案改成当前项目事实，而不是继续保留'模板/派生项目'措辞。" \
	"$DOC_HITS"

report_existing_paths \
	"仓库仍包含 K8s 相关部署物" \
	"若当前项目明确只走 docker compose，请按需移除 K8s 清单、dashboard 与相关文档；删除时默认移动到系统回收站。" \
	server/deploy/dev \
	server/deploy/prod \
	server/deploy/dashboard \
	server/docs/k8s.md

report_existing_paths \
	"仓库仍包含远端一键发布脚本" \
	"若当前项目不需要 SSH 发布 / 远端增量部署，可移除 publish/deploy/migrate 脚本；若保留，需优先替换远端目录、主机和容器名。" \
	server/deploy/compose/prod/deploy_server.sh \
	server/deploy/compose/prod/publish_server.sh \
	server/deploy/compose/prod/migrate_online.sh

ADMIN_MODULE_HITS="$(
	scan_pattern 'subscription|points\.|invite_code|AdminLevel|transfer_to_admin_id|UserPoints|Subscription|user_expiry_warning_days' \
		server/internal/biz/user_admin.go \
		server/internal/data/jsonrpc.go \
		server/internal/data/model/schema/invitecode.go \
		server/internal/data/model/schema/user.go \
		server/internal/data/model/schema/admin_user.go \
		server/internal/errcode/catalog.go
)"
report_advisory \
	"仓库重新引入了模板默认未保留的业务模块" \
	"模板主干已移除积分 / 订阅 / 管理员层级 / 邀请码等业务模块；若扫描再次命中，请确认这是当前项目的真实需求，而不是模板残留回流。" \
	"$ADMIN_MODULE_HITS"

JAEGER_HITS="$(
	scan_pattern 'jaeger|OTLP|TraceName|traceName' \
		server/deploy/compose/prod/compose.yml \
		server/configs/dev/config.yaml \
		server/configs/prod/config.yaml \
		server/cmd/server/main.go
)"
report_advisory \
	"观测链路仍默认启用 Jaeger / OTLP" \
	"若当前项目不需要自带 Jaeger，可删除 compose 服务与 trace 配置；若保留，至少替换服务名、端口与 Prometheus 地址。" \
	"$JAEGER_HITS"

echo "[init-project] 建议执行顺序:"
echo "  1) bash scripts/init-project.sh"
echo "  2) 完成项目名 / 配置 / 部署方式 / 页面文案收口"
echo "  3) bash scripts/bootstrap.sh"
echo "  4) bash scripts/doctor.sh"
echo "  5) bash scripts/init-project.sh --project --strict"
echo "  6) bash scripts/qa/fast.sh"
echo "  7) bash scripts/qa/full.sh"
echo

if [[ "$EFFECTIVE_MODE" == "project" && "$STRICT" -eq 1 && "$REQUIRED_COUNT" -gt 0 ]]; then
	echo "[init-project] 结果: 发现 $REQUIRED_COUNT 组必须处理项，strict 模式失败。" >&2
	exit 1
fi

if [[ "$REQUIRED_COUNT" -eq 0 && "$ADVISORY_COUNT" -eq 0 ]]; then
	echo "[init-project] 结果: 未发现需要处理的初始化残留。"
	exit 0
fi

if [[ "$EFFECTIVE_MODE" == "project" ]]; then
	echo "[init-project] 结果: 必须处理项 $REQUIRED_COUNT 组，建议确认项 $ADVISORY_COUNT 组。"
else
	echo "[init-project] 结果: 已列出派生项目必改项与建议确认项，供后续初始化仓库时直接使用。"
fi
