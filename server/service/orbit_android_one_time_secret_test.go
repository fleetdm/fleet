package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/stretchr/testify/require"
)

func TestEnrollRejectSharedSecretForAndroidMDMHosts(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		androidOneTimeEnrollSecrets bool
		androidMDMEnabled           bool
		wantReject                  bool
	}{
		{name: "enabled, Android MDM on", androidOneTimeEnrollSecrets: true, androidMDMEnabled: true, wantReject: true},
		{name: "enabled, Android MDM off", androidOneTimeEnrollSecrets: true, androidMDMEnabled: false},
		{name: "disabled", androidOneTimeEnrollSecrets: false, androidMDMEnabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newOneTimeEnrollFixtureWithMDM(t, func(mdmConfig *config.MDMConfig) {
				mdmConfig.AndroidOneTimeEnrollSecrets = tc.androidOneTimeEnrollSecrets
			})
			f.ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				ac := &fleet.AppConfig{}
				ac.MDM.EnabledAndConfigured = true
				ac.MDM.AndroidEnabledAndConfigured = tc.androidMDMEnabled
				return ac, nil
			}
			var orbitCfg *fleet.DatastoreEnrollOrbitConfig
			f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
				orbitCfg = orbitEnrollConfig(opts)
				return &fleet.Host{ID: 1, UUID: "other", Platform: "darwin"}, nil
			}
			var osqueryCfg *fleet.DatastoreEnrollOsqueryConfig
			f.ds.EnrollOsqueryFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOsqueryOption) (*fleet.Host, error) {
				osqueryCfg = osqueryEnrollConfig(opts)
				return &fleet.Host{ID: 1, OsqueryHostID: new(osqueryCfg.OsqueryHostID), NodeKey: new(osqueryCfg.NodeKey)}, nil
			}

			_, err := f.svc.EnrollOrbit(f.ctx, f.orbitInfo(), "shared-secret", "")
			require.NoError(t, err)
			require.Equal(t, tc.wantReject, orbitCfg.RejectSharedSecretForAndroidMDMHosts)
			require.False(t, orbitCfg.RejectSharedSecretForAppleMDMHosts)
			require.False(t, orbitCfg.RejectSharedSecretForWindowsMDMHosts)

			_, err = f.svc.EnrollOsquery(f.ctx, "shared-secret", f.row.HardwareUUID, f.osqueryDetails())
			require.NoError(t, err)
			require.Equal(t, tc.wantReject, osqueryCfg.RejectSharedSecretForAndroidMDMHosts)
		})
	}
}

