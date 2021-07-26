package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WatchBeam/clock"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
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

func newTestBasicService(ds fleet.Datastore, rs fleet.QueryResultStore, lq fleet.LiveQueryStore) fleet.Service {
	mailer := &mockMailService{SendEmailFn: func(e fleet.Email) error { return nil }}
	license := fleet.LicenseInfo{Tier: fleet.TierBasic}
	svc, err := NewService(ds, rs, kitlog.NewNopLogger(), config.TestConfig(), mailer, clock.C, nil, lq, ds, license)
	if err != nil {
		panic(err)
	}
	svc, err = eeservice.NewService(svc, ds, kitlog.NewNopLogger(), config.TestConfig(), mailer, clock.C, &license)
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
		role := fleet.RoleObserver
		if strings.Contains(u.Email, "admin") {
			role = fleet.RoleAdmin
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

type TestServerOpts struct {
	Tier string
}

func RunServerForTestsWithDS(t *testing.T, ds fleet.Datastore, opts ...TestServerOpts) (map[string]fleet.User, *httptest.Server) {
	newServiceFunc := newTestService
	if opts != nil && len(opts) > 0 {
		switch opts[0].Tier {
		case fleet.TierBasic:
			newServiceFunc = newTestBasicService
		}
	}
	svc := newServiceFunc(ds, nil, nil)
	users := createTestUsers(t, ds)
	logger := kitlog.NewLogfmtLogger(os.Stdout)

	serverOpts := []kithttp.ServerOption{
		kithttp.ServerBefore(
			setRequestsContexts(svc),
		),
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}
	r := mux.NewRouter()
	limitStore, _ := memstore.New(0)
	ke := MakeFleetServerEndpoints(svc, "", limitStore)
	kh := makeKitHandlers(ke, serverOpts)
	attachFleetAPIRoutes(r, kh)
	attachNewStyleFleetAPIRoutes(r, svc, serverOpts)
	r.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "index")
	}))

	server := httptest.NewServer(r)
	return users, server
}
