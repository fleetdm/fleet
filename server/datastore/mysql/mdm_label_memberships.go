package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// Host label membership lookups shared by the batched MDM profile
// reconcilers. Platform-neutral: the Apple reconcile snapshot uses the
// transaction variant today and other platforms' snapshots reuse it as
// they move to in-memory reconciliation.

// BulkGetHostLabelMemberships returns, for each given host ID, the set of
// label IDs (from the provided labelIDs) the host is a member of.
//
// Both lists may be empty; in either case the result is an empty (non-nil)
// map. The IN clauses are chunked to keep total placeholders well under
// MySQL's prepared-statement parameter limit.
func (ds *Datastore) BulkGetHostLabelMemberships(
	ctx context.Context,
	hostIDs []uint,
	labelIDs []uint,
) (map[uint]map[uint]struct{}, error) {
	return ds.bulkGetHostLabelMembershipsTransaction(ctx, ds.reader(ctx), hostIDs, labelIDs)
}

func (ds *Datastore) bulkGetHostLabelMembershipsTransaction(
	ctx context.Context,
	tx common_mysql.DBReadTx,
	hostIDs []uint,
	labelIDs []uint,
) (map[uint]map[uint]struct{}, error) {
	out := make(map[uint]map[uint]struct{}, len(hostIDs))
	if len(hostIDs) == 0 || len(labelIDs) == 0 {
		return out, nil
	}

	const (
		hostChunk  = 5000
		labelChunk = 1000
	)

	const stmt = `SELECT host_id, label_id FROM label_membership WHERE host_id IN (?) AND label_id IN (?)`

	type membershipRow struct {
		HostID  uint `db:"host_id"`
		LabelID uint `db:"label_id"`
	}

	for hi := 0; hi < len(hostIDs); hi += hostChunk {
		hEnd := min(hi+hostChunk, len(hostIDs))
		hostBatch := hostIDs[hi:hEnd]

		for li := 0; li < len(labelIDs); li += labelChunk {
			lEnd := min(li+labelChunk, len(labelIDs))
			labelBatch := labelIDs[li:lEnd]

			q, args, err := sqlx.In(stmt, hostBatch, labelBatch)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "build label membership query")
			}

			var rows []membershipRow
			if err := sqlx.SelectContext(ctx, tx, &rows, q, args...); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "query host label memberships")
			}

			for _, r := range rows {
				set, ok := out[r.HostID]
				if !ok {
					set = make(map[uint]struct{})
					out[r.HostID] = set
				}
				set[r.LabelID] = struct{}{}
			}
		}
	}

	return out, nil
}
