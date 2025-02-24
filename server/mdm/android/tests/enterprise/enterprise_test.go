package enterprise_test

import (
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/tests"
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
	var resp android.GetEnterpriseResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusNotFound, &resp)

	// TODO: Create enterprise
}
