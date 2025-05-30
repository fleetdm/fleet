package integrationtest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type WithDS struct {
	Suite *suite.Suite
	DS    *mysql.Datastore
}

func (ts *WithDS) SetupSuite(dbName string) {
	t := ts.Suite.T()
	ts.DS = mysql.CreateNamedMySQLDS(t, dbName)
	test.AddAllHostsLabel(t, ts.DS)

	// Set up the required fields on AppConfig
	appConf, err := ts.DS.AppConfig(context.Background())
	require.NoError(t, err)
	appConf.OrgInfo.OrgName = "FleetTest"
	appConf.ServerSettings.ServerURL = "https://example.org"
	err = ts.DS.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)
}

func (ts *WithDS) TearDownSuite() {
	_ = ts.DS.Close()
}

type WithServer struct {
	WithDS

	Server *httptest.Server
	Users  map[string]fleet.User
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (ts *WithServer) GetTestToken(email string, password string) string {
	params := loginRequest{
		Email:    email,
		Password: password,
	}
	j, err := json.Marshal(&params)
	require.NoError(ts.Suite.T(), err)

	requestBody := io.NopCloser(bytes.NewBuffer(j))
	resp, err := http.Post(ts.Server.URL+"/api/latest/fleet/login", "application/json", requestBody)
	require.NoError(ts.Suite.T(), err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(ts.Suite.T(), http.StatusOK, resp.StatusCode)

	jsn := struct {
		User  *fleet.User         `json:"user"`
		Token string              `json:"token"`
		Err   []map[string]string `json:"errors,omitempty"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.NoError(ts.Suite.T(), err)
	require.Len(ts.Suite.T(), jsn.Err, 0)

	return jsn.Token
}
