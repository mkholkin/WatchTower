"""Repeat unchanged Lab 4 workloads in a controlled tracing/logging matrix."""
import argparse
import base64
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


def otlp_id(value):
    """Collector OTLP IDs are hex; Tempo's protobuf JSON may encode bytes as base64."""
    if not value:
        return ''
    if len(value) in (16, 32) and all(char in '0123456789abcdefABCDEF' for char in value):
        return value.lower()
    return base64.b64decode(value, validate=True).hex()


def flush_telemetry(compose, directory, record, tracing):
    """Drain API exporters while resource sampling and downstream receivers stay live."""
    errors = []
    try:
        run(compose+['stop', 'api'], directory/'stop-api.log', 120)
        api_id = capture(compose+['ps', '-aq', 'api'])
        if api_id:
            record['api_exit_code'] = int(capture(['docker', 'inspect', api_id, '--format', '{{.State.ExitCode}}']))
        logs = capture(compose+['logs', '--no-color', 'api'])
        summary_lines = [line.split('OTEL_EXPORT_SUMMARY ', 1)[1] for line in logs.splitlines()
                         if 'OTEL_EXPORT_SUMMARY ' in line]
        if summary_lines:
            summary = json.loads(summary_lines[-1])
            (directory/'export-summary.json').write_text(json.dumps(summary, indent=2)+'\n')
            record['export_summary'] = summary
        if tracing and record.get('load_started_at') and not summary_lines:
            raise RuntimeError('API graceful shutdown did not emit OTEL_EXPORT_SUMMARY')
        if record.get('load_started_at') and record.get('api_exit_code') != 0:
            raise RuntimeError(f'API exited {record.get("api_exit_code")} during shutdown')
        # SDK Shutdown has completed; the synchronous Collector exporters can settle.
        time.sleep(2)
        run(compose+['exec', '-T', 'probe', 'curl', '--fail', '--silent',
                     'http://otel-collector:8888/metrics'], directory/'otel-collector-metrics.txt', 30, stream=False)
    except Exception as error:
        errors.append(str(error))
    try:
        run(compose+['stop', 'otel-collector'], directory/'stop-otel-collector.log', 70)
        path = directory/'traces-otlp.jsonl'
        span_count, unique_spans, trace_id = 0, set(), None
        http_roots, sql_parents = {}, {}
        if path.exists():
            with path.open() as source:
                for line in source:
                    for resource in json.loads(line).get('resourceSpans', []):
                        for scope in resource.get('scopeSpans', []):
                            for span in scope.get('spans', []):
                                span_count += 1
                                unique_spans.add((span['traceId'], span['spanId']))
                                trace_id = trace_id or span['traceId']
                                if span.get('kind') in (2, 'SPAN_KIND_SERVER'):
                                    http_roots.setdefault(otlp_id(span['traceId']), set()).add(otlp_id(span['spanId']))
                                elif span.get('name', '').startswith('postgres '):
                                    sql_parents.setdefault(otlp_id(span['traceId']), set()).add(otlp_id(span.get('parentSpanId')))
        counts = {'spans': span_count, 'unique_spans': len(unique_spans)}
        if tracing and record.get('load_started_at'):
            if not span_count:
                raise RuntimeError('Collector did not export any OTLP spans')
            correlated = [key for key, roots in http_roots.items() if roots & sql_parents.get(key, set())]
            if not correlated:
                raise RuntimeError('Collector export contains no correlated HTTP/SQL trace')
            trace_id = correlated[0]
            response = capture(compose+['exec', '-T', 'probe', 'curl', '--fail', '--silent',
                                        '--retry', '15', '--retry-all-errors', '--retry-delay', '1',
                                        '--max-time', '5', '-H', 'Accept: application/json',
                                        'http://tempo:3200/api/traces/'+trace_id])
            trace = json.loads(response)
            (directory/'tempo-live-trace.json').write_text(json.dumps(trace)+'\n')
            live_spans = [span for resource in trace.get('batches', trace.get('resourceSpans', []))
                          for scope in resource.get('scopeSpans', []) for span in scope.get('spans', [])]
            live_http = {otlp_id(span['spanId']) for span in live_spans
                         if span.get('kind') in (2, 'SPAN_KIND_SERVER')}
            if not any(span.get('name', '').startswith('postgres ') and otlp_id(span.get('parentSpanId')) in live_http
                       for span in live_spans):
                raise RuntimeError('Tempo live trace lacks correlated HTTP/SQL spans')
            counts.update(trace_id=trace_id, tempo_live_verified=True)
        else:
            counts['tempo_live_verified'] = False
            if span_count:
                raise RuntimeError('Tracing-disabled variant exported spans')
        (directory/'tempo-live-check.json').write_text(json.dumps(counts, indent=2)+'\n')
        record['telemetry'] = counts
    except Exception as error:
        errors.append(str(error))
    finally:
        try:
            run(compose+['stop', 'tempo'], directory/'stop-tempo.log', 70)
        except Exception as error:
            errors.append(str(error))
    if errors:
        raise RuntimeError('; '.join(errors))


def experiment(output, run_id, name, variant, repetition, app, postgres, smoke):
    directory = output/'variants'/variant/'repetitions'/str(repetition)/'profiles'/name
    directory.mkdir(parents=True)
    (directory/'tempo-data').mkdir()
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
                      LAB6_TRACING_ENABLED=str(options['tracing']).lower(),
                      LAB6_RESOURCE_ATTRIBUTES=f'run_id={run_id},profile={name},variant={variant},repetition={repetition}',
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
        for url in ('http://otel-collector:13133/', 'http://tempo:3200/ready'):
            run(compose+['exec', '-T', 'probe', 'curl', '--fail', '--silent', '--retry', '40',
                         '--retry-all-errors', '--retry-delay', '1', '--max-time', '2', url],
                directory/('ready-otel.log' if '13133' in url else 'ready-tempo.log'), 120)
        run(compose+['--profile', 'load', 'create', 'load'], directory/'create-load.log')
        ids = {service: capture(compose+['ps', '-aq', service]) for service in ('api', 'postgres', 'redis', 'probe', 'load', 'otel-collector', 'tempo')}
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
                flush_telemetry(compose, directory, record, options['tracing'])
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
    sources = [ROOT/p for p in ('go.mod', 'go.sum', 'Dockerfile.postgres', 'Makefile', 'tools/lab6/Dockerfile', 'tools/lab6/Dockerfile.dockerignore', 'tools/lab6/Dockerfile.dependencies', 'tools/lab6/Dockerfile.dependencies.dockerignore', 'tools/lab6/Dockerfile.cached', 'tools/lab6/Dockerfile.cached.dockerignore', '.github/workflows/tiopo-lab6.yml')]
    for folder in ('tools/lab6', 'tools/lab4', 'tests/lab4', 'tools/lab2', 'internal', 'pkg', 'cmd/api', 'configs'):
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
            dependencies = f'{app}-dependencies'
            run(['docker', 'build', '-f', 'tools/lab6/Dockerfile.dependencies',
                 '--build-arg', f'BASE={args.runtime_base}', '-t', dependencies, '.'],
                output/'build-dependencies.log', 900)
            run(['docker', 'build', '--network=none', '-f', 'tools/lab6/Dockerfile.cached',
                 '--build-arg', f'BASE={dependencies}', '-t', app, '.'], output/'build-app.log', 1500)
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
                                         for image in ('grafana/k6:1.3.0', 'redis:7.4-bookworm',
                                                       'otel/opentelemetry-collector-contrib:0.140.0', 'grafana/tempo:2.9.0')}
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
