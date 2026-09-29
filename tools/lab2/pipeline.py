"""Run real tests in order; always restore the disposable stores and build Allure."""
import fcntl
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import sys
import time
import urllib.request
import uuid

OUT = Path('/artifacts')
RESULTS = OUT / 'allure-results'
RESULTS.mkdir(parents=True, exist_ok=True)
os.environ['ALLURE_RESULTS_DIR'] = str(RESULTS)
os.environ['GIN_MODE'] = 'release'
assert os.environ.get('LAB2_TEST_ENV') == '1', 'isolated test environment required'
DSN = os.environ['LAB2_DATABASE_URL']
REDIS = os.environ['LAB2_REDIS_URL']
TABLES = ['user', 'target', 'probe_result', 'monitor', 'monitor_status_log',
          'alert_contact', 'monitor_alert_contact', 'maintenance_window', 'maintenance_window_monitor']
stages = {name: {'status': 'skipped', 'reason': 'Previous stage did not succeed'}
          for name in ['unit', 'integration', 'e2e']}
api = capture = None
initialized = False


def stop(proc, sig=signal.SIGTERM):
    if proc is not None and proc.poll() is None:
        os.killpg(proc.pid, sig)
        try:
            proc.wait(timeout=10)
        except subprocess.TimeoutExpired:
            os.killpg(proc.pid, signal.SIGKILL)
            proc.wait(timeout=5)


def run(cmd, name):
    with (OUT / name).open('w') as output:
        proc = subprocess.Popen(cmd, stdout=output, stderr=subprocess.STDOUT, start_new_session=True)
        try:
            code = proc.wait(timeout=600)
        finally:
            stop(proc)
    if code:
        raise RuntimeError(f'{name}: exit {code}; see artifact log')


def sql(query):
    return subprocess.check_output(['psql', DSN, '-XAt', '-v', 'ON_ERROR_STOP=1', '-c', query], text=True, timeout=30).strip()


def counts():
    args = ', '.join(f"'{name}', (SELECT count(*) FROM \"{name}\")" for name in TABLES)
    return json.loads(sql(f'SELECT json_build_object({args})'))


def reset():
    sql('TRUNCATE ' + ', '.join(f'"{name}"' for name in TABLES) + ' RESTART IDENTITY CASCADE')
    subprocess.run(['redis-cli', '-u', REDIS, 'FLUSHDB'], check=True, capture_output=True, timeout=15)


def stage_result(name, status, reason):
    now = int(time.time() * 1000)
    # Pipeline stage markers supplement official Go adapter results, including stages never started.
    item = {'uuid': str(uuid.uuid4()), 'name': name, 'fullName': f'lab2.pipeline.{name}',
            'historyId': f'lab2.pipeline.{name}', 'testCaseId': f'lab2.pipeline.{name}',
            'status': status, 'stage': 'finished', 'start': now, 'stop': now,
            'statusDetails': {'message': reason},
            'labels': [{'name': 'parentSuite', 'value': 'CI pipeline'}, {'name': 'suite', 'value': name}]}
    (RESULTS / f"{item['uuid']}-result.json").write_text(json.dumps(item))


def start_api():
    global api
    # All credentials are synthetic and scoped to this disposable instance.
    db = {name: {'type': 'postgres', 'dsn': DSN} for name in
          ['monitoring', 'maintenance', 'auth', 'metrics', 'healthchecker', 'analyzer', 'contacts', 'migrations']}
    config = {'logging': {'level': 'warn', 'format': 'json', 'output': 'stdout'},
              'auth': {'jwt_secret': secrets.token_urlsafe(32), 'jwt_ttl_hours': 1},
              'database': db, 'redis': {'url': REDIS}, 'migrations_dir': 'migrations'}
    # JSON is a YAML subset accepted by the production config loader.
    Path('/tmp/lab2-api.json').write_text(json.dumps(config))
    api = subprocess.Popen(['watchtower-api', '/tmp/lab2-api.json'],
                           stdout=(OUT / 'api.log').open('w'), stderr=subprocess.STDOUT, start_new_session=True)
    deadline = time.monotonic() + 45
    while time.monotonic() < deadline:
        if api.poll() is not None:
            raise RuntimeError('API exited before readiness; see api.log')
        try:
            with urllib.request.urlopen(os.environ['LAB2_API_URL'] + '/openapi.yaml', timeout=1) as response:
                if response.status == 200:
                    return
        except OSError:
            time.sleep(0.2)
    raise RuntimeError('API readiness timeout')


def interrupt(signum, frame):
    raise InterruptedError(f'pipeline interrupted by signal {signum}')


