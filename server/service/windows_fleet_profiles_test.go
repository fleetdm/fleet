package service

import (
	"context"
	"html"
	"log/slog"
	"net/http"
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
	admx := html.UnescapeString(cmds[0].Items[0].Data.Content)
	require.Contains(t, admx, `key="SOFTWARE\FleetDM\Orbit"`)
	require.Contains(t, admx, `valueName="EnrollSecret"`)
	require.Contains(t, admx, `class="Machine"`, "the value has to land in HKLM, not HKCU")

	require.Contains(t, html.UnescapeString(cmds[1].Items[0].Data.Content), `<enabled/>`)

	// The secret itself is never stored: the profile carries the placeholder, resolved per enrollment at delivery.
	require.Equal(t, []string{fleet.HostSecretEnrollSecret},
		fleet.ContainsPrefixVars(string(syncML), fleet.HostSecretPrefix))
}

func TestEnsureFleetWindowsProfiles(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	teamID := uint(3)
	noTeamProfile := &fleet.MDMWindowsConfigProfile{ProfileUUID: "w-none"}
	teamProfile := &fleet.MDMWindowsConfigProfile{ProfileUUID: "w-team", TeamID: &teamID}

	newDS := func(existing ...*fleet.MDMWindowsConfigProfile) *mock.Store {
		ds := new(mock.Store)
		ds.ListMDMWindowsConfigProfilesByNameFunc = func(ctx context.Context, name string) ([]*fleet.MDMWindowsConfigProfile, error) {
			require.Equal(t, mdm.FleetWindowsEnrollSecretProfileName, name)
			return existing, nil
		}
		ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
			return []*fleet.TeamSummary{{ID: teamID}}, nil
		}
		return ds
	}

	// Only missing profiles are written. Rewriting an existing one would redeliver it to every host in its team.
	for _, tc := range []struct {
		name     string
		existing []*fleet.MDMWindowsConfigProfile
		want     []*uint
	}{
		{"enabled: one profile per team and no team", nil, []*uint{nil, &teamID}},
		{"enabled: only the missing profile is written", []*fleet.MDMWindowsConfigProfile{noTeamProfile}, []*uint{&teamID}},
		{"enabled: existing profiles are never rewritten", []*fleet.MDMWindowsConfigProfile{noTeamProfile, teamProfile}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := newDS(tc.existing...)
			var written []*uint
			ds.SetOrUpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, cp fleet.MDMWindowsConfigProfile) error {
				require.Equal(t, mdm.FleetWindowsEnrollSecretProfileName, cp.Name)
				written = append(written, cp.TeamID)
				return nil
			}
			require.NoError(t, ensureFleetWindowsProfiles(t.Context(), ds, logger, true))
			require.Equal(t, tc.want, written)
		})
	}

	t.Run("disabled: the enroll secret profile is removed", func(t *testing.T) {
		ds := newDS(noTeamProfile, teamProfile)
		var deleted []string
		ds.DeleteMDMWindowsConfigProfileFunc = func(ctx context.Context, profileUUID string) error {
			deleted = append(deleted, profileUUID)
			return nil
		}

		require.NoError(t, ensureFleetWindowsProfiles(t.Context(), ds, logger, false))
		require.Equal(t, []string{"w-none", "w-team"}, deleted)
		require.False(t, ds.TeamsSummaryFuncInvoked, "removal needs only the profiles that exist")
	})

	t.Run("disabled: nothing to remove costs one read", func(t *testing.T) {
		ds := newDS()
		require.NoError(t, ensureFleetWindowsProfiles(t.Context(), ds, logger, false))
		require.False(t, ds.TeamsSummaryFuncInvoked)
	})
}

