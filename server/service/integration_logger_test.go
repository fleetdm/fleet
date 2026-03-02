package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsLoggerTestSuite(t *testing.T) {
	testingSuite := new(integrationLoggerTestSuite)
	testingSuite.withDS.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationLoggerTestSuite struct {
	withServer
	suite.Suite

	handler *testutils.TestHandler
}

func (s *integrationLoggerTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationLoggerTestSuite")

	s.handler = testutils.NewTestHandler()
	logger := slog.New(s.handler)
	redisPool := redistest.SetupRedis(s.T(), "zz", false, false, false)

	users, server := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		Logger: logger,
		Pool:   redisPool,
	})
	s.server = server
	s.users = users
}

func (s *integrationLoggerTestSuite) TearDownTest() {
	s.handler.Clear()
}

func (s *integrationLoggerTestSuite) TestLogger() {
	t := s.T()

	s.token = getTestAdminToken(t, s.server)

	s.getConfig()

	params := map[string]any{
		"name":        "somequery",
		"description": "desc",
		"query":       "select 1 from osquery;",
		"fleet_id":    nil,
	}
	var createResp createQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", params, http.StatusOK, &createResp)

	records := s.handler.Records()
	require.Len(t, records, 3)
	for i, rec := range records {
		attrs := testutils.RecordAttrs(&rec)

		assert.Contains(t, attrs, "took")

		switch i {
		case 0:
			assert.Equal(t, slog.LevelInfo, rec.Level)
			assert.Equal(t, "POST", attrs["method"])
			assert.Equal(t, "/api/latest/fleet/login", attrs["uri"])
		case 1:
			assert.Equal(t, slog.LevelDebug, rec.Level)
			assert.Equal(t, "GET", attrs["method"])
			assert.Equal(t, "/api/latest/fleet/config", attrs["uri"])
			assert.Equal(t, "admin1@example.com", attrs["user"])
		case 2:
			assert.Equal(t, slog.LevelWarn, rec.Level) // Warn because /queries is a deprecated path
			assert.Equal(t, "POST", attrs["method"])
			assert.Equal(t, "/api/latest/fleet/queries", attrs["uri"])
			assert.Equal(t, "admin1@example.com", attrs["user"])
			assert.Equal(t, "somequery", attrs["name"])
			assert.Equal(t, "select 1 from osquery;", attrs["sql"])
			assert.Equal(t, "/api/_version_/fleet/queries", attrs["deprecated_path"])
			assert.Contains(t, attrs["deprecation_warning"], "deprecated")
		default:
			t.Fail()
		}
	}
}

func (s *integrationLoggerTestSuite) TestLoggerLogin() {
	t := s.T()

	type expectedAttr struct {
		key string
		val any
	}

	testCases := []struct {
		loginRequest   contract.LoginRequest
		expectedStatus int
		expectedLevel  slog.Level
		expectedAttrs  []expectedAttr
	}{
		{
			loginRequest:   contract.LoginRequest{Email: testUsers["admin1"].Email, Password: testUsers["admin1"].PlaintextPassword},
			expectedStatus: http.StatusOK,
			expectedLevel:  slog.LevelInfo,
			expectedAttrs:  []expectedAttr{{"email", testUsers["admin1"].Email}},
		},
		{
			loginRequest:   contract.LoginRequest{Email: testUsers["admin1"].Email, Password: "n074v411dp455w02d"},
			expectedStatus: http.StatusUnauthorized,
			expectedLevel:  slog.LevelInfo,
			expectedAttrs: []expectedAttr{
				{"email", testUsers["admin1"].Email},
				{"internal", "invalid password"},
			},
		},
		{
			loginRequest:   contract.LoginRequest{Email: "h4x0r@3x4mp13.c0m", Password: "n074v411dp455w02d"},
			expectedStatus: http.StatusUnauthorized,
			expectedLevel:  slog.LevelInfo,
			expectedAttrs: []expectedAttr{
				{"email", "h4x0r@3x4mp13.c0m"},
				{"internal", "user not found"},
			},
		},
	}
	var resp loginResponse
	for _, tt := range testCases {
		s.DoJSON("POST", "/api/latest/fleet/login", tt.loginRequest, tt.expectedStatus, &resp)

		records := s.handler.Records()
		require.Len(t, records, 1)
		assert.Equal(t, tt.expectedLevel, records[0].Level)

		attrs := testutils.RecordAttrs(&records[0])
		require.NotContains(t, attrs, "user") // logger context is set to skip user

		for _, e := range tt.expectedAttrs {
			assert.Equal(t, e.val, attrs[e.key], fmt.Sprintf("%+v", tt.expectedAttrs))
		}
		s.handler.Clear()
	}
}

