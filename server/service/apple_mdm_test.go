package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	nanodep_client "github.com/micromdm/nanodep/client"
	nanodep_storage "github.com/micromdm/nanodep/storage"
	"github.com/micromdm/nanomdm/mdm"
	nanomdm_push "github.com/micromdm/nanomdm/push"
	"github.com/stretchr/testify/require"
)

type dummyDEPStorage struct {
	nanodep_storage.AllStorage
	testAuthAddr string
}

func (d dummyDEPStorage) RetrieveAuthTokens(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
	return &nanodep_client.OAuth1Tokens{}, nil
}

func (d dummyDEPStorage) RetrieveConfig(context.Context, string) (*nanodep_client.Config, error) {
	return &nanodep_client.Config{
		BaseURL: d.testAuthAddr,
	}, nil
}

type dummyMDMStorage struct {
	*mysql.NanoMDMStorage
}

func (d dummyMDMStorage) EnqueueCommand(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
	return nil, nil
}

type dummyMDMPusher struct{}

func (d dummyMDMPusher) Push(context.Context, []string) (map[string]*nanomdm_push.Response, error) {
	return nil, nil
}

func setupAppleMDMService(t *testing.T) fleet.Service {
	ds := new(mock.Store)
	cfg := config.TestConfig()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/server/devices"):
			w.Write([]byte("{}"))
			return
		case strings.Contains(r.URL.Path, "/session"):
			w.Write([]byte(`{"auth_session_token": "yoo"}`))
			return
		}
	}))
	svc := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{
		FleetConfig: &cfg,
		MDMStorage:  dummyMDMStorage{},
		DEPStorage:  dummyDEPStorage{testAuthAddr: ts.URL},
		MDMPusher:   dummyMDMPusher{},
	})
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgName: "Foo Inc.",
			},
			ServerSettings: fleet.ServerSettings{
				ServerURL: "https://foo.example.com",
			},
		}, nil
	}
	ds.GetMDMAppleEnrollmentProfileByTokenFunc = func(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
		return nil, nil
	}
	ds.NewMDMAppleEnrollmentProfileFunc = func(ctx context.Context, enrollmentPayload fleet.MDMAppleEnrollmentProfilePayload) (*fleet.MDMAppleEnrollmentProfile, error) {
		return &fleet.MDMAppleEnrollmentProfile{
			ID:            1,
			Token:         "foo",
			Type:          fleet.MDMAppleEnrollmentTypeManual,
			EnrollmentURL: "https://foo.example.com?token=foo",
		}, nil
	}
	ds.GetMDMAppleEnrollmentProfileByTokenFunc = func(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
		return nil, nil
	}
	ds.ListMDMAppleEnrollmentProfilesFunc = func(ctx context.Context) ([]*fleet.MDMAppleEnrollmentProfile, error) {
		return nil, nil
	}
	ds.GetMDMAppleCommandResultsFunc = func(ctx context.Context, commandUUID string) (map[string]*fleet.MDMAppleCommandResult, error) {
		return nil, nil
	}
	ds.NewMDMAppleInstallerFunc = func(ctx context.Context, name string, size int64, manifest string, installer []byte, urlToken string) (*fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.MDMAppleInstallerFunc = func(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.MDMAppleInstallerDetailsByIDFunc = func(ctx context.Context, id uint) (*fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.DeleteMDMAppleInstallerFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.MDMAppleInstallerDetailsByTokenFunc = func(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.ListMDMAppleInstallersFunc = func(ctx context.Context) ([]fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.MDMAppleListDevicesFunc = func(ctx context.Context) ([]fleet.MDMAppleDevice, error) {
		return nil, nil
	}
	return svc
}

func TestAppleMDMAuthorization(t *testing.T) {
	svc := setupAppleMDMService(t)

	checkAuthErr := func(t *testing.T, err error, shouldFailWithAuth bool) {
		t.Helper()

		if shouldFailWithAuth {
			require.Error(t, err)
			require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
		} else {
			require.NoError(t, err)
		}
	}

	testAuthdMethods := func(t *testing.T, user *fleet.User, shouldFailWithAuth bool) {
		ctx := test.UserContext(user)
		_, err := svc.NewMDMAppleEnrollmentProfile(ctx, fleet.MDMAppleEnrollmentProfilePayload{})
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.ListMDMAppleEnrollmentProfiles(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.GetMDMAppleCommandResults(ctx, "foo")
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.UploadMDMAppleInstaller(ctx, "foo", 3, bytes.NewReader([]byte("foo")))
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.GetMDMAppleInstallerByID(ctx, 42)
		checkAuthErr(t, err, shouldFailWithAuth)
		err = svc.DeleteMDMAppleInstaller(ctx, 42)
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.ListMDMAppleInstallers(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.ListMDMAppleDevices(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.ListMDMAppleDEPDevices(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)
		_, _, err = svc.EnqueueMDMAppleCommand(ctx, &fleet.MDMAppleCommand{Command: &mdm.Command{}}, nil, false)
		checkAuthErr(t, err, shouldFailWithAuth)
	}

	// Only global admins can access the endpoints.
	testAuthdMethods(t, test.UserAdmin, false)

	// All other users should not have access to the endpoints.
	for _, user := range []*fleet.User{
		test.UserNoRoles,
		test.UserMaintainer,
		test.UserObserver,
		test.UserTeamAdminTeam1,
	} {
		testAuthdMethods(t, user, true)
	}
	// Token authenticated endpoints can be accessed by anyone.
	ctx := test.UserContext(test.UserNoRoles)
	_, err := svc.GetMDMAppleInstallerByToken(ctx, "foo")
	require.NoError(t, err)
	_, err = svc.GetMDMAppleEnrollmentProfileByToken(ctx, "foo")
	require.NoError(t, err)
	_, err = svc.GetMDMAppleInstallerDetailsByToken(ctx, "foo")
	require.NoError(t, err)
}
