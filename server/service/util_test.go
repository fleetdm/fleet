package service

import (
	"strings"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func newTestService(ds fleet.Datastore, rs fleet.QueryResultStore, lq fleet.LiveQueryStore) fleet.Service {
	mailer := &mockMailService{SendEmailFn: func(e fleet.Email) error { return nil }}
	license := fleet.LicenseInfo{Tier: "core"}
	svc, err := NewService(ds, rs, kitlog.NewNopLogger(), config.TestConfig(), mailer, clock.C, nil, lq, ds, license)
	if err != nil {
		panic(err)
	}
	return svc
}

func newTestServiceWithClock(ds fleet.Datastore, rs fleet.QueryResultStore, lq fleet.LiveQueryStore, c clock.Clock) fleet.Service {
	mailer := &mockMailService{SendEmailFn: func(e fleet.Email) error { return nil }}
	license := fleet.LicenseInfo{Tier: "core"}
	svc, err := NewService(ds, rs, kitlog.NewNopLogger(), config.TestConfig(), mailer, c, nil, lq, ds, license)
	if err != nil {
		panic(err)
	}
	return svc
}

func createTestAppConfig(t *testing.T, ds fleet.Datastore) *fleet.AppConfig {
	config := &fleet.AppConfig{
		OrgName:                "Tyrell Corp",
		OrgLogoURL:             "https://tyrell.com/image.png",
		ServerURL:              "https://fleet.tyrell.com",
		SMTPConfigured:         true,
		SMTPSenderAddress:      "test@example.com",
		SMTPServer:             "smtp.tyrell.com",
		SMTPPort:               587,
		SMTPAuthenticationType: fleet.AuthTypeUserNamePassword,
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

func createTestUsers(t *testing.T, ds fleet.Datastore) map[string]fleet.User {
	users := make(map[string]fleet.User)
	for _, u := range testUsers {
		role := "member"
		if strings.Contains(u.Email, "admin") {
			role = "admin"
		}
		user := &fleet.User{
			Name:       "Test Name " + u.Email,
			Email:      u.Email,
			GlobalRole: &role,
		}
		err := user.SetPassword(u.PlaintextPassword, 10, 10)
		require.Nil(t, err)
		user, err = ds.NewUser(user)
		require.Nil(t, err)
		users[user.Email] = *user
	}
	return users
}

var testUsers = map[string]struct {
	Email             string
	PlaintextPassword string
	GlobalRole        *string
}{
	"admin1": {
		PlaintextPassword: "foobarbaz1234!",
		Email:             "admin1@example.com",
		GlobalRole:        ptr.String(fleet.RoleAdmin),
	},
	"user1": {
		PlaintextPassword: "foobarbaz1234!",
		Email:             "user1@example.com",
		GlobalRole:        ptr.String(fleet.RoleMaintainer),
	},
	"user2": {
		PlaintextPassword: "bazfoo1234!",
		Email:             "user2@example.com",
		GlobalRole:        ptr.String(fleet.RoleObserver),
	},
}

type mockMailService struct {
	SendEmailFn func(e fleet.Email) error
	Invoked     bool
}

func (svc *mockMailService) SendEmail(e fleet.Email) error {
	svc.Invoked = true
	return svc.SendEmailFn(e)
}
