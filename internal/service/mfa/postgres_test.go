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

func TestPostgresAuthentication(t *testing.T) {
	url := os.Getenv("MFA_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set MFA_TEST_DATABASE_URL to run real PostgreSQL authentication tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal("database configuration failed")
	}
	defer admin.Close()
	schema := "mfa_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("database configuration failed")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
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
	enroll := func(t *testing.T, name string) {
		t.Helper()
		if err := service.Enroll(ctx, name, name+"@example.org", password); err != nil {
			t.Fatal(err)
		}
	}
	login := func(t *testing.T, name, password string) mfa.AuthResult {
		t.Helper()
		challenge, err := service.BeginLogin(ctx, name, password)
		if err != nil {
			t.Fatal(err)
		}
		result, err := service.Verify(ctx, challenge, sender.secret(name+"@example.org", "login"))
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	assertError := func(t *testing.T, got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
	exec := func(t *testing.T, query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("MFA consumes challenge and rejects legacy algorithms and claims", func(t *testing.T) {
		enroll(t, "flow")
		assertError(t, service.Enroll(ctx, "flow", "flow@example.org", password), mfa.ErrConflict)
		challenge, err := service.BeginLogin(ctx, "flow", password)
		if err != nil {
			t.Fatal(err)
		}
		code := sender.secret("flow@example.org", "login")
		if len(code) != 6 {
			t.Fatal("OTP must have six digits")
		}
		result, err := service.Verify(ctx, challenge, code)
		if err != nil {
			t.Fatal(err)
		}
		if result.PasswordChangeRequired {
			t.Fatal("fresh password restricted")
		}
		if name, err := service.ParseToken(result.AccessToken); err != nil || name != "flow" {
			t.Fatalf("access rejected: %v", err)
		}
		_, err = service.Verify(ctx, challenge, code)
		assertError(t, err, mfa.ErrInvalid)
		for _, mutation := range []string{"legacy", "algorithm", "scope", "mfa", "expired", "issuer", "version"} {
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
			token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(config.JWTSecret))
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.ParseToken(token)
			assertError(t, err, authsvc.ErrTokenInvalid)
		}
	})
	t.Run("failed attempts expire only the challenge and preserve issued access", func(t *testing.T) {
		enroll(t, "locked")
		access := login(t, "locked", password)
		_, err := service.BeginLogin(ctx, "locked", "wrong-password")
		assertError(t, err, mfa.ErrInvalid)
		challenge, err := service.BeginLogin(ctx, "locked", password)
		if err != nil {
			t.Fatal(err)
		}
		for range 3 {
			_, err = service.Verify(ctx, challenge, "invalid")
			assertError(t, err, mfa.ErrInvalid)
		}
		_, err = service.Verify(ctx, challenge, sender.secret("locked@example.org", "login"))
		assertError(t, err, mfa.ErrInvalid)
		if _, err := service.ParseToken(access.AccessToken); err != nil {
			t.Fatal("failed attempts revoked existing session")
		}
		login(t, "locked", password)
		if err := service.Recover(ctx, "locked"); err != nil {
			t.Fatal(err)
		}
		recovery := sender.secret("locked@example.org", "recovery")
		if len(recovery) != 64 {
			t.Fatal("recovery must carry 256 bits")
		}
		assertError(t, service.Reset(ctx, "locked", recovery, password), mfa.ErrPolicy)
		if err := service.Reset(ctx, "locked", recovery, replacementPassword); err != nil {
			t.Fatal(err)
		}
		assertError(t, service.Reset(ctx, "locked", recovery, anotherPassword), mfa.ErrInvalid)
		login(t, "locked", replacementPassword)
	})
	t.Run("expiration restricts access and password change revokes all secrets", func(t *testing.T) {
		enroll(t, "expired")
		original := login(t, "expired", password)
		exec(t, `UPDATE auth_mfa SET password_changed_at=now()-interval '91 days' WHERE login='expired'`)
		_, err := service.ParseToken(original.AccessToken)
		assertError(t, err, authsvc.ErrTokenInvalid)
		restricted := login(t, "expired", password)
		if !restricted.PasswordChangeRequired {
			t.Fatal("expired password must be restricted")
		}
		_, err = service.ParseToken(restricted.AccessToken)
		assertError(t, err, authsvc.ErrTokenInvalid)
		challenge, err := service.BeginLogin(ctx, "expired", password)
		if err != nil {
			t.Fatal(err)
		}
		code := sender.secret("expired@example.org", "login")
		if err := service.Recover(ctx, "expired"); err != nil {
			t.Fatal(err)
		}
		recovery := sender.secret("expired@example.org", "recovery")
		if err := service.ChangePassword(ctx, restricted.AccessToken, password, replacementPassword); err != nil {
			t.Fatal(err)
		}
		assertError(t, service.ChangePassword(ctx, restricted.AccessToken, replacementPassword, anotherPassword), mfa.ErrInvalid)
		_, err = service.Verify(ctx, challenge, code)
		assertError(t, err, mfa.ErrInvalid)
		assertError(t, service.Reset(ctx, "expired", recovery, anotherPassword), mfa.ErrInvalid)
		fresh := login(t, "expired", replacementPassword)
		if fresh.PasswordChangeRequired {
			t.Fatal("changed password restricted")
		}
		if _, err := service.ParseToken(fresh.AccessToken); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("expired OTP and recovery secrets and generic unknown recovery", func(t *testing.T) {
		enroll(t, "timeouts")
		challenge, err := service.BeginLogin(ctx, "timeouts", password)
		if err != nil {
			t.Fatal(err)
		}
		exec(t, `UPDATE auth_mfa SET otp_expires_at=now()-interval '1 second' WHERE login='timeouts'`)
		_, err = service.Verify(ctx, challenge, sender.secret("timeouts@example.org", "login"))
		assertError(t, err, mfa.ErrExpired)
		if err := service.Recover(ctx, "timeouts"); err != nil {
			t.Fatal(err)
		}
		exec(t, `UPDATE auth_mfa SET recovery_expires_at=now()-interval '1 second' WHERE login='timeouts'`)
		assertError(t, service.Reset(ctx, "timeouts", sender.secret("timeouts@example.org", "recovery"), replacementPassword), mfa.ErrExpired)
		before := sender.calls
		if err := service.Recover(ctx, "missing"); err != nil {
			t.Fatal(err)
		}
		if sender.calls != before {
			t.Fatal("unknown recovery sent mail")
		}
	})
	t.Run("parallel verification and reset have exactly one winner", func(t *testing.T) {
		enroll(t, "parallel")
		challenge, err := service.BeginLogin(ctx, "parallel", password)
		if err != nil {
			t.Fatal(err)
		}
		code := sender.secret("parallel@example.org", "login")
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
		race(func() error { _, err := service.Verify(ctx, challenge, code); return err })
		if err := service.Recover(ctx, "parallel"); err != nil {
			t.Fatal(err)
		}
		recovery := sender.secret("parallel@example.org", "recovery")
		race(func() error { return service.Reset(ctx, "parallel", recovery, replacementPassword) })
	})
	t.Run("previously locked accounts can sign in", func(t *testing.T) {
		enroll(t, "previous-lock")
		exec(t, `UPDATE auth_mfa SET locked=true WHERE login='previous-lock'`)
		login(t, "previous-lock", password)
		var locked bool
		if err := pool.QueryRow(ctx, `SELECT locked FROM auth_mfa WHERE login='previous-lock'`).Scan(&locked); err != nil {
			t.Fatal(err)
		}
		if locked {
			t.Fatal("fresh login retained legacy lock flag")
		}
	})
	t.Run("parallel failures cannot lose increments", func(t *testing.T) {
		enroll(t, "attempts")
		results := make(chan error, 8)
		for range 8 {
			go func() { _, err := service.BeginLogin(ctx, "attempts", "wrong-password"); results <- err }()
		}
		for range 8 {
			err := <-results
			if !errors.Is(err, mfa.ErrInvalid) {
				t.Fatal(err)
			}
		}
		var attempts int
		var locked bool
		if err := pool.QueryRow(ctx, `SELECT failed_attempts,locked FROM auth_mfa WHERE login='attempts'`).Scan(&attempts, &locked); err != nil {
			t.Fatal(err)
		}
		if attempts != 3 || locked {
			t.Fatalf("attempts=%d locked=%v", attempts, locked)
		}
		login(t, "attempts", password)
	})
	t.Run("delivery failure is sanitized and rolls back replacement challenge", func(t *testing.T) {
		enroll(t, "delivery")
		challenge, err := service.BeginLogin(ctx, "delivery", password)
		if err != nil {
			t.Fatal(err)
		}
		code := sender.secret("delivery@example.org", "login")
		sender.fail = true
		_, err = service.BeginLogin(ctx, "delivery", password)
		assertError(t, err, mfa.ErrDelivery)
		assertError(t, service.Recover(ctx, "delivery"), mfa.ErrDelivery)
		sender.fail = false
		if _, err := service.Verify(ctx, challenge, code); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ordinary password changes and recovery revoke valid access", func(t *testing.T) {
		enroll(t, "rotation")
		first := login(t, "rotation", password)
		if err := service.ChangePassword(ctx, first.AccessToken, password, replacementPassword); err != nil {
			t.Fatal(err)
		}
		_, err := service.ParseToken(first.AccessToken)
		assertError(t, err, authsvc.ErrTokenInvalid)
		second := login(t, "rotation", replacementPassword)
		if err := service.Recover(ctx, "rotation"); err != nil {
			t.Fatal(err)
		}
		if err := service.Reset(ctx, "rotation", sender.secret("rotation@example.org", "recovery"), anotherPassword); err != nil {
			t.Fatal(err)
		}
		_, err = service.ParseToken(second.AccessToken)
		assertError(t, err, authsvc.ErrTokenInvalid)
		login(t, "rotation", anotherPassword)
	})
	t.Run("successful second factor clears failed attempts", func(t *testing.T) {
		enroll(t, "clear")
		for range 2 {
			_, err := service.BeginLogin(ctx, "clear", "wrong-password")
			assertError(t, err, mfa.ErrInvalid)
		}
		login(t, "clear", password)
		_, err := service.BeginLogin(ctx, "clear", "wrong-password")
		assertError(t, err, mfa.ErrInvalid)
		var attempts int
		if err := pool.QueryRow(ctx, `SELECT failed_attempts FROM auth_mfa WHERE login='clear'`).Scan(&attempts); err != nil {
			t.Fatal(err)
		}
		if attempts != 1 {
			t.Fatalf("counter=%d", attempts)
		}
	})
	t.Run("oversized password cannot authenticate by bcrypt prefix", func(t *testing.T) {
		full := runtimeSecret(t, 36)
		if err := service.Enroll(ctx, "length", "length@example.org", full); err != nil {
			t.Fatal(err)
		}
		_, err := service.BeginLogin(ctx, "length", full+"suffix")
		assertError(t, err, mfa.ErrInvalid)
		access := login(t, "length", full)
		assertError(t, service.ChangePassword(ctx, access.AccessToken, full+"suffix", replacementPassword), mfa.ErrInvalid)
		if err := service.ChangePassword(ctx, access.AccessToken, full, replacementPassword); err != nil {
			t.Fatal(err)
		}
	})
}
