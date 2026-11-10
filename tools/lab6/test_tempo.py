import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from tools.lab6.tempo import convert_span, import_traces


def span(**changes):
    event = dict(trace_id='a'*32, span_id='b'*32, name='GET /monitors/:id',
                 started_ns=1794311660243792377, duration_ns=685833, status=200)
    event.update(changes)
    return event


class TempoTests(unittest.TestCase):
    def test_otlp_preserves_times_trace_identity_and_parent_links(self):
        parent = convert_span(span())
        child = convert_span(span(span_id='c'*32, parent_id='b'*32, name='postgres SELECT', status=503))
        self.assertEqual(parent['traceId'], 'a'*32)
        self.assertEqual(parent['spanId'], 'b'*16)
        self.assertEqual(child['parentSpanId'], parent['spanId'])
        self.assertNotIn('parentSpanId', parent)
        self.assertEqual(parent['startTimeUnixNano'], '1794311660243792377')
        self.assertEqual(parent['endTimeUnixNano'], '1794311660244478210')
        self.assertEqual(parent['status'], {'code': 1})
        self.assertEqual(child['status'], {'code': 2})
        self.assertEqual((parent['kind'], child['kind']), (2, 3))
        self.assertEqual(parent['attributes'], [{'key': 'http.response.status_code', 'value': {'intValue': '200'}}])

    def test_rejects_unsafe_fields_names_and_malformed_values(self):
        for changes in ({'authorization': 'secret'}, {'name': 'GET /?token=secret'},
                        {'name': 'GET /users/12345678-1234-1234-1234-123456789abc'},
                        {'trace_id': 'a'*16}, {'span_id': '0'*32}, {'parent_id': 123},
                        {'status': True}, {'status': 600}, {'started_ns': 1.5},
                        {'duration_ns': -1}, {'started_ns': 2**64}, {'name': None}):
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                convert_span(span(**changes))

    def test_import_batches_manifest_profiles_and_checks_otlp_response(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            directory = root/'variants/trace/repetitions/1/profiles/ramp'
            directory.mkdir(parents=True)
            (root/'manifest.json').write_text(json.dumps({'run_id': 'run-1', 'profiles': [
                {'name': 'ramp', 'variant': 'trace', 'repetition': 1, 'path': str(directory.relative_to(root))},
                {'name': 'ramp', 'variant': 'baseline', 'repetition': 1, 'path': 'missing'},
            ]}))
            events = [span() for _ in range(4)]+[span(trace_id='c'*32, parent_id='d'*32, name='postgres SELECT')]
            (directory/'traces.jsonl').write_text('\n'.join(json.dumps(event) for event in events)+'\n')
            payloads = []

            def receive(request, timeout):
                self.assertEqual(request.full_url, 'http://tempo:4318/v1/traces')
                self.assertEqual(request.get_header('Content-type'), 'application/json')
                payloads.append(json.loads(request.data))
                return io.BytesIO(b'{}')

            with patch('tools.lab6.tempo.urlopen', side_effect=receive), patch('tools.lab6.tempo.BATCH_SIZE', 2):
                result = import_traces(root, 'http://tempo:4318')
            self.assertEqual([len(p['resourceSpans'][0]['scopeSpans'][0]['spans']) for p in payloads], [2, 2, 1])
            self.assertEqual(result, {'files': 1, 'spans': 5, 'root_spans': 4, 'batches': 3, 'example_trace_id': 'c'*32})
            attributes = payloads[0]['resourceSpans'][0]['resource']['attributes']
            self.assertEqual({a['key']: a['value'] for a in attributes}, {
                'service.name': {'stringValue': 'watchtower-lab6'}, 'run_id': {'stringValue': 'run-1'},
                'profile': {'stringValue': 'ramp'}, 'variant': {'stringValue': 'trace'}, 'repetition': {'intValue': '1'}})
            with patch('tools.lab6.tempo.urlopen', return_value=io.BytesIO(
                    b'{"partialSuccess":{"rejectedSpans":"1","errorMessage":"bad span"}}')):
                with self.assertRaisesRegex(ValueError, 'rejected'):
                    import_traces(root, 'http://tempo:4318/v1/traces')

    def test_empty_tracing_export_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root/'manifest.json').write_text(json.dumps({'run_id': 'run-1', 'profiles': [
                {'name': 'ramp', 'variant': 'trace', 'repetition': 1, 'path': '.'}]}))
            (root/'traces.jsonl').write_text(' \n\n')
            with self.assertRaises(ValueError):
                import_traces(root, 'http://tempo:4318')

    def test_manifest_path_cannot_escape_and_missing_trace_is_an_error(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for path in ('../outside', 'missing'):
                (root/'manifest.json').write_text(json.dumps({'run_id': 'run-1', 'profiles': [
                    {'name': 'ramp', 'variant': 'trace', 'repetition': 1, 'path': path}]}))
                with self.subTest(path=path), self.assertRaises((ValueError, FileNotFoundError)):
                    import_traces(root, 'http://tempo:4318')


if __name__ == '__main__':
    unittest.main()
