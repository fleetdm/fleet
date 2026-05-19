package service

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Property-based tests for the cursor state machine in
// ReconcileWindowsProfiles. These cover invariants that emerge across many
// cron ticks (coverage, monotonicity, failure semantics) which the existing
// table-driven tests cannot exercise directly.
//
// Run with more checks:
//   go test -run TestPBT_ReconcileWindowsProfiles ./server/service/ -args -rapid.checks=2000

// cursorFakeState is the in-memory model the fake datastore exposes to the
// cron. It mirrors only what ReconcileWindowsProfiles depends on: a sorted
// set of host UUIDs with pending work, the persisted Redis-style cursor,
// and a per-host visit counter so tests can assert coverage.
type cursorFakeState struct {
	cursor    string
	pending   []string // sorted ascending; mirrors what ListNextPendingMDMWindowsHostUUIDs returns
	visited   map[string]int
	cursorSet int // number of times SetMDMWindowsReconcileCursor was called
}

// newCursorFakeDS wires a mock.Store with funcs that route to cursorFakeState
// and stubs every other DS method ReconcileWindowsProfiles calls so the body
// is a successful no-op end-to-end (no install/remove targets, no upserts).
// That keeps the property tests focused on the cursor protocol; the body's
// per-profile behavior is covered by the existing table-driven tests.
func newCursorFakeDS(initialHosts []string) (*mock.Store, *cursorFakeState) {
	sorted := slices.Clone(initialHosts)
	slices.Sort(sorted)
	state := &cursorFakeState{pending: sorted, visited: map[string]int{}}

	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		cfg := &fleet.AppConfig{}
		cfg.MDM.WindowsEnabledAndConfigured = true
		return cfg, nil
	}
	ds.GetMDMWindowsReconcileCursorFunc = func(ctx context.Context) (string, error) {
		return state.cursor, nil
	}
	ds.SetMDMWindowsReconcileCursorFunc = func(ctx context.Context, c string) error {
		state.cursor = c
		state.cursorSet++
		return nil
	}
	ds.ListNextPendingMDMWindowsHostUUIDsFunc = func(ctx context.Context, after string, batch int) ([]string, error) {
		var out []string
		for _, h := range state.pending {
			if h > after {
				out = append(out, h)
				if len(out) == batch {
					break
				}
			}
		}
		for _, h := range out {
			state.visited[h]++
		}
		return out, nil
	}
	// Empty install/remove/contents/existence: the cron's body sails
	// through without enqueueing any work, the deferred SetCursor still
	// fires, and the cursor protocol is exercised in isolation.
	ds.ListMDMWindowsProfilesToInstallForHostsFunc = func(ctx context.Context, hostUUIDs []string) ([]*fleet.MDMWindowsProfilePayload, error) {
		return nil, nil
	}
	ds.ListMDMWindowsProfilesToRemoveForHostsFunc = func(ctx context.Context, hostUUIDs []string) ([]*fleet.MDMWindowsProfilePayload, error) {
		return nil, nil
	}
	ds.GetMDMWindowsProfilesContentsFunc = func(ctx context.Context, uuids []string) (map[string]fleet.MDMWindowsProfileContents, error) {
		return map[string]fleet.MDMWindowsProfileContents{}, nil
	}
	ds.GetExistingMDMWindowsProfileUUIDsFunc = func(ctx context.Context, uuids []string) (map[string]struct{}, error) {
		return map[string]struct{}{}, nil
	}
	// The body unconditionally calls these two upserts at the end of every
	// tick (even with an empty payload). Stub them as no-ops.
	ds.BulkUpsertMDMWindowsHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
		return nil
	}
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		return nil
	}
	// GetGroupedCertificateAuthorities is called unconditionally on every
	// tick (even when there is no profile content to process). Return a
	// zero-value struct so the dependent maps are empty and the rest of
	// the body short-circuits.
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}
	return ds, state
}

// pbtBatchSizeOverride installs a property-test-scoped batch size on the
// package-level reconcileWindowsProfilesBatchSize var. The single
// t.Cleanup restores the original value once the whole test (across all
// rapid trials) finishes; per-trial reassignments inside the rapid.Check
// closure overwrite each other, which is the desired behavior.
func pbtBatchSizeOverride(t *testing.T) {
	t.Helper()
	saved := reconcileWindowsProfilesBatchSize
	t.Cleanup(func() { reconcileWindowsProfilesBatchSize = saved })
}

