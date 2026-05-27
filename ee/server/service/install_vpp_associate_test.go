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
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
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

	// Common datastore mock setup, parameterized by isPersonalEnrollment.
	setupDS := func(t *testing.T, isPersonal bool) *mock.Store {
		t.Helper()
		ds := new(mock.Store)
		ds.GetHostMDMFunc = func(_ context.Context, id uint) (*fleet.HostMDM, error) {
			require.Equal(t, hostID, id)
			return &fleet.HostMDM{HostID: id, Enrolled: true, IsPersonalEnrollment: isPersonal}, nil
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

	t.Run("non-personal enrollment keeps SerialNumbers", func(t *testing.T) {
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
		require.Equal(t, []string{hostSerial}, got.SerialNumbers, "manually-enrolled hosts must use serialNumbers")
		require.Empty(t, got.ClientUserIds, "manually-enrolled hosts must not send clientUserIds")

		// User-provisioning datastore writes must NOT happen on the manual path.
		require.False(t, ds.GetVPPClientUserFuncInvoked)
		require.False(t, ds.InsertVPPClientUserFuncInvoked)
		require.False(t, ds.GetHostManagedAppleIDFuncInvoked)
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

	t.Run("personal enrollment self-heals by recovering Apple-side user", func(t *testing.T) {
		// First /assets/associate call returns Apple's 9609 "unable to find
		// the registered user" error. Fleet should:
		//   - call GET /users?managedAppleId=... and find the existing
		//     (drifted) user,
		//   - upsert that clientUserId back into vpp_client_users,
		//   - retry /assets/associate with the recovered UUID and succeed.
		//
		// Critically, /registerVPPUserSrv should NOT be called a second time
		// — Apple already has a user, re-registering would hit 9635.
		const recoveredUUID = "recovered-uuid-from-apple"
		var (
			associateCalls            int
			registerCalls             int
			getUsersCalls             int
			capturedFirstClientUserID string
			capturedSecondClientUser  string
		)
		setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assignments"):
				_, _ = w.Write([]byte(`{"assignments": []}`))
			case r.Method == http.MethodGet && r.URL.Path == "/users":
				getUsersCalls++
				assert.Equal(t, "user@example.com", r.URL.Query().Get("managedAppleId"))
				_, _ = fmt.Fprintf(w, `{"users":[{"clientUserId":%q,"idHash":"hash","status":"Associated"}]}`, recoveredUUID)
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets"):
				_, _ = fmt.Fprintf(w, `{"assets":[{"adamId":%q,"pricingParam":"STDQ","availableCount":5}]}`, adamID)
			case r.Method == http.MethodPost && r.URL.Path == "/registerVPPUserSrv":
				registerCalls++
				handleRegisterUserV1(t, w, r)
			case r.Method == http.MethodPost && r.URL.Path == "/assets/associate":
				associateCalls++
				var got vpp.AssociateAssetsRequest
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
				assert.Len(t, got.ClientUserIds, 1)
				if len(got.ClientUserIds) == 0 {
					t.Errorf("associate request missing ClientUserIds")
					return
				}
				if associateCalls == 1 {
					capturedFirstClientUserID = got.ClientUserIds[0]
					// Real-world Apple 9609 signature observed in #31138.
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"errorInfo":{},"errorMessage":"Unable to find the registered user.","errorNumber":9609}`))
					return
				}
				capturedSecondClientUser = got.ClientUserIds[0]
				_, _ = w.Write([]byte(`{"eventId":"associate-evt-2"}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})

		// Override managed-apple-id so the GET /users assertion above can pin it.
		ds := setupDS(t, true)
		ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
			return "user@example.com", nil
		}
		var upserts []*fleet.VPPClientUser
		ds.InsertVPPClientUserFunc = func(_ context.Context, row *fleet.VPPClientUser) error {
			upserts = append(upserts, row)
			return nil
		}

		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}
		cmdUUID, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, cmdUUID)

		require.Equal(t, 2, associateCalls, "associate must be retried exactly once")
		require.Equal(t, 1, getUsersCalls, "must look up the user by managed apple id before deciding to re-register")
		require.Equal(t, 1, registerCalls, "only the initial ensureVPPClientUser register; no second register since Apple still has the user")

		require.NotEqual(t, capturedFirstClientUserID, capturedSecondClientUser, "second associate must use the recovered clientUserId")
		require.Equal(t, recoveredUUID, capturedSecondClientUser, "second associate must use the clientUserId returned by Apple's GET /users")

		require.Len(t, upserts, 2, "initial register + cache resync from Apple")
		require.Equal(t, recoveredUUID, upserts[1].ClientUserID)
		require.Equal(t, fleet.VPPClientUserStatusRegistered, upserts[1].Status)
		require.True(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	})

	t.Run("personal enrollment falls back to re-register when Apple has no user", func(t *testing.T) {
		// Same trigger (9609) but Apple's GET /users returns no active user
		// — the prior one was retired or the Apple ID was somehow purged.
		// Fleet should fall through to /registerVPPUserSrv, get a fresh UUID,
		// upsert it, and retry the associate.
		var (
			associateCalls int
			registerCalls  int
			getUsersCalls  int
		)
		setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assignments"):
				_, _ = w.Write([]byte(`{"assignments": []}`))
			case r.Method == http.MethodGet && r.URL.Path == "/users":
				getUsersCalls++
				// Empty (or Retired-only) response — caller must re-register.
				_, _ = w.Write([]byte(`{"users":[{"clientUserId":"retired-ghost","status":"Retired"}]}`))
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets"):
				_, _ = fmt.Fprintf(w, `{"assets":[{"adamId":%q,"pricingParam":"STDQ","availableCount":5}]}`, adamID)
			case r.Method == http.MethodPost && r.URL.Path == "/registerVPPUserSrv":
				registerCalls++
				handleRegisterUserV1(t, w, r)
			case r.Method == http.MethodPost && r.URL.Path == "/assets/associate":
				associateCalls++
				if associateCalls == 1 {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"errorInfo":{},"errorMessage":"Unable to find the registered user.","errorNumber":9609}`))
					return
				}
				_, _ = w.Write([]byte(`{"eventId":"associate-evt-fallback"}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})

		ds := setupDS(t, true)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)

		require.Equal(t, 1, getUsersCalls)
		require.Equal(t, 2, registerCalls, "initial ensure + fallback re-register")
		require.Equal(t, 2, associateCalls, "associate retried after re-register")
	})

	t.Run("personal enrollment does not self-heal on unrelated associate error", func(t *testing.T) {
		// Make sure a non-9612 associate error still bubbles up — we don't
		// want to mask the real error or churn through pointless re-registers.
		var registerCalls int
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
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errorInfo":{},"errorMessage":"Cannot establish a connection.","errorNumber":9610}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})

		ds := setupDS(t, true)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "9610")
		// Only the initial ensureVPPClientUser register — no self-heal retry.
		require.Equal(t, 1, registerCalls)
	})

	// Pin the dev_mode override in scope until t.Cleanup runs — referenced by the
	// helper above, which already registers Cleanup, but make sure the variable is
	// not flagged as unused.
	_ = dev_mode.Env
}
