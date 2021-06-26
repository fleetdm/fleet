package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/datastore/inmem"
	"github.com/fleetdm/fleet/server/fleet"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

type testResource struct {
	server     *httptest.Server
	adminToken string
	userToken  string
	ds         fleet.Datastore
}

type endpointService struct {
	fleet.Service
}

func (svc endpointService) sendTestEmail(ctx context.Context, config *fleet.AppConfig) error {
	return nil
}
func setupEndpointTest(t *testing.T) *testResource {
	test := &testResource{}

	var err error
	test.ds, err = inmem.New(config.TestConfig())
	require.Nil(t, err)
	require.Nil(t, test.ds.MigrateData())

	devOrgInfo := &fleet.AppConfig{
		OrgName:                "Example",
		OrgLogoURL:             "http://foo.bar/image.png",
		SMTPPort:               465,
		SMTPAuthenticationType: fleet.AuthTypeUserNamePassword,
		SMTPEnableTLS:          true,
		SMTPVerifySSLCerts:     true,
		SMTPEnableStartTLS:     true,
	}
	test.ds.NewAppConfig(devOrgInfo)
	svc := newTestService(test.ds, nil, nil)
	svc = endpointService{svc}
	createTestUsers(t, test.ds)
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	limitStore, _ := memstore.New(0)

	routes := MakeHandler(svc, config.FleetConfig{}, logger, limitStore)

	test.server = httptest.NewServer(routes)

	userParam := loginRequest{
		Email:    "admin1",
		Password: testUsers["admin1"].PlaintextPassword,
	}

	marshalledUser, _ := json.Marshal(&userParam)

	requestBody := &nopCloser{bytes.NewBuffer(marshalledUser)}
	resp, _ := http.Post(test.server.URL+"/api/v1/fleet/login", "application/json", requestBody)

	var jsn = struct {
		User  *fleet.User `json:"user"`
		Token string      `json:"token"`
		Err   string      `json:"error,omitempty"`
	}{}
	json.NewDecoder(resp.Body).Decode(&jsn)
	test.adminToken = jsn.Token

	// log in non admin user
	userParam.Email = "user1"
	userParam.Password = testUsers["user1"].PlaintextPassword
	marshalledUser, _ = json.Marshal(userParam)
	requestBody = &nopCloser{bytes.NewBuffer(marshalledUser)}
	resp, err = http.Post(test.server.URL+"/api/v1/fleet/login", "application/json", requestBody)
	require.Nil(t, err)
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.Nil(t, err)
	test.userToken = jsn.Token

	return test
}
