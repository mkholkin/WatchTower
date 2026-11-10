"""Isolated BDD stand: private runtime secrets, safe logs and unconditional cleanup."""
from datetime import datetime, timezone
from collections import Counter
import html
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import sys
import tempfile
import time
import uuid

from progress import poll_progress

_PROGRESS = {}

ROOT = Path(__file__).resolve().parents[2]


def command(argv, env, timeout):
    # Service/Behave exceptions can include credentials. Never publish raw output,
    # command lines, Docker inspect/config output or process environments.
    build_log = None
    if argv[:2] == ['docker', 'build']:
        # Build commands run before runtime secrets exist and accept no secret args.
        name = 'postgres' if 'Dockerfile.postgres' in argv else 'app'
        build_log = (Path(env['LAB5_OUTPUT']) / f'build-{name}.log').open('w')
    try:
        with subprocess.Popen(argv, cwd=ROOT, env=env, stdout=build_log or subprocess.DEVNULL,
                              stderr=subprocess.STDOUT, start_new_session=True) as process:
            try:
                deadline = time.monotonic() + timeout
                heartbeat = time.monotonic() + 30
                positions = _PROGRESS.setdefault(env["LAB5_OUTPUT"], {})
                while True:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0:
                        raise subprocess.TimeoutExpired('private command', timeout)
                    try:
                        code = process.wait(timeout=min(1, remaining))
                        poll_progress(env["LAB5_OUTPUT"], positions)
                        if 'run' in argv and 'tests' in argv and '--domain-tests' not in argv:
                            counts = Counter(positions.get('_bdd_results', {}).values())
                            print('BDD summary: '+', '.join(f'{counts[status]} {status}' for status in
                                  ('passed', 'failed', 'skipped', 'undefined', 'untested')), flush=True)
                        return code
                    except subprocess.TimeoutExpired:
                        if time.monotonic() >= deadline:
                            raise
                        poll_progress(env["LAB5_OUTPUT"], positions)
                        if time.monotonic() >= heartbeat:
                            print('Lab 5: command still running; raw output withheld.', flush=True)
                            heartbeat = time.monotonic() + 30
            except (subprocess.TimeoutExpired, KeyboardInterrupt):
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
                raise
    except subprocess.TimeoutExpired:
        return 124
    except OSError:
        return 127
    finally:
        if build_log:
            build_log.close()


def private_file(path, content):
    with os.fdopen(os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'w') as stream:
        stream.write(content + '\n')


def prepare_secrets(directory):
    password = secrets.token_hex(32)
    dsn = f'postgres://postgres:{password}@postgres:5432/postgres?sslmode=disable'
    config = {
        'logging': {'level': 'warn', 'format': 'json', 'output': 'stdout'},
        'auth': {'jwt_secret': secrets.token_hex(32), 'jwt_ttl_hours': 1,
                 'mfa': {'enabled': True, 'smtp_address': 'mailpit:1025',
                         'smtp_from': 'watchtower@example.test', 'allow_insecure_smtp': True,
                         'max_attempts': 3, 'challenge_ttl_seconds': 300,
                         'password_max_age_hours': 2160}},
        'database': {name: {'type': 'postgres', 'dsn': dsn} for name in (
            'monitoring', 'maintenance', 'auth', 'metrics', 'healthchecker',
            'analyzer', 'contacts', 'migrations')},
        'redis': {'url': 'redis://redis:6379/0'}, 'migrations_dir': 'migrations',
    }
    private_file(directory / 'database_password', password)
    private_file(directory / 'database_url', dsn)
    # JSON is valid YAML and avoids an additional host dependency.
    private_file(directory / 'api_config', json.dumps(config))


def broken_result(output, name):
    results = output / 'allure-results'
    results.mkdir(exist_ok=True)
    identifier = str(uuid.uuid4())
    now = int(time.time() * 1000)
    result = {'uuid': identifier, 'name': name, 'fullName': f'lab5.infrastructure.{name}',
              'status': 'broken', 'stage': 'finished', 'start': now, 'stop': now,
              'statusDetails': {'message': 'Infrastructure command failed; see safe runner.log.'},
              'labels': [{'name': 'suite', 'value': 'Lab 5 infrastructure'}]}
    (results / f'{identifier}-result.json').write_text(json.dumps(result) + '\n')


def fallback_report(output):
    """Keep an honest readable summary if Docker/report dependencies never built."""
    directory = output / 'allure-report'
    directory.mkdir(exist_ok=True)
    items = []
    for path in sorted((output / 'allure-results').glob('*-result.json')):
        result = json.loads(path.read_text())
        items.append(f'<li>{html.escape(result["name"])}: {html.escape(result["status"])}</li>')
    (directory / 'index.html').write_text(
        '<!doctype html><meta charset="utf-8"><title>Lab 5 preparation failure</title>'
        '<h1>Lab 5 infrastructure report</h1><p>The Allure generator was unavailable. '
        'Raw Allure results are retained for regeneration.</p><ul>' + ''.join(items) + '</ul>\n')


