package ownership

import (
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func TestGetOwnedMonitor(t *testing.T) {
	tests := []struct {
		name    string
		userErr error
		owner   string
		repoErr error
		wantErr error
	}{
		{name: "returns monitor owned by user", owner: "alice"},
		{name: "rejects another user's monitor", owner: "bob", wantErr: baseservice.ErrPermissionDenied},
		{name: "returns authorization failure", userErr: baseservice.ErrUnauthorized, wantErr: baseservice.ErrUnauthorized},
		{name: "returns repository failure", owner: "alice", repoErr: repo.ErrDB, wantErr: repo.ErrDB},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "common/ownership", "GetOwnedMonitor", "decision-table", "london", func(t *testing.T, _ *allure.Context) {
				ctrl := gomock.NewController(t)
				provider := testmocks.NewMockUserProvider(ctrl)
				repository := testmocks.NewMockMonitorRepository(ctrl)
				monitorID := uuid.New()
				provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, tc.userErr)
				if tc.userErr == nil {
					mon := testutil.NewMonitorBuilder().WithOwner(tc.owner).Build()
					mon.ID = monitorID
					repository.EXPECT().GetByID(gomock.Any(), monitorID).Return(mon, tc.repoErr)
				}
				got, err := GetOwnedMonitor(context.Background(), provider, repository, monitorID)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("GetOwnedMonitor() error = %v, want %v", err, tc.wantErr)
				}
				if tc.wantErr == nil && (got == nil || got.ID != monitorID) {
					t.Fatalf("GetOwnedMonitor() = %#v", got)
				}
			})
		})
	}
}
