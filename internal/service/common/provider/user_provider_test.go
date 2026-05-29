package provider

import (
	"WatchTower/internal/domain/entity/user"
	auth "WatchTower/internal/service/auth"
	"WatchTower/internal/service/testmocks"
	"context"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestUserProviderImpl_GetAuthorizedUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := testmocks.NewMockUserRepository(ctrl)
	provider := NewUserProvider(userRepo)
	ctx := auth.ContextWithUser(context.Background(), "alice")
	userRepo.EXPECT().GetByLogin(ctx, "alice").Return(&user.User{Login: "alice"}, nil)
	usr, err := provider.GetAuthorizedUser(ctx)
	if err != nil {
		t.Fatalf("GetAuthorizedUser() error = %v", err)
	}
	if usr == nil || usr.Login != "alice" {
		t.Fatalf("unexpected user: %#v", usr)
	}
}