signal.signal(signal.SIGTERM, interrupt)
signal.signal(signal.SIGINT, interrupt)
status = 0
active = 'unit'
try:
    for active in stages:
        stages[active] = {'status': 'running', 'reason': ''}
        print(f'=== {active} ===', flush=True)
        if active == 'unit':
            packages = Path('tools/lab1/packages.txt').read_text().split() + ['./internal/api/http/v1/...']
            run(['go', 'test', '-json', '-count=1', '-timeout=120s', *packages], 'unit.jsonl')
        elif active == 'integration':
            run(['lab2-migrate'], 'migrations.log')
            initialized = True
            reset()
            (OUT / 'baseline.json').write_text(json.dumps(counts(), indent=2))
            repeat = int(os.environ.get('LAB2_INTEGRATION_REPEAT', '1'))
            if repeat < 1:
                raise ValueError('LAB2_INTEGRATION_REPEAT must be positive')
            run(['go', 'test', '-json', '-tags=lab2integration', f'-count={repeat}', '-timeout=120s',
                 './tests/lab2/integration'], 'integration.jsonl')
        else:
            reset()
            start_api()
            capture = subprocess.Popen(['tcpdump', '-i', 'lo', '-U', '-p', '-w', str(OUT / 'traffic.pcap'),
                                        'tcp port 8080'], stdout=subprocess.DEVNULL,
                                       stderr=(OUT / 'tcpdump.log').open('w'), start_new_session=True)
            # Wait for tcpdump's explicit readiness, not a guessed startup sleep.
            deadline = time.monotonic() + 10
            while 'listening on' not in (OUT / 'tcpdump.log').read_text():
                if capture.poll() is not None or time.monotonic() >= deadline:
                    raise RuntimeError('tcpdump failed to start')
                time.sleep(0.05)
            run(['go', 'test', '-json', '-tags=lab2e2e', '-count=1', '-timeout=120s',
                 './tests/lab2/e2e'], 'e2e.jsonl')
            run(['python3', 'tools/lab2/replay.py'], 'curl-replay.log')
        if os.environ.get('LAB2_FAIL_STAGE') == active:
            if initialized:
                sql("INSERT INTO \"user\" VALUES ('rollback-canary', 'not-a-real-password')")
                subprocess.run(['redis-cli', '-u', REDIS, 'SET', 'rollback-canary', 'present'], check=True, capture_output=True)
            raise RuntimeError(f'Injected {active} failure to verify rollback and skipped stages')
        stages[active] = {'status': 'passed', 'reason': ''}
except BaseException as error:
    status = 1
    stages[active] = {'status': 'failed', 'reason': str(error)}
    print(str(error), file=sys.stderr, flush=True)
finally:
    # Ignore further interrupts while bounded cleanup and report generation run.
    signal.signal(signal.SIGINT, signal.SIG_IGN)
    signal.signal(signal.SIGTERM, signal.SIG_IGN)
    stop(api)
    stop(capture, signal.SIGINT)
    Path('/tmp/lab2-api.json').unlink(missing_ok=True)
    cleanup = {'api_stopped': api is None or api.poll() is not None,
               'sessions': 'per-run JWT signing key discarded with API process',
               'bus': 'in-process GoChannel disposed with API; integration consumers drain and ack'}
    try:
        if initialized:
            cleanup['before_reset'] = counts()
            reset()
            cleanup['after_reset'] = counts()
            cleanup['redis_keys'] = int(subprocess.check_output(['redis-cli', '-u', REDIS, 'DBSIZE'], text=True, timeout=15))
            cleanup['restored'] = all(n == 0 for n in cleanup['after_reset'].values()) and cleanup['redis_keys'] == 0
            if not cleanup['restored']:
                raise RuntimeError('test stores were not restored')
        else:
            cleanup['restored'] = True
            cleanup['note'] = 'application migrations/tests not started; disposable instance is removed by Compose'
    except Exception as error:
        status = 1
        cleanup.update(restored=False, error=str(error))
    (OUT / 'cleanup.json').write_text(json.dumps(cleanup, indent=2))
    stage_result('cleanup', 'passed' if cleanup['restored'] else 'failed', cleanup.get('error', ''))
    for name, result in stages.items():
        stage_result(name, result['status'], result['reason'])
    if (OUT / 'traffic.pcap').exists():
        try:
            run(['tcpdump', '-nn', '-A', '-r', str(OUT / 'traffic.pcap')], 'traffic.txt')
        except Exception as error:
            status = 1
            print(error, file=sys.stderr)
    try:
        # The file lock spans reading history, report generation and appending this launch.
        with open('/history/.lock', 'w') as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            run(['tools/lab1/node_modules/.bin/allure', 'generate', str(RESULTS),
                 '--config', 'tools/lab2/allurerc.mjs', '--output', str(OUT / 'allure-report')], 'allure.log')
    except Exception as error:
        status = 1
        print(error, file=sys.stderr)
    (OUT / 'pipeline.json').write_text(json.dumps({'run': os.environ['LAB2_RUN_ID'], 'exit_code': status,
                                                  'stages': stages, 'cleanup': cleanup}, indent=2))
sys.exit(status)
