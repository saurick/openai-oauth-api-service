from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Settings:
    openai_api_key: str
    admin_token: str
    openai_base_url: str = "https://api.openai.com/v1"
    database_path: Path = Path("data/api_service.sqlite3")
    upstream_proxy_url: str | None = None
    upstream_timeout_seconds: float = 600.0

    @classmethod
    def from_env(cls) -> "Settings":
        api_key = os.getenv("OPENAI_API_KEY", "")
        admin_token = os.getenv("ADMIN_TOKEN", "")
        return cls(
            openai_api_key=api_key,
            admin_token=admin_token,
            openai_base_url=os.getenv("OPENAI_BASE_URL", "https://api.openai.com/v1").rstrip("/"),
            database_path=Path(os.getenv("DATABASE_PATH", "data/api_service.sqlite3")),
            upstream_proxy_url=os.getenv("UPSTREAM_PROXY_URL") or None,
            upstream_timeout_seconds=float(os.getenv("UPSTREAM_TIMEOUT_SECONDS", "600")),
        )

    def validate_runtime(self) -> None:
        missing = []
        if not self.openai_api_key:
            missing.append("OPENAI_API_KEY")
        if not self.admin_token:
            missing.append("ADMIN_TOKEN")
        if missing:
            joined = ", ".join(missing)
            raise RuntimeError(f"Missing required environment variable(s): {joined}")
