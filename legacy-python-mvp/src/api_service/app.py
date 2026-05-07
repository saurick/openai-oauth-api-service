from __future__ import annotations

from contextlib import asynccontextmanager
from typing import Annotated

import httpx
from fastapi import Depends, FastAPI, HTTPException, Request, status
from pydantic import BaseModel, Field

from .auth import authenticate_admin, authenticate_downstream, generate_token, hash_token, token_prefix
from .config import Settings
from .db import ApiKeyRecord, Database
from .proxy import proxy_request


class CreateKeyRequest(BaseModel):
    name: str = Field(min_length=1, max_length=120)
    rpm_limit: int | None = Field(default=None, ge=1)
    daily_token_limit: int | None = Field(default=None, ge=1)


def create_upstream_client(settings: Settings, transport: httpx.AsyncBaseTransport | None = None) -> httpx.AsyncClient:
    kwargs = {
        "timeout": httpx.Timeout(settings.upstream_timeout_seconds),
        "trust_env": False,
        "transport": transport,
    }
    if settings.upstream_proxy_url and transport is None:
        kwargs["proxy"] = settings.upstream_proxy_url
    return httpx.AsyncClient(**kwargs)


def create_app(
    settings: Settings | None = None,
    db: Database | None = None,
    upstream_transport: httpx.AsyncBaseTransport | None = None,
) -> FastAPI:
    runtime_settings = settings or Settings.from_env()
    owns_db = db is None

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        runtime_settings.validate_runtime()
        if not hasattr(app.state, "db"):
            app.state.db = Database(runtime_settings.database_path)
        if not hasattr(app.state, "upstream_client"):
            app.state.upstream_client = create_upstream_client(runtime_settings, upstream_transport)
        try:
            yield
        finally:
            await app.state.upstream_client.aclose()
            if owns_db:
                app.state.db.close()

    app = FastAPI(
        title="OpenAI API Gateway",
        version="0.1.0",
        lifespan=lifespan,
    )
    app.state.settings = runtime_settings
    if db is not None:
        app.state.db = db
    if upstream_transport is not None:
        app.state.upstream_client = create_upstream_client(runtime_settings, upstream_transport)

    @app.get("/healthz")
    async def healthz() -> dict[str, str]:
        return {"status": "ok"}

    @app.post("/admin/keys", dependencies=[Depends(authenticate_admin)])
    async def create_key(payload: CreateKeyRequest, request: Request) -> dict[str, object]:
        token = generate_token()
        record = request.app.state.db.create_key(
            name=payload.name,
            key_hash=hash_token(token),
            key_prefix=token_prefix(token),
            rpm_limit=payload.rpm_limit,
            daily_token_limit=payload.daily_token_limit,
        )
        return {
            "id": record.id,
            "name": record.name,
            "key": token,
            "key_prefix": record.key_prefix,
            "rpm_limit": record.rpm_limit,
            "daily_token_limit": record.daily_token_limit,
            "created_at": record.created_at,
        }

    @app.get("/admin/keys", dependencies=[Depends(authenticate_admin)])
    async def list_keys(request: Request) -> dict[str, object]:
        return {"data": request.app.state.db.list_keys()}

    @app.post("/admin/keys/{key_id}/revoke", dependencies=[Depends(authenticate_admin)])
    async def revoke_key(key_id: str, request: Request) -> dict[str, object]:
        revoked = request.app.state.db.revoke_key(key_id)
        if not revoked:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail={"error": "active_key_not_found"},
            )
        return {"id": key_id, "revoked": True}

    @app.get("/admin/usage/summary", dependencies=[Depends(authenticate_admin)])
    async def usage_summary(request: Request, hours: int = 24) -> dict[str, object]:
        bounded_hours = max(1, min(hours, 24 * 90))
        return {"hours": bounded_hours, "data": request.app.state.db.usage_summary(hours=bounded_hours)}

    @app.get("/admin/usage/recent", dependencies=[Depends(authenticate_admin)])
    async def recent_usage(request: Request, limit: int = 50) -> dict[str, object]:
        return {"data": request.app.state.db.recent_logs(limit=limit)}

    @app.api_route(
        "/v1/{path:path}",
        methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
    )
    async def openai_proxy(
        path: str,
        request: Request,
        key: Annotated[ApiKeyRecord, Depends(authenticate_downstream)],
    ):
        return await proxy_request(request, path, key)

    return app


app = create_app()
