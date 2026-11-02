"""Scenario cleanup is mandatory even when a setup or HTTP assertion fails."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from support import Fixture


def before_all(context):
    context.lab5_fixtures = []


def before_scenario(context, scenario):
    context.fixture = Fixture()
    context.lab5_fixtures.append(context.fixture)


def after_scenario(context, scenario):
    context.fixture.cleanup()


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
