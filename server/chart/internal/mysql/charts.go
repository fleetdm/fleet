// Package mysql provides the MySQL datastore implementation for the chart bounded context.
package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// Datastore is the MySQL implementation of the chart datastore.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  *slog.Logger
}

// NewDatastore creates a new MySQL datastore for the chart bounded context.
func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger}
}

// Ensure Datastore implements types.Datastore at compile time.
var _ types.Datastore = (*Datastore)(nil)

func (ds *Datastore) reader(ctx context.Context) sqlx.QueryerContext {
	if ctxdb.IsPrimaryRequired(ctx) {
		return ds.primary
	}
	return ds.replica
}

func (ds *Datastore) writer(_ context.Context) *sqlx.DB {
	return ds.primary
}

// rebind rewrites a query from ? placeholders to the driver-specific format.
func (ds *Datastore) rebind(query string) string {
	return ds.primary.Rebind(query)
}

func (ds *Datastore) GetHostIDsForFilter(ctx context.Context, hostFilter *types.HostFilter) ([]uint, error) {
	subquery, args := buildHostFilterClauses(hostFilter)

	query := fmt.Sprintf(`SELECT h.id FROM hosts h WHERE 1=1 %s`, subquery)

	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand host IDs filter query args")
	}
	query = ds.rebind(query)

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs for filter")
	}
	return ids, nil
}

// FindRecentlySeenHostIDs returns host IDs with any activity signal at or after `since`.
// "Activity signal" is the most recent of host_seen_times.seen_time, nano_enrollments.last_seen_at,
// or host details/creation timestamps.
func (ds *Datastore) FindRecentlySeenHostIDs(ctx context.Context, since time.Time) ([]uint, error) {
	const query = `
		SELECT h.id
		FROM hosts h
			LEFT JOIN host_seen_times hst ON h.id = hst.host_id
			LEFT JOIN nano_enrollments ne ON ne.id = h.uuid
				AND ne.type IN ('Device', 'User Enrollment (Device)')
		WHERE COALESCE(
			GREATEST(
				COALESCE(hst.seen_time, ne.last_seen_at),
				COALESCE(ne.last_seen_at, hst.seen_time)
			),
			NULLIF(h.detail_updated_at, '2000-01-01 00:00:00'),
			h.created_at
		) >= ?`

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, query, since.UTC()); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "find recently seen host IDs")
	}
	return ids, nil
}

// AffectedHostIDsByCVE returns host IDs grouped by CVE. It streams two joins
// (software-level and OS-level vulnerabilities) and merges the results into a
// single map. Duplicates across sources are harmless — the downstream
// HostIDsToBlob setBit is idempotent.
func (ds *Datastore) AffectedHostIDsByCVE(ctx context.Context) (map[string][]uint, error) {
	result := make(map[string][]uint)

	if err := streamCVEHostPairs(ctx, ds.reader(ctx), `
		SELECT sc.cve, hs.host_id
		FROM software_cve sc
		JOIN host_software hs ON hs.software_id = sc.software_id`, result); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "stream software CVE host pairs")
	}

	if err := streamCVEHostPairs(ctx, ds.reader(ctx), `
		SELECT osv.cve, hos.host_id
		FROM operating_system_vulnerabilities osv
		JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id`, result); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "stream OS CVE host pairs")
	}

	return result, nil
}

// streamCVEHostPairs runs a query yielding (cve, host_id) pairs and appends
// host IDs into out under each CVE key.
func streamCVEHostPairs(ctx context.Context, q sqlx.QueryerContext, query string, out map[string][]uint) error {
	rows, err := q.QueryxContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var cve string
	var hostID uint
	for rows.Next() {
		if err := rows.Scan(&cve, &hostID); err != nil {
			return err
		}
		out[cve] = append(out[cve], hostID)
	}
	return rows.Err()
}

// buildHostFilterClauses translates a HostFilter into SQL WHERE clauses for
// the hosts table. Uses "h" as the table alias. Args may contain slices —
// caller must use sqlx.In to expand them.
func buildHostFilterClauses(filter *types.HostFilter) (string, []any) {
	if filter == nil {
		return "", nil
	}

	var clauses []string
	var args []any

	if filter.TeamIDs != nil {
		// Empty non-nil: caller is team-scoped with zero accessible teams;
		// emit a guaranteed-empty clause so we never run IN () and never return
		// hosts the caller can't see.
		if len(filter.TeamIDs) == 0 {
			clauses = append(clauses, "1=0")
		} else {
			// Split "no team" (id 0) from real team ids — the two map to
			// different SQL (IS NULL vs = ?), so they're OR-ed together when
			// both are present.
			var positive []uint
			includesNoTeam := false
			for _, tid := range filter.TeamIDs {
				if tid == 0 {
					includesNoTeam = true
				} else {
					positive = append(positive, tid)
				}
			}
			switch {
			case includesNoTeam && len(positive) > 0:
				clauses = append(clauses, "(h.team_id IS NULL OR h.team_id IN (?))")
				args = append(args, positive)
			case includesNoTeam:
				clauses = append(clauses, "h.team_id IS NULL")
			default:
				clauses = append(clauses, "h.team_id IN (?)")
				args = append(args, positive)
			}
		}
	}

	if len(filter.LabelIDs) > 0 {
		clauses = append(clauses, "h.id IN (SELECT DISTINCT host_id FROM label_membership WHERE label_id IN (?))")
		args = append(args, filter.LabelIDs)
	}

	if len(filter.Platforms) > 0 {
		clauses = append(clauses, "h.platform IN (?)")
		args = append(args, filter.Platforms)
	}

	if len(filter.IncludeHostIDs) > 0 {
		clauses = append(clauses, "h.id IN (?)")
		args = append(args, filter.IncludeHostIDs)
	}

	if len(filter.ExcludeHostIDs) > 0 {
		clauses = append(clauses, "h.id NOT IN (?)")
		args = append(args, filter.ExcludeHostIDs)
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return " AND " + strings.Join(clauses, " AND "), args
}