func (s *integrationLoggerTestSuite) TestOsqueryEndpointsLogErrors() {
	t := s.T()

	_, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "1234"),
		UUID:            "1",
		Hostname:        "foo.local",
		OsqueryHostID:   ptr.String(t.Name()),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	requestBody := io.NopCloser(bytes.NewBuffer([]byte(`{"node_key":"1234","log_type":"status","data":[}`)))
	req, _ := http.NewRequest("POST", s.server.URL+"/api/osquery/log", requestBody)
	client := fleethttp.NewClient()
	resp, err := client.Do(req)
	require.NoError(t, err)
	jsn := struct {
		Message string              `json:"message"`
		Errs    []map[string]string `json:"errors,omitempty"`
		UUID    string              `json:"uuid"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.NoError(t, err)
	assert.Equal(t, "Bad request", jsn.Message)
	assert.Len(t, jsn.Errs, 1)
	assert.Equal(t, "base", jsn.Errs[0]["name"])
	assert.Equal(t, "json decoder error", jsn.Errs[0]["reason"])
	require.NotEmpty(t, jsn.UUID)

	records := s.handler.Records()
	require.NotEmpty(t, records)
	var foundErrRecord bool
	for i := range records {
		attrs := testutils.RecordAttrs(&records[i])
		if attrs["uuid"] == jsn.UUID {
			foundErrRecord = true
			assert.Equal(t, slog.LevelInfo, records[i].Level)
			assert.Equal(t, "/api/osquery/log", attrs["path"])
			assert.Contains(t, fmt.Sprint(attrs["internal"]), `invalid character '}' looking for beginning of value`)
			assert.Contains(t, attrs, "took")
			break
		}
	}
	require.True(t, foundErrRecord, "expected a log record with uuid %s", jsn.UUID)
}

func (s *integrationLoggerTestSuite) TestSubmitLog() {
	t := s.T()

	h, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "1234"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   ptr.String(t.Name()),
	})
	require.NoError(t, err)

	assertIPAddrLogged := func(records []slog.Record) {
		t.Helper()
		var ipAddrCount, xForIPAddrCount int
		for i := range records {
			attrs := testutils.RecordAttrs(&records[i])
			if _, ok := attrs["ip_addr"]; ok {
				ipAddrCount++
			}
			if _, ok := attrs["x_for_ip_addr"]; ok {
				xForIPAddrCount++
			}
		}
		assert.Equal(t, 1, ipAddrCount)
		assert.Equal(t, 1, xForIPAddrCount)
	}

	// submit status logs
	req := submitLogsRequest{
		NodeKey: *h.NodeKey,
		LogType: "status",
		Data:    nil,
	}
	res := submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", req, http.StatusOK, &res)

	assertIPAddrLogged(s.handler.Records())
	s.handler.Clear()

	// submit results logs
	req = submitLogsRequest{
		NodeKey: *h.NodeKey,
		LogType: "result",
		Data:    nil,
	}
	res = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", req, http.StatusOK, &res)

	assertIPAddrLogged(s.handler.Records())
	s.handler.Clear()

	// submit invalid type logs
	req = submitLogsRequest{
		NodeKey: *h.NodeKey,
		LogType: "unknown",
		Data:    nil,
	}
	var errRes map[string]string
	s.DoJSON("POST", "/api/osquery/log", req, http.StatusInternalServerError, &errRes)
	assert.Contains(t, errRes["error"], "unknown log type")
	s.handler.Clear()

	// submit gzip-encoded request
	var body bytes.Buffer
	gw := gzip.NewWriter(&body)
	_, err = fmt.Fprintf(gw, `{
		"node_key": %q,
		"log_type": "status",
		"data":     null
	}`, *h.NodeKey)
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	s.DoRawWithHeaders("POST", "/api/osquery/log", body.Bytes(), http.StatusOK, map[string]string{"Content-Encoding": "gzip"})
	assertIPAddrLogged(s.handler.Records())

	// submit same payload without specifying gzip encoding fails
	s.DoRawWithHeaders("POST", "/api/osquery/log", body.Bytes(), http.StatusBadRequest, nil)
}

func (s *integrationLoggerTestSuite) TestEnrollOsqueryLogsErrors() {
	t := s.T()
	_, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1234"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	j, err := json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   "1234",
		HostIdentifier: "4321",
		HostDetails:    nil,
	})
	require.NoError(t, err)

	s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusUnauthorized)

	records := s.handler.Records()
	require.Len(t, records, 1)
	assert.Equal(t, slog.LevelInfo, records[0].Level)
	attrs := testutils.RecordAttrs(&records[0])
	errStr := fmt.Sprint(attrs["err"])
	assert.Contains(t, errStr, "enroll failed:")
	assert.Contains(t, errStr, "no matching secret found")
}

func (s *integrationLoggerTestSuite) TestSetupExperienceEULAMetadataDoesNotLogErrorIfNotFound() {
	t := s.T()

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	originalAppConf := *appConf

	t.Cleanup(func() {
		// restore app config
		err = s.ds.SaveAppConfig(context.Background(), &originalAppConf)
		require.NoError(t, err)
	})

	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.WindowsEnabledAndConfigured = true
	appConf.MDM.AppleBMEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)

	s.token = getTestAdminToken(t, s.server)
	s.Do("GET", "/api/v1/fleet/setup_experience/eula/metadata", nil, http.StatusNotFound)

	records := s.handler.Records()
	require.Len(t, records, 2) // Login and not found

	assert.Equal(t, slog.LevelInfo, records[1].Level)
	attrs := testutils.RecordAttrs(&records[1])
	assert.Equal(t, "not found", fmt.Sprint(attrs["err"]))
}

func (s *integrationLoggerTestSuite) TestWindowsMDMEnrollEmptyBinarySecurityToken() {
	t := s.T()
	ctx := t.Context()

	appConf, err := s.ds.AppConfig(ctx)
	require.NoError(s.T(), err)
	originalAppConf := *appConf

	t.Cleanup(func() {
		// restore app config
		err = s.ds.SaveAppConfig(context.Background(), &originalAppConf)
		require.NoError(t, err)
	})

	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.WindowsEnabledAndConfigured = true
	appConf.MDM.AppleBMEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)

	host := createOrbitEnrolledHost(t, "windows", "", s.ds)
	mdmDevice := mdmtest.NewTestMDMClientWindowsEmptyBinarySecurityToken(s.server.URL, *host.OrbitNodeKey)
	err = mdmDevice.Enroll()
	require.Error(t, err)

	records := s.handler.Records()

	var foundDiscovery, foundPolicy, foundEnroll bool
	for i := range records {
		attrs := testutils.RecordAttrs(&records[i])
		uri, _ := attrs["uri"].(string)

		switch uri {
		case microsoft_mdm.MDE2DiscoveryPath:
			foundDiscovery = true
		case microsoft_mdm.MDE2PolicyPath:
			foundPolicy = true
			require.Equal(t, slog.LevelInfo, records[i].Level)
			require.Equal(t, "binarySecurityToken is empty", attrs["soap_fault"])
		case microsoft_mdm.MDE2EnrollPath:
			foundEnroll = true
		}
	}
	require.True(t, foundDiscovery)
	require.True(t, foundPolicy)
	// Will not enroll due to soap fault on prior request
	require.False(t, foundEnroll)
}
