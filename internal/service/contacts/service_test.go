package service

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/user"
	baseservice "WatchTower/internal/service"
	contactdto "WatchTower/internal/service/contacts/dto"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func newNotificationTestService(ctrl *gomock.Controller) (*contactService, *testmocks.MockAlertContactRepository, *testmocks.MockUserProvider) {
	repo := testmocks.NewMockAlertContactRepository(ctrl)
	provider := testmocks.NewMockUserProvider(ctrl)
	logger := testutil.NoopLogger()

	svc := NewContactService(repo, provider, logger).(*contactService)
	return svc, repo, provider
}

func TestNotificationService_CreateTelegramAlertContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, provider := newNotificationTestService(ctrl)
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	repo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&alert.Contact{})).Return(nil)

	_, err := svc.CreateTelegramAlertContact(context.Background(), contactdto.CreateTelegramAlertContactDTO{
		UserLogin: "ignored",
		Name:      "telegram",
		ChatID:    1,
		BotToken:  "token",
	})
	if err != nil {
		t.Fatalf("CreateTelegramAlertContact() error = %v", err)
	}
}

func TestNotificationService_GetAlertContacts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, provider := newNotificationTestService(ctrl)
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	repo.EXPECT().GetByUserLogin(gomock.Any(), "alice").Return([]alert.Contact{{Name: "a"}}, nil)

	contacts, err := svc.GetAllAlertContacts(context.Background())
	if err != nil {
		t.Fatalf("GetAllAlertContacts() error = %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("GetAllAlertContacts() len = %d, want 1", len(contacts))
	}
}

func TestNotificationService_UpdateAlertContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, provider := newNotificationTestService(ctrl)
	id := uuid.New()
	name := "new"
	active := true

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&alert.Contact{ID: id, User: &user.User{Login: "alice"}, Name: "old", Config: alert.TelegramContactConfig{ChatID: 1, BotToken: "token"}}, nil)
	repo.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&alert.Contact{})).Return(nil)

	err := svc.UpdateAlertContact(context.Background(), contactdto.UpdateAlertContactDTO{ContactID: id, Name: &name, IsActive: &active})
	if err != nil {
		t.Fatalf("UpdateAlertContact() error = %v", err)
	}
}

func TestNotificationService_DeleteAlertContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, provider := newNotificationTestService(ctrl)
	id := uuid.New()

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&alert.Contact{ID: id, User: &user.User{Login: "alice"}}, nil)
	repo.EXPECT().DeleteByID(gomock.Any(), id).Return(nil)

	err := svc.DeleteAlertContact(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteAlertContact() error = %v", err)
	}
}

func TestNotificationService_EnableAlertContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, provider := newNotificationTestService(ctrl)
	id := uuid.New()

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&alert.Contact{ID: id, User: &user.User{Login: "alice"}}, nil)
	repo.EXPECT().Enable(gomock.Any(), id).Return(nil)

	err := svc.EnableAlertContact(context.Background(), id)
	if err != nil {
		t.Fatalf("EnableAlertContact() error = %v", err)
	}
}

func TestNotificationService_DisableAlertContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, provider := newNotificationTestService(ctrl)
	id := uuid.New()

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&alert.Contact{ID: id, User: &user.User{Login: "alice"}}, nil)
	repo.EXPECT().Disable(gomock.Any(), id).Return(nil)

	err := svc.DisableAlertContact(context.Background(), id)
	if err != nil {
		t.Fatalf("DisableAlertContact() error = %v", err)
	}
}

func TestNotificationService_UpdateAlertContact_PermissionDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, repo, provider := newNotificationTestService(ctrl)
	id := uuid.New()

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&alert.Contact{ID: id, User: &user.User{Login: "bob"}}, nil)

	err := svc.UpdateAlertContact(context.Background(), contactdto.UpdateAlertContactDTO{ContactID: id})
	if err != baseservice.ErrPermissionDenied {
		t.Fatalf("UpdateAlertContact() error = %v, want %v", err, baseservice.ErrPermissionDenied)
	}
}
