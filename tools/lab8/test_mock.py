import json
import socket
import threading
import time
import unittest
from urllib.error import HTTPError
from urllib.request import Request, urlopen

from rdap_mock import DOMAINS, RDAPServer


class MockTests(unittest.TestCase):
    def setUp(self):
        self.server = RDAPServer(('127.0.0.1', 0))
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.base = f'http://127.0.0.1:{self.server.server_port}'

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)

    def request(self, path, method='GET', timeout=3):
        try:
            response = urlopen(Request(self.base + path, method=method), timeout=timeout)
        except HTTPError as error:
            response = error
        with response:
            return response.status, response.headers, response.read()

    def test_domain_object_content_type_events_and_partial(self):
        status, headers, raw = self.request('/com/v1/domain/example.com')
        self.assertEqual(status, 200)
        self.assertEqual(headers['Content-Type'], 'application/rdap+json')
        body = json.loads(raw)
        self.assertEqual(body['objectClassName'], 'domain')
        self.assertEqual(body['ldhName'], 'EXAMPLE.COM')
        self.assertEqual(body['events'][0]['eventAction'], 'expiration')
        self.assertEqual(body['events'][2]['eventDate'], '1995-08-14T04:00:00Z')
        self.assertNotEqual(body['events'][2]['eventDate'], body['entities'][0]['events'][0]['eventDate'])
        partial = json.loads(self.request('/com/v1/domain/partial.com')[2])
        self.assertFalse(any(e['eventAction'] == 'expiration' for e in partial['events']))

    def test_fault_responses_and_unknown_routes(self):
        for domain, expected in [('missing.com', 404), ('busy.com', 429), ('down.com', 503)]:
            with self.subTest(domain=domain):
                status, headers, raw = self.request('/com/v1/domain/' + domain)
                self.assertEqual(status, expected)
                self.assertEqual(json.loads(raw)['errorCode'], expected)
                if domain == 'busy.com':
                    self.assertEqual(headers['Retry-After'], '1')
        with self.assertRaises(json.JSONDecodeError):
            json.loads(self.request('/com/v1/domain/broken.com')[2])
        self.assertEqual(self.request('/unknown')[0], 404)
        self.assertEqual(self.request('/com/v1/domain/unknown.com')[0], 404)
        self.assertEqual(self.request('/com/v1/domain/example.com', 'POST')[0], 405)

    def test_slow_domain_times_out_without_blocking_other_requests(self):
        start = time.monotonic()
        with self.assertRaises((TimeoutError, socket.timeout)):
            self.request('/com/v1/domain/slow.com', timeout=.1)
        self.assertEqual(self.request('/com/v1/domain/example.com')[0], 200)
        self.assertLess(time.monotonic() - start, 1.5)

    def test_stats_count_methods_and_use_bounded_domain_buckets(self):
        self.request('/com/v1/domain/example.com')
        self.request('/com/v1/domain/example.com', 'HEAD')
        for i in range(5):
            self.request('/com/v1/domain/random' + str(i) + '.com')
        stats = json.loads(self.request('/__admin/stats')[2])
        self.assertEqual(stats, {'requests': {'example.com': {'GET': 1, 'HEAD': 1}, '_other': {'GET': 5}}})
        self.assertEqual(json.loads(self.request('/__admin/stats')[2]), stats)
        self.assertLessEqual(len(stats['requests']), len(DOMAINS) + 1)


if __name__ == '__main__':
    unittest.main()
