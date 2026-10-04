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


def run(cmd, log, timeout=600, stream=True):
    with log.open('wb') as output, ThreadPoolExecutor(max_workers=1) as readers:
        proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                start_new_session=True)

        def copy_output():
            with proc.stdout:
                for line in proc.stdout:
                    output.write(line)
                    output.flush()
                    if stream:
                        visible = line.decode('utf-8', errors='replace')
                        if log.suffix == '.jsonl':
                            try:
                                event = json.loads(visible)
                            except json.JSONDecodeError:
                                pass  # Preserve compiler/runtime diagnostics that are not JSON.
                            else:
                                visible = event.get('Output', '') if isinstance(event, dict) else visible
                        print(visible, end='', flush=True)

        reader = readers.submit(copy_output)
        try:
            code = proc.wait(timeout=timeout)
        finally:
            stop(proc)
            reader.result()
    if code:
        raise RuntimeError(f'{log.name}: exit {code}; see artifact log')
