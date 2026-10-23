package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/cron"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/mdm"
	fleetmock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/fleetdm/fleet/v4/server/service/redis_install_attempts"
	"github.com/fleetdm/fleet/v4/server/service/redis_lock"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// withDiskEncryptionDefaults fills in the explicit per-platform disk
// encryption booleans that every team spec apply now persists (settings
// absent from the spec are stored as explicit false).
func withDiskEncryptionDefaults(m fleet.TeamMDM) fleet.TeamMDM {
	if !m.MacOSSettings.EnableDiskEncryption.Valid {
		m.MacOSSettings.EnableDiskEncryption = optjson.SetBool(false)
	}
	if !m.MacOSSettings.EnableEscrowDiskEncryptionKey.Valid {
		m.MacOSSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(false)
	}
	if !m.WindowsSettings.EnableDiskEncryption.Valid {
		m.WindowsSettings.EnableDiskEncryption = optjson.SetBool(false)
	}
	if !m.LinuxSettings.EnableEscrowDiskEncryptionKey.Valid {
		m.LinuxSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(false)
	}
	return m
}

// withCreateDefaults adds the defaults only fleet creation persists on top of
// withDiskEncryptionDefaults: the canonical BitLocker PIN home is stored as
// explicit false for new fleets, while edits leave an absent value untouched.
func withCreateDefaults(m fleet.TeamMDM) fleet.TeamMDM {
	m = withDiskEncryptionDefaults(m)
	if !m.WindowsSettings.RequireBitLockerPIN.Valid {
		m.WindowsSettings.RequireBitLockerPIN = optjson.SetBool(false)
	}
	return m
}

func TestIntegrationsEnterprise(t *testing.T) {
	testingSuite := new(integrationEnterpriseTestSuite)
	testingSuite.withServer.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationEnterpriseTestSuite struct {
	withServer
	suite.Suite
	redisPool              fleet.RedisPool
	attemptCounter         fleet.SoftwareInstallAttemptCounter
	calendarSchedule       *schedule.Schedule
	softwareInstallStore   fleet.SoftwareInstallerStore
	softwareTitleIconStore fleet.SoftwareTitleIconStore

	lq *live_query_mock.MockLiveQuery
}

func (s *integrationEnterpriseTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationEnterpriseTestSuite")

	s.redisPool = redistest.SetupRedis(s.T(), s.T().Name(), false, false, false)
	s.attemptCounter = redis_install_attempts.NewTest(s.T(), s.redisPool)
	s.lq = live_query_mock.New(s.T())
	var calendarSchedule *schedule.Schedule

	// Create a software install store
	dir := s.T().TempDir()
	softwareInstallStore, err := filesystem.NewSoftwareInstallerStore(dir)
	require.NoError(s.T(), err)
	s.softwareInstallStore = softwareInstallStore

	// Create a software title icon store
	iconDir := s.T().TempDir()
	softwareTitleIconStore, err := filesystem.NewSoftwareTitleIconStore(iconDir)
	require.NoError(s.T(), err)
	s.softwareTitleIconStore = softwareTitleIconStore

	// Configure Windows WSTEP cert/key paths so the server initializes a
	// wstepCertManager and can validate Windows MSI EUA tokens sent via
	// /api/fleet/orbit/enroll. Required by TestOrbitEnrollWithEUAToken.
	fleetCfg := config.TestConfig()
	fleetCfg.MDM.WindowsWSTEPIdentityCert = "testdata/server.pem"
	fleetCfg.MDM.WindowsWSTEPIdentityKey = "testdata/server.key"

	config := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		Pool:                  s.redisPool,
		InstallAttemptCounter: s.attemptCounter,
		Rs:                    pubsub.NewInmemQueryResults(),
		Lq:                    s.lq,
		Logger:                slog.New(slog.NewTextHandler(os.Stdout, nil)),
		EnableCachedDS:        true,
		StartCronSchedules: []TestNewScheduleFunc{
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					// We set 24-hour interval so that it only runs when triggered.
					var err error
					cronLog := slog.New(slog.NewJSONHandler(os.Stdout, nil))
					if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
						cronLog = slog.New(slog.DiscardHandler)
					}
					calendarSchedule, err = cron.NewCalendarSchedule(
						ctx, s.T().Name(), s.ds, redis_lock.NewLock(s.redisPool), config.CalendarConfig{Periodicity: 24 * time.Hour},
						cronLog, &fleetmock.MockActivityService{},
					)
					return calendarSchedule, err
				}
			},
		},
		SoftwareInstallStore:            softwareInstallStore,
		SoftwareTitleIconStore:          softwareTitleIconStore,
		ConditionalAccessMicrosoftProxy: mockedConditionalAccessMicrosoftProxyInstance,
		DBConns:                         s.dbConns,
		FleetConfig:                     &fleetCfg,
	}
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		config.Logger = slog.New(slog.DiscardHandler)
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &config)
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.cachedTokens = make(map[string]string)
	s.calendarSchedule = calendarSchedule

	dev_mode.SetOverride("FLEET_DEV_BATCH_RETRY_INTERVAL", "1s")
}

