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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type WithDS struct {
	suite.Suite
	DS *mysql.Datastore
}

func (ts *WithDS) SetupSuite(t *testing.T, dbName string) {
	ts.DS = CreateNamedMySQLDS(t, dbName)
}

func CreateNamedMySQLDS(t *testing.T, name string) *mysql.Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}
	ds := mysql.InitializeDatabase(t, name, new(testing_utils.DatastoreTestOptions))
	t.Cleanup(func() { mysql.Close(ds) })
	return ds
}

func (ts *WithDS) TearDownSuite() {
	mysql.Close(ts.DS)
}

type WithServer struct {
	WithDS
	Server *httptest.Server
	Token  string
}

func (ts *WithServer) SetupSuite(t *testing.T, dbName string) {
	ts.WithDS.SetupSuite(t, dbName)
	ts.Server = runServerForTestsWithFleetDS(t, ts.DS)
}

func (ts *WithServer) TearDownSuite() {
	ts.WithDS.TearDownSuite()
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

func runServerForTestsWithFleetDS(t *testing.T, ds android.Datastore) *httptest.Server {
	fleetDS := ds_mock.Store{}
	fleetDS.GetAndroidDSFunc = func() android.Datastore {
		return ds
	}
	fleetSvc := mockService{}
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	svc, err := service.NewService(testCtx(), logger, &fleetDS)
	require.NoError(t, err)

	fleetAPIOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(
			kithttp.PopulateRequestContext,
			auth.SetRequestsContexts(&fleetSvc),
		),
		kithttp.ServerErrorHandler(&endpoint_utils.ErrorHandler{Logger: logger}),
		kithttp.ServerErrorEncoder(endpoint_utils.EncodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
			log.LogRequestEnd(logger),
		),
	}

	r := mux.NewRouter()
	service.GetRoutes(&fleetSvc, svc)(r, fleetAPIOptions)
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
