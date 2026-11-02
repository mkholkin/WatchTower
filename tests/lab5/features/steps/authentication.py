"""Business-visible steps never interpolate credentials into report text."""

from concurrent.futures import ThreadPoolExecutor
import secrets

from behave import given, then, when

from support import request, require, status


def incorrect_password(fixture):
    return fixture.http("/auth/mfa/login", {
        "login": fixture.account.login, "password": secrets.token_urlsafe(24),
    })


def incorrect_code(fixture):
    account = fixture.account
    code = str((int(account.code) + 1) % 1000000).zfill(6)
    return fixture.http("/auth/mfa/verify", {"challenge_id": account.challenge, "code": code})


def monitors(fixture, token, expected):
    status(request(fixture.api, "GET", "/monitors", token=token), expected)


@given("a fresh registered account")
def registered(context):
    f = context.fixture
    f.account = f.new_account()
    status(f.register(f.account), 201)


@given("another fresh registered account")
def other_registered(context):
    f = context.fixture
    f.other = f.new_account()
    status(f.register(f.other), 201)


@when("the account submits its correct password")
@when("the account opens another login challenge")
def login(context):
    f = context.fixture
    f.login(f.account)


@then("only a challenge is returned")
def challenge_only(context):
    f = context.fixture
    status(f.response, 202)
    require(set(f.response.body) == {"challenge_id"}, "Password stage disclosed more than a challenge")
    require(not f.response.has_session_cookie, "Password stage set a session cookie")
    monitors(f, f.account.challenge, 401)


@then("monitors reject absent and invalid credentials")
def anonymous_denied(context):
    f = context.fixture
    monitors(f, None, 401)
    monitors(f, secrets.token_urlsafe(24), 401)


@when("the account submits its delivered login code")
@when("the account submits its delivered login code again")
def verify(context):
    f = context.fixture
    f.verify(f.account)


@given("the account has a full session")
@when("the account completes MFA with its current password")
def authenticated(context):
    f = context.fixture
    f.authenticate(f.account)


@then("a full session can read monitors")
def full_access(context):
    f = context.fixture
    require(f.account.password_change_required is False, "Full session unexpectedly requires password change")
    monitors(f, f.account.token, 200)


@when("unknown credentials are submitted")
def unknown(context):
    f = context.fixture
    unknown_account = f.new_account()
    f.http("/auth/mfa/login", {"login": unknown_account.login, "password": unknown_account.password})


@when("the account submits an incorrect password")
def bad_password(context):
    incorrect_password(context.fixture)


@then("the response status is {expected:d}")
def response_status(context, expected):
    status(context.fixture.response, expected)


@when("two incorrect passwords are submitted")
def two_passwords(context):
    for _ in range(2):
        status(incorrect_password(context.fixture), 401)


@when("three incorrect passwords are submitted")
def three_passwords(context):
    for expected in (401, 401, 423):
        status(incorrect_password(context.fixture), expected)


@when("the account submits an incorrect login code")
def bad_code(context):
    incorrect_code(context.fixture)


@when("three incorrect login codes are submitted")
def three_codes(context):
    for expected in (401, 401, 423):
        status(incorrect_code(context.fixture), expected)


@then("the account is locked")
def locked(context):
    f = context.fixture
    status(f.http("/auth/mfa/login", {"login": f.account.login, "password": f.account.password}), 423)


@then("the account is locked and its prior session is rejected")
def locked_session(context):
    locked(context)
    monitors(context.fixture, context.fixture.account.token, 401)


@when("six incorrect passwords are submitted concurrently")
def concurrent_failures(context):
    f = context.fixture
    payloads = [{"login": f.account.login, "password": secrets.token_urlsafe(24)} for _ in range(6)]
    with ThreadPoolExecutor(max_workers=6) as pool:
        responses = list(pool.map(lambda payload: request(f.api, "POST", "/auth/mfa/login", payload), payloads))
    require(sorted(response.status for response in responses) == [401, 401, 423, 423, 423, 423],
            "Concurrent failures did not enforce the three-attempt limit")


@then("the account is locked with exactly three recorded failures")
def locked_count(context):
    locked(context)
    f = context.fixture
    row = f.sql("SELECT failed_attempts, locked FROM auth_mfa WHERE login = %s", (f.account.login,), fetch=True)
    require(row == (3, True), "The locked account must have exactly three recorded failures")


@when("its {field} expires in the database")
def expired(context, field):
    require(field in {"login code", "recovery token"}, "Unsupported expiry fixture")
    context.fixture.expire(field)


@when("the delivered login code is submitted twice concurrently")
def concurrent_codes(context):
    f = context.fixture
    payload = {"challenge_id": f.account.challenge, "code": f.account.code}
    with ThreadPoolExecutor(max_workers=2) as pool:
        f.concurrent_responses = list(pool.map(lambda _: request(f.api, "POST", "/auth/mfa/verify", payload), range(2)))


@then("exactly one verification succeeds")
def one_verification(context):
    responses = context.fixture.concurrent_responses
    require(sorted(response.status for response in responses) == [200, 401],
            "Concurrent code redemption must create exactly one session")
    accepted = next(response for response in responses if response.status == 200)
    token = accepted.body.get("access_token")
    require(isinstance(token, str) and bool(token), "Successful verification must create a token")
    monitors(context.fixture, token, 200)


