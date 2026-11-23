import json
from pathlib import Path
import tempfile
import unittest

from tools.lab8.report import collect


class ReportTests(unittest.TestCase):
    def test_preparation_failure_is_failed_in_allure_and_skips_unstarted_phases(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            result = collect(root, dict(run_id='failed-build', action='all', smoke=False,
                                        contours=[], error='image build failed'))
            self.assertEqual(result['status'], 'failed')
            self.assertEqual([p['status'] for p in result['phases']], ['skipped']*3)
            records = [json.loads(p.read_text()) for p in (root/'allure-results').glob('*-result.json')]
            self.assertEqual(sum(r['status'] == 'failed' for r in records), 1)

    def test_smoke_load_is_skipped_and_source_allure_attachments_are_preserved(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root/'load/allure-results'
            source.mkdir(parents=True)
            (source/'measurement.txt').write_text('raw evidence')
            result = collect(root, dict(run_id='smoke', action='load', mode='mock', smoke=True,
                contours=[dict(kind='load', mode='mock', exit_code=0, cleanup=True, rdap_evidence={})]))
            self.assertEqual(result['phases'][0]['status'], 'skipped')
            self.assertEqual((root/'allure-results/measurement.txt').read_text(), 'raw evidence')
            self.assertTrue((root/'results.md').exists())


if __name__ == '__main__':
    unittest.main()
