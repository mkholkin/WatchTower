"""Request-weighted latency, fixed-window throughput, recovery and component resources."""
import argparse
from collections import Counter
from datetime import datetime
import hashlib
import json
import math
from pathlib import Path
import shutil
import statistics
import uuid

PERCENTILES = {'p50': .5, 'p75': .75, 'p90': .9, 'p95': .95, 'p99': .99}
RESOURCE_KEYS = ('cpu_percent', 'memory_bytes', 'read_bytes_s', 'write_bytes_s')
CPU_SETS = {'api': '0', 'postgres': '1', 'redis': '1', 'probe': '1', 'load': '2', 'collector': '3'}


def distribution(values):
    if not values:
        return None
    values = sorted(values)
    return {'count': len(values), 'min': values[0], 'max': values[-1], 'mean': statistics.mean(values),
            'median': statistics.median(values),
            **{key: values[math.ceil(len(values)*p)-1] for key, p in PERCENTILES.items()}}


def windows(requests, start, end, width=5):
    bins = []
    for index in range(math.ceil((end-start)/width)):
        left = start+index*width
        seconds = min(width, end-left)
        rows = [r for r in requests if left <= r['t'] < left+seconds]
        unexpected = sum(r['tags']['outcome'] != 'ok' for r in rows)
        good = sum(r['tags']['outcome'] == 'ok' and r['tags']['expected_error'] == 'false' for r in rows)
        latency = distribution([r['ms'] for r in rows])
        bins.append({'start': left, 'seconds': seconds, 'requests': len(rows), 'rps': len(rows)/seconds,
                     'good_rps': good/seconds, 'unexpected_ratio': unexpected/len(rows) if rows else None,
                     'p95_ms': latency['p95'] if latency else None})
    return bins


def healthy(row):
    return row['p95_ms'] is not None and row['p95_ms'] <= 500 and row['unexpected_ratio'] <= .01


def recovery(rows, baseline_rps):
    streak = []
    for row in rows:
        if healthy(row) and row['good_rps'] >= .8*baseline_rps and row['seconds'] == 5:
            streak.append(row)
        else:
            streak = []
        if len(streak) == 3:
            origin = rows[0]['start']
            return {'first_healthy_s': streak[0]['start']-origin, 'confirmed_s': row['start']+5-origin}
    return {'first_healthy_s': None, 'confirmed_s': None}


def read_jsonl(path):
    with path.open() as stream:
        for line in stream:
            yield json.loads(line)


def clock_step(rows):
    offsets = [r['clock_offset_s'] for r in rows if 'clock_offset_s' in r]
    return max((abs(b-a) for a, b in zip(offsets, offsets[1:])), default=None)


def read_requests(path):
    requests, epochs, journeys = [], [], Counter()
    for row in read_jsonl(path):
        if row['type'] != 'Point':
            continue
        data = row['data']
        if row['metric'] == 'scenario_epoch_ms':
            epochs.append(data['value']/1000)
        elif row['metric'] == 'user_request_ms':
            requests.append({'t': datetime.fromisoformat(data['time'].replace('Z', '+00:00')).timestamp(),
                             'ms': data['value'], 'tags': data['tags']})
        elif row['metric'] == 'user_journey_ms':
            journeys[data['tags']['outcome']] += 1
    if not epochs or not requests:
        raise ValueError('No user scenario observations')
    origin = min(epochs)
    for row in requests:
        row['t'] -= origin
    return requests, origin, dict(journeys)


