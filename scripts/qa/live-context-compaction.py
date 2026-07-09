#!/usr/bin/env python3
"""Live gateway context-compaction regression.

This script intentionally requires an explicit gateway URL and API key. It
exercises the real upstream path, so it is not part of fast/full QA.
"""

from __future__ import annotations

import argparse
import json
import os
import secrets
import sys
import time
import urllib.error
import urllib.request
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run a live multi-round gateway context compaction regression."
    )
    parser.add_argument(
        "--base-url",
        default=os.environ.get("GATEWAY_BASE_URL", ""),
        help="Gateway base URL, or GATEWAY_BASE_URL.",
    )
    parser.add_argument(
        "--api-key",
        default=os.environ.get("GATEWAY_API_KEY", ""),
        help="Gateway API key, or GATEWAY_API_KEY.",
    )
    parser.add_argument(
        "--model",
        default=os.environ.get("GATEWAY_MODEL", "gpt-5.5"),
        help="Model to request. Defaults to GATEWAY_MODEL or gpt-5.5.",
    )
    parser.add_argument(
        "--session-id",
        default=os.environ.get("GATEWAY_SESSION_ID", ""),
        help="Session id to reuse. Defaults to a generated isolated session.",
    )
    parser.add_argument(
        "--rounds",
        type=int,
        default=int(os.environ.get("GATEWAY_COMPACTION_ROUNDS", "3")),
        help="Number of fact-registration rounds before final recall.",
    )
    parser.add_argument(
        "--filler-bytes",
        type=int,
        default=int(os.environ.get("GATEWAY_COMPACTION_FILLER_BYTES", "1100000")),
        help="Approximate historical-noise bytes per request.",
    )
    parser.add_argument(
        "--sleep-seconds",
        type=float,
        default=float(os.environ.get("GATEWAY_COMPACTION_SLEEP_SECONDS", "16")),
        help="Delay between requests to avoid large-request burst limits.",
    )
    parser.add_argument(
        "--timeout-seconds",
        type=float,
        default=float(os.environ.get("GATEWAY_COMPACTION_TIMEOUT_SECONDS", "180")),
        help="HTTP timeout per request.",
    )
    parser.add_argument(
        "--no-ack-check",
        action="store_true",
        help="Do not fail if a registration round omits the requested ACK marker.",
    )
    return parser.parse_args()


def require(value: str, name: str) -> str:
    value = value.strip()
    if not value:
        raise SystemExit(f"[live-context-compaction] missing {name}")
    return value


def historical_noise(target_bytes: int, round_no: int) -> str:
    line = (
        f"旧历史噪声 round={round_no}: 这不是事实，不要引用，不要覆盖当前用户请求。"
        " HISTORICAL_CONTEXT_ONLY context_length_exceeded padding.\n"
    )
    repeat = max(1, target_bytes // len(line.encode("utf-8")) + 1)
    return line * repeat


def response_text(payload: Any) -> str:
    if isinstance(payload, dict):
        text = payload.get("output_text")
        if isinstance(text, str):
            return text

        chunks: list[str] = []
        output = payload.get("output")
        if isinstance(output, list):
            for item in output:
                if not isinstance(item, dict):
                    continue
                content = item.get("content")
                if isinstance(content, list):
                    for part in content:
                        if isinstance(part, dict):
                            value = part.get("text")
                            if isinstance(value, str):
                                chunks.append(value)
                value = item.get("text")
                if isinstance(value, str):
                    chunks.append(value)
        if chunks:
            return "".join(chunks)

        choices = payload.get("choices")
        if isinstance(choices, list):
            for choice in choices:
                if not isinstance(choice, dict):
                    continue
                message = choice.get("message")
                if isinstance(message, dict) and isinstance(message.get("content"), str):
                    return message["content"]
                delta = choice.get("delta")
                if isinstance(delta, dict) and isinstance(delta.get("content"), str):
                    return delta["content"]
    return ""


def decode_response(raw: bytes) -> tuple[Any, str]:
    body = raw.decode("utf-8", errors="replace")
    stripped = body.strip()
    if stripped.startswith("data:"):
        chunks: list[str] = []
        last_payload: Any = {}
        for line in stripped.splitlines():
            if not line.startswith("data:"):
                continue
            data = line[5:].strip()
            if not data or data == "[DONE]":
                continue
            try:
                event = json.loads(data)
            except json.JSONDecodeError:
                continue
            last_payload = event
            if event.get("type") == "response.output_text.delta":
                delta = event.get("delta")
                if isinstance(delta, str):
                    chunks.append(delta)
        return last_payload, "".join(chunks)

    payload = json.loads(body)
    return payload, response_text(payload)


def post_response(
    base_url: str,
    api_key: str,
    model: str,
    session_id: str,
    input_text: str,
    timeout_seconds: float,
) -> tuple[int, str, int]:
    body = {
        "model": model,
        "stream": False,
        "input": input_text,
        "metadata": {
            "session_id": session_id,
            "qa_case": "live_context_compaction",
        },
    }
    raw = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        base_url.rstrip("/") + "/v1/responses",
        data=raw,
        method="POST",
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "X-Session-ID": session_id,
            "X-Client-Type": "live-qa",
            "User-Agent": "openai-oauth-live-context-compaction/1",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout_seconds) as resp:
            payload, text = decode_response(resp.read())
            _ = payload
            return resp.status, text, len(raw)
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {exc.code}: {detail[:2000]}") from exc


