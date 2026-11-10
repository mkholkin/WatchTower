"""Publish test identifiers and statuses, never raw process output or fixtures."""
import json
from pathlib import Path
import re

FEATURE = Path(__file__).resolve().parents[2]/'tests/lab5/features/authentication.feature'


def poll_progress(output, positions):
    for filename, kind in [('domain-tests.jsonl', 'GO'), ('bdd-progress.jsonl', 'BDD')]:
        path = Path(output)/filename
        try:
            with path.open() as stream:
                stream.seek(positions.get(filename, 0))
                while True:
                    offset = stream.tell()
                    line = stream.readline()
                    if not line or not line.endswith('\n'):
                        positions[filename] = offset
                        break
                    positions[filename] = stream.tell()
                    try:
                        event = json.loads(line)
                        if kind == 'GO':
                            if set(event) != {'test', 'status'} or event['status'] not in ('run', 'pass', 'fail', 'skip'):
                                continue
                            name = event['test']
                            if not isinstance(name, str) or not re.fullmatch(r'Test[A-Za-z0-9_]+', name):
                                continue
                        else:
                            if set(event) != {'line', 'status'} or event['status'] not in ('run', 'passed', 'failed', 'skipped', 'undefined', 'untested'):
                                continue
                            number = event['line']
                            source = FEATURE.read_text().splitlines()
                            if type(number) is not int or not 1 <= number <= len(source):
                                continue
                            candidates = [s.strip().split(':', 1)[1].strip() for s in source[:number]
                                          if s.strip().startswith(('Scenario:', 'Scenario Outline:'))]
                            if not candidates:
                                continue
                            name = f'{candidates[-1]} (line {number})'
                            if event['status'] != 'run':
                                positions.setdefault('_bdd_results', {})[number] = event['status']
                        print(f'{kind} {event["status"].upper()}: {name}', flush=True)
                    except (ValueError, KeyError, TypeError):
                        continue
        except OSError:
            continue
