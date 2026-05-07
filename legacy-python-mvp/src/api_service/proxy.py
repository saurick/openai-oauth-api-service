from __future__ import annotations

import json
import time
from collections.abc import AsyncIterator
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from typing import Any

import httpx
from fastapi import HTTPException, Request, Response, status
from starlette.responses import StreamingResponse

from .db import ApiKeyRecord, Database, RequestLog


HOP_BY_HOP_HEADERS = {
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailer",
    "transfer-encoding",
    "upgrade",
}

REQUEST_HEADERS_TO_DROP = HOP_BY_HOP_HEADERS | {
    "authorization",
    "content-length",
    "host",
    "x-api-key",
}

RESPONSE_HEADERS_TO_DROP = HOP_BY_HOP_HEADERS | {
    "content-encoding",
    "content-length",
}


@dataclass
class Usage:
    input_tokens: int | None = None
    output_tokens: int | None = None
    total_tokens: int | None = None


@dataclass
class StreamMetrics:
    response_bytes: int = 0
    usage: Usage | None = None
    error: str | None = None


def build_upstream_url(base_url: str, path: str, query: str) -> str:
    target = f"{base_url.rstrip('/')}/{path.lstrip('/')}"
    return f"{target}?{query}" if query else target


def forward_headers(request: Request, upstream_api_key: str) -> dict[str, str]:
    headers = {
        key: value
        for key, value in request.headers.items()
        if key.lower() not in REQUEST_HEADERS_TO_DROP
    }
    headers["authorization"] = f"Bearer {upstream_api_key}"
    return headers


def response_headers(headers: httpx.Headers) -> dict[str, str]:
    return {
        key: value
        for key, value in headers.items()
        if key.lower() not in RESPONSE_HEADERS_TO_DROP
    }


def extract_model(body: bytes, content_type: str | None) -> str | None:
    if not body or not content_type or "application/json" not in content_type:
        return None
    try:
        payload = json.loads(body)
    except json.JSONDecodeError:
        return None
    model = payload.get("model")
    return model if isinstance(model, str) else None


def wants_stream(body: bytes, content_type: str | None) -> bool:
    if not body or not content_type or "application/json" not in content_type:
        return False
    try:
        payload = json.loads(body)
    except json.JSONDecodeError:
        return False
    return payload.get("stream") is True


def usage_from_payload(payload: Any) -> Usage | None:
    if not isinstance(payload, dict):
        return None

    usage = payload.get("usage")
    if not isinstance(usage, dict):
        response = payload.get("response")
        if isinstance(response, dict):
            usage = response.get("usage")

    if not isinstance(usage, dict):
        return None

    input_tokens = usage.get("input_tokens", usage.get("prompt_tokens"))
    output_tokens = usage.get("output_tokens", usage.get("completion_tokens"))
    total_tokens = usage.get("total_tokens")
    if total_tokens is None and (input_tokens is not None or output_tokens is not None):
        total_tokens = int(input_tokens or 0) + int(output_tokens or 0)

    return Usage(
        input_tokens=_as_int(input_tokens),
        output_tokens=_as_int(output_tokens),
        total_tokens=_as_int(total_tokens),
    )


def _as_int(value: Any) -> int | None:
    if value is None:
        return None
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def parse_json_usage(body: bytes, content_type: str | None) -> Usage | None:
    if not body or not content_type or "application/json" not in content_type:
        return None
    try:
        payload = json.loads(body)
    except json.JSONDecodeError:
        return None
    return usage_from_payload(payload)


def update_usage_from_sse_block(block: str, current: Usage | None) -> Usage | None:
    data_lines = []
    for raw_line in block.splitlines():
        line = raw_line.strip()
        if line.startswith("data:"):
            data_lines.append(line.removeprefix("data:").strip())
    if not data_lines:
        return current

    data = "\n".join(data_lines)
    if data == "[DONE]":
        return current
    try:
        payload = json.loads(data)
    except json.JSONDecodeError:
        return current

    return usage_from_payload(payload) or current


def check_limits(db: Database, key: ApiKeyRecord) -> None:
    if key.rpm_limit is not None:
        since = datetime.now(UTC) - timedelta(seconds=60)
        if db.request_count_since(key.id, since) >= key.rpm_limit:
            raise HTTPException(
                status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                detail={"error": "rpm_limit_exceeded"},
            )

    if key.daily_token_limit is not None:
        now = datetime.now(UTC)
        start_of_day = datetime(now.year, now.month, now.day, tzinfo=UTC)
        if db.token_sum_since(key.id, start_of_day) >= key.daily_token_limit:
            raise HTTPException(
                status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                detail={"error": "daily_token_limit_exceeded"},
            )


