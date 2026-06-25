#!/usr/bin/env python3
"""Check host-side Codex runtime and the deployed OAuth API service."""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


COMPOSE_DIR = Path(os.getenv("OAUTH_API_COMPOSE_DIR", "/data/openai-oauth-api-service/compose"))
CODEX_BIN = os.getenv("CODEX_RUNTIME_BIN", "codex")
CODEX_MODE = os.getenv("CODEX_RUNTIME_MODE", "auto")
CONTAINER = os.getenv("OAUTH_API_CONTAINER", "openai-oauth-api-service-server")
STATE_FILE = Path(
    os.getenv("CODEX_RUNTIME_HEALTH_STATE_FILE", "/var/lib/codex-runtime-health/state.json")
)
LOG_FILE = Path(os.getenv("CODEX_RUNTIME_HEALTH_LOG_FILE", "/var/log/codex-runtime-health.log"))
UPGRADE_HISTORY_FILE = Path(
    os.getenv(
        "CODEX_RUNTIME_UPGRADE_HISTORY_FILE",
        "/var/lib/codex-runtime-health/upgrade-history.jsonl",
    )
)
LAST_UPGRADE_FILE = Path(
    os.getenv(
        "CODEX_RUNTIME_LAST_UPGRADE_FILE",
        "/var/lib/codex-runtime-health/last-upgrade.json",
    )
)
FAILOVER_CHECK = os.getenv(
    "CODEX_RUNTIME_FAILOVER_CHECK",
    "/usr/local/sbin/codex-upstream-proxy-failover.py --check",
)
LATEST_VERSION_COMMAND = os.getenv("CODEX_RUNTIME_LATEST_VERSION_COMMAND", "")
UPGRADE_COMMAND = os.getenv("CODEX_RUNTIME_UPGRADE_COMMAND", "")
REQUEST_TIMEOUT_SECONDS = float(os.getenv("CODEX_RUNTIME_HEALTH_TIMEOUT_SECONDS", "20"))
DISK_WARN_PERCENT = float(os.getenv("CODEX_RUNTIME_DISK_WARN_PERCENT", "90"))
COMMAND_OUTPUT_MAX_CHARS = int(os.getenv("CODEX_RUNTIME_COMMAND_OUTPUT_MAX_CHARS", "4000"))


def read_compose_env() -> dict[str, str]:
    env_path = COMPOSE_DIR / ".env"
    values: dict[str, str] = {}
    if not env_path.exists():
        return values
    for raw_line in env_path.read_text(errors="ignore").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip().strip('"').strip("'")
    return values


COMPOSE_ENV = read_compose_env()
APP_PORT = os.getenv("OAUTH_API_APP_PORT") or COMPOSE_ENV.get("APP_HTTP_PORT") or "8400"
APP_BASE_URL = os.getenv("OAUTH_API_BASE_URL", f"http://127.0.0.1:{APP_PORT}").rstrip("/")


def now_iso() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%S%z")


