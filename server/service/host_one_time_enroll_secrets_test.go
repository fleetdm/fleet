package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	hostidentity_types "github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mock"
	akvmock "github.com/fleetdm/fleet/v4/server/mock/redis_advanced"
	"github.com/stretchr/testify/require"
)

// memoryKVStore is an in-memory fleet.KeyValueStore for exercising the
// activity rate limit without Redis.
func memoryKVStore() *akvmock.AdvancedKeyValueStore {
	var mu sync.Mutex
	values := map[string]string{}
	return &akvmock.AdvancedKeyValueStore{
		SetFunc: func(ctx context.Context, key, value string, expireTime time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			values[key] = value
			return nil
		},
		GetFunc: func(ctx context.Context, key string) (*string, error) {
			mu.Lock()
			defer mu.Unlock()
			v, ok := values[key]
			if !ok {
				return nil, nil
			}
			return &v, nil
		},
	}
}

type oneTimeEnrollFixture struct {
	ds         *mock.DataStore
	svc        fleet.Service
	ctx        context.Context
	row        fleet.HostOneTimeEnrollSecret
	rejections *[]fleet.ActivityTypeHostEnrollmentRejected
	enrolled   *int
}

func newOneTimeEnrollFixture(t *testing.T, useOneTimeEnrollSecrets bool) *oneTimeEnrollFixture {
	hostID := uint(42)
	row := fleet.HostOneTimeEnrollSecret{
		ID:             7,
		Secret:         "one-time-secret",
		HostID:         &hostID,
		TeamID:         new(uint(3)),
		Platform:       "darwin",
		HardwareUUID:   "UUID-1",
		HardwareSerial: "SERIAL-1",
	}

	ds := new(mock.DataStore)
	cfg := config.TestConfig()
	cfg.Auth.UseOneTimeEnrollSecrets = useOneTimeEnrollSecrets
	opts := &TestServerOpts{KeyValueStore: memoryKVStore()}
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, opts)

	var rejections []fleet.ActivityTypeHostEnrollmentRejected
	var enrolled int
	opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, a activity_api.ActivityDetails) error {
		switch act := a.(type) {
		case fleet.ActivityTypeHostEnrollmentRejected:
			rejections = append(rejections, act)
		case fleet.ActivityTypeFleetEnrolled:
			enrolled++
		}
		return nil
	}

	ds.GetHostOneTimeEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.HostOneTimeEnrollSecret, error) {
		if secret == row.Secret {
			r := row
			return &r, nil
		}
		return nil, newNotFoundError()
	}
	ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		if secret == "shared-secret" {
			return &fleet.EnrollSecret{Secret: secret, TeamID: new(uint(9))}, nil
		}
		return nil, newNotFoundError()
	}
	ds.GetHostIdentityCertByNameFunc = func(ctx context.Context, name string) (*hostidentity_types.HostIdentityCertificate, error) {
		return nil, newNotFoundError()
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{}
		ac.MDM.EnabledAndConfigured = true
		return ac, nil
	}
	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{ID: tid}, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{ID: id, Hostname: "victim-mac"}, nil
	}
	ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
		return &fleet.Host{ID: hostID, UUID: row.HardwareUUID, Platform: "darwin"}, nil
	}
	ds.EnrollOsqueryFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOsqueryOption) (*fleet.Host, error) {
		return &fleet.Host{ID: hostID, UUID: row.HardwareUUID, Platform: "darwin"}, nil
	}
	ds.ExtendHostOrbitDebugUntilFunc = func(ctx context.Context, hostID uint, until time.Time) error { return nil }
	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error { return nil }
	ds.SerialUpdateHostFunc = func(ctx context.Context, host *fleet.Host) error { return nil }

	return &oneTimeEnrollFixture{ds: ds, svc: svc, ctx: ctx, row: row, rejections: &rejections, enrolled: &enrolled}
}

func (f *oneTimeEnrollFixture) orbitInfo() fleet.OrbitHostInfo {
	return fleet.OrbitHostInfo{HardwareUUID: f.row.HardwareUUID, HardwareSerial: f.row.HardwareSerial, Platform: f.row.Platform, Hostname: "mac"}
}

