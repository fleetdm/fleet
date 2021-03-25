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
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/test"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/store/memstore"
)

type testResource struct {
	server     *httptest.Server
	adminToken string
	userToken  string
	ds         kolide.Datastore
}

type endpointService struct {
	kolide.Service
}

func (svc endpointService) SendTestEmail(ctx context.Context, config *kolide.AppConfig) error {
	return nil
}
func setupEndpointTest(t *testing.T) *testResource {
	test := &testResource{}

	var err error
	test.ds, err = inmem.New(config.TestConfig())
	require.Nil(t, err)
	require.Nil(t, test.ds.MigrateData())

	devOrgInfo := &kolide.AppConfig{
		OrgName:                "Kolide",
		OrgLogoURL:             "http://foo.bar/image.png",
		SMTPPort:               465,
		SMTPAuthenticationType: kolide.AuthTypeUserNamePassword,
		SMTPEnableTLS:          true,
		SMTPVerifySSLCerts:     true,
		SMTPEnableStartTLS:     true,
	}
	test.ds.NewAppConfig(devOrgInfo)
	svc, _ := newTestService(test.ds, nil, nil)
	svc = endpointService{svc}
	createTestUsers(t, test.ds)
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	jwtKey := "CHANGEME"
	limitStore, _ := memstore.New(0)

	routes := MakeHandler(svc, config.KolideConfig{Auth: config.AuthConfig{JwtKey: jwtKey}}, logger, limitStore)

	test.server = httptest.NewServer(routes)

	userParam := loginRequest{
		Username: "admin1",
		Password: testUsers["admin1"].PlaintextPassword,
	}

	marshalledUser, _ := json.Marshal(&userParam)

	requestBody := &nopCloser{bytes.NewBuffer(marshalledUser)}
	resp, _ := http.Post(test.server.URL+"/api/v1/fleet/login", "application/json", requestBody)

	var jsn = struct {
		User  *kolide.User `json:"user"`
		Token string       `json:"token"`
		Err   string       `json:"error,omitempty"`
	}{}
	json.NewDecoder(resp.Body).Decode(&jsn)
	test.adminToken = jsn.Token

	// log in non admin user
	userParam.Username = "user1"
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

var testFunctions = [...]func(*testing.T, *testResource){
	testGetAppConfig,
	testModifyAppConfig,
	testModifyAppConfigWithValidationFail,
	testAdminUserSetAdmin,
	testNonAdminUserSetAdmin,
	testAdminUserSetEnabled,
	testNonAdminUserSetEnabled,
}

func TestEndpoints(t *testing.T) {
	for _, f := range testFunctions {
		r := setupEndpointTest(t)
		defer r.server.Close()
		t.Run(test.FunctionName(f), func(t *testing.T) {
			f(t, r)
		})
	}
}
