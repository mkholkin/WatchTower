"""Load measured Lab 6 samples into Prometheus and correlated spans into Tempo."""
import argparse
import collections
import json
import math
import os
import re
import socket
from pathlib import Path
import shutil
import subprocess
import sys
import time
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))
from tools.lab6.tempo import import_traces
from tools.lab6.report import resource_summary
from tools.lab4.collect import Engine, sample


def labels(values):
    return '{'+','.join(k+'='+json.dumps(str(v)) for k, v in sorted(values.items()))+'}'


def prepare(run, target):
    run, target = Path(run), Path(target)
    summary = json.loads((run/'summary.json').read_text())
    if summary['failed_profiles'] or any(r['integrity_issues'] for r in summary['profiles']):
        raise ValueError('Cannot visualize an invalid experiment as verified measurements')
    series = collections.defaultdict(list)
    minimum, maximum = math.inf, -math.inf
    def add(name, tags, value, stamp):
        if not isinstance(value, (int, float)) or not math.isfinite(value):
            return
        series[name+labels(tags)].append((stamp, value))
    for profile in summary['profiles']:
        common = dict(run=summary['run_id'], profile=profile['name'], variant=profile['variant'], repetition=profile['repetition'])
        start, end = profile['measurement_start'], profile['measurement_end']
        minimum, maximum = min(minimum, start), max(maximum, end)
        directory = (run/profile['artifact_path']).resolve()
        if not directory.is_relative_to(run.resolve()):
            raise ValueError('Artifact path escapes run directory')
        with (directory/'resources.jsonl').open() as source:
            for line in source:
                row = json.loads(line)
                if row.get('error') or not start <= row['time'] <= end:
                    continue
                tags = dict(common, component=row['component'])
                cpu = row.get('raw', {}).get('cpu_stats', {}).get('cpu_usage', {}).get('total_usage')
                if cpu is not None:
                    add('lab6_cpu_seconds_total', tags, cpu/1e9, row['time'])
                add('lab6_memory_bytes', tags, row.get('memory_bytes'), row['time'])
        for component, resource in profile['resources'].items():
            tags = dict(common, component=component)
            add('lab6_summary_cpu_seconds', tags, resource['cpu_seconds'], end-.001)
            for stat, value in resource['memory_bytes'].items():
                add('lab6_summary_memory_bytes', dict(tags, stat=stat), value, end-.001)
        add('lab6_summary_rps', dict(common, component='api'), profile['load']['rps']['median'], end-.001)
        add('lab6_summary_unexpected_ratio', dict(common, component='api'), profile['load']['unexpected_ratio'], end-.001)
    if not series:
        raise ValueError('No measured samples found')
    target.mkdir(parents=True, exist_ok=True)
    for name in ['prometheus-data','grafana-data','tempo-data','dashboards']:
        (target/name).mkdir(exist_ok=True)
    with (target/'metrics.om').open('w') as stream:
        stream.write('# TYPE lab6_cpu_seconds counter\n')
        for metric, samples in sorted(series.items()):
            for stamp, value in sorted(samples):
                stream.write(f'{metric} {value:.12g} {stamp:.6f}\n')
        stream.write('# EOF\n')
    dashboard = json.loads((ROOT/'tools/lab6/grafana/dashboards/lab6.json').read_text())
    dashboard['time'] = {'from': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(minimum-15)),
                         'to': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(maximum+15))}
    (target/'dashboards/lab6.json').write_text(json.dumps(dashboard, indent=2)+'\n')
    metadata = dict(run_id=summary['run_id'], original_from=minimum, original_to=maximum,
                    series=len(series), samples=sum(map(len, series.values())))
    (target/'import.json').write_text(json.dumps(metadata, indent=2)+'\n')
    return metadata


def request(url):
    with urllib.request.urlopen(url, timeout=10) as response:
        return json.load(response)


