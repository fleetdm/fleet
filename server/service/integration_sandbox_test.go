package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const enrollSecret = "xyz/abc$@"

type integrationSandboxTestSuite struct {
	suite.Suite
	withServer
	installers []fleet.Installer
}

func (s *integrationSandboxTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationSandboxTestSuite")
	t := s.T()

	// make sure sandbox is enabled
	cfg := config.TestConfig()
	cfg.Server.SandboxEnabled = true

	is := s3.SetupTestInstallerStore(t, "integration-tests", "")
	users, server := RunServerForTestsWithDS(t, s.ds, &TestServerOpts{FleetConfig: &cfg, Is: is})
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.installers = s3.SeedTestInstallerStore(t, is, enrollSecret)

	err := s.ds.ApplyEnrollSecrets(context.TODO(), nil, []*fleet.EnrollSecret{{Secret: enrollSecret}})
	require.NoError(t, err)
}

func TestIntegrationsSandbox(t *testing.T) {
	testingSuite := new(integrationSandboxTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationSandboxTestSuite) TestDemoLogin() {
	t := s.T()

	validEmail := testUsers["user1"].Email
	validPwd := testUsers["user1"].PlaintextPassword
	wrongPwd := "nope"
	hdrs := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}

	formBody := make(url.Values)
	formBody.Set("email", validEmail)
	formBody.Set("password", wrongPwd)
	res := s.DoRawWithHeaders("POST", "/api/v1/fleet/demologin", []byte(formBody.Encode()), http.StatusUnauthorized, hdrs)
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)

	formBody.Set("email", validEmail)
	formBody.Set("password", validPwd)
	res = s.DoRawWithHeaders("POST", "/api/v1/fleet/demologin", []byte(formBody.Encode()), http.StatusOK, hdrs)
	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Contains(t, string(resBody), `window.location = "/"`)
	require.Regexp(t, `window.localStorage.setItem\('FLEET::auth_token', '[^']+'\)`, string(resBody))
}

func (s *integrationSandboxTestSuite) TestInstallerGet() {
	t := s.T()

	validURL := installerURL(enrollSecret, "pkg", false)

	r := s.Do("GET", validURL, nil, http.StatusOK)
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	require.Equal(t, "mock", string(body))
	require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
	require.Equal(t, "4", r.Header.Get("Content-Length"))
	require.Equal(t, "attachment", r.Header.Get("Content-Disposition"))

	// unauthorized requests
	s.DoRawNoAuth("GET", validURL, nil, http.StatusUnauthorized)
	s.token = "invalid"
	s.Do("GET", validURL, nil, http.StatusUnauthorized)
	s.token = s.cachedAdminToken

	// wrong enroll secret
	s.Do("GET", installerURL("wrong-enroll", "pkg", false), nil, http.StatusInternalServerError)

	// non-existent package
	s.Do("GET", installerURL(enrollSecret, "exe", false), nil, http.StatusNotFound)
}

func installerURL(secret, kind string, desktop bool) string {
	url := fmt.Sprintf("/api/latest/fleet/download_installer/%s?enroll_secret=%s", kind, secret)
	if desktop {
		url = url + "&desktop=1"
	}
	return url
}
