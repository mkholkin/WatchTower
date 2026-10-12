import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

from tools.lab4 import run as runner


class RunnerTests(unittest.TestCase):
    def test_allure_generation_is_attempted_after_report_failure(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(runner, 'run', side_effect=[RuntimeError('integrity'), None]) as command:
            with self.assertRaisesRegex(RuntimeError, 'integrity'):
                runner.generate_reports('test-image', Path(tmp))
            self.assertEqual(command.call_count, 2)
            self.assertIn('/workspace/tools/lab1/node_modules/allure/cli.js', command.call_args.args[0])

    def test_sigterm_unwinds_main_finally(self):
        script = '''
import time
from tools.lab4 import run
def fake_main():
    try:
        print('ready', flush=True)
        time.sleep(30)
    finally:
        print('cleanup', flush=True)
run.main = fake_main
run.entrypoint()
'''
        process = subprocess.Popen([sys.executable, '-u', '-c', script], stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, text=True)
        try:
            self.assertEqual(process.stdout.readline().strip(), 'ready')
            os.kill(process.pid, signal.SIGTERM)
            stdout, _ = process.communicate(timeout=10)
            self.assertIn('cleanup', stdout)
            self.assertNotEqual(process.returncode, 0)
        finally:
            if process.poll() is None:
                process.kill()
                process.wait(timeout=5)
            process.stdout.close()
            process.stderr.close()


if __name__ == '__main__':
    unittest.main()
