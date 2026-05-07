from __future__ import annotations

import json

import httpx
import pytest
from httpx import ASGITransport

from api_service.app import create_app
from api_service.auth import generate_token, hash_token, token_prefix
from api_service.config import Settings
from api_service.db import Database


class AsyncIteratorStream(httpx.AsyncByteStream):
    def __init__(self, iterator):
        self.iterator = iterator

    async def __aiter__(self):
        async for chunk in self.iterator:
            yield chunk


async def make_client(tmp_path, handler):
    db = Database(tmp_path / "api_service.sqlite3")
    downstream_token = generate_token()
    key = db.create_key(
        name="alice",
        key_hash=hash_token(downstream_token),
        key_prefix=token_prefix(downstream_token),
        rpm_limit=None,
        daily_token_limit=None,
    )
    settings = Settings(
        openai_api_key="upstream-key",
        admin_token="admin-secret",
        database_path=tmp_path / "api_service.sqlite3",
    )
    app = create_app(
        settings=settings,
        db=db,
        upstream_transport=httpx.MockTransport(handler),
    )
    client = httpx.AsyncClient(
        transport=ASGITransport(app=app),
        base_url="http://testserver",
    )
    return client, db, downstream_token, key


@pytest.mark.asyncio
async def test_proxy_forwards_with_upstream_key_and_records_usage(tmp_path):
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.url == "https://api.openai.com/v1/responses"
        assert request.headers["authorization"] == "Bearer upstream-key"
        payload = json.loads(request.content)
        assert payload["model"] == "gpt-5.4"
        return httpx.Response(
            200,
            json={
                "id": "resp_test",
                "usage": {
                    "input_tokens": 3,
                    "output_tokens": 5,
                    "total_tokens": 8,
                },
            },
        )

    client, db, downstream_token, key = await make_client(tmp_path, handler)
    async with client:
        response = await client.post(
            "/v1/responses",
            headers={"Authorization": f"Bearer {downstream_token}"},
            json={"model": "gpt-5.4", "input": "hello"},
        )

    assert response.status_code == 200
    summary = db.usage_summary(hours=1)
    assert summary[0]["api_key_id"] == key.id
    assert summary[0]["requests"] == 1
    assert summary[0]["total_tokens"] == 8
    db.close()


@pytest.mark.asyncio
async def test_missing_downstream_key_is_rejected(tmp_path):
    async def handler(request: httpx.Request) -> httpx.Response:
        raise AssertionError("upstream should not be called")

    client, db, _, _ = await make_client(tmp_path, handler)
    async with client:
        response = await client.post("/v1/responses", json={"model": "gpt-5.4", "input": "hello"})

    assert response.status_code == 401
    db.close()


@pytest.mark.asyncio
async def test_admin_create_key_and_usage_summary(tmp_path):
    async def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"usage": {"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3}})

    client, db, _, _ = await make_client(tmp_path, handler)
    async with client:
        create_response = await client.post(
            "/admin/keys",
            headers={"X-Admin-Token": "admin-secret"},
            json={"name": "bob", "rpm_limit": 10, "daily_token_limit": 1000},
        )
        assert create_response.status_code == 200
        new_key = create_response.json()["key"]

        proxy_response = await client.post(
            "/v1/chat/completions",
            headers={"Authorization": f"Bearer {new_key}"},
            json={"model": "gpt-5.4", "messages": [{"role": "user", "content": "hi"}]},
        )
        assert proxy_response.status_code == 200

        summary_response = await client.get(
            "/admin/usage/summary",
            headers={"Authorization": "Bearer admin-secret"},
        )

    assert summary_response.status_code == 200
    rows = summary_response.json()["data"]
    assert any(row["name"] == "bob" and row["total_tokens"] == 3 for row in rows)
    db.close()


@pytest.mark.asyncio
async def test_streaming_response_usage_is_recorded(tmp_path):
    async def stream_body():
        yield b'event: response.completed\n'
        yield (
            b'data: {"type":"response.completed","response":{"usage":'
            b'{"input_tokens":4,"output_tokens":6,"total_tokens":10}}}\n\n'
        )

    async def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            headers={"content-type": "text/event-stream"},
            stream=AsyncIteratorStream(stream_body()),
        )

    client, db, downstream_token, _ = await make_client(tmp_path, handler)
    async with client:
        response = await client.post(
            "/v1/responses",
            headers={"Authorization": f"Bearer {downstream_token}"},
            json={"model": "gpt-5.4", "input": "hello", "stream": True},
        )
        assert response.status_code == 200
        assert "response.completed" in response.text

    assert db.usage_summary(hours=1)[0]["total_tokens"] == 10
    db.close()
