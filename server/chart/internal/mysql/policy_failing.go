package mysql

import (
	"context"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// CollectPolicyFailingChartData snapshots the current per-policy failing-host
// state into the SCD table.
//
// For each policy in the `policies` table we emit an entity_id -> bitmap pair.
// The bitmap encodes host IDs currently failing that policy (from
// policy_membership.passes = 0). A policy with zero failing hosts is still
// emitted with an empty bitmap, which lets callers answer "how many policies
// are tracked today" by counting distinct entity_ids with open SCD rows.
//
// Carryforward for silent hosts is handled upstream by policy_membership,
// which preserves a host's last-known pass/fail value until it reports again.
// We therefore do not need to merge yesterday's state — the source table
// already is yesterday's state for any host that hasn't checked in.
func (ds *Datastore) CollectPolicyFailingChartData(ctx context.Context, now time.Time) error {
	// All policies — used to ensure a row per policy even when no hosts are failing.
	var policyIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &policyIDs,
		`SELECT id FROM policies`); err != nil {
		return ctxerr.Wrap(ctx, err, "list policy IDs for chart collection")
	}

	// Failing (policy_id, host_id) pairs. ORDER BY policy_id keeps per-policy
	// host lists contiguous in the result set — not required for correctness,
	// but helps when inspecting the query in isolation.
	type failingRow struct {
		PolicyID uint `db:"policy_id"`
		HostID   uint `db:"host_id"`
	}
	var failing []failingRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &failing,
		`SELECT policy_id, host_id
		 FROM policy_membership
		 WHERE passes = 0
		 ORDER BY policy_id, host_id`); err != nil {
		return ctxerr.Wrap(ctx, err, "query failing policy memberships")
	}

	failingByPolicy := make(map[uint][]uint, len(policyIDs))
	for _, r := range failing {
		failingByPolicy[r.PolicyID] = append(failingByPolicy[r.PolicyID], r.HostID)
	}

	// Build one entry per policy. Empty bitmaps are coerced to a non-nil empty
	// slice because host_bitmap is MEDIUMBLOB NOT NULL — Go nil would bind as NULL.
	entityBitmaps := make(map[string][]byte, len(policyIDs))
	for _, pid := range policyIDs {
		key := strconv.FormatUint(uint64(pid), 10)
		blob := chart.HostIDsToBlob(failingByPolicy[pid])
		if blob == nil {
			blob = []byte{}
		}
		entityBitmaps[key] = blob
	}

	if err := ds.RecordSCDData(ctx, "policy_failing", entityBitmaps, now); err != nil {
		return ctxerr.Wrap(ctx, err, "record policy_failing SCD data")
	}
	return nil
}
