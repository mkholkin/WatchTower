"""Repeat unchanged Lab 4 workloads in a controlled tracing/logging matrix."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import secrets
import sys
import time

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))
from tools.lab4 import run as lab4  # noqa: E402

run = lab4.run
capture = lab4.capture
VARIANTS = {
    'baseline': {'tracing': False, 'extended_logs': False},
    'trace': {'tracing': True, 'extended_logs': False},
    'debug': {'tracing': False, 'extended_logs': True},
    'trace-debug': {'tracing': True, 'extended_logs': True},
}


def save_manifest(output, manifest):
    (output/'manifest.json').write_text(json.dumps(manifest, indent=2)+'\n')


def configuration(extended_logs):
    databases = {key: {'type': 'postgres', 'dsn': 'postgres://postgres:lab4-only@postgres:5432/postgres?sslmode=disable'}
                 for key in ('monitoring', 'maintenance', 'auth', 'metrics', 'healthchecker', 'analyzer', 'contacts', 'migrations')}
    return {'logging': {'level': 'debug' if extended_logs else 'info', 'format': 'json', 'output': 'stdout'},
            'auth': {'jwt_secret': secrets.token_hex(32), 'jwt_ttl_hours': 1}, 'database': databases,
            'redis': {'url': 'redis://redis:6379/0'}, 'migrations_dir': 'migrations'}


def experiment(output, run_id, name, variant, repetition, app, postgres, smoke):
    directory = output/'variants'/variant/'repetitions'/str(repetition)/'profiles'/name
    directory.mkdir(parents=True)
    image = f'watchtower-lab6-run:{run_id}-{variant}-{repetition}-{name}'
    dockerfile = directory/'Dockerfile.run'
    dockerfile.write_text(f'FROM {app}\nLABEL org.watchtower.lab6.run="{run_id}-{variant}-{repetition}-{name}"\n')
    (directory/'.dockerignore').write_text('*\n')
    config = directory/'config.json'
    options = VARIANTS[variant]
    config.write_text(json.dumps(configuration(options['extended_logs']))+'\n')
    config.chmod(0o600)
    os.environ.update(LAB4_APP_IMAGE=image, LAB4_POSTGRES_IMAGE=postgres, LAB4_CONFIG=str(config),
                      LAB4_OUTPUT=str(directory), LAB4_UID=str(os.getuid()), LAB4_GID=str(os.getgid()),
                      LAB4_PROFILE=name, LAB4_SMOKE='1' if smoke else '0', LAB4_CONTAINERS='{}',
                      LAB6_TRACE_FILE='/artifacts/traces.jsonl' if options['tracing'] else '',
                      LAB6_HTTP_DEBUG='true' if options['extended_logs'] else 'false')
    compose = ['docker', 'compose', '-p', f'wt-lab6-{run_id}-{variant}-{repetition}-{name}',
               '-f', 'tools/lab6/compose.yaml']
    record = {'name': name, 'variant': variant, 'repetition': repetition, 'path': str(directory.relative_to(output)),
              'image': image, 'exit_code': None, 'cleanup': {'success': False}}
    print(f'\n=== Lab 6: {variant}, {name}, repetition {repetition} ===', flush=True)
    try:
        run(['docker', 'build', '--network=none', '-f', str(dockerfile), '-t', image, str(directory)], directory/'build.log', 120)
        record['image_id'] = capture(['docker', 'image', 'inspect', image, '--format', '{{.Id}}'])
        run(compose+['up', '-d', '--wait', '--wait-timeout', '180'], directory/'start.log', 240)
        run(compose+['--profile', 'load', 'create', 'load'], directory/'create-load.log')
        ids = {service: capture(compose+['ps', '-aq', service]) for service in ('api', 'postgres', 'redis', 'probe', 'load')}
        os.environ['LAB4_CONTAINERS'] = json.dumps(ids)
        run(compose+['--profile', 'monitor', 'up', '-d', 'collector'], directory/'collector.log')
        ids['collector'] = capture(compose+['ps', '-aq', 'collector'])
        record['containers'] = json.loads(capture(['docker', 'inspect', *ids.values()]))
        (directory/'inspect.json').write_text(json.dumps(record['containers'], indent=2)+'\n')
        record['load_started_at'] = time.time()
        started = time.monotonic()
        run(['docker', 'start', '-a', ids['load']], directory/'k6.log', 700)
        record['elapsed_s'] = time.monotonic()-started
        # Leave a post-load sample to bracket the measurement interval.
        time.sleep(3)
        record['exit_code'] = int(capture(['docker', 'inspect', ids['load'], '--format', '{{.State.ExitCode}}']))
        if record['exit_code']:
            raise RuntimeError(f'k6 exited {record["exit_code"]}')
    except Exception as error:
        record['error'] = str(error)
        print(f'{variant}/{name}: {error}', flush=True)
    except BaseException as error:
        record['error'] = type(error).__name__
        raise
    finally:
        record['load_finished_at'] = time.time()
        try:
            try:
                run(compose+['stop', 'collector'], directory/'stop-collector.log', 60)
                run(compose+['logs', '--no-color', '--timestamps'], directory/'services.log', 60, stream=False)
                record['final_state'] = capture(compose+['ps', '-a'])
                ids = capture(compose+['ps', '-aq']).splitlines()
                record['final_containers'] = json.loads(capture(['docker', 'inspect', *ids])) if ids else []
            except Exception as error:
                record['diagnostics_error'] = str(error)
            finally:
                run(compose+['--profile', 'load', '--profile', 'monitor', 'down', '-v', '--remove-orphans'],
                    directory/'cleanup.log', 120)
                record['cleanup']['success'] = True
        except Exception as error:
            record['cleanup']['error'] = str(error)
        finally:
            config.unlink(missing_ok=True)
            # Persist interrupted experiments before propagating KeyboardInterrupt/SIGTERM.
            (directory/'experiment.json').write_text(json.dumps(record, indent=2)+'\n')
    return record


def generate_reports(app, output):
    try:
        run(['docker', 'run', '--rm', '--network=none', '--cpuset-cpus=3', '--cpus=1', '--memory=1g',
             '--user', f'{os.getuid()}:{os.getgid()}', '-e', 'MPLCONFIGDIR=/tmp/matplotlib',
             '-v', f'{ROOT}:/source:ro', '-v', f'{output}:/artifacts', '-w', '/source', app,
             '-c', 'from pathlib import Path; from tools.lab6.report import generate; generate(Path("/artifacts"))'],
            output/'report.log', 300)
    finally:
        run(['docker', 'run', '--rm', '--network=none', '--cpuset-cpus=3', '--memory=1g',
             '--user', f'{os.getuid()}:{os.getgid()}', '-v', f'{output}:/artifacts', '-v', f'{ROOT}/tools/lab4:/lab4:ro',
             '--entrypoint', 'node', app, '/workspace/tools/lab1/node_modules/allure/cli.js',
             'generate', '/artifacts/allure-results', '--config', '/lab4/allurerc.mjs',
             '-o', '/artifacts/allure-report'], output/'allure.log', 300)


def ensure_image(image, output):
    """Reuse locally cached load/dependency images without a registry request."""
    try:
        return capture(['docker', 'image', 'inspect', image, '--format', '{{.Id}}'])
    except Exception:
        run(['docker', 'pull', image], output/f'pull-{image.split(":")[0].split("/")[-1]}.log', 600)
        return capture(['docker', 'image', 'inspect', image, '--format', '{{.Id}}'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--smoke', action='store_true')
    parser.add_argument('--profile', choices=lab4.PROFILES, action='append')
    parser.add_argument('--variant', choices=VARIANTS, action='append')
    parser.add_argument('--repetitions', type=int, default=1)
    parser.add_argument('--runtime-base', help='Rebuild current sources using a locally cached Lab 4/6 runtime image')
    parser.add_argument('--postgres-image', help='Reuse an existing PostgreSQL image with pg_cron/pg_partman')
    args = parser.parse_args()
    if args.repetitions < 1:
        parser.error('--repetitions must be at least 1')
    os.chdir(ROOT)
    if int(capture(['docker', 'info', '--format', '{{.NCPU}}'])) < 4:
        raise RuntimeError('Lab 6 requires at least 4 Docker CPU cores (app:0, DB:1, k6:2, collector:3)')
    run_id = datetime.now(timezone.utc).strftime('%Y%m%dt%H%M%S')+'-'+secrets.token_hex(3)
    output = ROOT/'build/lab6/runs'/run_id
    output.mkdir(parents=True)
    (ROOT/'build/lab6/latest.txt').write_text(str(output)+'\n')
    variants = list(dict.fromkeys(args.variant or VARIANTS))
    profiles = list(dict.fromkeys(args.profile or lab4.PROFILES))
    manifest = {'run_id': run_id, 'smoke': args.smoke, 'repetitions': args.repetitions,
                'selected_profiles': profiles, 'variants': [{'name': name, **VARIANTS[name]} for name in variants],
                'git_commit': capture(['git', 'rev-parse', 'HEAD']), 'working_tree': capture(['git', 'status', '--short']),
                'docker': json.loads(capture(['docker', 'info', '--format', '{{json .}}'])), 'profiles': []}
    sources = [ROOT/p for p in ('go.mod', 'go.sum', 'Dockerfile.postgres', 'Makefile', 'tools/lab6/Dockerfile', 'tools/lab6/Dockerfile.dockerignore', 'tools/lab6/Dockerfile.cached', 'tools/lab6/Dockerfile.cached.dockerignore', '.github/workflows/tiopo-lab6.yml')]
    for folder in ('tools/lab6', 'tools/lab4', 'tests/lab4', 'tools/lab2', 'internal', 'cmd/api', 'configs'):
        sources.extend(p for p in (ROOT/folder).rglob('*')
                       if p.is_file() and p.suffix in ('.py', '.go', '.json', '.yaml', '.js', '.mjs')
                       and not {'node_modules', 'allure-results', '__pycache__', 'build', '.cache'}.intersection(p.relative_to(ROOT).parts))
    manifest['source_sha256'] = {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(sources)}
    save_manifest(output, manifest)
    app, postgres = (f'watchtower-lab6-{kind}:{run_id}' for kind in ('app', 'postgres'))
    app_ready = False
    try:
        if args.runtime_base:
            manifest['runtime_base'] = {'image': args.runtime_base,
                                        'image_id': capture(['docker', 'image', 'inspect', args.runtime_base, '--format', '{{.Id}}'])}
            save_manifest(output, manifest)
            run(['docker', 'build', '--network=none', '-f', 'tools/lab6/Dockerfile.cached',
                 '--build-arg', f'BASE={args.runtime_base}', '-t', app, '.'], output/'build-app.log', 1500)
        else:
            run(['docker', 'build', '-f', 'tools/lab6/Dockerfile', '-t', app, '.'], output/'build-app.log', 1500)
        app_ready = True
        if args.postgres_image:
            postgres = args.postgres_image
        else:
            postgres_context = output/'postgres-context'
            postgres_context.mkdir()
            run(['docker', 'build', '-f', str(ROOT/'Dockerfile.postgres'), '-t', postgres, str(postgres_context)], output/'build-postgres.log', 900)
        manifest['postgres_image'] = {'image': postgres,
                                      'image_id': capture(['docker', 'image', 'inspect', postgres, '--format', '{{.Id}}'])}
        manifest['dependency_images'] = {image: ensure_image(image, output)
                                         for image in ('grafana/k6:1.3.0', 'redis:7.4-bookworm')}
        save_manifest(output, manifest)
        for repetition in range(1, args.repetitions+1):
            for name in profiles:
                # Rotate order across profiles/repetitions to reduce systematic order bias.
                offset = (profiles.index(name)+repetition-1) % len(variants)
                for variant in variants[offset:]+variants[:offset]:
                    record = experiment(output, run_id, name, variant, repetition, app, postgres, args.smoke)
                    manifest['profiles'].append(record)
                    save_manifest(output, manifest)
                    if not record['cleanup']['success']:
                        raise RuntimeError('Container cleanup failed; refusing to start another experiment')
    except BaseException as error:
        manifest['error'] = f'{type(error).__name__}: {error}'
        raise
    finally:
        # Recover an interrupted leaf record whose function did not return.
        recorded = {r['path'] for r in manifest['profiles']}
        for path in sorted(output.glob('variants/*/repetitions/*/profiles/*/experiment.json')):
            record = json.loads(path.read_text())
            if record['path'] not in recorded:
                manifest['profiles'].append(record)
        save_manifest(output, manifest)
        if app_ready:
            try:
                generate_reports(app, output)
            except Exception as error:
                manifest['report_error'] = str(error)
                save_manifest(output, manifest)
                if 'error' not in manifest:
                    raise
    print(f'Lab 6 artifacts: {output}', flush=True)
    if any(r.get('error') or r.get('diagnostics_error') or r['exit_code'] != 0 for r in manifest['profiles']):
        raise RuntimeError('At least one experiment failed; see manifest.json')


def entrypoint():
    # Reuse Lab 4 SIGTERM/caffeinate handling while keeping its source unchanged.
    lab4.main = main
    lab4.entrypoint()


if __name__ == '__main__':
    entrypoint()
