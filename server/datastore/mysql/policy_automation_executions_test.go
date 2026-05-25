package mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// policyRunRow holds the column values we care about when asserting the
// post-transition state of a (policy, host) row in policy_runs.
type policyRunRow struct {
	ID                  uint
	OldStatus           *bool
	NewStatus           bool
	ConsecutiveFailures uint
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func readPolicyRun(t *testing.T, ds *Datastore, ctx context.Context, policyID, hostID uint) (policyRunRow, bool) {
	t.Helper()
	var row policyRunRow
	err := ds.writer(ctx).QueryRowxContext(ctx,
		`SELECT id, old_status, new_status, consecutive_failures, created_at, updated_at
		   FROM policy_runs WHERE policy_id = ? AND host_id = ?`,
		policyID, hostID,
	).Scan(&row.ID, &row.OldStatus, &row.NewStatus, &row.ConsecutiveFailures, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return policyRunRow{}, false
	}
	return row, true
}

func countPolicyRuns(t *testing.T, ds *Datastore, ctx context.Context, policyID, hostID uint) int {
	t.Helper()
	var n int
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &n,
		`SELECT COUNT(*) FROM policy_runs WHERE policy_id = ? AND host_id = ?`, policyID, hostID))
	return n
}

func TestRecordPolicyTransitions(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := t.Context()

	user := test.NewUser(t, ds, "Test", "test@example.com", true)
	policy := newTestPolicy(t, ds, user, "p1", "darwin", nil)

	hostSeq := 0
	newHost := func(name string) *fleet.Host {
		hostSeq++
		return test.NewHost(t, ds, name, fmt.Sprintf("10.1.0.%d", hostSeq), "key-"+name, "uuid-"+name, time.Now())
	}

	t.Run("empty policyResults is a no-op", func(t *testing.T) {
		host := newHost("empty")
		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID, nil, nil, nil)
		require.NoError(t, err)
		require.Empty(t, runIDs)
		require.Equal(t, 0, countPolicyRuns(t, ds, ctx, policy.ID, host.ID))
	})

	t.Run("nil-valued policyResults entry is skipped", func(t *testing.T) {
		host := newHost("nilvalue")
		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: nil}, nil, nil)
		require.NoError(t, err)
		require.Empty(t, runIDs)
		require.Equal(t, 0, countPolicyRuns(t, ds, ctx, policy.ID, host.ID))
	})

	t.Run("new policy fails first time: INSERT (NULL, false, 1)", func(t *testing.T) {
		host := newHost("firstFail")
		results := map[uint]*bool{policy.ID: new(false)}
		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID, results, []uint{policy.ID}, nil)
		require.NoError(t, err)
		require.Contains(t, runIDs, policy.ID)
		require.NotZero(t, runIDs[policy.ID])

		row, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.Nil(t, row.OldStatus, "old_status must be NULL on first-time failing")
		require.False(t, row.NewStatus)
		require.Equal(t, uint(1), row.ConsecutiveFailures)
	})

	// seedPassingRow creates a policy_runs row with new_status=true by running
	// the fail→pass flip — the only production path that creates a passing
	// row under Option A (first-time-passing takes the all-passing fast path).
	seedPassingRow := func(h *fleet.Host) policyRunRow {
		t.Helper()
		_, err := ds.RecordPolicyTransitions(ctx, h.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		_, err = ds.RecordPolicyTransitions(ctx, h.ID, map[uint]*bool{policy.ID: new(true)}, nil, []uint{policy.ID})
		require.NoError(t, err)
		row, ok := readPolicyRun(t, ds, ctx, policy.ID, h.ID)
		require.True(t, ok)
		require.True(t, row.NewStatus)
		return row
	}

	t.Run("was passing, now failing: UPDATE (true, false, 1)", func(t *testing.T) {
		host := newHost("passToFail")
		passingRow := seedPassingRow(host)

		// Flip to failing.
		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		require.Contains(t, runIDs, policy.ID)

		row, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.Equal(t, passingRow.ID, row.ID, "transition is an in-place UPDATE, row id is preserved")
		require.NotNil(t, row.OldStatus)
		require.True(t, *row.OldStatus, "old_status must record the previous (passing) state")
		require.False(t, row.NewStatus)
		require.Equal(t, uint(1), row.ConsecutiveFailures)
		require.Equal(t, 1, countPolicyRuns(t, ds, ctx, policy.ID, host.ID), "still only one row per (policy, host)")
	})

	t.Run("was failing, now passing: UPDATE (false, true, 0)", func(t *testing.T) {
		host := newHost("failToPass")

		// Seed with first-time fail.
		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		failingRow, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)

		// Flip to passing — newPassing carries the flip, disabling the fast path.
		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(true)}, nil, []uint{policy.ID})
		require.NoError(t, err)
		require.Empty(t, runIDs)

		row, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.Equal(t, failingRow.ID, row.ID, "transition is an in-place UPDATE")
		require.NotNil(t, row.OldStatus)
		require.False(t, *row.OldStatus, "old_status must record the previous (failing) state")
		require.True(t, row.NewStatus)
		require.Equal(t, uint(0), row.ConsecutiveFailures, "consecutive_failures resets on recovery")
		require.Equal(t, 1, countPolicyRuns(t, ds, ctx, policy.ID, host.ID))
	})

	t.Run("was passing, still passing: row untouched (fast path)", func(t *testing.T) {
		host := newHost("stillPass")
		first := seedPassingRow(host)

		// Sleep so updated_at would visibly advance if any write occurred.
		time.Sleep(1100 * time.Millisecond)

		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(true)}, nil, nil)
		require.NoError(t, err)
		require.Empty(t, runIDs)

		after, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.True(t, after.UpdatedAt.Equal(first.UpdatedAt), "fast path must not touch updated_at")
		require.Equal(t, first.ConsecutiveFailures, after.ConsecutiveFailures)
	})

	t.Run("was failing, still failing: bump consecutive_failures", func(t *testing.T) {
		host := newHost("stillFail")

		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		first, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.Equal(t, uint(1), first.ConsecutiveFailures)

		// Sleep so updated_at advances visibly.
		time.Sleep(1100 * time.Millisecond)

		// FlippingPoliciesForHost won't return this as "newFailing" (it's not
		// a flip), so simulate the hot-path call shape: newFailing empty, only
		// policyResults carries the still-failing signal.
		_, err = ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, nil, nil)
		require.NoError(t, err)

		after, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.Equal(t, uint(2), after.ConsecutiveFailures, "consecutive_failures must increment on still-failing")
		require.True(t, after.CreatedAt.Equal(first.CreatedAt), "created_at must be preserved across bumps (first-failure timestamp)")
		require.True(t, after.UpdatedAt.After(first.UpdatedAt), "updated_at must advance on the bump")
	})

	t.Run("returned failingRunIDs cover all newFailing entries, no recoveries", func(t *testing.T) {
		host := newHost("returnMap")
		p2 := newTestPolicy(t, ds, user, "p2_return", "darwin", nil)
		p3 := newTestPolicy(t, ds, user, "p3_return", "darwin", nil)

		// p2 already passing prior; p3 already failing prior — both will flip.
		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{p2.ID: new(true), p3.ID: new(false)}, nil, nil)
		require.NoError(t, err)

		// Now flip both: p2 → fail, p3 → pass. policy stays as a fresh first-time fail.
		results := map[uint]*bool{policy.ID: new(false), p2.ID: new(false), p3.ID: new(true)}
		newFailing := []uint{policy.ID, p2.ID}
		newPassing := []uint{p3.ID}
		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID, results, newFailing, newPassing)
		require.NoError(t, err)
		require.Contains(t, runIDs, policy.ID)
		require.Contains(t, runIDs, p2.ID)
		require.NotContains(t, runIDs, p3.ID, "recovered policy must not appear in failingRunIDs")
		require.NotZero(t, runIDs[policy.ID])
		require.NotZero(t, runIDs[p2.ID])
	})

	t.Run("repeated re-failures keep created_at anchored to the first failure", func(t *testing.T) {
		host := newHost("createdAnchor")

		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		first, _ := readPolicyRun(t, ds, ctx, policy.ID, host.ID)

		time.Sleep(1100 * time.Millisecond)
		for range 3 {
			_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, nil, nil)
			require.NoError(t, err)
		}

		after, _ := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, after.CreatedAt.Equal(first.CreatedAt))
		require.Equal(t, uint(4), after.ConsecutiveFailures)
	})

	t.Run("multiple policies on the same host dispatch each case independently", func(t *testing.T) {
		host := newHost("multi")
		pFirstFail := newTestPolicy(t, ds, user, "multi_firstFail", "darwin", nil)
		pPassToFail := newTestPolicy(t, ds, user, "multi_passToFail", "darwin", nil)
		pStillFail := newTestPolicy(t, ds, user, "multi_stillFail", "darwin", nil)
		pFailToPass := newTestPolicy(t, ds, user, "multi_failToPass", "darwin", nil)
		pStillPass := newTestPolicy(t, ds, user, "multi_stillPass", "darwin", nil)

		// Seed prior state.
		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{
			pPassToFail.ID: new(true),
			pStillFail.ID:  new(false),
			pFailToPass.ID: new(false),
			pStillPass.ID:  new(true),
		}, []uint{pStillFail.ID, pFailToPass.ID}, nil)
		require.NoError(t, err)

		// Now apply a check-in that exercises all five active cases at once.
		results := map[uint]*bool{
			pFirstFail.ID:  new(false), // first-time fail
			pPassToFail.ID: new(false), // flip pass→fail
			pStillFail.ID:  new(false), // still failing
			pFailToPass.ID: new(true),  // flip fail→pass
			pStillPass.ID:  new(true),  // still passing
		}
		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID, results,
			[]uint{pFirstFail.ID, pPassToFail.ID}, []uint{pFailToPass.ID})
		require.NoError(t, err)
		require.Contains(t, runIDs, pFirstFail.ID)
		require.Contains(t, runIDs, pPassToFail.ID)
		require.NotContains(t, runIDs, pStillFail.ID, "still-failing is not in newFailing")
		require.NotContains(t, runIDs, pFailToPass.ID)
		require.NotContains(t, runIDs, pStillPass.ID)

		// Per-policy assertions.
		ff, _ := readPolicyRun(t, ds, ctx, pFirstFail.ID, host.ID)
		require.Nil(t, ff.OldStatus)
		require.False(t, ff.NewStatus)
		require.Equal(t, uint(1), ff.ConsecutiveFailures)

		ptf, _ := readPolicyRun(t, ds, ctx, pPassToFail.ID, host.ID)
		require.NotNil(t, ptf.OldStatus)
		require.True(t, *ptf.OldStatus)
		require.False(t, ptf.NewStatus)
		require.Equal(t, uint(1), ptf.ConsecutiveFailures)

		sf, _ := readPolicyRun(t, ds, ctx, pStillFail.ID, host.ID)
		require.False(t, sf.NewStatus)
		require.Equal(t, uint(2), sf.ConsecutiveFailures, "bumped from 1 (seed) to 2")

		ftp, _ := readPolicyRun(t, ds, ctx, pFailToPass.ID, host.ID)
		require.NotNil(t, ftp.OldStatus)
		require.False(t, *ftp.OldStatus)
		require.True(t, ftp.NewStatus)
		require.Equal(t, uint(0), ftp.ConsecutiveFailures)

		sp, _ := readPolicyRun(t, ds, ctx, pStillPass.ID, host.ID)
		require.Nil(t, sp.OldStatus, "still-passing remains untouched, old_status from first-time insert stays NULL")
		require.True(t, sp.NewStatus)
		require.Equal(t, uint(0), sp.ConsecutiveFailures)
	})

	t.Run("hosts are isolated", func(t *testing.T) {
		hostA := newHost("isoA")
		hostB := newHost("isoB")
		_, err := ds.RecordPolicyTransitions(ctx, hostA.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)

		// Recording for hostA must not affect hostB.
		_, hasB := readPolicyRun(t, ds, ctx, policy.ID, hostB.ID)
		require.False(t, hasB)
	})

	t.Run("fast path: all-passing + no transitions skips the DB write entirely", func(t *testing.T) {
		// Stable-fleet hot-path optimization: when every observed policy is
		// currently passing AND no transitions were reported, the function
		// returns immediately without writing. First-time-passing rows are
		// intentionally not created — the UI's COALESCE fallback to
		// policy_membership.created_at covers that case.
		host := newHost("fastpath")
		p2 := newTestPolicy(t, ds, user, "fastpath_p2", "darwin", nil)

		runIDs, err := ds.RecordPolicyTransitions(ctx, host.ID,
			map[uint]*bool{policy.ID: new(true), p2.ID: new(true)},
			nil, nil)
		require.NoError(t, err)
		require.Empty(t, runIDs)
		require.Equal(t, 0, countPolicyRuns(t, ds, ctx, policy.ID, host.ID),
			"no row should have been created — the fast path returned before issuing the ODKU")
		require.Equal(t, 0, countPolicyRuns(t, ds, ctx, p2.ID, host.ID))
	})

	t.Run("fast path does NOT trigger when any policy is currently failing", func(t *testing.T) {
		// A single failing policy in policyResults disables the fast path —
		// otherwise still-failing bumps would silently be lost.
		host := newHost("fastpath_mixed")
		// Seed a failing prior so the next call would bump consecutive_failures.
		_, err := ds.RecordPolicyTransitions(ctx, host.ID,
			map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		first, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.Equal(t, uint(1), first.ConsecutiveFailures)

		// Mixed results with one still-failing policy: fast path must NOT
		// fire, the bump must happen.
		_, err = ds.RecordPolicyTransitions(ctx, host.ID,
			map[uint]*bool{policy.ID: new(false)}, nil, nil)
		require.NoError(t, err)
		after, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.Equal(t, uint(2), after.ConsecutiveFailures)
	})

	t.Run("concurrent callers for the same (host, policy) collapse to one row, no errors, sum of consecutive_failures", func(t *testing.T) {
		// The read-modify-write that previously implemented the case dispatch
		// could race two callers into a unique-key violation on first-time
		// failure. The current ODKU statement is one round-trip per chunk, so
		// InnoDB serializes the writes on the row lock and every call
		// completes successfully — the still-failing branch handles the
		// already-inserted row.
		host := newHost("concurrent")
		const goroutines = 16

		var wg sync.WaitGroup
		errCh := make(chan error, goroutines)
		for range goroutines {
			wg.Go(func() {
				_, err := ds.RecordPolicyTransitions(ctx, host.ID,
					map[uint]*bool{policy.ID: new(false)},
					[]uint{policy.ID}, nil)
				if err != nil {
					errCh <- err
				}
			})
		}
		wg.Wait()
		close(errCh)
		for err := range errCh {
			require.NoError(t, err, "no caller should see a unique-key violation")
		}

		require.Equal(t, 1, countPolicyRuns(t, ds, ctx, policy.ID, host.ID), "exactly one row per (policy, host)")
		row, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		require.False(t, row.NewStatus)
		require.Nil(t, row.OldStatus, "first-time insert wins, old_status stays NULL")
		require.Equal(t, uint(goroutines), row.ConsecutiveFailures,
			"each caller after the first bumps consecutive_failures via still-failing branch")
	})
}

