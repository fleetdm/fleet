package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
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
	opts := &TestServerOpts{FleetConfig: &cfg, Is: is}
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		opts.Logger = kitlog.NewNopLogger()
	}
	users, server := RunServerForTestsWithDS(t, s.ds, opts)
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

	validURL, formBody := installerPOSTReq(enrollSecret, "pkg", s.token, false)

	r := s.DoRaw("POST", validURL, formBody, http.StatusOK)
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	require.Equal(t, "mock", string(body))
	require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
	require.Equal(t, "4", r.Header.Get("Content-Length"))
	require.Equal(t, `attachment;filename="fleet-osquery.pkg"`, r.Header.Get("Content-Disposition"))
	require.Equal(t, `nosniff`, r.Header.Get("X-Content-Type-Options"))

	// unauthorized requests
	s.DoRawNoAuth("POST", validURL, nil, http.StatusUnauthorized)
	s.token = "invalid"
	s.Do("POST", validURL, nil, http.StatusUnauthorized)
	s.token = s.cachedAdminToken

	// wrong enroll secret
	wrongURL, wrongFormBody := installerPOSTReq("wrong-enroll", "pkg", s.token, false)
	s.Do("POST", wrongURL, wrongFormBody, http.StatusNotFound)

	// non-existent package
	wrongURL, wrongFormBody = installerPOSTReq(enrollSecret, "exe", s.token, false)
	s.Do("POST", wrongURL, wrongFormBody, http.StatusNotFound)
}

func (s *integrationSandboxTestSuite) TestInstallerHeadCheck() {
	validURL := installerURL(enrollSecret, "pkg", false)
	s.DoRaw("HEAD", validURL, nil, http.StatusOK)

	// unauthorized requests
	s.DoRawNoAuth("HEAD", validURL, nil, http.StatusUnauthorized)
	s.token = "invalid"
	s.DoRaw("HEAD", validURL, nil, http.StatusUnauthorized)
	s.token = s.cachedAdminToken

	// wrong enroll secret
	invalidURL := installerURL("wrong-enroll", "pkg", false)
	s.DoRaw("HEAD", invalidURL, nil, http.StatusNotFound)

	// non-existent package
	invalidURL = installerURL(enrollSecret, "exe", false)
	s.DoRaw("HEAD", invalidURL, nil, http.StatusNotFound)
}

func installerURL(secret, kind string, desktop bool) string {
	path := fmt.Sprintf("/api/latest/fleet/download_installer/%s?enroll_secret=%s", kind, secret)
	if desktop {
		path += "&desktop=1"
	}
	return path
}

func installerPOSTReq(secret, kind, token string, desktop bool) (string, []byte) {
	path := installerURL(secret, kind, desktop)
	d := "0"
	if desktop {
		d = "1"
	}
	formBody := make(url.Values)
	formBody.Set("token", token)
	formBody.Set("enroll_secret", secret)
	formBody.Set("desktop", d)
	return path, []byte(formBody.Encode())
}
