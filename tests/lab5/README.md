# Lab 5 BDD suite

The suite executes the real WatchTower HTTP API. SMTP messages are received by Mailpit; no authentication service or mail delivery mock is used. Run the complete isolated environment with `make lab5` from the repository root.

## Local checks

The standard library safety tests need no running services:

```sh
python3 -m unittest discover -s tests/lab5 -p 'test_*.py' -v
```

Install the dependencies pinned by `tools/lab5/requirements.txt` to check Gherkin parsing and step resolution:

```sh
python3 -m behave --dry-run --no-capture --no-capture-stderr --no-logcapture --format progress tests/lab5/features
```

A dry run validates 27 scenarios and 136 steps; it does not execute the HTTP checks. The real run uses `allure_behave.formatter:AllureFormatter`. The standalone runner owns report generation and environment teardown.

## Runtime configuration

| Variable | Purpose |
|---|---|
| `LAB5_API_URL` | WatchTower HTTP origin; default `http://api:8080` |
| `LAB5_MAIL_URL` | Mailpit HTTP origin; default `http://mailpit:8025` |
| `LAB5_DATABASE_URL` | PostgreSQL connection string, supplied at runtime by the stand |

No credentials belong in feature files, command arguments, reports, or attachments. Passwords, login codes, recovery tokens, session tokens and request/response bodies remain in process memory. HTTP and SQL failures use fixed messages without their original exception causes. Numeric expected/actual HTTP statuses are safe diagnostics. Capture of stdout, stderr and logging is disabled by the runner.

## Scenario coverage

- Successful MFA; no session or session cookie after the password stage; protected route rejects absent/invalid credentials.
- Unknown credentials, incorrect passwords, challenge expiry after failed attempts, preserved existing sessions, fresh login after failures and password recovery.
- Concurrent password failures and concurrent OTP redemption.
- OTP replay, expiry and challenge replacement.
- Recovery token replay, expiry, invalid tokens, account enumeration resistance and cross-user token binding.
- Ninety-day password rotation, restricted session, voluntary password change, existing session and outstanding challenge revocation.
- Registration byte-length boundaries, duplicate registration, unchanged/empty replacement passwords and incorrect current password.
- Legacy authentication routes cannot bypass MFA.

Requirement tags in the feature file map these behaviors to the Lab 5 requirements document.

## Fixture lifecycle

Every scenario starts with a fresh login and `@example.test` address. `secrets.token_urlsafe` generates passwords in memory. Every account is tracked before attempting registration, so even partially successful or rejected setup reaches cleanup. `after_scenario` deletes each tracked login from `"user"`, then verifies both the user and its cascaded `auth_mfa` state are absent. Failed cleanup attempts do not prevent cleanup of other users; `after_all` retries remaining accounts and fails the run if cleanup is still incomplete. The stand additionally removes its disposable database and mailbox volumes.

Expiry tests update `otp_expires_at`, `recovery_expires_at`, or `password_changed_at` in PostgreSQL. They never wait for the actual five-minute, ten-minute or ninety-day deadline. A read-only assertion checks the accessible account's capped failure count.

Mail retrieval uses [Mailpit's primary API contract](https://mailpit.axllent.org/docs/api-v1/): `GET /api/v1/search?query=to:...` locates addressed messages, then `GET /api/v1/message/{ID}` reads `Text`. The recipient is checked again before parsing the JSON message's `kind` and `secret` fields. Previously consumed message IDs are ignored. Polling waits at most ten seconds for delivery; OTP and recovery expiry use SQL instead of polling.
