package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationLoggerTestSuite(t *testing.T) {
	testingSuite := new(integrationLoggerTestSuite)
	testingSuite.s = &testingSuite.Suite
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

	users, server := RunServerForTestsWithDS(s.T(), s.ds, TestServerOpts{Logger: logger})
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
	payload := createQueryRequest{}
	s.DoJSON("POST", "/api/v1/fleet/queries", params, http.StatusOK, &payload)

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
			assert.Equal(t, "/api/v1/fleet/login", kv["uri"])
		case 1:
			assert.Equal(t, "debug", kv["level"])
			assert.Equal(t, "GET", kv["method"])
			assert.Equal(t, "/api/v1/fleet/config", kv["uri"])
			assert.Equal(t, "admin1@example.com", kv["user"])
		case 2:
			assert.Equal(t, "debug", kv["level"])
			assert.Equal(t, "POST", kv["method"])
			assert.Equal(t, "/api/v1/fleet/queries", kv["uri"])
			assert.Equal(t, "admin1@example.com", kv["user"])
			assert.Equal(t, "somequery", kv["name"])
			assert.Equal(t, "select 1 from osquery;", kv["sql"])
		default:
			t.Fail()
		}
	}
}

func (s *integrationLoggerTestSuite) TestOsqueryEndpointsLogErrors() {
	t := s.T()

	_, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "1234",
		UUID:            "1",
		Hostname:        "foo.local",
		OsqueryHostID:   t.Name(),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	requestBody := io.NopCloser(bytes.NewBuffer([]byte(`{"node_key":"1234","log_type":"status","data":[}`)))
	req, _ := http.NewRequest("POST", s.server.URL+"/api/v1/osquery/log", requestBody)
	client := fleethttp.NewClient()
	_, err = client.Do(req)
	require.Nil(t, err)

	logString := s.buf.String()
	assert.Contains(t, logString, `{"err":"decoding JSON:`)
	assert.Contains(t, logString, `invalid character '}' looking for beginning of value","level":"info","path":"/api/v1/osquery/log"}
`, logString)
}

func (s *integrationLoggerTestSuite) TestSubmitStatusLog() {
	t := s.T()

	_, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "1234",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   t.Name(),
	})
	require.NoError(t, err)

	req := submitLogsRequest{
		NodeKey: "1234",
		LogType: "status",
		Data:    nil,
	}
	res := submitLogsResponse{}
	s.DoJSON("POST", "/api/v1/osquery/log", req, http.StatusOK, &res)

	logString := s.buf.String()
	assert.Equal(t, 1, strings.Count(logString, "\"ip_addr\""))
	assert.Equal(t, 1, strings.Count(logString, "x_for_ip_addr"))
}

func (s *integrationLoggerTestSuite) TestEnrollAgentLogsErrors() {
	t := s.T()
	_, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1234",
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

	s.DoRawNoAuth("POST", "/api/v1/osquery/enroll", j, http.StatusUnauthorized)

	parts := strings.Split(strings.TrimSpace(s.buf.String()), "\n")
	require.Len(t, parts, 1)
	logData := make(map[string]json.RawMessage)
	require.NoError(t, json.Unmarshal([]byte(parts[0]), &logData))
	assert.Equal(t, `"error"`, string(logData["level"]))
	assert.Contains(t, string(logData["err"]), `"enroll failed:`)
	assert.Contains(t, string(logData["err"]), `no matching secret found`)
}
