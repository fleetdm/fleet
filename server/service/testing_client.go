package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/sso"
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

	server           *httptest.Server
	users            map[string]fleet.User
	token            string
	cachedAdminToken string
}

func (ts *withServer) SetupSuite(dbName string) {
	ts.withDS.SetupSuite(dbName)

	rs := pubsub.NewInmemQueryResults()
	users, server := RunServerForTestsWithDS(ts.s.T(), ts.ds, &TestServerOpts{Rs: rs})
	ts.server = server
	ts.users = users
	ts.token = ts.getTestAdminToken()
	ts.cachedAdminToken = ts.token
}

func (ts *withServer) TearDownSuite() {
	ts.withDS.TearDownSuite()
}

func (ts *withServer) Do(verb, path string, params interface{}, expectedStatusCode int, queryParams ...string) *http.Response {
	t := ts.s.T()

	j, err := json.Marshal(params)
	require.NoError(t, err)

	resp := ts.DoRaw(verb, path, j, expectedStatusCode, queryParams...)

	t.Cleanup(func() {
		resp.Body.Close()
	})
	return resp
}

func (ts *withServer) DoRawWithHeaders(
	verb string, path string, rawBytes []byte, expectedStatusCode int, headers map[string]string, queryParams ...string,
) *http.Response {
	t := ts.s.T()

	requestBody := io.NopCloser(bytes.NewBuffer(rawBytes))
	req, err := http.NewRequest(verb, ts.server.URL+path, requestBody)
	require.NoError(t, err)
	for key, val := range headers {
		req.Header.Add(key, val)
	}
	client := fleethttp.NewClient()

	if len(queryParams)%2 != 0 {
		require.Fail(t, "need even number of params: key value")
	}
	if len(queryParams) > 0 {
		q := req.URL.Query()
		for i := 0; i < len(queryParams); i += 2 {
			q.Add(queryParams[i], queryParams[i+1])
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, expectedStatusCode, resp.StatusCode)

	return resp
}

func (ts *withServer) DoRaw(verb string, path string, rawBytes []byte, expectedStatusCode int, queryParams ...string) *http.Response {
	return ts.DoRawWithHeaders(verb, path, rawBytes, expectedStatusCode, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ts.token),
	}, queryParams...)
}

func (ts *withServer) DoRawNoAuth(verb string, path string, rawBytes []byte, expectedStatusCode int) *http.Response {
	return ts.DoRawWithHeaders(verb, path, rawBytes, expectedStatusCode, nil)
}

func (ts *withServer) DoJSON(verb, path string, params interface{}, expectedStatusCode int, v interface{}, queryParams ...string) {
	resp := ts.Do(verb, path, params, expectedStatusCode, queryParams...)
	err := json.NewDecoder(resp.Body).Decode(v)
	require.NoError(ts.s.T(), err)
	if e, ok := v.(errorer); ok {
		require.NoError(ts.s.T(), e.error())
	}
}

func (ts *withServer) getTestAdminToken() string {
	testUser := testUsers["admin1"]

	// because the login endpoint is rate-limited, use the cached admin token
	// if available (if for some reason a test needs to logout the admin user,
	// then set cachedAdminToken = "" so that a new token is retrieved).
	if ts.cachedAdminToken == "" {
		ts.cachedAdminToken = ts.getTestToken(testUser.Email, testUser.PlaintextPassword)
	}
	return ts.cachedAdminToken
}

func (ts *withServer) getTestToken(email string, password string) string {
	params := loginRequest{
		Email:    email,
		Password: password,
	}
	j, err := json.Marshal(&params)
	require.NoError(ts.s.T(), err)

	requestBody := io.NopCloser(bytes.NewBuffer(j))
	resp, err := http.Post(ts.server.URL+"/api/latest/fleet/login", "application/json", requestBody)
	require.NoError(ts.s.T(), err)
	defer resp.Body.Close()
	assert.Equal(ts.s.T(), http.StatusOK, resp.StatusCode)

	jsn := struct {
		User  *fleet.User         `json:"user"`
		Token string              `json:"token"`
		Err   []map[string]string `json:"errors,omitempty"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.NoError(ts.s.T(), err)
	require.Len(ts.s.T(), jsn.Err, 0)

	return jsn.Token
}

func (ts *withServer) applyConfig(spec []byte) {
	var appConfigSpec interface{}
	err := yaml.Unmarshal(spec, &appConfigSpec)
	require.NoError(ts.s.T(), err)

	ts.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)
}

func (ts *withServer) getConfig() *appConfigResponse {
	var responseBody *appConfigResponse
	ts.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &responseBody)
	return responseBody
}

func (ts *withServer) LoginSSOUser(username, password string) (fleet.Auth, string) {
	t := ts.s.T()

	if _, ok := os.LookupEnv("SAML_IDP_TEST"); !ok {
		t.Skip("SSO tests are disabled")
	}

	var resIni initiateSSOResponse
	ts.DoJSON("POST", "/api/v1/fleet/sso", map[string]string{}, http.StatusOK, &resIni)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := fleethttp.NewClient(
		fleethttp.WithFollowRedir(false),
		fleethttp.WithCookieJar(jar),
	)

	resp, err := client.Get(resIni.URL)
	require.NoError(t, err)

	// From the redirect Location header we can get the AuthState and the URL to
	// which we submit the credentials
	parsed, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	data := url.Values{
		"username":  {username},
		"password":  {password},
		"AuthState": {parsed.Query().Get("AuthState")},
	}
	resp, err = client.PostForm(parsed.Scheme+"://"+parsed.Host+parsed.Path, data)
	require.NoError(t, err)

	// The response is an HTML form, we can extract the base64-encoded response
	// to submit to the Fleet server from here
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	re := regexp.MustCompile(`value="(.*)"`)
	matches := re.FindSubmatch(body)
	require.NotEmptyf(t, matches, "callback HTML doesn't contain a SAMLResponse value, got body: %s", body)
	rawSSOResp := string(matches[1])

	auth, err := sso.DecodeAuthResponse(rawSSOResp)
	require.NoError(t, err)
	q := url.QueryEscape(rawSSOResp)
	res := ts.DoRawNoAuth("POST", "/api/v1/fleet/sso/callback?SAMLResponse="+q, nil, http.StatusOK)

	defer res.Body.Close()
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	return auth, string(body)
}