@when("the account replaces its login challenge")
def replace_challenge(context):
    f = context.fixture
    f.account.previous_challenge = f.account.challenge
    f.account.previous_code = f.account.code
    f.login(f.account)
    require(f.account.challenge != f.account.previous_challenge, "Replacement challenge must be fresh")


@when("the account submits its previous challenge and code")
def old_challenge(context):
    f = context.fixture
    f.http("/auth/mfa/verify", {"challenge_id": f.account.previous_challenge, "code": f.account.previous_code})


@when("the account requests recovery")
def recover(context):
    f = context.fixture
    f.recover(f.account)


@when("the account submits its recovery token with a fresh password")
@when("the consumed recovery token is submitted again")
def reset(context):
    f = context.fixture
    f.reset(f.account)


@when("the account recovers with its delivered token and a fresh password")
def recover_reset(context):
    recover(context)
    reset(context)
    status(context.fixture.response, 204)


@then("the old password is rejected and the fresh password completes MFA")
def new_password(context):
    f = context.fixture
    status(f.http("/auth/mfa/login", {"login": f.account.login, "password": f.account.old_password}), 401)
    f.authenticate(f.account)
    full_access(context)


@when("an incorrect recovery token is submitted")
def invalid_recovery(context):
    f = context.fixture
    f.http("/auth/mfa/reset", {
        "login": f.account.login, "token": secrets.token_urlsafe(32), "new_password": secrets.token_urlsafe(24),
    })


@when("known and unknown accounts request recovery")
def unknown_recovery(context):
    f = context.fixture
    unknown_account = f.new_account()
    f.recovery_responses = [
        f.http("/auth/mfa/recover", {"login": login})
        for login in (f.account.login, unknown_account.login)
    ]


@then("both recovery responses are accepted and disclose no secrets")
def same_recovery(context):
    responses = context.fixture.recovery_responses
    for response in responses:
        status(response, 202)
        require(not response.has_session_cookie, "Recovery request must not set a session cookie")
    require(responses[0].body == responses[1].body, "Recovery responses must not disclose account existence")
    require(not any(key in responses[0].body for key in ("token", "secret", "access_token", "challenge_id")),
            "Recovery response disclosed a secret")


@when("its recovery token is submitted for the other account")
def foreign_recovery(context):
    f = context.fixture
    f.http("/auth/mfa/reset", {
        "login": f.other.login, "token": f.account.recovery, "new_password": secrets.token_urlsafe(24),
    })


@when("the other account completes MFA with its original password")
def other_authenticate(context):
    f = context.fixture
    f.authenticate(f.other)


@then("the other account has a full session")
def other_session(context):
    f = context.fixture
    monitors(f, f.other.token, 200)


@given("the password becomes older than ninety days")
def old_password(context):
    context.fixture.expire("password")


@then("the prior full session is rejected")
def aged_session(context):
    f = context.fixture
    monitors(f, f.account.token, 401)


@then("the session permits password change but cannot read monitors")
def restricted(context):
    f = context.fixture
    require(f.account.password_change_required is True, "Expired password must require change")
    monitors(f, f.account.token, 401)


@when("the account changes its password using its session")
def change_password(context):
    f = context.fixture
    a = f.account
    a.new_password = secrets.token_urlsafe(24)
    status(f.http("/auth/mfa/password", {
        "current_password": a.password, "new_password": a.new_password,
    }, token=a.token), 204)
    a.old_password, a.password = a.password, a.new_password


@then("all prior sessions are rejected")
def revoked(context):
    f = context.fixture
    require(bool(f.account.sessions), "Revocation check requires existing sessions")
    for token in f.account.sessions:
        monitors(f, token, 401)


@when("a fresh account registers with a password of {length:d} bytes")
def register_length(context, length):
    f = context.fixture
    f.register(f.new_account(length))


@when("the account attempts to reuse its current password")
def unchanged_password(context):
    f = context.fixture
    f.http("/auth/mfa/password", {
        "current_password": f.account.password, "new_password": f.account.password,
    }, token=f.account.token)


@when("the account attempts to change to a short password")
def short_password(context):
    f = context.fixture
    f.http("/auth/mfa/password", {
        "current_password": f.account.password, "new_password": secrets.token_urlsafe(6),
    }, token=f.account.token)


@when("recovery is attempted with the unchanged password")
def unchanged_reset(context):
    f = context.fixture
    f.reset(f.account, f.account.password)


@when("recovery is attempted with a short password")
def short_reset(context):
    f = context.fixture
    f.reset(f.account, secrets.token_urlsafe(6))


@when("password change is attempted with an incorrect current password")
def wrong_current(context):
    f = context.fixture
    f.http("/auth/mfa/password", {
        "current_password": secrets.token_urlsafe(24), "new_password": secrets.token_urlsafe(24),
    }, token=f.account.token)


@when("legacy registration and login are attempted")
def legacy(context):
    f = context.fixture
    a = f.new_account()
    f.legacy_responses = [
        f.http("/auth/register", {"login": a.login, "password": a.password}),
        f.http("/auth/login", {"login": f.account.login, "password": f.account.password}),
    ]


@then("neither legacy route creates an authenticated session")
def legacy_blocked(context):
    f = context.fixture
    for response in f.legacy_responses:
        require(response.status in {400, 401, 403, 404, 405, 409}, "Legacy authentication route must be disabled")
        require(not response.body.get("access_token") and not response.has_session_cookie,
                "Legacy authentication route created a session")
    monitors(f, None, 401)


@when("the account registers again")
def duplicate(context):
    f = context.fixture
    f.register(f.account)
