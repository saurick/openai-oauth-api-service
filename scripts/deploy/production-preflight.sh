#!/usr/bin/env bash
set -euo pipefail

fail() {
	echo "[production-preflight] ERROR: $*" >&2
	exit 1
}
warn() { echo "[production-preflight] WARN: $*" >&2; }
ok() { echo "[production-preflight] ok: $*"; }

trim() {
	local value="$1"
	value="${value#"${value%%[![:space:]]*}"}"
	value="${value%"${value##*[![:space:]]}"}"
	value="${value%\"}"
	value="${value#\"}"
	value="${value%\'}"
	value="${value#\'}"
	printf "%s" "$value"
}

root_dir="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
compose_dir="server/deploy/compose/prod"
env_file=""
mode="runtime-env"
runtime_check=0
skip_compose_config=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--env-file)
		env_file="${2:-}"
		shift 2
		;;
	--compose-dir)
		compose_dir="${2:-}"
		shift 2
		;;
	--runtime)
		runtime_check=1
		shift
		;;
	--example)
		mode="example"
		env_file="$compose_dir/.env.example"
		shift
		;;
	--skip-compose-config)
		skip_compose_config=1
		shift
		;;
	-h | --help)
		sed -n '1,80p' "$0" | sed -n '1,40p'
		exit 0
		;;
	*) fail "不支持的参数: $1" ;;
	esac
done

cd "$root_dir"
compose_file="$compose_dir/compose.yml"
deploy_readme="server/deploy/README.md"
compose_readme="$compose_dir/README.md"
[[ -n "$env_file" ]] || fail "必须传入 --env-file，或使用 --example 只检查样例结构"
[[ -f "$env_file" ]] || fail "env 文件不存在: $env_file"
[[ -f "$compose_file" ]] || fail "compose 文件不存在: $compose_file"
[[ -f "$deploy_readme" ]] || fail "部署说明不存在: $deploy_readme"
[[ -f "$compose_readme" ]] || fail "Compose 说明不存在: $compose_readme"

required_keys=(
	PROJECT_SLUG APP_IMAGE TZ POSTGRES_DSN POSTGRES_PASSWORD POSTGRES_DB POSTGRES_USER
	POSTGRES_DATA_DIR OAUTH_API_JWT_SECRET OAUTH_API_ADMIN_USERNAME OAUTH_API_ADMIN_PASSWORD
	CODEX_HOST_HOME CODEX_CONTAINER_HOME CODEX_UPSTREAM_MODE CODEX_UPSTREAM_FALLBACK_ENABLED
	APP_HTTP_PORT APP_GRPC_PORT
)

