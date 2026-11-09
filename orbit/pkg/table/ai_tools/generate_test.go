package ai_tools

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func typeConstraints(cs ...table.Constraint) table.QueryContext {
	return table.QueryContext{Constraints: map[string]table.ConstraintList{
		"type": {Affinity: table.ColumnTypeText, Constraints: cs},
	}}
}

func eq(v string) table.Constraint {
	return table.Constraint{Operator: table.OperatorEquals, Expression: v}
}

func TestRequestedTypes(t *testing.T) {
	all := slices.Sorted(slices.Values(allTypes))
	for _, tc := range []struct {
		name string
		qc   table.QueryContext
		want []string
	}{
		{"unconstrained", table.QueryContext{}, all},
		{"equals", typeConstraints(eq("agents")), []string{"agents"}},
		{"several equals", typeConstraints(eq("agents"), eq("apps")), []string{"agents", "apps"}},
		{"invalid equals", typeConstraints(eq("nope")), nil},
		{"other operator only", typeConstraints(table.Constraint{Operator: table.OperatorLike, Expression: "a%"}), all},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := slices.Sorted(maps.Keys(requestedTypes(tc.qc)))
			require.Equal(t, tc.want, got)
		})
	}
}

type inputCalls struct{ homes, snaps, walks int }

// stubSharedInputs replaces the shared-input producers with counting stubs
// returning hs, and starts from an empty cache.
func stubSharedInputs(t *testing.T, hs []homes.Home) *inputCalls {
	t.Helper()
	c := &inputCalls{}
	enumerateHomes = func() []homes.Home { c.homes++; return hs }
	takeSnapshot = func(context.Context) *proc.Snapshot {
		c.snaps++
		return proc.NewSnapshot(map[int]proc.Process{}, nil)
	}
	walkHome = func(h string, _ []string) []fsutil.WalkedDir { c.walks++; return []fsutil.WalkedDir{{Path: h}} }
	t.Cleanup(func() {
		enumerateHomes, takeSnapshot, walkHome = homes.All, proc.Take, fsutil.WalkHome
		shared.clear()
	})
	shared.clear()
	return c
}

func sharedExpiry() time.Time {
	shared.mu.Lock()
	defer shared.mu.Unlock()
	return shared.expiry
}

// osquery generates the table once per value of a `type IN (...)` list; the
// calls after the first must reuse the home enumeration, snapshot and walks,
// without extending the window they are kept for.
func TestSharedInputsReusedAcrossCalls(t *testing.T) {
	c := stubSharedInputs(t, []homes.Home{{Dir: t.TempDir(), Username: "u"}})

	_, err := generate(t.Context(), typeConstraints(eq("agents")))
	require.NoError(t, err)
	expiry := sharedExpiry()
	require.False(t, expiry.IsZero())
	for _, typ := range []string{"apps", "mcp_server", "agent_instruction", "sockets"} {
		_, err := generate(t.Context(), typeConstraints(eq(typ)))
		require.NoError(t, err)
	}
	require.Equal(t, inputCalls{homes: 1, snaps: 1, walks: 1}, *c)
	require.Equal(t, expiry, sharedExpiry(), "reuse must not extend the window")

	// Past the window the next query starts over.
	shared.mu.Lock()
	shared.expiry = time.Now().Add(-time.Second)
	shared.mu.Unlock()
	_, err = generate(t.Context(), typeConstraints(eq("agents")))
	require.NoError(t, err)
	require.Equal(t, inputCalls{homes: 2, snaps: 2, walks: 2}, *c)
}

// Inputs the first call didn't need are computed by the call that does, and
// a sockets-only first call never enumerates homes.
func TestSharedInputsComputedOnDemand(t *testing.T) {
	c := stubSharedInputs(t, nil)

	_, err := generate(t.Context(), typeConstraints(eq("sockets")))
	require.NoError(t, err)
	require.Equal(t, inputCalls{snaps: 1}, *c)

	_, err = generate(t.Context(), typeConstraints(eq("browser_extension")))
	require.NoError(t, err)
	require.Equal(t, inputCalls{homes: 1, snaps: 1}, *c)
}

// A snapshot taken while the query was being cancelled may be partial, so it
// isn't kept for the calls that follow.
func TestSharedInputsDropSnapshotOfCancelledQuery(t *testing.T) {
	c := stubSharedInputs(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	takeSnapshot = func(context.Context) *proc.Snapshot {
		c.snaps++
		cancel()
		return proc.NewSnapshot(map[int]proc.Process{}, nil)
	}

	_, err := generate(ctx, typeConstraints(eq("sockets")))
	require.ErrorIs(t, err, context.Canceled)

	takeSnapshot = func(context.Context) *proc.Snapshot {
		c.snaps++
		return proc.NewSnapshot(map[int]proc.Process{}, nil)
	}
	_, err = generate(t.Context(), typeConstraints(eq("sockets")))
	require.NoError(t, err)
	require.Equal(t, 2, c.snaps, "the next query takes a fresh snapshot")
}
