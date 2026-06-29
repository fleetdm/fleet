package service

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Property-based tests for the cursor state machine in ReconcileWindowsProfiles. These cover invariants that emerge across many
// cron ticks (coverage, monotonicity, failure semantics) which the existing table-driven tests cannot exercise directly.
//
// The reconciler walks every enrolled Windows host via GetWindowsProfileReconcileSnapshot and drains successive windows within a
// tick until a budget is hit. To pin the cursor protocol to a deterministic one-window-per-tick cadence, these tests set the
// delivery cap equal to the scan batch and make every host have exactly one pending install (one global, label-less profile +
// empty current state). The scan budget is set large so only the delivery cap governs.
//
// Run with more checks:
//   go test -run TestPBT_ReconcileWindowsProfiles ./server/service/ -args -rapid.checks=2000

// cursorFakeState is the in-memory model the fake datastore exposes to the cron. It mirrors only what ReconcileWindowsProfiles
// depends on: a sorted set of enrolled host UUIDs, the persisted Redis-style cursor, and a per-host visit counter so tests can
// assert coverage.
type cursorFakeState struct {
	cursor    string
	pending   []string // sorted ascending; the enrolled Windows host universe the snapshot pages through
	visited   map[string]int
	cursorSet int // number of times SetMDMWindowsReconcileCursor was called
}

// newCursorFakeDS wires a mock.Store with funcs that route to cursorFakeState and stubs every other DS method
// ReconcileWindowsProfiles calls so the body runs end-to-end without enqueueing real work. The snapshot returns the windowed
// hosts plus a single global profile so every host computes as one install; the existence pre-check then reports the profile
// gone, so execute short-circuits without commands. That keeps the property tests focused on the cursor protocol while still
// driving the delivery-cap accounting.
//
// COUPLING NOTE: the fake never actually enqueues, so the delivery cap is reached only because ReconcileWindowsProfiles counts
// intended work (pre-execute workHosts), not actual deliveries. If that accounting is ever changed to count only
// actually-scheduled hosts (the deferred CodeRabbit/Copilot review point), this fake would report zero delivered, the cap would
// never be reached, and the one-window-per-tick assumption these tests rely on (plus the coverage/monotonic/advance assertions)
// would break. The fake would then need to actually deliver work to keep driving the cap.
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
	ds.GetWindowsProfileReconcileSnapshotFunc = func(ctx context.Context, after string, batch int) (
		[]*fleet.WindowsHostReconcileInfo,
		[]*fleet.WindowsProfileForReconcile,
		map[uint]map[uint]struct{},
		map[string][]*fleet.MDMWindowsProfilePayload,
		error,
	) {
		var window []*fleet.WindowsHostReconcileInfo
		for i, h := range state.pending {
			if h > after {
				window = append(window, &fleet.WindowsHostReconcileInfo{HostID: uint(i + 1), UUID: h}) //nolint:gosec
				if len(window) == batch {
					break
				}
			}
		}
		for _, h := range window {
			state.visited[h.UUID]++
		}
		if len(window) == 0 {
			return nil, nil, nil, nil, nil
		}
		// One global, label-less profile so every host in the window computes as exactly one install (current state is empty), so
		// each host counts once against the per-tick delivery cap.
		profiles := []*fleet.WindowsProfileForReconcile{
			{ProfileUUID: "p-global", ProfileName: "Global", TeamID: 0, Checksum: []byte("c")},
		}
		return window, profiles, nil, map[string][]*fleet.MDMWindowsProfilePayload{}, nil
	}
	ds.GetMDMWindowsProfilesContentsFunc = func(ctx context.Context, uuids []string) (map[string]fleet.MDMWindowsProfileContents, error) {
		return map[string]fleet.MDMWindowsProfileContents{}, nil
	}
	// Report the profile as gone so the install loop short-circuits without enqueueing commands; the cursor protocol is unaffected.
	ds.GetExistingMDMWindowsProfileUUIDsFunc = func(ctx context.Context, uuids []string) (map[string]struct{}, error) {
		return map[string]struct{}{}, nil
	}
	// The body unconditionally calls these two upserts at the end of every window it executes (even with an empty payload). Stub them
	// as no-ops.
	ds.BulkUpsertMDMWindowsHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
		return nil
	}
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		return nil
	}
	// GetGroupedCertificateAuthorities is called whenever a window has work. Return a zero-value struct so the dependent maps are
	// empty and the rest of the body short-circuits.
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}
	return ds, state
}

