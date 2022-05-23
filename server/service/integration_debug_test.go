package service

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"testing"

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
	s.users = createTestUsers(s.T(), s.ds)
}

func (s *integrationDebugTestSuite) SetupTest() {
	_, server := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{SkipCreateTestUsers: true})
	s.server = server
	s.token = s.getTestAdminToken()
	s.Do("GET", "/debug/errors?flush=true", nil, http.StatusOK)
}

func TestIntegrationsDebug(t *testing.T) {
	testingSuite := new(integrationDebugTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationDebugTestSuite) TestBasic() {
	t := s.T()

	res := s.Do("GET", "/debug/errors?flush=true", nil, http.StatusOK)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.JSONEq(t, "[]", string(b))
}

func (s *integrationDebugTestSuite) TestUnwrappedErrors() {
	t := s.T()

	s.DoRawNoAuth("GET", "/api/v1/fleet/software?query=test", nil, http.StatusUnauthorized)

	res := s.Do("GET", "/debug/errors?flush=true", nil, http.StatusOK)
	rawErr, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var errs errorstore.JSONResponse
	require.NoError(t, json.Unmarshal(rawErr, &errs))
	require.Len(t, errs, 1)
	require.Equal(t, errs[0].Cause.Message, "Authorization header required")
	require.NotEmpty(t, errs[0].Wraps)
	require.NotEmpty(t, errs[0].Wraps[0].Stack)
}

func (s *integrationDebugTestSuite) Test404NoErrors() {
	t := s.T()

	s.Do("GET", "/api/v1/non/existent/path", nil, http.StatusNotFound)

	res := s.Do("GET", "/debug/errors?flush=true", nil, http.StatusOK)
	rawErr, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var errs errorstore.JSONResponse
	require.NoError(t, json.Unmarshal(rawErr, &errs))
	require.Len(t, errs, 0)
}

func (s *integrationDebugTestSuite) TestWrappedErrors() {
	t := s.T()

	s.Do("GET", "/api/latest/fleet/device/nonexistent", nil, http.StatusUnauthorized)

	res := s.Do("GET", "/debug/errors", nil, http.StatusOK)
	rawErr, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var errs errorstore.JSONResponse
	require.NoError(t, json.Unmarshal(rawErr, &errs))
	require.Len(t, errs, 1)
	require.Equal(t, errs[0].Cause.Message, "Authentication required")
	require.NotEmpty(t, errs[0].Wraps)
	require.NotEmpty(t, errs[0].Wraps[0].Stack)
	require.Regexp(t, regexp.MustCompile(`{"timestamp":".+","viewer":{"is_logged_in":true,"sso_enabled":false}}`), string(errs[0].Wraps[0].Data))
}
