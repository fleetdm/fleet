package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type withDS struct {
	s  *suite.Suite
	ds *mysql.Datastore
}

func (ts *withDS) SetupSuite(dbName string) {
	t := ts.s.T()
	ts.ds = mysql.CreateNamedMySQLDS(t, dbName)
	// remove any migration-created labels
	mysql.ExecAdhocSQL(t, ts.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), `DELETE FROM labels`)
		return err
	})
	test.AddBuiltinLabels(t, ts.ds)

	// setup the required fields on AppConfig
	appConf, err := ts.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConf.OrgInfo.OrgName = "FleetTest"
	appConf.ServerSettings.ServerURL = "https://example.org"
	err = ts.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)
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

	cachedTokensMu sync.Mutex
	cachedTokens   map[string]string // email -> auth token

	lq *live_query_mock.MockLiveQuery
}

func (ts *withServer) SetupSuite(dbName string) {
	ts.withDS.SetupSuite(dbName)

	rs := pubsub.NewInmemQueryResults()
	cfg := config.TestConfig()
	cfg.Osquery.EnrollCooldown = 0
	opts := &TestServerOpts{
		Rs:          rs,
		Lq:          ts.lq,
		FleetConfig: &cfg,
		Pool:        redistest.SetupRedis(ts.s.T(), "integration_core", false, false, false),
	}
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		opts.Logger = kitlog.NewNopLogger()
	}
	users, server := RunServerForTestsWithDS(ts.s.T(), ts.ds, opts)
	ts.server = server
	ts.users = users
	ts.token = ts.getTestAdminToken()
	ts.cachedAdminToken = ts.token
}

func (ts *withServer) TearDownSuite() {
	ts.withDS.TearDownSuite()
}

