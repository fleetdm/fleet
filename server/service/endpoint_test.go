package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/datastore/inmem"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/require"
)

type testResource struct {
	server    *httptest.Server
	userToken string
	ds        kolide.Datastore
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
	svc, _ := newTestService(test.ds, nil)
	createTestUsers(t, test.ds)
	logger := kitlog.NewLogfmtLogger(os.Stdout)

	jwtKey := "CHANGEME"
	opts := []kithttp.ServerOption{
		kithttp.ServerBefore(setRequestsContexts(svc, jwtKey)),
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerAfter(kithttp.SetContentType("application/json; charset=utf-8")),
	}

	router := mux.NewRouter()
	ke := MakeKolideServerEndpoints(svc, jwtKey)
	ctxt := context.Background()
	kh := makeKolideKitHandlers(ctxt, ke, opts)
	attachKolideAPIRoutes(router, kh)

	test.server = httptest.NewServer(router)

	userParam := loginRequest{
		Username: "admin1",
		Password: testUsers["admin1"].PlaintextPassword,
	}

	marshalledUser, _ := json.Marshal(&userParam)

	requestBody := &nopCloser{bytes.NewBuffer(marshalledUser)}
	resp, _ := http.Post(test.server.URL+"/api/v1/kolide/login", "application/json", requestBody)

	var jsn = struct {
		User  *kolide.User `json:"user"`
		Token string       `json:"token"`
		Err   string       `json:"error,omitempty"`
	}{}
	json.NewDecoder(resp.Body).Decode(&jsn)
	test.userToken = jsn.Token
	return test
}

func functionName(f func(*testing.T, *testResource)) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	elements := strings.Split(fullName, ".")
	return elements[len(elements)-1]
}

var testFunctions = [...]func(*testing.T, *testResource){
	testGetAppConfig,
	testModifyAppConfig,
	testModifyAppConfigWithValidationFail,
	testGetOptions,
	testModifyOptions,
	testModifyOptionsValidationFail,
	testImportConfig,
	testImportConfigMissingExternal,
	testImportConfigWithMissingGlob,
	testImportConfigWithGlob,
}

func TestEndpoints(t *testing.T) {
	for _, f := range testFunctions {
		r := setupEndpointTest(t)
		defer r.server.Close()
		t.Run(functionName(f), func(t *testing.T) {
			f(t, r)
		})
	}
}