func TestGetFailingPolicyRuns(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := t.Context()

	user := test.NewUser(t, ds, "Test", "test@example.com", true)
	policy := newTestPolicy(t, ds, user, "gp1", "darwin", nil)
	policy2 := newTestPolicy(t, ds, user, "gp2", "darwin", nil)

	mkHost := func(name string) *fleet.Host {
		return test.NewHost(t, ds, name, name+".ip", "key-"+name, "uuid-"+name, time.Now())
	}

	t.Run("returns the failure row for each (policy, host) pair that has one", func(t *testing.T) {
		hostA := mkHost("getA")
		hostB := mkHost("getB")
		hostNoFail := mkHost("getNoFail")

		_, err := ds.RecordPolicyTransitions(ctx, hostA.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		_, err = ds.RecordPolicyTransitions(ctx, hostB.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)

		refs, err := ds.GetFailingPolicyRuns(ctx, []uint{policy.ID}, []uint{hostA.ID, hostB.ID, hostNoFail.ID})
		require.NoError(t, err)

		got := make(map[uint]uint, len(refs))
		for _, r := range refs {
			require.Equal(t, policy.ID, r.PolicyID)
			require.NotZero(t, r.RunID)
			got[r.HostID] = r.RunID
		}
		require.Contains(t, got, hostA.ID)
		require.Contains(t, got, hostB.ID)
		require.NotContains(t, got, hostNoFail.ID)
	})

	t.Run("passing rows are excluded", func(t *testing.T) {
		hostPassing := mkHost("getPassOnly")
		_, err := ds.RecordPolicyTransitions(ctx, hostPassing.ID, map[uint]*bool{policy.ID: new(true)}, nil, nil)
		require.NoError(t, err)

		refs, err := ds.GetFailingPolicyRuns(ctx, []uint{policy.ID}, []uint{hostPassing.ID})
		require.NoError(t, err)
		require.Empty(t, refs, "rows with new_status=true must not be returned")
	})

	t.Run("a row that recovered from fail to pass is no longer returned", func(t *testing.T) {
		host := mkHost("getRecovered")
		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		// newPassing carries the fail→pass flip — disables the all-passing fast path.
		_, err = ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(true)}, nil, []uint{policy.ID})
		require.NoError(t, err)

		refs, err := ds.GetFailingPolicyRuns(ctx, []uint{policy.ID}, []uint{host.ID})
		require.NoError(t, err)
		require.Empty(t, refs, "after fail→pass UPDATE the row's new_status is true and must be filtered out")
	})

	t.Run("cross-product (N policies x N hosts) returns one row per failing pair", func(t *testing.T) {
		hostX := mkHost("getCrossX")
		hostY := mkHost("getCrossY")

		// hostX fails on policy and policy2; hostY only fails on policy2.
		_, err := ds.RecordPolicyTransitions(ctx, hostX.ID,
			map[uint]*bool{policy.ID: new(false), policy2.ID: new(false)},
			[]uint{policy.ID, policy2.ID}, nil)
		require.NoError(t, err)
		_, err = ds.RecordPolicyTransitions(ctx, hostY.ID,
			map[uint]*bool{policy2.ID: new(false)},
			[]uint{policy2.ID}, nil)
		require.NoError(t, err)

		refs, err := ds.GetFailingPolicyRuns(ctx,
			[]uint{policy.ID, policy2.ID},
			[]uint{hostX.ID, hostY.ID})
		require.NoError(t, err)

		gotPairs := make(map[[2]uint]uint, len(refs))
		for _, r := range refs {
			gotPairs[[2]uint{r.PolicyID, r.HostID}] = r.RunID
		}
		require.Contains(t, gotPairs, [2]uint{policy.ID, hostX.ID})
		require.Contains(t, gotPairs, [2]uint{policy2.ID, hostX.ID})
		require.Contains(t, gotPairs, [2]uint{policy2.ID, hostY.ID})
		require.NotContains(t, gotPairs, [2]uint{policy.ID, hostY.ID})
	})

	t.Run("empty input returns no rows", func(t *testing.T) {
		refs, err := ds.GetFailingPolicyRuns(ctx, nil, []uint{1})
		require.NoError(t, err)
		require.Empty(t, refs)

		refs, err = ds.GetFailingPolicyRuns(ctx, []uint{1}, nil)
		require.NoError(t, err)
		require.Empty(t, refs)
	})
}

