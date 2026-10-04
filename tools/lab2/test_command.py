"""Command output must reach CI before the command exits, and remain in artifacts."""
from contextlib import redirect_stdout
import io
import json
from pathlib import Path
import select
import subprocess
import sys
import tempfile
import unittest

from tools.lab2.command import run


class CommandTests(unittest.TestCase):
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
