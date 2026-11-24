import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from tools.lab6.report import compare, container_configuration, resource_summary, trace_summary, write_reports


def row(stamp, cpu, memory=100, component='api'):
    return {'time': stamp, 'component': component, 'memory_bytes': memory,
            'raw': {'cpu_stats': {'cpu_usage': {'total_usage': cpu}}}}


def span(name='GET /monitors/:id', **extra):
    return dict(trace_id='a'*32, span_id='b'*16, name=name, started_ns=2_000_000_000,
                duration_ns=1000, status=200, **extra)


class ReportTests(unittest.TestCase):
    def test_cpu_window_uses_raw_counter_and_interpolates_boundaries(self):
        rows = [row(0, 0, 9999), row(2, 2e9, 20), row(4, 6e9, 40), row(6, 10e9, 60)]
        result = resource_summary(rows, 'api', 1, 5)
        self.assertEqual(result['cpu_seconds'], 7)
        self.assertEqual(result['memory_bytes'], {'min': 20, 'median': 30, 'max': 40})
        self.assertEqual(result['mean_cpu_percent'], 175)

    def test_counter_reset_after_end_bracket_does_not_invalidate_window(self):
        rows = [row(0, 0), row(2, 2e9), row(4, 4e9), row(6, 6e9), row(7, 0)]
        result = resource_summary(rows, 'api', 1, 5)
        self.assertEqual(result['cpu_seconds'], 4)
        rows[2] = row(4, 0)
        with self.assertRaisesRegex(ValueError, 'counter reset'):
            resource_summary(rows, 'api', 1, 5)

    def test_missing_counters_reset_gap_and_invalid_memory_are_rejected(self):
        cases = [[row(1, 1e9), row(2, 2e9)],
                 [row(0, 2e9), row(2, 1e9), row(4, 3e9)],
                 [row(0, 0), row(8, 2e9)],
                 [row(0, 0), row(2, None), row(4, 4e9)],
                 [row(0, 0), row(2, 2e9, -1), row(4, 4e9)]]
        for rows in cases:
            with self.subTest(rows=rows), self.assertRaises(ValueError):
                resource_summary(rows, 'api', 1, 3)

    def test_delta_includes_collector_and_zero_cpu_baseline(self):
        def summary(cpu, memory, rps):
            return {'resources': {c: {'cpu_seconds': cpu, 'memory_bytes': dict(min=memory, median=memory, max=memory)}
                                  for c in ('api', 'collector')},
                    'load': {'rps': {'median': rps}, 'unexpected_ratio': 0}}
        result = compare(summary(2, 300, 10), summary(0, 100, 9))
        self.assertEqual(result['collector']['cpu_seconds_delta'], 2)
        self.assertIsNone(result['api']['cpu_seconds_delta_percent'])
        self.assertEqual(result['api']['memory_bytes_delta']['median'], 200)
        self.assertEqual(result['rps_median_delta'], 1)

    def test_sanitized_unmatched_route_is_a_valid_debug_span(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'traces.jsonl'
            event = span(name='GET unmatched')
            event['status'] = 404
            path.write_text(json.dumps(event)+'\n')
            self.assertEqual(trace_summary(path, True, 1, 3)['measurement_events'], 1)

    def test_resource_isolation_and_restart_validation(self):
        from tools.lab6.report import CPU_SETS
        containers = [{'Config': {'Labels': {'com.docker.compose.service': name}},
                       'HostConfig': {'CpusetCpus': cpu, 'NanoCpus': 250000000, 'Memory': 1024}}
                      for name, cpu in CPU_SETS.items()]
        self.assertEqual(container_configuration({'containers': containers})[1], [])
        containers[0]['HostConfig']['CpusetCpus'] = '99'
        issues = container_configuration({'containers': containers,
                                          'final_containers': [{'RestartCount': 1}]})[1]
        self.assertEqual(len(issues), 2)
        self.assertIn('Missing container configuration', container_configuration({})[1])

    def test_valid_smoke_generates_comparisons_and_skipped_allure(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            profiles = []
            for variant in ('baseline', 'debug'):
                directory = root/variant
                directory.mkdir()
                rows = [row(t, t*1e9, component=c) for c in ('api', 'collector', 'otel-collector', 'tempo') for t in range(5)]
                (directory/'resources.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in rows))
                profiles.append(dict(name='ramp', variant=variant, repetition=1, path=variant, exit_code=0))
            (root/'manifest.json').write_text(json.dumps(dict(run_id='smoke', smoke=True, profiles=profiles)))
            load = dict(origin_epoch_s=0, config={'phases': [{'seconds': 1}], 'duration': 3},
                        integrity_issues=[], rps={'median': 10}, unexpected_ratio=0, intentional_error_percent=0)
            with patch('tools.lab6.report.lab4.summarize', return_value=(load, [], [])), \
                 patch('tools.lab6.report.container_configuration', return_value=({'api': {}}, [])), \
                 patch('tools.lab6.report.plot_comparison'), patch('tools.lab6.report.export_integrity'):
                result = write_reports(root)
            self.assertEqual(len(result['profiles']), 2)
            self.assertEqual(result['profiles'][1]['baseline_delta']['collector']['cpu_seconds_delta'], 0)
            self.assertTrue((root/'results.md').exists())
            items = [json.loads(p.read_text()) for p in (root/'allure-results').glob('*-result.json')]
            self.assertEqual([i['status'] for i in items], ['skipped', 'skipped'])

    def test_missing_traces_and_events_outside_window_fail(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'traces.jsonl'
            self.assertEqual(trace_summary(path, False, 1, 3)['events'], 0)
            with self.assertRaises(ValueError):
                trace_summary(path, True, 1, 3)
            path.write_text(json.dumps(span())+'\n')
            with self.assertRaises(ValueError):
                trace_summary(path, True, 3, 4)
            with self.assertRaises(ValueError):
                trace_summary(path, False, 1, 3)
            self.assertEqual(trace_summary(path, True, 1, 3)['measurement_events'], 1)
            chrome = json.loads(path.with_name('trace.json').read_text())['traceEvents'][0]
            self.assertEqual(chrome['dur'], 1)
            self.assertIsInstance(chrome['tid'], int)
            self.assertLess(chrome['tid'], 2**48)
            self.assertEqual(chrome['args']['trace_id'], 'a'*32)
            self.assertEqual(chrome['name'], 'GET /monitors/:id')

    def test_trace_secrets_and_raw_identifiers_rejected(self):
        events = [span('GET /monitors?token=abc'), span('GET /users/person@example.com'),
                  span('GET /monitors/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee'),
                  span('postgres SELECT * FROM users'), span(body='password'), span(headers={'Authorization': 'secret'})]
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'traces.jsonl'
            for event in events:
                with self.subTest(event=event):
                    path.write_text(json.dumps(event)+'\n')
                    with self.assertRaises(ValueError):
                        trace_summary(path, True, 1, 3)

    def test_failed_run_leaves_broken_allure_and_json_evidence(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root/'manifest.json').write_text(json.dumps({'run_id': 'failed', 'smoke': True,
                'profiles': [{'name': 'ramp', 'variant': 'baseline', 'exit_code': 1}]}))
            with patch('tools.lab6.report.plot_comparison'), patch('tools.lab6.report.export_integrity'):
                for _ in range(2):
                    with self.assertRaises(ValueError):
                        write_reports(root)
            items = [json.loads(p.read_text()) for p in (root/'allure-results').glob('*-result.json')]
            self.assertEqual(len(items), 1)
            self.assertEqual(items[0]['status'], 'broken')
            self.assertEqual(len(json.loads((root/'summary.json').read_text())['failed_profiles']), 1)


def otlp_export():
    def attributes(values):
        return [{'key': key, 'value': {'intValue': str(value)} if isinstance(value, int) else {'stringValue': value}}
                for key, value in values.items()]
    http = dict(traceId='a'*32, spanId='b'*16, name='GET /monitors/:id', kind=2,
                startTimeUnixNano='2000000000', endTimeUnixNano='2000010000', status={'code': 1},
                attributes=attributes({'http.request.method': 'GET', 'http.route': '/monitors/:id', 'http.response.status_code': 200}))
    sql = dict(traceId='a'*32, spanId='c'*16, parentSpanId='b'*16, name='postgres SELECT', kind=3,
               startTimeUnixNano='2000001000', endTimeUnixNano='2000009000', status={},
               attributes=attributes({'db.system.name': 'postgresql', 'db.operation.name': 'SELECT'}))
    return {'resourceSpans': [{'resource': {'attributes': attributes({'service.name': 'watchtower-lab6'})},
                               'scopeSpans': [{'scope': {'name': 'watchtower'}, 'spans': [http, sql]}]}]}


class NativeOTLPTests(unittest.TestCase):
    def test_native_export_derives_display_artifact_and_preserves_ids(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'traces-otlp.jsonl'
            path.write_text(json.dumps(otlp_export())+'\n')
            result = trace_summary(path, True, 1, 3)
            self.assertEqual(result['events'], 2)
            events = [json.loads(line) for line in path.with_name('traces.jsonl').read_text().splitlines()]
            self.assertEqual(events[1]['parent_id'], events[0]['span_id'])
            self.assertEqual(events[0]['span_id'], 'b'*16)

    def test_startup_sql_root_is_allowed_alongside_correlated_request(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'traces-otlp.jsonl'
            data = otlp_export()
            spans = data['resourceSpans'][0]['scopeSpans'][0]['spans']
            root = dict(spans[1], traceId='d'*32, spanId='e'*16)
            del root['parentSpanId']
            spans.append(root)
            path.write_text(json.dumps(data)+'\n')
            self.assertEqual(trace_summary(path, True, 1, 3)['events'], 3)
            spans[:] = [root]
            path.write_text(json.dumps(data)+'\n')
            with self.assertRaisesRegex(ValueError, 'correlated HTTP'):
                trace_summary(path, True, 1, 3)

    def test_malformed_unsafe_or_incomplete_native_exports_fail(self):
        mutations = [lambda s: s[0]['attributes'][1].update(value={'stringValue': '/users/:id'}),
                     lambda s: s[0]['attributes'][1].update(value={'stringValue': '/monitors?token=secret'}),
                     lambda s: s[0].update(spanId='0'*16),
                     lambda s: s[0].update(spanId='b'*32),
                     lambda s: s[1].update(parentSpanId='d'*16),
                     lambda s: s[1].update(endTimeUnixNano='2000020000'),
                     lambda s: s[1].update(status={'code': 500}),
                     lambda s: s[1].update(events=[{'name': 'password'}]),
                     lambda s: s[0]['attributes'].append({'key': 'http.url', 'value': {'stringValue': 'secret'}}),
                     lambda s: s.append(dict(s[0]))]
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'traces-otlp.jsonl'
            for mutation in mutations:
                data = otlp_export()
                mutation(data['resourceSpans'][0]['scopeSpans'][0]['spans'])
                path.write_text(json.dumps(data)+'\n')
                with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                    trace_summary(path, True, 1, 3)
            path.write_text('{"resourceSpans": [')
            with self.assertRaises(ValueError):
                trace_summary(path, True, 1, 3)

    def test_export_evidence_rejects_lost_spans_and_collector_errors(self):
        from tools.lab6.report import export_integrity
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            evidence = dict(ended_spans=2, exported_spans=2, export_errors=0)
            (directory/'export-summary.json').write_text(json.dumps(evidence))
            (directory/'tempo-live-check.json').write_text(json.dumps(dict(tempo_live_verified=True, unique_spans=2)))
            metrics = 'otelcol_receiver_accepted_spans_total{receiver="otlp"} 2\notelcol_exporter_sent_spans_total{exporter="otlphttp/tempo"} 2\n'
            (directory/'otel-collector-metrics.txt').write_text(metrics)
            export_integrity(directory, {'events': 2}, True)
            with self.assertRaisesRegex(ValueError, 'SDK'):
                export_integrity(directory, {'events': 1}, True)
            (directory/'otel-collector-metrics.txt').write_text(metrics+'otelcol_exporter_send_failed_spans_total 1\n')
            with self.assertRaisesRegex(ValueError, 'Collector refused'):
                export_integrity(directory, {'events': 2}, True)


if __name__ == '__main__':
    unittest.main()
