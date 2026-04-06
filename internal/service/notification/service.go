package service

import (
	"WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service"
	"WatchTower/internal/service/common/provider"
	contactdto "WatchTower/internal/service/notification/dto"
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// NotificationService handles alert contact management.
type NotificationService interface {
	CreateTelegramAlertContact(ctx context.Context, dto contactdto.CreateTelegramAlertContactDTO) (*alert.Contact, error)
	GetAlertContacts(ctx context.Context) ([]alert.Contact, error)
	UpdateAlertContact(ctx context.Context, dto contactdto.UpdateAlertContactDTO) error
	DeleteAlertContact(ctx context.Context, contactID uuid.UUID) error
	EnableAlertContact(ctx context.Context, contactID uuid.UUID) error
	DisableAlertContact(ctx context.Context, contactID uuid.UUID) error
}

type notificationService struct {
	ContactRepo  repo.AlertContactRepository
	userProvider provider.UserProvider
	log          *slog.Logger
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	contactRepo repo.AlertContactRepository,
	userProvider provider.UserProvider,
	logger *slog.Logger,
) NotificationService {
	return &notificationService{
		ContactRepo:  contactRepo,
		userProvider: userProvider,
		log:          logger.With("service", "notification"),
	}
}

// CreateTelegramAlertContact creates a new Telegram alert contact.
func (s *notificationService) CreateTelegramAlertContact(
	ctx context.Context,
	dto contactdto.CreateTelegramAlertContactDTO,
) (*alert.Contact, error) {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}

	// Override the DTO's user representation with the one from context to assure secure creation.
	dto.UserLogin = usr.Login

	s.log.Debug("creating telegram alert contact", "user", dto.UserLogin, "name", dto.Name)

	contact, err := alert.NewTelegramAlertContact(
		&user.User{Login: dto.UserLogin},
		dto.Name,
		dto.ChatID,
		dto.BotToken,
	)
	if err != nil {
		s.log.Error("validation failed for telegram alert contact", "error", err)
		return nil, err
	}

	err = s.ContactRepo.Create(ctx, contact)
	if err != nil {
		s.log.Error("failed to create telegram alert contact", "error", err)
		return nil, err
	}

	s.log.Debug("telegram alert contact created", "id", contact.ID, "user", dto.UserLogin)
	return contact, nil
}

// UpdateAlertContact applies a partial update to an existing alert contact.
// Supports renaming, enabling/disabling and type-specific config changes.
func (s *notificationService) UpdateAlertContact(
	ctx context.Context,
	dto contactdto.UpdateAlertContactDTO,
) error {
	s.log.Debug("updating alert contact", "contact_id", dto.ContactID)

	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return err
	}

	contact, err := s.ContactRepo.GetByID(ctx, dto.ContactID)
	if err != nil {
		s.log.Error("failed to get alert contact for update", "contact_id", dto.ContactID, "error", err)
		return err
	}

	if usr.Login != contact.User.Login {
		return service.ErrPermissionDenied
	}

	if err := contact.ApplyUpdate(alert.ContactUpdate{
		Name:         dto.Name,
		IsActive:     dto.IsActive,
		ConfigUpdate: dto.ConfigUpdate,
	}); err != nil {
		s.log.Error("failed to apply update to alert contact", "contact_id", dto.ContactID, "error", err)
		return err
	}

	if err := s.ContactRepo.Update(ctx, contact); err != nil {
		s.log.Error("failed to persist alert contact update", "contact_id", dto.ContactID, "error", err)
		return err
	}

	s.log.Debug("alert contact updated", "contact_id", dto.ContactID)
	return nil
}

// DeleteAlertContact removes an alert contact by its ID.
func (s *notificationService) DeleteAlertContact(ctx context.Context, contactID uuid.UUID) error {
	s.log.Debug("deleting alert contact", "contact_id", contactID)

	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return err
	}

	contact, err := s.ContactRepo.GetByID(ctx, contactID)
	if err != nil {
		s.log.Error("failed to get alert contact for delete", "contact_id", contactID, "error", err)
		return err
	}

	if usr.Login != contact.User.Login {
		return service.ErrPermissionDenied
	}

	if err := s.ContactRepo.DeleteByID(ctx, contactID); err != nil {
		s.log.Error("failed to delete alert contact", "contact_id", contactID, "error", err)
		return err
	}

	s.log.Info("alert contact deleted", "contact_id", contactID)
	return nil
}

// EnableAlertContact activates an alert contact by its ID.
func (s *notificationService) EnableAlertContact(ctx context.Context, contactID uuid.UUID) error {
	s.log.Debug("enabling alert contact", "contact_id", contactID)

	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return err
	}

	contact, err := s.ContactRepo.GetByID(ctx, contactID)
	if err != nil {
		s.log.Error("failed to get alert contact for enable", "contact_id", contactID, "error", err)
		return err
	}

	if usr.Login != contact.User.Login {
		return service.ErrPermissionDenied
	}

	if err := s.ContactRepo.Enable(ctx, contactID); err != nil {
		return err
	}

	s.log.Info("alert contact enabled", "contact_id", contactID)
	return nil
}

// DisableAlertContact deactivates an alert contact by its ID.
func (s *notificationService) DisableAlertContact(ctx context.Context, contactID uuid.UUID) error {
	s.log.Debug("disabling alert contact", "contact_id", contactID)

	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return err
	}

	contact, err := s.ContactRepo.GetByID(ctx, contactID)
	if err != nil {
		s.log.Error("failed to get alert contact for disable", "contact_id", contactID, "error", err)
		return err
	}

	if usr.Login != contact.User.Login {
		return service.ErrPermissionDenied
	}

	if err := s.ContactRepo.Disable(ctx, contactID); err != nil {
		return err
	}

	s.log.Info("alert contact disabled", "contact_id", contactID)
	return nil
}

// GetAlertContacts retrieves all alert contacts for the authorized user.
func (s *notificationService) GetAlertContacts(ctx context.Context) ([]alert.Contact, error) {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Debug("retrieving alert contacts for user", "user_login", usr.Login)

	contacts, err := s.ContactRepo.GetByUserLogin(ctx, usr.Login)
	if err != nil {
		s.log.Error("failed to retrieve alert contacts", "user_login", usr.Login, "error", err)
		return nil, err
	}

	return contacts, nil
}