func TestCreatePolicyAutomationExecutions(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := t.Context()

	user := test.NewUser(t, ds, "Test", "test@example.com", true)
	policy := newTestPolicy(t, ds, user, "ce1", "darwin", nil)

	mkFailingRun := func(name string) fleet.PolicyRunRef {
		host := test.NewHost(t, ds, name, name+".ip", "key-"+name, "uuid-"+name, time.Now())
		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		row, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		return fleet.PolicyRunRef{PolicyID: policy.ID, HostID: host.ID, RunID: row.ID}
	}

	t.Run("empty input returns uuid.Nil and writes no rows", func(t *testing.T) {
		batchID, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, nil)
		require.NoError(t, err)
		require.Equal(t, uuid.Nil, batchID)
	})

	t.Run("creates one join row per execution and one batch row in pending status", func(t *testing.T) {
		run1 := mkFailingRun("ce_h1")
		run2 := mkFailingRun("ce_h2")
		executions := []fleet.PolicyRunRef{run1, run2}

		batchID, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, executions)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, batchID)

		// Two join-table rows, one per policy_run, both with the requested
		// automation_type and shared batch_id.
		var joinCount int
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &joinCount,
			`SELECT COUNT(*) FROM policy_runs_to_policy_automation_executions
			   WHERE batch_id = ? AND automation_type = ?`, batchID[:], fleet.PolicyAutomationWebhook))
		require.Equal(t, 2, joinCount)

		// Exactly one batch-status row, in 'pending'.
		var status string
		var errMsg *string
		require.NoError(t, ds.writer(ctx).QueryRowxContext(ctx,
			`SELECT status, error_message FROM policy_automation_executions WHERE batch_id = ?`, batchID[:]).
			Scan(&status, &errMsg))
		require.Equal(t, string(fleet.PolicyAutomationStatusPending), status)
		require.Nil(t, errMsg)
	})

	t.Run("different automation types produce independent batches for the same runs", func(t *testing.T) {
		run := mkFailingRun("ce_dual")
		executions := []fleet.PolicyRunRef{run}

		webhookBatch, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, executions)
		require.NoError(t, err)
		jiraBatch, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationJira, executions)
		require.NoError(t, err)
		require.NotEqual(t, webhookBatch, jiraBatch)

		// Join table now has two rows for this policy_run — one per type.
		var n int
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &n,
			`SELECT COUNT(*) FROM policy_runs_to_policy_automation_executions WHERE policy_run_id = ?`, run.RunID))
		require.Equal(t, 2, n)
	})

	t.Run("repeated dispatch of the same (run, type) creates a new batch each time", func(t *testing.T) {
		// Simulates a cron retry: the first webhook POST failed (status=failure
		// in the executions row), the next cron tick re-read the failing host
		// via GetFailingPolicyRuns and dispatched again. The second call must
		// not collide with the first — each attempt is its own batch.
		run := mkFailingRun("ce_retry")
		executions := []fleet.PolicyRunRef{run}

		first, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, executions)
		require.NoError(t, err)
		second, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, executions)
		require.NoError(t, err, "second dispatch of the same (run, type) must not fail with PK violation")
		require.NotEqual(t, first, second, "every dispatch gets a fresh batch_id")

		// Two join rows accumulate, one per batch.
		var joinCount int
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &joinCount,
			`SELECT COUNT(*) FROM policy_runs_to_policy_automation_executions
			   WHERE policy_run_id = ? AND automation_type = ?`, run.RunID, fleet.PolicyAutomationWebhook))
		require.Equal(t, 2, joinCount)

		// Two independent status rows, both pending until finalized.
		var execCount int
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &execCount,
			`SELECT COUNT(*) FROM policy_automation_executions
			   WHERE batch_id IN (?, ?) AND status = ?`, first[:], second[:], fleet.PolicyAutomationStatusPending))
		require.Equal(t, 2, execCount)
	})
}

