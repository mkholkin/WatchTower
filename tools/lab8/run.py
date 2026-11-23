"""Run isolated RDAP E2E, demonstration and fixed Lab 4 load contours."""
import argparse
import hashlib
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import secrets
import shutil
import sys
import time

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))
from tools.lab4 import run as lab4  # noqa: E402
from tools.lab6.run import ensure_image  # noqa: E402
from tools.lab8.report import generate as generate_report  # noqa: E402

run = lab4.run
capture = lab4.capture
DEMO = ROOT / 'build/lab8/demo.json'


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + '\n')


def configuration(mode):
    dsn = 'postgres://postgres:lab4-only@postgres:5432/postgres?sslmode=disable'
    return {'logging': {'level': 'warn', 'format': 'json', 'output': 'stdout'},
            'auth': {'jwt_secret': secrets.token_hex(32), 'jwt_ttl_hours': 1},
            'database': {name: {'type': 'postgres', 'dsn': dsn} for name in (
                'monitoring', 'maintenance', 'auth', 'metrics', 'healthchecker', 'analyzer', 'contacts', 'migrations')},
            'redis': {'url': 'redis://redis:6379/0'}, 'migrations_dir': 'migrations',
            'rdap': {'enabled': True, 'base_url': 'http://rdap-mock:8089/com/v1' if mode == 'mock'
                     else 'https://rdap.verisign.com/com/v1', 'timeout_ms': 750 if mode == 'mock' else 10000}}


def compose_command(project):
    return ['docker', 'compose', '-p', project, '-f', 'tools/lab8/compose.yaml']


def environment(directory, app, postgres, config, mode, smoke=False, demo=False):
    return {'LAB4_APP_IMAGE': app, 'LAB4_POSTGRES_IMAGE': postgres, 'LAB4_CONFIG': str(config),
            'LAB4_OUTPUT': str(directory), 'LAB4_UID': str(os.getuid()), 'LAB4_GID': str(os.getgid()),
            'LAB4_PROFILE': 'soak-0', 'LAB4_SMOKE': '1' if smoke else '0', 'LAB4_CONTAINERS': '{}',
            'LAB8_MODE': mode, 'LAB8_DEMO': '1' if demo else '0', 'LAB8_API_PORT': '3088' if demo else '0', 'LAB8_MOCK_PORT': '8089' if demo else '0'}


def mock_evidence(compose, directory):
    value = capture(compose + ['exec', '-T', 'rdap-mock', 'curl', '--fail', '--silent',
                               'http://localhost:8089/__admin/stats'])
    write_json(directory / 'rdap-mock-stats.json', json.loads(value))
    identifier = capture(compose + ['ps', '-aq', 'rdap-mock'])
    write_json(directory / 'rdap-mock-inspect.json', json.loads(capture(['docker', 'inspect', identifier])))


def cleanup(compose, directory, record):
    try:
        run(compose + ['logs', '--no-color', '--timestamps'], directory / 'services.log', 60, stream=False)
    except Exception as error:
        record['diagnostics_error'] = str(error)
    try:
        run(compose + ['--profile', 'tests', '--profile', 'load', '--profile', 'monitor',
                       'down', '-v', '--remove-orphans'], directory / 'cleanup.log', 120)
        record['cleanup'] = True
    except Exception as error:
        record['cleanup'] = False
        record['cleanup_error'] = str(error)


def fixed_load(compose, directory, record):
    run(compose + ['--profile', 'load', 'create', 'load'], directory / 'create-load.log')
    ids = {name: capture(compose + ['ps', '-aq', name]) for name in ('api', 'postgres', 'redis', 'probe', 'load')}
    os.environ['LAB4_CONTAINERS'] = json.dumps(ids)
    run(compose + ['--profile', 'monitor', 'up', '-d', 'collector'], directory / 'collector.log')
    ids['collector'] = capture(compose + ['ps', '-aq', 'collector'])
    record['containers'] = json.loads(capture(['docker', 'inspect', *ids.values()]))
    write_json(directory / 'inspect.json', record['containers'])
    started = time.monotonic()
    run(['docker', 'start', '-a', ids['load']], directory / 'k6.log', 700)
    record['elapsed_s'] = time.monotonic() - started
    time.sleep(3)
    run(compose + ['stop', 'collector'], directory / 'stop-collector.log', 60)
    record['exit_code'] = int(capture(['docker', 'inspect', ids['load'], '--format', '{{.State.ExitCode}}']))
    if record['exit_code']:
        raise RuntimeError('Load container failed')


