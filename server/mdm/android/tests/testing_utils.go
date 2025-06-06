package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	ds_mock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/fleetdm/fleet/v4/server/service/middleware/log"
	kithttp "github.com/go-kit/kit/transport/http"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
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

type AndroidDSWithMock struct {
	*mysql.Datastore
	ds_mock.Store
}

type WithServer struct {
	suite.Suite
	Svc      android.Service
	DS       AndroidDSWithMock
	FleetSvc mockService
	Server   *httptest.Server
	Token    string

	AppConfig   fleet.AppConfig
	AppConfigMu sync.Mutex

	AndroidAPIClient android_mock.Client
	ProxyCallbackURL string
}

func (ts *WithServer) SetupSuite(t *testing.T, dbName string) {
	ts.DS.Datastore = CreateNamedMySQLDS(t, dbName)
	ts.CreateCommonDSMocks()

	ts.AndroidAPIClient = android_mock.Client{}
	ts.createCommonProxyMocks(t)

	logger := kitlog.NewLogfmtLogger(os.Stdout)
	svc, err := service.NewServiceWithProxy(logger, &ts.DS, &ts.AndroidAPIClient, &ts.FleetSvc)
	require.NoError(t, err)
	ts.Svc = svc

	ts.Server = runServerForTests(t, logger, &ts.FleetSvc, svc)
}

func (ts *WithServer) CreateCommonDSMocks() {
	ts.DS.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		// Create a copy to prevent race conditions
		ts.AppConfigMu.Lock()
		appConfigCopy := ts.AppConfig
		ts.AppConfigMu.Unlock()
		return &appConfigCopy, nil
	}
	ts.DS.SetAndroidEnabledAndConfiguredFunc = func(_ context.Context, configured bool) error {
		ts.AppConfigMu.Lock()
		ts.AppConfig.MDM.AndroidEnabledAndConfigured = configured
		ts.AppConfigMu.Unlock()
		return nil
	}
	ts.DS.UserOrDeletedUserByIDFunc = func(_ context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{ID: id}, nil
	}
	ts.DS.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		result := make(map[fleet.MDMAssetName]fleet.MDMConfigAsset, len(assetNames))
		for _, name := range assetNames {
			result[name] = fleet.MDMConfigAsset{Value: []byte("value")}
		}
		return result, nil
	}
	ts.DS.InsertOrReplaceMDMConfigAssetFunc = func(ctx context.Context, asset fleet.MDMConfigAsset) error {
		return nil
	}
	ts.DS.DeleteMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName) error {
		return nil
	}
	ts.DS.BulkSetAndroidHostsUnenrolledFunc = func(ctx context.Context) error {
		return nil
	}
}

func (ts *WithServer) createCommonProxyMocks(t *testing.T) {
	ts.AndroidAPIClient.InitCommonMocks()
	ts.AndroidAPIClient.SignupURLsCreateFunc = func(_, callbackURL string) (*android.SignupDetails, error) {
		ts.ProxyCallbackURL = callbackURL
		return &android.SignupDetails{
			Url:  EnterpriseSignupURL,
			Name: "signupUrls/Cb08124d0999c464f",
		}, nil
	}
	ts.AndroidAPIClient.EnterprisesCreateFunc = func(_ context.Context, _ androidmgmt.EnterprisesCreateRequest) (androidmgmt.EnterprisesCreateResponse, error) {
		return androidmgmt.EnterprisesCreateResponse{
			EnterpriseName: "enterprises/" + EnterpriseID,
			TopicName:      "projects/android/topics/ae98ed130-5ce2-4ddb-a90a-191ec76976d5",
		}, nil
	}
	ts.AndroidAPIClient.EnterprisesPoliciesPatchFunc = func(policyName string, _ *androidmanagement.Policy) error {
		assert.Contains(t, policyName, EnterpriseID)
		return nil
	}
	ts.AndroidAPIClient.EnterpriseDeleteFunc = func(_ context.Context, enterpriseName string) error {
		assert.Equal(t, "enterprises/"+EnterpriseID, enterpriseName)
		return nil
	}
}

func (ts *WithServer) TearDownSuite() {
	mysql.Close(ts.DS.Datastore)
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

func (m *mockService) NewActivity(ctx context.Context, user *fleet.User, details fleet.ActivityDetails) error {
	return m.Called(ctx, user, details).Error(0)
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
