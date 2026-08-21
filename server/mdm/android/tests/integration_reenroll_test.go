package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/api/androidmanagement/v1"
)

func TestServiceReenroll(t *testing.T) {
	suite.Run(t, new(reenrollTestSuite))
}

type reenrollTestSuite struct {
	WithServer
}

func (s *reenrollTestSuite) SetupSuite() {
	s.WithServer.SetupSuite(s.T(), "androidReenrollTestSuite")
	s.Token = "testtoken"
}

// TestClearsStateOnReenrollment drives a full enroll -> re-enroll cycle through the
// Pub/Sub ENROLLMENT path and checks what a re-enrolled device keeps and loses: the
// state it no longer has is cleared, while manually assigned labels, install history
// and past activities survive.
func (s *reenrollTestSuite) TestClearsStateOnReenrollment() {
	ctx := context.Background()
	t := s.T()

	// Create the enterprise.
	var signupResp android.EnterpriseSignupResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupResp)
	s.FleetSvc.On("NewActivity", mock.Anything, mock.Anything, mock.AnythingOfType("fleet.ActivityTypeEnabledAndroidMDM")).Return(nil)
	s.Do("GET", s.ProxyCallbackURL, nil, http.StatusOK, "enterpriseToken", "enterpriseToken")
	s.AndroidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		return []*androidmanagement.Enterprise{{Name: "enterprises/" + EnterpriseID}}, nil
	}
	resp := android.GetEnterpriseResponse{}
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(t, EnterpriseID, resp.EnterpriseID)

	// App config is mocked in this harness, so set it directly. Past activities are
	// preserved so we can check that the re-enroll leaves the host's history alone.
	s.AppConfigMu.Lock()
	s.AppConfig.MDM.AndroidEnabledAndConfigured = true
	s.AppConfig.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment = true
	s.AppConfigMu.Unlock()

	require.NoError(t, s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: "enrollsecret"}}))
	assets, err := s.DS.Datastore.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := string(assets[fleet.MDMAssetAndroidPubSubToken].Value)
	require.NotEmpty(t, pubsubToken)

	enterpriseSpecificID := strings.ToUpper(uuid.New().String())
	enroll := func() {
		msg := enrollmentMessageWithEnterpriseSpecificID(t, androidmanagement.Device{
			Name:                createAndroidDeviceID("test-reenroll"),
			EnrollmentTokenData: `{"EnrollSecret": "enrollsecret"}`,
		}, enterpriseSpecificID)
		req := service.PubSubPushRequest{PubSubMessage: *msg}
		s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", pubsubToken)
	}

	// First enrollment: the host is new to Fleet.
	enroll()

	host, err := s.DS.Datastore.AndroidHostLite(ctx, enterpriseSpecificID)
	require.NoError(t, err)
	hostID, hostUUID := host.Host.ID, host.Host.UUID

	labelNames := func() []string {
		labels, err := s.DS.Datastore.ListLabelsForHost(ctx, hostID)
		require.NoError(t, err)
		names := make([]string, 0, len(labels))
		for _, label := range labels {
			names = append(names, label.Name)
		}
		return names
	}
	countFor := func(query string, args ...any) int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, query, args...)
		})
		return count
	}

	builtinLabels := labelNames()

	// Seed the state a re-enrolled device leaves behind.

	// A manual label (must survive) and a dynamic one (must not).
	manualLabel, err := s.DS.Datastore.NewLabel(ctx, &fleet.Label{
		Name: "manual-label", LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)
	dynamicLabel, err := s.DS.Datastore.NewLabel(ctx, &fleet.Label{
		Name: "dynamic-label", Query: "SELECT 1", LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	mysqltest.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO label_membership (host_id, label_id) VALUES (?, ?), (?, ?)`,
			hostID, manualLabel.ID, hostID, dynamicLabel.ID)
		return err
	})

	// A pending lock command, which also leaves a host_mdm_actions.lock_ref behind.
	lockCmd := &android.MDMAndroidCommand{
		CommandUUID:   uuid.NewString(),
		HostUUID:      hostUUID,
		OperationName: "operations/lock-" + enterpriseSpecificID,
		CommandType:   string(android.MDMAndroidCommandTypeLock),
		Status:        string(android.MDMAndroidCommandStatusPending),
	}
	require.NoError(t, s.DS.Datastore.LockHostViaAndroidMDM(ctx, host.Host, lockCmd))

	// A pending software install.
	vppApp, err := s.DS.Datastore.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "com.app.vpp1", Platform: fleet.AndroidPlatform}},
	}, nil)
	require.NoError(t, err)
	installCmdUUID := uuid.NewString()
	require.NoError(t, s.DS.Datastore.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
		HostID: hostID, AdamID: vppApp.AdamID, CommandUUID: installCmdUUID, AssociatedEventID: "1",
	}))

	// A past activity.
	mysqltest.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx,
			`INSERT INTO activity_past (user_name, activity_type, details) VALUES ('admin', 'ran_script', '{}')`)
		if err != nil {
			return err
		}
		activityID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		_, err = q.ExecContext(ctx, `INSERT INTO activity_host_past (host_id, activity_id) VALUES (?, ?)`, hostID, activityID)
		return err
	})

	// Sanity check the seed: the vitals from the first enrollment are there too.
	require.Equal(t, 1, countFor(`SELECT COUNT(*) FROM host_disks WHERE host_id = ?`, hostID))
	require.Equal(t, 1, countFor(`SELECT COUNT(*) FROM host_operating_system WHERE host_id = ?`, hostID))
	require.Equal(t, 1, countFor(`SELECT COUNT(*) FROM mdm_android_commands WHERE host_uuid = ? AND status = 'pending'`, hostUUID))
	require.Equal(t, 1, countFor(`SELECT COUNT(*) FROM host_mdm_actions WHERE host_id = ?`, hostID))

	// Re-enroll the same device.
	enroll()

	// Machine-derived state the device no longer has is gone.
	require.ElementsMatch(t, append(append([]string{}, builtinLabels...), "manual-label"), labelNames(),
		"a re-enroll must drop dynamic label membership but keep manual and builtin memberships")
	require.Subset(t, labelNames(), builtinLabels,
		"the builtin memberships must survive: nothing else re-adds them for Android")
	require.Zero(t, countFor(`SELECT COUNT(*) FROM mdm_android_commands WHERE host_uuid = ? AND status = 'pending'`, hostUUID),
		"pending commands from the previous enrollment must be cancelled")
	require.Equal(t, 1, countFor(
		`SELECT COUNT(*) FROM host_vpp_software_installs
		 WHERE command_uuid = ? AND verification_failed_at IS NOT NULL AND canceled = 0`, installCmdUUID),
		"the pending install must be marked failed, and its record left visible")

	// No host_mdm_actions ref may outlive the command it points at. This is the
	// invariant the reset has to hold, and it is stronger than the lock/wipe status
	// assertions below (those pass on their own thanks to ClearHostMDMActions).
	require.Zero(t, countFor(`
		SELECT COUNT(*) FROM host_mdm_actions hma
		WHERE hma.host_id = ?
		  AND (
			(hma.lock_ref IS NOT NULL AND NOT EXISTS (SELECT 1 FROM mdm_android_commands c WHERE c.command_uuid = hma.lock_ref)) OR
			(hma.wipe_ref IS NOT NULL AND NOT EXISTS (SELECT 1 FROM mdm_android_commands c WHERE c.command_uuid = hma.wipe_ref)) OR
			(hma.clear_passcode_ref IS NOT NULL AND NOT EXISTS (SELECT 1 FROM mdm_android_commands c WHERE c.command_uuid = hma.clear_passcode_ref))
		  )`, hostID),
		"a re-enroll must not leave a lock/wipe/clear-passcode ref pointing at a deleted command")

	// The host is not left looking locked or wiped.
	require.Zero(t, countFor(`SELECT COUNT(*) FROM host_mdm_actions WHERE host_id = ?`, hostID))
	lockWipe, err := s.DS.Datastore.GetHostLockWipeStatus(ctx, host.Host)
	require.NoError(t, err)
	require.False(t, lockWipe.IsLocked())
	require.False(t, lockWipe.IsWiped())
	require.False(t, lockWipe.IsPendingLock())
	require.False(t, lockWipe.IsPendingWipe())

	// History survives.
	require.Equal(t, 1, countFor(`SELECT COUNT(*) FROM activity_host_past WHERE host_id = ?`, hostID),
		"past activities must be preserved when preserve_host_activities_on_reenrollment is set")

	// The vitals reported by the re-enrollment survive it: the reset runs before they are
	// written, so it must not leave the host with no disk or OS row.
	require.Equal(t, 1, countFor(`SELECT COUNT(*) FROM host_disks WHERE host_id = ?`, hostID),
		"the re-enrollment's own disk vitals must not be wiped by the reset")
	require.Equal(t, 1, countFor(`SELECT COUNT(*) FROM host_operating_system WHERE host_id = ?`, hostID),
		"the re-enrollment's own OS vitals must not be wiped by the reset")

	// And the host is still enrolled.
	hostMDM, err := s.DS.Datastore.GetHostMDM(ctx, hostID)
	require.NoError(t, err)
	require.True(t, hostMDM.Enrolled)
}
