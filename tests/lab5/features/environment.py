"""Scenario cleanup is mandatory even when a setup or HTTP assertion fails."""

import sys
import os
import json
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from support import Fixture


def before_all(context):
    context.lab5_fixtures = []


def progress(scenario, status):
    path = os.environ.get('LAB5_PROGRESS_FILE')
    if path:
        with Path(path).open('a') as stream:
            stream.write(json.dumps({'line': int(scenario.line), 'status': status})+'\n')


def before_scenario(context, scenario):
    progress(scenario, 'run')
    context.fixture = Fixture()
    context.lab5_fixtures.append(context.fixture)


def after_scenario(context, scenario):
    try:
        context.fixture.cleanup()
    except Exception:
        progress(scenario, 'failed')
        raise
    progress(scenario, getattr(scenario.status, 'name', str(scenario.status)))


def after_all(context):
    # Retry cleanup if an earlier after_scenario hook could not reach PostgreSQL.
    failed = False
    for fixture in context.lab5_fixtures:
        try:
            fixture.cleanup()
        except AssertionError:
            failed = True
    if failed:
        raise AssertionError("Disposable account cleanup failed") from None
