"""Load the ephemeral DSN inside the container, never in Compose environment."""
import os
import json
from pathlib import Path
import re
import subprocess
import sys


def safe_test_event(line):
    """Only publish static top-level Go test identifiers and status, never output."""
    try:
        event = json.loads(line)
    except ValueError:
        return None
    name, action = event.get('Test', ''), event.get('Action')
    if action not in ('run', 'pass', 'fail', 'skip') or not re.fullmatch(r'Test[A-Za-z0-9_]+', name):
        return None
    return {'test': name, 'status': action}


def domain_tests(dsn):
    env = os.environ.copy()
    # Existing unit suites use allure-go. The image source tree is read-only for
    # the host UID; keep their raw diagnostics private and export safe events below.
    env.update(MFA_TEST_DATABASE_URL=dsn, GOCACHE='/tmp/go-build', HOME='/tmp',
               ALLURE_RESULTS_DIR='/tmp/unit-allure-results')
    with Path('/artifacts/domain-tests.jsonl').open('w') as report:
        with subprocess.Popen(['go', 'test', '-json', '-race', '-count=1',
                               '-timeout=10m', './internal/service/mfa', './internal/service/auth',
                               './internal/api/http/v1/handler', './internal/infra/notification', './configs'],
                              env=env, text=True, stdout=subprocess.PIPE,
                              stderr=subprocess.STDOUT) as process:
            for line in process.stdout:
                event = safe_test_event(line)
                if event:
                    report.write(json.dumps(event) + '\n')
                    report.flush()
            code = process.wait()
            report.write(json.dumps({'exit_code': code}) + '\n')
            return code


def main():
    dsn = Path('/run/secrets/database_url').read_text().strip()
    if sys.argv[1:] == ['--domain-tests']:
        return domain_tests(dsn)
    os.environ['LAB5_DATABASE_URL'] = dsn
    os.execvp('behave', [
        'behave', 'tests/lab5/features',
        '--no-capture', '--no-capture-stderr', '--no-logcapture',
        '--format', 'allure_behave.formatter:AllureFormatter',
        '--outfile', '/artifacts/allure-results',
        '--format', 'progress', '--outfile', '-',
    ])


if __name__ == '__main__':
    sys.exit(main())
