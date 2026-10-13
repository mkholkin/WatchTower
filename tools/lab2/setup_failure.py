"""Emit explicit stage results after launcher setup fails, without starting services."""
import argparse
import fcntl
import json
import os
from pathlib import Path
import shutil
import subprocess
import time
import uuid


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--artifacts', type=Path, default=Path('/artifacts'))
    parser.add_argument('--history', type=Path, default=Path('/history'))
    args = parser.parse_args()
    out, history = args.artifacts, args.history
    results = out / 'allure-results'
    results.mkdir(exist_ok=True)
    reason = (out / 'setup-failure.txt').read_text().strip()
    stages = {'setup': {'status': 'failed', 'reason': reason}}
    stages.update({name: {'status': 'skipped', 'reason': 'Environment preparation did not succeed'}
                   for name in ('unit', 'integration', 'e2e')})
    now = int(time.time() * 1000)
    for name, result in stages.items():
        identity = f'lab2.pipeline.{name}'
        item = {'uuid': str(uuid.uuid4()), 'name': name, 'fullName': identity,
                'historyId': identity, 'testCaseId': identity, 'stage': 'finished',
                'status': result['status'], 'start': now, 'stop': now,
                'statusDetails': {'message': result['reason']},
                'labels': [{'name': 'parentSuite', 'value': 'CI pipeline'}, {'name': 'suite', 'value': name}]}
        (results / f"setup-{name}-result.json").write_text(json.dumps(item))
    (out / 'pipeline.json').write_text(json.dumps({'run': os.environ['LAB2_RUN_ID'], 'exit_code': 1,
                                                  'stages': stages}, indent=2))
    local_allure = Path('tools/lab1/node_modules/.bin/allure')
    allure = str(local_allure.resolve()) if local_allure.exists() else shutil.which('allure')
    if not allure:
        with (out / 'setup-failure.txt').open('a') as diagnostic:
            diagnostic.write('Allure executable unavailable: raw results preserved; HTML report/history not generated.\n')
        return 1
    # Match normal report history/locking while allowing host paths for the fallback.
    config = out / 'setup-allurerc.mjs'
    config.write_text('export default ' + json.dumps({
        'name': 'WatchTower — unit → integration → E2E',
        'historyPath': str(history / 'history.jsonl'), 'appendHistory': True,
        'plugins': {'awesome': {'options': {'singleFile': True}}}}) + ';\n')
    try:
        with (history / '.lock').open('w') as lock, (out / 'allure.log').open('w') as log:
            fcntl.flock(lock, fcntl.LOCK_EX)
            subprocess.run([allure, 'generate', str(results), '--config', str(config),
                            '--output', str(out / 'allure-report')], stdout=log,
                           stderr=subprocess.STDOUT, check=True, timeout=120)
    except (OSError, subprocess.SubprocessError) as error:
        with (out / 'setup-failure.txt').open('a') as diagnostic:
            diagnostic.write(f'Allure generation failed: {error}; see allure.log.\n')
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
