#!/usr/bin/env python3

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path
from unittest import mock


MODULE_PATH = Path(__file__).with_name("codex-runtime-health-check.py")
sys.dont_write_bytecode = True
SPEC = importlib.util.spec_from_file_location("codex_runtime_health_check", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
HEALTH = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(HEALTH)


class PublicTLSCheckTest(unittest.TestCase):
    def report(self) -> dict:
        return {"status": "ok", "checks": []}

    def test_public_tls_is_ok_outside_warning_window(self) -> None:
        report = self.report()
        with mock.patch.object(
            HEALTH,
            "public_tls_details",
            return_value={"ok": True, "days_remaining": 45.0},
        ):
            HEALTH.check_public_tls(report)
        self.assertEqual(report["status"], "ok")
        self.assertEqual(report["checks"][0]["status"], "ok")

    def test_public_tls_warns_near_expiry(self) -> None:
        report = self.report()
        with mock.patch.object(
            HEALTH,
            "public_tls_details",
            return_value={"ok": True, "days_remaining": 10.0},
        ):
            HEALTH.check_public_tls(report)
        self.assertEqual(report["status"], "warn")
        self.assertEqual(report["checks"][0]["status"], "warn")

    def test_public_tls_fails_when_handshake_fails(self) -> None:
        report = self.report()
        with mock.patch.object(
            HEALTH,
            "public_tls_details",
            return_value={"ok": False, "error": "certificate expired"},
        ):
            HEALTH.check_public_tls(report)
        self.assertEqual(report["status"], "fail")
        self.assertEqual(report["checks"][0]["status"], "fail")


if __name__ == "__main__":
    unittest.main()
