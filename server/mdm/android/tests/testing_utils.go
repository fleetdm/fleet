package tests

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	ds_mock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/fleetdm/fleet/v4/server/service/middleware/log"
	"github.com/go-json-experiment/json"
	kithttp "github.com/go-kit/kit/transport/http"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
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

// resolve ambiguity between embedded datastore and mock methods
func (ds *AndroidDSWithMock) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	return ds.Store.AppConfig(ctx) // use mock datastore
}
func (ds *AndroidDSWithMock) CreateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *android.Device) (*android.Device, error) {
	return ds.Datastore.CreateDeviceTx(ctx, tx, device)
}

func (ds *AndroidDSWithMock) UpdateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *android.Device) error {
	return ds.Datastore.UpdateDeviceTx(ctx, tx, device)
}

func (ds *AndroidDSWithMock) CreateEnterprise(ctx context.Context, userID uint) (uint, error) {
	return ds.Datastore.CreateEnterprise(ctx, userID)
}

func (ds *AndroidDSWithMock) GetEnterpriseByID(ctx context.Context, id uint) (*android.EnterpriseDetails, error) {
	return ds.Datastore.GetEnterpriseByID(ctx, id)
}

func (ds *AndroidDSWithMock) GetEnterpriseBySignupToken(ctx context.Context, signupToken string) (*android.EnterpriseDetails, error) {
	return ds.Datastore.GetEnterpriseBySignupToken(ctx, signupToken)
}

func (ds *AndroidDSWithMock) GetEnterprise(ctx context.Context) (*android.Enterprise, error) {
	return ds.Datastore.GetEnterprise(ctx)
}

func (ds *AndroidDSWithMock) UpdateEnterprise(ctx context.Context, enterprise *android.EnterpriseDetails) error {
	return ds.Datastore.UpdateEnterprise(ctx, enterprise)
}

func (ds *AndroidDSWithMock) DeleteAllEnterprises(ctx context.Context) error {
	return ds.Datastore.DeleteAllEnterprises(ctx)
}

func (ds *AndroidDSWithMock) DeleteOtherEnterprises(ctx context.Context, id uint) error {
	return ds.Datastore.DeleteOtherEnterprises(ctx, id)
}

// Disambiguate method promoted from both mysql.Datastore and mock.Store
func (ds *AndroidDSWithMock) SetAndroidHostUnenrolled(ctx context.Context, hostID uint) error {
	return ds.Datastore.SetAndroidHostUnenrolled(ctx, hostID)
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
	svc, err := service.NewServiceWithClient(logger, &ts.DS, &ts.AndroidAPIClient, &ts.FleetSvc, "test-private-key")
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
		queryerContext sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
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
	ts.AndroidAPIClient.SignupURLsCreateFunc = func(_ context.Context, _, callbackURL string) (*android.SignupDetails, error) {
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
	ts.AndroidAPIClient.EnterprisesPoliciesPatchFunc = func(_ context.Context, policyName string, _ *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		assert.Contains(t, policyName, EnterpriseID)
		return &androidmanagement.Policy{}, nil
	}
	ts.AndroidAPIClient.EnterpriseDeleteFunc = func(_ context.Context, enterpriseName string) error {
		assert.Equal(t, "enterprises/"+EnterpriseID, enterpriseName)
		return nil
	}
}

func (ts *WithServer) TearDownSuite() {
	ts.DS.Datastore.Close()
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
	// use the standard Fleet datastore for Android integration tests
	return mysql.CreateMySQLDS(t)
}

func CreateEnrollmentMessage(t *testing.T, deviceInfo androidmanagement.Device) *android.PubSubMessage {
	return EnrollmentMessageWithEnterpriseSpecificID(t, deviceInfo, strings.ToUpper(uuid.New().String()))
}

func EnrollmentMessageWithEnterpriseSpecificID(t *testing.T, deviceInfo androidmanagement.Device, enterpriseSpecificID string) *android.PubSubMessage {
	deviceInfo.HardwareInfo = &androidmanagement.HardwareInfo{
		EnterpriseSpecificId: enterpriseSpecificID,
		Brand:                "TestBrand",
		Model:                "TestModel",
		SerialNumber:         "test-serial",
		Hardware:             "test-hardware",
	}
	deviceInfo.SoftwareInfo = &androidmanagement.SoftwareInfo{
		AndroidBuildNumber: "test-build",
		AndroidVersion:     "1",
	}
	deviceInfo.MemoryInfo = &androidmanagement.MemoryInfo{
		TotalRam:             int64(8 * 1024 * 1024 * 1024),  // 8GB RAM in bytes
		TotalInternalStorage: int64(64 * 1024 * 1024 * 1024), // 64GB system partition
	}

	deviceInfo.MemoryEvents = []*androidmanagement.MemoryEvent{
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(64 * 1024 * 1024 * 1024), // 64GB external/built-in storage total capacity
			CreateTime: "2024-01-15T09:00:00Z",
		},
		{
			EventType:  "INTERNAL_STORAGE_MEASURED",
			ByteCount:  int64(10 * 1024 * 1024 * 1024), // 10GB free in system partition
			CreateTime: "2024-01-15T10:00:00Z",
		},
		{
			EventType:  "EXTERNAL_STORAGE_MEASURED",
			ByteCount:  int64(25 * 1024 * 1024 * 1024), // 25GB free in external/built-in storage
			CreateTime: "2024-01-15T10:00:00Z",
		},
	}

	data, err := json.Marshal(deviceInfo)
	require.NoError(t, err)

	encodedData := base64.StdEncoding.EncodeToString(data)

	return &android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubEnrollment),
		},
		Data: encodedData,
	}
}

func CreateAndroidDeviceID(name string) string {
	return "enterprises/mock-enterprise-id/devices/" + name
}

func CreateStatusReportMessage(t *testing.T, deviceId, name, policyName string, policyVersion *int, nonComplianceDetails []*androidmanagement.NonComplianceDetail) android.PubSubMessage {
	device := androidmanagement.Device{
		Name:                 CreateAndroidDeviceID(name),
		NonComplianceDetails: nonComplianceDetails,
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: deviceId,
			Brand:                "TestBrand",
			Model:                "TestModel",
			SerialNumber:         "test-serial",
			Hardware:             "test-hardware",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidBuildNumber: "test-build",
			AndroidVersion:     "1",
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             int64(8 * 1024 * 1024 * 1024),  // 8GB RAM in bytes
			TotalInternalStorage: int64(64 * 1024 * 1024 * 1024), // 64GB system partition
		},
		AppliedPolicyName:    policyName,
		AppliedPolicyVersion: int64(*policyVersion),
		LastPolicySyncTime:   "2001-01-01T00:00:00Z",
		ApplicationReports: []*androidmanagement.ApplicationReport{{
			DisplayName: "Google Chrome",
			PackageName: "com.google.chrome",
			VersionName: "1.0.0",
		}},
	}

	data, err := json.Marshal(device)
	require.NoError(t, err)

	encodedData := base64.StdEncoding.EncodeToString(data)

	return android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubStatusReport),
		},
		Data: encodedData,
	}
}