def contour(output, run_id, kind, mode, app, postgres, smoke=False):
    directory = output / ('load/profiles/soak-0' if kind == 'load' else f'{kind}-{mode}')
    directory.mkdir(parents=True)
    project = f'wt-lab8-{run_id}-{kind}-{mode}'
    compose = compose_command(project)
    config = directory / 'config.json'
    write_json(config, configuration(mode))
    config.chmod(0o600)
    env = environment(directory, app, postgres, config, mode, smoke, kind == 'demo')
    os.environ.update(env)
    record = {'kind': kind, 'mode': mode, 'project': project, 'exit_code': None, 'cleanup': False}
    keep = False
    try:
        run(compose + ['up', '-d', '--wait', '--wait-timeout', '180'], directory / 'start.log', 240)
        if kind == 'load':
            record.update(name='soak-0', image=app,
                          image_id=capture(['docker', 'image', 'inspect', app, '--format', '{{.Id}}']))
            fixed_load(compose, directory, record)
        else:
            run(compose + ['--profile', 'tests', 'run', '--rm', '--no-deps', 'tests'], directory / 'e2e.log', 300)
            record['exit_code'] = 0
        mock_evidence(compose, directory)
        if kind == 'demo':
            write_json(DEMO, {'project': project, 'environment': env, 'directory': str(directory)})
            keep = True
            print('Lab 8 demo: http://localhost:3088/swagger ; sample: python3 tools/lab8/demo_request.py', flush=True)
    except BaseException as error:
        record['error'] = f'{type(error).__name__}: {error}'
        write_json(directory / 'allure-results/infrastructure-result.json', {
            'uuid': secrets.token_hex(16), 'name': f'{kind}/{mode} infrastructure', 'status': 'broken',
            'stage': 'finished', 'statusDetails': {'message': record['error']},
            'labels': [{'name': 'parentSuite', 'value': 'Lab 8'}]})
        if not isinstance(error, Exception):
            raise
    finally:
        if not keep:
            cleanup(compose, directory, record)
            config.unlink(missing_ok=True)
            (directory/'demo-session.json').unlink(missing_ok=True)
        write_json(directory / 'experiment.json', record)
    return record


def down():
    if not DEMO.exists():
        print('No Lab 8 demo is running.', flush=True)
        return
    state = json.loads(DEMO.read_text())
    os.environ.update(state['environment'])
    record = {}
    cleanup(compose_command(state['project']), Path(state['directory']), record)
    if not record['cleanup']:
        raise RuntimeError('Demo cleanup failed')
    Path(state['environment']['LAB4_CONFIG']).unlink(missing_ok=True)
    (Path(state['directory'])/'demo-session.json').unlink(missing_ok=True)
    DEMO.unlink()


def safe_context(directory):
    directory.mkdir()
    paths = ['go.mod', 'go.sum', 'cmd', 'internal', 'pkg', 'configs', 'migrations', 'api',
             'tools/lab1', 'tools/lab2', 'tools/lab4', 'tools/lab8', 'tests/lab4']
    ignore = shutil.ignore_patterns('node_modules', '__pycache__', '.cache', 'build', 'allure-results',
                                    '.env', '.env.*', '*.pem', '*.key')
    for name in paths:
        source, target = ROOT / name, directory / name
        if not source.exists():
            continue
        target.parent.mkdir(parents=True, exist_ok=True)
        if source.is_dir():
            shutil.copytree(source, target, ignore=ignore)
        else:
            shutil.copyfile(source, target)
    return directory


def validate_load(directory, record, smoke):
    stats = json.loads((directory / 'profiles/soak-0/rdap-mock-stats.json').read_text())
    summary = json.loads((directory / 'summary.json').read_text())['profiles'][0]
    operation = summary['operations'].get('domain-registration', {})
    calls = stats.get('requests', {}).get('example.com', {}).get('GET', 0)
    if summary['unexpected_requests'] or not operation.get('requests') or calls < operation.get('requests', 0):
        raise RuntimeError('RDAP load requires successful measured registration requests and upstream calls')
    record['rdap_evidence'] = {'upstream_requests': calls, 'measured_requests': operation['requests'],
                               'full_load': not smoke}


