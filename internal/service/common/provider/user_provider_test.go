package provider

import (
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	auth "WatchTower/internal/service/auth"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func TestUserProviderGetAuthorizedUser(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		repoUser  *user.User
		repoErr   error
		wantLogin string
		wantErr   error
		callsRepo bool
	}{
		{name: "loads user named in context", ctx: auth.ContextWithUser(context.Background(), "alice"), repoUser: &user.User{Login: "alice"}, wantLogin: "alice", callsRepo: true},
		{name: "rejects context without user", ctx: context.Background(), wantErr: baseservice.ErrUnauthorized},
		{name: "wraps repository failure", ctx: auth.ContextWithUser(context.Background(), "alice"), repoErr: repo.ErrDB, wantErr: repo.ErrDB, callsRepo: true},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "common/provider", "GetAuthorizedUser", "decision-table", "london", func(t *testing.T, _ *allure.Context) {
				ctrl := gomock.NewController(t)
				repository := testmocks.NewMockUserRepository(ctrl)
				if tc.callsRepo {
					repository.EXPECT().GetByLogin(tc.ctx, "alice").Return(tc.repoUser, tc.repoErr)
				}
				got, err := NewUserProvider(repository).GetAuthorizedUser(tc.ctx)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("GetAuthorizedUser() error = %v, want %v", err, tc.wantErr)
				}
				if tc.wantErr == nil && (got == nil || got.Login != tc.wantLogin) {
					t.Fatalf("GetAuthorizedUser() = %#v", got)
				}
			})
		})
	}
}
