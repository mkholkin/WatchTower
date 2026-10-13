package auth_service

import (
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"testing"
	"time"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func TestAuthServiceRegister(t *testing.T) {
	tests := []struct {
		name      string
		existing  *user.User
		lookupErr error
		createErr error
		wantErr   error
	}{
		{name: "creates user with hashed password", lookupErr: repo.ErrNotFound},
		{name: "rejects duplicate login", existing: &user.User{Login: "alice"}, wantErr: ErrUserAlreadyExists},
		{name: "returns lookup failure without creating", lookupErr: repo.ErrDB, wantErr: repo.ErrDB},
		{name: "returns create failure", lookupErr: repo.ErrNotFound, createErr: repo.ErrDB, wantErr: repo.ErrDB},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "auth", "Register", "equivalence", "london", func(t *testing.T, a *allure.Context) {
				ctrl := gomock.NewController(t)
				repository := testmocks.NewMockUserRepository(ctrl)
				var err error
				a.Step("Arrange user lookup and optional creation result", func(*allure.Context) {
					repository.EXPECT().GetByLogin(gomock.Any(), "alice").Return(tc.existing, tc.lookupErr)
					if errors.Is(tc.lookupErr, repo.ErrNotFound) {
						repository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *user.User) error {
							if got.Login != "alice" || bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte("secret")) != nil {
								t.Fatalf("persisted user has invalid credentials: %#v", got)
							}
							return tc.createErr
						})
					}
				})
				a.Step("Act: register the user", func(*allure.Context) {
					err = NewService(repository, "jwt-secret", time.Hour).Register(context.Background(), "alice", "secret")
				})
				a.Step("Assert: the expected outcome is returned", func(*allure.Context) {
					if !errors.Is(err, tc.wantErr) {
						t.Fatalf("Register() error = %v, want %v", err, tc.wantErr)
					}
				})
			})
		})
	}
}

func TestAuthServiceLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		lookupUser *user.User
		lookupErr  error
		password   string
		wantErr    error
	}{
		{name: "returns signed token for valid credentials", lookupUser: &user.User{Login: "alice", PasswordHash: string(hash)}, password: "secret"},
		{name: "hides repository failure as invalid credentials", lookupErr: repo.ErrDB, password: "secret", wantErr: ErrInvalidCredentials},
		{name: "rejects wrong password", lookupUser: &user.User{Login: "alice", PasswordHash: string(hash)}, password: "wrong", wantErr: ErrInvalidCredentials},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "auth", "Login", "decision-table", "london", func(t *testing.T, _ *allure.Context) {
				ctrl := gomock.NewController(t)
				repository := testmocks.NewMockUserRepository(ctrl)
				repository.EXPECT().GetByLogin(gomock.Any(), "alice").Return(tc.lookupUser, tc.lookupErr)

				tokenString, gotErr := NewService(repository, "jwt-secret", time.Hour).Login(context.Background(), "alice", tc.password)
				if !errors.Is(gotErr, tc.wantErr) {
					t.Fatalf("Login() error = %v, want %v", gotErr, tc.wantErr)
				}
				if tc.wantErr == nil {
					claims := jwt.MapClaims{}
					parsed, parseErr := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) { return []byte("jwt-secret"), nil })
					if parseErr != nil || !parsed.Valid || claims["sub"] != "alice" {
						t.Fatalf("Login() returned invalid token: token=%q claims=%v error=%v", tokenString, claims, parseErr)
					}
				}
			})
		})
	}
}

func TestAuthServiceParseToken(t *testing.T) {
	sign := func(secret string, claims jwt.MapClaims) string {
		t.Helper()
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	tests := []struct {
		name      string
		token     string
		wantLogin string
		wantErr   error
	}{
		{name: "returns subject from valid token", token: sign("jwt-secret", jwt.MapClaims{"sub": "alice", "exp": time.Now().Add(time.Hour).Unix()}), wantLogin: "alice"},
		{name: "rejects token signed with another secret", token: sign("other-secret", jwt.MapClaims{"sub": "alice", "exp": time.Now().Add(time.Hour).Unix()}), wantErr: ErrTokenInvalid},
		{name: "rejects expired token", token: sign("jwt-secret", jwt.MapClaims{"sub": "alice", "exp": time.Now().Add(-time.Hour).Unix()}), wantErr: ErrTokenInvalid},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "auth", "ParseToken", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
				login, gotErr := NewService(nil, "jwt-secret", time.Hour).ParseToken(tc.token)
				if login != tc.wantLogin || !errors.Is(gotErr, tc.wantErr) {
					t.Fatalf("ParseToken() = (%q, %v), want (%q, %v)", login, gotErr, tc.wantLogin, tc.wantErr)
				}
			})
		})
	}
}

func TestAuthContextHelpers(t *testing.T) {
	t.Run("round trip preserves login", func(t *testing.T) {
		testutil.Case(t, "auth", "ContextWithUser/UserFromContext", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			ctx := ContextWithUser(context.Background(), "alice")
			login, ok := UserFromContext(ctx)
			if !ok || login != "alice" {
				t.Fatalf("UserFromContext() = (%q, %v)", login, ok)
			}
		})
	})
	t.Run("missing login is reported", func(t *testing.T) {
		testutil.Case(t, "auth", "UserFromContext", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			login, ok := UserFromContext(context.Background())
			if ok || login != "" {
				t.Fatalf("UserFromContext() = (%q, %v)", login, ok)
			}
		})
	})
}