func (f *oneTimeEnrollFixture) osqueryDetails() map[string]map[string]string {
	return map[string]map[string]string{
		"system_info": {"uuid": f.row.HardwareUUID, "hardware_serial": f.row.HardwareSerial},
		"os_version":  {"platform": f.row.Platform},
	}
}

func orbitEnrollConfig(opts []fleet.DatastoreEnrollOrbitOption) *fleet.DatastoreEnrollOrbitConfig {
	c := &fleet.DatastoreEnrollOrbitConfig{}
	for _, o := range opts {
		o(c)
	}
	return c
}

func osqueryEnrollConfig(opts []fleet.DatastoreEnrollOsqueryOption) *fleet.DatastoreEnrollOsqueryConfig {
	c := &fleet.DatastoreEnrollOsqueryConfig{}
	for _, o := range opts {
		o(c)
	}
	return c
}

func TestEnrollOrbitWithOneTimeEnrollSecret(t *testing.T) {
	t.Run("matching identifiers consume the secret and use its team", func(t *testing.T) {
		f := newOneTimeEnrollFixture(t, true)
		var got *fleet.DatastoreEnrollOrbitConfig
		f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			got = orbitEnrollConfig(opts)
			return &fleet.Host{ID: *f.row.HostID, UUID: f.row.HardwareUUID, Platform: "darwin"}, nil
		}

		nodeKey, err := f.svc.EnrollOrbit(f.ctx, f.orbitInfo(), f.row.Secret, "")
		require.NoError(t, err)
		require.NotEmpty(t, nodeKey)
		require.NotNil(t, got.OneTimeEnrollSecretID)
		require.Equal(t, f.row.ID, *got.OneTimeEnrollSecretID)
		require.Equal(t, f.row.TeamID, got.TeamID)
		require.False(t, got.RejectSharedSecretForMDMHosts)
		require.Empty(t, *f.rejections)
		require.Equal(t, 1, *f.enrolled)
	})

	t.Run("identifier mismatch is refused before touching the datastore", func(t *testing.T) {
		f := newOneTimeEnrollFixture(t, true)
		info := f.orbitInfo()
		info.HardwareSerial = "SOMEONE-ELSE"

		for range 3 {
			_, err := f.svc.EnrollOrbit(f.ctx, info, f.row.Secret, "")
			requireAuthFailed(t, err)
		}
		require.False(t, f.ds.EnrollOrbitFuncInvoked)
		// one activity for three attempts against the same host and reason
		require.Len(t, *f.rejections, 1)
		got := (*f.rejections)[0]
		require.Equal(t, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, got.Reason)
		require.Equal(t, f.row.HostID, got.HostID)
		require.Equal(t, "victim-mac", got.HostDisplayName)
		require.Equal(t, "SOMEONE-ELSE", got.HostSerial)
		require.Equal(t, string(fleet.EnrollmentPlaneOrbit), got.EnrollmentPlane)

		// a different reason for the same host is its own activity
		f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			return nil, &fleet.EnrollmentRejectedError{Reason: fleet.EnrollmentRejectedOneTimeSecretSpent, HostID: f.row.HostID}
		}
		_, err := f.svc.EnrollOrbit(f.ctx, f.orbitInfo(), f.row.Secret, "")
		requireAuthFailed(t, err)
		require.Len(t, *f.rejections, 2)
		require.Equal(t, fleet.EnrollmentRejectedOneTimeSecretSpent, (*f.rejections)[1].Reason)
	})

	t.Run("shared secret carries the rejection option only when the flag is on", func(t *testing.T) {
		for _, flag := range []bool{true, false} {
			f := newOneTimeEnrollFixture(t, flag)
			var got *fleet.DatastoreEnrollOrbitConfig
			f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
				got = orbitEnrollConfig(opts)
				return &fleet.Host{ID: 1, UUID: "other", Platform: "darwin"}, nil
			}
			_, err := f.svc.EnrollOrbit(f.ctx, f.orbitInfo(), "shared-secret", "")
			require.NoError(t, err)
			require.Nil(t, got.OneTimeEnrollSecretID)
			require.Equal(t, new(uint(9)), got.TeamID)
			require.Equal(t, flag, got.RejectSharedSecretForMDMHosts)
		}
	})

	t.Run("datastore rejection of a shared secret is generic to the caller and recorded", func(t *testing.T) {
		f := newOneTimeEnrollFixture(t, true)
		victim := uint(77)
		f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			return nil, &fleet.EnrollmentRejectedError{Reason: fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, HostID: &victim}
		}
		_, err := f.svc.EnrollOrbit(f.ctx, f.orbitInfo(), "shared-secret", "")
		requireAuthFailed(t, err)
		var orbitErr fleet.OrbitError
		require.NotErrorAs(t, err, &orbitErr)
		require.Len(t, *f.rejections, 1)
		require.Equal(t, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, (*f.rejections)[0].Reason)
		require.Equal(t, &victim, (*f.rejections)[0].HostID)
		require.Zero(t, *f.enrolled)
	})

	t.Run("unknown secret is still invalid", func(t *testing.T) {
		f := newOneTimeEnrollFixture(t, true)
		_, err := f.svc.EnrollOrbit(f.ctx, f.orbitInfo(), "nope", "")
		requireAuthFailed(t, err)
		require.Empty(t, *f.rejections)
	})
}

