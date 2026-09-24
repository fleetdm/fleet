package service

import (
	"context"
	"html"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
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

	// The secret itself is never stored: the profile carries the placeholder, resolved per enrollment at delivery.
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

		// Leaving it behind would leave an administrator resend able to mint secrets that, with the switch off, enrollment
		// no longer accepts.
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

func TestResendWindowsEnrollSecretProfileRequiresTheSwitch(t *testing.T) {
	const secretProfileUUID = "w-secret"
	host := &fleet.Host{ID: 1, UUID: "host-uuid", Platform: "windows"}

	newService := func(t *testing.T, windowsSwitch bool) (*Service, *mock.Store) {
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			cfg := &fleet.AppConfig{}
			cfg.MDM.WindowsEnabledAndConfigured = true
			return cfg, nil
		}
		ds.GetHostMDMProfileInstallStatusFunc = func(ctx context.Context, hostUUID, profileUUID string) (fleet.MDMDeliveryStatus, error) {
			return fleet.MDMDeliveryVerified, nil
		}
		ds.ResendHostMDMProfileFunc = func(ctx context.Context, hostUUID, profileUUID string) error { return nil }
		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, profileUUID string) (*fleet.MDMWindowsConfigProfile, error) {
			return &fleet.MDMWindowsConfigProfile{ProfileUUID: profileUUID, Name: mdm.FleetWindowsEnrollSecretProfileName}, nil
		}
		ds.BatchResendMDMProfileToHostsFunc = func(ctx context.Context, profileUUID string, f fleet.BatchResendMDMProfileFilters) (int64, error) {
			return 0, nil
		}
		cfg := config.TestConfig()
		cfg.Auth.MDMWindowsOneTimeEnrollSecrets = windowsSwitch
		opts := &TestServerOpts{}
		svc, _ := newTestServiceWithConfig(t, ds, cfg, nil, nil, opts)
		opts.ActivityMock.NewActivityFunc = func(ctx context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}
		return svc.(validationMiddleware).Service.(*Service), ds
	}
	resend := func(svc *Service, profileUUID, profileName string) (error, bool) {
		var gotErr error
		var rejected bool
		checkAndResendHostMDMProfile(t.Context(), svc, host, func(err error, r bool) { gotErr, rejected = err, r },
			profileUUID, profileName, nil)
		return gotErr, rejected
	}

	t.Run("switch off refuses the carrier on every resend path, before anything is minted", func(t *testing.T) {
		// Covers the window before the reconciler deletes the profile, and a delete that failed. The datastore would mint on the
		// profile's name, so this is the only thing between a resend and a credential for a feature that is off.
		svc, ds := newService(t, false)

		err, rejected := resend(svc, secretProfileUUID, mdm.FleetWindowsEnrollSecretProfileName)
		require.True(t, rejected)
		var status interface{ Status() int }
		require.ErrorAs(t, err, &status)
		require.Equal(t, http.StatusConflict, status.Status())
		require.False(t, ds.ResendHostMDMProfileFuncInvoked)

		err = svc.BatchResendMDMProfileToHosts(test.UserContext(t.Context(), test.UserAdmin), secretProfileUUID,
			fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
		require.ErrorAs(t, err, &status)
		require.Equal(t, http.StatusConflict, status.Status())
		require.False(t, ds.BatchResendMDMProfileToHostsFuncInvoked)
	})

	t.Run("switch off leaves other Windows profiles alone", func(t *testing.T) {
		svc, ds := newService(t, false)
		err, _ := resend(svc, "w-custom", "Custom settings")
		require.NoError(t, err)
		require.True(t, ds.ResendHostMDMProfileFuncInvoked)
	})

	t.Run("switch on lets the carrier through", func(t *testing.T) {
		svc, ds := newService(t, true)
		err, _ := resend(svc, secretProfileUUID, mdm.FleetWindowsEnrollSecretProfileName)
		require.NoError(t, err)
		require.True(t, ds.ResendHostMDMProfileFuncInvoked)
	})
}

