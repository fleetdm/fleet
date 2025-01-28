package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli/v2"
)

type withDS struct {
	suite *suite.Suite
	ds    *mysql.Datastore
}

func (ts *withDS) SetupSuite(dbName string) {
	t := ts.suite.T()
	ts.ds = mysql.CreateNamedMySQLDS(t, dbName)
	test.AddAllHostsLabel(t, ts.ds)

	// Set up the required fields on AppConfig
	appConf, err := ts.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConf.OrgInfo.OrgName = "FleetTest"
	appConf.ServerSettings.ServerURL = "https://example.org"
	err = ts.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)
}

func (ts *withDS) TearDownSuite() {
	_ = ts.ds.Close()
}

type withServer struct {
	withDS

	server *httptest.Server
	users  map[string]fleet.User
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (ts *withServer) getTestToken(email string, password string) string {
	params := loginRequest{
		Email:    email,
		Password: password,
	}
	j, err := json.Marshal(&params)
	require.NoError(ts.suite.T(), err)

	requestBody := io.NopCloser(bytes.NewBuffer(j))
	resp, err := http.Post(ts.server.URL+"/api/latest/fleet/login", "application/json", requestBody)
	require.NoError(ts.suite.T(), err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(ts.suite.T(), http.StatusOK, resp.StatusCode)

	jsn := struct {
		User  *fleet.User         `json:"user"`
		Token string              `json:"token"`
		Err   []map[string]string `json:"errors,omitempty"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.NoError(ts.suite.T(), err)
	require.Len(ts.suite.T(), jsn.Err, 0)

	return jsn.Token
}

// runServerWithMockedDS runs the fleet server with several mocked DS methods.
//
// NOTE: Assumes the current session is always from the admin user (see ds.SessionByKeyFunc below).
func runServerWithMockedDS(t *testing.T, opts ...*service.TestServerOpts) (*httptest.Server, *mock.Store) {
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
	apnsCert, apnsKey, err := mysql.GenerateTestCertBytes()
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
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
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

func runAppForTest(t *testing.T, args []string) string {
	w, err := runAppNoChecks(args)
	require.NoError(t, err)
	return w.String()
}

func runAppCheckErr(t *testing.T, args []string, errorMsg string) string {
	w, err := runAppNoChecks(args)
	require.Error(t, err)
	require.Equal(t, errorMsg, err.Error())
	return w.String()
}

func runAppNoChecks(args []string) (*bytes.Buffer, error) {
	// first arg must be the binary name. Allow tests to omit it.
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := createApp(nil, w, os.Stderr, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func runWithErrWriter(args []string, errWriter io.Writer) (*bytes.Buffer, error) {
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := createApp(nil, w, errWriter, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func noopExitErrHandler(c *cli.Context, err error) {}

func serveMDMBootstrapPackage(t *testing.T, pkgPath, pkgName string) (*httptest.Server, int) {
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