def run_command(command: list[str] | str, timeout: float = REQUEST_TIMEOUT_SECONDS) -> dict[str, Any]:
    try:
        result = subprocess.run(
            command,
            shell=isinstance(command, str),
            check=False,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        return {
            "ok": result.returncode == 0,
            "returncode": result.returncode,
            "stdout": result.stdout.strip(),
            "stderr": result.stderr.strip(),
        }
    except FileNotFoundError as exc:
        return {"ok": False, "returncode": 127, "stdout": "", "stderr": str(exc)}
    except subprocess.TimeoutExpired as exc:
        return {
            "ok": False,
            "returncode": 124,
            "stdout": (exc.stdout or "").strip() if isinstance(exc.stdout, str) else "",
            "stderr": f"timeout after {timeout}s",
        }


def command_label(command: list[str] | str) -> str:
    return command if isinstance(command, str) else " ".join(command)


def truncate_text(value: str, limit: int = COMMAND_OUTPUT_MAX_CHARS) -> str:
    if len(value) <= limit:
        return value
    return value[:limit] + f"...[truncated {len(value) - limit} chars]"


def compact_result(result: dict[str, Any] | None) -> dict[str, Any] | None:
    if result is None:
        return None
    compact: dict[str, Any] = {}
    for key, value in result.items():
        if isinstance(value, str):
            compact[key] = truncate_text(value)
        else:
            compact[key] = value
    return compact


def get_codex_version() -> dict[str, Any]:
    if CODEX_MODE not in {"auto", "host", "container"}:
        return {"ok": False, "mode": CODEX_MODE, "error": "invalid CODEX_RUNTIME_MODE"}

    host_path = shutil.which(CODEX_BIN)
    if CODEX_MODE in {"auto", "host"} and host_path:
        result = run_command([host_path, "--version"], timeout=10)
        result.update({"mode": "host", "bin": host_path})
        return result
    if CODEX_MODE == "host":
        return {"ok": False, "mode": "host", "bin": CODEX_BIN, "error": "not found"}

    docker = shutil.which("docker")
    if CODEX_MODE in {"auto", "container"} and docker:
        result = run_command([docker, "exec", CONTAINER, CODEX_BIN, "--version"], timeout=10)
        result.update({"mode": "container", "bin": CODEX_BIN, "container": CONTAINER})
        return result
    return {"ok": False, "mode": CODEX_MODE, "bin": CODEX_BIN, "error": "docker not found"}


def extract_semver(output: str) -> str:
    for token in reversed(output.replace("\n", " ").split()):
        if token and token[0].isdigit():
            return token
    return ""


def resolve_latest_command(version: dict[str, Any]) -> list[str] | str | None:
    if LATEST_VERSION_COMMAND:
        return LATEST_VERSION_COMMAND
    if version.get("mode") == "container":
        docker = shutil.which("docker")
        if not docker:
            return None
        return [docker, "exec", CONTAINER, "npm", "view", "@openai/codex", "version"]
    if version.get("mode") == "host":
        npm = shutil.which("npm")
        if not npm:
            return None
        return [npm, "view", "@openai/codex", "version"]
    return None


def resolve_upgrade_command(version: dict[str, Any]) -> list[str] | str | None:
    if UPGRADE_COMMAND:
        return UPGRADE_COMMAND
    if version.get("mode") == "container":
        docker = shutil.which("docker")
        if not docker:
            return None
        return [docker, "exec", CONTAINER, "npm", "install", "-g", "@openai/codex@latest"]
    if version.get("mode") == "host":
        npm = shutil.which("npm")
        if not npm:
            return None
        return [npm, "install", "-g", "@openai/codex@latest"]
    return None


def http_get(path: str) -> dict[str, Any]:
    url = APP_BASE_URL + path
    request = urllib.request.Request(url, method="GET", headers={"User-Agent": "codex-runtime-health"})
    started = time.time()
    try:
        with urllib.request.urlopen(request, timeout=REQUEST_TIMEOUT_SECONDS) as response:
            body = response.read(1024 * 1024).decode("utf-8", errors="replace")
            return {
                "ok": 200 <= response.status < 300,
                "status": response.status,
                "body": body,
                "elapsed_ms": int((time.time() - started) * 1000),
            }
    except urllib.error.HTTPError as exc:
        body = exc.read(4096).decode("utf-8", errors="replace")
        return {
            "ok": False,
            "status": exc.code,
            "body": body,
            "elapsed_ms": int((time.time() - started) * 1000),
            "error": str(exc),
        }
    except Exception as exc:
        return {
            "ok": False,
            "status": None,
            "body": "",
            "elapsed_ms": int((time.time() - started) * 1000),
            "error": str(exc),
        }


def add_check(report: dict[str, Any], name: str, status: str, details: dict[str, Any]) -> None:
    report["checks"].append({"name": name, "status": status, "details": details})
    rank = {"ok": 0, "warn": 1, "fail": 2}
    if rank[status] > rank[report["status"]]:
        report["status"] = status


def check_codex(report: dict[str, Any]) -> dict[str, Any]:
    version = get_codex_version()
    add_check(report, "codex_binary", "ok" if version["ok"] else "fail", {"version": version})

    latest_command = resolve_latest_command(version)
    if not latest_command:
        add_check(
            report,
            "codex_latest_version",
            "warn",
            {"skipped": True, "reason": "npm latest version command is not available"},
        )
        return {"version": version, "latest": None, "update_available": False}

    latest = run_command(latest_command, timeout=REQUEST_TIMEOUT_SECONDS)
    current_version = extract_semver(version.get("stdout", ""))
    latest_version = extract_semver(latest.get("stdout", ""))
    status = "ok" if latest["ok"] else "warn"
    update_available = bool(latest["ok"] and current_version and latest_version and current_version != latest_version)
    details = {
        "command": command_label(latest_command),
        "current": current_version,
        "latest": latest_version,
        "result": latest,
    }
    if update_available:
        status = "warn"
        details["update_available"] = True
    add_check(report, "codex_latest_version", status, details)
    return {"version": version, "latest": latest, "update_available": update_available}


def check_http(report: dict[str, Any], strict_balance: bool) -> None:
    healthz = http_get("/healthz")
    add_check(report, "healthz", "ok" if healthz["ok"] and healthz["body"].strip() == "ok" else "fail", healthz)

    readyz = http_get("/readyz")
    add_check(report, "readyz", "ok" if readyz["ok"] and readyz["body"].strip() == "ready" else "fail", readyz)

    balance = http_get("/public/codex/balance")
    balance_status = "ok" if balance["ok"] else "fail"
    try:
        parsed = json.loads(balance.get("body") or "{}")
        balance["parsed"] = parsed
        if parsed.get("stale") is True and balance_status == "ok":
            balance_status = "warn"
    except json.JSONDecodeError:
        if balance["ok"]:
            balance_status = "warn"
    body = balance.pop("body", "")
    if body:
        balance["body_bytes"] = len(body.encode("utf-8"))
        balance["body_excerpt"] = body[:200]
    if balance_status == "warn" and strict_balance:
        balance_status = "fail"
    add_check(report, "codex_balance", balance_status, balance)


def check_docker(report: dict[str, Any]) -> None:
    docker = shutil.which("docker")
    if not docker:
        add_check(report, "docker_container", "warn", {"skipped": True, "reason": "docker not found"})
        return
    result = run_command(
        [docker, "inspect", "-f", "{{.State.Running}} {{.Config.Image}}", CONTAINER],
        timeout=10,
    )
    status = "ok" if result["ok"] and result["stdout"].startswith("true ") else "fail"
    add_check(report, "docker_container", status, {"container": CONTAINER, "result": result})


def check_disk(report: dict[str, Any]) -> None:
    usage = shutil.disk_usage("/")
    used_percent = round((usage.used / usage.total) * 100, 2)
    status = "warn" if used_percent >= DISK_WARN_PERCENT else "ok"
    add_check(
        report,
        "root_disk",
        status,
        {
            "path": "/",
            "used_percent": used_percent,
            "total_bytes": usage.total,
            "used_bytes": usage.used,
            "free_bytes": usage.free,
            "warn_percent": DISK_WARN_PERCENT,
        },
    )


def check_failover(report: dict[str, Any]) -> None:
    if not FAILOVER_CHECK:
        add_check(report, "codex_failover", "warn", {"skipped": True, "reason": "disabled"})
        return
    result = run_command(FAILOVER_CHECK, timeout=REQUEST_TIMEOUT_SECONDS)
    add_check(report, "codex_failover", "ok" if result["ok"] else "warn", {"command": FAILOVER_CHECK, "result": result})


def write_report(report: dict[str, Any]) -> None:
    STATE_FILE.parent.mkdir(parents=True, exist_ok=True)
    tmp = STATE_FILE.with_suffix(".tmp")
    tmp.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    tmp.replace(STATE_FILE)

    try:
        LOG_FILE.parent.mkdir(parents=True, exist_ok=True)
        with LOG_FILE.open("a") as handle:
            handle.write(json.dumps(report, ensure_ascii=False, sort_keys=True) + "\n")
    except PermissionError:
        pass


def write_upgrade_event(event: dict[str, Any]) -> None:
    event["recorded_at"] = now_iso()
    event["history_file"] = str(UPGRADE_HISTORY_FILE)
    event["last_file"] = str(LAST_UPGRADE_FILE)
    UPGRADE_HISTORY_FILE.parent.mkdir(parents=True, exist_ok=True)
    LAST_UPGRADE_FILE.parent.mkdir(parents=True, exist_ok=True)

    tmp = LAST_UPGRADE_FILE.with_suffix(".tmp")
    tmp.write_text(json.dumps(event, ensure_ascii=False, indent=2) + "\n")
    tmp.replace(LAST_UPGRADE_FILE)

    with UPGRADE_HISTORY_FILE.open("a") as handle:
        handle.write(json.dumps(event, ensure_ascii=False, sort_keys=True) + "\n")


def summarize_checks(report: dict[str, Any]) -> dict[str, str]:
    return {item["name"]: item["status"] for item in report.get("checks", [])}


def non_ok_checks(report: dict[str, Any]) -> list[dict[str, Any]]:
    return [
        {"name": item["name"], "status": item["status"], "details": item.get("details", {})}
        for item in report.get("checks", [])
        if item.get("status") != "ok"
    ]


def perform_upgrade(report: dict[str, Any], dry_run: bool, only_if_update: bool) -> dict[str, Any]:
    before = get_codex_version()
    event: dict[str, Any] = {
        "event": "codex_runtime_upgrade",
        "started_at": now_iso(),
        "mode": before.get("mode"),
        "container": before.get("container"),
        "only_if_update": only_if_update,
        "dry_run": dry_run,
        "before": compact_result(before),
        "before_version": extract_semver(before.get("stdout", "")),
    }
    latest_command = resolve_latest_command(before)
    latest: dict[str, Any] | None = None
    update_available = True
    if latest_command:
        latest = run_command(latest_command, timeout=REQUEST_TIMEOUT_SECONDS)
        current_version = extract_semver(before.get("stdout", ""))
        latest_version = extract_semver(latest.get("stdout", ""))
        event.update(
            {
                "latest_command": command_label(latest_command),
                "latest_result": compact_result(latest),
                "current_version": current_version,
                "latest_version": latest_version,
            }
        )
        if only_if_update and not latest["ok"]:
            event.update(
                {
                    "status": "warn",
                    "action": "latest_query_failed",
                    "reason": "latest version query failed",
                    "update_available": False,
                }
            )
            add_check(
                report,
                "codex_upgrade",
                "warn",
                {
                    "skipped": True,
                    "reason": "latest version query failed",
                    "current": current_version,
                    "latest": latest_version,
                    "latest_result": latest,
                },
            )
            return event
        if only_if_update and (not current_version or not latest_version):
            event.update(
                {
                    "status": "warn",
                    "action": "version_not_comparable",
                    "reason": "version is not comparable",
                    "update_available": False,
                }
            )
            add_check(
                report,
                "codex_upgrade",
                "warn",
                {
                    "skipped": True,
                    "reason": "version is not comparable",
                    "current": current_version,
                    "latest": latest_version,
                    "latest_result": latest,
                },
            )
            return event
        update_available = bool(latest["ok"] and current_version and latest_version and current_version != latest_version)
        event["update_available"] = update_available
        if only_if_update and not update_available:
            event.update(
                {
                    "status": "ok",
                    "action": "already_latest",
                    "reason": "already latest",
                }
            )
            add_check(
                report,
                "codex_upgrade",
                "ok",
                {
                    "skipped": True,
                    "reason": "already latest",
                    "current": current_version,
                    "latest": latest_version,
                    "latest_result": latest,
                },
            )
            return event

    upgrade_command = resolve_upgrade_command(before)
    if not upgrade_command:
        event.update(
            {
                "status": "fail",
                "action": "upgrade_command_unavailable",
                "reason": "npm upgrade command is not available",
                "update_available": update_available,
            }
        )
        add_check(
            report,
            "codex_upgrade",
            "fail",
            {"skipped": True, "reason": "npm upgrade command is not available", "before": before},
        )
        return event
    if dry_run:
        event.update(
            {
                "status": "warn",
                "action": "dry_run",
                "reason": "dry run",
                "command": command_label(upgrade_command),
                "update_available": update_available,
            }
        )
        add_check(
            report,
            "codex_upgrade",
            "warn",
            {
                "dry_run": True,
                "command": command_label(upgrade_command),
                "only_if_update": only_if_update,
                "update_available": update_available,
                "latest_result": latest,
            },
        )
        return event
    upgrade = run_command(upgrade_command, timeout=600)
    after = get_codex_version()
    event.update(
        {
            "status": "ok" if upgrade["ok"] else "fail",
            "action": "upgraded" if upgrade["ok"] else "upgrade_failed",
            "reason": "" if upgrade["ok"] else "upgrade command failed",
            "command": command_label(upgrade_command),
            "update_available": update_available,
            "upgrade_result": compact_result(upgrade),
            "after": compact_result(after),
            "after_version": extract_semver(after.get("stdout", "")),
        }
    )
    add_check(
        report,
        "codex_upgrade",
        "ok" if upgrade["ok"] else "fail",
        {
            "command": command_label(upgrade_command),
            "before": before,
            "latest_result": latest,
            "upgrade": upgrade,
            "after": after,
            "only_if_update": only_if_update,
        },
    )
    return event


def build_report(strict_balance: bool) -> dict[str, Any]:
    report: dict[str, Any] = {
        "checked_at": now_iso(),
        "status": "ok",
        "app_base_url": APP_BASE_URL,
        "compose_dir": str(COMPOSE_DIR),
        "upgrade_history_file": str(UPGRADE_HISTORY_FILE),
        "last_upgrade_file": str(LAST_UPGRADE_FILE),
        "checks": [],
    }
    codex = check_codex(report)
    report["codex_update_available"] = codex["update_available"]
    check_docker(report)
    check_http(report, strict_balance)
    check_failover(report)
    check_disk(report)
    return report


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Check or upgrade the host-side Codex runtime.")
    action = parser.add_mutually_exclusive_group()
    action.add_argument("--check", action="store_true", help="run health checks")
    action.add_argument("--upgrade", action="store_true", help="run configured upgrade command, then health checks")
    action.add_argument("--auto-upgrade", action="store_true", help="upgrade Codex when latest differs, then run checks")
    parser.add_argument("--strict-balance", action="store_true", help="treat stale balance as a failure")
    parser.add_argument("--dry-run", action="store_true", help="show upgrade command without running it")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if not args.check and not args.upgrade and not args.auto_upgrade:
        args.check = True

    report = build_report(args.strict_balance)
    upgrade_event: dict[str, Any] | None = None
    if args.upgrade or args.auto_upgrade:
        upgrade_event = perform_upgrade(report, args.dry_run, only_if_update=args.auto_upgrade)
        if not args.dry_run:
            upgrade_checks = [item for item in report["checks"] if item["name"] == "codex_upgrade"]
            post_report = build_report(args.strict_balance)
            post_report["pre_upgrade_checks"] = [
                item for item in report["checks"] if item["name"] != "codex_upgrade"
            ]
            post_report["checks"].extend(upgrade_checks)
            if any(item["status"] == "fail" for item in upgrade_checks):
                post_report["status"] = "fail"
            report = post_report
            report["post_upgrade"] = True
        if upgrade_event is not None:
            upgrade_event["finished_at"] = now_iso()
            upgrade_event["health_status_after"] = report["status"]
            upgrade_event["check_statuses_after"] = summarize_checks(report)
            upgrade_event["non_ok_checks_after"] = non_ok_checks(report)
            write_upgrade_event(upgrade_event)
            report["upgrade_event"] = upgrade_event

    write_report(report)
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 1 if report["status"] == "fail" else 0


if __name__ == "__main__":
    raise SystemExit(main())