func (ts *withServer) commonTearDownTest(t *testing.T) {
	// By setting DISABLE_TABLES_CLEANUP a developer can troubleshoot tests
	// by inspecting mysql tables.
	if os.Getenv("DISABLE_CLEANUP_TABLES") != "" {
		return
	}

	ctx := context.Background()

	u := ts.users["admin1@example.com"]
	filter := fleet.TeamFilter{User: &u}
	hosts, err := ts.ds.ListHosts(ctx, filter, fleet.HostListOptions{})
	require.NoError(t, err)
	for _, host := range hosts {
		_, err := ts.ds.UpdateHostSoftware(context.Background(), host.ID, nil)
		require.NoError(t, err)
		require.NoError(t, ts.ds.DeleteHost(ctx, host.ID))
	}

	teams, err := ts.ds.ListTeams(ctx, fleet.TeamFilter{User: &u}, fleet.ListOptions{})
	require.NoError(t, err)
	for _, tm := range teams {
		err := ts.ds.DeleteTeam(ctx, tm.ID)
		require.NoError(t, err)
	}

	mysql.ExecAdhocSQL(t, ts.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM policies;`)
		return err
	})

	// Clean software installers in "No team" (the others are deleted in ts.ds.DeleteTeam above).
	mysql.ExecAdhocSQL(t, ts.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM software_installers WHERE global_or_team_id = 0;`)
		return err
	})

	lbls, err := ts.ds.ListLabels(ctx, fleet.TeamFilter{}, fleet.ListOptions{})
	require.NoError(t, err)
	for _, lbl := range lbls {
		if lbl.LabelType != fleet.LabelTypeBuiltIn {
			err := ts.ds.DeleteLabel(ctx, lbl.Name)
			require.NoError(t, err)
		}
	}

	queries, _, _, err := ts.ds.ListQueries(ctx, fleet.ListQueryOptions{})
	require.NoError(t, err)
	queryIDs := make([]uint, 0, len(queries))
	for _, query := range queries {
		queryIDs = append(queryIDs, query.ID)
	}
	if len(queryIDs) > 0 {
		count, err := ts.ds.DeleteQueries(ctx, queryIDs)
		require.NoError(t, err)
		require.EqualValues(t, len(queries), count)
	}

	users, err := ts.ds.ListUsers(ctx, fleet.UserListOptions{})
	require.NoError(t, err)
	for _, u := range users {
		if _, ok := ts.users[u.Email]; !ok {
			err := ts.ds.DeleteUser(ctx, u.ID)
			require.NoError(t, err)
		}
	}

	// Clean scripts in "No team" (the others are deleted in ts.ds.DeleteTeam above).
	mysql.ExecAdhocSQL(t, ts.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM scripts WHERE global_or_team_id = 0;`)
		return err
	})

	globalPolicies, err := ts.ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	if len(globalPolicies) > 0 {
		var globalPolicyIDs []uint
		for _, gp := range globalPolicies {
			globalPolicyIDs = append(globalPolicyIDs, gp.ID)
		}
		_, err = ts.ds.DeleteGlobalPolicies(ctx, globalPolicyIDs)
		require.NoError(t, err)
	}

	packs, err := ts.ds.ListPacks(ctx, fleet.PackListOptions{})
	require.NoError(t, err)
	for _, pack := range packs {
		err := ts.ds.DeletePack(ctx, pack.Name)
		require.NoError(t, err)
	}

	// Do the software/titles cleanup.
	err = ts.ds.SyncHostsSoftware(ctx, time.Now())
	require.NoError(t, err)
	err = ts.ds.ReconcileSoftwareTitles(ctx)
	require.NoError(t, err)
	err = ts.ds.SyncHostsSoftwareTitles(ctx, time.Now())
	require.NoError(t, err)

	// delete orphaned scripts
	mysql.ExecAdhocSQL(t, ts.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM scripts`)
		return err
	})

	// delete orphaned host_script_results
	mysql.ExecAdhocSQL(t, ts.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_script_results`)
		return err
	})

	mysql.ExecAdhocSQL(t, ts.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM vpp_tokens;")
		return err
	})

	mysql.ExecAdhocSQL(t, ts.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM secret_variables")
		return err
	})
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

	opts := []fleethttp.ClientOpt{}
	if expectedStatusCode >= 300 && expectedStatusCode <= 399 {
		opts = append(opts, fleethttp.WithFollowRedir(false))
	}
	client := fleethttp.NewClient(opts...)

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

	if resp.StatusCode != expectedStatusCode {
		defer resp.Body.Close()
		var je jsonError
		err := json.NewDecoder(resp.Body).Decode(&je)
		if err != nil {
			t.Logf("Error trying to decode response body as Fleet jsonError: %s", err)
			require.Equal(t, expectedStatusCode, resp.StatusCode, fmt.Sprintf("response: %+v", resp))
		}
		require.Equal(t, expectedStatusCode, resp.StatusCode, fmt.Sprintf("Fleet jsonError: %+v", je))
	}

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

func (ts *withServer) DoJSONWithoutAuth(verb, path string, params interface{}, expectedStatusCode int, v interface{}, queryParams ...string) {
	t := ts.s.T()
	rawBytes, err := json.Marshal(params)
	require.NoError(t, err)
	resp := ts.DoRawWithHeaders(verb, path, rawBytes, expectedStatusCode, map[string]string{}, queryParams...)
	t.Cleanup(func() {
		resp.Body.Close()
	})
	err = json.NewDecoder(resp.Body).Decode(v)
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

func (ts *withServer) setTokenForTest(t *testing.T, email, password string) {
	oldToken := ts.token
	t.Cleanup(func() {
		ts.token = oldToken
	})

	ts.token = ts.getCachedUserToken(email, password)
}

// getCachedUserToken returns the cached auth token for the given test user email.
// If it's not found, then a login request is performed and the token cached.
func (ts *withServer) getCachedUserToken(email, password string) string {
	ts.cachedTokensMu.Lock()
	defer ts.cachedTokensMu.Unlock()

	if ts.cachedTokens == nil {
		ts.cachedTokens = make(map[string]string)
	}

	token, ok := ts.cachedTokens[email]
	if !ok {
		token = ts.getTestToken(email, password)
		ts.cachedTokens[email] = token
	}
	return token
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

func (ts *withServer) applyTeamSpec(yamlSpec []byte) {
	var teamSpec any
	err := yaml.Unmarshal(yamlSpec, &teamSpec)
	require.NoError(ts.s.T(), err)

	specsReq := map[string]any{
		"specs": []any{teamSpec},
	}
	ts.Do("POST", "/api/latest/fleet/spec/teams", specsReq, http.StatusOK)
}

func (ts *withServer) LoginSSOUser(username, password string) (fleet.Auth, string) {
	t := ts.s.T()
	auth, res := ts.loginSSOUser(username, password, "/api/v1/fleet/sso", http.StatusOK)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return auth, string(body)
}

func (ts *withServer) LoginMDMSSOUser(username, password string) *http.Response {
	_, res := ts.loginSSOUser(username, password, "/api/v1/fleet/mdm/sso", http.StatusSeeOther)
	return res
}

func (ts *withServer) loginSSOUser(username, password string, basePath string, callbackStatus int) (fleet.Auth, *http.Response) {
	t := ts.s.T()

	if _, ok := os.LookupEnv("SAML_IDP_TEST"); !ok {
		t.Skip("SSO tests are disabled")
	}

	var resIni initiateSSOResponse
	ts.DoJSON("POST", basePath, map[string]string{}, http.StatusOK, &resIni)

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
	res := ts.DoRawNoAuth("POST", basePath+"/callback?SAMLResponse="+q, nil, callbackStatus)

	return auth, res
}

// gets the latest activity and checks that it matches any provided properties.
// empty string or 0 id means do not check that property. It returns the ID of that
// latest activity.
func (ts *withServer) lastActivityMatches(name, details string, id uint) uint {
	t := ts.s.T()
	var listActivities listActivitiesResponse
	ts.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities, "order_key", "a.id", "order_direction", "desc", "per_page", "1")
	require.True(t, len(listActivities.Activities) > 0)

	act := listActivities.Activities[0]
	if name != "" {
		assert.Equal(t, name, act.Type)
	}
	if details != "" {
		require.NotNil(t, act.Details)
		assert.JSONEq(t, details, string(*act.Details))
	}
	if id > 0 {
		assert.Equal(t, id, act.ID)
	}
	return act.ID
}

// gets the latest activity with the specified type name and checks that it
// matches any provided properties. empty string or 0 id means do not check
// that property. It returns the ID of that latest activity.
//
// The difference with lastActivityMatches is that the asserted activity does
// not need to be the very last one, it will look for the last one of this
// specified type, which must be in one of the last 10 activities otherwise the
// test is failed.
func (ts *withServer) lastActivityOfTypeMatches(name, details string, id uint) uint {
	t := ts.s.T()

	var listActivities listActivitiesResponse
	ts.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
		&listActivities, "order_key", "a.id", "order_direction", "desc", "per_page", "10")
	require.True(t, len(listActivities.Activities) > 0)

	for _, act := range listActivities.Activities {
		if act.Type == name {
			if details != "" {
				require.NotNil(t, act.Details)
				assert.JSONEq(t, details, string(*act.Details))
			}
			if id > 0 {
				assert.Equal(t, id, act.ID)
			}
			return act.ID
		}
	}

	t.Fatalf("no activity of type %s found in the last %d activities", name, len(listActivities.Activities))
	return 0
}

func (ts *withServer) lastActivityOfTypeDoesNotMatch(name, details string, id uint) {
	t := ts.s.T()

	var listActivities listActivitiesResponse
	ts.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
		&listActivities, "order_key", "a.id", "order_direction", "desc", "per_page", "10")
	require.True(t, len(listActivities.Activities) > 0)

	for _, act := range listActivities.Activities {
		if act.Type == name {
			if details != "" {
				require.NotNil(t, act.Details)
				assert.NotEqual(t, details, string(*act.Details))
			}
			if id > 0 {
				assert.NotEqual(t, id, act.ID)
			}
		}
	}
}

func (ts *withServer) uploadSoftwareInstaller(
	t *testing.T,
	payload *fleet.UploadSoftwareInstallerPayload,
	expectedStatus int,
	expectedError string,
) {
	t.Helper()

	tfr, err := fleet.NewKeepFileReader(filepath.Join("testdata", "software-installers", payload.Filename))
	require.NoError(t, err)
	defer tfr.Close()

	payload.InstallerFile = tfr

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// add the software field
	fw, err := w.CreateFormFile("software", payload.Filename)
	require.NoError(t, err)
	n, err := io.Copy(fw, payload.InstallerFile)
	require.NoError(t, err)
	require.NotZero(t, n)

	// add the team_id field
	if payload.TeamID != nil {
		require.NoError(t, w.WriteField("team_id", fmt.Sprintf("%d", *payload.TeamID)))
	}
	// add the remaining fields
	require.NoError(t, w.WriteField("install_script", payload.InstallScript))
	require.NoError(t, w.WriteField("pre_install_query", payload.PreInstallQuery))
	require.NoError(t, w.WriteField("post_install_script", payload.PostInstallScript))
	require.NoError(t, w.WriteField("uninstall_script", payload.UninstallScript))
	if payload.SelfService {
		require.NoError(t, w.WriteField("self_service", "true"))
	}
	if payload.LabelsIncludeAny != nil {
		for _, l := range payload.LabelsIncludeAny {
			require.NoError(t, w.WriteField("labels_include_any", l))
		}
	}
	if payload.LabelsExcludeAny != nil {
		for _, l := range payload.LabelsExcludeAny {
			require.NoError(t, w.WriteField("labels_exclude_any", l))
		}
	}

	w.Close()

	headers := map[string]string{
		"Content-Type":  w.FormDataContentType(),
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", ts.token),
	}

	r := ts.DoRawWithHeaders("POST", "/api/latest/fleet/software/package", b.Bytes(), expectedStatus, headers)
	defer r.Body.Close()

	if expectedError != "" {
		errMsg := extractServerErrorText(r.Body)
		require.Contains(t, errMsg, expectedError)
	}
}
