"""Execute pipeline commands with bounded process cleanup."""
from concurrent.futures import ThreadPoolExecutor
import json
import os
import signal
import subprocess


def stop(proc, sig=signal.SIGTERM):
    if proc is not None and proc.poll() is None:
        os.killpg(proc.pid, sig)
        try:
            proc.wait(timeout=10)
        except subprocess.TimeoutExpired:
            os.killpg(proc.pid, signal.SIGKILL)
            proc.wait(timeout=5)


def run(cmd, log, timeout=600, stream=True, fail_fast=False):
    failure = None
    with log.open('wb') as output, ThreadPoolExecutor(max_workers=1) as readers:
        proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                start_new_session=True)

        def copy_output():
            nonlocal failure
            with proc.stdout:
                for line in proc.stdout:
                    output.write(line)
                    output.flush()
                    visible = line.decode('utf-8', errors='replace')
                    event = None
                    if log.suffix == '.jsonl':
                        try:
                            event = json.loads(visible)
                        except json.JSONDecodeError:
                            pass  # Preserve compiler/runtime diagnostics that are not JSON.
                        if isinstance(event, dict):
                            visible = event.get('Output', '')
                    if stream:
                        print(visible, end='', flush=True)
                    if fail_fast and failure is None and isinstance(event, dict) and event.get('Action') == 'fail':
                        failure = event
                        # -failfast applies within a Go package; stop all remaining packages too.
                        try:
                            stop(proc)
                        finally:
                            # The leader can exit while helpers keep stdout open and ignore TERM.
                            try:
                                os.killpg(proc.pid, signal.SIGKILL)
                            except ProcessLookupError:
                                pass

        reader = readers.submit(copy_output)
        try:
            code = proc.wait(timeout=timeout)
        finally:
            stop(proc)
            reader.result()
    if failure is not None:
        name = failure.get('Test') or failure.get('Package', 'unknown test')
        raise RuntimeError(f'{log.name}: first test failure in {failure.get("Package", "")}: {name}; see artifact log')
    if code:
        raise RuntimeError(f'{log.name}: exit {code}; see artifact log')
