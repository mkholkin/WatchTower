"""Exercise the real pre-commit hook in a disposable Git repository."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
VALID = 'package sample\n\nfunc Value() int {\n\treturn 1\n}\n'


class GateIntegrationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.directory = tempfile.TemporaryDirectory(prefix='lab7-gate-test-')
        cls.addClassCleanup(cls.directory.cleanup)
        cls.root = Path(cls.directory.name)
        cls.env = dict(os.environ, LAB7_CACHE_DIR=str(ROOT/'.cache/lab7'),
                       GOCACHE=str(ROOT/'.cache/go-build'),
                       GOMODCACHE=str(ROOT/'.cache/lab7/modules'))
        for key in ('GIT_DIR', 'GIT_WORK_TREE', 'GIT_INDEX_FILE', 'GIT_COMMON_DIR'):
            cls.env.pop(key, None)
        shutil.copytree(ROOT/'tools/lab7/metrics', cls.root/'tools/lab7/metrics')
        shutil.copy2(ROOT/'tools/lab7/check.py', cls.root/'tools/lab7/check.py')
        shutil.copytree(ROOT/'.githooks', cls.root/'.githooks')
        (cls.root/'go.mod').write_text('module gatefixture\n\ngo 1.26\n')
        (cls.root/'.gitignore').write_text('/build/\n/.cache/\n')
        cls.sample = cls.root/'cmd/sample/sample.go'
        cls.sample.parent.mkdir(parents=True)
        cls.sample.write_text(VALID)
        for args in [('init', '-q'), ('config', 'user.name', 'Gate Test'),
                     ('config', 'user.email', 'gate-test@example.invalid'),
                     ('config', 'commit.gpgsign', 'false'),
                     ('config', 'core.hooksPath', '.githooks'), ('add', '--all')]:
            cls.git(*args, check=True)
        initial = cls.git('commit', '-m', 'valid initial fixture')
        if initial.returncode:
            raise AssertionError(initial.stdout)
        cls.initial_head = cls.git('rev-parse', 'HEAD', check=True).stdout.strip()

    @classmethod
    def git(cls, *args, check=False):
        return subprocess.run(['git', *args], cwd=cls.root, env=cls.env,
                              text=True, stdout=subprocess.PIPE,
                              stderr=subprocess.STDOUT, check=check)

    def setUp(self):
        self.git('reset', '--hard', self.initial_head, check=True)

    def stage(self, source):
        self.sample.write_text(source)
        self.git('add', 'cmd/sample/sample.go', check=True)

    def assert_blocked(self, diagnostic, *commit_flags):
        before = self.git('rev-parse', 'HEAD', check=True).stdout
        result = self.git('commit', *commit_flags, '-m', 'must be rejected')
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertIn(diagnostic, result.stdout)
        self.assertEqual(self.git('rev-parse', 'HEAD', check=True).stdout, before)

    def test_valid_commit_passes(self):
        self.stage(VALID.replace('return 1', 'return 2'))
        result = self.git('commit', '-m', 'valid change')
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn('Static gate passed:', result.stdout)
        self.assertNotEqual(self.git('rev-parse', 'HEAD').stdout.strip(), self.initial_head)

    def test_staged_complexity_rejected_when_working_tree_fixed(self):
        source = 'package sample\n\nfunc Value(x int) int {\n'
        source += ''.join(f'\tif x == {i} {{\n\t\tx++\n\t}}\n' for i in range(10))
        source += '\treturn x\n}\n'
        self.stage(source)
        self.sample.write_text(VALID)
        self.assert_blocked('functions exceed cyclomatic complexity 10')
        report = json.loads((self.root/'build/lab7/metrics.json').read_text())
        self.assertEqual(report['summary']['max_cyclomatic'], 11)
        self.assertEqual(report['summary']['violations'], 1)

    def test_bad_format_rejected(self):
        self.stage('package sample\nfunc Value()int{return 3}\n')
        self.assert_blocked('gofmt rejected:')

    def test_type_error_rejected(self):
        self.stage(VALID.replace('return 1', 'return "wrong type"'))
        self.assert_blocked('go vet/type checking')

    def test_commit_all_checks_temporary_index(self):
        self.sample.write_text(VALID.replace('return 1', 'return "wrong type"'))
        self.assert_blocked('go vet/type checking', '--all')

    def test_root_level_type_error_rejected(self):
        (self.root/'root.go').write_text('package gatefixture\n\nfunc RootValue() int {\n\treturn "wrong type"\n}\n')
        self.git('add', 'root.go', check=True)
        self.assert_blocked('go vet/type checking')

    def test_no_verify_bypasses_and_valid_commit_recovers(self):
        self.stage('package sample\nfunc Value()int{return 3}\n')
        bypass = self.git('commit', '--no-verify', '-m', 'explicit bypass')
        self.assertEqual(bypass.returncode, 0, bypass.stdout)
        self.assertNotIn('Static gate passed:', bypass.stdout)
        self.stage(VALID.replace('return 1', 'return 4'))
        recovered = self.git('commit', '-m', 'recover valid source')
        self.assertEqual(recovered.returncode, 0, recovered.stdout)
        self.assertIn('Static gate passed:', recovered.stdout)


if __name__ == '__main__':
    unittest.main()
