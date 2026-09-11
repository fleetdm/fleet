package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testABMTokenID = uint(9)

// depVerifyEnv wires a service against a fake Apple DEP API. statusFor decides
// what Apple answers for a given serial: an empty string omits it from the
// response entirely.
type depVerifyEnv struct {
	svc      *Service
	ctx      context.Context
	ds       *mock.Store
	requests func() int
	// batchSizes reports how many serials each /devices request carried, in order.
	batchSizes func() []int
	// orgNames reports the Apple Business organizations the client authenticated as.
	orgNames func() []string
}

func newDEPVerifyEnv(t *testing.T, statusFor func(serial string) string, failDevices bool) *depVerifyEnv {
	t.Helper()

	var mu sync.Mutex
	var deviceRequests int
	var batchSizes []int
	var orgNames []string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session":
			_, err := w.Write([]byte(`{"auth_session_token": "yoo"}`))
			assert.NoError(t, err)
		case "/devices":
			mu.Lock()
			deviceRequests++
			mu.Unlock()

			if failDevices {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			var req struct {
				Devices []string `json:"devices"`
			}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))

			mu.Lock()
			batchSizes = append(batchSizes, len(req.Devices))
			mu.Unlock()

			devices := map[string]any{}
			for _, serial := range req.Devices {
				status := statusFor(serial)
				if status == "" {
					continue
				}
				devices[serial] = map[string]any{"serial_number": serial, "response_status": status}
			}
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"devices": devices}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	depStorage := &nanodep_mock.Storage{}
	depStorage.RetrieveAuthTokensFunc = func(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
		return &nanodep_client.OAuth1Tokens{}, nil
	}
	depStorage.RetrieveConfigFunc = func(_ context.Context, name string) (*nanodep_client.Config, error) {
		mu.Lock()
		orgNames = append(orgNames, name)
		mu.Unlock()
		return &nanodep_client.Config{BaseURL: ts.URL}, nil
	}

	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{DEPStorage: depStorage})
	ctx = test.UserContext(ctx, test.UserAdmin)
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{}
		ac.MDM.AppleBMEnabledAndConfigured = true
		return ac, nil
	}
	ds.GetABMTokenByIDFunc = func(ctx context.Context, tokenID uint) (*fleet.ABMToken, error) {
		return &fleet.ABMToken{ID: tokenID, OrganizationName: "org"}, nil
	}
	// The DEP client's after-hook runs on every request to keep the token's
	// validity flags in sync.
	ds.SetABMTokenInvalidForOrgNameFunc = func(ctx context.Context, orgName string, invalid bool) (bool, error) {
		return false, nil
	}
	ds.IsABMTokenInvalidForOrgNameFunc = func(ctx context.Context, orgName string) (bool, error) {
		return false, nil
	}
	ds.CountABMTokensWithTermsExpiredFunc = func(ctx context.Context) (int, error) {
		return 0, nil
	}

	return &depVerifyEnv{
		svc: svc.(validationMiddleware).Service.(*Service),
		ctx: ctx,
		ds:  ds,
		requests: func() int {
			mu.Lock()
			defer mu.Unlock()
			return deviceRequests
		},
		batchSizes: func() []int {
			mu.Lock()
			defer mu.Unlock()
			return append([]int(nil), batchSizes...)
		},
		orgNames: func() []string {
			mu.Lock()
			defer mu.Unlock()
			return append([]string(nil), orgNames...)
		},
	}
}

func liveAssignment(hostID uint, serial string) *fleet.HostDEPAssignment {
	tok := testABMTokenID
	return &fleet.HostDEPAssignment{HostID: hostID, ABMTokenID: &tok, HardwareSerial: serial}
}

