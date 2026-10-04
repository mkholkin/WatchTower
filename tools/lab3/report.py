"""Validate independent trials; export statistics, standalone plots and Allure."""
import argparse
import csv
from datetime import datetime
import hashlib
import json
import math
from pathlib import Path
import shutil
import statistics
import uuid

PERCENTILES = {'p50': .5, 'p75': .75, 'p90': .9, 'p95': .95, 'p99': .99}
CASE_IDS = {f'{lib}/{scenario}/{size}' for lib in ('encoding-json', 'goccy-go-json')
            for scenario in ('encode', 'decode', 'roundtrip') for size in (1, 32)}


def percentile(values, fraction):
    """Nearest rank: the smallest observation covering the requested fraction."""
    if not values or not 0 < fraction <= 1:
        raise ValueError('nonempty observations and 0 < percentile <= 1 required')
    return sorted(values)[math.ceil(len(values) * fraction) - 1]


def distribution(values):
    return {'min': min(values), 'max': max(values), 'mean': statistics.mean(values),
            **{name: percentile(values, fraction) for name, fraction in PERCENTILES.items()}}


def summarize(trials, minimum=100, expected_cases=None):
    expected_cases = CASE_IDS if expected_cases is None else expected_cases
    if len(trials) < minimum:
        raise ValueError(f'Need at least {minimum} independent trials; got {len(trials)}')
    ids = [t['trial'] for t in trials]
    if sorted(ids) != list(range(1, len(trials) + 1)):
        raise ValueError('Trial numbers must be unique and contiguous from 1')
    same = ('config', 'go_version', 'goos', 'goarch', 'gomaxprocs', 'dependencies')
    if trials[0]['gomaxprocs'] != 1:
        raise ValueError('GOMAXPROCS must be 1')
    cfg = trials[0]['config']
    groups = {key: [] for key in expected_cases}
    for trial in trials:
        if any(trial[key] != trials[0][key] for key in same):
            raise ValueError('Trials have different configuration or runtime')
        cases = trial['cases']
        if len(cases) != len(expected_cases) or {c['id'] for c in cases} != expected_cases:
            raise ValueError('Missing or duplicate benchmark cases')
        for c in cases:
            expected_ops = {'warmup': cfg['warmup'], 'measurement': cfg['batches'] * cfg['batch_size'],
                            'latency': cfg['latency_samples'], 'teardown': cfg['teardown']}
            if (any(c[k]['operations'] != n or c[k]['elapsed_ns'] <= 0 for k, n in expected_ops.items())
                    or c['total_operations'] != sum(expected_ops.values())
                    or len(c['latencies_ns']) != cfg['latency_samples'] or len(c['batches']) != cfg['batches']
                    or any(n <= 0 for n in c['latencies_ns'])
                    or any(b['ns_per_op'] <= 0 for b in c['batches']) or c['duration_ns'] <= 0):
                raise ValueError('Invalid timing samples or operation accounting')
            groups[c['id']].append(c)
    rows = []
    for key, cases in sorted(groups.items()):
        first = cases[0]
        latencies = [n for c in cases for n in c['latencies_ns']]
        batches = [b['ns_per_op'] for c in cases for b in c['batches']]
        means = [c['measurement']['elapsed_ns'] / c['measurement']['operations'] for c in cases]
        rows.append({'id': key, 'library': first['library'], 'scenario': first['scenario'], 'size': first['size'],
                     'payload_bytes': first['payload_bytes'], 'trials': len(cases), 'requests_sampled': len(latencies),
                     'operations': {p: sum(c[p]['operations'] for c in cases) for p in ('warmup', 'measurement', 'latency', 'teardown')},
                     'duration_ns': sum(c['duration_ns'] for c in cases), 'latency_ns': distribution(latencies),
                     'batch_ns_per_op': distribution(batches), 'trial_mean_ns_per_op': distribution(means)})
    overhead = distribution([t['overhead_ns_per_op'] for t in trials])
    amortized = distribution([t['overhead_ns_per_op'] + t['timer_pair_ns_per_op'] / t['config']['batch_size']
                             for t in trials])
    timer = distribution([n for t in trials for n in t['timer_probe_ns']])
    return {'trials': len(trials), 'runtime': {k: trials[0][k] for k in same}, 'cases': rows,
            'duration_ns': sum(t['duration_ns'] for t in trials), 'overhead_ns_per_op': overhead,
            'timer_probe_ns': timer,
            'timer_pair_ns_per_op': distribution([t['timer_pair_ns_per_op'] for t in trials]),
            'amortized_overhead_ns_per_op': amortized,
            'warnings': (['Median harness overhead exceeds the ideal 5 ns; results are not overhead-subtracted.']
                         if amortized['p50'] > 5 else []) +
                        [f"{c['id']}: median operation outside recommended 100 ns–50 ms interval"
                         for c in rows if not 100 <= c['trial_mean_ns_per_op']['p50'] <= 50_000_000]}


