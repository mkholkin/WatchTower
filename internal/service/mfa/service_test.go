package mfa_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	authsvc "WatchTower/internal/service/auth"
	"WatchTower/internal/service/mfa"
	"github.com/golang-jwt/jwt/v5"
)

func TestConfigurationAndPasswordPolicy(t *testing.T) {
	if _, err := mfa.New(nil, nil, mfa.Config{}); err == nil {
		t.Fatal("accepted missing dependencies")
	}
	// Policy checks precede persistence and never accept truncated bcrypt input.
	s := &mfa.Service{}
	for _, password := range []string{"", strings.Repeat("a", 73), strings.Repeat("я", 37)} {
		if err := s.Enroll(context.Background(), "test", "test@example.org", password); !errors.Is(err, mfa.ErrPolicy) {
			t.Fatalf("invalid password: %v", err)
		}
	}
	for _, email := range []string{"", "bad", "Name <test@example.org>", "x@example.org\r\nBcc: victim@example.org"} {
		if err := s.Enroll(context.Background(), "test", email, "valid-password"); !errors.Is(err, mfa.ErrPolicy) {
			t.Fatalf("invalid email: %v", err)
		}
	}
}

func TestLegacyEndpointsAndTokensAreRejected(t *testing.T) {
	s := &mfa.Service{}
	if err := s.Register(context.Background(), "user", "valid-password"); !errors.Is(err, authsvc.ErrInvalidCredentials) {
		t.Fatalf("register bypass: %v", err)
	}
	if _, err := s.Login(context.Background(), "user", "valid-password"); !errors.Is(err, authsvc.ErrInvalidCredentials) {
		t.Fatalf("login bypass: %v", err)
	}
	for _, method := range []jwt.SigningMethod{jwt.SigningMethodHS256, jwt.SigningMethodHS384} {
		token, err := jwt.NewWithClaims(method, jwt.MapClaims{"sub": "user", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(""))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.ParseToken(token); !errors.Is(err, authsvc.ErrTokenInvalid) {
			t.Fatalf("legacy token accepted: %v", err)
		}
	}
}
