import json
from pathlib import Path
import tempfile
import unittest
import uuid
from unittest.mock import patch

import e2e

from e2e import Suite, expect_status, run, validate_registration


class E2ETests(unittest.TestCase):
    def response(self):
        return {'domain': 'example.com', 'registered_at': '1995-08-14T04:00:00Z',
                'expires_at': '2027-08-13T04:00:00Z', 'source_url': 'http://rdap-mock:8089/domain/example.com'}

    def test_main_rejects_failure_outside_an_individual_step(self):
        with tempfile.TemporaryDirectory() as directory, patch.dict('os.environ',
                {'LAB8_MODE': 'mock', 'LAB8_OUTPUT': directory}), \
                patch.object(e2e, 'run', side_effect=RuntimeError('mock stats unavailable')):
            self.assertEqual(e2e.main(), 1)
            report = json.loads((Path(directory)/'e2e.json').read_text())
            self.assertEqual(report['status'], 'failed')

    def test_expected_status_rejects_errors_without_dumping_response(self):
        self.assertEqual(expect_status((200, {'ok': True}), 200), {'ok': True})
        with self.assertRaisesRegex(AssertionError, '^HTTP 502, expected 200$'):
            expect_status((502, {'sensitive': 'body'}), 200)

    def test_mock_dates_and_source_are_checked(self):
        body = self.response()
        validate_registration(body, 'example.com', 'mock', 'http://rdap-mock:8089')
        for key, value in [('domain', 'other.com'), ('registered_at', '2000-01-01T00:00:00Z'),
                           ('source_url', 'https://evil.invalid/domain/example.com')]:
            with self.subTest(key=key), self.assertRaises(AssertionError):
                validate_registration({**body, key: value}, 'example.com', 'mock', 'http://rdap-mock:8089')

    def test_partial_and_real_dates(self):
        body = self.response()
        validate_registration({**body, 'expires_at': None}, 'example.com', 'mock', partial=True)
        body.update(registered_at='1996-01-01T01:02:03+01:00', expires_at='2030-01-01T01:02:03.123Z')
        validate_registration(body, 'example.com', 'real')
        for value in (None, '2030-01-01', '2030-99-01T00:00:00Z', 123):
            with self.subTest(value=value), self.assertRaises((AssertionError, ValueError)):
                validate_registration({**body, 'expires_at': value}, 'example.com', 'real')
        del body['expires_at']
        with self.assertRaises(AssertionError):
            validate_registration(body, 'example.com', 'real')

    def test_real_scenario_performs_one_lookup_and_deletes_owned_monitor(self):
        class RunningAPI:
            def __init__(self):
                self.monitors = {}
                self.lookups = 0
                self.tokens = []

            def request(self, method, path, body=None, token=''):
                if path == '/auth/register':
                    return 201, {'login': body['login']}
                if path == '/auth/login':
                    self.tokens.append('token' + str(len(self.tokens)))
                    return 200, {'access_token': self.tokens[-1]}
                if method == 'POST' and path == '/monitors':
                    monitor_id = str(uuid.uuid4())
                    self.monitors[monitor_id] = body
                    return 201, {'id': monitor_id}
                monitor_id = path.split('/')[2]
                if path.endswith('/disable'):
                    return 200, {}
                if method == 'DELETE':
                    del self.monitors[monitor_id]
                    return 204, None
                if not token:
                    return 401, {}
                if token != self.tokens[0]:
                    return 403, {}
                try:
                    uuid.UUID(monitor_id)
                except ValueError:
                    return 400, {}
                if monitor_id not in self.monitors:
                    return 404, {}
                self.lookups += 1
                return 200, {'domain': 'example.com', 'registered_at': '2001-01-01T00:00:00Z',
                             'expires_at': '2032-01-01T00:00:00Z',
                             'source_url': 'https://rdap.verisign.com/com/v1/domain/example.com'}

        with tempfile.TemporaryDirectory() as directory:
            api = RunningAPI()
            suite = Suite('real', directory)
            run(api, suite, 'http://unused.invalid')
            self.assertEqual(suite.finish(), 0)
            self.assertEqual(api.lookups, 1)
            self.assertEqual(api.monitors, {})

    def test_allure_and_summary_record_failed_steps_and_cleanup(self):
        with tempfile.TemporaryDirectory() as directory:
            suite = Suite('mock', directory)
            suite.step('setup', lambda: True)
            with self.assertRaises(AssertionError):
                suite.step('lookup', lambda: expect_status((503, None), 200))
            suite.step('cleanup', lambda: None)
            self.assertEqual(suite.finish(), 1)
            summary = json.loads((Path(directory) / 'e2e.json').read_text())
            self.assertEqual(summary['status'], 'failed')
            self.assertEqual([s['status'] for s in summary['steps']], ['passed', 'failed', 'passed'])
            records = [json.loads(p.read_text()) for p in (Path(directory) / 'allure-results').glob('*-result.json')]
            self.assertEqual(len(records), 3)
            failed = next(record for record in records if record['status'] == 'failed')
            self.assertEqual(failed['statusDetails']['message'], 'AssertionError: HTTP 503, expected 200')
            self.assertLessEqual(failed['start'], failed['stop'])


if __name__ == '__main__':
    unittest.main()