func TestEnrollOsqueryWithOneTimeEnrollSecret(t *testing.T) {
	t.Run("matching identifiers consume the secret and use its team", func(t *testing.T) {
		f := newOneTimeEnrollFixture(t, true)
		var got *fleet.DatastoreEnrollOsqueryConfig
		f.ds.EnrollOsqueryFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOsqueryOption) (*fleet.Host, error) {
			got = osqueryEnrollConfig(opts)
			return &fleet.Host{ID: *f.row.HostID, UUID: f.row.HardwareUUID, Platform: "darwin", OsqueryHostID: new(got.OsqueryHostID), NodeKey: new(got.NodeKey)}, nil
		}

		nodeKey, err := f.svc.EnrollOsquery(f.ctx, f.row.Secret, f.row.HardwareUUID, f.osqueryDetails())
		require.NoError(t, err)
		require.NotEmpty(t, nodeKey)
		require.NotNil(t, got.OneTimeEnrollSecretID)
		require.Equal(t, f.row.ID, *got.OneTimeEnrollSecretID)
		require.Equal(t, f.row.TeamID, got.TeamID)
		require.Equal(t, f.row.HardwareUUID, got.HardwareUUID)
		require.Equal(t, f.row.HardwareSerial, got.HardwareSerial)
	})

	t.Run("missing or mismatched identifiers are refused", func(t *testing.T) {
		f := newOneTimeEnrollFixture(t, true)
		cases := map[string]map[string]map[string]string{
			"no host details":  nil,
			"wrong uuid":       {"system_info": {"uuid": "OTHER", "hardware_serial": f.row.HardwareSerial}, "os_version": {"platform": "darwin"}},
			"wrong platform":   {"system_info": {"uuid": f.row.HardwareUUID, "hardware_serial": f.row.HardwareSerial}, "os_version": {"platform": "ubuntu"}},
			"missing platform": {"system_info": {"uuid": f.row.HardwareUUID, "hardware_serial": f.row.HardwareSerial}},
		}
		for name, details := range cases {
			t.Run(name, func(t *testing.T) {
				_, err := f.svc.EnrollOsquery(f.ctx, f.row.Secret, f.row.HardwareUUID, details)
				require.Error(t, err)
				require.ErrorContains(t, err, "enroll failed")
			})
		}
		require.False(t, f.ds.EnrollOsqueryFuncInvoked)
		// same host, same reason: a single activity across all attempts
		require.Len(t, *f.rejections, 1)
		require.Equal(t, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, (*f.rejections)[0].Reason)
		require.Equal(t, string(fleet.EnrollmentPlaneOsquery), (*f.rejections)[0].EnrollmentPlane)
	})

	t.Run("shared secret carries the rejection option only when the flag is on", func(t *testing.T) {
		for _, flag := range []bool{true, false} {
			f := newOneTimeEnrollFixture(t, flag)
			var got *fleet.DatastoreEnrollOsqueryConfig
			f.ds.EnrollOsqueryFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOsqueryOption) (*fleet.Host, error) {
				got = osqueryEnrollConfig(opts)
				return &fleet.Host{ID: 1, OsqueryHostID: new(got.OsqueryHostID), NodeKey: new(got.NodeKey)}, nil
			}
			_, err := f.svc.EnrollOsquery(f.ctx, "shared-secret", f.row.HardwareUUID, f.osqueryDetails())
			require.NoError(t, err)
			require.Nil(t, got.OneTimeEnrollSecretID)
			require.Equal(t, flag, got.RejectSharedSecretForMDMHosts)
		}
	})

	t.Run("datastore rejection is generic and recorded once", func(t *testing.T) {
		f := newOneTimeEnrollFixture(t, true)
		f.ds.EnrollOsqueryFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOsqueryOption) (*fleet.Host, error) {
			return nil, &fleet.EnrollmentRejectedError{Reason: fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, HostID: f.row.HostID}
		}
		for range 2 {
			_, err := f.svc.EnrollOsquery(f.ctx, "shared-secret", f.row.HardwareUUID, f.osqueryDetails())
			require.ErrorContains(t, err, "enroll failed")
		}
		require.Len(t, *f.rejections, 1)
	})
}