// pbtBudgetOverride installs property-test-scoped budgets on the package-level reconciler tunables. The single t.Cleanup restores
// the original values once the whole test (across all rapid trials) finishes; per-trial reassignments inside the rapid.Check
// closure overwrite each other, which is the desired behavior. Tests set the delivery cap equal to the scan batch (one window per
// tick) and leave the scan budget large so only the cap governs.
func pbtBudgetOverride(t *testing.T) {
	t.Helper()
	savedBatch := reconcileWindowsProfilesBatchSize
	savedCap := reconcileWindowsProfilesDeliveryCap
	savedBudget := reconcileWindowsProfilesScanBudget
	t.Cleanup(func() {
		reconcileWindowsProfilesBatchSize = savedBatch
		reconcileWindowsProfilesDeliveryCap = savedCap
		reconcileWindowsProfilesScanBudget = savedBudget
	})
}

// pbtSetBudgets pins the per-trial budgets: scan batch == delivery cap (one window of delivered work per tick) and a large scan
// budget so wall clock never ends a tick early.
func pbtSetBudgets(batch int) {
	reconcileWindowsProfilesBatchSize = batch
	reconcileWindowsProfilesDeliveryCap = batch
	reconcileWindowsProfilesScanBudget = time.Hour
}

// hostGen draws short distinct strings; only their relative lexicographic order matters to the cursor protocol.
var hostGen = rapid.StringMatching(`[a-z]{1,8}`)

// pbtLogger discards everything
var pbtLogger = slog.New(slog.DiscardHandler)

// TestPBT_ReconcileWindowsProfilesCoverage verifies that for any stable population of enrolled hosts, the cursor protocol reaches
// a state where every host has been visited and the cursor has returned to "" within a bounded number of ticks. The bound is
// ⌈N/B⌉+2 to absorb the extra "empty pass after exact-multiple full pass" tick.
//
// We stop ticking as soon as that joint state is reached because any further tick restarts the pass (cursor moves back off ""),
// which would make a fixed-tick-count assertion brittle.
func TestPBT_ReconcileWindowsProfilesCoverage(t *testing.T) {
	pbtBudgetOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 25).Draw(rt, "batchSize")
		hosts := rapid.SliceOfNDistinct(hostGen, 0, 200, rapid.ID[string]).Draw(rt, "hosts")
		pbtSetBudgets(batch)

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

// TestPBT_ReconcileWindowsProfilesMonotonic verifies that within a pass the cursor strictly increases between two consecutive
// non-reset ticks. Reset transitions ("" -> non-empty starting fresh, or non-empty -> "" at end of pass) are allowed and
// expected.
func TestPBT_ReconcileWindowsProfilesMonotonic(t *testing.T) {
	pbtBudgetOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 20).Draw(rt, "batchSize")
		hosts := rapid.SliceOfNDistinct(hostGen, 1, 100, rapid.ID[string]).Draw(rt, "hosts")
		pbtSetBudgets(batch)

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

