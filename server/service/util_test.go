package service

import (
	"testing"

	"github.com/WatchBeam/clock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/require"
)

func newTestService(ds kolide.Datastore, rs kolide.QueryResultStore) (kolide.Service, error) {
	mailer := &mockMailService{SendEmailFn: func(e kolide.Email) error { return nil }}
	return NewService(ds, rs, kitlog.NewNopLogger(), config.TestConfig(), mailer, clock.C)
}

func newTestServiceWithClock(ds kolide.Datastore, rs kolide.QueryResultStore, c clock.Clock) (kolide.Service, error) {
	mailer := &mockMailService{SendEmailFn: func(e kolide.Email) error { return nil }}
	return NewService(ds, rs, kitlog.NewNopLogger(), config.TestConfig(), mailer, c)
}

func createTestAppConfig(t *testing.T, ds kolide.Datastore) *kolide.AppConfig {
	config := &kolide.AppConfig{
		OrgName:                "Tyrell Corp",
		OrgLogoURL:             "https://tyrell.com/image.png",
		KolideServerURL:        "https://kolide.tyrell.com",
		SMTPConfigured:         true,
		SMTPSenderAddress:      "kolide@tyrell.com",
		SMTPServer:             "smtp.tyrell.com",
		SMTPPort:               587,
		SMTPAuthenticationType: kolide.AuthTypeUserNamePassword,
		SMTPUserName:           "deckard",
		SMTPPassword:           "replicant",
		SMTPVerifySSLCerts:     true,
		SMTPEnableTLS:          true,
	}
	result, err := ds.NewAppConfig(config)
	require.Nil(t, err)
	require.NotNil(t, result)
	return result
}

func createTestUsers(t *testing.T, ds kolide.Datastore) map[string]kolide.User {
	users := make(map[string]kolide.User)
	for _, u := range testUsers {
		user := &kolide.User{
			Name:     "Test Name " + u.Username,
			Username: u.Username,
			Email:    u.Email,
			Admin:    u.IsAdmin,
			Enabled:  u.Enabled,
		}
		err := user.SetPassword(u.PlaintextPassword, 10, 10)
		require.Nil(t, err)
		user, err = ds.NewUser(user)
		require.Nil(t, err)
		users[user.Username] = *user
	}
	return users
}

var testUsers = map[string]struct {
	Username          string
	Email             string
	PlaintextPassword string
	IsAdmin           bool
	Enabled           bool
}{
	"admin1": {
		Username:          "admin1",
		PlaintextPassword: "foobar",
		Email:             "admin1@example.com",
		IsAdmin:           true,
		Enabled:           true,
	},
	"user1": {
		Username:          "user1",
		PlaintextPassword: "foobar",
		Email:             "user1@example.com",
		Enabled:           true,
	},
	"user2": {
		Username:          "user2",
		PlaintextPassword: "bazfoo",
		Email:             "user2@example.com",
		Enabled:           true,
	},
	"disabled1": {
		Username:          "disabled1",
		PlaintextPassword: "bazfoo",
		Email:             "disabled1@example.com",
		Enabled:           false,
	},
}

type mockMailService struct {
	SendEmailFn func(e kolide.Email) error
	Invoked     bool
}

func (svc *mockMailService) SendEmail(e kolide.Email) error {
	svc.Invoked = true
	return svc.SendEmailFn(e)
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func uintPtr(n uint) *uint {
	return &n
}