func TestPushEnrollSecretToOrphanedEnrollment(t *testing.T) {
	// A programmatic enrollment, so the fleetd presence check answers without the datastore and only the push is exercised. Its
	// linked host is gone: LinkedHostID is nil, as the session's enrollment load reports a deleted host.
	orphaned := &fleet.MDMWindowsEnrolledDevice{ID: 17, MDMDeviceID: "device-17", HostUUID: "host-uuid", MDMEnrollUserID: "not-a-upn"}

	type state struct {
		minted bool
		pushed *fleet.MDMWindowsCommand
	}
	newService := func(t *testing.T, windowsSwitch bool, pending []*fleet.MDMWindowsCommand) (*Service, *mock.Store, *state) {
		st := &state{}
		ds := new(mock.Store)
		ds.MDMWindowsGetPendingCommandsFunc = func(ctx context.Context, enrollmentID uint) ([]*fleet.MDMWindowsCommand, error) {
			require.Equal(t, orphaned.ID, enrollmentID)
			return pending, nil
		}
		ds.MintWindowsMDMOneTimeEnrollSecretFunc = func(ctx context.Context, enrollmentID uint) error {
			require.Equal(t, orphaned.ID, enrollmentID)
			st.minted = true
			return nil
		}
		ds.MDMWindowsInsertCommandForHostsFunc = func(ctx context.Context, deviceIDs []string, cmd *fleet.MDMWindowsCommand) error {
			require.True(t, st.minted, "the secret must exist before the command that resolves it is queued")
			require.Equal(t, []string{orphaned.MDMDeviceID}, deviceIDs)
			st.pushed = cmd
			return nil
		}
		cfg := config.TestConfig()
		cfg.Auth.MDMWindowsOneTimeEnrollSecrets = windowsSwitch
		svc, _ := newTestServiceWithConfig(t, ds, cfg, nil, nil)
		return svc.(validationMiddleware).Service.(*Service), ds, st
	}
	session := func(t *testing.T, svc *Service, d *fleet.MDMWindowsEnrolledDevice) {
		require.NoError(t, svc.processNewSessionAlert(t.Context(), "1", d, fleet.ProtoCmdOperation{}))
	}

	t.Run("a deleted host's enrollment is pushed a secret through the profile's own SyncML", func(t *testing.T) {
		svc, _, st := newService(t, true, nil)
		session(t, svc, orphaned)

		require.NotNil(t, st.pushed)
		require.Equal(t, windowsEnrollSecretPolicyURI, st.pushed.TargetLocURI)
		raw := string(st.pushed.RawCommand)
		require.Contains(t, raw, windowsEnrollSecretADMXInstallURI)
		require.Contains(t, raw, windowsEnrollSecretPolicyURI)
		require.Contains(t, raw, fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret), "the secret is resolved at delivery")
	})

	withHost := *orphaned
	withHost.LinkedHostID = new(uint(1))
	unlinked := *orphaned
	unlinked.HostUUID = ""
	for _, tc := range []struct {
		name          string
		windowsSwitch bool
		device        *fleet.MDMWindowsEnrolledDevice
		pending       []*fleet.MDMWindowsCommand
	}{
		{name: "switch off", device: orphaned},
		{name: "enrollment never linked to a host", windowsSwitch: true, device: &unlinked},
		{name: "host exists", windowsSwitch: true, device: &withHost},
		{name: "a push is already queued", windowsSwitch: true, device: orphaned,
			pending: []*fleet.MDMWindowsCommand{{TargetLocURI: windowsEnrollSecretPolicyURI}}},
	} {
		t.Run("nothing is pushed: "+tc.name, func(t *testing.T) {
			svc, ds, st := newService(t, tc.windowsSwitch, tc.pending)
			session(t, svc, tc.device)
			require.False(t, st.minted)
			require.Nil(t, st.pushed)
			if tc.pending == nil {
				require.False(t, ds.MDMWindowsGetPendingCommandsFuncInvoked, "a session that needs no push costs no query")
			}
		})
	}
}
