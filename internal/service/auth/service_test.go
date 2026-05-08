package auth_service

import (
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/service/testmocks"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := testmocks.NewMockUserRepository(ctrl)
	svc := NewService(userRepo, "secret", time.Hour)

	userRepo.EXPECT().GetByLogin(gomock.Any(), "alice").Return(nil, errors.New("not found"))
	userRepo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&user.User{})).DoAndReturn(func(_ context.Context, u *user.User) error {
		if u.Login != "alice" {
			t.Fatalf("unexpected login: %s", u.Login)
		}
		if u.PasswordHash == "password" || u.PasswordHash == "" {
			t.Fatalf("password hash was not generated")
		}
		return nil
	})

	if err := svc.Register(context.Background(), "alice", "password"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := testmocks.NewMockUserRepository(ctrl)
	svc := NewService(userRepo, "secret", time.Hour)

	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt hash error: %v", err)
	}

	userRepo.EXPECT().GetByLogin(gomock.Any(), "alice").Return(&user.User{Login: "alice", PasswordHash: string(hash)}, nil)

	token, err := svc.Login(context.Background(), "alice", "password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token == "" {
		t.Fatalf("Login() returned empty token")
	}
}

func TestAuthService_ParseToken(t *testing.T) {
	svc := NewService(nil, "secret", time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "alice",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token error: %v", err)
	}

	login, err := svc.ParseToken(tokenString)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if login != "alice" {
		t.Fatalf("ParseToken() login = %s, want alice", login)
	}
}
