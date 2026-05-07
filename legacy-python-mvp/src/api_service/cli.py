from __future__ import annotations

import argparse
import json
from dataclasses import asdict

from .auth import generate_token, hash_token, token_prefix
from .config import Settings
from .db import Database


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="oauth-api-service")
    subparsers = parser.add_subparsers(dest="command", required=True)

    create_key = subparsers.add_parser("create-key", help="create a downstream API key")
    create_key.add_argument("--name", required=True)
    create_key.add_argument("--rpm-limit", type=int)
    create_key.add_argument("--daily-token-limit", type=int)

    subparsers.add_parser("list-keys", help="list downstream API keys")

    revoke_key = subparsers.add_parser("revoke-key", help="revoke a downstream API key")
    revoke_key.add_argument("key_id")

    usage = subparsers.add_parser("usage", help="show usage summary")
    usage.add_argument("--hours", type=int, default=24)

    recent = subparsers.add_parser("recent", help="show recent request logs")
    recent.add_argument("--limit", type=int, default=50)

    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    settings = Settings.from_env()
    db = Database(settings.database_path)
    try:
        if args.command == "create-key":
            token = generate_token()
            record = db.create_key(
                name=args.name,
                key_hash=hash_token(token),
                key_prefix=token_prefix(token),
                rpm_limit=args.rpm_limit,
                daily_token_limit=args.daily_token_limit,
            )
            payload = asdict(record)
            payload.pop("key_hash")
            print_json({"key": token, **payload})
            return 0

        if args.command == "list-keys":
            print_json({"data": db.list_keys()})
            return 0

        if args.command == "revoke-key":
            print_json({"id": args.key_id, "revoked": db.revoke_key(args.key_id)})
            return 0

        if args.command == "usage":
            print_json({"hours": args.hours, "data": db.usage_summary(hours=args.hours)})
            return 0

        if args.command == "recent":
            print_json({"data": db.recent_logs(limit=args.limit)})
            return 0
    finally:
        db.close()
    return 1


def print_json(payload: object) -> None:
    print(json.dumps(payload, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    raise SystemExit(main())
