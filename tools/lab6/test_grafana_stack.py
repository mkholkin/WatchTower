import json
from pathlib import Path
import tempfile
import unittest

from tools.lab6.grafana_stack import prepare


class GrafanaImportTests(unittest.TestCase):
    def fixture(self, root):
        profile = dict(name='ramp', variant='trace', repetition=1, artifact_path='profile',
                       measurement_start=100, measurement_end=110, integrity_issues=[],
                       resources={name: dict(cpu_seconds=2, memory_bytes=dict(min=10, median=20, max=30))
                                  for name in ('api', 'collector')},
                       load=dict(rps=dict(median=36), unexpected_ratio=0))
        summary = dict(run_id='test-run', failed_profiles=[], profiles=[profile])
        (root/'summary.json').write_text(json.dumps(summary))
        (root/'profile').mkdir()
        rows = [dict(time=t, component='api', memory_bytes=128,
                     raw=dict(cpu_stats=dict(cpu_usage=dict(total_usage=3_000_000_000))))
                for t in (99, 101, 111)]
        (root/'profile/resources.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in rows))
        return summary

    def test_import_preserves_time_and_raw_cpu_and_filters_warmup(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            meta = prepare(root, root/'output')
            data = (root/'output/metrics.om').read_text()
            self.assertIn('lab6_cpu_seconds_total{component="api",profile="ramp",repetition="1",run="test-run",variant="trace"} 3 101.000000', data)
            self.assertNotIn('99.000000', data)
            self.assertNotIn('111.000000', data)
            self.assertEqual(meta['samples'], 12)
            dashboard = json.loads((root/'output/dashboards/lab6.json').read_text())
            self.assertEqual(dashboard['time']['from'], '1970-01-01T00:01:25Z')
            self.assertTrue(data.endswith('# EOF\n'))

    def test_failed_experiment_and_escaping_artifacts_are_rejected(self):
        for change in ('failed_profiles', 'artifact_path'):
            with self.subTest(change=change), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                summary = self.fixture(root)
                if change == 'failed_profiles':
                    summary['failed_profiles'] = ['ramp']
                else:
                    summary['profiles'][0]['artifact_path'] = '../escape'
                (root/'summary.json').write_text(json.dumps(summary))
                with self.assertRaises(ValueError):
                    prepare(root, root/'output')


if __name__ == '__main__':
    unittest.main()
