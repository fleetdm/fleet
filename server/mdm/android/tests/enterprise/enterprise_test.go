package enterprise_test

import (
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestServiceEnterprise(t *testing.T) {
	testingSuite := new(enterpriseTestSuite)
	suite.Run(t, testingSuite)
}

type enterpriseTestSuite struct {
	tests.WithServer
}

func (s *enterpriseTestSuite) SetupSuite() {
	s.WithServer.SetupSuite(s.T(), "androidEnterpriseTestSuite")
	s.Token = "bozo"
}

func (s *enterpriseTestSuite) TearDownSuite() {
	s.WithServer.TearDownSuite()
}

func (s *enterpriseTestSuite) TestGetEnterprise() {
	// Enterprise doesn't exist.
	var resp android.GetEnterpriseResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusNotFound, &resp)

	// Create enterprise
	var signupResp android.EnterpriseSignupResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupResp)
	assert.Equal(s.T(), tests.EnterpriseSignupURL, signupResp.Url)
	s.T().Logf("callbackURL: %s", s.ProxyCallbackURL)
	const enterpriseToken = "enterpriseToken"
	s.DoJSON("GET", s.ProxyCallbackURL, nil, http.StatusOK, &resp, "enterpriseToken", enterpriseToken)

	// Now enterprise exists and we can retrieve it.
	resp = android.GetEnterpriseResponse{}
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(s.T(), tests.EnterpriseID, resp.EnterpriseID)
}
