package service

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	contactdto "WatchTower/internal/service/contacts/dto"
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

func contactTestService(t *testing.T) (*contactService, *testmocks.MockAlertContactRepository, *testmocks.MockUserProvider) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repository := testmocks.NewMockAlertContactRepository(ctrl)
	provider := testmocks.NewMockUserProvider(ctrl)
	return NewContactService(repository, provider, testutil.NoopLogger()).(*contactService), repository, provider
}

func TestContactServiceCreateTelegramAlertContact(t *testing.T) {
	t.Run("persists authorized owner and Telegram details", func(t *testing.T) {
		testutil.Case(t, "contacts", "CreateTelegramAlertContact", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, repository, provider := contactTestService(t)
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			repository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *alert.Contact) error {
				cfg, ok := got.Config.(alert.TelegramContactConfig)
				if got.User.Login != "alice" || got.Name != "Operations" || !got.IsActive || !ok || cfg.ChatID != 42 || cfg.BotToken != "token" {
					t.Fatalf("persisted contact = %#v, config=%#v", got, got.Config)
				}
				return nil
			})

			got, err := svc.CreateTelegramAlertContact(context.Background(), contactdto.CreateTelegramAlertContactDTO{UserLogin: "attacker", Name: "Operations", ChatID: 42, BotToken: "token"})
			if err != nil || got == nil || got.User.Login != "alice" {
				t.Fatalf("CreateTelegramAlertContact() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("rejects invalid contact before persistence", func(t *testing.T) {
		testutil.Case(t, "contacts", "CreateTelegramAlertContact", "boundary", "london", func(t *testing.T, _ *allure.Context) {
			svc, _, provider := contactTestService(t)
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			if got, err := svc.CreateTelegramAlertContact(context.Background(), contactdto.CreateTelegramAlertContactDTO{ChatID: 42, BotToken: "token"}); err == nil || got != nil {
				t.Fatalf("CreateTelegramAlertContact() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("returns persistence failure", func(t *testing.T) {
		testutil.Case(t, "contacts", "CreateTelegramAlertContact", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, repository, provider := contactTestService(t)
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			repository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(repo.ErrDB)
			_, err := svc.CreateTelegramAlertContact(context.Background(), contactdto.CreateTelegramAlertContactDTO{Name: "Operations", ChatID: 42, BotToken: "token"})
			if !errors.Is(err, repo.ErrDB) {
				t.Fatalf("CreateTelegramAlertContact() error = %v", err)
			}
		})
	})
}

func TestContactServiceGetAllAlertContacts(t *testing.T) {
	tests := []struct {
		name     string
		contacts []alert.Contact
		repoErr  error
	}{
		{name: "returns authorized user's contacts", contacts: []alert.Contact{*testutil.ObjectMother{}.Contact()}},
		{name: "returns repository failure", repoErr: repo.ErrDB},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "contacts", "GetAllAlertContacts", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
				svc, repository, provider := contactTestService(t)
				provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				repository.EXPECT().GetByUserLogin(gomock.Any(), "alice").Return(tc.contacts, tc.repoErr)
				got, err := svc.GetAllAlertContacts(context.Background())
				if !errors.Is(err, tc.repoErr) || (tc.repoErr == nil && len(got) != 1) {
					t.Fatalf("GetAllAlertContacts() = (%#v, %v)", got, err)
				}
			})
		})
	}
}

func TestContactServiceGetAlertContact(t *testing.T) {
	for _, owner := range testutil.Shuffle(t, []string{"alice", "bob"}) {
		owner := owner
		t.Run("owner="+owner, func(t *testing.T) {
			testutil.Case(t, "contacts", "GetAlertContact", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
				svc, repository, provider := contactTestService(t)
				contact := testutil.ObjectMother{}.Contact()
				contact.User.Login = owner
				provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				repository.EXPECT().GetByID(gomock.Any(), contact.ID).Return(contact, nil)
				got, err := svc.GetAlertContact(context.Background(), contact.ID)
				if owner == "alice" && (err != nil || got != contact) {
					t.Fatalf("GetAlertContact() = (%#v, %v)", got, err)
				}
				if owner != "alice" && !errors.Is(err, baseservice.ErrPermissionDenied) {
					t.Fatalf("GetAlertContact() error = %v", err)
				}
			})
		})
	}
}