def ready(url):
    for _ in range(90):
        try:
            with urllib.request.urlopen(url, timeout=3): return
        except OSError: time.sleep(1)
    raise RuntimeError(f'Service did not become ready: {url}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['up','down','check','prepare'])
    parser.add_argument('--run-dir', type=Path)
    parser.add_argument('--measure-seconds', type=int, default=30)
    args = parser.parse_args()
    if args.measure_seconds < 5:
        parser.error('--measure-seconds must be at least 5')
    run = (args.run_dir or Path((ROOT/'build/lab6/latest.txt').read_text().strip())).resolve()
    target = ROOT/'build/lab6/grafana'/run.name
    env = dict(os.environ, LAB6_GRAFANA_DATA=str(target), LAB6_UID=str(os.getuid()), LAB6_GID=str(os.getgid()))
    compose = ['docker','compose','-p','wt-lab6-grafana','-f',str(ROOT/'tools/lab6/grafana/compose.yaml')]
    def command(*argv): subprocess.run([*argv], env=env, cwd=ROOT, check=True)
    if args.action == 'down':
        command(*compose,'down'); return
    if args.action in ('prepare','up'):
        meta = prepare(run, target)
        if args.action == 'prepare': print(json.dumps(meta)); return
        command(*compose,'down')
        # Import while Prometheus is stopped: no mutable head blocks can overlap.
        for name in ('prometheus-data', 'tempo-data'):
            shutil.rmtree(target/name)
            (target/name).mkdir()
        command(*compose,'run','--rm','--no-deps','--entrypoint','/bin/promtool','prometheus',
                'tsdb','create-blocks-from','openmetrics','/imports/metrics.om','/prometheus')
        command(*compose,'up','-d')
        ready('http://localhost:9096/-/ready'); ready('http://localhost:3206/ready'); ready('http://localhost:3036/api/health')
        traces = import_traces(run, 'http://localhost:4316')
        (target/'traces-import.json').write_text(json.dumps(traces, indent=2)+'\n')
        dashboard = target/'dashboards/lab6.json'
        dashboard.write_text(dashboard.read_text().replace('__EXAMPLE_TRACE_ID__', traces['example_trace_id']))
    meta = json.loads((target/'import.json').read_text())
    import urllib.parse
    query = urllib.parse.urlencode({'query':f'count(last_over_time(lab6_summary_cpu_seconds[{math.ceil(meta["original_to"]-meta["original_from"])+30}s]))','time':meta['original_to']})
    result = request('http://localhost:9096/api/v1/query?'+query)
    expected = len(json.loads((run/'summary.json').read_text())['profiles'])*2
    if not result['data']['result'] or int(result['data']['result'][0]['value'][1]) != expected:
        raise RuntimeError('Prometheus summary series do not match the experiment')
    dashboard = request('http://localhost:3036/api/dashboards/uid/lab6-observability')['dashboard']
    panel_checks = []
    for panel in dashboard['panels']:
        for expression in panel.get('targets', []):
            expression = re.sub(r'\$\{[a-z]+:regex\}', '.*', expression['expr'])
            expression = expression.replace('$__range', f'{math.ceil(meta["original_to"]-meta["original_from"])+30}s')
            query = urllib.parse.urlencode(dict(query=expression, start=meta['original_from'],
                                                end=meta['original_to']+15, step=10))
            result = request('http://localhost:3036/api/datasources/proxy/uid/lab6-prometheus/api/v1/query_range?'+query)
            if result.get('status') != 'success' or not result['data']['result']:
                raise RuntimeError(f'Grafana panel has no data: {panel["title"]}')
            panel_checks.append(dict(panel=panel['title'], series=len(result['data']['result'])))
    traces = json.loads((target/'traces-import.json').read_text())
    trace_url = 'http://localhost:3036/api/datasources/proxy/uid/lab6-tempo/api/traces/'+traces['example_trace_id']
    ready(trace_url)
    trace = request(trace_url)
    spans = [span for batch in trace.get('batches', []) for scope in batch.get('scopeSpans', []) for span in scope['spans']]
    span_ids = {span['spanId'] for span in spans}
    if not any(span['name'].startswith('postgres ') and span.get('parentSpanId') in span_ids for span in spans):
        raise RuntimeError('Example Tempo trace does not contain correlated HTTP/SQL spans')
    (target/'query-checks.json').write_text(json.dumps(dict(panels=panel_checks, correlated_spans=len(spans)), indent=2)+'\n')
    services = ['grafana', 'prometheus', 'tempo']
    ids = {s: subprocess.check_output(compose+['ps', '-q', s], env=env, text=True).strip() for s in services}
    host = (os.environ.get('DOCKER_HOST') if not os.environ.get('DOCKER_CONTEXT') else None) or subprocess.check_output(
        ['docker', 'context', 'inspect', '--format', '{{.Endpoints.docker.Host}}'], env=env, text=True).strip()
    if not host.startswith('unix://'):
        raise RuntimeError('Resource measurement requires a local Docker Unix socket')
    class LocalEngine(Engine):
        def connect(self):
            self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            self.sock.settimeout(self.timeout)
            self.sock.connect(host[7:])
    rows, previous = [], {}
    deadline = time.monotonic()+args.measure_seconds
    while True:
        for name, identity in ids.items():
            conn = LocalEngine()
            try:
                conn.request('GET', f'/containers/{identity}/stats?stream=false&one-shot=true')
                response = conn.getresponse()
                raw = json.loads(response.read())
                if response.status != 200:
                    raise RuntimeError(f'Docker stats for {name}: {response.status}')
                stamp = time.time()
                old_time, old = previous.get(name, (stamp, {}))
                rows.append(dict(time=stamp, component=name, raw=raw, **sample(raw, old, stamp-old_time)))
                previous[name] = stamp, raw
            finally:
                conn.close()
        if time.monotonic() >= deadline:
            break
        time.sleep(1)
    start = max(r['time'] for r in rows[:len(services)])
    end = min(r['time'] for r in rows[-len(services):])
    resources = {name: resource_summary(rows, name, start, end) for name in services}
    (target/'viewer-resources.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in rows))
    evidence = dict(phase='post-import viewer, not original workload', run_id=meta['run_id'],
                    measurement_start=start, measurement_end=end, measurement_seconds=end-start,
                    resources=resources, checks=dict(prometheus=True, grafana=True, tempo=True))
    (target/'viewer-resources.json').write_text(json.dumps(evidence, indent=2)+'\n')
    print('Grafana: http://localhost:3036/d/lab6-observability')
    print('Original timestamps preserved; viewer resource samples:',target/'viewer-resources.json')

if __name__ == '__main__': main()
