package service

import (
	"context"
	"errors"
	"testing"
	"time"

	hostidentity_types "github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
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
	)
	hostInfo := fleet.OrbitHostInfo{HardwareUUID: "hw-uuid-1", HardwareSerial: "SER-OTES", Hostname: "DESKTOP-OTES", Platform: "windows"}
	// The first-install shape: minted before the host existed, so bound to the enrollment only.
	firstInstall := fleet.HostOneTimeEnrollSecret{ID: 7, Secret: "one-time-secret", MDMWindowsEnrollmentID: new(enrollmentID), Platform: "windows"}

	newSvc := func(t *testing.T, secret fleet.HostOneTimeEnrollSecret) (fleet.Service, *mock.DataStore) {
		ds := new(mock.DataStore)
		cfg := config.TestConfig()
		// The Windows switch alone, so this stays a test of the Windows path rather than passing on the macOS one.
		cfg.MDM.WindowsOneTimeEnrollSecrets = true
		svc, _ := newTestServiceWithConfig(t, ds, cfg, nil, nil)

		device := &fleet.MDMWindowsEnrolledDevice{ID: enrollmentID, MDMDeviceID: deviceID, MDMHardwareID: "hw-otes"}
		ds.GetHostOneTimeEnrollSecretFunc = func(ctx context.Context, _ string) (*fleet.HostOneTimeEnrollSecret, error) {
			return &secret, nil
		}
		ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			return &fleet.Host{ID: 42, UUID: "host-uuid-1", Platform: "windows"}, nil
		}
		ds.GetHostIdentityCertByNameFunc = func(ctx context.Context, name string) (*hostidentity_types.HostIdentityCertificate, error) {
			return nil, newNotFoundError()
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			cfg := &fleet.AppConfig{}
			cfg.MDM.WindowsEnabledAndConfigured = true
			return cfg, nil
		}
		ds.MaybeAssociateHostWithScimUserFunc = func(ctx context.Context, hostID uint) error { return nil }
		ds.MDMWindowsGetEnrolledDeviceByIDFunc = func(ctx context.Context, id uint) (*fleet.MDMWindowsEnrolledDevice, error) {
			require.Equal(t, enrollmentID, id)
			return device, nil
		}
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, id string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return device, nil
		}
		ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, _ fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
			return []*fleet.Host{{ID: 42, UUID: "host-uuid-1"}}, nil
		}
		ds.MDMWindowsClaimEnrolledActivityFunc = func(ctx context.Context, mdmHardwareID string, claimedAt time.Time) (bool, error) {
			return true, nil
		}
		return svc, ds
	}

	t.Run("links the host to the enrollment the secret was minted for", func(t *testing.T) {
		// On an administrator resend the enrollment is already linked, so the secret is bound to the host too. Its hardware UUID is
		// this host's, recorded in different case; the comparison is case-insensitive, as on the Apple path.
		resend := firstInstall
		resend.HostID, resend.HardwareUUID = new(uint(42)), "HW-UUID-1"
		for name, secret := range map[string]fleet.HostOneTimeEnrollSecret{"first install": firstInstall, "resend": resend} {
			t.Run(name, func(t *testing.T) {
				svc, ds := newSvc(t, secret)
				var linkedHostUUID, linkedDeviceID string
				ds.MDMWindowsClearDeletedHostTeamFunc = func(ctx context.Context, mdmDeviceID string) error { return nil }
				ds.UpdateMDMWindowsEnrollmentsHostUUIDFunc = func(ctx context.Context, hostUUID string, id string) (bool, error) {
					linkedHostUUID, linkedDeviceID = hostUUID, id
					return true, nil
				}
				_, err := svc.EnrollOrbit(t.Context(), hostInfo, secret.Secret, "")
				require.NoError(t, err)
				require.Equal(t, "host-uuid-1", linkedHostUUID)
				require.Equal(t, deviceID, linkedDeviceID)
			})
		}
	})

	// The secret came over the enrollment's own MDM channel, so a device cannot pair it with another device's Autopilot device
	// ID to claim that device's pending host.
	for _, tc := range []struct {
		name          string
		enrollmentZTD string
		want          string
	}{
		{"the Autopilot device ID comes from the enrollment, not fleetd", "EFDB13F9-44D6-4F99-A93F-08833FCCEF82", "efdb13f9-44d6-4f99-a93f-08833fccef82"},
		{"an enrollment without an Autopilot device ID clears the one fleetd sent", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, ds := newSvc(t, firstInstall)
			ds.MDMWindowsGetEnrolledDeviceByIDFunc = func(ctx context.Context, id uint) (*fleet.MDMWindowsEnrolledDevice, error) {
				return &fleet.MDMWindowsEnrolledDevice{ID: enrollmentID, MDMDeviceID: deviceID, MDMHardwareID: "hw-otes", ZTDRegistrationID: tc.enrollmentZTD}, nil
			}
			var received fleet.DatastoreEnrollOrbitConfig
			ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
				for _, opt := range opts {
					opt(&received)
				}
				return &fleet.Host{ID: 42, UUID: "host-uuid-1", Platform: "windows"}, nil
			}
			ds.MDMWindowsClearDeletedHostTeamFunc = func(ctx context.Context, mdmDeviceID string) error { return nil }
			ds.UpdateMDMWindowsEnrollmentsHostUUIDFunc = func(ctx context.Context, hostUUID string, id string) (bool, error) {
				return true, nil
			}

			withOtherDevicesID := hostInfo
			withOtherDevicesID.AutopilotDeviceID = "11111111-2222-3333-4444-555555555555"
			_, err := svc.EnrollOrbit(t.Context(), withOtherDevicesID, firstInstall.Secret, "")
			require.NoError(t, err)
			require.Equal(t, tc.want, received.HostInfo.AutopilotDeviceID)
		})
	}

	t.Run("an enrollment that no longer exists clears the one fleetd sent", func(t *testing.T) {
		svc, ds := newSvc(t, firstInstall)
		ds.MDMWindowsGetEnrolledDeviceByIDFunc = func(ctx context.Context, id uint) (*fleet.MDMWindowsEnrolledDevice, error) {
			return nil, newNotFoundError()
		}
		var received fleet.DatastoreEnrollOrbitConfig
		ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			for _, opt := range opts {
				opt(&received)
			}
			return &fleet.Host{ID: 42, UUID: "host-uuid-1", Platform: "windows"}, nil
		}

		withOtherDevicesID := hostInfo
		withOtherDevicesID.AutopilotDeviceID = "11111111-2222-3333-4444-555555555555"
		_, err := svc.EnrollOrbit(t.Context(), withOtherDevicesID, firstInstall.Secret, "")
		require.NoError(t, err)
		require.Empty(t, received.HostInfo.AutopilotDeviceID)
	})

	t.Run("failing to read the enrollment fails the enrollment", func(t *testing.T) {
		svc, ds := newSvc(t, firstInstall)
		ds.MDMWindowsGetEnrolledDeviceByIDFunc = func(ctx context.Context, id uint) (*fleet.MDMWindowsEnrolledDevice, error) {
			return nil, errors.New("db unavailable")
		}

		_, err := svc.EnrollOrbit(t.Context(), hostInfo, firstInstall.Secret, "")
		require.Error(t, err, "enrolling without the enrollment's Autopilot device ID could claim another device's pending host")
		require.False(t, ds.EnrollOrbitFuncInvoked)
	})
}