// hostGen draws short distinct strings; only their relative lexicographic
// order matters to the cursor protocol, not that they look like real UUIDs.
var hostGen = rapid.StringMatching(`[a-z]{1,8}`)

// pbtLogger discards everything
var pbtLogger = slog.New(slog.DiscardHandler)

// TestPBT_ReconcileWindowsProfilesCoverage verifies that for any stable
// population of pending hosts, the cursor protocol reaches a state where
// every host has been visited and the cursor has returned to "" within a
// bounded number of ticks. The bound is ⌈N/B⌉+2 to absorb the extra
// "empty pass after exact-multiple full pass" tick.
//
// We stop ticking as soon as that joint state is reached because any
// further tick restarts the pass (cursor moves back off ""), which would
// make a fixed-tick-count assertion brittle.
func TestPBT_ReconcileWindowsProfilesCoverage(t *testing.T) {
	pbtBatchSizeOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 25).Draw(rt, "batchSize")
		hosts := rapid.SliceOfNDistinct(hostGen, 0, 200, rapid.ID[string]).Draw(rt, "hosts")
		reconcileWindowsProfilesBatchSize = batch

		ds, state := newCursorFakeDS(hosts)
		ctx := t.Context()

		maxTicks := (len(hosts)+batch-1)/batch + 2

		passComplete := func() bool {
			if state.cursor != "" {
				return false
			}
			for _, h := range hosts {
				if state.visited[h] == 0 {
					return false
				}
			}
			return true
		}

		var completedAt int
		for tick := 1; tick <= maxTicks; tick++ {
			require.NoError(rt, ReconcileWindowsProfiles(ctx, ds, pbtLogger))
			if passComplete() {
				completedAt = tick
				break
			}
		}
		require.NotZerof(rt, completedAt,
			"pass did not complete within %d ticks (N=%d, B=%d, cursor=%q, visited=%v)",
			maxTicks, len(hosts), batch, state.cursor, state.visited)
	})
}

// TestPBT_ReconcileWindowsProfilesMonotonic verifies that within a pass the
// cursor strictly increases between two consecutive non-reset ticks. Reset
// transitions ("" -> non-empty starting fresh, or non-empty -> "" at end of
// pass) are allowed and expected.
func TestPBT_ReconcileWindowsProfilesMonotonic(t *testing.T) {
	pbtBatchSizeOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 20).Draw(rt, "batchSize")
		hosts := rapid.SliceOfNDistinct(hostGen, 1, 100, rapid.ID[string]).Draw(rt, "hosts")
		reconcileWindowsProfilesBatchSize = batch

		ds, state := newCursorFakeDS(hosts)
		ctx := t.Context()

		var prev string
		ticks := (len(hosts)+batch-1)/batch + 2
		for range ticks {
			require.NoError(rt, ReconcileWindowsProfiles(ctx, ds, pbtLogger))
			cur := state.cursor
			if prev != "" && cur != "" {
				require.Greaterf(rt, cur, prev,
					"cursor went backward within a pass: %q -> %q (N=%d, B=%d)",
					prev, cur, len(hosts), batch)
			}
			prev = cur
		}
	})
}

