//go:build lab2integration

package integration

import (
	"context"
	"errors"
	"testing"

	auth "WatchTower/internal/service/auth"
	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestAuthPersistsCredentialsAndRejectsImpersonation(t *testing.T) {
	testsupport.Case(t, "integration", "auth-credentials", func(t *testing.T, a *allure.Context) {
		f := setup(t, false)
		ctx := context.Background()
		a.Step("Регистрация сохраняет хэш пароля", func(*allure.Context) {
			if err := f.auth.Register(ctx, "alice", "correct-password"); err != nil {
				t.Fatal(err)
			}
			u, err := f.users.GetByLogin(ctx, "alice")
			if err != nil {
				t.Fatal(err)
			}
			if u.PasswordHash == "correct-password" || u.PasswordHash == "" {
				t.Fatalf("password stored incorrectly: %q", u.PasswordHash)
			}
		})
		a.Step("Повторная регистрация и неверный пароль отклоняются", func(*allure.Context) {
			if err := f.auth.Register(ctx, "alice", "another"); !errors.Is(err, auth.ErrUserAlreadyExists) {
				t.Fatalf("duplicate registration: %v", err)
			}
			if _, err := f.auth.Login(ctx, "alice", "wrong"); !errors.Is(err, auth.ErrInvalidCredentials) {
				t.Fatalf("wrong password: %v", err)
			}
		})
		a.Step("Токен содержит владельца и отвергает подмену", func(*allure.Context) {
			token, err := f.auth.Login(ctx, "alice", "correct-password")
			if err != nil {
				t.Fatal(err)
			}
			login, err := f.auth.ParseToken(token)
			if err != nil || login != "alice" {
				t.Fatalf("parsed token = %q, %v", login, err)
			}
			if _, err := f.auth.ParseToken(token + "changed"); !errors.Is(err, auth.ErrTokenInvalid) {
				t.Fatalf("tampered token: %v", err)
			}
		})
	})
}
