package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type integrationSMTPTestSuite struct {
	suite.Suite
	withServer
}

func (s *integrationSMTPTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationSMTPTestSuite")

	users, server := RunServerForTestsWithDS(
		s.T(),
		s.ds,
		&TestServerOpts{
			UseMailService: true,
		})
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
}

func TestIntegrationsSMTP(t *testing.T) {
	testingSuite := new(integrationSMTPTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationSMTPTestSuite) TestSMTPValidation() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"smtp_settings": {
			"enable_smtp": true,
			"sender_address": "sender@email.com",
			"server": "http://localhost:62000"
		}
	}`), http.StatusUnprocessableEntity, &acResp)
	require.NotNil(t, acResp)
}
