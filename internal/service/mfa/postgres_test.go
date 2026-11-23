package mfa_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	authsvc "WatchTower/internal/service/auth"
	"WatchTower/internal/service/mfa"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type recordingSender struct {
	mu      sync.Mutex
	secrets map[string]string
	fail    bool
	calls   int
}

func (s *recordingSender) Send(_ context.Context, to, kind, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.fail {
		return errors.New("sensitive provider error must never escape")
	}
	s.secrets[to+kind] = secret
	return nil
}
func (s *recordingSender) secret(to, kind string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.secrets[to+kind]
}

// runtimeSecret keeps actual fixture credentials out of source and test output.
func runtimeSecret(t *testing.T, bytes int) string {
	t.Helper()
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		t.Fatal("cannot generate fixture secret")
	}
	return hex.EncodeToString(value)
}

type authenticationFixture struct {
	ctx                                            context.Context
	pool                                           *pgxpool.Pool
	sender                                         *recordingSender
	config                                         mfa.Config
	service                                        *mfa.Service
	password, replacementPassword, anotherPassword string
}

func newAuthenticationFixture(t *testing.T) *authenticationFixture {
	t.Helper()
	url := os.Getenv("MFA_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set MFA_TEST_DATABASE_URL to run real PostgreSQL authentication tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal("database configuration failed")
	}
	t.Cleanup(admin.Close)
	schema := "mfa_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`) })
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("database configuration failed")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err = pool.Exec(ctx, `CREATE TABLE "user" (login varchar PRIMARY KEY,password_hash varchar NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../migrations/20261102120000_auth_mfa.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, strings.Split(string(migration), "-- +goose Down")[0]); err != nil {
		t.Fatal(err)
	}
	sender := &recordingSender{secrets: map[string]string{}}
	config := mfa.Config{JWTSecret: runtimeSecret(t, 32)}
	service, err := mfa.New(pool, sender, config)
	if err != nil {
		t.Fatal(err)
	}
	password := runtimeSecret(t, 24)
	replacementPassword := runtimeSecret(t, 24)
	anotherPassword := runtimeSecret(t, 24)
	return &authenticationFixture{ctx: ctx, pool: pool, sender: sender, config: config, service: service, password: password, replacementPassword: replacementPassword, anotherPassword: anotherPassword}
}

func TestPostgresAuthentication(t *testing.T) {
	f := newAuthenticationFixture(t)
	t.Run("MFA consumes challenge and rejects legacy algorithms and claims", f.consumesChallengeAndRejectsTokens)
	t.Run("failed attempts expire only the challenge and preserve issued access", f.failedAttemptsAndRecovery)
	t.Run("expiration restricts access and password change revokes all secrets", f.passwordExpirationRevokesSecrets)
	t.Run("expired OTP and recovery secrets and generic unknown recovery", f.expiredSecretsAndUnknownRecovery)
	t.Run("parallel verification and reset have exactly one winner", f.parallelVerificationAndReset)
	t.Run("previously locked accounts can sign in", f.previouslyLockedSignIn)
	t.Run("parallel failures cannot lose increments", f.parallelFailures)
	t.Run("delivery failure is sanitized and rolls back replacement challenge", f.deliveryFailureRollsBack)
	t.Run("ordinary password changes and recovery revoke valid access", f.passwordRotationRevokesAccess)
	t.Run("successful second factor clears failed attempts", f.secondFactorClearsAttempts)
	t.Run("oversized password cannot authenticate by bcrypt prefix", f.oversizedPasswordRejected)
}

func (f *authenticationFixture) enroll(t *testing.T, name string) {
	t.Helper()
	if err := f.service.Enroll(f.ctx, name, name+"@example.org", f.password); err != nil {
		t.Fatal(err)
	}
}
func (f *authenticationFixture) login(t *testing.T, name, password string) mfa.AuthResult {
	t.Helper()
	challenge, err := f.service.BeginLogin(f.ctx, name, password)
	if err != nil {
		t.Fatal(err)
	}
	result, err := f.service.Verify(f.ctx, challenge, f.sender.secret(name+"@example.org", "login"))
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func assertError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
func (f *authenticationFixture) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(f.ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

func (f *authenticationFixture) consumesChallengeAndRejectsTokens(t *testing.T) {
	f.enroll(t, "flow")
	assertError(t, f.service.Enroll(f.ctx, "flow", "flow@example.org", f.password), mfa.ErrConflict)
	challenge, err := f.service.BeginLogin(f.ctx, "flow", f.password)
	if err != nil {
		t.Fatal(err)
	}
	code := f.sender.secret("flow@example.org", "login")
	if len(code) != 6 {
		t.Fatal("OTP must have six digits")
	}
	result, err := f.service.Verify(f.ctx, challenge, code)
	if err != nil {
		t.Fatal(err)
	}
	if result.PasswordChangeRequired {
		t.Fatal("fresh password restricted")
	}
	if name, err := f.service.ParseToken(result.AccessToken); err != nil || name != "flow" {
		t.Fatalf("access rejected: %v", err)
	}
	_, err = f.service.Verify(f.ctx, challenge, code)
	assertError(t, err, mfa.ErrInvalid)
	for _, mutation := range []string{"legacy", "algorithm", "scope", "mfa", "expired", "issuer", "version"} {
		f.assertRejectedToken(t, mutation)
	}
}

func (f *authenticationFixture) failedAttemptsAndRecovery(t *testing.T) {
	f.enroll(t, "locked")
	access := f.login(t, "locked", f.password)
	_, err := f.service.BeginLogin(f.ctx, "locked", "wrong-password")
	assertError(t, err, mfa.ErrInvalid)
	challenge, err := f.service.BeginLogin(f.ctx, "locked", f.password)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		_, err = f.service.Verify(f.ctx, challenge, "invalid")
		assertError(t, err, mfa.ErrInvalid)
	}
	_, err = f.service.Verify(f.ctx, challenge, f.sender.secret("locked@example.org", "login"))
	assertError(t, err, mfa.ErrInvalid)
	if _, err := f.service.ParseToken(access.AccessToken); err != nil {
		t.Fatal("failed attempts revoked existing session")
	}
	f.login(t, "locked", f.password)
	if err := f.service.Recover(f.ctx, "locked"); err != nil {
		t.Fatal(err)
	}
	recovery := f.sender.secret("locked@example.org", "recovery")
	if len(recovery) != 64 {
		t.Fatal("recovery must carry 256 bits")
	}
	assertError(t, f.service.Reset(f.ctx, "locked", recovery, f.password), mfa.ErrPolicy)
	if err := f.service.Reset(f.ctx, "locked", recovery, f.replacementPassword); err != nil {
		t.Fatal(err)
	}
	assertError(t, f.service.Reset(f.ctx, "locked", recovery, f.anotherPassword), mfa.ErrInvalid)
	f.login(t, "locked", f.replacementPassword)
}

func (f *authenticationFixture) passwordExpirationRevokesSecrets(t *testing.T) {
	f.enroll(t, "expired")
	original := f.login(t, "expired", f.password)
	f.exec(t, `UPDATE auth_mfa SET password_changed_at=now()-interval '91 days' WHERE login='expired'`)
	_, err := f.service.ParseToken(original.AccessToken)
	assertError(t, err, authsvc.ErrTokenInvalid)
	restricted := f.login(t, "expired", f.password)
	if !restricted.PasswordChangeRequired {
		t.Fatal("expired password must be restricted")
	}
	_, err = f.service.ParseToken(restricted.AccessToken)
	assertError(t, err, authsvc.ErrTokenInvalid)
	challenge, err := f.service.BeginLogin(f.ctx, "expired", f.password)
	if err != nil {
		t.Fatal(err)
	}
	code := f.sender.secret("expired@example.org", "login")
	if err := f.service.Recover(f.ctx, "expired"); err != nil {
		t.Fatal(err)
	}
	recovery := f.sender.secret("expired@example.org", "recovery")
	if err := f.service.ChangePassword(f.ctx, restricted.AccessToken, f.password, f.replacementPassword); err != nil {
		t.Fatal(err)
	}
	assertError(t, f.service.ChangePassword(f.ctx, restricted.AccessToken, f.replacementPassword, f.anotherPassword), mfa.ErrInvalid)
	_, err = f.service.Verify(f.ctx, challenge, code)
	assertError(t, err, mfa.ErrInvalid)
	assertError(t, f.service.Reset(f.ctx, "expired", recovery, f.anotherPassword), mfa.ErrInvalid)
	fresh := f.login(t, "expired", f.replacementPassword)
	if fresh.PasswordChangeRequired {
		t.Fatal("changed password restricted")
	}
	if _, err := f.service.ParseToken(fresh.AccessToken); err != nil {
		t.Fatal(err)
	}
}

func (f *authenticationFixture) expiredSecretsAndUnknownRecovery(t *testing.T) {
	f.enroll(t, "timeouts")
	challenge, err := f.service.BeginLogin(f.ctx, "timeouts", f.password)
	if err != nil {
		t.Fatal(err)
	}
	f.exec(t, `UPDATE auth_mfa SET otp_expires_at=now()-interval '1 second' WHERE login='timeouts'`)
	_, err = f.service.Verify(f.ctx, challenge, f.sender.secret("timeouts@example.org", "login"))
	assertError(t, err, mfa.ErrExpired)
	if err := f.service.Recover(f.ctx, "timeouts"); err != nil {
		t.Fatal(err)
	}
	f.exec(t, `UPDATE auth_mfa SET recovery_expires_at=now()-interval '1 second' WHERE login='timeouts'`)
	assertError(t, f.service.Reset(f.ctx, "timeouts", f.sender.secret("timeouts@example.org", "recovery"), f.replacementPassword), mfa.ErrExpired)
	before := f.sender.calls
	if err := f.service.Recover(f.ctx, "missing"); err != nil {
		t.Fatal(err)
	}
	if f.sender.calls != before {
		t.Fatal("unknown recovery sent mail")
	}
}

func (f *authenticationFixture) parallelVerificationAndReset(t *testing.T) {
	f.enroll(t, "parallel")
	challenge, err := f.service.BeginLogin(f.ctx, "parallel", f.password)
	if err != nil {
		t.Fatal(err)
	}
	code := f.sender.secret("parallel@example.org", "login")
	race := func(operation func() error) {
		t.Helper()
		results := make(chan error, 2)
		for range 2 {
			go func() { results <- operation() }()
		}
		successes := 0
		for range 2 {
			if err := <-results; err == nil {
				successes++
			} else {
				assertError(t, err, mfa.ErrInvalid)
			}
		}
		if successes != 1 {
			t.Fatalf("successes=%d", successes)
		}
	}
	race(func() error { _, err := f.service.Verify(f.ctx, challenge, code); return err })
	if err := f.service.Recover(f.ctx, "parallel"); err != nil {
		t.Fatal(err)
	}
	recovery := f.sender.secret("parallel@example.org", "recovery")
	race(func() error { return f.service.Reset(f.ctx, "parallel", recovery, f.replacementPassword) })
}

func (f *authenticationFixture) previouslyLockedSignIn(t *testing.T) {
	f.enroll(t, "previous-lock")
	f.exec(t, `UPDATE auth_mfa SET locked=true WHERE login='previous-lock'`)
	f.login(t, "previous-lock", f.password)
	var locked bool
	if err := f.pool.QueryRow(f.ctx, `SELECT locked FROM auth_mfa WHERE login='previous-lock'`).Scan(&locked); err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Fatal("fresh login retained legacy lock flag")
	}
}

func (f *authenticationFixture) parallelFailures(t *testing.T) {
	f.enroll(t, "attempts")
	results := make(chan error, 8)
	for range 8 {
		go func() { _, err := f.service.BeginLogin(f.ctx, "attempts", "wrong-password"); results <- err }()
	}
	for range 8 {
		err := <-results
		if !errors.Is(err, mfa.ErrInvalid) {
			t.Fatal(err)
		}
	}
	var attempts int
	var locked bool
	if err := f.pool.QueryRow(f.ctx, `SELECT failed_attempts,locked FROM auth_mfa WHERE login='attempts'`).Scan(&attempts, &locked); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 || locked {
		t.Fatalf("attempts=%d locked=%v", attempts, locked)
	}
	f.login(t, "attempts", f.password)
}

func (f *authenticationFixture) deliveryFailureRollsBack(t *testing.T) {
	f.enroll(t, "delivery")
	challenge, err := f.service.BeginLogin(f.ctx, "delivery", f.password)
	if err != nil {
		t.Fatal(err)
	}
	code := f.sender.secret("delivery@example.org", "login")
	f.sender.fail = true
	_, err = f.service.BeginLogin(f.ctx, "delivery", f.password)
	assertError(t, err, mfa.ErrDelivery)
	assertError(t, f.service.Recover(f.ctx, "delivery"), mfa.ErrDelivery)
	f.sender.fail = false
	if _, err := f.service.Verify(f.ctx, challenge, code); err != nil {
		t.Fatal(err)
	}
}

func (f *authenticationFixture) passwordRotationRevokesAccess(t *testing.T) {
	f.enroll(t, "rotation")
	first := f.login(t, "rotation", f.password)
	if err := f.service.ChangePassword(f.ctx, first.AccessToken, f.password, f.replacementPassword); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.ParseToken(first.AccessToken)
	assertError(t, err, authsvc.ErrTokenInvalid)
	second := f.login(t, "rotation", f.replacementPassword)
	if err := f.service.Recover(f.ctx, "rotation"); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Reset(f.ctx, "rotation", f.sender.secret("rotation@example.org", "recovery"), f.anotherPassword); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.ParseToken(second.AccessToken)
	assertError(t, err, authsvc.ErrTokenInvalid)
	f.login(t, "rotation", f.anotherPassword)
}

func (f *authenticationFixture) secondFactorClearsAttempts(t *testing.T) {
	f.enroll(t, "clear")
	for range 2 {
		_, err := f.service.BeginLogin(f.ctx, "clear", "wrong-password")
		assertError(t, err, mfa.ErrInvalid)
	}
	f.login(t, "clear", f.password)
	_, err := f.service.BeginLogin(f.ctx, "clear", "wrong-password")
	assertError(t, err, mfa.ErrInvalid)
	var attempts int
	if err := f.pool.QueryRow(f.ctx, `SELECT failed_attempts FROM auth_mfa WHERE login='clear'`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("counter=%d", attempts)
	}
}

func (f *authenticationFixture) oversizedPasswordRejected(t *testing.T) {
	full := runtimeSecret(t, 36)
	if err := f.service.Enroll(f.ctx, "length", "length@example.org", full); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.BeginLogin(f.ctx, "length", full+"suffix")
	assertError(t, err, mfa.ErrInvalid)
	access := f.login(t, "length", full)
	assertError(t, f.service.ChangePassword(f.ctx, access.AccessToken, full+"suffix", f.replacementPassword), mfa.ErrInvalid)
	if err := f.service.ChangePassword(f.ctx, access.AccessToken, full, f.replacementPassword); err != nil {
		t.Fatal(err)
	}
}

func (f *authenticationFixture) assertRejectedToken(t *testing.T, mutation string) {
	t.Helper()
	claims := jwt.MapClaims{"sub": "flow", "iss": "watchtower-mfa", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "mfa": true, "version": 1, "scope": "access"}
	method := jwt.SigningMethodHS256
	switch mutation {
	case "legacy":
		delete(claims, "mfa")
		delete(claims, "scope")
	case "algorithm":
		method = jwt.SigningMethodHS384
	case "scope":
		claims["scope"] = "password_change"
	case "mfa":
		claims["mfa"] = false
	case "expired":
		claims["exp"] = time.Now().Add(-time.Minute).Unix()
	case "issuer":
		claims["iss"] = "other"
	case "version":
		claims["version"] = 2
	}
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(f.config.JWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.ParseToken(token)
	assertError(t, err, authsvc.ErrTokenInvalid)
}