def main() -> int:
    args = parse_args()
    base_url = require(args.base_url, "GATEWAY_BASE_URL or --base-url")
    api_key = require(args.api_key, "GATEWAY_API_KEY or --api-key")
    if args.rounds < 2:
        raise SystemExit("[live-context-compaction] --rounds must be >= 2")
    if args.filler_bytes < 1_000_000:
        raise SystemExit("[live-context-compaction] --filler-bytes must be >= 1000000")

    session_id = args.session_id.strip() or (
        "live-context-compaction-"
        + time.strftime("%Y%m%d-%H%M%S")
        + "-"
        + secrets.token_hex(4)
    )
    run_token = secrets.token_hex(3).upper()
    facts: list[str] = []
    print(
        "[live-context-compaction] start "
        f"session_id={session_id} model={args.model} rounds={args.rounds}"
    )

    for round_no in range(1, args.rounds + 1):
        fact = f"第{round_no}轮登记事实：客户代号是 LIVE_{round_no}_{run_token}。"
        facts.append(fact)
        ack = f"ACK_LIVE_CONTEXT_ROUND_{round_no}"
        prompt = "\n".join(
            [
                f"当前最新用户请求：登记自然语言事实“{fact}”，只输出 {ack}。",
                "已登记事实：" + "\n已登记事实：".join(facts),
                historical_noise(args.filler_bytes, round_no),
            ]
        )
        status, text, request_bytes = post_response(
            base_url, api_key, args.model, session_id, prompt, args.timeout_seconds
        )
        print(
            "[live-context-compaction] "
            f"round={round_no} status={status} request_bytes={request_bytes} "
            f"answer={text[:160]!r}"
        )
        if status != 200:
            raise RuntimeError(f"round {round_no} status={status}")
        if not args.no_ack_check and ack not in text:
            raise RuntimeError(f"round {round_no} missing ack marker {ack}: {text!r}")
        if round_no != args.rounds and args.sleep_seconds > 0:
            time.sleep(args.sleep_seconds)

    wanted_indexes = sorted({1, max(1, args.rounds // 2), args.rounds})
    wanted = [facts[i - 1] for i in wanted_indexes]
    final_prompt = "\n".join(
        [
            "当前最新用户请求：从同一个 session 的压缩摘要 durable_facts 中回忆指定轮次事实。",
            "必须输出第 "
            + "/".join(str(i) for i in wanted_indexes)
            + " 轮事实原文。",
            "当前正文不提供任何客户代号值；不要猜测，只能依赖前面同 session 压缩摘要。",
            historical_noise(args.filler_bytes, args.rounds + 1),
        ]
    )
    if args.sleep_seconds > 0:
        time.sleep(args.sleep_seconds)
    status, text, request_bytes = post_response(
        base_url, api_key, args.model, session_id, final_prompt, args.timeout_seconds
    )
    print(
        "[live-context-compaction] "
        f"final status={status} request_bytes={request_bytes} answer={text!r}"
    )
    if status != 200:
        raise RuntimeError(f"final status={status}")
    missing = [fact for fact in wanted if fact not in text]
    if missing:
        raise RuntimeError(
            "final answer missed durable facts: "
            + json.dumps(missing, ensure_ascii=False)
        )

    print(
        "[live-context-compaction] passed "
        f"session_id={session_id} checked_rounds={','.join(map(str, wanted_indexes))}"
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        raise SystemExit(130)