def summarize(directory):
    raw = json.loads((directory/'k6-summary.json').read_text())
    cfg = raw['profile']
    requests, origin, journeys = read_requests(directory/'requests.jsonl')
    warmup = cfg['phases'][0]['seconds']
    measured = [r for r in requests if warmup <= r['t'] < cfg['duration']]
    timeline = windows(requests, warmup, cfg['duration'])
    unexpected = [r for r in measured if r['tags']['outcome'] != 'ok']
    intentional = [r for r in measured if r['tags']['expected_error'] == 'true']
    phases, start = [], 0
    for phase in cfg['phases']:
        selected = [r for r in requests if start <= r['t'] < start+phase['seconds']]
        bins = windows(requests, start, start+phase['seconds'])
        latency = distribution([r['ms'] for r in selected])
        ratio = sum(r['tags']['outcome'] != 'ok' for r in selected)/len(selected) if selected else 1
        phases.append({**phase, 'start': start, 'requests': len(selected), 'latency_ms': latency,
                       'unexpected_ratio': ratio, 'rps': distribution([b['rps'] for b in bins]),
                       'good_rps': distribution([b['good_rps'] for b in bins]), 'windows': bins,
                       'meets_slo': bool(latency and latency['p95'] <= 500 and ratio <= .01)})
        start += phase['seconds']
    resources = [r for r in read_jsonl(directory/'resources.jsonl') if origin+warmup <= r['time'] < origin+cfg['duration']]
    components = {}
    for name in CPU_SETS:
        rows = [r for r in resources if r['component'] == name]
        values = {key: distribution([r[key] for r in rows if r.get(key) is not None]) for key in RESOURCE_KEYS}
        memory = [r['memory_bytes'] for r in rows if r.get('memory_bytes') is not None]
        values['memory_first_last_bytes'] = [memory[0], memory[-1]] if memory else None
        values['sampling_errors'] = sum('error' in r for r in rows)
        values['max_sample_gap_s'] = max((b['time']-a['time'] for a, b in zip(rows, rows[1:])), default=None)
        values['observation_span_s'] = rows[-1]['time']-rows[0]['time'] if rows else 0
        values['max_clock_step_s'] = clock_step(rows)
        components[name] = values
    summary = {'name': cfg['name'], 'config': cfg, 'origin_epoch_s': origin, 'requests': len(measured),
               'journeys_including_warmup_cooldown': journeys,
               'latency_ms': distribution([r['ms'] for r in measured]),
               'successful_valid_latency_ms': distribution([r['ms'] for r in measured if r['tags']['outcome'] == 'ok' and r['tags']['expected_error'] == 'false']),
               'unexpected_requests': len(unexpected), 'unexpected_ratio': len(unexpected)/len(measured) if measured else None,
               'intentional_error_requests': len(intentional), 'intentional_error_percent': 100*len(intentional)/len(measured) if measured else None,
               'statuses': dict(Counter(r['tags']['status'] for r in measured)),
               'rps': distribution([b['rps'] for b in timeline]), 'good_rps': distribution([b['good_rps'] for b in timeline]),
               'windows': timeline, 'phases': phases, 'resources': components,
               'operations': {op: {'requests': len(rows), 'latency_ms': distribution([r['ms'] for r in rows]),
                                  'unexpected': sum(r['tags']['outcome'] != 'ok' for r in rows)}
                              for op in sorted({r['tags']['operation'] for r in measured})
                              if (rows := [r for r in measured if r['tags']['operation'] == op])}}
    if cfg['name'] == 'ramp':
        summary['stable_vus_observed'] = [p['vus'] for p in phases[1:] if p['meets_slo']]
        summary['overload_vus_observed'] = [p['vus'] for p in phases[1:] if not p['meets_slo']]
    if cfg['name'] == 'stress':
        baseline, overload, after = phases[1:]
        summary['recovery'] = {**recovery(after['windows'], baseline['good_rps']['median']),
                               'baseline_meets_slo': baseline['meets_slo'], 'overload_observed': not overload['meets_slo'],
                               'baseline_good_rps': baseline['good_rps']['median']}
    issues = []
    if any(not math.isfinite(r['ms']) or r['ms'] < 0 for r in requests):
        issues.append('Invalid request duration: negative or non-finite latency')
    if len(measured) < (1 if cfg['smoke'] else 1000):
        issues.append('Insufficient request sample: need at least 1000 in a full profile')
    if max(r['t'] for r in requests) < cfg['duration']-3:
        issues.append('Scenario stopped before the scheduled end')
    if max(r['t'] for r in requests) > cfg['duration']+13:
        issues.append('Request timestamps exceed duration plus graceful stop: host sleep or clock jump')
    for name, values in components.items():
        if any(values[key] is None for key in RESOURCE_KEYS):
            issues.append(f'{name}: incomplete CPU/RAM/disk counters')
        if values['sampling_errors'] or (values['max_sample_gap_s'] or 0) > 5:
            issues.append(f'{name}: monitoring gaps/errors')
        if values['observation_span_s'] < .8*(cfg['duration']-warmup):
            issues.append(f'{name}: monitoring does not cover the measurement interval')
        if values['max_clock_step_s'] is None or values['max_clock_step_s'] > .02:
            issues.append(f'{name}: clock step over 20 ms or missing clock validation')
    if cfg['name'].startswith('soak') and not cfg['smoke']:
        if cfg['phases'][1]['seconds'] < 300:
            issues.append('Soak must last at least 300 seconds')
        if abs(summary['intentional_error_percent']-cfg['errorPercent']) > .5:
            issues.append('Actual erroneous-operation share differs from the specified share')
    summary['integrity_issues'] = issues
    return summary, measured, resources


