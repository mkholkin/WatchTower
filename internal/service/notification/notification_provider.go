package notification

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"fmt"
)

type NotificationProvider interface {
	SendNotification(contact *alert.Contact, msg string) error
}

type NotificationProviderRegistry interface {
	Register(protocol alert.ContactType, prober NotificationProvider)
	Get(protocol alert.ContactType) (NotificationProvider, error)
}

// proberRegistryImpl maps protocols to their Prober implementations.
// To add a new protocol, register a new Prober.
type notificationProviderRegistryImpl struct {
	probers map[alert.ContactType]NotificationProvider
}

// NewProviderRegistry creates an empty ProberRegistry.
func NewProviderRegistry() NotificationProviderRegistry {
	return &notificationProviderRegistryImpl{
		probers: make(map[alert.ContactType]NotificationProvider),
	}
}

// Register adds a Prober for the given protocol.
func (r *notificationProviderRegistryImpl) Register(contactType alert.ContactType, prober NotificationProvider) {
	r.probers[contactType] = prober
}

// Get returns the Prober for the given protocol or an error if none is registered.
func (r *notificationProviderRegistryImpl) Get(contactType alert.ContactType) (NotificationProvider, error) {
	p, ok := r.probers[contactType]
	if !ok {
		return nil, fmt.Errorf("no notification provider registered for contact type %s", contactType)
	}
	return p, nil
}