func TestRecordEnrollmentRejectedWithoutKeyValueStore(t *testing.T) {
	// With no key-value store the rejection is still refused and recorded; the
	// rate limit simply does not apply.
	f := newOneTimeEnrollFixture(t, true)
	svc, ctx := newTestServiceWithConfig(t, f.ds, func() config.FleetConfig {
		cfg := config.TestConfig()
		cfg.Auth.UseOneTimeEnrollSecrets = true
		return cfg
	}(), nil, nil, &TestServerOpts{})
	info := f.orbitInfo()
	info.HardwareUUID = "OTHER"
	for range 2 {
		_, err := svc.EnrollOrbit(ctx, info, f.row.Secret, "")
		requireAuthFailed(t, err)
	}
}

// requireAuthFailed asserts the generic authentication failure orbit gets for
// every rejected enrollment, whatever the reason.
func requireAuthFailed(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var authErr *fleet.AuthFailedError
	require.ErrorAs(t, err, &authErr)
}

func TestEnsureFleetdConfigWithOneTimeEnrollSecrets(t *testing.T) {
	ctx := t.Context()
	ds := new(mock.Store)
	mdmConfig := config.MDMConfig{AppleSCEPCert: "./testdata/server.pem", AppleSCEPKey: "./testdata/server.key"}
	signingCert, _, _, err := mdmConfig.AppleSCEP()
	require.NoError(t, err)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{}
		ac.ServerSettings.ServerURL = "https://fleet.example.com"
		return ac, nil
	}
	// team 1 has a shared secret, team 2 has none, and "no team" has none at
	// all (so the aggregate omits it entirely).
	ds.GetMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, identifier string) (*fleet.MDMAppleConfigProfile, error) {
		return nil, newNotFoundError()
	}
	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{
			{Secret: "team-1-shared", TeamID: new(uint(1))},
			{Secret: "", TeamID: new(uint(2))},
		}, nil
	}
	var upserted []*fleet.MDMAppleConfigProfile
	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
		upserted = ps
		return nil
	}

	require.NoError(t, ensureFleetProfiles(ctx, ds, slog.New(slog.DiscardHandler), signingCert.Certificate[0], true))

	fleetdByTeam := map[string]*fleet.MDMAppleConfigProfile{}
	caCount := 0
	for _, p := range upserted {
		key := "no-team"
		if p.TeamID != nil {
			key = fmt.Sprintf("team-%d", *p.TeamID)
		}
		switch p.Name {
		case mdm.FleetdConfigProfileName:
			fleetdByTeam[key] = p
		case mdm.FleetCAConfigProfileName:
			caCount++
		}
	}
	require.Len(t, fleetdByTeam, 3, "fleetd profile for team 1, team 2 and no team")
	require.Equal(t, 3, caCount)
	for key, p := range fleetdByTeam {
		require.Contains(t, string(p.Mobileconfig), fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret), key)
		require.NotContains(t, string(p.Mobileconfig), "team-1-shared", key)
	}

	// flag off keeps today's behavior: team 2 falls back to... nothing, since
	// there is no global secret, and no "no team" profile is generated.
	upserted = nil
	require.NoError(t, ensureFleetProfiles(ctx, ds, slog.New(slog.DiscardHandler), signingCert.Certificate[0], false))
	var fleetdOff int
	for _, p := range upserted {
		if p.Name == mdm.FleetdConfigProfileName {
			fleetdOff++
			require.Contains(t, string(p.Mobileconfig), "team-1-shared")
			require.NotContains(t, string(p.Mobileconfig), fleet.HostSecretPrefix)
		}
	}
	require.Equal(t, 1, fleetdOff)
}

