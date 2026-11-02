"""Exercise failure paths without starting Docker or handling real credentials."""
import json
from pathlib import Path
import tempfile
import unittest

import run
from entrypoint import safe_test_event


class RunnerTests(unittest.TestCase):
    def execute_fake(self, fail_phase=None, interrupt=False):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        output = Path(temporary.name) / 'published'
        calls, private_paths, credentials = [], [], []

        def invoke(argv, env, timeout):
            calls.append(argv)
            private = Path(env['LAB5_SECRET_DIR'])
            private_paths.append(private)
            if 'up' in argv:
                self.assertFalse(private.is_relative_to(output))
                self.assertEqual(private.stat().st_mode & 0o777, 0o700)
                for path in private.iterdir():
                    self.assertEqual(path.stat().st_mode & 0o777, 0o600)
                config = json.loads((private / 'api_config').read_text())
                self.assertGreaterEqual(len(config['auth']['jwt_secret']), 64)
                credentials.extend([config['auth']['jwt_secret'],
                                    (private / 'database_password').read_text().strip()])
            if '--domain-tests' in argv:
                return 4 if fail_phase == 'domain' else 0
            if 'tests' in argv and 'run' in argv:
                if interrupt:
                    raise KeyboardInterrupt
                (output / 'allure-results' / 'scenario-result.json').write_text(
                    json.dumps({'name': 'Scenario', 'status': 'failed' if fail_phase == 'tests' else 'passed'}))
                if fail_phase == 'tests':
                    return 7
            if 'generate' in argv:
                self.assertEqual(list(private.iterdir()), [])
                if fail_phase == 'report':
                    return 9
            if fail_phase == 'build' and 'build' in argv:
                return 5
            if fail_phase == 'start' and 'up' in argv:
                return 6
            if fail_phase == 'exception' and 'up' in argv:
                raise RuntimeError(credentials[0])
            if fail_phase == 'cleanup' and 'down' in argv:
                return 8
            return 0

        code = run.execute(output, 'test-run', invoke, Path(temporary.name) / 'private')
        self.assertTrue(any('down' in argv and '-v' in argv for argv in calls))
        self.assertTrue(all(not path.exists() for path in private_paths))
        self.assertEqual(json.loads((output / 'manifest.json').read_text())['exit_code'], code)
        # Artifact files and safe logs must never contain any generated config.
        self.assertFalse(any(path.name in ('api_config', 'database_url', 'database_password')
                             for path in output.rglob('*')))
        for path in output.rglob('*'):
            if path.is_file():
                for secret in credentials:
                    self.assertNotIn(secret, path.read_text())
        return code, calls, output

    def test_success_generates_allure_after_cleanup(self):
        code, calls, _ = self.execute_fake()
        self.assertEqual(code, 0)
        self.assertLess(next(i for i, args in enumerate(calls) if 'down' in args),
                        next(i for i, args in enumerate(calls) if 'generate' in args))
        self.assertTrue(any(args[:3] == ['docker', 'rm', '-f'] for args in calls))

    def test_failed_scenarios_preserve_status_and_generate_allure(self):
        code, calls, _ = self.execute_fake('tests')
        self.assertEqual(code, 7)
        self.assertTrue(any('generate' in argv for argv in calls))

    def test_domain_failure_still_runs_bdd_and_preserves_failure(self):
        code, calls, output = self.execute_fake('domain')
        self.assertEqual(code, 4)
        self.assertTrue((output / 'allure-results/scenario-result.json').exists())
        self.assertTrue(any('generate' in argv for argv in calls))

    def test_domain_events_drop_output_and_dynamic_names(self):
        self.assertIsNone(safe_test_event(json.dumps({'Action': 'output', 'Output': 'secret DSN'})))
        self.assertIsNone(safe_test_event(json.dumps({'Action': 'fail', 'Test': 'TestAuth/secret-token'})))
        self.assertEqual(safe_test_event(json.dumps({'Action': 'fail', 'Test': 'TestAuth',
                                                    'Output': 'secret DSN'})),
                         {'test': 'TestAuth', 'status': 'fail'})

    def test_start_failure_is_broken_and_generates_allure(self):
        code, calls, output = self.execute_fake('start')
        self.assertEqual(code, 6)
        self.assertTrue(any('generate' in argv for argv in calls))
        results = [json.loads(path.read_text()) for path in (output / 'allure-results').glob('*-result.json')]
        self.assertTrue(any(result['status'] == 'broken' for result in results))

    def test_build_failure_still_cleans_up_and_writes_readable_report(self):
        code, _, output = self.execute_fake('build')
        self.assertEqual(code, 5)
        self.assertIn('generator was unavailable', (output / 'allure-report/index.html').read_text())

    def test_report_failure_does_not_report_success(self):
        code, _, output = self.execute_fake('report')
        self.assertNotEqual(code, 0)
        self.assertTrue((output / 'allure-report/index.html').exists())

    def test_cleanup_failure_does_not_report_success(self):
        code, calls, _ = self.execute_fake('cleanup')
        self.assertEqual(code, 8)
        self.assertTrue(any('generate' in argv for argv in calls))

    def test_interruption_cleans_up_and_generates_report(self):
        code, calls, _ = self.execute_fake(interrupt=True)
        self.assertEqual(code, 130)
        self.assertTrue(any('generate' in argv for argv in calls))

    def test_secret_bearing_exception_is_never_published(self):
        code, calls, _ = self.execute_fake('exception')
        self.assertEqual(code, 1)
        self.assertTrue(any('generate' in argv for argv in calls))


if __name__ == '__main__':
    unittest.main()