def build_images(args, output, run_id):
    app = f'watchtower-lab8-app:{run_id}'
    context = safe_context(output / 'source-context')
    write_json(output / 'source-sha256.json', {str(path.relative_to(context)): hashlib.sha256(path.read_bytes()).hexdigest()
               for path in sorted(context.rglob('*')) if path.is_file()})
    runtime = args.runtime_base
    if runtime:
        capture(['docker', 'image', 'inspect', runtime, '--format', '{{.Id}}'])
    else:
        base, runtime = (f'watchtower-lab8-{name}:{run_id}' for name in ('base', 'runtime'))
        run(['docker', 'build', '-f', 'tools/lab2/Dockerfile', '-t', base, str(context)], output / 'build-base.log', 1500)
        run(['docker', 'build', '-f', 'tools/lab4/Dockerfile', '--build-arg', f'BASE={base}', '-t', runtime, str(context)],
            output / 'build-runtime.log', 900)
    run(['docker', 'build', '--network=none', '-f', 'tools/lab8/Dockerfile', '--build-arg', f'BASE={runtime}',
         '-t', app, str(context)], output / 'build-app.log', 1500)
    postgres = args.postgres_image or f'watchtower-lab8-postgres:{run_id}'
    if args.postgres_image:
        capture(['docker', 'image', 'inspect', postgres, '--format', '{{.Id}}'])
    else:
        postgres_context = output / 'postgres-context'
        postgres_context.mkdir()
        run(['docker', 'build', '-f', str(ROOT / 'Dockerfile.postgres'), '-t', postgres, str(postgres_context)],
            output / 'build-postgres.log', 900)
    ensure_image('redis:7.4-bookworm', output)
    if args.action in ('all', 'load'):
        ensure_image('grafana/k6:1.3.0', output)
    return app, postgres


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('all', 'e2e', 'load', 'demo', 'down'), nargs='?', default='all')
    parser.add_argument('--mode', choices=('mock', 'real'), default='mock')
    parser.add_argument('--smoke', action='store_true')
    parser.add_argument('--runtime-base')
    parser.add_argument('--postgres-image')
    args = parser.parse_args()
    os.chdir(ROOT)
    if args.action == 'down':
        down()
        return
    if args.action == 'demo' and DEMO.exists():
        raise RuntimeError('Run tools/lab8/run.py down before starting another demo')
    if args.action in ('all', 'load') and int(capture(['docker', 'info', '--format', '{{.NCPU}}'])) < 4:
        raise RuntimeError('Lab 8 load requires at least 4 Docker CPU cores')
    run_id = datetime.now(timezone.utc).strftime('%Y%m%dt%H%M%S') + '-' + secrets.token_hex(3)
    output = ROOT / 'build/lab8/runs' / run_id
    output.mkdir(parents=True)
    (ROOT / 'build/lab8/latest.txt').write_text(str(output) + '\n')
    manifest = {'run_id': run_id, 'smoke': args.smoke, 'action': args.action, 'mode': args.mode, 'contours': []}
    app = None
    try:
        app, postgres = build_images(args, output, run_id)
        manifest['images'] = {name: {'image': image, 'id': capture(['docker', 'image', 'inspect', image, '--format', '{{.Id}}'])}
                              for name, image in [('app', app), ('postgres', postgres)]}
        tasks = [('e2e', 'mock'), ('load', 'mock'), ('e2e', 'real')] if args.action == 'all' else [(args.action, 'mock' if args.action == 'load' else args.mode)]
        for kind, mode in tasks:
            record = contour(output, run_id, kind, mode, app, postgres, args.smoke)
            manifest['contours'].append(record)
            write_json(output / 'manifest.json', manifest)
            if kind == 'load':
                load = output / 'load'
                write_json(load / 'manifest.json', {'profiles': [record], 'run_id': run_id, 'smoke': args.smoke})
                lab4.generate_reports(app, load)
                validate_load(load, record, args.smoke)
            if record.get('error') or record['exit_code'] != 0 or (kind != 'demo' and not record['cleanup']):
                raise RuntimeError(f'{kind}/{mode} failed; see experiment.json')
    except BaseException as error:
        manifest['error'] = f'{type(error).__name__}: {error}'
        raise
    finally:
        recorded = {item['project'] for item in manifest['contours']}
        for path in output.glob('**/experiment.json'):
            record = json.loads(path.read_text())
            if record['project'] not in recorded:
                manifest['contours'].append(record)
        write_json(output / 'manifest.json', manifest)
        try:
            generate_report(output, manifest, app)
        except Exception as error:
            manifest['report_error'] = str(error)
            write_json(output / 'manifest.json', manifest)
            if 'error' not in manifest:
                raise
        print(f'Lab 8 artifacts: {output}', flush=True)


if __name__ == '__main__':
    lab4.main = main
    lab4.entrypoint()