func TestUpdatePolicyAutomationExecutions(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := t.Context()

	user := test.NewUser(t, ds, "Test", "test@example.com", true)
	policy := newTestPolicy(t, ds, user, "ue1", "darwin", nil)

	mkBatch := func(name string, typ fleet.PolicyAutomationType) uuid.UUID {
		host := test.NewHost(t, ds, name, name+".ip", "key-"+name, "uuid-"+name, time.Now())
		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		row, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)
		batchID, err := ds.CreatePolicyAutomationExecutions(ctx, typ, []fleet.PolicyRunRef{
			{PolicyID: policy.ID, HostID: host.ID, RunID: row.ID},
		})
		require.NoError(t, err)
		return batchID
	}

	readStatus := func(t *testing.T, batchID uuid.UUID) (fleet.PolicyAutomationExecutionStatus, *string) {
		t.Helper()
		var s fleet.PolicyAutomationExecutionStatus
		var msg *string
		require.NoError(t, ds.writer(ctx).QueryRowxContext(ctx,
			`SELECT status, error_message FROM policy_automation_executions WHERE batch_id = ?`, batchID[:]).
			Scan(&s, &msg))
		return s, msg
	}

	t.Run("uuid.Nil is a no-op", func(t *testing.T) {
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, uuid.Nil, errors.New("ignored")))
	})

	t.Run("pending → success on nil outcome", func(t *testing.T) {
		batchID := mkBatch("ue_success", fleet.PolicyAutomationWebhook)
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, batchID, nil))
		s, msg := readStatus(t, batchID)
		require.Equal(t, fleet.PolicyAutomationStatusSuccess, s)
		require.Nil(t, msg)
	})

	t.Run("pending → failure stores the error message", func(t *testing.T) {
		batchID := mkBatch("ue_fail", fleet.PolicyAutomationJira)
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, batchID, errors.New("first cause")))
		s, msg := readStatus(t, batchID)
		require.Equal(t, fleet.PolicyAutomationStatusFailure, s)
		require.NotNil(t, msg)
		require.Equal(t, "first cause", *msg)
	})

	t.Run("retry semantics: failure sticks, success upgrades, success is terminal", func(t *testing.T) {
		batchID := mkBatch("ue_retry", fleet.PolicyAutomationJira)

		// First attempt fails — message captured.
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, batchID, errors.New("first cause")))
		s, msg := readStatus(t, batchID)
		require.Equal(t, fleet.PolicyAutomationStatusFailure, s)
		require.Equal(t, "first cause", *msg)

		// Second failure must not overwrite the first message.
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, batchID, errors.New("second cause")))
		_, msg = readStatus(t, batchID)
		require.Equal(t, "first cause", *msg, "failure → failure writes must not rewrite error_message")

		// A later success upgrades the status and clears the error.
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, batchID, nil))
		s, msg = readStatus(t, batchID)
		require.Equal(t, fleet.PolicyAutomationStatusSuccess, s)
		require.Nil(t, msg)

		// Success is terminal — a subsequent error does not regress.
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, batchID, errors.New("post-success")))
		s, _ = readStatus(t, batchID)
		require.Equal(t, fleet.PolicyAutomationStatusSuccess, s)
	})

	t.Run("long error messages with multi-byte runes are stored intact (TEXT column)", func(t *testing.T) {
		batchID := mkBatch("ue_long", fleet.PolicyAutomationJira)
		bigMsg := strings.Repeat("a", 4000) + strings.Repeat("α", 200)
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, batchID, errors.New(bigMsg)))

		s, msg := readStatus(t, batchID)
		require.Equal(t, fleet.PolicyAutomationStatusFailure, s)
		require.NotNil(t, msg)
		require.Equal(t, bigMsg, *msg)
		require.True(t, utf8.ValidString(*msg))
	})

	t.Run("non-existent batchID is a silent no-op", func(t *testing.T) {
		// A finalize call against an unknown batch_id must not error — the
		// UPDATE simply matches zero rows. Documented contract on the interface.
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, uuid.New(), errors.New("stray")))
	})

	t.Run("concurrent finalize: first error wins, late successes upgrade", func(t *testing.T) {
		// Two workers race to finalize the same batch with different errors.
		// InnoDB's row-X lock on the UPDATE serializes them, the WHERE clause
		// rejects the second's failure (state already 'failure'), so the
		// first error message is preserved. A subsequent success then
		// upgrades the row.
		batchID := mkBatch("ue_concurrent", fleet.PolicyAutomationJira)
		const racers = 8

		var wg sync.WaitGroup
		errCh := make(chan error, racers)
		for i := range racers {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				if err := ds.UpdatePolicyAutomationExecutions(ctx, batchID, fmt.Errorf("racer-%d", idx)); err != nil {
					errCh <- err
				}
			}(i)
		}
		wg.Wait()
		close(errCh)
		for err := range errCh {
			require.NoError(t, err)
		}

		s, msg := readStatus(t, batchID)
		require.Equal(t, fleet.PolicyAutomationStatusFailure, s)
		require.NotNil(t, msg)
		require.Regexp(t, `^racer-\d$`, *msg, "exactly one racer's message must be persisted")

		// Success after a stable failure upgrades and clears error_message.
		require.NoError(t, ds.UpdatePolicyAutomationExecutions(ctx, batchID, nil))
		s, msg = readStatus(t, batchID)
		require.Equal(t, fleet.PolicyAutomationStatusSuccess, s)
		require.Nil(t, msg)
	})
}

