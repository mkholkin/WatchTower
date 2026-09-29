"""Replay the public MVP flow with curl; the runner captures its real HTTP traffic."""
import json
import os
import subprocess
import time

BASE = os.environ['LAB2_API_URL']
token = ''


def request(method, path, expected=200, body=None, authenticated=True):
    args = ['curl', '--silent', '--show-error', '--max-time', '10', '-X', method,
            '-H', 'Content-Type: application/json', '-w', '\n%{http_code}', BASE + path]
    if token and authenticated:
        args += ['-H', 'Authorization: Bearer ' + token]
    if body is not None:
        args += ['--data', json.dumps(body)]
    result = subprocess.check_output(args, text=True)
    payload, code = result.rsplit('\n', 1)
    print(f'{method} {path} -> {code}', flush=True)
    assert int(code) == expected, (method, path, code, payload)
    return json.loads(payload) if payload else None


credentials = {'login': 'lab2_curl', 'password': 'Lab2Pass-123!'}
request('POST', '/auth/register', 201, credentials)
token = request('POST', '/auth/login', body=credentials)['access_token']
request('GET', '/monitors', 401, authenticated=False)
monitor = request('POST', '/monitors', 201, {
    'label': 'Curl demo', 'endpoint': os.environ['LAB2_PROBE_URL'], 'probe_interval': 1,
    'alert_contact_ids': [], 'maintenance_window_ids': [],
    'network_config': {'protocol': 'HTTP', 'method': 'GET', 'headers': {}, 'body': '', 'follow_redirects': False},
    'expectations': {'protocol': 'HTTP', 'expected_status_codes': [200], 'expected_response_time_ms': 5000}})
path = '/monitors/' + monitor['id']
deadline = time.monotonic() + 30
while True:
    detail = request('GET', path)
    checks = request('GET', path + '/checks?limit=10')
    if detail['status'] == 'up' and any(c['status_code'] == 200 for c in checks):
        break
    if time.monotonic() >= deadline:
        raise RuntimeError('real probe did not become UP')
    time.sleep(0.25)
request('GET', '/monitors')
request('GET', path + '/sla')
request('PATCH', path, body={'label': 'Curl demo renamed'})
request('POST', path + '/disable')
request('POST', path + '/enable')
request('DELETE', path, 204)
request('GET', path, 404)
print('curl MVP replay passed')
