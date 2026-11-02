"""HTTP-only behavior checks; SQL is restricted to expiry, state assertions and fixture lifecycle.

No helper prints payloads, exception causes, URLs, credentials, or database DSNs.
Mailpit API contract: https://mailpit.axllent.org/docs/api-v1/
"""

import json
import os
import secrets
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass, field


def require(condition, message):
    if not condition:
        raise AssertionError(message) from None


@dataclass(repr=False)
class Response:
    status: int
    body: dict = field(default_factory=dict)
    has_session_cookie: bool = False


def request(base, method, path, payload=None, token=None):
    try:
        headers = {"Accept": "application/json"}
        data = None
        if payload is not None:
            headers["Content-Type"] = "application/json"
            data = json.dumps(payload).encode()
        if token is not None:
            headers["Authorization"] = "Bearer " + token
        req = urllib.request.Request(base + path, data, headers, method=method)
        try:
            response = urllib.request.urlopen(req, timeout=10)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            raw = response.read()
            # Errors are checked by status only; their content is never retained.
            body = json.loads(raw) if raw and response.status < 300 else {}
            return Response(response.status, body, bool(response.headers.get("Set-Cookie")))
    except Exception:
        raise AssertionError("HTTP transport or response decoding failed") from None


def status(response, expected):
    require(response.status == expected, f"Expected HTTP {expected}; got HTTP {response.status}")


@dataclass(repr=False)
class Account:
    login: str = field(default_factory=lambda: "lab5_" + secrets.token_hex(12))
    password: str = field(default_factory=lambda: secrets.token_urlsafe(24))
    token: str = ""
    password_change_required: bool = False
    challenge: str = ""
    code: str = ""
    recovery: str = ""
    old_password: str = ""
    new_password: str = ""
    previous_challenge: str = ""
    previous_code: str = ""
    sessions: list = field(default_factory=list)
    messages_seen: set = field(default_factory=set)

    @property
    def email(self):
        return self.login + "@example.test"


class Fixture:
    def __init__(self):
        self.api = os.environ.get("LAB5_API_URL", "http://api:8080").rstrip("/")
        self.mail = os.environ.get("LAB5_MAIL_URL", "http://mailpit:8025").rstrip("/")
        self.accounts = []
        self.account = None
        self.other = None
        self.response = None

    def http(self, path, payload=None, token=None, method="POST"):
        self.response = request(self.api, method, path, payload, token)
        return self.response

    def new_account(self, length=None):
        account = Account()
        if length is not None:
            account.password = secrets.token_urlsafe(length)[:length]
        # Register cleanup BEFORE issuing a request, including failed registration.
        self.accounts.append(account)
        return account

    def register(self, account):
        return self.http("/auth/mfa/register", {
            "login": account.login, "email": account.email, "password": account.password,
        })

    def login(self, account):
        response = self.http("/auth/mfa/login", {
            "login": account.login, "password": account.password,
        })
        status(response, 202)
        require(isinstance(response.body, dict), "Login response must be an object")
        require(set(response.body) == {"challenge_id"}, "Password stage must return only a challenge")
        require(not response.has_session_cookie, "Password stage must not set a session cookie")
        account.challenge = response.body.get("challenge_id")
        require(isinstance(account.challenge, str) and bool(account.challenge), "Login challenge is missing")
        account.code = self.mail_secret(account, "login")
        require(account.code.isascii() and account.code.isdigit() and len(account.code) == 6,
                "Delivered login code must contain six digits")
        return response

    def verify(self, account):
        response = self.http("/auth/mfa/verify", {"challenge_id": account.challenge, "code": account.code})
        if response.status == 200:
            token = response.body.get("access_token")
            require(isinstance(token, str) and bool(token), "Verified session token is missing")
            require(isinstance(response.body.get("password_change_required"), bool),
                    "Password rotation flag is missing")
            account.token = token
            account.password_change_required = response.body["password_change_required"]
            account.sessions.append(token)
        return response

    def authenticate(self, account):
        self.login(account)
        status(self.verify(account), 200)

    def recover(self, account):
        status(self.http("/auth/mfa/recover", {"login": account.login}), 202)
        account.recovery = self.mail_secret(account, "recovery")

    def reset(self, account, password=None):
        if password is None:
            account.new_password = secrets.token_urlsafe(24)
            password = account.new_password
        response = self.http("/auth/mfa/reset", {
            "login": account.login, "token": account.recovery, "new_password": password,
        })
        if response.status == 204:
            account.old_password = account.password
            account.password = password
        return response

    def mail_secret(self, account, kind):
        query = urllib.parse.urlencode({"query": "to:" + account.email, "limit": 100})
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            listing = request(self.mail, "GET", "/api/v1/search?" + query)
            status(listing, 200)
            require(isinstance(listing.body, dict), "Mailbox search response is invalid")
            for summary in listing.body.get("messages", []):
                message_id = summary.get("ID")
                if not message_id or message_id in account.messages_seen:
                    continue
                message = request(self.mail, "GET", "/api/v1/message/" + urllib.parse.quote(message_id, safe=""))
                status(message, 200)
                require(any(recipient.get("Address") == account.email for recipient in message.body.get("To", [])),
                        "Mail message recipient does not match the account")
                try:
                    content = json.loads(message.body.get("Text", ""))
                    value = content.get("secret")
                    delivered_kind = content.get("kind")
                except Exception:
                    raise AssertionError("Mail message has an invalid structured body") from None
                account.messages_seen.add(message_id)
                if delivered_kind == kind:
                    require(isinstance(value, str) and bool(value), "Mail message secret is missing")
                    return value
            time.sleep(0.1)
        raise AssertionError("Expected addressed mail did not arrive") from None

    def sql(self, query, values=(), fetch=False):
        try:
            import psycopg

            with psycopg.connect(os.environ["LAB5_DATABASE_URL"], connect_timeout=10) as connection:
                with connection.cursor() as cursor:
                    cursor.execute(query, values)
                    return cursor.fetchone() if fetch else None
        except Exception:
            raise AssertionError("Database fixture operation failed") from None

    def expire(self, field):
        statements = {
            "login code": "UPDATE auth_mfa SET otp_expires_at = CURRENT_TIMESTAMP - INTERVAL '1 second' WHERE login = %s",
            "recovery token": "UPDATE auth_mfa SET recovery_expires_at = CURRENT_TIMESTAMP - INTERVAL '1 second' WHERE login = %s",
            "password": "UPDATE auth_mfa SET password_changed_at = CURRENT_TIMESTAMP - INTERVAL '91 days' WHERE login = %s",
        }
        self.sql(statements[field], (self.account.login,))

    def cleanup(self):
        remaining = []
        for account in self.accounts:
            try:
                self.sql('DELETE FROM "user" WHERE login = %s', (account.login,))
                row = self.sql('SELECT count(*) FROM "user" WHERE login = %s', (account.login,), fetch=True)
                require(row == (0,), "Disposable account still exists after cleanup")
                row = self.sql("SELECT count(*) FROM auth_mfa WHERE login = %s", (account.login,), fetch=True)
                require(row == (0,), "MFA state still exists after account cleanup")
            except AssertionError:
                remaining.append(account)
        self.accounts = remaining
        require(not remaining, "Disposable account cleanup failed")
