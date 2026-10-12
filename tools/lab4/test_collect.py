import unittest

from tools.lab4.collect import sample


class ResourceTests(unittest.TestCase):
    def test_cpu_working_set_and_io_deltas(self):
        def stats(cpu, system, io):
            return {'cpu_stats': {'cpu_usage': {'total_usage': cpu}, 'system_cpu_usage': system, 'online_cpus': 4},
                    'memory_stats': {'usage': 1000, 'stats': {'inactive_file': 200}},
                    'blkio_stats': {'io_service_bytes_recursive': [{'op': 'Read', 'value': io}, {'op': 'Write', 'value': io*2}]}}
        previous = stats(100, 1000, 1000)
        current = stats(150, 1400, 1100)
        row = sample(current, previous, 2)
        self.assertEqual(row, {'cpu_percent': 50, 'memory_bytes': 800, 'read_bytes_s': 50, 'write_bytes_s': 100})

    def test_missing_counters_are_not_zero_measurements(self):
        row = sample({}, {}, 1)
        self.assertIsNone(row['cpu_percent'])
        self.assertIsNone(row['read_bytes_s'])
        self.assertIsNone(row['memory_bytes'])

    def test_readable_empty_cgroup_file_means_zero_io(self):
        row = sample({'cgroup_io_stat': ''}, {'cgroup_io_stat': ''}, 1)
        self.assertEqual(row['read_bytes_s'], 0)
        self.assertEqual(row['write_bytes_s'], 0)
        row = sample({'cgroup_io_stat': '253:0 rbytes=100 wbytes=250 rios=1 wios=2\n'},
                     {'cgroup_io_stat': ''}, 2)
        self.assertEqual(row['read_bytes_s'], 50)
        self.assertEqual(row['write_bytes_s'], 125)


if __name__ == '__main__':
    unittest.main()
