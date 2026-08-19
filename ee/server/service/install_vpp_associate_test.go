package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstallVPPAppPostValidation_AssociateAssetsRouting verifies that the
// VPP-asset Associate call uses clientUserIds for Account-Driven User Enrolled
// (BYOD) hosts and falls back to serialNumbers for everything else. This is
// the silent-failure path identified in #31138.
func TestInstallVPPAppPostValidation_AssociateAssetsRouting(t *testing.T) {
	const (
		hostID         = uint(7)
		hostUUID       = "host-uuid-7"
		hostSerial     = "SERIAL-7"
		managedAppleID = "user@example.com"
		adamID         = "989804926"
		bearerToken    = "bearer-tok-1"
	)

	type captured struct {
		body []byte
	}

	// handleRegisterUserV1 emulates Apple's synchronous v1 registerVPPUserSrv
	// endpoint — the token is in the request body, not the Authorization header.
	handleRegisterUserV1 := func(t *testing.T, w http.ResponseWriter, r *http.Request) {
		t.Helper()
		var body struct {
			SToken            string `json:"sToken"`
			ClientUserIDStr   string `json:"clientUserIdStr"`
			ManagedAppleIDStr string `json:"managedAppleIDStr"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, bearerToken, body.SToken)
		_, _ = fmt.Fprintf(w, `{"status":0,"user":{"userId":12345,"status":"Registered","clientUserIdStr":%q,"managedAppleIDStr":%q}}`,
			body.ClientUserIDStr, body.ManagedAppleIDStr)
	}

	// Common mock-server setup. Captures the AssociateAssets body so the test
	// can assert on the wire payload.
	setupServer := func(t *testing.T, capt *captured) {
		t.Helper()
		setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assignments"):
				assert.Equal(t, "Bearer "+bearerToken, r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(`{"assignments": []}`))
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets"):
				assert.Equal(t, "Bearer "+bearerToken, r.Header.Get("Authorization"))
				_, _ = fmt.Fprintf(w, `{"assets":[{"adamId":%q,"pricingParam":"STDQ","availableCount":5}]}`, adamID)
			case r.Method == http.MethodPost && r.URL.Path == "/registerVPPUserSrv":
				handleRegisterUserV1(t, w, r)
			case r.Method == http.MethodPost && r.URL.Path == "/assets/associate":
				assert.Equal(t, "Bearer "+bearerToken, r.Header.Get("Authorization"))
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				capt.body = b
				_, _ = w.Write([]byte(`{"eventId":"associate-evt"}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})
	}

	// Common datastore mock setup, parameterized by whether the host is enrolled
	// via Account-Driven User Enrollment (ADUE). The routing decision keys off
	// the host's primary nano_enrollments row (id = host UUID): ADUE devices
	// enroll as "User Enrollment (Device)", every other device-channel
	// enrollment (including manual-profile BYOD) is "Device". It does NOT key
	// off host_mdm.is_personal_enrollment (see #48879), so this drives
	// GetNanoMDMEnrollment rather than GetHostMDM.
	setupDS := func(t *testing.T, isUserEnrollment bool) *mock.Store {
		t.Helper()
		ds := new(mock.Store)
		ds.GetNanoMDMEnrollmentFunc = func(_ context.Context, id string) (*fleet.NanoEnrollment, error) {
			require.Equal(t, hostUUID, id)
			if isUserEnrollment {
				return &fleet.NanoEnrollment{ID: hostUUID, DeviceID: hostUUID, Type: "User Enrollment (Device)", Enabled: true}, nil
			}
			// Device-channel enrollment (company-owned manual OR manual-profile
			// BYOD): the primary row is type "Device".
			return &fleet.NanoEnrollment{ID: hostUUID, DeviceID: hostUUID, Type: "Device", Enabled: true}, nil
		}
		ds.GetVPPTokenByTeamIDFunc = func(_ context.Context, _ *uint) (*fleet.VPPTokenDB, error) {
			return &fleet.VPPTokenDB{ID: 99, Token: bearerToken, RenewDate: time.Now().Add(24 * time.Hour)}, nil
		}
		ds.GetHostManagedAppleIDFunc = func(_ context.Context, id uint) (string, error) {
			require.Equal(t, hostID, id)
			return managedAppleID, nil
		}
		ds.GetVPPClientUserFunc = func(_ context.Context, _ uint, _ string) (*fleet.VPPClientUser, error) {
			return nil, &notFoundError{}
		}
		ds.InsertVPPClientUserFunc = func(_ context.Context, _ *fleet.VPPClientUser) error {
			return nil
		}
		ds.InsertHostVPPSoftwareInstallFunc = func(_ context.Context, _ uint, _ fleet.VPPAppID, _ string, _ string, _ fleet.HostSoftwareInstallOptions) error {
			return nil
		}
		// No managed app configuration → the pre-flight substitution check is a
		// no-op and this test exercises the license-assignment routing only.
		ds.GetVPPAppConfigurationFunc = func(_ context.Context, _ fleet.InstallableDevicePlatform, _ string, _ uint) ([]byte, error) {
			return nil, &notFoundError{}
		}
		return ds
	}

	host := &fleet.Host{ID: hostID, UUID: hostUUID, HardwareSerial: hostSerial, Platform: "ios"}
	vppApp := &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: adamID, Platform: fleet.IOSPlatform}}}

	t.Run("personal enrollment routes via clientUserIds", func(t *testing.T) {
		var capt captured
		setupServer(t, &capt)

		ds := setupDS(t, true)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)

		require.NotEmpty(t, capt.body, "AssociateAssets request must have been sent")
		var got struct {
			ClientUserIds []string `json:"clientUserIds"`
			SerialNumbers []string `json:"serialNumbers"`
		}
		require.NoError(t, json.Unmarshal(capt.body, &got))
		require.Len(t, got.ClientUserIds, 1)
		require.NotEmpty(t, got.ClientUserIds[0])
		require.Empty(t, got.SerialNumbers, "personal enrollment must not send serialNumbers")

		// The cached row was also persisted so subsequent calls reuse the same UUID.
		require.True(t, ds.InsertVPPClientUserFuncInvoked)
	})

	t.Run("device-channel enrollment keeps SerialNumbers", func(t *testing.T) {
		var capt captured
		setupServer(t, &capt)

		ds := setupDS(t, false)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)

		require.NotEmpty(t, capt.body)
		var got struct {
			ClientUserIds []string `json:"clientUserIds"`
			SerialNumbers []string `json:"serialNumbers"`
		}
		require.NoError(t, json.Unmarshal(capt.body, &got))
		require.Equal(t, []string{hostSerial}, got.SerialNumbers, "device-channel hosts must use serialNumbers")
		require.Empty(t, got.ClientUserIds, "device-channel hosts must not send clientUserIds")

		// User-provisioning datastore writes must NOT happen on the device path.
		require.False(t, ds.GetVPPClientUserFuncInvoked)
		require.False(t, ds.InsertVPPClientUserFuncInvoked)
		require.False(t, ds.GetHostManagedAppleIDFuncInvoked)
	})

	// Regression test for #48879: a manual-profile BYOD host carries
	// host_mdm.is_personal_enrollment=1 but is a DEVICE-channel enrollment (its
	// primary nano_enrollments row is type "Device", not "User Enrollment
	// (Device)", and it has no Managed Apple ID). It must install device-scoped
	// (serialNumbers), exactly like company-owned manual — and must NOT attempt
	// user provisioning, which previously failed with errMissingManagedAppleID.
	t.Run("manual-profile BYOD (personal flag, device channel) routes via serialNumbers", func(t *testing.T) {
		var capt captured
		setupServer(t, &capt)

		// isUserEnrollment=false → the primary enrollment row is type "Device"
		// even though this host would have is_personal_enrollment=1 in host_mdm.
		ds := setupDS(t, false)
		// Make it explicit that the Managed Apple ID is absent for this host, so a
		// regression that re-introduces the user path would fail loudly here.
		ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
			return "", nil
		}
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)

		require.NotEmpty(t, capt.body)
		var got struct {
			ClientUserIds []string `json:"clientUserIds"`
			SerialNumbers []string `json:"serialNumbers"`
		}
		require.NoError(t, json.Unmarshal(capt.body, &got))
		require.Equal(t, []string{hostSerial}, got.SerialNumbers, "manual-profile BYOD must use serialNumbers (device-scoped)")
		require.Empty(t, got.ClientUserIds, "manual-profile BYOD must not send clientUserIds")

		// The whole point of #48879: no VPP user lookup/registration for device-channel BYOD.
		require.False(t, ds.GetHostManagedAppleIDFuncInvoked, "must not look up a Managed Apple ID for device-channel BYOD")
		require.False(t, ds.GetVPPClientUserFuncInvoked)
		require.False(t, ds.InsertVPPClientUserFuncInvoked)
	})

	t.Run("personal enrollment queries assignments by clientUserId", func(t *testing.T) {
		var assignmentsQuery string
		setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assignments"):
				assignmentsQuery = r.URL.RawQuery
				_, _ = w.Write([]byte(`{"assignments": []}`))
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets"):
				_, _ = fmt.Fprintf(w, `{"assets":[{"adamId":%q,"pricingParam":"STDQ","availableCount":5}]}`, adamID)
			case r.Method == http.MethodPost && r.URL.Path == "/registerVPPUserSrv":
				handleRegisterUserV1(t, w, r)
			case r.Method == http.MethodPost && r.URL.Path == "/assets/associate":
				_, _ = w.Write([]byte(`{"eventId":"associate-evt"}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})

		ds := setupDS(t, true)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)

		require.Contains(t, assignmentsQuery, "clientUserId=")
		require.NotContains(t, assignmentsQuery, "serialNumber=", "personal-enrollment assignments query must not filter by serial")
	})

	t.Run("personal enrollment with existing assignment skips AssociateAssets", func(t *testing.T) {
		var associateCalls int
		setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assignments"):
				// User already has the asset — Apple returns a non-empty assignment list.
				_, _ = fmt.Fprintf(w, `{"assignments":[{"adamId":%q,"pricingParam":"STDQ"}]}`, adamID)
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets"):
				t.Errorf("/assets must not be queried when assignments already exist")
			case r.Method == http.MethodPost && r.URL.Path == "/registerVPPUserSrv":
				handleRegisterUserV1(t, w, r)
			case r.Method == http.MethodPost && r.URL.Path == "/assets/associate":
				associateCalls++
				_, _ = w.Write([]byte(`{"eventId":"associate-evt"}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})

		ds := setupDS(t, true)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		cmdUUID, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, cmdUUID, "the command must still be enqueued for the install row")
		require.Equal(t, 0, associateCalls, "AssociateAssets must not run when the user already has the asset")
		require.True(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	})

	t.Run("max-devices error surfaces a friendly user-facing message", func(t *testing.T) {
		setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assignments"):
				_, _ = w.Write([]byte(`{"assignments": []}`))
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets"):
				_, _ = fmt.Fprintf(w, `{"assets":[{"adamId":%q,"pricingParam":"STDQ","availableCount":5}]}`, adamID)
			case r.Method == http.MethodPost && r.URL.Path == "/registerVPPUserSrv":
				handleRegisterUserV1(t, w, r)
			case r.Method == http.MethodPost && r.URL.Path == "/assets/associate":
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errorInfo":{},"errorMessage":"User has reached the maximum number of devices for this license.","errorNumber":9622}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})

		ds := setupDS(t, true)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.Error(t, err)

		var bre *fleet.BadRequestError
		require.ErrorAs(t, err, &bre)
		require.Contains(t, bre.Message, "maximum number of devices")
		// Internal error preserves Apple's raw response for debugging.
		require.Error(t, bre.InternalErr)
		require.Contains(t, bre.InternalErr.Error(), "9622")
	})

	t.Run("personal enrollment surfaces associate error without retry", func(t *testing.T) {
		// An associate error bubbles up directly — Fleet registers the user
		// once up front and does not retry or re-register on failure.
		var (
			registerCalls  int
			associateCalls int
		)
		setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assignments"):
				_, _ = w.Write([]byte(`{"assignments": []}`))
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets"):
				_, _ = fmt.Fprintf(w, `{"assets":[{"adamId":%q,"pricingParam":"STDQ","availableCount":5}]}`, adamID)
			case r.Method == http.MethodPost && r.URL.Path == "/registerVPPUserSrv":
				registerCalls++
				handleRegisterUserV1(t, w, r)
			case r.Method == http.MethodPost && r.URL.Path == "/assets/associate":
				associateCalls++
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errorInfo":{},"errorMessage":"Unable to find the registered user.","errorNumber":9609}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})

		ds := setupDS(t, true)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "9609")
		require.Equal(t, 1, registerCalls, "only the initial ensureVPPClientUser register; no retry")
		require.Equal(t, 1, associateCalls, "associate is attempted exactly once")
	})

	// Pin the dev_mode override in scope until t.Cleanup runs — referenced by the
	// helper above, which already registers Cleanup, but make sure the variable is
	// not flagged as unused.
	_ = dev_mode.Env
}
