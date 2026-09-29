"""Launcher regression check; no daemon or downloaded dependencies required."""
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import time
import unittest


FAKE_DOCKER = '''#!/usr/bin/env python3
import os, pathlib, sys, time
args = sys.argv[1:]
root = pathlib.Path(os.environ['FAKE_ROOT'])
with (root / 'docker.log').open('a') as f:
    f.write(' '.join(args) + '\\n')
mode = os.environ.get('FAKE_MODE', '')
if args[:2] == ['image', 'inspect']:
    sys.exit(1 if mode == 'no-image' else 0)
if args[0] == 'compose':
    command = args[5]
    if command == 'build' and mode in ('build-fail', 'no-image'):
        sys.exit(7)
    if command == 'up' and mode == 'up-fail':
        sys.exit(8)
    if command == 'run':
        (root / 'started').touch()
    if command == 'down' and mode == 'interrupt':
        assert (root / 'stopped').exists(), 'dependencies removed before runner stopped'
if args[0] == 'stop':
    (root / 'stopped').touch()
if args[0] == 'wait':
    while mode == 'interrupt' and not (root / 'stopped').exists():
        time.sleep(0.02)
    print(0)
if args[0] == 'run':
    # Execute the actual report helper, substituting host paths for mounts.
    os.execv(sys.executable, [sys.executable, str(root / 'tools/lab2/setup_failure.py'),
        '--artifacts', os.environ['LAB2_OUTPUT'], '--history', os.environ['LAB2_HISTORY_DIR']])
'''


class LauncherTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        target = self.root / 'tools/lab2'
        target.mkdir(parents=True)
        for name in ('run.sh', 'setup_failure.py'):
            source = Path(__file__).with_name(name)
            if source.exists():
                shutil.copy(source, target / name)
        binary = self.root / 'bin'
        binary.mkdir()
        (binary / 'docker').write_text(FAKE_DOCKER)
        (binary / 'docker').chmod(0o755)
        self.env = dict(os.environ, PATH=f'{binary}:{os.environ["PATH"]}',
                        FAKE_ROOT=str(self.root), LAB2_RUN_ID='test-run', LAB2_SKIP_BUILD='0')
        self.command = ['bash', str(target / 'run.sh')]
        self.output = self.root / 'build/lab2/runs/test-run'

    def run_launcher(self, mode=''):
        return subprocess.run(self.command, env=dict(self.env, FAKE_MODE=mode),
                              capture_output=True, text=True, timeout=15)

    def test_clean_checkout(self):
        result = self.run_launcher()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue((self.output / 'teardown.log').exists())

    def test_rejects_colliding_ids(self):
        for value in ('Foo', 'x_y', '../escape'):
            self.env['LAB2_RUN_ID'] = value
            self.assertEqual(self.run_launcher().returncode, 2, value)
        self.assertFalse((self.root / 'docker.log').exists())

    def test_setup_failures_record_failed_setup_and_skipped_stages(self):
        for mode in ('build-fail', 'up-fail', 'no-image'):
            with self.subTest(mode=mode):
                self.env['LAB2_RUN_ID'] = mode
                self.assertNotEqual(self.run_launcher(mode).returncode, 0)
                output = self.output.parent / mode
                results = [json.loads(p.read_text()) for p in (output / 'allure-results').glob('*-result.json')]
                self.assertEqual({r['name']: r['status'] for r in results},
                                 {'setup': 'failed', 'unit': 'skipped', 'integration': 'skipped', 'e2e': 'skipped'})
                self.assertTrue((output / 'setup-failure.txt').exists())
                self.assertTrue((output / 'teardown.log').exists())

    def test_interrupt_stops_runner_before_dependencies(self):
        for signum, expected in ((signal.SIGTERM, 143), (signal.SIGINT, 130)):
            with self.subTest(signal=signum):
                self.env['LAB2_RUN_ID'] = f'signal-{signum}'
                for name in ('started', 'stopped'):
                    (self.root / name).unlink(missing_ok=True)
                proc = subprocess.Popen(self.command, env=dict(self.env, FAKE_MODE='interrupt'),
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                try:
                    deadline = time.monotonic() + 5
                    while not (self.root / 'started').exists() and proc.poll() is None and time.monotonic() < deadline:
                        time.sleep(0.02)
                    self.assertTrue((self.root / 'started').exists())
                    proc.send_signal(signum)
                    stdout, stderr = proc.communicate(timeout=10)
                    self.assertEqual(proc.returncode, expected, stdout + stderr)
                    self.assertTrue((self.root / 'stopped').exists())
                finally:
                    if proc.poll() is None:
                        proc.kill()
                    proc.communicate()


if __name__ == '__main__':
    unittest.main()
