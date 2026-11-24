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
import urllib.parse
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
        window = dict(common, window_start_ms=str(int(start*1000)), window_end_ms=str(int(end*1000)+1000))
        for component, resource in profile['resources'].items():
            tags = dict(window, component=component)
            add('lab6_summary_cpu_seconds', tags, resource['cpu_seconds'], end-.001)
            for stat, value in resource['memory_bytes'].items():
                add('lab6_summary_memory_bytes', dict(tags, stat=stat), value, end-.001)
        add('lab6_summary_rps', dict(window, component='api'), profile['load']['rps']['median'], end-.001)
        add('lab6_summary_unexpected_ratio', dict(window, component='api'), profile['load']['unexpected_ratio'], end-.001)
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
    for source in (ROOT/'tools/lab6/grafana/dashboards').glob('*.json'):
        dashboard = json.loads(source.read_text())
        start, end = minimum-15, maximum+15
        profile_names = list(dict.fromkeys(p['name'] for p in summary['profiles']))
        selected_profile = next((v['current']['value'] for v in dashboard['templating']['list']
                                 if v['name'] == 'profile' and v['current']['value'] in profile_names), profile_names[0])
        repetitions = list(dict.fromkeys(str(p['repetition']) for p in summary['profiles']
                                        if p['name'] == selected_profile))
        for variable in dashboard['templating']['list']:
            name = variable['name']
            values = profile_names if name == 'profile' else repetitions if name == 'repetition' else None
            if name == 'variant':
                values = list(dict.fromkeys(p['variant'] for p in summary['profiles'] if p['name'] == selected_profile))
            if name == 'run':
                values = [summary['run_id']]
            if values is None:
                continue
            captions = {}
            if variable.get('type') == 'custom':
                for item in variable.get('query', '').split(','):
                    caption, separator, value = item.strip().partition(' : ')
                    captions[value if separator else caption] = caption
            options = [dict(text=captions.get(value, value), value=value, selected=False) for value in values]
            current = variable['current']['value']
            if name == 'profile':
                current = selected_profile
            elif current not in values:
                current = values[0]
            for option in options:
                option['selected'] = option['value'] == current
            variable.update(type='custom', query=', '.join(option['text']+' : '+option['value'] for option in options),
                            options=options, current=next(dict(option) for option in options if option['selected']),
                            skipUrlSync=len(values) == 1)
            for key in ('definition', 'datasource', 'refresh'):
                variable.pop(key, None)
        reset = urllib.parse.urlencode({'from': int((minimum-15)*1000), 'to': int((maximum+15)*1000),
                                        'var-run': summary['run_id'], 'var-profile': selected_profile,
                                        'var-repetition': repetitions[0], 'var-component': 'api'})
        dashboard.setdefault('links', []).append(dict(title='Показать данные текущего запуска', type='link',
                                                      url='/d/lab6-observability?'+reset, targetBlank=False))
        if dashboard['uid'] == 'lab6-detail':
            defaults = {v['name']: v['current']['value'] for v in dashboard['templating']['list']}
            variants = [p['variant'] for p in summary['profiles'] if p['name'] == defaults['profile']]
            if defaults['variant'] not in variants:
                defaults['variant'] = variants[0]
                variable = next(v for v in dashboard['templating']['list'] if v['name'] == 'variant')
                variable['current'] = dict(text=variants[0], value=variants[0], selected=True)
            selected = next((p for p in summary['profiles'] if p['name'] == defaults['profile']
                             and p['variant'] == defaults['variant'] and str(p['repetition']) == defaults['repetition']), None)
            if selected:
                start, end = selected['measurement_start'], selected['measurement_end']+1
        for panel in dashboard['panels']:
            for query in panel.get('targets', []):
                if 'expr' in query and 'lab6_summary_' in query['expr']:
                    duration = math.ceil(maximum-minimum)+30
                    query['expr'] = query['expr'].replace('[$__range]', f'[{duration}s] @{maximum+15:.6f}')
        dashboard['time'] = {'from': str(int(start*1000)), 'to': str(int(end*1000))}
        (target/'dashboards'/source.name).write_text(json.dumps(dashboard, indent=2, ensure_ascii=False)+'\n')
    metadata = dict(run_id=summary['run_id'], original_from=minimum, original_to=maximum,
                    series=len(series), samples=sum(map(len, series.values())))
    (target/'import.json').write_text(json.dumps(metadata, indent=2)+'\n')
    return metadata


def request(url, payload=None):
    if payload is not None:
        url = urllib.request.Request(url, data=json.dumps(payload).encode(), headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(url, timeout=10) as response:
        return json.load(response)


def check_panel(dashboard, panel, base_url='http://localhost:3036'):
    defaults = {v['name']: v['current']['value'] for v in dashboard['templating']['list']}
    checks = []
    for target in panel.get('targets', []):
        if 'expr' not in target:
            continue
        query = dict(target)
        query['expr'] = re.sub(r'\$\{([a-z]+):regex\}',
                               lambda match: json.dumps(re.escape(defaults[match[1]]))[1:-1], query['expr'])
        query['datasource'] = query.get('datasource') or panel.get('datasource') or {'type': 'prometheus', 'uid': 'lab6-prometheus'}
        query.update(intervalMs=1000, maxDataPoints=1000)
        end = dashboard['time']['to']
        payload = {'from': end if query.get('instant') else dashboard['time']['from'],
                   'to': end, 'queries': [query]}
        response = request(base_url+'/api/ds/query', payload)
        result = response.get('results', {}).get(query['refId'], {})
        if response.get('error') or result.get('error') or result.get('status', 200) != 200:
            raise RuntimeError(f'Grafana panel query failed: {panel["title"]}')
        numeric_fields = 0
        for frame in result.get('frames', []):
            values = frame.get('data', {}).get('values', [])
            for index, field in enumerate(frame.get('schema', {}).get('fields', [])):
                if field.get('type') == 'number' and index < len(values) and any(
                        isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value)
                        for value in values[index]):
                    numeric_fields += 1
        if not numeric_fields:
            raise RuntimeError(f'Grafana panel has no numeric data: {panel["title"]}')
        checks.append(dict(dashboard=dashboard['uid'], panel=panel['title'], numeric_fields=numeric_fields))
    return checks



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
    run = (args.run_dir or Path((ROOT/'build/lab6/latest.txt').read_text().strip())).expanduser().resolve()
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
        for dashboard in (target/'dashboards').glob('*.json'):
            dashboard.write_text(dashboard.read_text().replace('__EXAMPLE_TRACE_ID__', traces['example_trace_id'] or 'No exported traces'))
    meta = json.loads((target/'import.json').read_text())
    import urllib.parse
    query = urllib.parse.urlencode({'query':f'count(last_over_time(lab6_summary_cpu_seconds[{math.ceil(meta["original_to"]-meta["original_from"])+30}s]))','time':meta['original_to']})
    result = request('http://localhost:9096/api/v1/query?'+query)
    expected = sum(len(p['resources']) for p in json.loads((run/'summary.json').read_text())['profiles'])
    if not result['data']['result'] or int(result['data']['result'][0]['value'][1]) != expected:
        raise RuntimeError('Prometheus summary series do not match the experiment')
    panel_checks = []
    for uid in ('lab6-observability', 'lab6-detail'):
        dashboard = request('http://localhost:3036/api/dashboards/uid/'+uid)['dashboard']
        for panel in dashboard['panels']:
            panel_checks.extend(check_panel(dashboard, panel))
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