func TestEnrollOrbitAndroidOneTimeSecret(t *testing.T) {
	const esid = "esid-1"
	androidHostID := uint(42)
	androidInfo := fleet.OrbitHostInfo{HardwareUUID: esid, HardwareSerial: esid, Platform: "android", ComputerName: "Google Pixel"}

	newFixture := func(t *testing.T, androidOneTimeEnrollSecrets bool) (*oneTimeEnrollFixture, *[]softwareWorkerJobArgs) {
		f := newOneTimeEnrollFixtureWithMDM(t, func(mdmConfig *config.MDMConfig) {
			mdmConfig.AndroidOneTimeEnrollSecrets = androidOneTimeEnrollSecrets
			// Another platform's switch keeps the one-time lookup running when Android's is off.
			mdmConfig.AppleOneTimeEnrollSecrets = true
		})
		f.row = fleet.HostOneTimeEnrollSecret{
			ID: 8, Secret: "android-secret", HostID: &androidHostID, TeamID: new(uint(3)), Platform: "android", HardwareUUID: esid,
		}
		f.ds.GetHostOneTimeEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.HostOneTimeEnrollSecret, error) {
			if secret == f.row.Secret {
				r := f.row
				return &r, nil
			}
			return nil, newNotFoundError()
		}
		f.ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.AndroidEnabledAndConfigured = true
			return ac, nil
		}
		f.ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return &fleet.Host{ID: id, UUID: esid, Platform: "android"}, nil
		}
		f.ds.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
			return &android.Enterprise{ID: 1, EnterpriseID: "LC0123"}, nil
		}
		var jobs []softwareWorkerJobArgs
		f.ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			require.Equal(t, "software_worker", job.Name)
			var args softwareWorkerJobArgs
			require.NoError(t, json.Unmarshal(*job.Args, &args))
			jobs = append(jobs, args)
			return job, nil
		}
		return f, &jobs
	}
	rejectWith := func(f *oneTimeEnrollFixture, reason string, hostID *uint) {
		f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			return nil, &fleet.EnrollmentRejectedError{Reason: reason, HostID: hostID}
		}
	}
	wantResend := softwareWorkerJobArgs{Task: "send_android_fleet_agent_config", HostUUID: esid, EnterpriseName: "enterprises/LC0123"}

	t.Run("the host's secret enrolls into its fleet", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		var received *fleet.DatastoreEnrollOrbitConfig
		f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			received = orbitEnrollConfig(opts)
			return &fleet.Host{ID: androidHostID, UUID: esid, Platform: "android"}, nil
		}
		_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, f.row.Secret, "")
		require.NoError(t, err)
		require.Equal(t, &f.row.ID, received.OneTimeEnrollSecretID)
		require.Equal(t, f.row.TeamID, received.TeamID)
		require.Empty(t, *jobs)
	})

	t.Run("another device's ID is refused", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		info := androidInfo
		info.HardwareUUID = "esid-2"
		_, err := f.svc.EnrollOrbit(f.ctx, info, f.row.Secret, "")
		require.Error(t, err)
		require.Len(t, *f.rejections, 1)
		require.Equal(t, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, (*f.rejections)[0].Reason)
		require.Empty(t, *jobs)
	})

	t.Run("a refused shared secret resends the agent configuration once", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		rejectWith(f, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID)
		for range 3 {
			_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, "shared-secret", "")
			var authErr *fleet.AuthFailedError
			require.ErrorAs(t, err, &authErr)
		}
		require.Equal(t, []softwareWorkerJobArgs{wantResend}, *jobs)
		require.Len(t, *f.rejections, 1)
	})

	t.Run("no resend when the rate limit can't be checked", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		rejectWith(f, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID)
		f.kv.GetFunc = func(ctx context.Context, key string) (*string, error) { return nil, errors.New("redis down") }
		_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, "shared-secret", "")
		require.Error(t, err)
		require.Empty(t, *jobs)
	})

	t.Run("a spent secret resends the agent configuration", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		rejectWith(f, fleet.EnrollmentRejectedOneTimeSecretSpent, &androidHostID)
		_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, f.row.Secret, "")
		require.Error(t, err)
		require.Equal(t, []softwareWorkerJobArgs{wantResend}, *jobs)
	})

	t.Run("no resend", func(t *testing.T) {
		for _, tc := range []struct {
			name                        string
			androidOneTimeEnrollSecrets bool
			reason                      string
			hostID                      *uint
			platform                    string
		}{
			{"switch off", false, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID, "android"},
			{"not an android host", true, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID, "windows"},
			{"no host, as for a deleted one", true, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, nil, "android"},
			{"another reason", true, fleet.EnrollmentRejectedHostIdentityCertRequired, &androidHostID, "android"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				f, jobs := newFixture(t, tc.androidOneTimeEnrollSecrets)
				f.ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
					return &fleet.Host{ID: id, UUID: esid, Platform: tc.platform}, nil
				}
				rejectWith(f, tc.reason, tc.hostID)
				_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, "shared-secret", "")
				require.Error(t, err)
				require.Empty(t, *jobs)
			})
		}
	})
}

// softwareWorkerJobArgs mirrors the fields of the worker's job arguments that these tests check.
type softwareWorkerJobArgs struct {
	Task           string `json:"task"`
	HostUUID       string `json:"host_uuid,omitempty"`
	EnterpriseName string `json:"enterprise_name,omitempty"`
}
