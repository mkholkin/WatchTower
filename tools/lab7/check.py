"""Run the same Go static gate on the working tree or the staged Git snapshot."""
import argparse
import contextlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[2]


def command(argv, root, env):
    print('+ '+ ' '.join(map(str, argv)), flush=True)
    subprocess.run(list(map(str, argv)), cwd=root, env=env, check=True)


def git_env():
    return {key: value for key, value in os.environ.items()
            if key not in ('GIT_DIR', 'GIT_WORK_TREE', 'GIT_INDEX_FILE', 'GIT_COMMON_DIR')}


@contextlib.contextmanager
def source_tree(root, staged, cache):
    if not staged:
        yield root
        return
    with tempfile.TemporaryDirectory(prefix='index-', dir=cache) as directory:
        snapshot = Path(directory)
        # Read the caller's exact index, including a temporary index used by git commit -a.
        subprocess.run(['git', 'checkout-index', '--all', '--prefix='+str(snapshot)+'/'], cwd=root, check=True)
        clean = git_env()
        command(['git', 'init', '-q', snapshot], root, clean)
        command(['git', 'add', '--force', '--all'], snapshot, clean)
        yield snapshot


def source_files(root, env):
    data = subprocess.check_output(['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z', '--', '*.go'], cwd=root, env=env)
    return [name for name in data.decode().split('\0') if name]


def check(root, output, cache):
    env = git_env()
    env.setdefault('GOCACHE', str(ROOT/'.cache/go-build'))
    files = source_files(root, env)
    if not files:
        raise RuntimeError('No Go source files to check')
    failures = []
    formatted = subprocess.check_output(['gofmt', '-l', *files], cwd=root, env=env, text=True)
    if formatted:
        print('gofmt rejected:\n'+formatted, file=sys.stderr)
        failures.append('gofmt')
    module = root/'tools/lab7/metrics'
    tool_env = dict(env, GOMODCACHE=str(cache/'modules'))
    with tempfile.TemporaryDirectory(prefix='binary-', dir=cache) as directory:
        binary = Path(directory)/'metrics'
        command(['go', 'build', '-mod=readonly', '-o', binary, '.'], module, tool_env)
        try:
            command([binary, '-root', root, '-output', output, '-max-cyclomatic', '10'], root, env)
        except subprocess.CalledProcessError:
            failures.append('complexity/Halstead')
    packages = {}
    for name in files:
        directory = (root/name).parent
        owner = directory
        while owner != root and not (owner/'go.mod').is_file():
            owner = owner.parent
        packages.setdefault(owner, set()).add('./'+str(directory.relative_to(owner)))
    for owner, names in sorted(packages.items()):
        try:
            command(['go', 'vet', '-mod=readonly', '-tags=integration,lab2integration,lab2e2e', *sorted(names)],
                    owner, tool_env if owner == module else env)
        except subprocess.CalledProcessError:
            failures.append('go vet/type checking')
    if failures:
        raise RuntimeError('Static gate failed: '+', '.join(failures))
    summary = json.loads(output.read_text())['summary']
    print(f"Static gate passed: {summary['functions']} functions/closures, max CC={summary['max_cyclomatic']} (limit 10); Halstead: {output}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--staged', action='store_true')
    args = parser.parse_args()
    cache = Path(os.environ.get('LAB7_CACHE_DIR', ROOT/'.cache/lab7')).resolve()
    cache.mkdir(parents=True, exist_ok=True)
    output = ROOT/'build/lab7/metrics.json'
    output.parent.mkdir(parents=True, exist_ok=True)
    output.unlink(missing_ok=True)
    try:
        with source_tree(ROOT, args.staged, cache) as source:
            check(source, output, cache)
    except (OSError, RuntimeError, subprocess.CalledProcessError) as error:
        print(str(error), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
