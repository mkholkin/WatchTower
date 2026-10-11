"""Command output must reach CI before the command exits, and remain in artifacts."""
from contextlib import redirect_stdout
import io
import json
import os
from pathlib import Path
import select
import signal
import subprocess
import sys
import tempfile
import unittest

from tools.lab2.command import run


class CommandTests(unittest.TestCase):
    def test_fail_fast_kills_descendants_that_keep_output_open(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            leader = root / 'leader'
            ready = root / 'ready'
            child = (
                'import json, os, pathlib, signal, sys, time\n'
                f'pathlib.Path({str(leader)!r}).write_text(str(os.getpid()))\n'
                'signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))\n'
                'if os.fork() == 0:\n'
                '    signal.signal(signal.SIGTERM, signal.SIG_IGN)\n'
                f'    pathlib.Path({str(ready)!r}).touch()\n'
                '    time.sleep(30)\n'
                '    sys.exit(0)\n'
                f'while not pathlib.Path({str(ready)!r}).exists(): time.sleep(0.01)\n'
                'print(json.dumps({"Action": "fail", "Package": "example/unit"}), flush=True)\n'
                'time.sleep(30)\n'
            )
            runner = (
                'import sys\nfrom pathlib import Path\nfrom tools.lab2.command import run\n'
                'try:\n'
                f'    run({[sys.executable, "-c", child]!r}, Path({str(root / "unit.jsonl")!r}), '
                'timeout=2, fail_fast=True)\n'
                'except RuntimeError:\n    sys.exit(0)\n'
                'sys.exit(1)\n'
            )
            proc = subprocess.Popen([sys.executable, '-c', runner], stdout=subprocess.PIPE,
                                    stderr=subprocess.STDOUT, text=True)
            try:
                try:
                    output, _ = proc.communicate(timeout=4)
                except subprocess.TimeoutExpired:
                    self.fail('Fail-fast hung on a descendant holding stdout open')
                self.assertEqual(proc.returncode, 0, output)
            finally:
                if leader.exists():
                    try:
                        os.killpg(int(leader.read_text()), signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                if proc.poll() is None:
                    proc.kill()
                proc.communicate(timeout=5)

    def test_fail_fast_stops_remaining_work_and_preserves_failure(self):
        for test_name in ('TestBroken/case', None):
            with self.subTest(test_name=test_name), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                marker = root / 'continued'
                log = root / 'unit.jsonl'
                event = {'Action': 'fail', 'Package': 'example/unit', 'Elapsed': 0.01}
                if test_name:
                    event['Test'] = test_name
                raw = json.dumps(event) + '\n'
                child = (
                    'import pathlib, signal, sys, time\n'
                    # A graceful termination can return zero; the test failure must still win.
                    'signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))\n'
                    f'print({raw!r}, end="", flush=True)\n'
                    'time.sleep(0.5)\n'
                    f'pathlib.Path({str(marker)!r}).touch()\n'
                )
                with self.assertRaisesRegex(RuntimeError, 'example/unit'):
                    run([sys.executable, '-c', child], log, timeout=5, fail_fast=True)
                self.assertFalse(marker.exists(), 'Work continued after the first failure')
                self.assertIn(raw, log.read_text())

    def test_fail_fast_ignores_diagnostics_and_successful_events(self):
        with tempfile.TemporaryDirectory() as directory, redirect_stdout(io.StringIO()):
            log = Path(directory) / 'unit.jsonl'
            raw = ('compiler diagnostic\n' + json.dumps(['fail']) + '\n' +
                   json.dumps({'Action': 'output', 'Output': 'FAIL is just test output\n'}) + '\n' +
                   json.dumps({'Action': 'pass', 'Package': 'example/unit'}) + '\n')
            run([sys.executable, '-c', f'print({raw!r}, end="", flush=True)'],
                log, fail_fast=True, stream=False)
            self.assertEqual(log.read_text(), raw)

    def test_streams_output_before_exit_and_keeps_raw_artifact(self):
        for go_json in (False, True):
            with self.subTest(go_json=go_json), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                log = root / ('e2e.jsonl' if go_json else 'curl-replay.log')
                release = root / 'release'
                message = 'POST /monitors -> 201\n'
                raw = json.dumps({'Action': 'output', 'Output': message}) + '\n' if go_json else message
                child = (
                    'import pathlib, sys, time\n'
                    f'sys.stderr.write({raw!r}); sys.stderr.flush()\n'
                    f'while not pathlib.Path({str(release)!r}).exists(): time.sleep(0.01)\n'
                )
                runner = (
                    'from pathlib import Path\n'
                    'from tools.lab2.command import run\n'
                    f'run({[sys.executable, "-c", child]!r}, Path({str(log)!r}), timeout=10)\n'
                )
                proc = subprocess.Popen([sys.executable, '-c', runner], stdout=subprocess.PIPE,
                                        stderr=subprocess.STDOUT, text=True)
                try:
                    ready, _, _ = select.select([proc.stdout], [], [], 3)
                    self.assertTrue(ready, 'Output is buffered until command completion')
                    self.assertEqual(proc.stdout.readline(), message)
                    self.assertEqual(log.read_text(), raw)
                    self.assertIsNone(proc.poll())
                finally:
                    release.touch()
                    output, _ = proc.communicate(timeout=10)
                self.assertEqual(proc.returncode, 0, output)
                self.assertEqual(log.read_text(), raw)

    def test_failed_command_preserves_output_and_raises(self):
        with tempfile.TemporaryDirectory() as directory:
            log = Path(directory) / 'failure.log'
            with self.assertRaisesRegex(RuntimeError, 'exit 7'):
                run([sys.executable, '-c', "print('failed'); raise SystemExit(7)"], log)
            self.assertEqual(log.read_text(), 'failed\n')

    def test_timeout_stops_command(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            marker = root / 'terminated'
            child = (
                'import pathlib, signal, time, sys\n'
                f'def stop(*_): pathlib.Path({str(marker)!r}).touch(); sys.exit(0)\n'
                'signal.signal(signal.SIGTERM, stop)\n'
                'print("started", flush=True)\n'
                'time.sleep(30)\n'
            )
            with self.assertRaises(subprocess.TimeoutExpired):
                run([sys.executable, '-c', child], root / 'timeout.log', timeout=1)
            self.assertTrue(marker.exists())

    def test_captured_traffic_stays_in_artifacts(self):
        with tempfile.TemporaryDirectory() as directory, redirect_stdout(io.StringIO()) as console:
            log = Path(directory) / 'traffic.txt'
            run([sys.executable, '-c', "import sys; sys.stdout.buffer.write(b'HTTP body: \\xff\\n')"],
                log, stream=False)
            self.assertEqual(console.getvalue(), '')
            self.assertEqual(log.read_bytes(), b'HTTP body: \xff\n')


if __name__ == '__main__':
    unittest.main()
