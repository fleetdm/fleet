package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	proxy_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service"
	ds_mock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/fleetdm/fleet/v4/server/service/middleware/log"
	kithttp "github.com/go-kit/kit/transport/http"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/api/androidmanagement/v1"
)

const (
	EnterpriseSignupURL = "https://enterprise.google.com/signup/android/email?origin=android&thirdPartyToken=B4D779F1C4DD9A440"
	EnterpriseID        = "LC02k5wxw7"
)

type WithServer struct {
	suite.Suite
	DS               *mysql.Datastore
	FleetDS          ds_mock.Store
	Server           *httptest.Server
	Token            string
	AppConfig        fleet.AppConfig
	Proxy            proxy_mock.Proxy
	ProxyCallbackURL string
}

func (ts *WithServer) SetupSuite(t *testing.T, dbName string) {
	ts.DS = CreateNamedMySQLDS(t, dbName)
	ts.CreateCommonDSMocks()

	ts.Proxy = proxy_mock.Proxy{}
	ts.createCommonProxyMocks(t)

	fleetSvc := mockService{}
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	svc, err := service.NewServiceWithProxy(logger, &ts.FleetDS, &ts.Proxy)
	require.NoError(t, err)

	ts.Server = runServerForTests(t, logger, &fleetSvc, svc)
}

func (ts *WithServer) CreateCommonDSMocks() {
	ts.FleetDS.GetAndroidDSFunc = func() android.Datastore {
		return ts.DS
	}
	ts.FleetDS.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &ts.AppConfig, nil
	}
	ts.FleetDS.SetAndroidEnabledAndConfiguredFunc = func(_ context.Context, configured bool) error {
		ts.AppConfig.MDM.AndroidEnabledAndConfigured = configured
		return nil
	}
}

func (ts *WithServer) createCommonProxyMocks(t *testing.T) {
	ts.Proxy.SignupURLsCreateFunc = func(callbackURL string) (*android.SignupDetails, error) {
		ts.ProxyCallbackURL = callbackURL
		return &android.SignupDetails{
			Url:  EnterpriseSignupURL,
			Name: "signupUrls/Cb08124d0999c464f",
		}, nil
	}
	ts.Proxy.EnterprisesCreateFunc = func(ctx context.Context, req android.ProxyEnterprisesCreateRequest) (string, string, error) {
		return EnterpriseID, "projects/android/topics/ae98ed130-5ce2-4ddb-a90a-191ec76976d5", nil
	}
	ts.Proxy.EnterprisesPoliciesPatchFunc = func(enterpriseID string, policyName string, policy *androidmanagement.Policy) error {
		assert.Equal(t, EnterpriseID, enterpriseID)
		return nil
	}
	ts.Proxy.EnterpriseDeleteFunc = func(enterpriseID string) error {
		assert.Equal(t, EnterpriseID, enterpriseID)
		return nil
	}
}

func (ts *WithServer) TearDownSuite() {
	mysql.Close(ts.DS)
}

type mockService struct {
	mock.Mock
	fleet.Service
}

func (m *mockService) GetSessionByKey(ctx context.Context, sessionKey string) (*fleet.Session, error) {
	return &fleet.Session{UserID: 1}, nil
}

func (m *mockService) UserUnauthorized(ctx context.Context, userId uint) (*fleet.User, error) {
	return &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}, nil
}

func runServerForTests(t *testing.T, logger kitlog.Logger, fleetSvc fleet.Service, androidSvc android.Service) *httptest.Server {

	fleetAPIOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(
			kithttp.PopulateRequestContext,
			auth.SetRequestsContexts(fleetSvc),
		),
		kithttp.ServerErrorHandler(&endpoint_utils.ErrorHandler{Logger: logger}),
		kithttp.ServerErrorEncoder(endpoint_utils.EncodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
			log.LogRequestEnd(logger),
		),
	}

	r := mux.NewRouter()
	service.GetRoutes(fleetSvc, androidSvc)(r, fleetAPIOptions)
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("/api/", r.ServeHTTP)

	server := httptest.NewUnstartedServer(rootMux)
	serverConfig := config.ServerConfig{}
	server.Config = serverConfig.DefaultHTTPServer(testCtx(), rootMux)
	require.NotZero(t, server.Config.WriteTimeout)
	server.Config.Handler = rootMux
	server.Start()
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

func testCtx() context.Context {
	return context.Background()
}

func CreateNamedMySQLDS(t *testing.T, name string) *mysql.Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}
	ds := mysql.InitializeDatabase(t, name, new(testing_utils.DatastoreTestOptions))
	t.Cleanup(func() { mysql.Close(ds) })
	return ds
}