func (s *integrationEnterpriseTestSuite) TearDownSuite() {
	dev_mode.ClearOverride("FLEET_DEV_BATCH_RETRY_INTERVAL")
}

func (s *integrationEnterpriseTestSuite) TearDownTest() {
	// reset the mock
	s.lq.Mock = mock.Mock{}
	s.withServer.commonTearDownTest(s.T())
}

// defaultExpectedWindowsSettings returns the WindowsSettings shape produced by a team config
// save/load round trip with nothing configured: an empty custom settings slice and the managed
// local account toggle force-defaulted to disabled.
func defaultExpectedWindowsSettings() fleet.WindowsSettings {
	return fleet.WindowsSettings{
		CustomSettings:            optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		EnableManagedLocalAccount: optjson.SetBool(false),
	}
}

func createHostAndDeviceToken(t *testing.T, ds *mysql.Datastore, token string) *fleet.Host {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		HardwareSerial:  uuid.New().String(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	createDeviceTokenForHost(t, ds, host.ID, token)

	return host
}

func updateDeviceTokenForHost(t *testing.T, ds *mysql.Datastore, hostID uint, token string) {
	mysqltest.ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE host_device_auth SET token = ? WHERE host_id = ?`, token, hostID)
		return err
	})
}

func createDeviceTokenForHost(t *testing.T, ds *mysql.Datastore, hostID uint, token string) {
	mysqltest.ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, hostID, token)
		return err
	})
}

// generates the body and headers part of a multipart request ready to be
// used via s.DoRawWithHeaders to POST /api/_version_/fleet/scripts.
func generateNewScriptMultipartRequest(t *testing.T,
	fileName string, fileContent []byte, token string, extraFields map[string][]string,
) (*bytes.Buffer, map[string]string) {
	return generateMultipartRequest(t, "script", fileName, fileContent, token, extraFields)
}

// checks that the specified team/no-team has the Windows OS Updates profile with
// the specified deadline/grace settings (or checks that it doesn't have the
// profile if wantSettings is nil). It returns the profile_uuid if it exists,
// empty string otherwise.
func checkWindowsOSUpdatesProfile(t *testing.T, ds *mysql.Datastore, teamID *uint, wantSettings *fleet.WindowsUpdates) string {
	ctx := context.Background()

	var prof fleet.MDMWindowsConfigProfile
	mysqltest.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		var globalOrTeamID uint
		if teamID != nil {
			globalOrTeamID = *teamID
		}
		err := sqlx.GetContext(ctx, tx, &prof, `SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`, globalOrTeamID, mdm.FleetWindowsOSUpdatesProfileName)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	if wantSettings == nil {
		require.Empty(t, prof.ProfileUUID)
	} else {
		require.NotEmpty(t, prof.ProfileUUID)
		require.Contains(t, string(prof.SyncML), fmt.Sprintf(`<Data>%d</Data>`, wantSettings.DeadlineDays.Value))
		require.Contains(t, string(prof.SyncML), fmt.Sprintf(`<Data>%d</Data>`, wantSettings.GracePeriodDays.Value))
	}

	if len(prof.ProfileUUID) > 0 {
		require.Equal(t, byte('w'), prof.ProfileUUID[0])
	}

	return prof.ProfileUUID
}

func (s *integrationEnterpriseTestSuite) createHosts(t *testing.T, platforms ...string) []*fleet.Host {
	var hosts []*fleet.Host
	if len(platforms) == 0 {
		platforms = []string{"debian", "rhel", "linux", "windows", "darwin"}
	}
	for i, platform := range platforms {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        platform,
		})
		require.NoError(t, err)
		hosts = append(hosts, host)
	}
	return hosts
}

func genDistributedReqWithPolicyResults(host *fleet.Host, policyResults map[uint]*bool) submitDistributedQueryResultsRequestShim {
	var (
		results  = make(map[string]json.RawMessage)
		statuses = make(map[string]any)
		messages = make(map[string]string)
	)
	for policyID, policyResult := range policyResults {
		distributedQueryName := hostPolicyQueryPrefix + fmt.Sprint(policyID)
		switch {
		case policyResult == nil:
			results[distributedQueryName] = json.RawMessage(`[]`)
			statuses[distributedQueryName] = 1
			messages[distributedQueryName] = "policy failed execution"
		case *policyResult:
			results[distributedQueryName] = json.RawMessage(`[{"1": "1"}]`)
			statuses[distributedQueryName] = 0
		case !*policyResult:
			results[distributedQueryName] = json.RawMessage(`[]`)
			statuses[distributedQueryName] = 0
		}
	}
	return submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: statuses,
		Messages: messages,
		Stats:    map[string]*fleet.Stats{},
	}
}

func checkSoftwareTitle(t *testing.T, ds *mysql.Datastore, title string, source string) uint {
	var id uint
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &id, `SELECT id FROM software_titles WHERE name = ? AND source = ? AND extension_for = ''`, title, source)
	})
	return id
}

func checkScriptContentsID(t *testing.T, ds *mysql.Datastore, id uint, expectedContents string) {
	var contents string
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &contents, `SELECT contents FROM script_contents WHERE id = ?`, id)
	})
	require.Equal(t, expectedContents, contents)
}

func checkSoftwareInstaller(t *testing.T, ds *mysql.Datastore, payload *fleet.UploadSoftwareInstallerPayload) (installerID uint, titleID uint) {
	var tid uint
	if payload.TeamID != nil {
		tid = *payload.TeamID
	}
	var id uint
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &id, `SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, tid, payload.Filename)
	})
	require.NotZero(t, id)

	var platform string
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &platform, `SELECT platform FROM software_installers WHERE id = ?`, id)
	})
	require.Equal(t, "linux", payload.Platform)

	meta, err := ds.GetSoftwareInstallerMetadataByID(context.Background(), id)
	require.NoError(t, err)

	if payload.TeamID != nil && *payload.TeamID > 0 {
		require.Equal(t, *payload.TeamID, *meta.TeamID)
	} else {
		require.Nil(t, meta.TeamID)
	}

	checkScriptContentsID(t, ds, meta.InstallScriptContentID, payload.InstallScript)

	if payload.PostInstallScript != "" {
		require.NotNil(t, meta.PostInstallScriptContentID)
		checkScriptContentsID(t, ds, *meta.PostInstallScriptContentID, payload.PostInstallScript)
	} else {
		require.Nil(t, meta.PostInstallScriptContentID)
	}

	require.Equal(t, payload.PreInstallQuery, meta.PreInstallQuery)
	require.Equal(t, payload.StorageID, meta.StorageID)
	require.Equal(t, payload.Filename, meta.Name)
	require.Equal(t, payload.Version, meta.Version)
	require.Equal(t, checkSoftwareTitle(t, ds, payload.Title, "deb_packages"), *meta.TitleID)
	require.NotZero(t, meta.UploadedAt)

	// get metadata by team and title ID so we can check labels
	meta2, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(context.Background(), payload.TeamID, *meta.TitleID, false)
	require.NoError(t, err)

	// check labels include any
	require.Len(t, meta2.LabelsIncludeAny, len(payload.LabelsIncludeAny))
	byName := make(map[string]struct{}, len(meta2.LabelsIncludeAny))
	for _, l := range meta2.LabelsIncludeAny {
		byName[l.LabelName] = struct{}{}
		require.Equal(t, *meta2.TitleID, l.TitleID)
		require.False(t, l.Exclude)
		require.False(t, l.RequireAll)
	}
	require.Len(t, byName, len(payload.LabelsIncludeAny))
	for _, l := range payload.LabelsIncludeAny {
		_, ok := byName[l]
		require.True(t, ok)
	}

	// check labels exclude any
	require.Len(t, meta2.LabelsExcludeAny, len(payload.LabelsExcludeAny))
	byName = make(map[string]struct{}, len(meta2.LabelsExcludeAny))
	for _, l := range meta2.LabelsExcludeAny {
		byName[l.LabelName] = struct{}{}
		require.Equal(t, *meta2.TitleID, l.TitleID)
		require.True(t, l.Exclude)
		require.False(t, l.RequireAll)
	}
	require.Len(t, byName, len(payload.LabelsExcludeAny))
	for _, l := range payload.LabelsExcludeAny {
		_, ok := byName[l]
		require.True(t, ok)
	}

	// check labels include all
	require.Len(t, meta2.LabelsIncludeAll, len(payload.LabelsIncludeAll))
	byName = make(map[string]struct{}, len(meta2.LabelsIncludeAll))
	for _, l := range meta2.LabelsIncludeAll {
		byName[l.LabelName] = struct{}{}
		require.Equal(t, *meta2.TitleID, l.TitleID)
		require.False(t, l.Exclude)
		require.True(t, l.RequireAll)
	}
	require.Len(t, byName, len(payload.LabelsIncludeAll))
	for _, l := range payload.LabelsIncludeAll {
		_, ok := byName[l]
		require.True(t, ok)
	}

	return meta.InstallerID, *meta.TitleID
}

