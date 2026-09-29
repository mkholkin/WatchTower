//go:build lab2integration

package integration

import (
	"errors"
	"testing"

	"WatchTower/internal/service"
	cdto "WatchTower/internal/service/contacts/dto"
	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestContactLinkPersistsAndEnforcesOwner(t *testing.T) {
	testsupport.Case(t, "integration", "contact-link-ownership", func(t *testing.T, a *allure.Context) {
		f := setup(t, false)
		alice := f.user(t, "alice")
		bob := f.user(t, "bob")
		m := f.createMonitor(t, alice, "API", "http://probe:8081/health", 30)
		contact, err := f.contacts.CreateTelegramAlertContact(alice, cdto.CreateTelegramAlertContactDTO{Name: "Ops", ChatID: 42, BotToken: "test-token"})
		if err != nil {
			t.Fatal(err)
		}
		a.Step("Контакт связывается с монитором и читается из БД", func(*allure.Context) {
			if err := f.monitoring.LinkAlertContact(alice, m.ID, contact.ID); err != nil {
				t.Fatal(err)
			}
			got, err := f.monitoring.GetMonitor(alice, m.ID)
			if err != nil || len(got.AlertContacts) != 1 || got.AlertContacts[0].ID != contact.ID {
				t.Fatalf("linked contacts = %#v, %v", got, err)
			}
		})
		a.Step("Чужой пользователь не читает контакт и не удаляет связь", func(*allure.Context) {
			if _, err := f.contacts.GetAlertContact(bob, contact.ID); !errors.Is(err, service.ErrPermissionDenied) {
				t.Fatalf("foreign contact read: %v", err)
			}
			if err := f.monitoring.UnlinkAlertContact(bob, m.ID, contact.ID); !errors.Is(err, service.ErrPermissionDenied) {
				t.Fatalf("foreign unlink: %v", err)
			}
		})
		a.Step("Владелец разрывает связь, запись контакта остается", func(*allure.Context) {
			if err := f.monitoring.UnlinkAlertContact(alice, m.ID, contact.ID); err != nil {
				t.Fatal(err)
			}
			got, err := f.monitoring.GetMonitor(alice, m.ID)
			if err != nil || len(got.AlertContacts) != 0 {
				t.Fatalf("contacts after unlink = %#v, %v", got, err)
			}
			if _, err := f.contacts.GetAlertContact(alice, contact.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
}
