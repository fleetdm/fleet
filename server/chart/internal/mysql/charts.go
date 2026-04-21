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

func (ds *Datastore) CountHostsForChartFilter(ctx context.Context, hostFilter *types.HostFilter) (int, error) {
	subquery, args := buildHostCountFilterClauses(hostFilter)

	query := fmt.Sprintf(`SELECT COUNT(*) FROM hosts h WHERE 1=1 %s`, subquery)

	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "expand count hosts query args")
	}
	query = ds.rebind(query)

	var count int
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, query, args...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count hosts for chart filter")
	}
	return count, nil
}

func (ds *Datastore) GetHostIDsForFilter(ctx context.Context, hostFilter *types.HostFilter) ([]uint, error) {
	subquery, args := buildHostCountFilterClauses(hostFilter)

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

// FindRecentlySeenHostIDs returns host IDs with any activity signal newer than now-lookback.
// "Activity signal" is the most recent of host_seen_times.seen_time, nano_enrollments.last_seen_at,
// or host details/creation timestamps.
func (ds *Datastore) FindRecentlySeenHostIDs(ctx context.Context, lookback time.Duration) ([]uint, error) {
	cutoff := time.Now().UTC().Add(-lookback)

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
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &ids, query, cutoff); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "find recently seen host IDs")
	}
	return ids, nil
}

// buildHostCountFilterClauses builds filter clauses for counting hosts directly from the hosts table.
// Uses "h" as the table alias. Args may contain slices — caller must use sqlx.In to expand them.
func buildHostCountFilterClauses(filter *types.HostFilter) (string, []any) {
	if filter == nil {
		return "", nil
	}

	var clauses []string
	var args []any

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
