package testingutils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdmtesting "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// RunServerWithMockedDS runs the fleet server with several mocked DS methods.
//
// NOTE: Assumes the current session is always from the admin user (see ds.SessionByKeyFunc below).
func RunServerWithMockedDS(t *testing.T, opts ...*service.TestServerOpts) (*httptest.Server, *mock.Store) {
	ds := new(mock.Store)
	var users []*fleet.User
	var admin *fleet.User
	ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		if user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin {
			admin = user
		}
		users = append(users, user)
		return user, nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return &fleet.Session{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			ID:              1,
			AccessedAt:      time.Now(),
			UserID:          admin.ID,
			Key:             key,
		}, nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, session *fleet.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return admin, nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return users, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	apnsCert, apnsKey, err := mysql.GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)
	certPEM, keyPEM, tokenBytes, err := mysql.GenerateTestABMAssets(t)
	require.NoError(t, err)
	ds.GetAllMDMConfigAssetsHashesFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName) (map[fleet.MDMAssetName]string, error) {
		return map[fleet.MDMAssetName]string{
			fleet.MDMAssetABMCert:            "abmcert",
			fleet.MDMAssetABMKey:             "abmkey",
			fleet.MDMAssetABMTokenDeprecated: "abmtoken",
			fleet.MDMAssetAPNSCert:           "apnscert",
			fleet.MDMAssetAPNSKey:            "apnskey",
			fleet.MDMAssetCACert:             "scepcert",
			fleet.MDMAssetCAKey:              "scepkey",
		}, nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetABMCert:            {Name: fleet.MDMAssetABMCert, Value: certPEM},
			fleet.MDMAssetABMKey:             {Name: fleet.MDMAssetABMKey, Value: keyPEM},
			fleet.MDMAssetABMTokenDeprecated: {Name: fleet.MDMAssetABMTokenDeprecated, Value: tokenBytes},
			fleet.MDMAssetAPNSCert:           {Name: fleet.MDMAssetAPNSCert, Value: apnsCert},
			fleet.MDMAssetAPNSKey:            {Name: fleet.MDMAssetAPNSKey, Value: apnsKey},
			fleet.MDMAssetCACert:             {Name: fleet.MDMAssetCACert, Value: certPEM},
			fleet.MDMAssetCAKey:              {Name: fleet.MDMAssetCAKey, Value: keyPEM},
		}, nil
	}

	ds.ApplyYaraRulesFunc = func(context.Context, []fleet.YaraRule) error {
		return nil
	}
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}
	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return nil, nil
	}
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}
	var cachedDS fleet.Datastore
	if len(opts) > 0 && opts[0].NoCacheDatastore {
		cachedDS = ds
	} else {
		cachedDS = cached_mysql.New(ds)
	}
	_, server := service.RunServerForTestsWithDS(t, cachedDS, opts...)
	os.Setenv("FLEET_SERVER_ADDRESS", server.URL)

	return server, ds
}

func ServeMDMBootstrapPackage(t *testing.T, pkgPath, pkgName string) (*httptest.Server, int) {
	pkgBytes, err := os.ReadFile(pkgPath)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(pkgBytes)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, pkgName))
		if n, err := w.Write(pkgBytes); err != nil {
			require.NoError(t, err)
			require.Equal(t, len(pkgBytes), n)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, len(pkgBytes)
}

func StartSoftwareInstallerServer(t *testing.T) {
	// start the web server that will serve the installer
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "server", "service", "testdata", "software-installers", "ruby.deb"))
	require.NoError(t, err)

	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, "notfound"):
					w.WriteHeader(http.StatusNotFound)
					return
				case strings.HasSuffix(r.URL.Path, ".txt"):
					w.Header().Set("Content-Type", "text/plain")
					_, _ = w.Write([]byte(`a simple text file`))
					return
				case strings.Contains(r.URL.Path, "toolarge"):
					w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
					var sz int
					for sz < 3000*1024*1024 {
						n, _ := w.Write(b)
						sz += n
					}
				default:
					w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
					_, _ = w.Write(b)
				}
			},
		),
	)
	t.Cleanup(srv.Close)
	t.Setenv("SOFTWARE_INSTALLER_URL", srv.URL)
}