func TestDeliversOneTimeEnrollSecret(t *testing.T) {
	// Each platform answers only to its own one-time enroll secrets setting.
	appleOnly := config.AuthConfig{UseOneTimeEnrollSecrets: true}
	windowsOnly := config.AuthConfig{MDMWindowsOneTimeEnrollSecrets: true}

	for _, tc := range []struct {
		name        string
		auth        config.AuthConfig
		profileUUID string
		profileName string
		want        bool
	}{
		{"apple fleetd config", appleOnly, "a-1", mdm.FleetdConfigProfileName, true},
		{"windows enroll secret", windowsOnly, "w-1", mdm.FleetWindowsEnrollSecretProfileName, true},
		{"windows one-time enroll secrets do not guard the apple profile", windowsOnly, "a-1", mdm.FleetdConfigProfileName, false},
		{"apple one-time enroll secrets do not guard the windows profile", appleOnly, "w-1", mdm.FleetWindowsEnrollSecretProfileName, false},
		{"other windows profiles carry no secret", windowsOnly, "w-1", mdm.FleetWindowsOSUpdatesProfileName, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, deliversOneTimeEnrollSecret(tc.auth, tc.profileUUID, tc.profileName))
		})
	}
}

func TestResendWindowsEnrollSecretProfileRequiresWindowsOneTimeEnrollSecrets(t *testing.T) {
	const secretProfileUUID = "w-secret"
	host := &fleet.Host{ID: 1, UUID: "host-uuid", Platform: "windows"}

	newService := func(t *testing.T, windowsOneTimeEnrollSecrets bool) (*Service, *mock.Store) {
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
		cfg.Auth.MDMWindowsOneTimeEnrollSecrets = windowsOneTimeEnrollSecrets
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

	t.Run("disabled: the enroll secret profile is refused on every resend path, before anything is minted", func(t *testing.T) {
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

	t.Run("disabled: other Windows profiles resend as usual", func(t *testing.T) {
		svc, ds := newService(t, false)
		err, _ := resend(svc, "w-custom", "Custom settings")
		require.NoError(t, err)
		require.True(t, ds.ResendHostMDMProfileFuncInvoked)
	})

	t.Run("enabled: the enroll secret profile resends", func(t *testing.T) {
		svc, ds := newService(t, true)
		err, _ := resend(svc, secretProfileUUID, mdm.FleetWindowsEnrollSecretProfileName)
		require.NoError(t, err)
		require.True(t, ds.ResendHostMDMProfileFuncInvoked)
	})
}

func TestPushEnrollSecretToOrphanedEnrollment(t *testing.T) {
	orphaned := &fleet.MDMWindowsEnrolledDevice{ID: 17, MDMDeviceID: "device-17", HostUUID: "host-uuid", MDMEnrollUserID: "not-a-upn"}

	type state struct {
		minted bool
		pushed *fleet.MDMWindowsCommand
	}
	newService := func(t *testing.T, windowsOneTimeEnrollSecrets bool, pending []*fleet.MDMWindowsCommand) (*Service, *mock.Store, *state) {
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
		cfg.Auth.MDMWindowsOneTimeEnrollSecrets = windowsOneTimeEnrollSecrets
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
		require.Contains(t, string(st.pushed.RawCommand), fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret),
			"the secret is resolved at delivery")
	})

	withHost := *orphaned
	withHost.LinkedHostID = new(uint(1))
	unlinked := *orphaned
	unlinked.HostUUID = ""
	for _, tc := range []struct {
		name                        string
		windowsOneTimeEnrollSecrets bool
		device                      *fleet.MDMWindowsEnrolledDevice
		pending                     []*fleet.MDMWindowsCommand
	}{
		{name: "windows one-time enroll secrets disabled", device: orphaned},
		{name: "enrollment never linked to a host", windowsOneTimeEnrollSecrets: true, device: &unlinked},
		{name: "host exists", windowsOneTimeEnrollSecrets: true, device: &withHost},
		{name: "a push is already queued", windowsOneTimeEnrollSecrets: true, device: orphaned,
			pending: []*fleet.MDMWindowsCommand{{TargetLocURI: windowsEnrollSecretPolicyURI}}},
	} {
		t.Run("nothing is pushed: "+tc.name, func(t *testing.T) {
			svc, ds, st := newService(t, tc.windowsOneTimeEnrollSecrets, tc.pending)
			session(t, svc, tc.device)
			require.False(t, st.minted)
			if tc.pending == nil {
				require.False(t, ds.MDMWindowsGetPendingCommandsFuncInvoked, "a session that needs no push costs no query")
			}
		})
	}
}