def plots_for(directory, summary, requests, resources):
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
    plots = directory/'plots'
    plots.mkdir(exist_ok=True)
    latency = sorted(r['ms'] for r in requests)
    fig, axes = plt.subplots(1, 3, figsize=(16, 4.5), constrained_layout=True)
    fig.suptitle(summary['name']+' — measured requests (warmup excluded)')
    axes[0].scatter([r['t'] for r in requests], [r['ms'] for r in requests], s=1, alpha=.15)
    axes[0].plot([b['start'] for b in summary['windows']], [b['p95_ms'] for b in summary['windows']], label='5 s p95', color='red')
    axes[0].set(xlabel='Scenario time (s)', ylabel='Full request latency (ms)', title='Latency vs time')
    axes[0].legend()
    axes[1].plot(latency, [(i+1)/len(latency)*100 for i in range(len(latency))])
    for key in PERCENTILES:
        axes[1].axvline(summary['latency_ms'][key], linestyle='--', alpha=.4, label=key)
    axes[1].set(xlabel='Latency (ms)', ylabel='Requests ≤ latency (%)', title='Request-weighted ECDF')
    axes[1].legend(fontsize='small')
    axes[2].hist(latency, bins=50)
    axes[2].set(xlabel='Latency (ms)', ylabel='Requests', title='Latency distribution')
    for ax in axes:
        ax.grid(alpha=.2)
    fig.savefig(plots/'latency.png', dpi=150)
    plt.close(fig)
    fig, axes = plt.subplots(1, 2, figsize=(12, 4), constrained_layout=True)
    bins = summary['windows']
    axes[0].plot([b['start'] for b in bins], [b['rps'] for b in bins], label='All requests')
    axes[0].plot([b['start'] for b in bins], [b['good_rps'] for b in bins], label='Successful valid requests')
    axes[0].set(xlabel='Scenario time (s)', ylabel='Requests/s', title='5-second throughput')
    axes[0].legend()
    axes[1].hist([b['rps'] for b in bins], bins=20)
    axes[1].set(xlabel='Requests/s', ylabel='5-second windows', title='Throughput distribution')
    fig.savefig(plots/'throughput.png', dpi=150)
    plt.close(fig)
    fig, axes = plt.subplots(len(CPU_SETS), 4, figsize=(16, 17), constrained_layout=True)
    for row, name in enumerate(CPU_SETS):
        selected = [r for r in resources if r['component'] == name]
        for column, (key, divisor, unit) in enumerate(zip(RESOURCE_KEYS, [1, 2**20, 1024, 1024], ['CPU % of one core', 'RAM MiB', 'Read KiB/s', 'Write KiB/s'])):
            pairs = [(r['time']-summary['origin_epoch_s'], r[key]/divisor) for r in selected if r.get(key) is not None]
            if pairs:
                axes[row, column].plot(*zip(*pairs), linewidth=.8)
            axes[row, column].set(title=f'{name}: {unit}', xlabel='Scenario time (s)')
            axes[row, column].grid(alpha=.2)
    fig.savefig(plots/'resources.png', dpi=130)
    plt.close(fig)