def execute(output, run_id, invoke=command, private_parent=None):
    output.mkdir(parents=True, exist_ok=True)
    (output / 'allure-results').mkdir(exist_ok=True)
    env = os.environ.copy()
    app = f'watchtower-lab5-app:{run_id}'
    postgres = f'watchtower-lab5-postgres:{run_id}'
    project = f'wt-lab5-{run_id}'
    compose = ['docker', 'compose', '-p', project, '-f', 'tools/lab5/compose.yaml']
    manifest = {'run_id': run_id, 'project': project, 'phases': [], 'exit_code': None}
    code, app_ready = 0, False

    def phase(name, argv, timeout=120):
        print(f'Lab 5: {name}...', flush=True)
        started = time.monotonic()
        status = invoke(argv, env, timeout)
        record = {'phase': name, 'exit_code': status, 'elapsed_s': round(time.monotonic() - started, 2)}
        manifest['phases'].append(record)
        with (output / 'runner.log').open('a') as stream:
            stream.write(json.dumps(record) + '\n')
        print(f'Lab 5: {name}: exit {status}', flush=True)
        return status

    # Keep secrets in the mounted workspace for Colima, outside published runs.
    private_parent = private_parent or ROOT / 'build/lab5/private'
    private_parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    private_parent.chmod(0o700)
    with tempfile.TemporaryDirectory(prefix='watchtower-lab5-', dir=private_parent) as temporary:
        env.update(LAB5_APP_IMAGE=app, LAB5_POSTGRES_IMAGE=postgres,
                   LAB5_OUTPUT=str(output.resolve()), LAB5_SECRET_DIR=temporary,
                   LAB5_UID=str(os.getuid()), LAB5_GID=str(os.getgid()))
        try:
            for name, argv, timeout in [
                ('build BDD and API', ['docker', 'build', '-f', 'tools/lab5/Dockerfile', '-t', app, '.'], 1200),
                ('build PostgreSQL', ['docker', 'build', '-f', 'Dockerfile.postgres', '-t', postgres, '.'], 900),
                ('start isolated services', compose + ['up', '-d', '--wait', '--wait-timeout', '180'], 240),
            ]:
                if name == 'start isolated services':
                    prepare_secrets(Path(temporary))
                code = phase(name, argv, timeout)
                if code:
                    broken_result(output, name)
                    break
                if name == 'build BDD and API':
                    app_ready = True
            if code == 0:
                domain_code = phase('PostgreSQL auth regressions',
                                    compose + ['run', '--rm', '--no-deps', 'tests', '--domain-tests'], 900)
                bdd_code = phase('BDD scenarios', compose + ['run', '--rm', '--no-deps', 'tests'], 900)
                code = domain_code or bdd_code
                if not list((output / 'allure-results').glob('*-result.json')):
                    broken_result(output, 'BDD runner produced no results')
                    code = code or 1
                if domain_code:
                    broken_result(output, 'PostgreSQL auth regressions failed')
        except KeyboardInterrupt:
            code = 130
            broken_result(output, 'Interrupted run')
        except Exception:
            # Do not serialize arbitrary exceptions: they can embed DSNs or payloads.
            code = 1
            broken_result(output, 'Runner preparation failed')
        finally:
            try:
                cleanup = phase('remove containers networks and volumes',
                                compose + ['--profile', 'tests', 'down', '-v', '--remove-orphans'], 120)
                if cleanup:
                    broken_result(output, 'Cleanup failed')
                    code = code or cleanup
            except Exception:
                broken_result(output, 'Cleanup command failed')
                code = code or 1
            finally:
                # Delete secrets before running any artifact publication tooling.
                for path in Path(temporary).iterdir():
                    path.unlink()
            report_code = 1
            if app_ready:
                report_container = f'{project}-report'
                try:
                    report_code = phase('generate Allure HTML', [
                        'docker', 'run', '--name', report_container, '--network=none',
                        '--user', f'{os.getuid()}:{os.getgid()}',
                        '-v', f'{output.resolve()}:/artifacts', '--entrypoint', 'node', app,
                        '/workspace/tools/lab1/node_modules/allure/cli.js', 'generate',
                        '/artifacts/allure-results', '--config', '/workspace/tools/lab5/allurerc.mjs',
                        '-o', '/artifacts/allure-report',
                    ], 300)
                except (Exception, KeyboardInterrupt):
                    report_code = 1
                finally:
                    # A timed-out docker client does not stop its container.
                    # Removal is also attempted if creation failed (then exit 1 is harmless).
                    phase('remove report container', ['docker', 'rm', '-f', report_container], 60)
            if report_code:
                broken_result(output, 'Allure generation unavailable')
                fallback_report(output)
                code = code or report_code
            manifest['exit_code'] = code
            (output / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    return code


def main():
    def interrupted(_signum, _frame):
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
        raise KeyboardInterrupt

    signal.signal(signal.SIGTERM, interrupted)
    run_id = datetime.now(timezone.utc).strftime('%Y%m%dt%H%M%S') + '-' + secrets.token_hex(3)
    output = ROOT / 'build/lab5/runs' / run_id
    output.mkdir(parents=True)
    (ROOT / 'build/lab5/latest.txt').write_text(str(output) + '\n')
    code = execute(output, run_id)
    print(f'Lab 5 artifacts: {output}', flush=True)
    return code


if __name__ == '__main__':
    sys.exit(main())
