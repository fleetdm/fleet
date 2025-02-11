package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
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

	buf *bytes.Buffer
}

func (s *integrationLoggerTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationLoggerTestSuite")

	s.buf = new(bytes.Buffer)
	logger := log.NewJSONLogger(s.buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	users, server := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{Logger: logger})
	s.server = server
	s.users = users
}

func (s *integrationLoggerTestSuite) TearDownTest() {
	s.buf.Reset()
}

func (s *integrationLoggerTestSuite) TestLogger() {
	t := s.T()

	s.token = getTestAdminToken(t, s.server)

	s.getConfig()

	params := fleet.QueryPayload{
		Name:        ptr.String("somequery"),
		Description: ptr.String("desc"),
		Query:       ptr.String("select 1 from osquery;"),
	}
	var createResp createQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", params, http.StatusOK, &createResp)

	logs := s.buf.String()
	parts := strings.Split(strings.TrimSpace(logs), "\n")
	assert.Len(t, parts, 3)
	for i, part := range parts {
		kv := make(map[string]string)
		err := json.Unmarshal([]byte(part), &kv)
		require.NoError(t, err)

		assert.NotEqual(t, "", kv["took"])

		switch i {
		case 0:
			assert.Equal(t, "info", kv["level"])
			assert.Equal(t, "POST", kv["method"])
			assert.Equal(t, "/api/latest/fleet/login", kv["uri"])
		case 1:
			assert.Equal(t, "debug", kv["level"])
			assert.Equal(t, "GET", kv["method"])
			assert.Equal(t, "/api/latest/fleet/config", kv["uri"])
			assert.Equal(t, "admin1@example.com", kv["user"])
		case 2:
			assert.Equal(t, "debug", kv["level"])
			assert.Equal(t, "POST", kv["method"])
			assert.Equal(t, "/api/latest/fleet/queries", kv["uri"])
			assert.Equal(t, "admin1@example.com", kv["user"])
			assert.Equal(t, "somequery", kv["name"])
			assert.Equal(t, "select 1 from osquery;", kv["sql"])
		default:
			t.Fail()
		}
	}
}

func (s *integrationLoggerTestSuite) TestLoggerLogin() {
	t := s.T()

	type logEntry struct {
		key string
		val string
	}

	testCases := []struct {
		loginRequest   loginRequest
		expectedStatus int
		expectedLogs   []logEntry
	}{
		{
			loginRequest:   loginRequest{Email: testUsers["admin1"].Email, Password: testUsers["admin1"].PlaintextPassword},
			expectedStatus: http.StatusOK,
			expectedLogs:   []logEntry{{"email", testUsers["admin1"].Email}},
		},
		{
			loginRequest:   loginRequest{Email: testUsers["admin1"].Email, Password: "n074v411dp455w02d"},
			expectedStatus: http.StatusUnauthorized,
			expectedLogs: []logEntry{
				{"email", testUsers["admin1"].Email},
				{"level", "error"},
				{"internal", "invalid password"},
			},
		},
		{
			loginRequest:   loginRequest{Email: "h4x0r@3x4mp13.c0m", Password: "n074v411dp455w02d"},
			expectedStatus: http.StatusUnauthorized,
			expectedLogs: []logEntry{
				{"email", "h4x0r@3x4mp13.c0m"},
				{"level", "error"},
				{"internal", "user not found"},
			},
		},
	}
	var resp loginResponse
	for _, tt := range testCases {
		s.DoJSON("POST", "/api/latest/fleet/login", tt.loginRequest, tt.expectedStatus, &resp)
		logString := s.buf.String()
		parts := strings.Split(strings.TrimSpace(logString), "\n")
		require.Len(t, parts, 1)
		logData := make(map[string]string)
		require.NoError(t, json.Unmarshal([]byte(parts[0]), &logData))

		require.NotContains(t, logData, "user") // logger context is set to skip user

		for _, e := range tt.expectedLogs {
			assert.Equal(t, e.val, logData[e.key], fmt.Sprintf("%+v", tt.expectedLogs))
		}
		s.buf.Reset()
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

	logString := s.buf.String()
	assert.Contains(t, logString, `invalid character '}' looking for beginning of value","level":"info","path":"/api/osquery/log","uuid":"`+jsn.UUID+`"}`, logString)
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

	// submit status logs
	req := submitLogsRequest{
		NodeKey: *h.NodeKey,
		LogType: "status",
		Data:    nil,
	}
	res := submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", req, http.StatusOK, &res)

	logString := s.buf.String()
	assert.Equal(t, 1, strings.Count(logString, `"ip_addr"`))
	assert.Equal(t, 1, strings.Count(logString, "x_for_ip_addr"))
	s.buf.Reset()

	// submit results logs
	req = submitLogsRequest{
		NodeKey: *h.NodeKey,
		LogType: "result",
		Data:    nil,
	}
	res = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", req, http.StatusOK, &res)

	logString = s.buf.String()
	assert.Equal(t, 1, strings.Count(logString, `"ip_addr"`))
	assert.Equal(t, 1, strings.Count(logString, "x_for_ip_addr"))
	s.buf.Reset()

	// submit invalid type logs
	req = submitLogsRequest{
		NodeKey: *h.NodeKey,
		LogType: "unknown",
		Data:    nil,
	}
	var errRes map[string]string
	s.DoJSON("POST", "/api/osquery/log", req, http.StatusInternalServerError, &errRes)
	assert.Contains(t, errRes["error"], "unknown log type")
	s.buf.Reset()

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
	logString = s.buf.String()
	assert.Equal(t, 1, strings.Count(logString, `"ip_addr"`))
	assert.Equal(t, 1, strings.Count(logString, "x_for_ip_addr"))

	// submit same payload without specifying gzip encoding fails
	s.DoRawWithHeaders("POST", "/api/osquery/log", body.Bytes(), http.StatusBadRequest, nil)
}

func (s *integrationLoggerTestSuite) TestEnrollAgentLogsErrors() {
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

	j, err := json.Marshal(&enrollAgentRequest{
		EnrollSecret:   "1234",
		HostIdentifier: "4321",
		HostDetails:    nil,
	})
	require.NoError(t, err)

	s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusUnauthorized)

	parts := strings.Split(strings.TrimSpace(s.buf.String()), "\n")
	require.Len(t, parts, 1)
	logData := make(map[string]json.RawMessage)
	require.NoError(t, json.Unmarshal([]byte(parts[0]), &logData))
	assert.Equal(t, `"error"`, string(logData["level"]))
	assert.Contains(t, string(logData["err"]), `"enroll failed:`)
	assert.Contains(t, string(logData["err"]), `no matching secret found`)
}