// TestPBT_ReconcileWindowsProfilesFailureNoAdvance verifies the universal
// invariant "any body failure leaves the cursor untouched." The cron's
// SetCursor write is gated by a named-return-aware defer that skips on
// error, plus pre-defer failure paths that simply return without
// registering the defer at all. The property must hold for both classes,
// and rapid randomly samples across them so a regression in either path
// surfaces here.
//
// Catches regressions like:
//   - Switching the named-return-aware defer to a bare `return ...`
//     mid-body (would advance on post-defer failure).
//   - Moving the defer registration earlier so a pre-listing failure no
//     longer skips it (would advance with a stale nextCursor).
//   - Adding a SetCursor call elsewhere in the body that fires before
//     the err check.
func TestPBT_ReconcileWindowsProfilesFailureNoAdvance(t *testing.T) {
	pbtBatchSizeOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 10).Draw(rt, "batchSize")
		// Need at least one host so the host listing returns non-empty
		// when it succeeds; otherwise the cron takes the empty-pop early
		// return and does not exercise the failure path we want.
		hosts := rapid.SliceOfNDistinct(hostGen, 1, 30, rapid.ID[string]).Draw(rt, "hosts")
		failurePoint := rapid.SampledFrom([]string{
			"ListNextPendingMDMWindowsHostUUIDs",      // pre-defer
			"ListMDMWindowsProfilesToInstallForHosts", // pre-defer
			"ListMDMWindowsProfilesToRemoveForHosts",  // pre-defer
			"GetMDMWindowsProfilesContents",           // post-defer
			"GetGroupedCertificateAuthorities",        // post-defer
			"BulkUpsertMDMWindowsHostProfiles",        // post-defer (end of body)
		}).Draw(rt, "failurePoint")
		reconcileWindowsProfilesBatchSize = batch

		ds, state := newCursorFakeDS(hosts)
		// Seed a non-empty cursor so "cursor untouched on failure" is a
		// real assertion rather than trivially-true on the empty default.
		// "0" sorts before any value hostGen can produce ([a-z]{1,8}), so
		// the host listing still returns every host and the cron reaches
		// the injected failure point.
		const initialCursor = "0"
		state.cursor = initialCursor
		simErr := errors.New("simulated failure at " + failurePoint)
		switch failurePoint {
		case "ListNextPendingMDMWindowsHostUUIDs":
			ds.ListNextPendingMDMWindowsHostUUIDsFunc = func(ctx context.Context, after string, b int) ([]string, error) {
				return nil, simErr
			}
		case "ListMDMWindowsProfilesToInstallForHosts":
			ds.ListMDMWindowsProfilesToInstallForHostsFunc = func(ctx context.Context, hostUUIDs []string) ([]*fleet.MDMWindowsProfilePayload, error) {
				return nil, simErr
			}
		case "ListMDMWindowsProfilesToRemoveForHosts":
			ds.ListMDMWindowsProfilesToRemoveForHostsFunc = func(ctx context.Context, hostUUIDs []string) ([]*fleet.MDMWindowsProfilePayload, error) {
				return nil, simErr
			}
		case "GetMDMWindowsProfilesContents":
			ds.GetMDMWindowsProfilesContentsFunc = func(ctx context.Context, uuids []string) (map[string]fleet.MDMWindowsProfileContents, error) {
				return nil, simErr
			}
		case "GetGroupedCertificateAuthorities":
			ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
				return nil, simErr
			}
		case "BulkUpsertMDMWindowsHostProfiles":
			ds.BulkUpsertMDMWindowsHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
				return simErr
			}
		}

		err := ReconcileWindowsProfiles(t.Context(), ds, pbtLogger)
		require.Errorf(rt, err, "failure at %q did not propagate", failurePoint)
		require.Equalf(rt, initialCursor, state.cursor,
			"cursor advanced despite failure at %q; got %q (expected %q)", failurePoint, state.cursor, initialCursor)
		require.Equalf(rt, 0, state.cursorSet,
			"SetMDMWindowsReconcileCursor was called %d times despite failure at %q",
			state.cursorSet, failurePoint)
	})
}

// TestPBT_ReconcileWindowsProfilesCursorAdvanceMatchesLastVisited verifies
// the per-tick cursor invariant: after a non-empty tick, either the cursor
// equals the lexicographically last host visited in that tick (full batch),
// or it is "" (short batch, signaling end of pass).
func TestPBT_ReconcileWindowsProfilesCursorAdvanceMatchesLastVisited(t *testing.T) {
	pbtBatchSizeOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 20).Draw(rt, "batchSize")
		hosts := rapid.SliceOfNDistinct(hostGen, 1, 100, rapid.ID[string]).Draw(rt, "hosts")
		reconcileWindowsProfilesBatchSize = batch
		sorted := slices.Clone(hosts)
		slices.Sort(sorted)

		ds, state := newCursorFakeDS(hosts)
		ctx := t.Context()

		// Walk the population by ticks; at each step, predict the batch
		// from sorted/cursor and check the resulting cursor matches the
		// rule.
		seen := 0
		for seen < len(sorted) {
			expectedBatch := sorted[seen:min(seen+batch, len(sorted))]
			require.NoError(rt, ReconcileWindowsProfiles(ctx, ds, pbtLogger))

			if len(expectedBatch) >= batch {
				require.Equalf(rt, expectedBatch[len(expectedBatch)-1], state.cursor,
					"full batch must leave cursor at last UUID; expected=%q got=%q",
					expectedBatch[len(expectedBatch)-1], state.cursor)
			} else {
				require.Emptyf(rt, state.cursor,
					"short batch must leave cursor empty; got=%q", state.cursor)
			}
			seen += len(expectedBatch)
		}
	})
}
