"""Deterministic RDAP server for Lab 8; request counters contain no credentials."""
import copy
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import threading
import time
from urllib.parse import urlsplit

FIXTURE = Path(__file__).resolve().parents[2] / 'api/contracts/rdap/fixtures/example.com.json'
DOMAINS = ('example.com', 'missing.com', 'busy.com', 'broken.com', 'down.com', 'slow.com', 'partial.com')
METHODS = ('GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS')


class RDAPServer(ThreadingHTTPServer):
    daemon_threads = True

    def __init__(self, address):
        super().__init__(address, Handler)
        self.fixture = json.loads(FIXTURE.read_text())
        self.counts = {}
        self.count_lock = threading.Lock()

    def stats(self):
        with self.count_lock:
            return {'requests': copy.deepcopy(self.counts)}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        pass

    def reply(self, status, body, retry_after=None, log=True):
        encoded = body if isinstance(body, bytes) else json.dumps(body).encode()
        self.send_response(status)
        self.send_header('Content-Type', 'application/rdap+json')
        self.send_header('Content-Length', str(len(encoded)))
        if retry_after:
            self.send_header('Retry-After', retry_after)
        self.end_headers()
        if self.command != 'HEAD':
            try:
                self.wfile.write(encoded)
            except (BrokenPipeError, ConnectionResetError):
                pass
        if log:
            print(f'rdap-mock method={self.command if self.command in METHODS else "other"} status={status}', flush=True)

    def handle_request(self):
        path = urlsplit(self.path).path
        if path == '/__admin/stats' and self.command == 'GET':
            self.reply(200, self.server.stats(), log=False)
            return
        domain = path.removeprefix('/com/v1/domain/') if path.startswith('/com/v1/domain/') else ''
        bucket = domain if domain in DOMAINS else '_other'
        method = self.command if self.command in METHODS else '_other'
        with self.server.count_lock:
            methods = self.server.counts.setdefault(bucket, {})
            methods[method] = methods.get(method, 0) + 1
        if domain not in DOMAINS:
            self.reply(404, {'errorCode': 404})
        elif self.command != 'GET':
            self.reply(405, {'errorCode': 405})
        elif domain == 'missing.com':
            self.reply(404, {'errorCode': 404})
        elif domain == 'busy.com':
            self.reply(429, {'errorCode': 429}, '1')
        elif domain == 'broken.com':
            self.reply(200, b'{invalid-json')
        elif domain == 'down.com':
            self.reply(503, {'errorCode': 503})
        else:
            if domain == 'slow.com':
                time.sleep(2)
            body = copy.deepcopy(self.server.fixture)
            body['ldhName'] = domain.upper()
            if domain == 'partial.com':
                body['events'] = [e for e in body['events'] if e['eventAction'] != 'expiration']
            self.reply(200, body)

    do_GET = handle_request
    do_POST = handle_request
    do_PUT = handle_request
    do_PATCH = handle_request
    do_DELETE = handle_request
    do_HEAD = handle_request
    do_OPTIONS = handle_request


if __name__ == '__main__':
    with RDAPServer(('0.0.0.0', int(os.environ.get('LAB8_MOCK_PORT', '8089')))) as server:
        server.serve_forever()