func TestResendFleetdProfileWithOneTimeEnrollSecrets(t *testing.T) {
	const (
		fleetdProfileUUID = "a-fleetd-profile"
		customProfileUUID = "a-custom-profile"
		hostID            = uint(1)
	)
	newSvc := func(t *testing.T, flag bool, status fleet.MDMDeliveryStatus) (*mock.Store, fleet.Service, context.Context) {
		ds := new(mock.Store)
		cfg := config.TestConfig()
		cfg.Auth.UseOneTimeEnrollSecrets = flag
		svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{
			License:             &fleet.LicenseInfo{Tier: fleet.TierPremium},
			SkipCreateTestUsers: true,
		})
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
		}
		ds.HostLiteFunc = func(ctx context.Context, hid uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hid, UUID: "host-uuid-1", Platform: "darwin"}, nil
		}
		ds.GetMDMAppleConfigProfileFunc = func(ctx context.Context, pid string) (*fleet.MDMAppleConfigProfile, error) {
			name := "Custom profile"
			if pid == fleetdProfileUUID {
				name = mdm.FleetdConfigProfileName
			}
			return &fleet.MDMAppleConfigProfile{ProfileUUID: pid, Name: name}, nil
		}
		ds.GetHostMDMProfileInstallStatusFunc = func(ctx context.Context, hostUUID string, profUUID string) (fleet.MDMDeliveryStatus, error) {
			return status, nil
		}
		ds.ResendHostMDMProfileFunc = func(ctx context.Context, hostUUID, profUUID string) error { return nil }
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
		return ds, svc, ctx
	}

	t.Run("admin can resend the fleetd profile while verifying", func(t *testing.T) {
		ds, svc, ctx := newSvc(t, true, fleet.MDMDeliveryVerifying)
		require.NoError(t, svc.ResendHostMDMProfile(ctx, hostID, fleetdProfileUUID))
		require.True(t, ds.ResendHostMDMProfileFuncInvoked)
	})

	t.Run("verified and failed are resendable as before", func(t *testing.T) {
		for _, status := range []fleet.MDMDeliveryStatus{fleet.MDMDeliveryVerified, fleet.MDMDeliveryFailed} {
			ds, svc, ctx := newSvc(t, true, status)
			require.NoError(t, svc.ResendHostMDMProfile(ctx, hostID, fleetdProfileUUID))
			require.True(t, ds.ResendHostMDMProfileFuncInvoked)
		}
	})

	t.Run("pending stays a conflict", func(t *testing.T) {
		ds, svc, ctx := newSvc(t, true, fleet.MDMDeliveryPending)
		err := svc.ResendHostMDMProfile(ctx, hostID, fleetdProfileUUID)
		require.Error(t, err)
		require.ErrorContains(t, err, "can’t be resent")
		require.False(t, ds.ResendHostMDMProfileFuncInvoked)
	})

	t.Run("custom profiles in verifying are still a conflict", func(t *testing.T) {
		ds, svc, ctx := newSvc(t, true, fleet.MDMDeliveryVerifying)
		err := svc.ResendHostMDMProfile(ctx, hostID, customProfileUUID)
		require.ErrorContains(t, err, "can’t be resent")
		require.False(t, ds.ResendHostMDMProfileFuncInvoked)
	})

	t.Run("flag off: fleetd profile in verifying is a conflict", func(t *testing.T) {
		ds, svc, ctx := newSvc(t, false, fleet.MDMDeliveryVerifying)
		err := svc.ResendHostMDMProfile(ctx, hostID, fleetdProfileUUID)
		require.ErrorContains(t, err, "can’t be resent")
		require.False(t, ds.ResendHostMDMProfileFuncInvoked)

		ds, svc, ctx = newSvc(t, false, fleet.MDMDeliveryVerified)
		require.NoError(t, svc.ResendHostMDMProfile(ctx, hostID, fleetdProfileUUID))
		require.True(t, ds.ResendHostMDMProfileFuncInvoked)
	})

	t.Run("device-token resend of the fleetd profile is refused with a descriptive error", func(t *testing.T) {
		ds, svc, ctx := newSvc(t, true, fleet.MDMDeliveryVerified)
		host := &fleet.Host{ID: hostID, UUID: "host-uuid-1", Platform: "darwin"}
		err := svc.ResendDeviceHostMDMProfile(ctx, host, fleetdProfileUUID)
		require.Error(t, err)
		require.ErrorContains(t, err, "Ask your IT admin")
		var statusErr interface{ Status() int }
		require.ErrorAs(t, err, &statusErr)
		require.Equal(t, http.StatusForbidden, statusErr.Status())
		require.False(t, ds.ResendHostMDMProfileFuncInvoked)

		// other profiles resend as before
		require.NoError(t, svc.ResendDeviceHostMDMProfile(ctx, host, customProfileUUID))
		require.True(t, ds.ResendHostMDMProfileFuncInvoked)
	})

	t.Run("flag off: device-token resend of the fleetd profile behaves as before", func(t *testing.T) {
		ds, svc, ctx := newSvc(t, false, fleet.MDMDeliveryVerified)
		host := &fleet.Host{ID: hostID, UUID: "host-uuid-1", Platform: "darwin"}
		require.NoError(t, svc.ResendDeviceHostMDMProfile(ctx, host, fleetdProfileUUID))
		require.True(t, ds.ResendHostMDMProfileFuncInvoked)
	})
}