def write_reports(root, smoke=False):
    trials = [json.loads(p.read_text()) for p in sorted((root / 'raw').glob('trial-*.json'))]
    summary = summarize(trials, minimum=1 if smoke else 100)
    manifest = json.loads((root / 'manifest.json').read_text())
    if manifest['exit_code'] or manifest['requested_trials'] != len(trials):
        raise ValueError('Experiment did not complete all requested trials')
    if manifest['mode'] == 'docker':
        records = manifest['trials']
        if (len(records) != len(trials) or len({r['image'] for r in records}) != len(trials)
                or len({r['container'] for r in records}) != len(trials)):
            raise ValueError('Each trial needs a separate image and container')
        for r in records:
            h = r['host_config']
            if (r['exit_code'] != 0 or h != records[0]['host_config'] or r['env'] != records[0]['env']
                    or h['CpusetCpus'] != '0' or h['NanoCpus'] != 1_000_000_000
                    or h['Memory'] != 512 * 1024 * 1024 or h['NetworkMode'] != 'none'):
                raise ValueError('Trial isolation/configuration does not match the protocol')
    summary['execution'] = manifest
    (root / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    with (root / 'summary.csv').open('w', newline='') as f:
        writer = csv.writer(f, lineterminator='\n')
        writer.writerow(['case', 'trials', 'requests_sampled', 'warmup_ops', 'measurement_ops', 'latency_ops', 'teardown_ops',
                         'duration_s', 'trial_median_ns_op', 'request_p50_ns', 'request_p75_ns', 'request_p90_ns', 'request_p95_ns', 'request_p99_ns'])
        for c in summary['cases']:
            writer.writerow([c['id'], c['trials'], c['requests_sampled'], *c['operations'].values(), c['duration_ns']/1e9,
                             c['trial_mean_ns_per_op']['p50'], *[c['latency_ns'][p] for p in PERCENTILES]])
    with (root / 'samples.csv').open('w', newline='') as f:
        writer = csv.writer(f, lineterminator='\n'); writer.writerow(['trial', 'case', 'kind', 'index', 'value_ns'])
        for t in trials:
            for c in t['cases']:
                for i, b in enumerate(c['batches']): writer.writerow([t['trial'], c['id'], 'batch_mean', i, b['ns_per_op']])
                for i, n in enumerate(c['latencies_ns']): writer.writerow([t['trial'], c['id'], 'request_latency', i, n])
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
    plots = root / 'plots'; plots.mkdir(exist_ok=True)
    results = root / 'allure-results'; results.mkdir(exist_ok=True)
    origin = min(datetime.fromisoformat(c['started_at'].replace('Z', '+00:00')).timestamp() for t in trials for c in t['cases'])
    for row in summary['cases']:
        key = row['id']; slug = key.replace('/', '-')
        observations = [(t['trial'], c) for t in trials for c in t['cases'] if c['id'] == key]
        latencies = sorted(n / 1000 for _, c in observations for n in c['latencies_ns'])
        timeline = sorted((datetime.fromisoformat(c['started_at'].replace('Z', '+00:00')).timestamp() - origin + b['offset_ns']/1e9,
                           b['ns_per_op']/1000) for _, c in observations for b in c['batches'])
        fig, axes = plt.subplots(1, 3, figsize=(16, 4.5), constrained_layout=True)
        fig.suptitle(f"{key} — {len(observations)} independent trials")
        axes[0].plot([x for x,y in timeline], [y for x,y in timeline], linewidth=.6)
        axes[0].set(xlabel='Elapsed experiment time (s)', ylabel='Batch mean (µs/op)', title='Time series')
        axes[1].plot(latencies, [(i+1)/len(latencies)*100 for i in range(len(latencies))])
        axes[1].set(xlabel='Request latency incl. timer (µs)', ylabel='Requests ≤ latency (%)', title='Empirical CDF', xscale='log')
        for name in PERCENTILES:
            axes[1].axvline(row['latency_ns'][name]/1000, alpha=.4, linestyle='--', label=name)
        axes[1].legend(fontsize='small')
        low, high = min(latencies), max(latencies)
        bins = [low * (max(high, low*1.01)/low)**(i/40) for i in range(41)]
        axes[2].hist(latencies, bins=bins)
        axes[2].set(xlabel='Request latency incl. timer (µs)', ylabel='Requests', title='Latency histogram', xscale='log')
        for ax in axes: ax.grid(alpha=.2)
        for ext in ('png', 'svg'): fig.savefig(plots / f'{slug}.{ext}', dpi=160)
        plt.close(fig)
        attachment = f'{slug}.png'; shutil.copyfile(plots / attachment, results / attachment)
        detail = f'{slug}.json'; (results / detail).write_text(json.dumps(row, indent=2)+'\n')
        isolated = manifest['mode'] == 'docker' and not smoke
        started = min(datetime.fromisoformat(c['started_at'].replace('Z', '+00:00')).timestamp() for _, c in observations)
        stopped = max(datetime.fromisoformat(c['started_at'].replace('Z', '+00:00')).timestamp()+c['duration_ns']/1e9 for _,c in observations)
        item = {'uuid': str(uuid.uuid4()), 'name': key, 'fullName': 'lab3.'+key,
                'historyId': hashlib.sha256(key.encode()).hexdigest(), 'status': 'passed' if isolated else 'skipped',
                'statusDetails': {'message': '100+ isolated trials validated; timings are measurements, not speed assertions.' if isolated else 'Diagnostic run: Docker isolation or 100-trial requirement not met.'},
                'stage': 'finished', 'start': int(started*1000), 'stop': int(stopped*1000),
                'labels': [{'name': 'parentSuite', 'value': 'Lab 3 microbenchmarks'}, {'name': 'suite', 'value': row['scenario']}],
                'parameters': [{'name': k, 'value': str(row[k])} for k in ('library','size','trials','requests_sampled')],
                'attachments': [{'name': 'Time series, ECDF, histogram', 'source': attachment, 'type': 'image/png'},
                                {'name': 'Statistics and phase counts', 'source': detail, 'type': 'application/json'}]}
        (results / f"{item['uuid']}-result.json").write_text(json.dumps(item)+'\n')
    lines = ['# Lab 3 measured results', '', f"Mode: {manifest['mode']}; trials: {summary['trials']}; workload time: {summary['duration_ns']/1e9:.3f} s.", '',
             f"Harness overhead p50: {summary['overhead_ns_per_op']['p50']:.3f} ns/op; timer-pair cost p50: {summary['timer_pair_ns_per_op']['p50']:.1f} ns.", '',
             f"Amortized loop + timer overhead p50: {summary['amortized_overhead_ns_per_op']['p50']:.3f} ns/op.", '',
             'Request percentiles include timer overhead. Batch means and single-request latencies are separate distributions. Warmup and teardown are excluded.', '',
             '| Case | Median batch ns/op per trial | Request p50 | p75 | p90 | p95 | p99 (ns) |', '|---|---:|---:|---:|---:|---:|---:|']
    for c in summary['cases']:
        lines.append('| '+c['id']+' | '+f"{c['trial_mean_ns_per_op']['p50']:.2f}"+' | '+' | '.join(f"{c['latency_ns'][p]:.0f}" for p in PERCENTILES)+' |')
    lines += ['', '## Paired comparison', '', '| Scenario | Size | Median stdlib / go-json ratio |', '|---|---:|---:|']
    for scenario in ('encode','decode','roundtrip'):
        for size in (1,32):
            ratios=[]
            for t in trials:
                means={c['library']: c['measurement']['elapsed_ns']/c['measurement']['operations'] for c in t['cases'] if c['scenario']==scenario and c['size']==size}
                ratios.append(means['encoding-json']/means['goccy-go-json'])
            lines.append(f'| {scenario} | {size} | {statistics.median(ratios):.3f} |')
    lines += ['', 'Ratios > 1 favor go-json on this workload and machine only. No universal performance claim.', '', *summary['warnings'], '']
    (root / 'results.md').write_text('\n'.join(lines))
    print(f"Report: {root}; {summary['trials']} trials, {len(summary['cases'])} cases", flush=True)
    return summary


if __name__ == '__main__':
    parser=argparse.ArgumentParser();parser.add_argument('directory',type=Path);parser.add_argument('--smoke',action='store_true')
    args=parser.parse_args();write_reports(args.directory,args.smoke)
