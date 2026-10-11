"""Exercise separate CI stages with external tools replaced by local executables."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


FAKE_TOOL = '''#!/usr/bin/env python3
import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with pathlib.Path('commands.jsonl').open('a') as log:
    log.write(json.dumps([name, *args]) + '\\n')
if name == 'go':
    stage = 'integration' if '-tags=lab2integration' in args else 'e2e' if '-tags=lab2e2e' in args else 'unit'
    failed = os.environ.get('FAKE_FAIL_STAGE') == stage
    print(json.dumps({'Action': 'fail' if failed else 'pass', 'Package': 'example/' + stage, 'Test': 'TestScenario'}), flush=True)
    sys.exit(1 if failed else 0)
if name == 'psql':
    print(json.dumps({'user': 0, 'monitor': 0}) if 'json_build_object' in args[-1] else '')
if name == 'redis-cli':
    print('0' if args[-1] == 'DBSIZE' else 'OK')
if name == 'allure':
    pathlib.Path('report-generated').touch()
    print('Report generated')
if name == 'watchtower-api':
    sys.exit(9)
'''


class PipelineStageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.output = self.root / 'artifacts'
        self.output.mkdir()
        self.history = self.root / 'history'
        self.history.mkdir()
        self.pipeline = Path(__file__).with_name('pipeline.py').resolve()
        binary = self.root / 'bin'
        binary.mkdir()
        for name in ('go', 'psql', 'redis-cli', 'lab2-migrate', 'watchtower-api'):
            tool = binary / name
            tool.write_text(FAKE_TOOL)
            tool.chmod(0o755)
        lab1 = self.root / 'tools/lab1'
        allure = lab1 / 'node_modules/.bin/allure'
        allure.parent.mkdir(parents=True)
        allure.write_text(FAKE_TOOL)
        allure.chmod(0o755)
        (lab1 / 'packages.txt').write_text('./example/unit\n')
        self.env = dict(os.environ, PATH=f'{binary}:{os.environ["PATH"]}', LAB2_TEST_ENV='1',
                        LAB2_DATABASE_URL='unused', LAB2_REDIS_URL='unused', LAB2_RUN_ID='ci-stages',
                        LAB2_API_URL='http://127.0.0.1:1', LAB2_FAIL_STAGE='')

    def run_stage(self, stage):
        return subprocess.run([sys.executable, str(self.pipeline), stage,
                               '--artifacts', str(self.output), '--history', str(self.history)],
                              cwd=self.root, env=self.env, capture_output=True, text=True, timeout=10)

    def state(self):
        return json.loads((self.output / 'pipeline.json').read_text())

    def commands(self):
        path = self.root / 'commands.jsonl'
        return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []

    def test_stages_run_separately_and_report_is_generated_only_on_finish(self):
        unit = self.run_stage('unit')
        self.assertEqual(unit.returncode, 0, unit.stderr)
        self.assertEqual({n: r['status'] for n, r in self.state()['stages'].items()},
                         {'unit': 'passed', 'integration': 'skipped', 'e2e': 'skipped'})
        self.assertEqual([c[0] for c in self.commands()], ['go'])
        integration = self.run_stage('integration')
        self.assertEqual(integration.returncode, 0, integration.stderr)
        self.assertEqual(self.state()['stages']['integration']['status'], 'passed')
        self.assertTrue(self.state()['initialized'])
        self.assertFalse((self.root / 'report-generated').exists())
        finish = self.run_stage('finish')
        self.assertEqual(finish.returncode, 0, finish.stderr)
        self.assertTrue((self.root / 'report-generated').exists())
        self.assertEqual(sum(c[0] == 'allure' for c in self.commands()), 1)
        results = [json.loads(p.read_text()) for p in (self.output / 'allure-results').glob('*-result.json')]
        self.assertEqual({r['name']: r['status'] for r in results},
                         {'unit': 'passed', 'integration': 'passed', 'e2e': 'skipped', 'cleanup': 'passed'})

    def assert_failure_stops_chain(self, failed_stage):
        self.env['FAKE_FAIL_STAGE'] = failed_stage
        order = ['unit', 'integration', 'e2e']
        for stage in order[:order.index(failed_stage)]:
            result = self.run_stage(stage)
            self.assertEqual(result.returncode, 0, result.stderr)
        result = self.run_stage(failed_stage)
        self.assertNotEqual(result.returncode, 0)
        before = self.commands()
        for stage in order[order.index(failed_stage) + 1:]:
            self.assertNotEqual(self.run_stage(stage).returncode, 0)
        self.assertEqual(self.commands(), before, 'A later stage executed after failure')
        finish = self.run_stage('finish')
        self.assertNotEqual(finish.returncode, 0)
        state = self.state()
        self.assertEqual(state['stages'][failed_stage]['status'], 'failed')
        for stage in order[order.index(failed_stage) + 1:]:
            self.assertEqual(state['stages'][stage]['status'], 'skipped')
        self.assertTrue(state['cleanup']['restored'])
        self.assertTrue((self.root / 'report-generated').exists())

    def test_unit_failure_blocks_later_stages_but_finish_still_cleans_and_reports(self):
        self.assert_failure_stops_chain('unit')

    def test_integration_failure_blocks_e2e_but_finish_still_cleans_and_reports(self):
        self.assert_failure_stops_chain('integration')

    def test_e2e_failure_still_cleans_and_reports(self):
        self.assert_failure_stops_chain('e2e')

    def test_cannot_start_integration_before_unit(self):
        result = self.run_stage('integration')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.commands(), [])

    def test_local_full_run_stops_after_failure_and_generates_report(self):
        self.env['FAKE_FAIL_STAGE'] = 'integration'
        result = self.run_stage('run')
        self.assertNotEqual(result.returncode, 0)
        state = self.state()
        self.assertEqual({n: r['status'] for n, r in state['stages'].items()},
                         {'unit': 'passed', 'integration': 'failed', 'e2e': 'skipped'})
        self.assertTrue(state['cleanup']['restored'])
        self.assertTrue((self.root / 'report-generated').exists())
        self.assertEqual(sum(c[0] == 'allure' for c in self.commands()), 1)


if __name__ == '__main__':
    unittest.main()
