"""Compare matched Lab 4 workloads with tracing and HTTP diagnostics enabled."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import re
import shutil
import statistics
import uuid

from tools.lab4 import report as lab4

VARIANTS = ('baseline', 'trace', 'debug', 'trace-debug')
PROFILES = ('ramp', 'stress', 'soak-0', 'soak-10', 'soak-20')
COMPONENTS = ('api', 'collector')


def number(value):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value)


def resource_summary(rows, component, start, end):
    """Interpolate cumulative nanoseconds at both boundaries; never sum CPU %."""
    if not number(start) or not number(end) or end <= start:
        raise ValueError('Invalid measurement window')
    selected = sorted((r for r in rows if r.get('component') == component), key=lambda r: r['time'])
    if len(selected) < 2:
        raise ValueError(f'{component}: missing resource samples')
    relevant = [r for r in selected if start-5 <= r['time'] <= end+5]
    counters = []
    for row in relevant:
        value = row.get('raw', {}).get('cpu_stats', {}).get('cpu_usage', {}).get('total_usage')
        if row.get('error') or not number(value) or value < 0:
            raise ValueError(f'{component}: missing/invalid raw CPU counter')
        counters.append((row['time'], value))
    if any(b[0] <= a[0] or b[0]-a[0] > 5 or b[1] < a[1] for a, b in zip(counters, counters[1:])):
        raise ValueError(f'{component}: CPU counter reset, duplicate time or sampling gap')

    def at(stamp):
        for left, right in zip(counters, counters[1:]):
            if left[0] <= stamp <= right[0]:
                return left[1]+(right[1]-left[1])*(stamp-left[0])/(right[0]-left[0])
        raise ValueError(f'{component}: raw CPU counters do not bracket measurement window')

    cpu_start, cpu_end = at(start), at(end)
    memory = [r.get('memory_bytes') for r in selected if start <= r['time'] < end]
    if not memory or any(not number(v) or v < 0 for v in memory):
        raise ValueError(f'{component}: missing/invalid memory samples')
    return {'cpu_seconds': (cpu_end-cpu_start)/1e9,
            'cpu_start_ns': cpu_start, 'cpu_end_ns': cpu_end,
            'cpu_boundary_method': 'linear interpolation of adjacent raw cumulative counters',
            'mean_cpu_percent': (cpu_end-cpu_start)/1e9/(end-start)*100,
            'memory_bytes': {'min': min(memory), 'median': statistics.median(memory), 'max': max(memory)},
            'memory_samples': len(memory)}


def trace_summary(path, enabled, start, end):
    if not path.exists() or path.stat().st_size == 0:
        if enabled:
            raise ValueError('Tracing enabled but traces.jsonl is missing or empty')
        return {'events': 0, 'measurement_events': 0, 'bytes': 0}
    if not enabled:
        raise ValueError('Tracing disabled but trace events were recorded')
    events = list(lab4.read_jsonl(path))
    measured = []
    for event in events:
        required = {'trace_id', 'span_id', 'name', 'started_ns', 'duration_ns', 'status'}
        if not required <= set(event) or set(event)-required-{'parent_id'}:
            raise ValueError('Unsafe or invalid trace event fields')
        if any(not number(event[key]) or event[key] < 0 for key in ('started_ns', 'duration_ns')):
            raise ValueError('Invalid trace timestamp/duration')
        for key in ('trace_id', 'span_id', 'parent_id'):
            if event.get(key) and not re.fullmatch(r'[0-9a-f]{16,32}', event[key]):
                raise ValueError('Invalid trace identity')
        if not isinstance(event['status'], int) or not 100 <= event['status'] <= 599:
            raise ValueError('Invalid trace status')
        name = event['name']
        if not isinstance(name, str) or not re.fullmatch(
                r'(?:postgres (?:SELECT|INSERT|UPDATE|DELETE|BEGIN|COMMIT|ROLLBACK|CREATE|ALTER|DROP|GRANT|DO|OTHER)|'
                r'(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|OTHER) (?:/[A-Za-z0-9_/:.{}-]*|unmatched))', name):
            raise ValueError('Trace includes an unsanitized route or secret')
        if re.search(r'[0-9a-f]{8}-[0-9a-f-]{27,}', name, re.I):
            raise ValueError('Trace includes an unsanitized route identifier')
        if start <= event['started_ns']/1e9 < end:
            measured.append(event)
    if not measured:
        raise ValueError('Tracing enabled but no events cover the measurement window')
    chrome = {'traceEvents': [{'name': e['name'], 'cat': 'watchtower', 'ph': 'X',
                              'ts': e['started_ns']/1000, 'dur': e['duration_ns']/1000,
                              'pid': 1, 'tid': int(e['trace_id'][:12], 16),
                              'args': {'status': e['status'], 'trace_id': e['trace_id'], 'span_id': e['span_id'],
                                       'parent_id': e.get('parent_id', '')}}
                             for e in measured]}
    path.with_name('trace.json').write_text(json.dumps(chrome)+'\n')
    return {'events': len(events), 'measurement_events': len(measured), 'bytes': path.stat().st_size}


def compare(current, baseline):
    result = {}
    for component in COMPONENTS:
        now, before = current['resources'][component], baseline['resources'][component]
        cpu_delta = now['cpu_seconds']-before['cpu_seconds']
        result[component] = {'cpu_seconds_delta': cpu_delta,
                             'cpu_seconds_delta_percent': cpu_delta/before['cpu_seconds']*100 if before['cpu_seconds'] else None,
                             'memory_bytes_delta': {key: now['memory_bytes'][key]-before['memory_bytes'][key]
                                                    for key in ('min', 'median', 'max')}}
    result['rps_median_delta'] = current['load']['rps']['median']-baseline['load']['rps']['median']
    result['unexpected_ratio_delta'] = current['load']['unexpected_ratio']-baseline['load']['unexpected_ratio']
    return result


def container_configuration(record):
    configuration, issues = {}, []
    for container in record.get('containers', []):
        name = container.get('Config', {}).get('Labels', {}).get('com.docker.compose.service')
        if name not in lab4.CPU_SETS:
            issues.append('Unknown container component')
            continue
        host = container.get('HostConfig', {})
        configuration[name] = {key: host.get(key) for key in ('CpusetCpus', 'NanoCpus', 'Memory')}
        if host.get('CpusetCpus') != lab4.CPU_SETS[name] or not number(host.get('Memory')) or host['Memory'] <= 0 or not number(host.get('NanoCpus')) or host['NanoCpus'] <= 0:
            issues.append(f'{name}: invalid CPU isolation/resource limits')
    if set(configuration) != set(lab4.CPU_SETS):
        issues.append('Missing container configuration')
    for container in record.get('final_containers', []):
        if container.get('State', {}).get('OOMKilled') or container.get('RestartCount'):
            issues.append('OOM killed or unexpected container restart')
    return configuration, issues


def plot_comparison(root, summaries):
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
    plots = root/'plots'
    plots.mkdir(exist_ok=True)
    for profile in PROFILES:
        rows = [r for r in summaries if r['name'] == profile]
        if not rows:
            continue
        fig, axes = plt.subplots(2, 2, figsize=(13, 8), constrained_layout=True)
        mode = 'diagnostic smoke' if rows[0]['load']['config'].get('smoke') else 'full workload'
        fig.suptitle(f'Lab 6: {profile} — {mode}')
        labels = [f'{r["variant"]} / {r["repetition"]}' for r in rows]
        for index, component in enumerate(COMPONENTS):
            values = [r['resources'][component] for r in rows]
            axes[index, 0].bar(labels, [v['cpu_seconds'] for v in values])
            axes[index, 0].set(title=component, ylabel='CPU-seconds in measurement window')
            medians = [v['memory_bytes']['median']/2**20 for v in values]
            errors = [[m-v['memory_bytes']['min']/2**20 for m, v in zip(medians, values)],
                      [v['memory_bytes']['max']/2**20-m for m, v in zip(medians, values)]]
            axes[index, 1].bar(labels, medians, yerr=errors, capsize=4)
            axes[index, 1].set(title=component, ylabel='RAM MiB: median, whiskers min/max')
        for ax in axes.flat:
            ax.tick_params(axis='x', rotation=25)
            ax.grid(axis='y', alpha=.2)
        fig.savefig(plots/f'{profile}.png', dpi=130)
        plt.close(fig)


def write_reports(root):
    root = Path(root)
    manifest = json.loads((root/'manifest.json').read_text())
    output = root/'allure-results'
    output.mkdir(exist_ok=True)
    for previous in output.glob('*-result.json'):
        previous.unlink()
    summaries, failures = [], []
    for record in manifest['profiles']:
        label = f'{record["name"]}/{record["variant"]}/{record.get("repetition", 1)}'
        try:
            if record.get('exit_code') != 0 or record.get('error'):
                raise ValueError('Profile execution failed')
            if record['variant'] not in VARIANTS or record['name'] not in PROFILES:
                raise ValueError('Unknown profile/variant')
            directory = (root/record['path']).resolve()
            if not directory.is_relative_to(root.resolve()):
                raise ValueError('Profile path escapes artifact directory')
            load, _, _ = lab4.summarize(directory)
            start = load['origin_epoch_s']+load['config']['phases'][0]['seconds']
            end = load['origin_epoch_s']+load['config']['duration']
            for key, expected in (('measurement_start', start), ('measurement_end', end)):
                if key in record and abs(record[key]-expected) > .001:
                    raise ValueError('Manifest measurement window differs from request timestamps')
            raw = list(lab4.read_jsonl(directory/'resources.jsonl'))
            resources = {name: resource_summary(raw, name, start, end) for name in COMPONENTS}
            traces = trace_summary(directory/'traces.jsonl', record['variant'] in ('trace', 'trace-debug'), start, end)
            summary = {'name': record['name'], 'variant': record['variant'], 'repetition': record.get('repetition', 1),
                       'artifact_path': record['path'],
                       'measurement_start': start, 'measurement_end': end, 'measurement_seconds': end-start,
                       'resources': resources, 'traces': traces, 'load': load,
                       'integrity_issues': list(load['integrity_issues'])}
            configuration, issues = container_configuration(record)
            summary['container_limits'] = configuration
            summary['integrity_issues'].extend(issues)
            summaries.append(summary)
        except (OSError, ValueError, KeyError, TypeError) as error:
            failures.append({'name': label, 'error': str(error)})
    baselines = {(s['name'], s['repetition']): s for s in summaries if s['variant'] == 'baseline'}
    for summary in summaries:
        baseline = baselines.get((summary['name'], summary['repetition']))
        if baseline is None:
            summary['integrity_issues'].append('Missing matched baseline')
        elif baseline['load']['config'] != summary['load']['config']:
            summary['integrity_issues'].append('Workload differs from matched baseline')
        else:
            summary['baseline_delta'] = compare(summary, baseline)
            if baseline['container_limits'] != summary['container_limits']:
                summary['integrity_issues'].append('Container limits differ from matched baseline')
    keys = [(r['name'], r['variant'], r.get('repetition', 1)) for r in manifest['profiles']]
    if len(keys) != len(set(keys)):
        failures.append({'name': 'matrix', 'error': 'Duplicate profile/variant/repetition'})
    if not manifest.get('smoke'):
        for repetition in {key[2] for key in keys} or {1}:
            missing = [(p, v) for p in PROFILES for v in VARIANTS if (p, v, repetition) not in keys]
            if missing:
                failures.append({'name': 'matrix', 'error': f'Missing full matrix entries: {missing}'})
    combined = {'run_id': manifest['run_id'], 'smoke': manifest['smoke'], 'profiles': summaries,
                'failed_profiles': failures, 'report_source_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
    (root/'summary.json').write_text(json.dumps(combined, indent=2)+'\n')
    plot_comparison(root, summaries)
    for entry in [*summaries, *failures]:
        broken = 'variant' not in entry
        label = entry['name'] if broken else f'{entry["name"]}/{entry["variant"]}/{entry["repetition"]}'
        identity = str(uuid.uuid4())
        attachment = identity+'-summary.json'
        (output/attachment).write_text(json.dumps(entry, indent=2)+'\n')
        attachments = [{'name': 'Measured summary', 'source': attachment, 'type': 'application/json'}]
        if not broken and entry['traces']['measurement_events']:
            trace_attachment = identity+'-trace.json'
            shutil.copyfile(root/entry['artifact_path']/'trace.json', output/trace_attachment)
            attachments.append({'name': 'Chrome Trace (measurement window)', 'source': trace_attachment,
                                'type': 'application/json'})
        if not broken and (root/'plots'/f'{entry["name"]}.png').exists():
            target = entry['name']+'.png'
            shutil.copyfile(root/'plots'/target, output/target)
            attachments.append({'name': 'Matched comparison', 'source': target, 'type': 'image/png'})
        issues = [entry['error']] if broken else entry['integrity_issues']
        item = {'uuid': identity, 'name': label, 'fullName': 'lab6.'+label,
                'historyId': hashlib.sha256(('lab6.'+label).encode()).hexdigest(),
                'status': 'broken' if broken else 'failed' if issues else 'skipped' if manifest['smoke'] else 'passed',
                'statusDetails': {'message': '; '.join(issues) or ('Diagnostic smoke only' if manifest['smoke'] else 'Measurement integrity validated; performance is described in attachments.')},
                'stage': 'finished', 'labels': [{'name': 'parentSuite', 'value': 'Lab 6 tracing overhead'}],
                'attachments': attachments}
        (output/f'{identity}-result.json').write_text(json.dumps(item)+'\n')
    lines = ['# Lab 6 measured overhead', '', f'Run: `{manifest["run_id"]}`; smoke: `{manifest["smoke"]}`.', '',
             'CPU-seconds are raw cumulative CPU nanosecond differences at measurement boundaries, linearly interpolated between adjacent samples. RAM is working set; warmup and cooldown are excluded. Collector overhead is reported separately. Δ compares the same profile and repetition with baseline.', '',
             '| Profile | Variant / repetition | Component | CPU s | Δ CPU s | Δ CPU % | RAM min / median / max MiB | Δ median MiB | RPS median | Unexpected % | Intentional % |',
             '|---|---|---|---:|---:|---:|---|---:|---:|---:|---:|']
    for s in summaries:
        for component in COMPONENTS:
            values = s['resources'][component]
            delta = s.get('baseline_delta', {}).get(component, {})
            fmt = lambda value: f'{value:.3f}' if value is not None else 'n/a'
            memory = ' / '.join(fmt(values['memory_bytes'][key]/2**20) for key in ('min', 'median', 'max'))
            median_delta = delta.get('memory_bytes_delta', {}).get('median')
            lines.append(f'| {s["name"]} | {s["variant"]} / {s["repetition"]} | {component} | {values["cpu_seconds"]:.3f} | {fmt(delta.get("cpu_seconds_delta"))} | {fmt(delta.get("cpu_seconds_delta_percent"))} | {memory} | {fmt(median_delta/2**20 if median_delta is not None else None)} | {s["load"]["rps"]["median"]:.2f} | {s["load"]["unexpected_ratio"]*100:.2f} | {s["load"]["intentional_error_percent"]:.2f} |')
        if s['integrity_issues']:
            lines += ['', f'{s["name"]}/{s["variant"]}: '+ '; '.join(s['integrity_issues']), '']
    for failure in failures:
        lines += ['', f'Failed `{failure["name"]}`: {failure["error"]}', '']
    lines += ['', 'The workload is closed-loop: CPU changes must be read together with RPS and errors. One run does not estimate run-to-run uncertainty or establish causality. Traces and HTTP diagnostics include file/logging costs; OS and Docker daemon remain shared. Smoke runs are diagnostics, not final measurements.', '']
    (root/'results.md').write_text('\n'.join(lines))
    if failures or any(s['integrity_issues'] for s in summaries):
        raise ValueError('Experiment integrity checks failed; see summary.json')
    return combined


generate = write_reports

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('directory', type=Path)
    write_reports(parser.parse_args().directory)
