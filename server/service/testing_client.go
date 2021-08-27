package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type withDS struct {
	s  *suite.Suite
	ds *mysql.Datastore
}

func (ts *withDS) SetupSuite(dbName string) {
	ts.ds = mysql.CreateNamedMySQLDS(ts.s.T(), dbName)
	test.AddAllHostsLabel(ts.s.T(), ts.ds)
}

func (ts *withDS) TearDownSuite() {
	ts.ds.Close()
}

type withServer struct {
	withDS

	server *httptest.Server
	users  map[string]fleet.User
	token  string
}

func (ts *withServer) SetupSuite(dbName string) {
	ts.withDS.SetupSuite(dbName)

	users, server := RunServerForTestsWithDS(ts.s.T(), ts.ds)
	ts.server = server
	ts.users = users
	ts.token = ts.getTestAdminToken()
}

func (ts *withServer) TearDownSuite() {
	ts.withDS.TearDownSuite()

	ts.server.Close()
}

func (ts *withServer) Do(verb, path string, params interface{}, expectedStatusCode int) *http.Response {
	t := ts.s.T()

	j, err := json.Marshal(params)
	require.NoError(t, err)

	resp := ts.DoRaw(verb, path, j, expectedStatusCode)

	return resp
}

func (ts *withServer) DoRaw(verb string, path string, rawBytes []byte, expectedStatusCode int) *http.Response {
	t := ts.s.T()

	requestBody := io.NopCloser(bytes.NewBuffer(rawBytes))
	req, _ := http.NewRequest(verb, ts.server.URL+path, requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", ts.token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, expectedStatusCode, resp.StatusCode)
	return resp
}

func (ts *withServer) DoJSON(verb, path string, params interface{}, expectedStatusCode int, v interface{}) {
	resp := ts.Do(verb, path, params, expectedStatusCode)
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(v)
	require.NoError(ts.s.T(), err)
	if e, ok := v.(errorer); ok {
		require.NoError(ts.s.T(), e.error())
	}
}

func (ts *withServer) getTestAdminToken() string {
	testUser := testUsers["admin1"]

	params := loginRequest{
		Email:    testUser.Email,
		Password: testUser.PlaintextPassword,
	}
	j, err := json.Marshal(&params)
	require.NoError(ts.s.T(), err)

	requestBody := io.NopCloser(bytes.NewBuffer(j))
	resp, err := http.Post(ts.server.URL+"/api/v1/fleet/login", "application/json", requestBody)
	require.NoError(ts.s.T(), err)
	defer resp.Body.Close()
	assert.Equal(ts.s.T(), http.StatusOK, resp.StatusCode)

	var jsn = struct {
		User  *fleet.User         `json:"user"`
		Token string              `json:"token"`
		Err   []map[string]string `json:"errors,omitempty"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.Nil(ts.s.T(), err)

	return jsn.Token
}

func (ts *withServer) applyConfig(spec []byte) {
	var appConfigSpec interface{}
	err := yaml.Unmarshal(spec, &appConfigSpec)
	require.NoError(ts.s.T(), err)

	resp := ts.Do("PATCH", "/api/v1/fleet/config", appConfigSpec, http.StatusOK)
	resp.Body.Close()
}

func (ts *withServer) getConfig() *appConfigResponse {
	var responseBody *appConfigResponse
	ts.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &responseBody)
	return responseBody
}
