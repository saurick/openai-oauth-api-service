from __future__ import annotations

import sqlite3
import threading
import uuid
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any


def utc_now() -> datetime:
    return datetime.now(UTC)


def iso_now() -> str:
    return utc_now().isoformat()


@dataclass(frozen=True)
class ApiKeyRecord:
    id: str
    name: str
    key_hash: str
    key_prefix: str
    active: bool
    rpm_limit: int | None
    daily_token_limit: int | None
    created_at: str
    revoked_at: str | None


@dataclass(frozen=True)
class RequestLog:
    api_key_id: str | None
    method: str
    path: str
    model: str | None = None
    upstream_status: int | None = None
    request_bytes: int = 0
    response_bytes: int = 0
    input_tokens: int | None = None
    output_tokens: int | None = None
    total_tokens: int | None = None
    duration_ms: int | None = None
    error: str | None = None


class Database:
    def __init__(self, path: Path | str):
        self.path = Path(path)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._lock = threading.Lock()
        self._conn = sqlite3.connect(self.path, check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._conn.execute("PRAGMA journal_mode=WAL")
        self._conn.execute("PRAGMA foreign_keys=ON")
        self.init_schema()

    def close(self) -> None:
        with self._lock:
            self._conn.close()

    def init_schema(self) -> None:
        with self._lock:
            self._conn.executescript(
                """
                CREATE TABLE IF NOT EXISTS api_keys (
                    id TEXT PRIMARY KEY,
                    name TEXT NOT NULL,
                    key_hash TEXT NOT NULL UNIQUE,
                    key_prefix TEXT NOT NULL,
                    active INTEGER NOT NULL DEFAULT 1,
                    rpm_limit INTEGER,
                    daily_token_limit INTEGER,
                    created_at TEXT NOT NULL,
                    revoked_at TEXT
                );

                CREATE TABLE IF NOT EXISTS request_logs (
                    id TEXT PRIMARY KEY,
                    api_key_id TEXT REFERENCES api_keys(id),
                    method TEXT NOT NULL,
                    path TEXT NOT NULL,
                    model TEXT,
                    upstream_status INTEGER,
                    request_bytes INTEGER NOT NULL DEFAULT 0,
                    response_bytes INTEGER NOT NULL DEFAULT 0,
                    input_tokens INTEGER,
                    output_tokens INTEGER,
                    total_tokens INTEGER,
                    duration_ms INTEGER,
                    error TEXT,
                    created_at TEXT NOT NULL
                );

                CREATE INDEX IF NOT EXISTS idx_request_logs_key_created
                    ON request_logs(api_key_id, created_at);
                CREATE INDEX IF NOT EXISTS idx_request_logs_created
                    ON request_logs(created_at);
                """
            )
            self._conn.commit()

    def create_key(
        self,
        *,
        name: str,
        key_hash: str,
        key_prefix: str,
        rpm_limit: int | None,
        daily_token_limit: int | None,
    ) -> ApiKeyRecord:
        key_id = str(uuid.uuid4())
        created_at = iso_now()
        with self._lock:
            self._conn.execute(
                """
                INSERT INTO api_keys (
                    id, name, key_hash, key_prefix, active,
                    rpm_limit, daily_token_limit, created_at
                )
                VALUES (?, ?, ?, ?, 1, ?, ?, ?)
                """,
                (key_id, name, key_hash, key_prefix, rpm_limit, daily_token_limit, created_at),
            )
            self._conn.commit()
        return ApiKeyRecord(
            id=key_id,
            name=name,
            key_hash=key_hash,
            key_prefix=key_prefix,
            active=True,
            rpm_limit=rpm_limit,
            daily_token_limit=daily_token_limit,
            created_at=created_at,
            revoked_at=None,
        )

    def find_key_by_hash(self, key_hash: str) -> ApiKeyRecord | None:
        with self._lock:
            row = self._conn.execute(
                "SELECT * FROM api_keys WHERE key_hash = ?",
                (key_hash,),
            ).fetchone()
        return self._row_to_key(row) if row else None

    def list_keys(self) -> list[dict[str, Any]]:
        with self._lock:
            rows = self._conn.execute(
                """
                SELECT id, name, key_prefix, active, rpm_limit,
                       daily_token_limit, created_at, revoked_at
                FROM api_keys
                ORDER BY created_at DESC
                """
            ).fetchall()
        return [{**dict(row), "active": bool(row["active"])} for row in rows]

    def revoke_key(self, key_id: str) -> bool:
        with self._lock:
            cursor = self._conn.execute(
                """
                UPDATE api_keys
                SET active = 0, revoked_at = ?
                WHERE id = ? AND active = 1
                """,
                (iso_now(), key_id),
            )
            self._conn.commit()
            return cursor.rowcount > 0

    def insert_request_log(self, log: RequestLog) -> None:
        with self._lock:
            self._conn.execute(
                """
                INSERT INTO request_logs (
                    id, api_key_id, method, path, model, upstream_status,
                    request_bytes, response_bytes, input_tokens, output_tokens,
                    total_tokens, duration_ms, error, created_at
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    str(uuid.uuid4()),
                    log.api_key_id,
                    log.method,
                    log.path,
                    log.model,
                    log.upstream_status,
                    log.request_bytes,
                    log.response_bytes,
                    log.input_tokens,
                    log.output_tokens,
                    log.total_tokens,
                    log.duration_ms,
                    log.error,
                    iso_now(),
                ),
            )
            self._conn.commit()

    def request_count_since(self, key_id: str, since: datetime) -> int:
        with self._lock:
            row = self._conn.execute(
                """
                SELECT COUNT(*) AS count
                FROM request_logs
                WHERE api_key_id = ? AND created_at >= ?
                """,
                (key_id, since.isoformat()),
            ).fetchone()
        return int(row["count"])

    def token_sum_since(self, key_id: str, since: datetime) -> int:
        with self._lock:
            row = self._conn.execute(
                """
                SELECT COALESCE(SUM(total_tokens), 0) AS total
                FROM request_logs
                WHERE api_key_id = ? AND created_at >= ?
                """,
                (key_id, since.isoformat()),
            ).fetchone()
        return int(row["total"])

    def usage_summary(self, *, hours: int = 24) -> list[dict[str, Any]]:
        since = (utc_now() - timedelta(hours=hours)).isoformat()
        with self._lock:
            rows = self._conn.execute(
                """
                SELECT
                    k.id AS api_key_id,
                    k.name AS name,
                    k.key_prefix AS key_prefix,
                    COUNT(l.id) AS requests,
                    SUM(CASE WHEN l.upstream_status >= 200 AND l.upstream_status < 300 THEN 1 ELSE 0 END) AS ok_requests,
                    SUM(CASE WHEN l.upstream_status >= 400 OR l.error IS NOT NULL THEN 1 ELSE 0 END) AS failed_requests,
                    COALESCE(SUM(l.request_bytes), 0) AS request_bytes,
                    COALESCE(SUM(l.response_bytes), 0) AS response_bytes,
                    COALESCE(SUM(l.input_tokens), 0) AS input_tokens,
                    COALESCE(SUM(l.output_tokens), 0) AS output_tokens,
                    COALESCE(SUM(l.total_tokens), 0) AS total_tokens,
                    CAST(AVG(l.duration_ms) AS INTEGER) AS avg_duration_ms
                FROM api_keys k
                LEFT JOIN request_logs l
                    ON l.api_key_id = k.id AND l.created_at >= ?
                GROUP BY k.id
                ORDER BY total_tokens DESC, requests DESC
                """,
                (since,),
            ).fetchall()
        return [dict(row) for row in rows]

    def recent_logs(self, *, limit: int = 50) -> list[dict[str, Any]]:
        bounded_limit = max(1, min(limit, 200))
        with self._lock:
            rows = self._conn.execute(
                """
                SELECT
                    l.id, l.created_at, k.name AS api_key_name, k.key_prefix,
                    l.method, l.path, l.model, l.upstream_status, l.request_bytes,
                    l.response_bytes, l.input_tokens, l.output_tokens,
                    l.total_tokens, l.duration_ms, l.error
                FROM request_logs l
                LEFT JOIN api_keys k ON k.id = l.api_key_id
                ORDER BY l.created_at DESC
                LIMIT ?
                """,
                (bounded_limit,),
            ).fetchall()
        return [dict(row) for row in rows]

    @staticmethod
    def _row_to_key(row: sqlite3.Row) -> ApiKeyRecord:
        return ApiKeyRecord(
            id=row["id"],
            name=row["name"],
            key_hash=row["key_hash"],
            key_prefix=row["key_prefix"],
            active=bool(row["active"]),
            rpm_limit=row["rpm_limit"],
            daily_token_limit=row["daily_token_limit"],
            created_at=row["created_at"],
            revoked_at=row["revoked_at"],
        )
