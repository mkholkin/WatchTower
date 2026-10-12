import unittest
import json
from pathlib import Path
import tempfile

from tools.lab4.report import distribution, windows, recovery, write_reports, clock_step


class StatisticsTests(unittest.TestCase):
    def test_clock_correction_is_detected_independently_of_request_outcomes(self):
        self.assertAlmostEqual(clock_step([{'clock_offset_s': 100}, {'clock_offset_s': 99.9}]), .1)
        self.assertEqual(clock_step([{'clock_offset_s': 100}, {'clock_offset_s': 100}]), 0)
        self.assertIsNone(clock_step([{}]))

    def test_nearest_rank_percentiles(self):
        row = distribution(list(range(1, 101)))
        self.assertEqual([row[k] for k in ('p50', 'p75', 'p90', 'p95', 'p99')], [50, 75, 90, 95, 99])
        self.assertIsNone(distribution([]))

    def test_zero_traffic_bins_and_expected_errors(self):
        requests = [{'t': 1, 'ms': 10, 'tags': {'outcome': 'ok', 'expected_error': 'false'}},
                    {'t': 2, 'ms': 20, 'tags': {'outcome': 'ok', 'expected_error': 'true'}},
                    {'t': 3, 'ms': 30, 'tags': {'outcome': 'unexpected', 'expected_error': 'false'}}]
        bins = windows(requests, 0, 10)
        self.assertEqual(bins[0]['rps'], .6)
        self.assertEqual(bins[0]['good_rps'], .2)
        self.assertEqual(bins[0]['unexpected_ratio'], 1/3)
        self.assertEqual(bins[1]['rps'], 0)
        self.assertIsNone(bins[1]['p95_ms'])

    def test_recovery_requires_three_consecutive_healthy_windows(self):
        good = {'good_rps': 9, 'unexpected_ratio': 0, 'p95_ms': 100}
        rows = [dict(good, start=i*5, seconds=5) for i in range(5)]
        rows[1]['unexpected_ratio'] = .1
        result = recovery(rows, 10)
        self.assertEqual(result['first_healthy_s'], 10)
        self.assertEqual(result['confirmed_s'], 25)
        self.assertIsNone(recovery(rows[:4], 10)['confirmed_s'])

    def test_failed_and_unreadable_profiles_both_leave_allure_evidence(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            manifest = {'run_id': 'failure-test', 'smoke': True, 'profiles': [
                {'name': 'ramp', 'exit_code': 1, 'image_id': 'a', 'error': 'k6 exited 1'},
                {'name': 'stress', 'exit_code': 0, 'image_id': 'b'}]}
            (root/'manifest.json').write_text(json.dumps(manifest))
            for _ in range(2):
                with self.assertRaises(ValueError):
                    write_reports(root)
            items = [json.loads(p.read_text()) for p in (root/'allure-results').glob('*-result.json')]
            self.assertEqual({item['name'] for item in items}, {'ramp', 'stress'})
            self.assertTrue(all(item['status'] == 'broken' for item in items))
            self.assertEqual(len(json.loads((root/'summary.json').read_text())['failed_profiles']), 2)


if __name__ == '__main__':
    unittest.main()