def write_reports(root):
    manifest = json.loads((root/'manifest.json').read_text())
    output = root/'allure-results'
    output.mkdir(exist_ok=True)
    for previous in output.glob('*-result.json'):
        previous.unlink()
    summaries, failures, baseline_config = [], [], None
    image_ids = [r['image_id'] for r in manifest['profiles']]
    if len(image_ids) != len(set(image_ids)):
        raise ValueError('Each profile must use a distinct Docker image')
    for record in manifest['profiles']:
        directory = root/'profiles'/record['name']
        try:
            if record['exit_code'] != 0 or record.get('error'):
                raise ValueError(f'Profile execution failed: {record.get("error") or record["exit_code"]}')
            summary, requests, resources = summarize(directory)
        except (OSError, ValueError, KeyError, TypeError) as error:
            failures.append({'name': record['name'], 'error': str(error)})
            item = {'uuid': str(uuid.uuid4()), 'name': record['name'], 'fullName': 'lab4.'+record['name'],
                    'historyId': hashlib.sha256(('lab4.'+record['name']).encode()).hexdigest(),
                    'status': 'broken', 'statusDetails': {'message': str(error)}, 'stage': 'finished',
                    'labels': [{'name': 'parentSuite', 'value': 'Lab 4 load testing'}]}
            (output/f'{item["uuid"]}-result.json').write_text(json.dumps(item)+'\n')
            continue
        configuration = {}
        for container in record['containers']:
            name = container['Config']['Labels']['com.docker.compose.service']
            host = container['HostConfig']
            configuration[name] = {key: host[key] for key in ('CpusetCpus', 'NanoCpus', 'Memory')}
            if host['CpusetCpus'] != CPU_SETS[name] or host['Memory'] <= 0 or host['NanoCpus'] <= 0:
                summary['integrity_issues'].append(f'{name}: invalid CPU isolation/resource limits')
            if container['State'].get('OOMKilled'):
                summary['integrity_issues'].append(f'{name}: OOM killed')
        if set(configuration) != set(CPU_SETS):
            summary['integrity_issues'].append('Missing container configuration')
        if baseline_config is not None and configuration != baseline_config:
            summary['integrity_issues'].append('Configuration differs across profiles')
        baseline_config = configuration
        summary['container_limits'] = configuration
        for container in record.get('final_containers', []):
            if container['State'].get('OOMKilled') or container['RestartCount']:
                summary['integrity_issues'].append(f'{container["Name"]}: OOM or unexpected restart')
        plots_for(directory, summary, requests, resources)
        detail = directory/'summary.json'
        detail.write_text(json.dumps(summary, indent=2)+'\n')
        attachments = []
        for source in [detail, *sorted((directory/'plots').glob('*.png'))]:
            target = record['name']+'-'+source.name
            shutil.copyfile(source, output/target)
            attachments.append({'name': source.name, 'source': target, 'type': 'image/png' if source.suffix == '.png' else 'application/json'})
        item = {'uuid': str(uuid.uuid4()), 'name': record['name'], 'fullName': 'lab4.'+record['name'],
                'historyId': hashlib.sha256(('lab4.'+record['name']).encode()).hexdigest(),
                'status': 'failed' if summary['integrity_issues'] else 'skipped' if manifest['smoke'] else 'passed',
                'statusDetails': {'message': '; '.join(summary['integrity_issues']) or ('Diagnostic smoke only' if manifest['smoke'] else 'Experiment data and isolation validated. Performance findings are in attachments; passed does not assert an SLO.')},
                'stage': 'finished', 'start': int(summary['origin_epoch_s']*1000),
                'stop': int((summary['origin_epoch_s']+summary['config']['duration'])*1000),
                'labels': [{'name': 'parentSuite', 'value': 'Lab 4 load testing'}, {'name': 'suite', 'value': record['name']}],
                'attachments': attachments}
        (output/f'{item["uuid"]}-result.json').write_text(json.dumps(item)+'\n')
        summaries.append(summary)
    combined = {'run_id': manifest['run_id'], 'smoke': manifest['smoke'], 'profiles': summaries,
                'failed_profiles': failures, 'report_source_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
    (root/'summary.json').write_text(json.dumps(combined, indent=2)+'\n')
    lines = ['# Lab 4 measured results', '', f'Run: `{manifest["run_id"]}`; smoke: `{manifest["smoke"]}`.', '',
             'All latencies include client blocking, connection setup and response transfer. Setup/warmup/cooldown excluded. Percentiles use nearest rank over individual requests. Throughput uses fixed 5-second windows, including zero-traffic windows.', '',
             '| Profile | Requests | Intentional errors % | Unexpected % | p50 ms | p75 | p90 | p95 | p99 | RPS median | RPS p95 |',
             '|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|']
    for s in summaries:
        lines.append(f'| {s["name"]} | {s["requests"]} | {s["intentional_error_percent"]:.2f} | {s["unexpected_ratio"]*100:.2f} | '+
                     ' | '.join(f'{s["latency_ms"][p]:.1f}' for p in PERCENTILES)+f' | {s["rps"]["median"]:.1f} | {s["rps"]["p95"]:.1f} |')
        if s['integrity_issues']:
            lines += ['', f'{s["name"]} integrity: '+', '.join(s['integrity_issues']), '']
    for failure in failures:
        lines += ['', f'Failed profile `{failure["name"]}`: {failure["error"]}', '']
    lines += ['', '## Load phases', '', '| Profile / phase | VUs | Seconds | Requests | p95 ms | Unexpected % | RPS median | SLO |', '|---|---:|---:|---:|---:|---:|---:|---|']
    for s in summaries:
        for p in s['phases'][1:]:
            lines.append(f'| {s["name"]} / {p["name"]} | {p["vus"]} | {p["seconds"]} | {p["requests"]} | {p["latency_ms"]["p95"] if p["latency_ms"] else "n/a"} | {p["unexpected_ratio"]*100:.2f} | {p["rps"]["median"]:.1f} | {p["meets_slo"]} |')
    for s in summaries:
        if 'recovery' in s:
            lines += ['', 'Recovery evidence: `'+json.dumps(s['recovery'])+'`.', '']
    lines += ['', '## Resources', '', 'CPU: percent of one core. RAM: working set, excluding inactive file cache. Disk: cgroup block I/O deltas (cache hits do not count as physical reads). GPU is not used.', '',
              '| Profile / component | CPU min / median / max % | RAM min / median / max MiB | Read min / median / max KiB/s | Write min / median / max KiB/s |', '|---|---|---|---|---|']
    for s in summaries:
        for name, resource in s['resources'].items():
            cells = []
            for key, divisor in zip(RESOURCE_KEYS, [1, 2**20, 1024, 1024]):
                values = resource[key]
                cells.append(' / '.join(f'{values[p]/divisor:.2f}' for p in ('min', 'median', 'max')) if values else 'unavailable')
            lines.append(f'| {s["name"]} / {name} | '+' | '.join(cells)+' |')
    lines += ['', 'Raw data, per-operation and per-phase summaries, and graphs are under `profiles/<name>/`. SLO: p95 ≤ 500 ms and unexpected responses ≤ 1%. Recovery requires three consecutive 5-second SLO-compliant windows at ≥ 80% of baseline successful RPS. A null recovery time means recovery was not confirmed within the observation interval.', '',
              'These observations describe a closed user workload on this machine. Five minutes can expose short-term resource trends but cannot prove absence of long-term leaks. A single run per load profile does not estimate between-run variability.', '']
    (root/'results.md').write_text('\n'.join(lines))
    print(f'Report saved: {root}', flush=True)
    if failures or any(s['integrity_issues'] for s in summaries):
        raise ValueError('Experiment integrity checks failed; see summary.json')
    return combined


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('directory', type=Path)
    write_reports(parser.parse_args().directory)
