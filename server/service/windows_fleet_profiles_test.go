package service

import (
	"context"
	"html"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestWindowsEnrollSecretProfileSyncML(t *testing.T) {
	syncML, err := windowsEnrollSecretProfileSyncML()
	require.NoError(t, err)

	// The delivery path parses the stored profile with this, so the profile has to survive it. Two top-level elements: the
	// ADMX ingest must be applied before the policy it declares can be set.
	cmds, err := fleet.UnmarshallMultiTopLevelXMLProfile(syncML)
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	require.Equal(t, "Add", cmds[0].XMLName.Local, "the ingest must come first")
	require.Equal(t, "Replace", cmds[1].XMLName.Local)

	require.Len(t, cmds[0].Items, 1)
	require.Len(t, cmds[1].Items, 1)
	require.Equal(t, windowsEnrollSecretADMXInstallURI, *cmds[0].Items[0].Target)
	require.Equal(t, windowsEnrollSecretPolicyURI, *cmds[1].Items[0].Target)

	// The ingested ADMX is what points the policy at the key orbit reads. If either of these drifts, the secret lands
	// somewhere orbit never looks and the recovery path silently stops working.
	// Data holds the raw inner XML, so the ADMX is still entity-escaped here; that is exactly what goes on the wire, and
	// the device's parser is what unescapes it back into a policy definition.
	admx := html.UnescapeString(cmds[0].Items[0].Data.Content)
	require.Contains(t, admx, `key="SOFTWARE\FleetDM\Orbit"`)
	require.Contains(t, admx, `valueName="EnrollSecret"`)
	require.Contains(t, admx, `class="Machine"`, "the value has to land in HKLM, not HKCU")

	// The secret itself is never stored: the profile carries the placeholder, expanded per enrollment at delivery.
	policy := html.UnescapeString(cmds[1].Items[0].Data.Content)
	placeholder := fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret)
	require.Contains(t, policy, placeholder)
	require.Contains(t, policy, `<enabled/>`)

	// And the expander has to be able to find it: it scans for the prefix, so an escaping change that mangled the
	// placeholder would leave the profile delivering a literal "$FLEET_HOST_SECRET_ENROLL_SECRET" to the registry.
	require.Equal(t, []string{fleet.HostSecretEnrollSecret},
		fleet.ContainsPrefixVars(string(syncML), fleet.HostSecretPrefix))
}

func TestEnsureFleetWindowsProfiles(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	teamID := uint(3)

	newDS := func() *mock.Store {
		ds := new(mock.Store)
		// A team with no shared secret still gets a row; "no team" is omitted when it has none, which is why the target
		// list adds it back.
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return []*fleet.EnrollSecret{{TeamID: &teamID}}, nil
		}
		return ds
	}

	t.Run("enabled: one profile per team and no team", func(t *testing.T) {
		ds := newDS()
		var gotTeams []*uint
		var gotName string
		var gotSyncML []byte
		ds.SetOrUpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, cp fleet.MDMWindowsConfigProfile) error {
			gotTeams = append(gotTeams, cp.TeamID)
			gotName = cp.Name
			gotSyncML = cp.SyncML
			return nil
		}

		require.NoError(t, ensureFleetWindowsProfiles(t.Context(), ds, logger, true))
		require.Equal(t, []*uint{&teamID, nil}, gotTeams)
		require.Equal(t, mdm.FleetWindowsEnrollSecretProfileName, gotName)
		require.Contains(t, string(gotSyncML), fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret))
		require.False(t, ds.DeleteMDMWindowsConfigProfileByTeamAndNameFuncInvoked)
	})

	t.Run("disabled: the carrier is removed", func(t *testing.T) {
		ds := newDS()
		var deletedTeams []*uint
		ds.DeleteMDMWindowsConfigProfileByTeamAndNameFunc = func(ctx context.Context, tid *uint, name string) error {
			require.Equal(t, mdm.FleetWindowsEnrollSecretProfileName, name)
			deletedTeams = append(deletedTeams, tid)
			return nil
		}

		// Leaving it behind would deliver a profile whose placeholder nothing expands, writing a literal
		// "$FLEET_HOST_SECRET_ENROLL_SECRET" into the registry for orbit to try to enroll with.
		require.NoError(t, ensureFleetWindowsProfiles(t.Context(), ds, logger, false))
		require.Equal(t, []*uint{&teamID, nil}, deletedTeams)
		require.False(t, ds.SetOrUpdateMDMWindowsConfigProfileFuncInvoked)
	})

	t.Run("disabled: an absent profile is not an error", func(t *testing.T) {
		ds := newDS()
		ds.DeleteMDMWindowsConfigProfileByTeamAndNameFunc = func(ctx context.Context, tid *uint, name string) error {
			return newNotFoundError()
		}
		require.NoError(t, ensureFleetWindowsProfiles(t.Context(), ds, logger, false))
	})
}

func TestDeliversOneTimeEnrollSecret(t *testing.T) {
	bothOn := config.AuthConfig{UseOneTimeEnrollSecrets: true, MDMWindowsOneTimeEnrollSecrets: true}

	for _, tc := range []struct {
		name        string
		auth        config.AuthConfig
		profileUUID string
		profileName string
		want        bool
	}{
		{"apple fleetd config", bothOn, "a" + "-1", mdm.FleetdConfigProfileName, true},
		{"windows enroll secret", bothOn, "w" + "-1", mdm.FleetWindowsEnrollSecretProfileName, true},
		{"windows os updates carries no secret", bothOn, "w" + "-1", mdm.FleetWindowsOSUpdatesProfileName, false},
		{"apple profile with the windows name", bothOn, "a" + "-1", mdm.FleetWindowsEnrollSecretProfileName, false},
		{"windows profile with the apple name", bothOn, "w" + "-1", mdm.FleetdConfigProfileName, false},
		{"a user profile that copied the name", bothOn, "w" + "-1", "Fleetd enroll secret " + strings.Repeat("x", 3), false},
		{"declaration", bothOn, "d" + "-1", mdm.FleetdConfigProfileName, false},

		// Each platform answers to its own switch, so one being on must not gate the other in.
		{
			"windows off leaves the windows profile to the ordinary resend rules",
			config.AuthConfig{UseOneTimeEnrollSecrets: true},
			"w" + "-1", mdm.FleetWindowsEnrollSecretProfileName, false,
		},
		{
			"apple off leaves the apple profile to the ordinary resend rules",
			config.AuthConfig{MDMWindowsOneTimeEnrollSecrets: true},
			"a" + "-1", mdm.FleetdConfigProfileName, false,
		},
		{
			"windows on, apple off still guards the windows profile",
			config.AuthConfig{MDMWindowsOneTimeEnrollSecrets: true},
			"w" + "-1", mdm.FleetWindowsEnrollSecretProfileName, true,
		},
		{
			"both off guards nothing",
			config.AuthConfig{},
			"w" + "-1", mdm.FleetWindowsEnrollSecretProfileName, false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, deliversOneTimeEnrollSecret(tc.auth, tc.profileUUID, tc.profileName))
		})
	}

	// The resend-from-verifying carve-out stays Apple-only: a Windows profile reaches verified on the SyncML ack, so it
	// never gets stuck there and does not need the exemption.
	require.True(t, isFleetdConfigProfile("a-1", mdm.FleetdConfigProfileName))
	require.False(t, isFleetdConfigProfile("w-1", mdm.FleetWindowsEnrollSecretProfileName))
}
