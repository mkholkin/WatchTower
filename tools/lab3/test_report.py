"""Check statistical definitions and reject incomparable/incomplete experiments."""
import copy
import unittest
from tools.lab3.report import percentile, summarize


def trial(number):
    phase = {'operations': 2, 'elapsed_ns': 2000}
    return {'trial': number, 'config': {'warmup': 2, 'batches': 1, 'batch_size': 2, 'latency_samples': 2, 'teardown': 2},
            'go_version': 'go1.26.3', 'goos': 'linux', 'goarch': 'arm64', 'gomaxprocs': 1, 'dependencies': {},
            'duration_ns': 9000, 'overhead_ns_per_op': 2, 'timer_probe_ns': [20, 30], 'timer_pair_ns_per_op': 30,
            'cases': [{'id': 'encoding-json/encode/1', 'library': 'encoding-json', 'scenario': 'encode', 'size': 1,
                       'payload_bytes': 100, 'started_at': '2026-01-01T00:00:00Z', 'duration_ns': 8000,
                       'warmup': phase, 'measurement': phase, 'latency': phase, 'teardown': phase,
                       'total_operations': 8, 'latencies_ns': [100, 1900], 'batches': [{'offset_ns': 0, 'ns_per_op': 1000}]}]}


class StatisticsTests(unittest.TestCase):
    def test_nearest_rank_percentiles(self):
        self.assertEqual([percentile(list(range(1, 101)), p) for p in (.5, .75, .9, .95, .99)], [50, 75, 90, 95, 99])
        with self.assertRaises(ValueError): percentile([], .5)

    def test_request_quantiles_are_not_quantiles_of_trial_means(self):
        s = summarize([trial(n) for n in range(1, 101)], expected_cases={'encoding-json/encode/1'})
        c = s['cases'][0]
        self.assertEqual(c['latency_ns']['p99'], 1900)
        self.assertEqual(c['batch_ns_per_op']['p99'], 1000)
        self.assertEqual(c['operations'], {'warmup': 200, 'measurement': 200, 'latency': 200, 'teardown': 200})
        self.assertEqual(c['requests_sampled'], 200)
        # 2 ns loop + one 30 ns timer pair amortized over a two-operation batch.
        self.assertEqual(s['amortized_overhead_ns_per_op']['p50'], 17)

    def test_rejects_insufficient_trials_duplicates_or_changed_configuration(self):
        valid = [trial(n) for n in range(1, 101)]
        broken = [valid[:-1], valid[:-1] + [valid[0]]]
        for field, value in [('config', {}), ('gomaxprocs', 2), ('go_version', 'other')]:
            data = copy.deepcopy(valid); data[-1][field] = value; broken.append(data)
        data = copy.deepcopy(valid); data[-1]['cases'] = []; broken.append(data)
        data = copy.deepcopy(valid); data[-1]['cases'][0]['latencies_ns'] = []; broken.append(data)
        for data in broken:
            with self.subTest(data=data[-1]['trial']), self.assertRaises(ValueError):
                summarize(data, expected_cases={'encoding-json/encode/1'})


if __name__ == '__main__': unittest.main()
