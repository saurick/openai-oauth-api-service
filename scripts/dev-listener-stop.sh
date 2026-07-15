#!/usr/bin/env bash
set -euo pipefail

project_root="${1:-}"
shift || true
if [[ -z "${project_root}" || "$#" -eq 0 ]]; then
	echo "usage: $0 <project-root> <port>..." >&2
	exit 2
fi
if ! command -v lsof >/dev/null 2>&1; then
	echo "ERROR: lsof is required; refusing unsafe port-based process cleanup" >&2
	exit 1
fi

project_root="$(cd "${project_root}" && pwd -P)"
owned_pids=()
listener_cwd() {
	lsof -a -p "$1" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | head -n 1
}
is_owned_cwd() {
	case "$1" in
	"${project_root}" | "${project_root}"/*) return 0 ;;
	*) return 1 ;;
	esac
}

for port in "$@"; do
	[[ "${port}" =~ ^[0-9]+$ ]] || {
		echo "ERROR: invalid development port: ${port}" >&2
		exit 2
	}
	while IFS= read -r pid; do
		[[ -n "${pid}" ]] || continue
		cwd="$(listener_cwd "${pid}")"
		if ! is_owned_cwd "${cwd}"; then
			command="$(ps -p "${pid}" -o command= 2>/dev/null || true)"
			echo "ERROR: port ${port} belongs to foreign process pid=${pid} cwd=${cwd:-unknown} command=${command:-unknown}" >&2
			echo "ERROR: no listener was stopped" >&2
			exit 1
		fi
		if [[ " ${owned_pids[*]} " != *" ${pid} "* ]]; then
			owned_pids+=("${pid}")
		fi
	done < <(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)
done

if [[ "${#owned_pids[@]}" -eq 0 ]]; then
	echo ">>> no owned development listener found"
	exit 0
fi
echo ">>> stopping owned development listener(s): ${owned_pids[*]}"
kill "${owned_pids[@]}"
for _ in 1 2 3 4 5; do
	remaining=()
	for pid in "${owned_pids[@]}"; do
		if kill -0 "${pid}" 2>/dev/null; then
			remaining+=("${pid}")
		fi
	done
	[[ "${#remaining[@]}" -eq 0 ]] && exit 0
	sleep 1
done
for pid in "${remaining[@]}"; do
	cwd="$(listener_cwd "${pid}")"
	if is_owned_cwd "${cwd}"; then
		echo ">>> force stopping owned development listener pid=${pid}"
		kill -9 "${pid}"
	else
		echo "ERROR: pid=${pid} changed ownership; refusing force kill" >&2
		exit 1
	fi
done
