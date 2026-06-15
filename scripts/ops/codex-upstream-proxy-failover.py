#!/usr/bin/env python3
"""Switch mihomo node when Codex backend streaming fails."""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import time
import urllib.parse
import urllib.request
from pathlib import Path


CONTAINER = os.getenv("CODEX_FAILOVER_CONTAINER", "openai-oauth-api-service-server")
MIHOMO_CONFIG = Path(os.getenv("MIHOMO_CONFIG", "/etc/mihomo/config.yaml"))
MIHOMO_API = os.getenv("MIHOMO_API", "http://127.0.0.1:9090")
SELECTOR = os.getenv("MIHOMO_SELECTOR", "节点选择")
INHERIT_GROUPS = [
    item.strip()
    for item in os.getenv("MIHOMO_INHERIT_GROUPS", "ChatGPT").split(",")
    if item.strip()
]
FAILOVER_NODES = [
    item.strip()
    for item in os.getenv(
        "CODEX_FAILOVER_NODES",
        "日本JP-HY2,日本-优化3,日本-优化2,日本-优化",
    ).split(",")
    if item.strip()
]
COOLDOWN_SECONDS = int(os.getenv("CODEX_FAILOVER_COOLDOWN_SECONDS", "180"))
STATE_FILE = Path(
    os.getenv("CODEX_FAILOVER_STATE_FILE", "/var/lib/codex-upstream-proxy-failover/state.json")
)

ERROR_RE = re.compile(
    r"backend-api/codex/responses.*"
    r"(EOF|INTERNAL_ERROR|stream disconnected|error sending request|connection reset)"
    r"|stream disconnected before completion.*backend-api/codex/responses",
    re.IGNORECASE,
)


def log(message: str) -> None:
    print(message, flush=True)


def read_secret() -> str:
    if not MIHOMO_CONFIG.exists():
        return ""
    for line in MIHOMO_CONFIG.read_text(errors="ignore").splitlines():
        stripped = line.strip()
        if stripped.startswith("secret:"):
            return stripped.split(":", 1)[1].strip().strip('"').strip("'")
    return ""


def mihomo_request(method: str, path: str, body: dict | None = None) -> dict | None:
    headers = {"Content-Type": "application/json"}
    secret = read_secret()
    if secret:
        headers["Authorization"] = "Bearer " + secret
    data = None if body is None else json.dumps(body, ensure_ascii=False).encode()
    req = urllib.request.Request(MIHOMO_API + path, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=5) as response:
        raw = response.read()
        return json.loads(raw) if raw else None


def load_state() -> dict:
    try:
        return json.loads(STATE_FILE.read_text())
    except Exception:
        return {}


def save_state(state: dict) -> None:
    STATE_FILE.parent.mkdir(parents=True, exist_ok=True)
    tmp = STATE_FILE.with_suffix(".tmp")
    tmp.write_text(json.dumps(state, ensure_ascii=False, indent=2) + "\n")
    tmp.replace(STATE_FILE)


def set_selector(group: str, target: str, proxies: dict) -> bool:
    current = proxies.get(group)
    if not current:
        log(f"skip group={group}: group missing")
        return False
    if target not in (current.get("all") or []):
        log(f"skip group={group}: target={target} unavailable")
        return False
    if current.get("now") == target:
        return True
    encoded = urllib.parse.quote(group, safe="")
    mihomo_request("PUT", f"/proxies/{encoded}", {"name": target})
    log(f"set group={group} target={target}")
    return True


def check_config() -> int:
    data = mihomo_request("GET", "/proxies") or {}
    proxies = data.get("proxies") or {}
    errors: list[str] = []

    selector = proxies.get(SELECTOR)
    if not selector:
        errors.append(f"selector missing: {SELECTOR}")
    else:
        available = selector.get("all") or []
        for node in FAILOVER_NODES:
            if node not in available:
                errors.append(f"failover node unavailable in {SELECTOR}: {node}")

    for group in INHERIT_GROUPS:
        item = proxies.get(group)
        if not item:
            errors.append(f"inherit group missing: {group}")
            continue
        if SELECTOR not in (item.get("all") or []):
            errors.append(f"inherit group cannot select {SELECTOR}: {group}")

    result = {
        "selector": SELECTOR,
        "selector_now": selector.get("now") if selector else None,
        "inherit_groups": {
            group: (proxies.get(group) or {}).get("now") for group in INHERIT_GROUPS
        },
        "failover_nodes": FAILOVER_NODES,
        "container": CONTAINER,
        "mihomo_api": MIHOMO_API,
        "errors": errors,
    }
    print(json.dumps(result, ensure_ascii=False, indent=2), flush=True)
    return 1 if errors else 0


def switch_next(reason: str) -> None:
    state = load_state()
    now = time.time()
    last_switch_at = float(state.get("last_switch_at") or 0)
    if now - last_switch_at < COOLDOWN_SECONDS:
        log(f"cooldown active; skip switch reason={reason!r}")
        return

    data = mihomo_request("GET", "/proxies") or {}
    proxies = data.get("proxies") or {}
    selector = proxies.get(SELECTOR)
    if not selector:
        log(f"selector missing selector={SELECTOR}")
        return

    current = selector.get("now")
    if current in FAILOVER_NODES:
        next_node = FAILOVER_NODES[(FAILOVER_NODES.index(current) + 1) % len(FAILOVER_NODES)]
    else:
        next_node = FAILOVER_NODES[0]

    if next_node not in (selector.get("all") or []):
        log(f"next node unavailable selector={SELECTOR} current={current} next={next_node}")
        return

    set_selector(SELECTOR, next_node, proxies)
    proxies = (mihomo_request("GET", "/proxies") or {}).get("proxies") or {}
    for group in INHERIT_GROUPS:
        set_selector(group, SELECTOR, proxies)

    state.update(
        {
            "last_switch_at": now,
            "last_reason": reason,
            "last_from": current,
            "last_to": next_node,
        }
    )
    save_state(state)
    log(f"switched selector={SELECTOR} from={current} to={next_node} reason={reason!r}")


def follow_logs() -> None:
    while True:
        cmd = ["docker", "logs", "--follow", "--since", "0s", CONTAINER]
        log("starting log follower: " + " ".join(cmd))
        proc = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            errors="replace",
        )
        assert proc.stdout is not None
        for line in proc.stdout:
            line = line.rstrip("\n")
            if ERROR_RE.search(line):
                switch_next(line[:500])
        rc = proc.wait()
        log(f"log follower exited rc={rc}; retrying")
        time.sleep(5)


if __name__ == "__main__":
    if not FAILOVER_NODES:
        sys.exit("CODEX_FAILOVER_NODES is empty")
    if len(sys.argv) == 2 and sys.argv[1] == "--check":
        raise SystemExit(check_config())
    if len(sys.argv) >= 2 and sys.argv[1] == "--switch-next":
        switch_next(" ".join(sys.argv[2:]) or "manual switch")
        raise SystemExit(0)
    if len(sys.argv) != 1:
        sys.exit("usage: codex-upstream-proxy-failover.py [--check|--switch-next REASON]")
    follow_logs()