func waitBatchSetSoftwareInstallers(t *testing.T, s *withServer, teamName string, requestUUID string) batchSetSoftwareInstallersResultResponse {
	timeout := time.After(1 * time.Minute)
	for {
		var batchResultResponse batchSetSoftwareInstallersResultResponse
		s.DoJSON("GET", "/api/latest/fleet/software/batch/"+requestUUID, nil, http.StatusOK, &batchResultResponse, "team_name", teamName)
		if batchResultResponse.Status == fleet.BatchSetSoftwareInstallersStatusCompleted ||
			batchResultResponse.Status == fleet.BatchSetSoftwareInstallersStatusFailed {
			return batchResultResponse
		}
		select {
		case <-timeout:
			t.Fatalf("timeout: %s, %s", teamName, requestUUID)
		case <-time.After(500 * time.Millisecond):
			// OK, continue
		}
	}
}

func waitBatchSetSoftwareInstallersCompleted(t *testing.T, s *withServer, teamName string, requestUUID string) []fleet.SoftwarePackageResponse {
	timeout := time.After(1 * time.Minute)
	for {
		var batchResultResponse batchSetSoftwareInstallersResultResponse
		s.DoJSON("GET", "/api/latest/fleet/software/batch/"+requestUUID, nil, http.StatusOK, &batchResultResponse, "fleet_name", teamName)
		if batchResultResponse.Status == fleet.BatchSetSoftwareInstallersStatusCompleted {
			return batchResultResponse.Packages
		}
		select {
		case <-timeout:
			t.Fatalf("timeout: %s, %s", teamName, requestUUID)
		case <-time.After(500 * time.Millisecond):
			// OK, continue
		}
	}
}

