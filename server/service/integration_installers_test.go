package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const enrollSecret = "xyz"

type integrationInstallersTestSuite struct {
	suite.Suite
	withServer
	installers []fleet.Installer
}

func (s *integrationInstallersTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationInstallersTestSuite")
	t := s.T()

	is := s3.SetupTestInstallerStore(t, "integration-tests", "")
	users, server := RunServerForTestsWithDS(t, s.ds, &TestServerOpts{Is: is})
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.installers = s3.SeedTestInstallerStore(t, is, enrollSecret)

	err := s.ds.ApplyEnrollSecrets(context.TODO(), nil, []*fleet.EnrollSecret{{Secret: enrollSecret}})
	require.NoError(t, err)
}

func TestIntegrationsInstallers(t *testing.T) {
	testingSuite := new(integrationInstallersTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationInstallersTestSuite) TestInstallerGet() {
	t := s.T()

	// make sure FLEET_DEMO is not set
	os.Unsetenv("FLEET_DEMO")
	validURL := installerURL(enrollSecret, "pkg", false)
	s.Do("GET", validURL, nil, http.StatusInternalServerError)

	os.Setenv("FLEET_DEMO", "1")
	defer os.Unsetenv("FLEET_DEMO")

	// works when FLEET_DEMO is set
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
	url := fmt.Sprintf("/api/latest/fleet/download_installer/%s/%s", secret, kind)
	if desktop {
		url = url + "?desktop=1"
	}
	return url
}