func TestCheckDEPAssignmentsForDelete(t *testing.T) {
	// Apple's answer for a serial decides whether the delete can stand. Only a
	// failure to answer blocks it.
	cases := []struct {
		name   string
		status string
		want   depDeleteCheck
	}{
		{"released from Apple Business", "NOT_ACCESSIBLE", depDeleteDisowned},
		{"still assigned", "SUCCESS", depDeleteAssigned},
		{"apple reports the lookup failed", "FAILED", depDeleteUnverified},
		{"apple omits the serial", "", depDeleteAssigned},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newDEPVerifyEnv(t, func(string) string { return tc.status }, false)
			env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
				return []*fleet.HostDEPAssignment{liveAssignment(1, "SERIAL1")}, nil
			}

			checks, err := env.svc.checkDEPAssignmentsForDelete(env.ctx, []*fleet.Host{{ID: 1, HardwareSerial: "SERIAL1"}})
			require.NoError(t, err)
			require.Equal(t, tc.want, checks[1].check)
		})
	}

	t.Run("apple unreachable blocks the delete", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(string) string { return "SUCCESS" }, true)
		env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
			return []*fleet.HostDEPAssignment{liveAssignment(1, "SERIAL1")}, nil
		}

		checks, err := env.svc.checkDEPAssignmentsForDelete(env.ctx, []*fleet.Host{{ID: 1, HardwareSerial: "SERIAL1"}})
		require.NoError(t, err, "an Apple failure is a verdict, not an error")
		require.Equal(t, depDeleteUnverified, checks[1].check)
	})

	t.Run("no live assignment means nothing to verify", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(string) string { return "SUCCESS" }, false)
		env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
			return nil, nil
		}

		checks, err := env.svc.checkDEPAssignmentsForDelete(env.ctx, []*fleet.Host{{ID: 1, HardwareSerial: "SERIAL1"}})
		require.NoError(t, err)
		require.Equal(t, depDeleteNoAssignment, checks[1].check)
		require.Zero(t, env.requests(), "Apple must not be called for a host that cannot be restored")
	})

	// Removing an Apple Business token from Fleet nulls out abm_token_id on every
	// assignment it owned (ON DELETE SET NULL). There is no organization left to
	// ask, and nothing can restore the host through a token that is gone, so the
	// record is cleared rather than the delete being blocked forever.
	t.Run("assignment whose Apple Business token was removed", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(string) string { return "SUCCESS" }, false)
		env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
			return []*fleet.HostDEPAssignment{{HostID: 1, ABMTokenID: nil, HardwareSerial: "SERIAL1"}}, nil
		}

		checks, err := env.svc.checkDEPAssignmentsForDelete(env.ctx, []*fleet.Host{{ID: 1, HardwareSerial: "SERIAL1"}})
		require.NoError(t, err)
		require.Equal(t, depDeleteDisowned, checks[1].check)
		require.Zero(t, env.requests(), "there is no organization to ask")
	})

	t.Run("free tier never calls apple", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(string) string { return "SUCCESS" }, false)
		env.ctx = license.NewContext(env.ctx, &fleet.LicenseInfo{Tier: fleet.TierFree})

		checks, err := env.svc.checkDEPAssignmentsForDelete(env.ctx, []*fleet.Host{{ID: 1, HardwareSerial: "SERIAL1"}})
		require.NoError(t, err)
		require.Equal(t, depDeleteNoAssignment, checks[1].check)
		require.Zero(t, env.requests())
		require.False(t, env.ds.GetHostDEPAssignmentsByHostIDsFuncInvoked)
	})

	t.Run("serials are chunked to Apple's per-request ceiling", func(t *testing.T) {
		total := apple_mdm.DEPSyncLimit + 50

		env := newDEPVerifyEnv(t, func(string) string { return "NOT_ACCESSIBLE" }, false)
		hosts := make([]*fleet.Host, 0, total)
		assignments := make([]*fleet.HostDEPAssignment, 0, total)
		for i := 1; i <= total; i++ {
			serial := fmt.Sprintf("SERIAL%d", i)
			hosts = append(hosts, &fleet.Host{ID: uint(i), HardwareSerial: serial}) //nolint:gosec // test data
			assignments = append(assignments, liveAssignment(uint(i), serial))      //nolint:gosec // test data
		}
		env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
			return assignments, nil
		}

		checks, err := env.svc.checkDEPAssignmentsForDelete(env.ctx, hosts)
		require.NoError(t, err)
		require.Len(t, checks, total)
		for _, h := range hosts {
			require.Equal(t, depDeleteDisowned, checks[h.ID].check, "host %d", h.ID)
		}
		require.Equal(t, []int{apple_mdm.DEPSyncLimit, 50}, env.batchSizes(),
			"serials must go out in batches, not one request per host")
	})

	t.Run("Apple Business not configured skips the check", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(string) string { return "SUCCESS" }, false)
		env.ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil // AppleBMEnabledAndConfigured false
		}

		checks, err := env.svc.checkDEPAssignmentsForDelete(env.ctx, []*fleet.Host{{ID: 1, HardwareSerial: "SERIAL1"}})
		require.NoError(t, err)
		require.Equal(t, depDeleteNoAssignment, checks[1].check)
		require.Zero(t, env.requests())
	})

	// Each Apple Business token authenticates as its own organization, so a serial
	// must only ever be asked about under the token that owns it. Asking the wrong
	// organization returns NOT_ACCESSIBLE, which would read as "released" and
	// delete a perfectly healthy host.
	t.Run("serials are asked under their own Apple Business token", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(serial string) string {
			if serial == "SERIAL-A" {
				return "NOT_ACCESSIBLE"
			}
			return "SUCCESS"
		}, false)

		tokenA, tokenB := uint(1), uint(2)
		env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
			return []*fleet.HostDEPAssignment{
				{HostID: 1, ABMTokenID: &tokenA, HardwareSerial: "SERIAL-A"},
				{HostID: 2, ABMTokenID: &tokenB, HardwareSerial: "SERIAL-B"},
			}, nil
		}
		env.ds.GetABMTokenByIDFunc = func(ctx context.Context, tokenID uint) (*fleet.ABMToken, error) {
			return &fleet.ABMToken{ID: tokenID, OrganizationName: fmt.Sprintf("org-%d", tokenID)}, nil
		}

		checks, err := env.svc.checkDEPAssignmentsForDelete(env.ctx, []*fleet.Host{
			{ID: 1, HardwareSerial: "SERIAL-A"},
			{ID: 2, HardwareSerial: "SERIAL-B"},
		})
		require.NoError(t, err)
		require.Equal(t, depDeleteDisowned, checks[1].check)
		require.Equal(t, depDeleteAssigned, checks[2].check)

		require.Equal(t, 2, env.requests(), "one request per organization")
		require.ElementsMatch(t, []string{"org-1", "org-2"}, env.orgNames())
		require.Equal(t, []int{1, 1}, env.batchSizes(), "a token's serials must not leak into another token's request")
	})
}

