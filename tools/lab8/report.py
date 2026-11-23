"""Combine mock/real E2E and load evidence into one Lab 8 report."""
import json
import os
from pathlib import Path
import shutil
import time
import uuid

from tools.lab4.run import run


def collect(output, manifest):
    output = Path(output)
    results = output/'allure-results'
    results.mkdir(exist_ok=True)
    for previous in results.iterdir():
        if previous.is_file():
            previous.unlink()
    for directory in (output/'e2e-mock/allure-results', output/'e2e-real/allure-results',
                      output/'demo-mock/allure-results', output/'demo-real/allure-results',
                      output/'load/allure-results'):
        if directory.is_dir():
            for source in directory.iterdir():
                if source.is_file():
                    shutil.copyfile(source, results/source.name)
    action = manifest['action']
    planned = [('e2e', 'mock'), ('load', 'mock'), ('e2e', 'real')] if action == 'all' else [(action, manifest.get('mode', 'mock'))]
    phases = []
    for kind, mode in planned:
        record = next((r for r in manifest['contours'] if r['kind'] == kind and r['mode'] == mode), None)
        if record is None:
            status = 'skipped'
        elif record.get('error') or record['exit_code'] != 0 or (kind != 'demo' and not record['cleanup']):
            status = 'failed'
        elif kind == 'load':
            status = 'skipped' if manifest['smoke'] else ('passed' if record.get('rdap_evidence') else 'failed')
        else:
            status = 'passed'
        phases.append(dict(kind=kind, mode=mode, status=status))
        stamp = int(time.time()*1000)
        item = dict(uuid=str(uuid.uuid4()), name=f'{kind}: {mode}', fullName=f'lab8.pipeline.{kind}.{mode}',
                    historyId=f'lab8.pipeline.{kind}.{mode}', status=status, stage='finished', start=stamp, stop=stamp,
                    labels=[dict(name='parentSuite', value='ЛР №8'), dict(name='suite', value='Этапы сценария')])
        if record and record.get('error'):
            item['statusDetails'] = dict(message=record['error'])
        (results/(item['uuid']+'-result.json')).write_text(json.dumps(item, ensure_ascii=False)+'\n')
    status = 'failed' if 'error' in manifest or any(p['status'] == 'failed' for p in phases) else 'passed'
    stamp = int(time.time()*1000)
    item = dict(uuid=str(uuid.uuid4()), name='Lab 8 pipeline', fullName='lab8.pipeline', historyId='lab8.pipeline',
                status=status, stage='finished', start=stamp, stop=stamp,
                labels=[dict(name='parentSuite', value='ЛР №8'), dict(name='suite', value='Этапы сценария')])
    if 'error' in manifest:
        item['statusDetails'] = dict(message=manifest['error'])
    (results/(item['uuid']+'-result.json')).write_text(json.dumps(item, ensure_ascii=False)+'\n')
    summary = dict(run_id=manifest['run_id'], smoke=manifest['smoke'], phases=phases,
                   status=status)
    load = output/'load/summary.json'
    if load.exists():
        summary['load'] = json.loads(load.read_text())
    (output/'summary.json').write_text(json.dumps(summary, indent=2, ensure_ascii=False)+'\n')
    lines = ['# ЛР №8: RDAP-интеграция', '', f'Запуск: `{manifest["run_id"]}`.', '',
             '| Этап | Сервис | Результат |', '|---|---|---|']
    lines += [f'| {p["kind"]} | {p["mode"]} | {p["status"]} |' for p in phases]
    if 'load' in summary:
        for profile in summary['load'].get('profiles', []):
            operation = profile['operations'].get('domain-registration', {})
            lines += ['', f'Измерено запросов: {profile["requests"]}; из них RDAP: {operation.get("requests", 0)}.',
                      f'Неожиданных ошибок: {profile["unexpected_requests"]}. Медиана RPS: {profile["rps"]["median"]:.2f}.',
                      f'p95 запросов: {profile["latency_ms"]["p95"]:.2f} мс.']
    lines += ['', 'Нагрузка направлена на mock. Реальный Verisign используется для единичной проверки контракта.', '']
    (output/'results.md').write_text('\n'.join(lines))
    return summary


def generate(output, manifest, app):
    collect(output, manifest)
    if app:
        command = ['docker', 'run', '--rm', '--network=none', '--user', f'{os.getuid()}:{os.getgid()}',
                   '-v', f'{output}:/artifacts', '-v', f'{Path(__file__).parent}:/lab8:ro', '--entrypoint', 'node', app,
                   '/workspace/tools/lab1/node_modules/allure/cli.js', 'generate', '/artifacts/allure-results',
                   '--config', '/lab8/allurerc.mjs', '-o', '/artifacts/allure-report']
    else:
        root = Path(__file__).resolve().parents[2]
        command = [str(root/'tools/lab1/node_modules/.bin/allure'), 'generate', str(output/'allure-results'),
                   '--config', str(root/'tools/lab8/allurerc.mjs'), '-o', str(output/'allure-report')]
    run(command, output/'allure.log', 300)
