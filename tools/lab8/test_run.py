import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from tools.lab8 import run as runner


class RunnerTests(unittest.TestCase):
    def test_configuration_keeps_real_and_mock_upstream_separate(self):
        mock = runner.configuration('mock')['rdap']
        real = runner.configuration('real')['rdap']
        self.assertEqual(mock['base_url'], 'http://rdap-mock:8089/com/v1')
        self.assertEqual(mock['timeout_ms'], 750)
        self.assertEqual(real['base_url'], 'https://rdap.verisign.com/com/v1')
        self.assertEqual(real['timeout_ms'], 10000)

    def test_start_failure_still_removes_volumes_config_and_records_failure(self):
        calls = []

        def command(argv, *_args, **_kwargs):
            calls.append(argv)
            if 'up' in argv:
                raise RuntimeError('start failed')

        with tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ), patch.object(runner, 'run', command):
            output = Path(tmp)
            record = runner.contour(output, 'id', 'e2e', 'mock', 'app', 'postgres')
            directory = output / 'e2e-mock'
            self.assertIn('start failed', record['error'])
            self.assertTrue(record['cleanup'])
            self.assertFalse((directory / 'config.json').exists())
            self.assertTrue(any('down' in command and '-v' in command for command in calls))
            self.assertEqual(json.loads((directory / 'experiment.json').read_text()), record)

    def test_failed_demo_publication_removes_temporary_token(self):
        with tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ):
            output = Path(tmp)
            directory = output/'demo-mock'
            def command(argv, *_args, **_kwargs):
                if 'run' in argv and 'tests' in argv:
                    (directory/'demo-session.json').write_text('{"token":"ephemeral-fixture"}')
            with patch.object(runner, 'run', command), \
                    patch.object(runner, 'mock_evidence', side_effect=RuntimeError('stats failed')):
                record = runner.contour(output, 'id', 'demo', 'mock', 'app', 'postgres')
            self.assertIn('stats failed', record['error'])
            self.assertTrue(record['cleanup'])
            self.assertFalse((directory/'demo-session.json').exists())

    def test_interrupt_during_load_still_cleans_and_persists_record(self):
        calls = []

        def command(argv, *_args, **_kwargs):
            calls.append(argv)

        with tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ), patch.object(runner, 'run', command), \
                patch.object(runner, 'capture', return_value='image-id'), \
                patch.object(runner, 'fixed_load', side_effect=KeyboardInterrupt):
            output = Path(tmp)
            with self.assertRaises(KeyboardInterrupt):
                runner.contour(output, 'id', 'load', 'mock', 'app', 'postgres')
            directory = output / 'load/profiles/soak-0'
            self.assertFalse((directory / 'config.json').exists())
            record = json.loads((directory / 'experiment.json').read_text())
            self.assertTrue(record['cleanup'])
            self.assertIn('KeyboardInterrupt', record['error'])
            self.assertTrue(any('down' in command for command in calls))

    def test_safe_build_context_excludes_private_and_dependency_directories(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'root'
            for name in ['internal/service.go', 'tools/lab1/package.json',
                         'tools/lab1/node_modules/private', 'docker/web/secret',
                         'tools/lab8/__pycache__/private', 'tools/lab8/.env']:
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text('source')
            with patch.object(runner, 'ROOT', root):
                context = runner.safe_context(Path(tmp) / 'context')
            files = {str(path.relative_to(context)) for path in context.rglob('*') if path.is_file()}
            self.assertEqual(files, {'internal/service.go', 'tools/lab1/package.json'})

    def test_load_evidence_rejects_missing_upstream_calls(self):
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            runner.write_json(directory / 'summary.json', {'profiles': [{
                'unexpected_requests': 0, 'operations': {'domain-registration': {'requests': 50}}}]})
            runner.write_json(directory / 'profiles/soak-0/rdap-mock-stats.json', {'requests': {}})
            with self.assertRaisesRegex(RuntimeError, 'upstream calls'):
                runner.validate_load(directory, {}, False)
            runner.write_json(directory / 'profiles/soak-0/rdap-mock-stats.json', {
                'requests': {'example.com': {'GET': 50}}})
            runner.write_json(directory / 'profiles/soak-0/rdap-mock-stats.json', {
                'requests': {'example.com': {'GET': 49}}})
            with self.assertRaises(RuntimeError):
                runner.validate_load(directory, {}, False)
            runner.write_json(directory / 'profiles/soak-0/rdap-mock-stats.json', {
                'requests': {'example.com': {'GET': 50}}})
            record = {}
            runner.validate_load(directory, record, False)
            self.assertTrue(record['rdap_evidence']['full_load'])

    def test_full_profile_contract(self):
        compose = (runner.ROOT / 'tools/lab8/compose.yaml').read_text()
        self.assertIn('LAB8_RDAP:', compose)
        self.assertIn('http://example.com:8081/health', compose)
        self.assertIn('aliases: [example.com]', compose)
        self.assertNotIn('internal: true', compose)
        env = runner.environment(Path('/tmp/out'), 'app', 'db', Path('/tmp/config'), 'mock')
        self.assertEqual(env['LAB4_PROFILE'], 'soak-0')
        self.assertEqual(env['LAB4_SMOKE'], '0')
        self.assertEqual(env['LAB8_API_PORT'], '0')


if __name__ == '__main__':
    unittest.main()