func waitBatchSetSoftwareInstallersFailed(t *testing.T, s *withServer, teamName string, requestUUID string) string {
	timeout := time.After(1 * time.Minute)
	for {
		var batchResultResponse batchSetSoftwareInstallersResultResponse
		s.DoJSON("GET", "/api/latest/fleet/software/batch/"+requestUUID, nil, http.StatusOK, &batchResultResponse, "fleet_name", teamName)
		if batchResultResponse.Status == fleet.BatchSetSoftwareInstallersStatusFailed {
			require.Empty(t, batchResultResponse.Packages)
			return batchResultResponse.Message
		}
		select {
		case <-timeout:
			t.Fatalf("timeout: %s, %s", teamName, requestUUID)
		case <-time.After(500 * time.Millisecond):
			// OK, continue
		}
	}
}

func getSoftwareTitleID(t *testing.T, ds *mysql.Datastore, title, source string) uint {
	var id uint
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &id, `SELECT id FROM software_titles WHERE name = ? AND source = ? AND extension_for = ''`, title, source)
	})
	return id
}

type mockedConditionalAccessMicrosoftProxy struct {
	createResponse *conditional_access_microsoft_proxy.CreateResponse
	getResponse    *conditional_access_microsoft_proxy.GetResponse
	deleteErr      error
	deleteResponse *conditional_access_microsoft_proxy.DeleteResponse

	setComplianceStatusFunc func(
		ctx context.Context,
		tenantID string, secret string,
		deviceID string,
		userPrincipalName string,
		mdmEnrolled bool,
		deviceName, osName, osVersion string,
		compliant bool,
		lastCheckInTime time.Time,
	) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error)

	getMessageStatusFunc func(
		ctx context.Context,
		tenantID string,
		secret string,
		messageID string,
	) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error)
}

