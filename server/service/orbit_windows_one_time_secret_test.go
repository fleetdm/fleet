package service

import (
	"context"
	"testing"
	"time"

	hostidentity_types "github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// TestEnrollOrbitWindowsOneTimeSecretLink covers the linkage half of one-time enroll secrets on Windows: a host that presents a
// secret minted for a specific MDM enrollment is linked to that enrollment directly, instead of the enrollment being inferred
// from a hardware serial the device asserted about itself.
func TestEnrollOrbitWindowsOneTimeSecretLink(t *testing.T) {
	const (
		enrollmentID = uint(11)
		deviceID     = "device-otes"
		hardwareID   = "hw-otes"
	)

	hostInfo := fleet.OrbitHostInfo{
		HardwareUUID:   "hw-uuid-1",
		HardwareSerial: "SER-OTES",
		Hostname:       "DESKTOP-OTES",
		Platform:       "windows",
	}

	device := func() *fleet.MDMWindowsEnrolledDevice {
		return &fleet.MDMWindowsEnrolledDevice{
			ID:              enrollmentID,
			MDMDeviceID:     deviceID,
			MDMHardwareID:   hardwareID,
			MDMEnrollUserID: "user@example.com", // valid UPN: user-driven enrollment
			CreatedAt:       time.Now().UTC().Add(-2 * time.Minute),
		}
	}

	newSvc := func(t *testing.T) (fleet.Service, *enrollOrbitStore, *TestServerOpts) {
		inner := new(mock.Store)
		ds := &enrollOrbitStore{
			Store: inner,
			enrollOrbitFunc: func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
				return &fleet.Host{ID: 42, UUID: "host-uuid-1", Platform: "windows"}, nil
			},
		}
		cfg := config.TestConfig()
		cfg.Auth.UseOneTimeEnrollSecrets = true
		serverOpts := &TestServerOpts{KeyValueStore: memoryKVStore()}
		svc, _ := newTestServiceWithConfig(t, ds, cfg, nil, nil, serverOpts)

		// A Windows secret binds to the enrollment, so host_id and the hardware identifiers are unset at mint time.
		inner.GetHostOneTimeEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.HostOneTimeEnrollSecret, error) {
			if secret != "one-time-secret" {
				return nil, newNotFoundError()
			}
			return &fleet.HostOneTimeEnrollSecret{
				ID:                     7,
				Secret:                 secret,
				MDMWindowsEnrollmentID: new(enrollmentID),
				Platform:               "windows",
			}, nil
		}
		inner.GetHostIdentityCertByNameFunc = func(ctx context.Context, name string) (*hostidentity_types.HostIdentityCertificate, error) {
			return nil, newNotFoundError()
		}
		inner.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			cfg := &fleet.AppConfig{}
			cfg.MDM.WindowsEnabledAndConfigured = true
			return cfg, nil
		}
		inner.MaybeAssociateHostWithScimUserFunc = func(ctx context.Context, hostID uint) error { return nil }
		inner.MDMWindowsGetEnrolledDeviceByIDFunc = func(ctx context.Context, id uint) (*fleet.MDMWindowsEnrolledDevice, error) {
			require.Equal(t, enrollmentID, id)
			return device(), nil
		}
		inner.MDMWindowsConflictingEnrollmentHardwareIDFunc = func(ctx context.Context, hostUUID, mdmHardwareID string) (bool, string, error) {
			return false, "", nil
		}
		inner.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, id string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return device(), nil
		}
		inner.GetWindowsEnrollmentDefaultFleetFunc = func(ctx context.Context) (*uint, string, error) { return nil, "", nil }
		inner.ReplaceHostDeviceMappingFunc = func(ctx context.Context, hostID uint, m []*fleet.HostDeviceMapping, source string) error {
			return nil
		}
		inner.ScimUserByUserNameOrEmailFunc = func(ctx context.Context, name string, email string) (*fleet.ScimUser, error) {
			return nil, newNotFoundError()
		}
		inner.DeleteHostSCIMUserMappingFunc = func(ctx context.Context, hostID uint) ([]fleet.ActivityTypeResentCertificate, error) {
			return nil, nil
		}
		inner.ListHostsLiteByUUIDsFunc = func(ctx context.Context, _ fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
			return []*fleet.Host{{ID: 42, UUID: "host-uuid-1", Hostname: hostInfo.Hostname}}, nil
		}
		inner.MDMWindowsClaimEnrolledActivityFunc = func(ctx context.Context, mdmHardwareID string, claimedAt time.Time) (bool, error) {
			return true, nil
		}
		serverOpts.ActivityMock.NewActivityFunc = func(context.Context, *activity_api.User, activity_api.ActivityDetails) error {
			return nil
		}
		return svc, ds, serverOpts
	}

	t.Run("links to the enrollment the secret was minted for, without consulting the serial", func(t *testing.T) {
		svc, ds, _ := newSvc(t)
		var linkedHostUUID, linkedDeviceID string
		ds.UpdateMDMWindowsEnrollmentsHostUUIDFunc = func(ctx context.Context, hostUUID string, id string) (bool, error) {
			linkedHostUUID, linkedDeviceID = hostUUID, id
			return true, nil
		}

		nodeKey, err := svc.EnrollOrbit(t.Context(), hostInfo, "one-time-secret", "")
		require.NoError(t, err)
		require.NotEmpty(t, nodeKey)

		require.True(t, ds.MDMWindowsGetEnrolledDeviceByIDFuncInvoked, "the enrollment must come from the secret")
		require.Equal(t, "host-uuid-1", linkedHostUUID)
		require.Equal(t, deviceID, linkedDeviceID)
		require.False(t, ds.MDMWindowsGetUnlinkedEnrolledDeviceWithHardwareSerialFuncInvoked,
			"a device-asserted serial must not be consulted when the secret already identifies the enrollment")
	})

	t.Run("a host already claimed by other hardware is not relinked", func(t *testing.T) {
		svc, ds, _ := newSvc(t)
		ds.MDMWindowsConflictingEnrollmentHardwareIDFunc = func(ctx context.Context, hostUUID, mdmHardwareID string) (bool, string, error) {
			return true, "other-hw", nil
		}
		ds.UpdateMDMWindowsEnrollmentsHostUUIDFunc = func(ctx context.Context, hostUUID string, id string) (bool, error) {
			t.Fatal("must not link a host claimed by different hardware")
			return false, nil
		}

		nodeKey, err := svc.EnrollOrbit(t.Context(), hostInfo, "one-time-secret", "")
		require.NoError(t, err, "a refused link must not fail the enrollment")
		require.NotEmpty(t, nodeKey)
	})

	t.Run("a failed enrollment lookup leaves the host enrolled but unlinked", func(t *testing.T) {
		svc, ds, _ := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceByIDFunc = func(ctx context.Context, id uint) (*fleet.MDMWindowsEnrolledDevice, error) {
			return nil, newNotFoundError()
		}
		ds.UpdateMDMWindowsEnrollmentsHostUUIDFunc = func(ctx context.Context, hostUUID string, id string) (bool, error) {
			t.Fatal("must not link when the enrollment could not be loaded")
			return false, nil
		}

		nodeKey, err := svc.EnrollOrbit(t.Context(), hostInfo, "one-time-secret", "")
		require.NoError(t, err, "linkage is bookkeeping and must never fail the enrollment")
		require.NotEmpty(t, nodeKey)
	})

	t.Run("a shared secret still falls back to the serial branch", func(t *testing.T) {
		svc, ds, _ := newSvc(t)
		ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
			return &fleet.EnrollSecret{Secret: secret}, nil
		}
		ds.MDMWindowsGetUnlinkedEnrolledDeviceWithHardwareSerialFunc = func(ctx context.Context, serial string) (*fleet.MDMWindowsEnrolledDevice, error) {
			require.Equal(t, hostInfo.HardwareSerial, serial)
			return nil, newNotFoundError()
		}

		nodeKey, err := svc.EnrollOrbit(t.Context(), hostInfo, "shared-secret", "")
		require.NoError(t, err)
		require.NotEmpty(t, nodeKey)
		require.False(t, ds.MDMWindowsGetEnrolledDeviceByIDFuncInvoked)
		require.True(t, ds.MDMWindowsGetUnlinkedEnrolledDeviceWithHardwareSerialFuncInvoked,
			"hosts that predate one-time secrets must keep the existing linkage path")
	})
}
