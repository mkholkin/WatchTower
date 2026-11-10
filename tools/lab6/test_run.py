"""Control paths are tested without Docker or an application process."""
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from tools.lab6 import run as runner


class RunnerTests(unittest.TestCase):
    def experiment(self, root, variant='baseline'):
        with patch.object(runner.time, 'sleep'):
            return runner.experiment(root, 'test', 'ramp', variant, 1, 'app', 'postgres', True)

    def test_start_failure_still_removes_config_and_cleans_volumes(self):
        def command(args, *_args, **_kwargs):
            if '--wait' in args:
                raise RuntimeError('startup failed')
        with tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ), patch.object(runner, 'run', side_effect=command) as run, patch.object(runner, 'capture', return_value=''):
            record = self.experiment(Path(tmp))
            self.assertIn('startup failed', record['error'])
            self.assertTrue(record['cleanup']['success'])
            self.assertTrue(any('down' in call.args[0] and '-v' in call.args[0] for call in run.call_args_list))
            self.assertFalse(list(Path(tmp).rglob('config.json')))
            self.assertEqual(json.loads(next(Path(tmp).rglob('experiment.json')).read_text()), record)

    def test_interrupt_still_cleans_and_persists_record(self):
        def command(args, *_args, **_kwargs):
            if '--wait' in args:
                raise KeyboardInterrupt()
        with tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ), patch.object(runner, 'run', side_effect=command), patch.object(runner, 'capture', return_value=''):
            with self.assertRaises(KeyboardInterrupt):
                self.experiment(Path(tmp))
            record = json.loads(next(Path(tmp).rglob('experiment.json')).read_text())
            self.assertEqual(record['error'], 'KeyboardInterrupt')
            self.assertTrue(record['cleanup']['success'])
            self.assertFalse(list(Path(tmp).rglob('config.json')))

    def test_cleanup_failure_is_explicit(self):
        def command(args, *_args, **_kwargs):
            if '--wait' in args or 'down' in args:
                raise RuntimeError('docker unavailable')
        with tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ), patch.object(runner, 'run', side_effect=command), patch.object(runner, 'capture', return_value=''):
            record = self.experiment(Path(tmp))
            self.assertFalse(record['cleanup']['success'])
            self.assertIn('docker unavailable', record['cleanup']['error'])
            self.assertFalse(list(Path(tmp).rglob('config.json')))

    def test_matrix_flags_and_successful_load(self):
        def capture(args):
            if args[:2] == ['docker', 'inspect']:
                return '0' if '{{.State.ExitCode}}' in args else '[]'
            if '--format' in args:
                return 'sha256:test'
            return 'container' if '-aq' in args else ''
        for variant, options in runner.VARIANTS.items():
            with self.subTest(variant=variant), tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ), patch.object(runner, 'run'), patch.object(runner, 'capture', side_effect=capture):
                record = self.experiment(Path(tmp), variant)
                self.assertEqual(record['exit_code'], 0)
                self.assertIn(f'-{variant}-1-ramp', record['image'])
                self.assertEqual((Path(tmp)/record['path']/'.dockerignore').read_text(), '*\n')
                self.assertTrue(record['cleanup']['success'])
                self.assertEqual(os.environ['LAB6_HTTP_DEBUG'], str(options['extended_logs']).lower())
                self.assertEqual(bool(os.environ['LAB6_TRACE_FILE']), options['tracing'])
                self.assertEqual(runner.configuration(options['extended_logs'])['logging']['level'], 'debug' if options['extended_logs'] else 'info')

    def test_cached_dependency_does_not_contact_registry(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(runner, 'capture', return_value='sha256:cached'), patch.object(runner, 'run') as command:
            self.assertEqual(runner.ensure_image('redis:7.4-bookworm', Path(tmp)), 'sha256:cached')
            command.assert_not_called()

    def test_missing_dependency_is_pulled_then_inspected(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(runner, 'capture', side_effect=[RuntimeError('missing'), 'sha256:pulled']), patch.object(runner, 'run') as command:
            self.assertEqual(runner.ensure_image('redis:7.4-bookworm', Path(tmp)), 'sha256:pulled')
            self.assertEqual(command.call_args.args[0], ['docker', 'pull', 'redis:7.4-bookworm'])

    def test_allure_is_attempted_after_python_report_failure(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(runner, 'run', side_effect=[RuntimeError('report failed'), None]) as command:
            with self.assertRaisesRegex(RuntimeError, 'report failed'):
                runner.generate_reports('app', Path(tmp))
            self.assertEqual(command.call_count, 2)
            self.assertIn('/workspace/tools/lab1/node_modules/allure/cli.js', command.call_args.args[0])

    def test_image_context_excludes_local_secrets(self):
        patterns = (runner.ROOT/'tools/lab6/Dockerfile.dockerignore').read_text().splitlines()
        self.assertEqual(patterns, (runner.ROOT/'tools/lab6/Dockerfile.cached.dockerignore').read_text().splitlines())
        for path in ('docker', '.git', '.agents', '.codex', '.aws', '.env', '.env.*', 'build'):
            self.assertIn(path, patterns)

    def test_nonpositive_repetitions_fail_before_docker(self):
        with patch('sys.argv', ['lab6', '--repetitions', '0']), patch.object(runner, 'capture') as capture:
            with self.assertRaises(SystemExit):
                runner.main()
            capture.assert_not_called()


if __name__ == '__main__':
    unittest.main()