func TestDeleteHostVerifiesAppleBusinessAssignment(t *testing.T) {
	// Wires the delete through to the MDM lifecycle, which is where the decision
	// to recreate the host as a pending ADE host actually happens.
	setupHost := func(env *depVerifyEnv) *bool { //nolint:revive // pointer lets subtests read the flag
		var cleared bool

		env.ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return &fleet.Host{ID: id, HardwareSerial: "SERIAL1", Platform: "darwin"}, nil
		}
		env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
			return []*fleet.HostDEPAssignment{liveAssignment(1, "SERIAL1")}, nil
		}
		env.ds.DeleteHostFunc = func(ctx context.Context, hid uint) error { return nil }
		env.ds.MarkHostDEPAssignmentsDeletedFunc = func(ctx context.Context, hostIDs []uint) error {
			cleared = len(hostIDs) > 0
			return nil
		}
		// What the lifecycle reads to decide whether to restore the host.
		env.ds.GetHostDEPAssignmentFunc = func(ctx context.Context, hostID uint) (*fleet.HostDEPAssignment, error) {
			a := liveAssignment(hostID, "SERIAL1")
			if cleared {
				a.DeletedAt = new(time.Now())
			}
			return a, nil
		}
		env.ds.ReconcileDuplicateDEPHostOnDeleteFunc = func(ctx context.Context, serial, platform string, deletedHostID uint) (bool, error) {
			return false, nil
		}
		env.ds.RestoreMDMApplePendingDEPHostFunc = func(ctx context.Context, h *fleet.Host) error { return nil }
		env.ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) { return job, nil }

		return &cleared
	}

	t.Run("apple unreachable leaves the host in place", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(string) string { return "SUCCESS" }, true)
		setupHost(env)

		err := env.svc.DeleteHost(env.ctx, 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), fleet.CantDeleteHostUnverifiedABMMessage)
		require.False(t, env.ds.DeleteHostFuncInvoked, "nothing may be deleted when Apple can't be reached")

		// An upstream Apple failure is not the caller's fault, so it must not come
		// back as a 4xx telling them to fix their request.
		var gwErr *fleet.GatewayError
		require.ErrorAs(t, err, &gwErr)
		require.Equal(t, http.StatusBadGateway, gwErr.StatusCode())
	})

	t.Run("released host stays deleted", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(string) string { return "NOT_ACCESSIBLE" }, false)
		cleared := setupHost(env)

		require.NoError(t, env.svc.DeleteHost(env.ctx, 1))
		require.True(t, env.ds.DeleteHostFuncInvoked)
		require.True(t, *cleared, "the stale assignment must be marked deleted")
		// The point of clearing the assignment: the lifecycle no longer recreates
		// the host, so the "deleted" activity is not a lie.
		require.False(t, env.ds.RestoreMDMApplePendingDEPHostFuncInvoked)
	})

	t.Run("still-assigned host is restored as before", func(t *testing.T) {
		env := newDEPVerifyEnv(t, func(string) string { return "SUCCESS" }, false)
		cleared := setupHost(env)

		require.NoError(t, env.svc.DeleteHost(env.ctx, 1))
		require.True(t, env.ds.DeleteHostFuncInvoked)
		require.False(t, *cleared, "a device Apple still owns must keep its assignment")
		require.True(t, env.ds.RestoreMDMApplePendingDEPHostFuncInvoked, "a device Apple still owns must keep coming back")
	})
}

