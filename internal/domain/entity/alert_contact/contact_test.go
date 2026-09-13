package alert_test

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/service"
	"WatchTower/internal/testutil"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
	"testing"
)

func ptr[T any](v T) *T { return &v }
func TestContact_NewTelegramAlertContact(t *testing.T) {
	cases := []struct {
		name, label, token string
		chat               int64
		missingUser, bad   bool
	}{{"valid", "Ops", "token", 1, false, false}, {"group_chat", "Ops", "token", -42, false, false}, {"missing_user", "Ops", "token", 1, true, true}, {"empty_name", "", "token", 1, false, true}, {"zero_chat", "Ops", "token", 0, false, true}, {"empty_token", "Ops", "", 1, false, true}}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "Contact", "NewTelegramAlertContact", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
				var usr *user.User
				var got *alert.Contact
				var err error
				a.Step("Arrange", func(*allure.Context) {
					if !tc.missingUser {
						usr = (testutil.ObjectMother{}).User()
					}
				})
				a.Step("Act", func(*allure.Context) { got, err = alert.NewTelegramAlertContact(usr, tc.label, tc.chat, tc.token) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
						require.Nil(t, got)
					} else {
						require.NoError(t, err)
						require.True(t, got.IsActive)
						require.Equal(t, alert.ContactTypeTelegram, got.Config.Type())
						require.Equal(t, tc.chat, got.Config.(alert.TelegramContactConfig).ChatID)
					}
				})
			})
		})
	}
}
func TestContact_ApplyUpdate(t *testing.T) {
	cases := []struct {
		name   string
		update alert.ContactUpdate
		bad    bool
	}{{"rename_disable", alert.ContactUpdate{Name: ptr("New"), IsActive: ptr(false)}, false}, {"keep_fields", alert.ContactUpdate{}, false}, {"invalid_name", alert.ContactUpdate{Name: ptr("")}, true}, {"config_update", alert.ContactUpdate{ConfigUpdate: alert.TelegramConfigUpdate{ChatID: ptr(int64(99))}}, false}, {"invalid_config", alert.ContactUpdate{ConfigUpdate: alert.TelegramConfigUpdate{ChatID: ptr(int64(0))}}, true}}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "Contact", "ApplyUpdate", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
				var c *alert.Contact
				var err error
				a.Step("Arrange", func(*allure.Context) { c = (testutil.ObjectMother{}).Contact() })
				a.Step("Act", func(*allure.Context) { err = c.ApplyUpdate(tc.update) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
					} else {
						require.NoError(t, err)
						switch tc.name {
						case "rename_disable":
							require.Equal(t, "New", c.Name)
							require.False(t, c.IsActive)
						case "keep_fields":
							require.Equal(t, "Operations", c.Name)
							require.True(t, c.IsActive)
						case "config_update":
							require.EqualValues(t, 99, c.Config.(alert.TelegramContactConfig).ChatID)
						}
					}
				})
			})
		})
	}
}
func TestTelegramConfigUpdate_Apply(t *testing.T) {
	cases := []struct {
		name   string
		cfg    alert.ContactConfig
		update alert.TelegramConfigUpdate
		bad    bool
	}{{"replace", alert.TelegramContactConfig{ChatID: 1, BotToken: "old"}, alert.TelegramConfigUpdate{ChatID: ptr(int64(2)), BotToken: ptr("new")}, false}, {"preserve", alert.TelegramContactConfig{ChatID: 1, BotToken: "old"}, alert.TelegramConfigUpdate{}, false}, {"zero_chat", alert.TelegramContactConfig{}, alert.TelegramConfigUpdate{ChatID: ptr(int64(0))}, true}, {"empty_token", alert.TelegramContactConfig{}, alert.TelegramConfigUpdate{BotToken: ptr("")}, true}, {"wrong_type", nil, alert.TelegramConfigUpdate{}, true}}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "TelegramConfigUpdate", "Apply", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
				var got alert.ContactConfig
				var err error
				a.Step("Arrange", func(*allure.Context) {})
				a.Step("Act", func(*allure.Context) { got, err = tc.update.Apply(tc.cfg) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
						require.Nil(t, got)
					} else {
						require.NoError(t, err)
						want := alert.TelegramContactConfig{ChatID: 1, BotToken: "old"}
						if tc.name == "replace" {
							want = alert.TelegramContactConfig{ChatID: 2, BotToken: "new"}
						}
						require.Equal(t, want, got)
					}
				})
			})
		})
	}
}

func TestContact_ApplyUpdate_IsAtomic(t *testing.T) {
	testutil.Case(t, "Contact", "ApplyUpdate", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
		var c *alert.Contact
		var before alert.Contact
		var err error
		a.Step("Arrange", func(*allure.Context) { c = (testutil.ObjectMother{}).Contact(); before = *c })
		a.Step("Act", func(*allure.Context) {
			err = c.ApplyUpdate(alert.ContactUpdate{Name: ptr("Changed"), IsActive: ptr(false), ConfigUpdate: alert.TelegramConfigUpdate{ChatID: ptr(int64(0))}})
		})
		a.Step("Assert", func(*allure.Context) { require.ErrorIs(t, err, service.ErrInvalidData); require.Equal(t, before, *c) })
	})
}