func TestPolicyRunsForeignKeyBehavior(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := t.Context()

	user := test.NewUser(t, ds, "Test", "test@example.com", true)

	t.Run("deleting a policy cascades to its policy_runs rows", func(t *testing.T) {
		policy := newTestPolicy(t, ds, user, "fk_cascade", "darwin", nil)
		host := test.NewHost(t, ds, "fkCascadeHost", "10.2.0.1", "key-fkc", "uuid-fkc", time.Now())
		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		require.Equal(t, 1, countPolicyRuns(t, ds, ctx, policy.ID, host.ID))

		_, err = ds.DeleteGlobalPolicies(ctx, []uint{policy.ID})
		require.NoError(t, err)

		var n int
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &n,
			`SELECT COUNT(*) FROM policy_runs WHERE policy_id = ?`, policy.ID))
		require.Equal(t, 0, n, "policy_runs rows must cascade-delete when the parent policy is removed")
	})

	t.Run("deleting a policy_runs row sets host_script_results.policy_run_id to NULL", func(t *testing.T) {
		policy := newTestPolicy(t, ds, user, "fk_setnull", "darwin", nil)
		host := test.NewHost(t, ds, "fkSetNullHost", "10.2.0.2", "key-fks", "uuid-fks", time.Now())
		_, err := ds.RecordPolicyTransitions(ctx, host.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		runRow, ok := readPolicyRun(t, ds, ctx, policy.ID, host.ID)
		require.True(t, ok)

		// Insert a host_script_results row stamped with the policy_run_id.
		res, err := ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO host_script_results (host_id, execution_id, output, policy_run_id) VALUES (?, ?, '', ?)`,
			host.ID, "exec-fk-setnull", runRow.ID)
		require.NoError(t, err)
		scriptResultID, err := res.LastInsertId()
		require.NoError(t, err)

		// Delete the policy (which cascades to delete the policy_runs row,
		// which in turn must SET NULL the script row's policy_run_id).
		_, err = ds.DeleteGlobalPolicies(ctx, []uint{policy.ID})
		require.NoError(t, err)

		var runID *uint
		require.NoError(t, ds.writer(ctx).QueryRowxContext(ctx,
			`SELECT policy_run_id FROM host_script_results WHERE id = ?`, scriptResultID).Scan(&runID))
		require.Nil(t, runID, "host_script_results.policy_run_id must be SET NULL when the policy_run is deleted")
	})
}

func TestLookupFailingPolicyRunRefsHandlesLargeInputs(t *testing.T) {
	// Exercises the chunked branch of lookupFailingPolicyRunRefs without
	// having to materialize 1000+ rows in MySQL — the query runs against
	// non-existent IDs and is expected to match nothing, but must not error
	// or panic on the chunking math.
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := t.Context()

	const oversize = policyAutomationBatchSize + 500

	bigPolicyIDs := make([]uint, oversize)
	bigHostIDs := make([]uint, oversize)
	for i := range oversize {
		bigPolicyIDs[i] = uint(i + 1)
		bigHostIDs[i] = uint(i + 1)
	}

	// Both sides exceed the chunk size: chunked side is whichever is larger
	// (here policyIDs by tie-break), and each chunk inlines the full fixed
	// side. No rows match, so the result is empty but the call must succeed.
	refs, err := ds.GetFailingPolicyRuns(ctx, bigPolicyIDs, bigHostIDs)
	require.NoError(t, err)
	require.Empty(t, refs)

	// Swap which side is larger to exercise the other chunkSide branch.
	refs, err = ds.GetFailingPolicyRuns(ctx, bigPolicyIDs[:10], bigHostIDs)
	require.NoError(t, err)
	require.Empty(t, refs)
}
