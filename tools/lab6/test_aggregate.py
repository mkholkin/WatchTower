import unittest

from tools.lab6.aggregate import COMPONENTS, memory_mean, totals


def row(stamp, memory, component='api'):
    return dict(time=stamp, memory_bytes=memory, component=component)


class AggregateTests(unittest.TestCase):
    def test_integrates_spike_instead_of_using_sample_median(self):
        rows = [row(0, 0), row(1, 0), row(2, 100), row(3, 0), row(4, 0)]
        self.assertEqual(memory_mean(rows, 'api', 0, 4), 25)

    def test_interpolates_both_boundaries_and_ignores_later_shutdown(self):
        rows = [row(0, 0), row(2, 20), row(4, 40), row(6, 60), row(7, -1)]
        self.assertEqual(memory_mean(rows, 'api', 1, 5), 30)
        rows[2]['memory_bytes'] = -1
        with self.assertRaisesRegex(ValueError, 'invalid memory'):
            memory_mean(rows, 'api', 1, 5)

    def test_sums_components_with_different_sampling_times_over_same_window(self):
        rows = []
        resources = {}
        for index, component in enumerate(COMPONENTS, 1):
            # Every component has a distinct cadence; each follows memory=10*t+index.
            for stamp in (-index/10, 1+index/10, 3+index/10, 5+index/10):
                rows.append(row(stamp+10, 10*stamp+index+10, component))
            resources[component] = dict(cpu_seconds=index)
        result = totals(dict(resources=resources, measurement_start=11, measurement_end=15), rows)
        self.assertEqual(result['total_cpu_seconds'], 10)
        self.assertEqual(result['diagnostics_cpu_seconds'], 9)
        self.assertAlmostEqual(result['total_memory_mean_bytes'], 170)
        self.assertAlmostEqual(result['diagnostics_memory_mean_bytes'], 129)

    def test_rejects_invalid_samples_and_gaps_including_boundary_brackets(self):
        cases = [[row(0, 1), row(6, 1)], [row(1, 1), row(2, 1)],
                 [row(0, 1), row(2, float('nan')), row(4, 1)],
                 [row(0, 1), row(2, True), row(4, 1)],
                 [row(0, 1), row(2, 1), row(2, 2), row(4, 1)]]
        for rows in cases:
            with self.subTest(rows=rows), self.assertRaises(ValueError):
                memory_mean(rows, 'api', 1, 3)


if __name__ == '__main__':
    unittest.main()
