from __future__ import annotations

import hashlib
import hmac
import secrets

from fastapi import Header, HTTPException, Request, status

from .db import ApiKeyRecord, Database


TOKEN_PREFIX = "ogw_"


def hash_token(token: str) -> str:
    return hashlib.sha256(token.encode("utf-8")).hexdigest()


def generate_token() -> str:
    return f"{TOKEN_PREFIX}{secrets.token_urlsafe(32)}"


def token_prefix(token: str) -> str:
    return token[:12]


def extract_bearer_token(value: str | None) -> str | None:
    if not value:
        return None
    scheme, _, token = value.partition(" ")
    if scheme.lower() != "bearer" or not token:
        return None
    return token.strip()


def authenticate_downstream(request: Request) -> ApiKeyRecord:
    auth_header = request.headers.get("authorization")
    token = extract_bearer_token(auth_header) or request.headers.get("x-api-key")
    if not token:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={"error": "missing_api_key"},
        )

    db: Database = request.app.state.db
    record = db.find_key_by_hash(hash_token(token))
    if not record or not record.active:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={"error": "invalid_api_key"},
        )
    return record


def authenticate_admin(
    request: Request,
    authorization: str | None = Header(default=None),
    x_admin_token: str | None = Header(default=None),
) -> None:
    expected = request.app.state.settings.admin_token
    provided = extract_bearer_token(authorization) or x_admin_token
    if not expected or not provided or not hmac.compare_digest(provided, expected):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={"error": "invalid_admin_token"},
        )