// A bulk delete must not be all-or-nothing: hosts Apple couldn't answer for are
// left alone and reported, while the rest of the batch still goes through.
func TestDeleteHostsSkipsHostsAppleCouldNotVerify(t *testing.T) {
	const (
		releasedHost   = uint(1) // Apple says it is not ours -> deleted for good
		assignedHost   = uint(2) // Apple says it is ours -> deleted, then restored
		unverifiedHost = uint(3) // Apple's lookup failed -> left in place
	)
	serials := map[uint]string{releasedHost: "SERIAL-REL", assignedHost: "SERIAL-ASG", unverifiedHost: "SERIAL-UNV"}

	env := newDEPVerifyEnv(t, func(serial string) string {
		switch serial {
		case "SERIAL-REL":
			return "NOT_ACCESSIBLE"
		case "SERIAL-UNV":
			return "FAILED"
		default:
			return "SUCCESS"
		}
	}, false)

	var cleared []string
	env.ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{ID: id, HardwareSerial: serials[id], Platform: "darwin"}, nil
	}
	env.ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
		hosts := make([]*fleet.Host, 0, len(ids))
		for _, id := range ids {
			hosts = append(hosts, &fleet.Host{ID: id, HardwareSerial: serials[id], Platform: "darwin"})
		}
		return hosts, nil
	}
	env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
		out := make([]*fleet.HostDEPAssignment, 0, len(ids))
		for _, id := range ids {
			out = append(out, liveAssignment(id, serials[id]))
		}
		return out, nil
	}
	env.ds.MarkHostDEPAssignmentsDeletedFunc = func(ctx context.Context, hostIDs []uint) error {
		for _, id := range hostIDs {
			cleared = append(cleared, serials[id])
		}
		return nil
	}
	env.ds.GetHostDEPAssignmentFunc = func(ctx context.Context, hostID uint) (*fleet.HostDEPAssignment, error) {
		a := liveAssignment(hostID, serials[hostID])
		for _, s := range cleared {
			if s == serials[hostID] {
				a.DeletedAt = new(time.Now())
			}
		}
		return a, nil
	}
	env.ds.ReconcileDuplicateDEPHostOnDeleteFunc = func(ctx context.Context, serial, platform string, deletedHostID uint) (bool, error) {
		return false, nil
	}
	env.ds.RestoreMDMApplePendingDEPHostFunc = func(ctx context.Context, h *fleet.Host) error { return nil }
	env.ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) { return job, nil }

	var deleted []uint
	env.ds.DeleteHostsFunc = func(ctx context.Context, ids []uint) error {
		deleted = append(deleted, ids...)
		return nil
	}

	err := env.svc.DeleteHosts(env.ctx, []uint{releasedHost, assignedHost, unverifiedHost}, nil)

	// The caller is told which hosts were left behind, by name.
	require.Error(t, err)
	require.Contains(t, err.Error(), fleet.CantDeleteHostUnverifiedABMMessage)
	require.Contains(t, err.Error(), serials[unverifiedHost])

	require.ElementsMatch(t, []uint{releasedHost, assignedHost}, deleted,
		"the rest of the batch must still be deleted")
	// A caller must be able to tell a partial delete from one that removed nothing,
	// so it knows whether retrying the whole request is safe.
	require.Contains(t, err.Error(), "The other 2 host(s) were deleted.")
	require.Equal(t, []string{serials[releasedHost]}, cleared,
		"only the released host's assignment may be cleared")
}