async def proxy_request(request: Request, path: str, key: ApiKeyRecord) -> Response:
    settings = request.app.state.settings
    db: Database = request.app.state.db
    client: httpx.AsyncClient = request.app.state.upstream_client

    check_limits(db, key)

    body = await request.body()
    content_type = request.headers.get("content-type")
    model = extract_model(body, content_type)
    target_url = build_upstream_url(settings.openai_base_url, path, request.url.query)
    headers = forward_headers(request, settings.openai_api_key)
    started = time.perf_counter()

    if wants_stream(body, content_type):
        return await _proxy_streaming(
            client=client,
            db=db,
            key=key,
            request=request,
            path=path,
            model=model,
            target_url=target_url,
            headers=headers,
            body=body,
            started=started,
        )

    try:
        upstream = await client.request(
            request.method,
            target_url,
            content=body,
            headers=headers,
        )
        response_body = upstream.content
        usage = parse_json_usage(response_body, upstream.headers.get("content-type"))
        duration_ms = int((time.perf_counter() - started) * 1000)
        db.insert_request_log(
            RequestLog(
                api_key_id=key.id,
                method=request.method,
                path=f"/v1/{path}",
                model=model,
                upstream_status=upstream.status_code,
                request_bytes=len(body),
                response_bytes=len(response_body),
                input_tokens=usage.input_tokens if usage else None,
                output_tokens=usage.output_tokens if usage else None,
                total_tokens=usage.total_tokens if usage else None,
                duration_ms=duration_ms,
            )
        )
        return Response(
            content=response_body,
            status_code=upstream.status_code,
            headers=response_headers(upstream.headers),
            media_type=upstream.headers.get("content-type"),
        )
    except httpx.HTTPError as exc:
        duration_ms = int((time.perf_counter() - started) * 1000)
        db.insert_request_log(
            RequestLog(
                api_key_id=key.id,
                method=request.method,
                path=f"/v1/{path}",
                model=model,
                request_bytes=len(body),
                duration_ms=duration_ms,
                error=exc.__class__.__name__,
            )
        )
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail={"error": "upstream_request_failed"},
        ) from exc


async def _proxy_streaming(
    *,
    client: httpx.AsyncClient,
    db: Database,
    key: ApiKeyRecord,
    request: Request,
    path: str,
    model: str | None,
    target_url: str,
    headers: dict[str, str],
    body: bytes,
    started: float,
) -> StreamingResponse:
    upstream_request = client.build_request(
        request.method,
        target_url,
        content=body,
        headers=headers,
    )
    upstream = await client.send(upstream_request, stream=True)
    metrics = StreamMetrics()

    async def body_iterator() -> AsyncIterator[bytes]:
        buffer = ""
        try:
            async for chunk in upstream.aiter_bytes():
                metrics.response_bytes += len(chunk)
                text = chunk.decode("utf-8", errors="ignore")
                buffer += text
                while "\n\n" in buffer:
                    block, buffer = buffer.split("\n\n", 1)
                    metrics.usage = update_usage_from_sse_block(block, metrics.usage)
                yield chunk
            if buffer:
                metrics.usage = update_usage_from_sse_block(buffer, metrics.usage)
        except httpx.HTTPError as exc:
            metrics.error = exc.__class__.__name__
            raise
        finally:
            await upstream.aclose()
            duration_ms = int((time.perf_counter() - started) * 1000)
            usage = metrics.usage
            db.insert_request_log(
                RequestLog(
                    api_key_id=key.id,
                    method=request.method,
                    path=f"/v1/{path}",
                    model=model,
                    upstream_status=upstream.status_code,
                    request_bytes=len(body),
                    response_bytes=metrics.response_bytes,
                    input_tokens=usage.input_tokens if usage else None,
                    output_tokens=usage.output_tokens if usage else None,
                    total_tokens=usage.total_tokens if usage else None,
                    duration_ms=duration_ms,
                    error=metrics.error,
                )
            )

    return StreamingResponse(
        body_iterator(),
        status_code=upstream.status_code,
        headers=response_headers(upstream.headers),
        media_type=upstream.headers.get("content-type"),
    )
