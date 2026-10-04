"""Run independent microbenchmark trials; keep measurements outside Docker output."""
import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import platform
import signal
import subprocess
import sys
import time
import uuid

ROOT = Path(__file__).resolve().parents[2]


def execute(command, log, timeout=600, **kwargs):
    with log.open('ab') as output:
        return subprocess.run(command, cwd=ROOT, stdout=output, stderr=subprocess.STDOUT,
                              check=True, timeout=timeout, **kwargs)


def arguments():
    p = argparse.ArgumentParser()
    p.add_argument('--trials', type=int, default=100)
    p.add_argument('--smoke', action='store_true', help='diagnostic run; does not satisfy the lab')
    p.add_argument('--native', action='store_true', help='diagnostic fallback without Docker isolation')
    a = p.parse_args()
    if a.trials < (1 if a.smoke else 100): p.error('At least 100 trials required (or use --smoke for diagnostics)')
    return a


def main():
    args = arguments()
    run_id = datetime.now(timezone.utc).strftime('%Y%m%dt%H%M%S') + '-' + uuid.uuid4().hex[:8]
    out = ROOT / 'build/lab3/runs' / run_id
    (out / 'raw').mkdir(parents=True)
    base = f'watchtower-lab3-{run_id}:base'
    current = None
    images = []
    start = time.monotonic()
    manifest = {'run_id': run_id, 'mode': 'native' if args.native else 'docker', 'smoke': args.smoke,
                'requested_trials': args.trials, 'host': platform.platform(), 'trials': [], 'exit_code': 1}
    log = out / 'setup.log'
    print(f'Lab 3 artifacts: {out}', flush=True)
    def interrupted(signum, _): raise InterruptedError(f'signal {signum}')
    signal.signal(signal.SIGTERM, interrupted)
    try:
        if args.native:
            binary = out / 'benchmark'
            execute(['go','build','-trimpath','-o',str(binary),'./benchmarks/lab3'], log)
        else:
            print('Building benchmark/report image...', flush=True)
            images.append(base)
            execute(['docker','build','-f','tools/lab3/Dockerfile','-t',base,'.'], log, timeout=1200)
        for trial in range(1, args.trials + 1):
            target = out / 'raw' / f'trial-{trial:03d}.json'
            if args.native:
                cmd = [str(binary), '-trial', str(trial)]
                metadata = {'trial':trial}
            else:
                image = f'watchtower-lab3-{run_id}:trial-{trial:03d}'
                images.append(image)
                # Separate immutable images; only the trial label differs from the common base.
                dockerfile = f'FROM {base}\nLABEL org.watchtower.lab3.trial="{trial}"\n'
                execute(['docker','build','--network=none','-t',image,'-'], log, input=dockerfile.encode())
                current = f'wt-lab3-{run_id}-{trial:03d}'
                cmd = ['docker','run','--name',current,'--network=none','--cpuset-cpus=0','--cpus=1',
                       '--memory=512m','--memory-swap=512m','--pids-limit=128','--read-only','--cap-drop=ALL',
                       '--security-opt=no-new-privileges',image,'-trial',str(trial)]
            with target.open('wb') as stdout, target.with_suffix('.stderr').open('wb') as stderr:
                subprocess.run(cmd, cwd=ROOT, stdout=stdout, stderr=stderr, timeout=180, check=True)
            if not args.native:
                inspection = json.loads(subprocess.check_output(['docker','inspect',current], timeout=30))[0]
                metadata = {'trial':trial,'image':inspection['Image'],'container':inspection['Id'],
                            'host_config':inspection['HostConfig'],'env':inspection['Config']['Env'],
                            'exit_code':inspection['State']['ExitCode']}
                execute(['docker','rm',current],log,timeout=30); current = None
                execute(['docker','image','rm',image],log,timeout=30); images.remove(image)
            data = json.loads(target.read_text())
            manifest['trials'].append(metadata)
            print(f"[{trial:03d}/{args.trials}] {len(data['cases'])} cases; {data['duration_ns']/1e9:.3f}s; harness {data['overhead_ns_per_op']:.3f} ns/op", flush=True)
        if not args.native:
            trials = manifest['trials']
            if len({t['image'] for t in trials}) != args.trials: raise RuntimeError('Trial images are not unique')
            for t in trials:
                if t['host_config'] != trials[0]['host_config'] or t['env'] != trials[0]['env']:
                    raise RuntimeError('Container configurations differ between trials')
        manifest['exit_code'] = 0
    except BaseException as error:
        manifest['error'] = str(error)
        print(f'Lab 3 failed: {error}', file=sys.stderr, flush=True)
    finally:
        signal.signal(signal.SIGINT, signal.SIG_IGN)
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
        manifest['campaign_seconds'] = time.monotonic() - start
        (out / 'manifest.json').write_text(json.dumps(manifest, indent=2)+'\n')
        try:
            extra = ['--smoke'] if args.smoke else []
            if args.native:
                execute([sys.executable,'tools/lab3/report.py',str(out),*extra],out/'report.log')
                execute(['tools/lab1/node_modules/.bin/allure','generate',str(out/'allure-results'),
                         '--config','tools/lab3/allurerc.mjs','--output',str(out/'allure-report')],out/'allure.log')
            else:
                execute(['docker','run','--rm','--network=none','--user',f'{os.getuid()}:{os.getgid()}',
                         '-v',f'{out}:/artifacts','--entrypoint','python3',base,
                         'tools/lab3/report.py','/artifacts',*extra],out/'report.log')
                execute(['docker','run','--rm','--network=none','--user',f'{os.getuid()}:{os.getgid()}',
                         '-v',f'{out}:/artifacts','--entrypoint','tools/lab1/node_modules/.bin/allure',base,
                         'generate','/artifacts/allure-results','--config','tools/lab3/allurerc.mjs',
                         '--output','/artifacts/allure-report'],out/'allure.log')
        except Exception as error:
            manifest['exit_code']=1;manifest['report_error']=str(error)
            print(f'Report failed: {error}',file=sys.stderr,flush=True)
        if current:
            try: execute(['docker','rm','-f',current],log,timeout=30)
            except Exception as error: manifest['cleanup_error']=str(error);manifest['exit_code']=1
        for image in reversed(images):
            try: execute(['docker','image','rm',image],log,timeout=30)
            except Exception as error: manifest['cleanup_error']=str(error);manifest['exit_code']=1
        (out/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
    print(f"Lab 3 finished (exit {manifest['exit_code']}): {out}",flush=True)
    return manifest['exit_code']


if __name__ == '__main__': sys.exit(main())