func TestClassifyDEPDeviceDetails(t *testing.T) {
	// Only Apple failing to answer blocks a delete. Silence about a serial is
	// unexpected but is read as "still ours", so one Apple quirk can't stop
	// deletions across a fleet.
	cases := []struct {
		name    string
		details *godep.DeviceDetails
		want    depDeleteCheck
	}{
		{"not accessible", &godep.DeviceDetails{ResponseStatus: "NOT_ACCESSIBLE"}, depDeleteDisowned},
		{"lookup failed", &godep.DeviceDetails{ResponseStatus: "FAILED"}, depDeleteUnverified},
		{"success", &godep.DeviceDetails{ResponseStatus: "SUCCESS"}, depDeleteAssigned},
		{"unrecognised status", &godep.DeviceDetails{ResponseStatus: "SOMETHING_NEW"}, depDeleteAssigned},
		{"serial not mentioned at all", nil, depDeleteAssigned},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, classifyDEPDeviceDetails(tc.details))
		})
	}
}

// When Apple can't be reached for any host in the batch there is nothing left to
// delete, so the caller gets the failure rather than a silent no-op.
func TestDeleteHostsWhenNoHostCanBeVerified(t *testing.T) {
	env := newDEPVerifyEnv(t, func(string) string { return "SUCCESS" }, true)

	serials := map[uint]string{1: "SERIAL-A", 2: "SERIAL-B"}
	env.ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{ID: id, HardwareSerial: serials[id], Platform: "darwin"}, nil
	}
	env.ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{
			{ID: 1, HardwareSerial: serials[1], Platform: "darwin"},
			{ID: 2, HardwareSerial: serials[2], Platform: "darwin"},
		}, nil
	}
	env.ds.GetHostDEPAssignmentsByHostIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.HostDEPAssignment, error) {
		return []*fleet.HostDEPAssignment{liveAssignment(1, serials[1]), liveAssignment(2, serials[2])}, nil
	}
	env.ds.MarkHostDEPAssignmentsDeletedFunc = func(ctx context.Context, hostIDs []uint) error { return nil }
	env.ds.DeleteHostsFunc = func(ctx context.Context, ids []uint) error { return nil }

	err := env.svc.DeleteHosts(env.ctx, []uint{1, 2}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), serials[1])
	require.Contains(t, err.Error(), serials[2])
	require.NotContains(t, err.Error(), "were deleted", "nothing was deleted, so the message must not imply otherwise")
	require.False(t, env.ds.DeleteHostsFuncInvoked, "nothing may be deleted when no host could be verified")

	var gwErr *fleet.GatewayError
	require.ErrorAs(t, err, &gwErr)
	require.Equal(t, http.StatusBadGateway, gwErr.StatusCode())
}