func TestEnsureFleetdConfigRemovesPlaceholderProfilesWhenOff(t *testing.T) {
	ctx := t.Context()
	mdmConfig := config.MDMConfig{AppleSCEPCert: "./testdata/server.pem", AppleSCEPKey: "./testdata/server.key"}
	signingCert, _, _, err := mdmConfig.AppleSCEP()
	require.NoError(t, err)

	type deletion struct {
		teamID     *uint
		identifier string
	}
	newDS := func(existingProfile string) (*mock.Store, *[]deletion) {
		ds := new(mock.Store)
		var deleted []deletion
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.ServerSettings.ServerURL = "https://fleet.example.com"
			return ac, nil
		}
		// team 1 has no shared secret and there is no global secret, so the
		// aggregate has no "no team" entry at all.
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return []*fleet.EnrollSecret{{Secret: "", TeamID: new(uint(1))}}, nil
		}
		ds.GetMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, identifier string) (*fleet.MDMAppleConfigProfile, error) {
			return &fleet.MDMAppleConfigProfile{TeamID: teamID, Identifier: identifier, Mobileconfig: []byte(existingProfile)}, nil
		}
		ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, identifier string) error {
			deleted = append(deleted, deletion{teamID, identifier})
			return nil
		}
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
			require.Empty(t, ps)
			return nil
		}
		return ds, &deleted
	}
	placeholderProfile := "<string>" + fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret) + "</string>"

	t.Run("placeholder fleetd profiles are removed for the team and for no team", func(t *testing.T) {
		ds, deleted := newDS(placeholderProfile)
		require.NoError(t, ensureFleetProfiles(ctx, ds, slog.New(slog.DiscardHandler), signingCert.Certificate[0], false))
		// only the fleetd profile; the CA profile has nothing to do with enroll secrets
		require.ElementsMatch(t, []deletion{
			{nil, mobileconfig.FleetdConfigPayloadIdentifier},
			{new(uint(1)), mobileconfig.FleetdConfigPayloadIdentifier},
		}, *deleted)
	})

	t.Run("profiles built from a shared secret are left alone", func(t *testing.T) {
		ds, deleted := newDS("<string>an-old-shared-secret</string>")
		require.NoError(t, ensureFleetProfiles(ctx, ds, slog.New(slog.DiscardHandler), signingCert.Certificate[0], false))
		require.Empty(t, *deleted)
	})

	t.Run("nothing is removed while the setting is on", func(t *testing.T) {
		ds, deleted := newDS(placeholderProfile)
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
			require.NotEmpty(t, ps)
			return nil
		}
		require.NoError(t, ensureFleetProfiles(ctx, ds, slog.New(slog.DiscardHandler), signingCert.Certificate[0], true))
		require.Empty(t, *deleted)
		require.False(t, ds.GetMDMAppleConfigProfileByTeamAndIdentifierFuncInvoked)
	})
}