func TestContactServiceUpdateAlertContact(t *testing.T) {
	t.Run("persists requested state changes", func(t *testing.T) {
		testutil.Case(t, "contacts", "UpdateAlertContact", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			svc, repository, provider := contactTestService(t)
			contact := testutil.ObjectMother{}.Contact()
			newName, inactive, chatID := "Primary", false, int64(84)
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			repository.EXPECT().GetByID(gomock.Any(), contact.ID).Return(contact, nil)
			repository.EXPECT().Update(gomock.Any(), contact).DoAndReturn(func(_ context.Context, got *alert.Contact) error {
				cfg := got.Config.(alert.TelegramContactConfig)
				if got.Name != newName || got.IsActive || cfg.ChatID != chatID {
					t.Fatalf("updated contact = %#v, config=%#v", got, cfg)
				}
				return nil
			})
			err := svc.UpdateAlertContact(context.Background(), contactdto.UpdateAlertContactDTO{ContactID: contact.ID, Name: &newName, IsActive: &inactive, ConfigUpdate: alert.TelegramConfigUpdate{ChatID: &chatID}})
			if err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("denies another user's contact", func(t *testing.T) {
		testutil.Case(t, "contacts", "UpdateAlertContact", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, repository, provider := contactTestService(t)
			contact := testutil.ObjectMother{}.Contact()
			contact.User.Login = "bob"
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			repository.EXPECT().GetByID(gomock.Any(), contact.ID).Return(contact, nil)
			if err := svc.UpdateAlertContact(context.Background(), contactdto.UpdateAlertContactDTO{ContactID: contact.ID}); !errors.Is(err, baseservice.ErrPermissionDenied) {
				t.Fatalf("UpdateAlertContact() error = %v", err)
			}
		})
	})
}

func TestContactServiceDeleteAlertContact(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		deleteErr error
		wantErr   error
	}{
		{name: "deletes owned contact", owner: "alice"},
		{name: "denies another user's contact", owner: "bob", wantErr: baseservice.ErrPermissionDenied},
		{name: "returns delete failure", owner: "alice", deleteErr: repo.ErrDB, wantErr: repo.ErrDB},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "contacts", "DeleteAlertContact", "decision-table", "london", func(t *testing.T, _ *allure.Context) {
				svc, repository, provider := contactTestService(t)
				contact := testutil.ObjectMother{}.Contact()
				contact.User.Login = tc.owner
				provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				repository.EXPECT().GetByID(gomock.Any(), contact.ID).Return(contact, nil)
				if tc.owner == "alice" {
					repository.EXPECT().DeleteByID(gomock.Any(), contact.ID).Return(tc.deleteErr)
				}
				err := svc.DeleteAlertContact(context.Background(), contact.ID)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("DeleteAlertContact() error = %v, want %v", err, tc.wantErr)
				}
			})
		})
	}
}

func TestContactServiceToggleAlertContact(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		owner   string
		repoErr error
		wantErr error
		invoke  func(*contactService, context.Context, uuid.UUID) error
		expect  func(*testmocks.MockAlertContactRepository, uuid.UUID, error)
	}{
		{name: "enables owned contact", method: "EnableAlertContact", owner: "alice", invoke: (*contactService).EnableAlertContact, expect: func(r *testmocks.MockAlertContactRepository, id uuid.UUID, err error) {
			r.EXPECT().Enable(gomock.Any(), id).Return(err)
		}},
		{name: "enable returns repository failure", method: "EnableAlertContact", owner: "alice", repoErr: repo.ErrDB, wantErr: repo.ErrDB, invoke: (*contactService).EnableAlertContact, expect: func(r *testmocks.MockAlertContactRepository, id uuid.UUID, err error) {
			r.EXPECT().Enable(gomock.Any(), id).Return(err)
		}},
		{name: "enable denies another user's contact", method: "EnableAlertContact", owner: "bob", wantErr: baseservice.ErrPermissionDenied, invoke: (*contactService).EnableAlertContact},
		{name: "disables owned contact", method: "DisableAlertContact", owner: "alice", invoke: (*contactService).DisableAlertContact, expect: func(r *testmocks.MockAlertContactRepository, id uuid.UUID, err error) {
			r.EXPECT().Disable(gomock.Any(), id).Return(err)
		}},
		{name: "disable returns repository failure", method: "DisableAlertContact", owner: "alice", repoErr: repo.ErrDB, wantErr: repo.ErrDB, invoke: (*contactService).DisableAlertContact, expect: func(r *testmocks.MockAlertContactRepository, id uuid.UUID, err error) {
			r.EXPECT().Disable(gomock.Any(), id).Return(err)
		}},
		{name: "disable denies another user's contact", method: "DisableAlertContact", owner: "bob", wantErr: baseservice.ErrPermissionDenied, invoke: (*contactService).DisableAlertContact},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "contacts", tc.method, "decision-table", "london", func(t *testing.T, _ *allure.Context) {
				svc, repository, provider := contactTestService(t)
				contact := testutil.ObjectMother{}.Contact()
				contact.User.Login = tc.owner
				provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				repository.EXPECT().GetByID(gomock.Any(), contact.ID).Return(contact, nil)
				if tc.expect != nil {
					tc.expect(repository, contact.ID, tc.repoErr)
				}
				err := tc.invoke(svc, context.Background(), contact.ID)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("%s() error = %v, want %v", tc.method, err, tc.wantErr)
				}
			})
		})
	}
}
