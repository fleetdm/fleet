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

	// Common mock-server setup. Captures the AssociateAssets body so the test
	// can assert on the wire payload.
	setupServer := func(t *testing.T, cap *captured) {
		t.Helper()
		setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer "+bearerToken, r.Header.Get("Authorization"))
			switch {
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assignments"):
				_, _ = w.Write([]byte(`{"assignments": []}`))
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets"):
				_, _ = fmt.Fprintf(w, `{"assets":[{"adamId":%q,"pricingParam":"STDQ","availableCount":5}]}`, adamID)
			case r.Method == http.MethodPost && r.URL.Path == "/users/create":
				body := struct {
					Users []struct {
						ClientUserId   string `json:"clientUserId"`
						ManagedAppleId string `json:"managedAppleId"`
					} `json:"users"`
				}{}
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Len(t, body.Users, 1)
				_, _ = fmt.Fprintf(w, `{"eventId":"evt","users":[{"userId":"apple-1","clientUserId":%q,"managedAppleId":%q,"status":"Registered"}]}`,
					body.Users[0].ClientUserId, body.Users[0].ManagedAppleId)
			case r.Method == http.MethodPost && r.URL.Path == "/assets/associate":
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				cap.body = b
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
		var cap captured
		setupServer(t, &cap)

		ds := setupDS(t, true)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)

		require.NotEmpty(t, cap.body, "AssociateAssets request must have been sent")
		var got struct {
			ClientUserIds []string `json:"clientUserIds"`
			SerialNumbers []string `json:"serialNumbers"`
		}
		require.NoError(t, json.Unmarshal(cap.body, &got))
		require.Len(t, got.ClientUserIds, 1)
		require.NotEmpty(t, got.ClientUserIds[0])
		require.Empty(t, got.SerialNumbers, "personal enrollment must not send serialNumbers")

		// The cached row was also persisted so subsequent calls reuse the same UUID.
		require.True(t, ds.InsertVPPClientUserFuncInvoked)
	})

	t.Run("non-personal enrollment keeps SerialNumbers", func(t *testing.T) {
		var cap captured
		setupServer(t, &cap)

		ds := setupDS(t, false)
		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		_, err := svc.InstallVPPAppPostValidation(context.Background(), host, vppApp, bearerToken, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)

		require.NotEmpty(t, cap.body)
		var got struct {
			ClientUserIds []string `json:"clientUserIds"`
			SerialNumbers []string `json:"serialNumbers"`
		}
		require.NoError(t, json.Unmarshal(cap.body, &got))
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
			case r.Method == http.MethodPost && r.URL.Path == "/users/create":
				body := struct {
					Users []struct {
						ClientUserId   string `json:"clientUserId"`
						ManagedAppleId string `json:"managedAppleId"`
					} `json:"users"`
				}{}
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				_, _ = fmt.Fprintf(w, `{"eventId":"evt","users":[{"userId":"apple-1","clientUserId":%q,"managedAppleId":%q,"status":"Registered"}]}`,
					body.Users[0].ClientUserId, body.Users[0].ManagedAppleId)
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
			case r.Method == http.MethodPost && r.URL.Path == "/users/create":
				body := struct {
					Users []struct {
						ClientUserId   string `json:"clientUserId"`
						ManagedAppleId string `json:"managedAppleId"`
					} `json:"users"`
				}{}
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				_, _ = fmt.Fprintf(w, `{"eventId":"evt","users":[{"userId":"apple-1","clientUserId":%q,"managedAppleId":%q,"status":"Registered"}]}`,
					body.Users[0].ClientUserId, body.Users[0].ManagedAppleId)
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
			case r.Method == http.MethodPost && r.URL.Path == "/users/create":
				body := struct {
					Users []struct {
						ClientUserId   string `json:"clientUserId"`
						ManagedAppleId string `json:"managedAppleId"`
					} `json:"users"`
				}{}
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				_, _ = fmt.Fprintf(w, `{"eventId":"evt","users":[{"userId":"apple-1","clientUserId":%q,"managedAppleId":%q,"status":"Registered"}]}`,
					body.Users[0].ClientUserId, body.Users[0].ManagedAppleId)
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

	// Pin the dev_mode override in scope until t.Cleanup runs — referenced by the
	// helper above, which already registers Cleanup, but make sure the variable is
	// not flagged as unused.
	_ = dev_mode.Env
}
