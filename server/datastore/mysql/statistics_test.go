package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatistics(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ShouldSend", testStatisticsShouldSend},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testStatisticsShouldSend(t *testing.T, ds *Datastore) {
	eh := ctxerr.MockHandler{}
	// Mock the error handler to always return an error
	eh.RetrieveImpl = func(flush bool) ([]*ctxerr.StoredError, error) {
		require.False(t, flush)
		return []*ctxerr.StoredError{
			{Count: 10, Chain: json.RawMessage(`[{"stack": ["a","b","c","d"]}]`)},
		}, nil
	}
	ctxb := context.Background()
	ctx := ctxerr.NewContext(ctxb, eh)

	fleetConfig := config.FleetConfig{Osquery: config.OsqueryConfig{DetailUpdateInterval: 1 * time.Hour}}

	premiumLicense := &fleet.LicenseInfo{Tier: fleet.TierPremium, Organization: "Fleet"}
	freeLicense := &fleet.LicenseInfo{Tier: fleet.TierFree}

	var builtinLabels int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &builtinLabels, `SELECT COUNT(*) FROM labels`)
	})

	// First time running with no hosts
	stats, shouldSend, err := ds.ShouldSendStatistics(license.NewContext(ctx, premiumLicense), time.Millisecond, fleetConfig)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, "premium", stats.LicenseTier)
	assert.Equal(t, "Fleet", stats.Organization)
	assert.Equal(t, 0, stats.NumHostsEnrolled)
	assert.Equal(t, 0, stats.NumUsers)
	assert.Equal(t, 0, stats.NumSoftwareVersions)
	assert.Equal(t, 0, stats.NumHostSoftwares)
	assert.Equal(t, 0, stats.NumSoftwareTitles)
	assert.Equal(t, 0, stats.NumHostSoftwareInstalledPaths)
	assert.Equal(t, 0, stats.NumSoftwareCPEs)
	assert.Equal(t, 0, stats.NumSoftwareCVEs)
	assert.Equal(t, 0, stats.NumTeams)
	assert.Equal(t, 0, stats.NumPolicies)
	assert.Equal(t, 0, stats.NumQueries)
	assert.Equal(t, builtinLabels, stats.NumLabels)
	assert.Equal(t, false, stats.SoftwareInventoryEnabled)
	assert.Equal(t, true, stats.SystemUsersEnabled)
	assert.Equal(t, false, stats.VulnDetectionEnabled)
	assert.Equal(t, false, stats.HostsStatusWebHookEnabled)
	assert.Equal(t, 0, stats.NumWeeklyActiveUsers)
	assert.Equal(t, 0, stats.NumWeeklyPolicyViolationDaysActual)
	assert.Equal(t, 0, stats.NumWeeklyPolicyViolationDaysPossible)
	assert.Equal(t, `[{"count":10,"loc":["a","b","c"]}]`, string(stats.StoredErrors))
	assert.Equal(t, []fleet.HostsCountByOsqueryVersion{}, stats.HostsEnrolledByOsqueryVersion) // should be empty slice instead of nil
	assert.Equal(t, []fleet.HostsCountByOrbitVersion{}, stats.HostsEnrolledByOrbitVersion)     // should be empty slice instead of nil
	assert.Equal(t, false, stats.MDMMacOsEnabled)
	assert.Equal(t, false, stats.HostExpiryEnabled)
	assert.Equal(t, false, stats.MDMWindowsEnabled)
	assert.Equal(t, false, stats.LiveQueryDisabled)
	assert.Equal(t, false, stats.AIFeaturesDisabled)
	assert.Equal(t, false, stats.MaintenanceWindowsEnabled)
	assert.Equal(t, false, stats.MaintenanceWindowsConfigured)
	assert.Equal(t, 0, stats.NumHostsFleetDesktopEnabled)

	firstIdentifier := stats.AnonymousIdentifier

	err = ds.RecordStatisticsSent(ctx)
	require.NoError(t, err)

	// Create new host for test
	h1, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   ptr.String("M"),
		OsqueryVersion:  "4.9.0",
	})
	require.NoError(t, err)

	// Create host_orbit_info record for test
	require.NoError(
		t, ds.SetOrUpdateHostOrbitInfo(
			ctx, h1.ID, "1.1.0", sql.NullString{String: "1.1.0", Valid: true}, sql.NullBool{Bool: true, Valid: true},
		),
	)

	// Create two new users for test
	u1, err := ds.NewUser(ctx, &fleet.User{
		Password:                 []byte("foobar"),
		AdminForcedPasswordReset: false,
		Email:                    "baz@example.com",
		SSOEnabled:               false,
		GlobalRole:               ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)
	_, err = ds.NewUser(ctx, &fleet.User{
		Password:                 []byte("foobar"),
		AdminForcedPasswordReset: false,
		Email:                    "qux@example.com",
		SSOEnabled:               false,
		GlobalRole:               ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)
	// Create a session for user baz, but not qux (so only 1 is active)
	_, err = ds.NewSession(ctx, u1.ID, "session_key")
	require.NoError(t, err)

	// Create new team for test
	_, err = ds.NewTeam(ctx, &fleet.Team{
		Name:        "footeam",
		Description: "team of foo",
	})
	require.NoError(t, err)

	// Create new global policy for test
	_, err = ds.NewGlobalPolicy(ctx, ptr.Uint(1), fleet.PolicyPayload{
		Name:        "testpolicy",
		Query:       "select 1;",
		Description: "test policy desc",
		Resolution:  "test policy resolution",
	})
	require.NoError(t, err)

	// Create new label for test
	_, err = ds.NewLabel(ctx, &fleet.Label{
		Name:        "testlabel",
		Query:       "select 1;",
		Platform:    "darwin",
		Description: "test label description",
	})
	require.NoError(t, err)

	// Create new app cfg for test
	cfg, err := ds.NewAppConfig(ctx, &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName:    "Test",
			OrgLogoURL: "localhost:8080/logo.png",
		},
	})
	require.NoError(t, err)

	// Initialize policy violation days for test
	pvdJSON, err := json.Marshal(PolicyViolationDays{FailingHostCount: 5, TotalHostCount: 10})
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO
			aggregated_stats (id, global_stats, type, json_value, created_at, updated_at)
		VALUES (?, ?, ?, CAST(? AS JSON), ?, ?)
		ON DUPLICATE KEY UPDATE
			json_value = VALUES(json_value),
			updated_at = VALUES(updated_at)
	`, 0, true, aggregatedStatsTypePolicyViolationsDays, pvdJSON, time.Now().Add(-48*time.Hour), time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)

	require.NoError(t, err)
	cfg.Features.EnableSoftwareInventory = false
	cfg.Features.EnableHostUsers = false
	cfg.VulnerabilitySettings.DatabasesPath = ""
	cfg.WebhookSettings.HostStatusWebhook.Enable = true
	cfg.MDM.EnabledAndConfigured = true
	cfg.HostExpirySettings.HostExpiryEnabled = true
	cfg.MDM.WindowsEnabledAndConfigured = true
	cfg.ServerSettings.LiveQueryDisabled = true
	err = ds.SaveAppConfig(ctx, cfg)
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond) // ensure the DB timestamp is not in the same second

	// Running with 1 host
	stats, shouldSend, err = ds.ShouldSendStatistics(license.NewContext(ctx, premiumLicense), time.Millisecond, fleetConfig)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.NotEmpty(t, stats.AnonymousIdentifier)
	assert.NotEmpty(t, stats.FleetVersion)
	assert.Equal(t, "premium", stats.LicenseTier)
	assert.Equal(t, "Fleet", stats.Organization)
	assert.Equal(t, 1, stats.NumHostsEnrolled)
	assert.Equal(t, 2, stats.NumUsers)
	assert.Equal(t, 0, stats.NumSoftwareVersions)
	assert.Equal(t, 0, stats.NumHostSoftwares)
	assert.Equal(t, 0, stats.NumSoftwareTitles)
	assert.Equal(t, 0, stats.NumHostSoftwareInstalledPaths)
	assert.Equal(t, 0, stats.NumSoftwareCPEs)
	assert.Equal(t, 0, stats.NumSoftwareCVEs)
	assert.Equal(t, 1, stats.NumTeams)
	assert.Equal(t, 1, stats.NumPolicies)
	assert.Equal(t, 1, stats.NumQueries)
	assert.Equal(t, builtinLabels+1, stats.NumLabels)
	assert.Equal(t, false, stats.SoftwareInventoryEnabled)
	assert.Equal(t, false, stats.SystemUsersEnabled)
	assert.Equal(t, false, stats.VulnDetectionEnabled)
	assert.Equal(t, true, stats.HostsStatusWebHookEnabled)
	assert.Equal(t, 1, stats.NumWeeklyActiveUsers)
	assert.Equal(t, 5, stats.NumWeeklyPolicyViolationDaysActual)
	assert.Equal(t, 10, stats.NumWeeklyPolicyViolationDaysPossible)
	assert.Equal(t, `[{"count":10,"loc":["a","b","c"]}]`, string(stats.StoredErrors))
	assert.Equal(t, []fleet.HostsCountByOsqueryVersion{{OsqueryVersion: "4.9.0", NumHosts: 1}}, stats.HostsEnrolledByOsqueryVersion)
	assert.Equal(t, []fleet.HostsCountByOrbitVersion{{OrbitVersion: "1.1.0", NumHosts: 1}}, stats.HostsEnrolledByOrbitVersion)
	assert.Equal(t, false, stats.AIFeaturesDisabled)
	assert.Equal(t, false, stats.MaintenanceWindowsEnabled)
	assert.Equal(t, false, stats.MaintenanceWindowsConfigured)
	assert.Equal(t, 1, stats.NumHostsFleetDesktopEnabled)

	err = ds.RecordStatisticsSent(ctx)
	require.NoError(t, err)

	// If we try right away, it shouldn't ask to send
	stats, shouldSend, err = ds.ShouldSendStatistics(license.NewContext(ctx, premiumLicense), fleet.StatisticsFrequency, fleetConfig)
	require.NoError(t, err)
	assert.False(t, shouldSend)

	time.Sleep(1100 * time.Millisecond) // ensure the DB timestamp is not in the same second

	// create a few more hosts, with platforms and os versions
	_, err = ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		OsqueryHostID:   ptr.String("S"),
		Platform:        "rhel",
		OSVersion:       "Fedora 35",
	})
	require.NoError(t, err)

	_, err = ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("3"),
		UUID:            "3",
		Hostname:        "foo.local3",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "40-65-EC-6F-C4-59",
		OsqueryHostID:   ptr.String("T"),
		Platform:        "rhel",
		OSVersion:       "Fedora 35",
	})
	require.NoError(t, err)

	_, err = ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("4"),
		UUID:            "4",
		Hostname:        "foo.local4",
		PrimaryIP:       "192.168.1.4",
		PrimaryMac:      "50-65-EC-6F-C4-59",
		OsqueryHostID:   ptr.String("U"),
		Platform:        "macos",
		OSVersion:       "10.11.12",
	})
	require.NoError(t, err)

	_, err = ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("5"),
		UUID:            "5",
		Hostname:        "foo.local5",
		PrimaryIP:       "192.168.1.5",
		PrimaryMac:      "60-65-EC-6F-C4-59",
		OsqueryHostID:   ptr.String("V"),
		Platform:        "rhel",
		OSVersion:       "Fedora 36",
	})
	require.NoError(t, err)

	// Lower the frequency to trigger an "outdated" sent
	stats, shouldSend, err = ds.ShouldSendStatistics(license.NewContext(ctx, premiumLicense), time.Millisecond, fleetConfig)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, firstIdentifier, stats.AnonymousIdentifier)
	assert.Equal(t, "premium", stats.LicenseTier)
	assert.Equal(t, "Fleet", stats.Organization)
	assert.Equal(t, 5, stats.NumHostsEnrolled)
	assert.Equal(t, 2, stats.NumUsers)
	assert.Equal(t, 0, stats.NumQueries)
	assert.Equal(t, 0, stats.NumSoftwareVersions)
	assert.Equal(t, 0, stats.NumHostSoftwares)
	assert.Equal(t, 0, stats.NumSoftwareTitles)
	assert.Equal(t, 0, stats.NumHostSoftwareInstalledPaths)
	assert.Equal(t, 0, stats.NumSoftwareCPEs)
	assert.Equal(t, 0, stats.NumSoftwareCVEs)
	assert.Equal(t, 0, stats.NumWeeklyActiveUsers)          // no active user since last stats were sent
	require.Len(t, stats.HostsEnrolledByOperatingSystem, 3) // empty platform, rhel and macos
	assert.Equal(t, 5, stats.NumWeeklyPolicyViolationDaysActual)
	require.ElementsMatch(t, []fleet.HostsCountByOSVersion{
		{Version: "Fedora 35", NumEnrolled: 2},
		{Version: "Fedora 36", NumEnrolled: 1},
	}, stats.HostsEnrolledByOperatingSystem["rhel"])
	require.ElementsMatch(t, []fleet.HostsCountByOSVersion{
		{Version: "10.11.12", NumEnrolled: 1},
	}, stats.HostsEnrolledByOperatingSystem["macos"])
	require.ElementsMatch(t, []fleet.HostsCountByOSVersion{
		{Version: "", NumEnrolled: 1},
	}, stats.HostsEnrolledByOperatingSystem[""])
	assert.Equal(t, `[{"count":10,"loc":["a","b","c"]}]`, string(stats.StoredErrors))
	assert.Equal(t, false, stats.AIFeaturesDisabled)
	assert.Equal(t, false, stats.MaintenanceWindowsEnabled)
	assert.Equal(t, false, stats.MaintenanceWindowsConfigured)
	assert.Equal(t, 1, stats.NumHostsFleetDesktopEnabled)

	// Create multiple new sessions for a single user
	_, err = ds.NewSession(ctx, u1.ID, "session_key2")
	require.NoError(t, err)
	_, err = ds.NewSession(ctx, u1.ID, "session_key3")
	require.NoError(t, err)
	_, err = ds.NewSession(ctx, u1.ID, "session_key4")
	require.NoError(t, err)

	// CleanupStatistics resets policy violation days
	err = ds.CleanupStatistics(ctx)
	require.NoError(t, err)

	// wait a bit and resend statistics
	time.Sleep(1100 * time.Millisecond) // ensure the DB timestamp is not in the same second

	stats, shouldSend, err = ds.ShouldSendStatistics(license.NewContext(ctx, premiumLicense), time.Millisecond, fleetConfig)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, stats.AnonymousIdentifier, firstIdentifier)
	assert.Equal(t, "premium", stats.LicenseTier)
	assert.Equal(t, "Fleet", stats.Organization)
	assert.Equal(t, 5, stats.NumHostsEnrolled)
	assert.Equal(t, 2, stats.NumUsers)
	assert.Equal(t, 0, stats.NumQueries)
	assert.Equal(t, 0, stats.NumSoftwareVersions)
	assert.Equal(t, 0, stats.NumHostSoftwares)
	assert.Equal(t, 0, stats.NumSoftwareTitles)
	assert.Equal(t, 0, stats.NumHostSoftwareInstalledPaths)
	assert.Equal(t, 0, stats.NumSoftwareCPEs)
	assert.Equal(t, 0, stats.NumSoftwareCVEs)
	assert.Equal(t, 1, stats.NumWeeklyActiveUsers)
	assert.Equal(t, 0, stats.NumWeeklyPolicyViolationDaysActual)
	assert.Equal(t, 0, stats.NumWeeklyPolicyViolationDaysPossible)
	assert.Equal(t, `[{"count":10,"loc":["a","b","c"]}]`, string(stats.StoredErrors))
	assert.Equal(t, false, stats.AIFeaturesDisabled)
	assert.Equal(t, false, stats.MaintenanceWindowsEnabled)
	assert.Equal(t, false, stats.MaintenanceWindowsConfigured)
	assert.Equal(t, 1, stats.NumHostsFleetDesktopEnabled)

	// Add host to test hosts not responding stats
	_, err = ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now().Add(-3 * time.Hour),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("6"),
		UUID:            "6",
		Hostname:        "non-responsive.local",
		PrimaryIP:       "192.168.1.6",
		PrimaryMac:      "30-65-EC-6F-C4-66",
		OsqueryHostID:   ptr.String("NR"),
	})
	require.NoError(t, err)

	stats, shouldSend, err = ds.ShouldSendStatistics(license.NewContext(ctx, premiumLicense), time.Millisecond, fleetConfig)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, firstIdentifier, stats.AnonymousIdentifier)
	assert.Equal(t, "premium", stats.LicenseTier)
	assert.Equal(t, "Fleet", stats.Organization)
	assert.Equal(t, 6, stats.NumHostsEnrolled)
	assert.Equal(t, 1, stats.NumHostsNotResponding)

	// trigger again with a free license, organization should be "unknown"
	time.Sleep(1100 * time.Millisecond) // ensure the DB timestamp is not in the same second
	stats, shouldSend, err = ds.ShouldSendStatistics(license.NewContext(ctx, freeLicense), time.Millisecond, fleetConfig)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, firstIdentifier, stats.AnonymousIdentifier)
	assert.Equal(t, "free", stats.LicenseTier)
	assert.Equal(t, "unknown", stats.Organization)

	fleetConfig.Vulnerabilities = config.VulnerabilitiesConfig{DatabasesPath: "some/path/vulns"}
	stats, shouldSend, err = ds.ShouldSendStatistics(license.NewContext(ctx, freeLicense), time.Millisecond, fleetConfig)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.True(t, stats.VulnDetectionEnabled)
}