normalized_env="$(mktemp)"
trap 'rm -f "$normalized_env"' EXIT
while IFS='=' read -r raw_key raw_value; do
	key="$(trim "$(printf '%s' "$raw_key" | sed -E 's/^[[:space:]]*export[[:space:]]+//')")"
	value="$(trim "${raw_value:-}")"
	[[ -z "$key" || "$key" =~ ^# ]] && continue
	printf '%s=%s\n' "$key" "$value" >>"$normalized_env"
done < <(grep -vE '^[[:space:]]*(#|$)' "$env_file" || true)

has_value() {
	local key="$1"
	awk -F= -v key="$key" '$1 == key { found = 1 } END { exit !found }' "$normalized_env"
}
value_of() {
	local key="$1"
	awk -F= -v key="$key" '$1 == key { sub(/^[^=]*=/, ""); value = $0 } END { print value }' "$normalized_env"
}

for key in "${required_keys[@]}"; do
	has_value "$key" || fail "缺少必需变量: $key"
done
ok "env 必需变量齐全"

if [[ "$mode" == "example" ]]; then
	ok "example 模式仅检查结构，不作为生产放行"
else
	placeholder_pattern='(change-this|placeholder|replace-me|replace-with|<release-tag>|example\.invalid)'
	for key in POSTGRES_DSN POSTGRES_PASSWORD APP_IMAGE POSTGRES_DATA_DIR OAUTH_API_JWT_SECRET; do
		value="$(value_of "$key")"
		if grep -Eiq "$placeholder_pattern" <<<"$value"; then
			fail "$key 仍包含 placeholder"
		fi
	done

	app_image="$(value_of APP_IMAGE)"
	jwt_secret="$(value_of OAUTH_API_JWT_SECRET)"
	postgres_dsn="$(value_of POSTGRES_DSN)"
	fallback_enabled="$(value_of CODEX_UPSTREAM_FALLBACK_ENABLED)"
	oauth_client_id="$(value_of OAUTH_API_OAUTH_CLIENT_ID)"
	oauth_client_secret="$(value_of OAUTH_API_OAUTH_CLIENT_SECRET)"

	[[ "$app_image" != *":dev" && "$app_image" != *":latest" ]] || fail "APP_IMAGE 不能使用 :dev 或 :latest"
	[[ "${#jwt_secret}" -ge 32 ]] || fail "OAUTH_API_JWT_SECRET 至少需要 32 字符"
	[[ "$postgres_dsn" == postgres://* || "$postgres_dsn" == postgresql://* ]] || fail "POSTGRES_DSN 必须是 postgres/postgresql URL"
	[[ "$fallback_enabled" == "false" ]] || fail "CODEX_UPSTREAM_FALLBACK_ENABLED 生产默认必须为 false"
	if [[ -n "$oauth_client_id" && -z "$oauth_client_secret" ]]; then
		fail "已配置 OAUTH_API_OAUTH_CLIENT_ID 时必须配置 OAUTH_API_OAUTH_CLIENT_SECRET"
	fi
	ok "生产 secret、镜像 tag 和 Codex upstream 边界通过"
fi

if grep -Eq '^[[:space:]]+build:' "$compose_file"; then
	fail "生产 Compose 不允许包含 build:，低配服务器只 docker load / restart"
fi
if grep -qs 'arigaio/atlas' "$compose_file" "$compose_dir"/compose.*.yml 2>/dev/null; then
	fail "生产 Compose 主路径不允许使用 arigaio/atlas 容器执行 migration"
fi
grep -q '/usr/local/bin/atlas' "$deploy_readme" "$compose_readme" || fail "部署文档必须明确宿主机 /usr/local/bin/atlas migration 主路径"
grep -q 'flock' "$deploy_readme" "$compose_readme" || fail "部署文档必须明确 flock 串行化 migration"
ok "Compose 和 migration 文档边界通过"

if [[ "$skip_compose_config" -eq 0 ]]; then
	if docker compose version >/dev/null 2>&1; then
		docker compose --env-file "$env_file" -f "$compose_file" config -q
		ok "docker compose config -q 通过"
	elif command -v docker-compose >/dev/null 2>&1; then
		docker-compose --env-file "$env_file" -f "$compose_file" config -q
		ok "docker-compose config -q 通过"
	elif [[ "$mode" == "example" ]]; then
		warn "未找到 docker compose，example 模式跳过 compose config"
	else
		fail "未找到 docker compose / docker-compose"
	fi
fi

if [[ "$runtime_check" -eq 1 ]]; then
	command -v docker >/dev/null 2>&1 || fail "--runtime 需要 docker"
	if docker compose version >/dev/null 2>&1; then
		compose_cmd=(docker compose --env-file "$env_file" -f "$compose_file")
	elif command -v docker-compose >/dev/null 2>&1; then
		compose_cmd=(docker-compose --env-file "$env_file" -f "$compose_file")
	else
		fail "--runtime 需要 docker compose / docker-compose"
	fi
	for service in postgres app-server; do
		cid="$("${compose_cmd[@]}" ps -q "$service" 2>/dev/null | head -n1 || true)"
		[[ -n "$cid" ]] || fail "运行态缺少 Compose 服务: $service"
	done
	ok "Compose 运行服务存在"
	if command -v curl >/dev/null 2>&1; then
		app_port="$(value_of APP_HTTP_PORT)"
		app_port="${app_port:-8400}"
		curl -fsS "http://127.0.0.1:${app_port}/healthz" >/dev/null || fail "healthz 失败"
		curl -fsS "http://127.0.0.1:${app_port}/readyz" >/dev/null || fail "readyz 失败"
		ok "healthz / readyz 通过"
	else
		warn "未找到 curl，跳过 healthz / readyz"
	fi
fi

echo "[production-preflight] all checks passed"
