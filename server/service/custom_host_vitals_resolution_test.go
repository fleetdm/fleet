package service

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// fakeExpandCustomHostVitals mimics the datastore's ExpandCustomHostVitals
// behavior (missing/empty value -> MissingCustomHostVitalValueError) for the given
// per-host value map, so service-layer tests don't need a real DB.
func fakeExpandCustomHostVitals(valueByID map[uint]string) func(context.Context, uint, string) (string, error) {
	return func(_ context.Context, _ uint, document string) (string, error) {
		refIDs := fleet.FindCustomHostVitalIDs(document)
		if len(refIDs) == 0 {
			return document, nil
		}
		var missing []uint
		for _, id := range refIDs {
			if v, ok := valueByID[id]; !ok || v == "" {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			return "", &fleet.MissingCustomHostVitalValueError{MissingIDs: missing}
		}
		expanded := fleet.MaybeExpand(document, func(s string, _, _ int) (string, bool) {
			if !strings.HasPrefix(s, fleet.CustomHostVitalPrefix) {
				return "", false
			}
			id, err := strconv.ParseUint(strings.TrimPrefix(s, fleet.CustomHostVitalPrefix), 10, 64)
			if err != nil {
				return "", false
			}
			v, ok := valueByID[uint(id)]
			return v, ok
		})
		return expanded, nil
	}
}

func TestGetHostScriptExpandsCustomHostVitals(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	host := &fleet.Host{ID: 42, UUID: "host-uuid-42", OrbitNodeKey: new("nk")}

	ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, doc string) (string, error) {
		return doc, nil
	}

	t.Run("substitutes the host's value", func(t *testing.T) {
		ds.GetHostScriptExecutionResultFunc = func(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
			return &fleet.HostScriptResult{HostID: host.ID, ExecutionID: execID, ScriptContents: "echo $FLEET_HOST_VITAL_7"}, nil
		}
		ds.ExpandCustomHostVitalsFunc = fakeExpandCustomHostVitals(map[uint]string{7: "engineering"})

		hctx := hostctx.NewContext(ctx, host)
		res, err := svc.GetHostScript(hctx, "exec-1")
		require.NoError(t, err)
		require.Equal(t, "echo engineering", res.ScriptContents)
	})

	t.Run("empty/missing value fails the script fetch", func(t *testing.T) {
		ds.GetHostScriptExecutionResultFunc = func(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
			return &fleet.HostScriptResult{HostID: host.ID, ExecutionID: execID, ScriptContents: "echo $FLEET_HOST_VITAL_9"}, nil
		}
		// host 42 has no value for vital 9
		ds.ExpandCustomHostVitalsFunc = fakeExpandCustomHostVitals(map[uint]string{7: "engineering"})

		hctx := hostctx.NewContext(ctx, host)
		_, err := svc.GetHostScript(hctx, "exec-2")
		require.Error(t, err)
		var missing *fleet.MissingCustomHostVitalValueError
		require.ErrorAs(t, err, &missing)
		require.Equal(t, []uint{9}, missing.MissingIDs)
	})
}

func TestCreateScriptValidatesCustomHostVitals(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error { return nil }

	// Simulate the real datastore: unknown ids -> MissingCustomHostVitalsError.
	ds.ValidateReferencedCustomHostVitalsFunc = func(ctx context.Context, documents []string) error {
		want := map[uint]struct{}{}
		for _, d := range documents {
			for _, id := range fleet.FindCustomHostVitalIDs(d) {
				want[id] = struct{}{}
			}
		}
		// only id 1 exists
		var missing []uint
		for id := range want {
			if id != 1 {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			return &fleet.MissingCustomHostVitalsError{MissingIDs: missing}
		}
		return nil
	}

	// Unknown id (999) should be rejected.
	_, err := svc.NewScript(ctx, nil, "myscript.sh", strings.NewReader("#!/bin/sh\necho $FLEET_HOST_VITAL_999\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "FLEET_HOST_VITAL_999")

	// ValidateReferencedCustomHostVitals must actually have been called.
	require.True(t, ds.ValidateReferencedCustomHostVitalsFuncInvoked)
}
