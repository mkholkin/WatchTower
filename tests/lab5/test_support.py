"""Regression checks for secret-safe failures and reliable fixture cleanup."""

import io
import json
import secrets
import sys
import traceback
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent / "features"))

from support import Fixture, Response, request, status


class SafeFailureTests(unittest.TestCase):
    def test_transport_error_does_not_disclose_sensitive_cause(self):
        secret = secrets.token_urlsafe(24)
        with patch("urllib.request.urlopen", side_effect=OSError(secret)):
            try:
                request("http://api:8080", "POST", "/auth/mfa/login", {"password": secret})
            except AssertionError:
                report = traceback.format_exc()
            else:
                self.fail("Transport failure must fail the check")
        self.assertNotIn(secret, report)
        self.assertIn("HTTP transport or response decoding failed", report)

    def test_malformed_response_does_not_disclose_body(self):
        secret = secrets.token_urlsafe(24)
        response = io.BytesIO(json.dumps({"secret": secret}).encode() + b" invalid")
        response.status = 200
        response.headers = {}
        with patch("urllib.request.urlopen", return_value=response):
            try:
                request("http://api:8080", "POST", "/auth/mfa/verify")
            except AssertionError:
                report = traceback.format_exc()
            else:
                self.fail("Malformed response must fail the check")
        self.assertNotIn(secret, report)

    def test_status_assertion_does_not_disclose_body(self):
        secret = secrets.token_urlsafe(24)
        with self.assertRaises(AssertionError) as error:
            status(Response(500, {"access_token": secret}), 200)
        self.assertNotIn(secret, str(error.exception))
        self.assertEqual(str(error.exception), "Expected HTTP 200; got HTTP 500")

    def test_cleanup_continues_after_failure_and_retries_failed_accounts(self):
        fixture = Fixture()
        first, second = fixture.new_account(), fixture.new_account()
        failed_once = False
        removed = []

        def database(query, values=(), fetch=False):
            nonlocal failed_once
            if values == (first.login,) and not failed_once:
                failed_once = True
                raise AssertionError("Database fixture operation failed")
            if query.startswith("DELETE"):
                removed.append(values[0])
            return (0,) if fetch else None

        with patch.object(fixture, "sql", side_effect=database):
            with self.assertRaisesRegex(AssertionError, "Disposable account cleanup failed"):
                fixture.cleanup()
            self.assertEqual(removed, [second.login])
            self.assertEqual(fixture.accounts, [first])
            fixture.cleanup()
        self.assertEqual(removed, [second.login, first.login])
        self.assertEqual(fixture.accounts, [])


if __name__ == "__main__":
    unittest.main()