func (m *mockedConditionalAccessMicrosoftProxy) Create(ctx context.Context, tenantID string) (*conditional_access_microsoft_proxy.CreateResponse, error) {
	return m.createResponse, nil
}

func (m *mockedConditionalAccessMicrosoftProxy) Get(ctx context.Context, tenantID string, secret string) (*conditional_access_microsoft_proxy.GetResponse, error) {
	return m.getResponse, nil
}

func (m *mockedConditionalAccessMicrosoftProxy) Delete(ctx context.Context, tenantID string, secret string) (*conditional_access_microsoft_proxy.DeleteResponse, error) {
	if m.deleteErr != nil {
		return nil, m.deleteErr
	}
	return m.deleteResponse, nil
}

func (m *mockedConditionalAccessMicrosoftProxy) SetComplianceStatus(
	ctx context.Context,
	tenantID string, secret string,
	deviceID string,
	userPrincipalName string,
	mdmEnrolled bool,
	deviceName, osName, osVersion string,
	compliant bool,
	lastCheckInTime time.Time,
) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error) {
	return m.setComplianceStatusFunc(ctx, tenantID, secret, deviceID, userPrincipalName, mdmEnrolled, deviceName, osName, osVersion, compliant, lastCheckInTime)
}

func (m *mockedConditionalAccessMicrosoftProxy) GetMessageStatus(
	ctx context.Context, tenantID string, secret string, messageID string,
) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error) {
	return m.getMessageStatusFunc(ctx, tenantID, secret, messageID)
}

var mockedConditionalAccessMicrosoftProxyInstance = &mockedConditionalAccessMicrosoftProxy{}

func genDistributedReqWithLabelResults(host *fleet.Host, labelResults map[uint]*bool) submitDistributedQueryResultsRequestShim {
	var (
		results  = make(map[string]json.RawMessage)
		statuses = make(map[string]any)
		messages = make(map[string]string)
	)
	for labelID, labelResult := range labelResults {
		distributedQueryName := hostLabelQueryPrefix + fmt.Sprint(labelID)
		switch {
		case labelResult == nil:
			results[distributedQueryName] = json.RawMessage(`[]`)
			statuses[distributedQueryName] = 1
			messages[distributedQueryName] = "label failed execution"
		case *labelResult:
			results[distributedQueryName] = json.RawMessage(`[{"1": "1"}]`)
			statuses[distributedQueryName] = 0
		case !*labelResult:
			results[distributedQueryName] = json.RawMessage(`[]`)
			statuses[distributedQueryName] = 0
		}
	}
	return submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: statuses,
		Messages: messages,
		Stats:    map[string]*fleet.Stats{},
	}
}
