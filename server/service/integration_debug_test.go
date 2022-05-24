package service

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type integrationDebugTestSuite struct {
	suite.Suite
	withServer
}

func (s *integrationDebugTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationDebugTestSuite")

	redisPool := redistest.SetupRedis(s.T(), "debugtest:", false, false, false)
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{Pool: redisPool})
	s.server = server
	s.token = s.getTestAdminToken()
	s.users = users
}

func TestIntegrationsDebug(t *testing.T) {
	testingSuite := new(integrationDebugTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationDebugTestSuite) TestDebugErrorsIntegration() {
	var errs errorstore.JSONResponse
	t := s.T()

	// ensure we start on a clean state
	res := s.Do("GET", "/debug/errors?flush=true", nil, http.StatusOK)

	// no errors stored if nothing happened
	res = s.Do("GET", "/debug/errors?flush=true", nil, http.StatusOK)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.JSONEq(t, "[]", string(b))

	// unwrapped errors are still captured
	s.DoRawNoAuth("GET", "/api/v1/fleet/software?query=test", nil, http.StatusUnauthorized)
	res = s.Do("GET", "/debug/errors?flush=true", nil, http.StatusOK)
	rawErr, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawErr, &errs))
	require.Len(t, errs, 1)
	require.Equal(t, errs[0].Cause.Message, "Authorization header required")
	require.NotEmpty(t, errs[0].Wraps)
	require.NotEmpty(t, errs[0].Wraps[0].Stack)

	// 404s are not captured
	s.Do("GET", "/api/v1/non/existent/path", nil, http.StatusNotFound)
	res = s.Do("GET", "/debug/errors?flush=true", nil, http.StatusOK)
	rawErr, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawErr, &errs))
	require.Len(t, errs, 0)

	// wrapped errors are stored
	s.Do("GET", "/api/latest/fleet/device/nonexistent", nil, http.StatusUnauthorized)
	res = s.Do("GET", "/debug/errors", nil, http.StatusOK)
	rawErr, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawErr, &errs))
	require.Len(t, errs, 1)
	require.Equal(t, errs[0].Cause.Message, "Authentication required")
	require.NotEmpty(t, errs[0].Wraps)
	require.NotEmpty(t, errs[0].Wraps[0].Stack)
	require.Regexp(t, regexp.MustCompile(`{"timestamp":".+","viewer":{"is_logged_in":true,"sso_enabled":false}}`), string(errs[0].Wraps[0].Data))
}
