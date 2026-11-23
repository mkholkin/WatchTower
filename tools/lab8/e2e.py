"""Exercise the registration feature through the running WatchTower HTTP API."""
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import time
from urllib.error import HTTPError
from urllib.parse import urlsplit
from urllib.request import Request, urlopen
import uuid


class API:
    def __init__(self, base):
        self.base = base.rstrip('/')

    def request(self, method, path, body=None, token=''):
        headers = {'Content-Type': 'application/json'}
        if token:
            headers['Authorization'] = 'Bearer ' + token
        request = Request(self.base + path, data=None if body is None else json.dumps(body).encode(),
                          headers=headers, method=method)
        try:
            response = urlopen(request, timeout=10)
        except HTTPError as error:
            response = error
        with response:
            raw = response.read(1024 * 1024)
            return response.status, json.loads(raw) if raw else None


def expect_status(response, expected):
    status, body = response
    if status != expected:
        raise AssertionError(f'HTTP {status}, expected {expected}')
    return body


def validate_registration(body, domain, mode, source_base=None, partial=False):
    if not isinstance(body, dict) or body.get('domain') != domain:
        raise AssertionError('Response domain does not match monitor domain')
    for key in ('registered_at', 'expires_at'):
        if key not in body:
            raise AssertionError(f'Missing {key}')
        value = body[key]
        if value is not None:
            if not isinstance(value, str) or not re.fullmatch(
                    r'\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?(?:Z|[+-]\d\d:\d\d)', value):
                raise AssertionError(f'{key} must be RFC3339 or null')
            datetime.fromisoformat(value.replace('Z', '+00:00'))
    source = body.get('source_url')
    if not isinstance(source, str):
        raise AssertionError('Missing source_url')
    parsed = urlsplit(source)
    if parsed.scheme not in ('http', 'https') or not parsed.netloc or not parsed.path.endswith('/domain/' + domain):
        raise AssertionError('Invalid RDAP source_url')
    if source_base and source != source_base.rstrip('/') + '/domain/' + domain:
        raise AssertionError('source_url must use configured fixed RDAP base URL')
    if mode == 'mock':
        expected = ('1995-08-14T04:00:00Z', None if partial else '2027-08-13T04:00:00Z')
        if (body['registered_at'], body['expires_at']) != expected:
            raise AssertionError('Mock registration dates differ from top-level fixture events')
    elif body['registered_at'] is None or body['expires_at'] is None:
        raise AssertionError('Real example.com response must include both domain dates')
    return body


def monitor_payload(host):
    return {'label': 'Lab8 disposable ' + host, 'endpoint': f'http://{host}:8081/health',
            'probe_interval': 3600, 'alert_contact_ids': [], 'maintenance_window_ids': [],
            'network_config': {'protocol': 'HTTP', 'method': 'GET', 'headers': {}, 'body': '', 'follow_redirects': False},
            'expectations': {'protocol': 'HTTP', 'expected_status_codes': [200], 'expected_response_time_ms': 5000}}


class Suite:
    def __init__(self, mode, output, demo=False):
        self.mode, self.output, self.steps = mode, Path(output), []
        self.demo = demo
        (self.output / 'allure-results').mkdir(parents=True, exist_ok=True)

    def step(self, name, action, attach=False):
        start = int(time.time() * 1000)
        row = {'name': name, 'status': 'passed'}
        try:
            result = action()
        except Exception as error:
            row.update(status='failed', error=f'{type(error).__name__}: {error}')
            raise
        finally:
            row.update(start=start, stop=int(time.time() * 1000))
            self.steps.append(row)
            full_name = f'lab8.{self.mode}.{name}'
            record = {'uuid': str(uuid.uuid4()), 'name': name, 'fullName': full_name,
                      'historyId': hashlib.sha256(full_name.encode()).hexdigest(),
                      'status': row['status'], 'stage': 'finished', 'start': row['start'], 'stop': row['stop'],
                      'labels': [{'name': 'suite', 'value': f'Lab8 {self.mode} E2E'}]}
            if attach and row['status'] == 'passed':
                filename = f'registration-{self.mode}.json'
                (self.output/'allure-results'/filename).write_text(json.dumps(result, indent=2)+'\n')
                (self.output/'registration.json').write_text(json.dumps(result, indent=2)+'\n')
                record['attachments'] = [dict(name='Регистрация домена', source=filename, type='application/json')]
            if 'error' in row:
                record['statusDetails'] = {'message': row['error']}
            (self.output / 'allure-results' / f'{record["uuid"]}-result.json').write_text(json.dumps(record) + '\n')
            print(f'{row["status"]}: {name}', flush=True)
        return result

    def finish(self):
        passed = bool(self.steps) and all(row['status'] == 'passed' for row in self.steps)
        report = {'mode': self.mode, 'status': 'passed' if passed else 'failed', 'steps': self.steps}
        (self.output / 'e2e.json').write_text(json.dumps(report, indent=2) + '\n')
        return 0 if passed else 1


