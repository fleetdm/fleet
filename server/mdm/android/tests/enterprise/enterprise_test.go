package enterprise_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/android/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	s.Svc.(*service.Service).SignupSSEInterval = 10 * time.Millisecond
}

func (s *enterpriseTestSuite) SetupTest() {
	s.AppConfig.MDM.AndroidEnabledAndConfigured = false
	s.CreateCommonDSMocks()
}

func (s *enterpriseTestSuite) TearDownSuite() {
	s.WithServer.TearDownSuite()
}

func (s *enterpriseTestSuite) TestEnterprise() {
	s.SetupTest()

	// Enterprise doesn't exist.
	var resp android.GetEnterpriseResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusNotFound, &resp)

	// Create enterprise
	var signupResp android.EnterpriseSignupResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupResp)
	assert.Equal(s.T(), tests.EnterpriseSignupURL, signupResp.Url)
	s.T().Logf("callbackURL: %s", s.ProxyCallbackURL)

	s.FleetSvc.On("NewActivity", mock.Anything, mock.Anything, mock.AnythingOfType("fleet.ActivityTypeEnabledAndroidMDM")).Return(nil)
	const enterpriseToken = "enterpriseToken"
	res := s.Do("GET", s.ProxyCallbackURL, nil, http.StatusOK, "enterpriseToken", enterpriseToken)
	s.FleetSvc.AssertNumberOfCalls(s.T(), "NewActivity", 1)
	body, err := io.ReadAll(res.Body)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "text/html; charset=UTF-8", res.Header.Get("Content-Type"))
	assert.Contains(s.T(), string(body), "If this page does not close automatically, please close it manually.")
	assert.Contains(s.T(), string(body), "window.close()")

	// Now enterprise exists and we can retrieve it.
	resp = android.GetEnterpriseResponse{}
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(s.T(), tests.EnterpriseID, resp.EnterpriseID)

	// Delete enterprise and make sure we can't find it.
	s.FleetSvc.On("NewActivity", mock.Anything, mock.Anything, mock.AnythingOfType("fleet.ActivityTypeDisabledAndroidMDM")).Return(nil)
	s.Do("DELETE", "/api/v1/fleet/android_enterprise", nil, http.StatusOK)
	s.FleetSvc.AssertNumberOfCalls(s.T(), "NewActivity", 2)
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusNotFound, &resp)
}

func (s *enterpriseTestSuite) TestEnterpriseSSE() {
	s.SetupTest()

	// Test happy path
	resp := s.Do("GET", "/api/v1/fleet/android_enterprise/signup_sse", nil, http.StatusOK)
	sseDone := make(chan struct{})
	go func() {
		data, err := io.ReadAll(resp.Body)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), service.SignupSSESuccess, string(data))
		close(sseDone)
	}()

	time.Sleep(50 * time.Millisecond)
	s.AppConfigMu.Lock()
	s.AppConfig.MDM.AndroidEnabledAndConfigured = true
	s.AppConfigMu.Unlock()

	select {
	case <-sseDone:
		s.T().Log("SSE done")
	case <-time.After(2 * time.Second):
		s.T().Fatal("Timed out waiting for SSE")
	}

	// Test with Android already enabled
	resp = s.Do("GET", "/api/v1/fleet/android_enterprise/signup_sse", nil, http.StatusOK)
	data, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), service.SignupSSESuccess, string(data))

	// Test with error
	s.WithServer.DS.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return nil, assert.AnError
	}
	resp = s.Do("GET", "/api/v1/fleet/android_enterprise/signup_sse", nil, http.StatusOK)
	data, err = io.ReadAll(resp.Body)
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), string(data), assert.AnError.Error())
}