// TestPBT_ReconcileWindowsProfilesFailureNoAdvance verifies the universal invariant "any body failure leaves the cursor
// untouched." The cron's SetCursor write is gated by a named-return-aware defer that skips on error, and commitCursor only
// advances past a fully-delivered window, so a failure in the first window never persists a cursor. rapid randomly samples across
// pre-execute and in-execute failure points so a regression in either path surfaces here.
func TestPBT_ReconcileWindowsProfilesFailureNoAdvance(t *testing.T) {
	pbtBudgetOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 10).Draw(rt, "batchSize")
		// Need at least one host so the snapshot returns a non-empty window when it succeeds; otherwise the cron takes the empty-pop
		// early return and does not exercise the failure path we want.
		hosts := rapid.SliceOfNDistinct(hostGen, 1, 30, rapid.ID[string]).Draw(rt, "hosts")
		failurePoint := rapid.SampledFrom([]string{
			"GetWindowsProfileReconcileSnapshot", // pre-execute
			"GetMDMWindowsProfilesContents",      // in execute
			"GetGroupedCertificateAuthorities",   // in execute
			"BulkUpsertMDMWindowsHostProfiles",   // in execute (end of body)
		}).Draw(rt, "failurePoint")
		pbtSetBudgets(batch)

		ds, state := newCursorFakeDS(hosts)
		// Seed a non-empty cursor so "cursor untouched on failure" is a real assertion rather than trivially-true on the empty default.
		// "0" sorts before any value hostGen can produce ([a-z]{1,8}), so the snapshot still returns the first window and the cron
		// reaches the injected failure point.
		const initialCursor = "0"
		state.cursor = initialCursor
		simErr := errors.New("simulated failure at " + failurePoint)
		switch failurePoint {
		case "GetWindowsProfileReconcileSnapshot":
			ds.GetWindowsProfileReconcileSnapshotFunc = func(ctx context.Context, after string, batch int) (
				[]*fleet.WindowsHostReconcileInfo,
				[]*fleet.WindowsProfileForReconcile,
				map[uint]map[uint]struct{},
				map[string][]*fleet.MDMWindowsProfilePayload,
				error,
			) {
				return nil, nil, nil, nil, simErr
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

// TestPBT_ReconcileWindowsProfilesFailureNoAdvanceMultiWindow extends the no-advance-on-error invariant into the multi-window
// drain regime: with the delivery cap set high so one tick drains several windows, a failure on a LATER window (after earlier
// windows in the same tick already succeeded) must still leave the cursor untouched. This guards against a regression that
// persists per-window progress mid-tick (e.g. moving SetCursor inside the loop), which the single-window FailureNoAdvance test
// above cannot catch because there the failing window is always the first.
func TestPBT_ReconcileWindowsProfilesFailureNoAdvanceMultiWindow(t *testing.T) {
	pbtBudgetOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 5).Draw(rt, "batchSize")
		// At least 6 hosts (> max batch) guarantees the tick spans >= 2 windows.
		hosts := rapid.SliceOfNDistinct(hostGen, 6, 40, rapid.ID[string]).Draw(rt, "hosts")
		numWindows := (len(hosts) + batch - 1) / batch
		failWindow := rapid.IntRange(2, numWindows).Draw(rt, "failWindow")

		// Large cap and scan budget so neither ends the tick early; only the injected failure stops it, after failWindow-1 successful
		// windows.
		reconcileWindowsProfilesBatchSize = batch
		reconcileWindowsProfilesDeliveryCap = 1_000_000
		reconcileWindowsProfilesScanBudget = time.Hour

		ds, state := newCursorFakeDS(hosts)
		const initialCursor = "0"
		state.cursor = initialCursor

		// Every window has work, so GetMDMWindowsProfilesContents is called once per window. Fail the failWindow-th call; earlier windows
		// succeed.
		simErr := errors.New("simulated failure")
		contentsCalls := 0
		ds.GetMDMWindowsProfilesContentsFunc = func(ctx context.Context, uuids []string) (map[string]fleet.MDMWindowsProfileContents, error) {
			contentsCalls++
			if contentsCalls == failWindow {
				return nil, simErr
			}
			return map[string]fleet.MDMWindowsProfileContents{}, nil
		}

		err := ReconcileWindowsProfiles(t.Context(), ds, pbtLogger)
		require.Errorf(rt, err, "failure on window %d/%d did not propagate", failWindow, numWindows)
		// Confirms earlier windows really ran (so the precondition isn't vacuous).
		require.Equalf(rt, failWindow, contentsCalls,
			"expected to reach window %d before failing (N=%d, B=%d)", failWindow, len(hosts), batch)
		require.Equalf(rt, initialCursor, state.cursor,
			"cursor advanced despite failure on window %d after %d successful windows", failWindow, failWindow-1)
		require.Equalf(rt, 0, state.cursorSet,
			"SetMDMWindowsReconcileCursor called despite mid-tick failure on window %d", failWindow)
	})
}

// TestPBT_ReconcileWindowsProfilesCursorAdvanceMatchesLastVisited verifies the per-tick cursor invariant: with delivery cap ==
// scan batch and every host having work, each tick delivers exactly one window, so after a non-empty tick the cursor equals the
// lexicographically last host in that window (full batch), or it is "" (short batch, signaling end of pass).
func TestPBT_ReconcileWindowsProfilesCursorAdvanceMatchesLastVisited(t *testing.T) {
	pbtBudgetOverride(t)
	rapid.Check(t, func(rt *rapid.T) {
		batch := rapid.IntRange(1, 20).Draw(rt, "batchSize")
		hosts := rapid.SliceOfNDistinct(hostGen, 1, 100, rapid.ID[string]).Draw(rt, "hosts")
		pbtSetBudgets(batch)
		sorted := slices.Clone(hosts)
		slices.Sort(sorted)

		ds, state := newCursorFakeDS(hosts)
		ctx := t.Context()

		// Walk the population by ticks; at each step, predict the window from sorted/cursor and check the resulting cursor matches the
		// rule.
		seen := 0
		for seen < len(sorted) {
			expectedWindow := sorted[seen:min(seen+batch, len(sorted))]
			require.NoError(rt, ReconcileWindowsProfiles(ctx, ds, pbtLogger))

			if len(expectedWindow) >= batch {
				require.Equalf(rt, expectedWindow[len(expectedWindow)-1], state.cursor,
					"full window must leave cursor at last UUID; expected=%q got=%q",
					expectedWindow[len(expectedWindow)-1], state.cursor)
			} else {
				require.Emptyf(rt, state.cursor,
					"short window must leave cursor empty; got=%q", state.cursor)
			}
			seen += len(expectedWindow)
		}
	})
}
