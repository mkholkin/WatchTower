"""Five isolated load experiments; live logs, bounded commands and unconditional cleanup."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))
from tools.lab2.command import run  # noqa: E402

PROFILES = ['ramp', 'stress', 'soak-0', 'soak-10', 'soak-20']


def capture(command):
    return subprocess.check_output(command, text=True, timeout=120).strip()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--smoke', action='store_true')
    parser.add_argument('--profile', choices=PROFILES, action='append')
    args = parser.parse_args()
    os.chdir(ROOT)
    if int(capture(['docker', 'info', '--format', '{{.NCPU}}'])) < 4:
        raise RuntimeError('Lab 4 requires at least 4 Docker CPU cores (app:0, DB:1, k6:2, collector:3)')
    run_id = datetime.now(timezone.utc).strftime('%Y%m%dt%H%M%S')+'-'+secrets.token_hex(3)
    output = ROOT/'build/lab4/runs'/run_id
    output.mkdir(parents=True)
    (ROOT/'build/lab4/latest.txt').write_text(str(output)+'\n')
    manifest = {'run_id': run_id, 'smoke': args.smoke, 'git_commit': capture(['git', 'rev-parse', 'HEAD']),
                'working_tree': capture(['git', 'status', '--short']),
                'docker': json.loads(capture(['docker', 'info', '--format', '{{json .}}'])), 'profiles': []}
    manifest['source_sha256'] = {str(path.relative_to(ROOT)): hashlib.sha256(path.read_bytes()).hexdigest()
                                 for folder in ('tests/lab4', 'tools/lab4') for path in sorted((ROOT/folder).glob('*'))
                                 if path.is_file()}
    # Unique base tags avoid interference with other concurrent runs.
    base, app, postgres = (f'watchtower-lab4-{kind}:{run_id}' for kind in ('base', 'app', 'postgres'))
    run(['docker', 'build', '-f', 'tools/lab2/Dockerfile', '-t', base, '.'], output/'build-base.log', 1200)
    run(['docker', 'build', '-f', 'tools/lab4/Dockerfile', '--build-arg', f'BASE={base}', '-t', app, '.'], output/'build-report.log', 900)
    run(['docker', 'build', '-f', 'Dockerfile.postgres', '-t', postgres, '.'], output/'build-postgres.log', 900)
    run(['docker', 'pull', 'grafana/k6:1.3.0'], output/'pull-k6.log', 600)
    run(['docker', 'pull', 'redis:7.4-bookworm'], output/'pull-redis.log', 600)
    failed = False
    for name in args.profile or PROFILES:
        directory = output/'profiles'/name
        directory.mkdir(parents=True)
        image = f'watchtower-lab4-run:{run_id}-{name}'
        dockerfile = directory/'Dockerfile.run'
        dockerfile.write_text(f'FROM {app}\nLABEL org.watchtower.lab4.run="{run_id}-{name}"\n')
        run(['docker', 'build', '--network=none', '-f', str(dockerfile), '-t', image, str(directory)], directory/'build.log')
        db = {key: {'type': 'postgres', 'dsn': 'postgres://postgres:lab4-only@postgres:5432/postgres?sslmode=disable'}
              for key in ('monitoring', 'maintenance', 'auth', 'metrics', 'healthchecker', 'analyzer', 'contacts', 'migrations')}
        config = directory/'config.json'
        config.write_text(json.dumps({'logging': {'level': 'warn', 'format': 'json', 'output': 'stdout'},
                                     'auth': {'jwt_secret': secrets.token_hex(32), 'jwt_ttl_hours': 1},
                                     'database': db, 'redis': {'url': 'redis://redis:6379/0'}, 'migrations_dir': 'migrations'})+'\n')
        os.environ.update(LAB4_APP_IMAGE=image, LAB4_POSTGRES_IMAGE=postgres, LAB4_CONFIG=str(config),
                          LAB4_OUTPUT=str(directory), LAB4_UID=str(os.getuid()), LAB4_GID=str(os.getgid()),
                          LAB4_PROFILE=name, LAB4_SMOKE='1' if args.smoke else '0', LAB4_CONTAINERS='{}')
        compose = ['docker', 'compose', '-p', f'wt-lab4-{run_id}-{name}', '-f', 'tools/lab4/compose.yaml']
        record = {'name': name, 'image': image, 'image_id': capture(['docker', 'image', 'inspect', image, '--format', '{{.Id}}']),
                  'exit_code': None}
        print(f'\n=== Lab 4: {name} ({"smoke" if args.smoke else "full"}) ===', flush=True)
        try:
            run(compose+['up', '-d', '--wait', '--wait-timeout', '180'], directory/'start.log', 240)
            run(compose+['--profile', 'load', 'create', 'load'], directory/'create-load.log')
            ids = {service: capture(compose+['ps', '-aq', service]) for service in ('api', 'postgres', 'redis', 'probe', 'load')}
            os.environ['LAB4_CONTAINERS'] = json.dumps(ids)
            run(compose+['--profile', 'monitor', 'up', '-d', 'collector'], directory/'collector.log')
            ids['collector'] = capture(compose+['ps', '-aq', 'collector'])
            record['containers'] = json.loads(capture(['docker', 'inspect', *ids.values()]))
            (directory/'inspect.json').write_text(json.dumps(record['containers'], indent=2)+'\n')
            started = time.monotonic()
            run(['docker', 'start', '-a', ids['load']], directory/'k6.log', 700)
            record['elapsed_s'] = time.monotonic()-started
            record['exit_code'] = int(capture(['docker', 'inspect', ids['load'], '--format', '{{.State.ExitCode}}']))
            if record['exit_code']:
                raise RuntimeError(f'k6 exited {record["exit_code"]}')
        except Exception as error:
            record['error'] = str(error)
            failed = True
            print(f'{name}: {error}', flush=True)
        finally:
            try:
                try:
                    run(compose+['stop', 'collector'], directory/'stop-collector.log', 60)
                    run(compose+['logs', '--no-color', '--timestamps'], directory/'services.log', 60, stream=False)
                    record['final_state'] = capture(compose+['ps', '-a'])
                    container_ids = capture(compose+['ps', '-aq']).splitlines()
                    record['final_containers'] = json.loads(capture(['docker', 'inspect', *container_ids])) if container_ids else []
                finally:
                    run(compose+['--profile', 'load', '--profile', 'monitor', 'down', '-v', '--remove-orphans'], directory/'cleanup.log', 120)
            except Exception as error:
                record['error'] = f'Cleanup/diagnostics: {error}'
                raise
            finally:
                config.unlink(missing_ok=True)
                manifest['profiles'].append(record)
                (output/'manifest.json').write_text(json.dumps(manifest, indent=2)+'\n')
    generate_reports(app, output)
    print(f'Lab 4 artifacts: {output}', flush=True)
    if failed:
        raise RuntimeError('At least one load profile failed; see manifest.json')


def generate_reports(app, output):
    # Report tools and results are mounted separately: the image is identical across profiles.
    report_cmd = ['docker', 'run', '--rm', '--network=none', '--cpuset-cpus=3', '--cpus=1', '--memory=1g',
                  '--user', f'{os.getuid()}:{os.getgid()}', '-e', 'MPLCONFIGDIR=/tmp/matplotlib',
                  '-v', f'{ROOT}:/source:ro', '-v', f'{output}:/artifacts', '-w', '/source', app,
                  'tools/lab4/report.py', '/artifacts']
    try:
        run(report_cmd, output/'report.log', 300)
    finally:
        run(['docker', 'run', '--rm', '--network=none', '--cpuset-cpus=3', '--memory=1g',
             '--user', f'{os.getuid()}:{os.getgid()}', '-v', f'{output}:/artifacts', '-v', f'{ROOT}/tools/lab4:/lab4:ro',
             '--entrypoint', 'node', app, '/workspace/tools/lab1/node_modules/allure/cli.js',
             'generate', '/artifacts/allure-results', '--config', '/lab4/allurerc.mjs',
             '-o', '/artifacts/allure-report'], output/'allure.log', 300)


def entrypoint():
    def interrupted(signum, _frame):
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
        raise KeyboardInterrupt(f'Terminated by signal {signum}')

    signal.signal(signal.SIGTERM, interrupted)
    keeper = subprocess.Popen(['caffeinate', '-i', '-w', str(os.getpid())]) if sys.platform == 'darwin' else None
    try:
        main()
    finally:
        if keeper is not None:
            keeper.terminate()
            keeper.wait(timeout=5)


if __name__ == '__main__':
    entrypoint()