def run(api, suite, mock_url):
    monitors = []
    owner = ''
    source = mock_url.rstrip('/') + '/com/v1' if suite.mode == 'mock' else 'https://rdap.verisign.com/com/v1'

    def account():
        credentials = {'login': 'lab8_' + uuid.uuid4().hex[:20], 'password': secrets.token_urlsafe(24)}
        expect_status(api.request('POST', '/auth/register', credentials), 201)
        body = expect_status(api.request('POST', '/auth/login', credentials), 200)
        token = body.get('access_token')
        if not isinstance(token, str) or not token:
            raise AssertionError('Login did not return an access token')
        return token

    def create(host):
        body = expect_status(api.request('POST', '/monitors', monitor_payload(host), owner), 201)
        monitor_id = str(uuid.UUID(body['id']))
        monitors.append(monitor_id)
        expect_status(api.request('POST', f'/monitors/{monitor_id}/disable', token=owner), 200)
        return f'/monitors/{monitor_id}/registration'

    def stats():
        return expect_status(API(mock_url).request('GET', '/__admin/stats'), 200)

    try:
        owner = suite.step('register and log in owner', account)
        foreign = suite.step('register and log in second owner', account)
        path = suite.step('create own example.com monitor', lambda: create('example.com'))
        before = suite.step('read initial upstream counters', stats) if suite.mode == 'mock' else None
        suite.step('missing JWT returns 401', lambda: expect_status(api.request('GET', path), 401))
        suite.step('foreign owner returns 403', lambda: expect_status(api.request('GET', path, token=foreign), 403))
        suite.step('invalid monitor UUID returns 400', lambda: expect_status(api.request('GET', '/monitors/not-a-uuid/registration', token=owner), 400))
        suite.step('missing monitor returns 404', lambda: expect_status(api.request('GET', f'/monitors/{uuid.uuid4()}/registration', token=owner), 404))
        if suite.mode == 'mock':
            def check_authorization():
                if stats() != before:
                    raise AssertionError('Rejected requests reached RDAP upstream')
            suite.step('authorization and UUID failures make zero upstream calls', check_authorization)
        suite.step('read example.com registration', lambda: validate_registration(
            expect_status(api.request('GET', path, token=owner), 200), 'example.com', suite.mode, source), attach=True)
        if suite.mode == 'mock':
            for host, status in [('missing.com', 404), ('busy.com', 503), ('broken.com', 502),
                                 ('down.com', 502), ('slow.com', 504), ('partial.com', 200),
                                 ('127.0.0.1', 400), ('example.org', 400)]:
                def check(host=host, status=status):
                    target = create(host)
                    body = expect_status(api.request('GET', target, token=owner), status)
                    if status == 200:
                        validate_registration(body, host, 'mock', source, partial=True)
                suite.step(f'{host} returns {status}', check)
            def check_calls():
                after = stats()['requests']
                previous = before['requests']
                for host in ('example.com', 'missing.com', 'busy.com', 'broken.com', 'down.com', 'slow.com', 'partial.com'):
                    delta = after.get(host, {}).get('GET', 0) - previous.get(host, {}).get('GET', 0)
                    if delta != 1:
                        raise AssertionError(f'{host} made {delta} upstream requests, expected one without retries')
                if after.get('_other', {}) != previous.get('_other', {}):
                    raise AssertionError('Unsupported monitor domain reached RDAP')
            suite.step('each lookup makes exactly one upstream request', check_calls)
        if suite.demo:
            path = suite.output/'demo-session.json'
            descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
            with os.fdopen(descriptor, 'w') as stream:
                json.dump(dict(token=owner, monitor_id=monitors[0]), stream)
    finally:
        for index, monitor_id in enumerate(monitors, 1):
            if suite.demo and index == 1:
                continue
            try:
                suite.step(f'delete owned monitor {index}', lambda monitor_id=monitor_id:
                           expect_status(api.request('DELETE', '/monitors/' + monitor_id, token=owner), 204))
            except Exception:
                pass


def main():
    mode = os.environ.get('LAB8_MODE', 'mock')
    if mode not in ('mock', 'real'):
        raise SystemExit('LAB8_MODE must be mock or real')
    suite = Suite(mode, os.environ.get('LAB8_OUTPUT', '/artifacts'), demo=os.environ.get('LAB8_DEMO') == '1')
    try:
        suite.step('complete E2E scenario', lambda: run(
            API(os.environ.get('LAB8_API_URL', 'http://api:8080')), suite,
            os.environ.get('LAB8_MOCK_URL', 'http://rdap-mock:8089')))
    except Exception as error:
        print(f'E2E failed: {type(error).__name__}: {error}', flush=True)
    return suite.finish()


if __name__ == '__main__':
    raise SystemExit(main())
