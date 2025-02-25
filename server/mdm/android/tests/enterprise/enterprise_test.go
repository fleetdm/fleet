package enterprise_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service"
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
	service.SignupSSEInterval = 10 * time.Millisecond
}

func (s *enterpriseTestSuite) SetupTest() {
	s.AppConfig.MDM.AndroidEnabledAndConfigured = false
	s.CreateCommonDSMocks()
}

func (s *enterpriseTestSuite) TearDownSuite() {
	s.WithServer.TearDownSuite()
}

func (s *enterpriseTestSuite) TestGetEnterprise() {
	s.SetupTest()

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

	// Delete enterprise and make sure we can't find it.
	s.Do("DELETE", "/api/v1/fleet/android_enterprise", nil, http.StatusOK)
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusNotFound, &resp)
}

func (s *enterpriseTestSuite) TestEnterpriseSSE() {
	s.SetupTest()

	// Test happy path
	resp := s.Do("GET", "/api/v1/fleet/android_enterprise/signup_sse", nil, http.StatusOK)
	sseDone := make(chan struct{})
	buf := make([]byte, 1024)
	go func() {
		n, _ := resp.Body.Read(buf)
		assert.Equal(s.T(), service.SignupSSESuccess, string(buf[:n]))
		close(sseDone)
	}()

	time.Sleep(50 * time.Millisecond)
	s.AppConfig.MDM.AndroidEnabledAndConfigured = true

	select {
	case <-sseDone:
		s.T().Log("SSE done")
	case <-time.After(2 * time.Second):
		s.T().Fatal("Timed out waiting for SSE")
	}

	// Test with Android already enabled
	resp = s.Do("GET", "/api/v1/fleet/android_enterprise/signup_sse", nil, http.StatusOK)
	n, _ := resp.Body.Read(buf)
	assert.Equal(s.T(), service.SignupSSESuccess, string(buf[:n]))

	// Test with error
	s.WithServer.FleetDS.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return nil, assert.AnError
	}
	resp = s.Do("GET", "/api/v1/fleet/android_enterprise/signup_sse", nil, http.StatusOK)
	n, _ = resp.Body.Read(buf)
	assert.Contains(s.T(), string(buf[:n]), assert.AnError.Error())
}
