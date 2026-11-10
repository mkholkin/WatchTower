import contextlib
import io
import json
from pathlib import Path
import tempfile
import os
import sys
import threading
import importlib.util
from types import SimpleNamespace
from unittest.mock import patch
import time

import run
import unittest

from progress import poll_progress


class ProgressTests(unittest.TestCase):
    def test_completed_events_are_printed_once_and_partial_lines_wait(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            go = root/'domain-tests.jsonl'
            go.write_text('{"test":"TestLogin","status":"run"}\n')
            bdd = root/'bdd-progress.jsonl'
            bdd.write_text('{"line":11,"status":"passed"}')
            seen = {}
            log = io.StringIO()
            with contextlib.redirect_stdout(log):
                poll_progress(root, seen)
                poll_progress(root, seen)
                bdd.write_text(bdd.read_text()+'\n')
                poll_progress(root, seen)
            self.assertEqual(log.getvalue().count('TestLogin'), 1)
            self.assertIn('BDD PASSED', log.getvalue())
            self.assertIn('Password alone', log.getvalue())

    def test_runtime_payloads_and_malformed_events_are_not_printed(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root/'domain-tests.jsonl').write_text('\n'.join([
                json.dumps({'test': 'TestLogin', 'status': 'fail', 'Output': 'private-password'}),
                json.dumps({'test': 'private-password', 'status': 'fail'}), 'invalid',
            ])+'\n')
            (root/'bdd-progress.jsonl').write_text(json.dumps({'line': 11, 'status': 'failed', 'error': 'private-token'})+'\n')
            log = io.StringIO()
            with contextlib.redirect_stdout(log):
                poll_progress(root, {})
            self.assertEqual(log.getvalue(), '')


class CommandProgressTests(unittest.TestCase):
    def test_progress_arrives_before_process_exit_and_raw_output_stays_private(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            code = """import pathlib, sys, time
root=pathlib.Path(sys.argv[1])
(root/'domain-tests.jsonl').write_text('{"test":"TestLive","status":"run"}\\n')
print('private-password', flush=True)
while not (root/'release').exists(): time.sleep(.01)
with (root/'domain-tests.jsonl').open('a') as f: f.write('{"test":"TestLive","status":"pass"}\\n')
"""
            log = io.StringIO()
            result = []
            env = dict(os.environ, LAB5_OUTPUT=tmp)
            with contextlib.redirect_stdout(log):
                worker = threading.Thread(target=lambda: result.append(run.command([sys.executable, '-c', code, tmp], env, 10)))
                worker.start()
                try:
                    deadline = time.monotonic()+5
                    while 'GO RUN: TestLive' not in log.getvalue() and time.monotonic()<deadline:
                        time.sleep(.02)
                    self.assertIn('GO RUN: TestLive', log.getvalue())
                    self.assertTrue(worker.is_alive())
                finally:
                    (root/'release').touch()
                    worker.join(5)
            self.assertEqual(result, [0])
            self.assertIn('GO PASS: TestLive', log.getvalue())
            self.assertNotIn('private-password', log.getvalue())


class HookProgressTests(unittest.TestCase):
    def test_hook_progress_uses_only_source_line_and_status_even_on_cleanup_failure(self):
        spec = importlib.util.spec_from_file_location('lab5_environment', Path(__file__).resolve().parents[2]/'tests/lab5/features/environment.py')
        hooks = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(hooks)
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'progress.jsonl'
            scenario = SimpleNamespace(line=6, name='private-password', status=SimpleNamespace(name='passed'))
            class Fixture:
                def cleanup(self):
                    raise AssertionError('private-token')
            context = SimpleNamespace(lab5_fixtures=[])
            with patch.dict(os.environ, LAB5_PROGRESS_FILE=str(path)), patch.object(hooks, 'Fixture', Fixture):
                hooks.before_scenario(context, scenario)
                with self.assertRaises(AssertionError):
                    hooks.after_scenario(context, scenario)
            events = [json.loads(line) for line in path.read_text().splitlines()]
            self.assertEqual(events, [{'line': 6, 'status': 'run'}, {'line': 6, 'status': 'failed'}])
            self.assertNotIn('private', path.read_text())
