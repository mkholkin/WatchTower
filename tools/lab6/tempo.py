"""Stream sanitized Lab 6 JSONL spans to Tempo without shifting experiment time.

OTLP JSON encoding: https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding
Only the exporter's allowlisted fields enter the payload. Memory is bounded by
one 500-span batch; root_spans counts root events, not unique trace IDs.
"""
import json
from pathlib import Path
import re
from urllib.request import Request, urlopen

from tools.lab6.report import PROFILES, VARIANTS

BATCH_SIZE = 500
REQUIRED = {'trace_id', 'span_id', 'name', 'started_ns', 'duration_ns', 'status'}
SAFE_NAME = re.compile(
    r'(?:postgres (?:SELECT|INSERT|UPDATE|DELETE|BEGIN|COMMIT|ROLLBACK|CREATE|ALTER|DROP|GRANT|DO|OTHER)|'
    r'(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|OTHER) (?:/[A-Za-z0-9_/:.{}-]*|unmatched))')


def convert_span(event):
    """Validate the privacy allowlist and return one OTLP JSON span.

    The native exporter uses random 128-bit span IDs. OTLP requires 64-bit IDs:
    retain the first 16 hex characters for both span and parent references.
    """
    if not isinstance(event, dict) or not REQUIRED <= event.keys() or event.keys()-REQUIRED-{'parent_id'}:
        raise ValueError('Unsafe or invalid trace event fields')
    for key in ('started_ns', 'duration_ns'):
        if type(event[key]) is not int or not 0 <= event[key] < 2**64:
            raise ValueError('Invalid trace timestamp/duration')
    end = event['started_ns']+event['duration_ns']
    if end >= 2**64:
        raise ValueError('Invalid trace end timestamp')
    for key in ('trace_id', 'span_id', 'parent_id'):
        value = event.get(key)
        if key == 'parent_id' and value in (None, ''):
            continue
        pattern = r'[0-9a-f]{32}' if key == 'trace_id' else r'(?:[0-9a-f]{16}|[0-9a-f]{32})'
        if not isinstance(value, str) or not re.fullmatch(pattern, value) or int(value, 16) == 0:
            raise ValueError('Invalid trace identity')
        if key != 'trace_id' and int(value[:16], 16) == 0:
            raise ValueError('Invalid truncated span identity')
    status, name = event['status'], event['name']
    if type(status) is not int or not 100 <= status <= 599:
        raise ValueError('Invalid trace status')
    if not isinstance(name, str) or not SAFE_NAME.fullmatch(name) or re.search(
            r'[0-9a-f]{8}-[0-9a-f-]{27,}', name, re.I):
        raise ValueError('Trace includes an unsanitized route or secret')
    postgres = name.startswith('postgres ')
    result = {'traceId': event['trace_id'], 'spanId': event['span_id'][:16], 'name': name,
              'kind': 3 if postgres else 2,
              'startTimeUnixNano': str(event['started_ns']), 'endTimeUnixNano': str(end),
              'status': {'code': 2 if status >= 500 else 1},
              'attributes': [] if postgres else [
                  {'key': 'http.response.status_code', 'value': {'intValue': str(status)}}]}
    if event.get('parent_id'):
        result['parentSpanId'] = event['parent_id'][:16]
    return result


def import_traces(root, endpoint):
    """Import manifest-listed trace profiles; raise on unsafe data or rejected spans.

    Original nanosecond timestamps remain unchanged. The accompanying Tempo
    configuration allows historical block timestamps within a ten-year window.
    Re-importing the same artifacts retains IDs and does not create new traces.
    """
    root = Path(root).resolve()
    manifest = json.loads((root/'manifest.json').read_text())
    run_id = manifest['run_id']
    if not isinstance(run_id, str) or not re.fullmatch(r'[A-Za-z0-9_-]{1,128}', run_id):
        raise ValueError('Invalid run ID')
    endpoint = endpoint.rstrip('/')
    if not endpoint.endswith('/v1/traces'):
        endpoint += '/v1/traces'
    counts = dict(files=0, spans=0, root_spans=0, batches=0, example_trace_id=None)
    example_has_parent = False
    for record in manifest['profiles']:
        variant, profile = record['variant'], record['name']
        repetition = record.get('repetition', 1)
        if variant not in VARIANTS or profile not in PROFILES or type(repetition) is not int or repetition < 1:
            raise ValueError('Invalid profile/variant/repetition')
        if variant not in ('trace', 'trace-debug'):
            continue
        path = (root/record['path']/'traces.jsonl').resolve()
        if not path.is_relative_to(root):
            raise ValueError('Profile path escapes artifact directory')
        attributes = [{'key': key, 'value': {'stringValue': value}} for key, value in (
            ('service.name', 'watchtower-lab6'), ('run_id', run_id), ('profile', profile), ('variant', variant))]
        attributes.append({'key': 'repetition', 'value': {'intValue': str(repetition)}})

        def send(batch):
            nonlocal example_has_parent
            payload = {'resourceSpans': [{'resource': {'attributes': attributes},
                                         'scopeSpans': [{'scope': {'name': 'watchtower.lab6.import'}, 'spans': batch}]}]}
            request = Request(endpoint, data=json.dumps(payload).encode(), headers={'Content-Type': 'application/json'})
            with urlopen(request, timeout=60) as response:
                result = json.loads(response.read() or b'{}')
            if int(result.get('partialSuccess', {}).get('rejectedSpans', 0)):
                raise ValueError('Tempo rejected spans in OTLP response')
            counts['batches'] += 1
            counts['spans'] += len(batch)
            counts['root_spans'] += sum('parentSpanId' not in event for event in batch)
            candidate = next((event for event in batch if 'parentSpanId' in event), None)
            if candidate and not example_has_parent:
                counts['example_trace_id'] = candidate['traceId']
                example_has_parent = True
            counts['example_trace_id'] = counts['example_trace_id'] or batch[0]['traceId']

        batch = []
        previous_spans = counts['spans']
        with path.open() as source:
            for line_number, line in enumerate(source, 1):
                if not line.strip():
                    continue
                try:
                    batch.append(convert_span(json.loads(line)))
                except ValueError as error:
                    raise ValueError(f'{path}:{line_number}: {error}') from error
                if len(batch) == BATCH_SIZE:
                    send(batch)
                    batch = []
        if batch:
            send(batch)
        if counts['spans'] == previous_spans:
            raise ValueError(f'Tracing profile has an empty trace file: {path}')
        counts['files'] += 1
    return counts
