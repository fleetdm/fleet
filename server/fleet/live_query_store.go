package fleet

import (
	"context"
	"time"
)

// LiveQueryStore defines an interface for storing and retrieving the status of
// live queries in the Fleet system.
type LiveQueryStore interface {
	// RunQuery starts a query with the given name and SQL, targeting the
	// provided host IDs.
	RunQuery(name, sql string, hostIDs []uint) error
	// StopQuery stops a running query with the given name. Hosts will no longer
	// receive the query after StopQuery has been called.
	StopQuery(name string) error
	// QueriesForHost returns the active queries for the given host ID. The
	// return value maps from query name to SQL.
	QueriesForHost(hostID uint) (map[string]string, error)
	// QueryCompletedByHost marks the query with the given name as completed by the
	// given host. After calling QueryCompleted, that query will no longer be
	// sent to the host.
	QueryCompletedByHost(name string, hostID uint) error
	// IsQueryTargetingHost reports whether the query with the given name is
	// active and still targets the given host (i.e. the host has not yet
	// completed it).
	IsQueryTargetingHost(name string, hostID uint) (bool, error)
	// CleanupInactiveQueries removes any inactive queries. This is used via a
	// cron job to regularly cleanup any queries that may have failed to be
	// stopped properly in Redis.
	CleanupInactiveQueries(ctx context.Context, inactiveCampaignIDs []uint) error
	// LoadActiveQueryNames returns the names of all active queries.
	LoadActiveQueryNames() ([]string, error)

	// GetQueryResultsCounts returns the current count of query results for multiple queries.
	// Queries with no stored count are absent from the result so callers can fall back to the
	// database.
	GetQueryResultsCounts(queryIDs []uint) (map[uint]int, error)
	// SetQueryResultsCountsIfAbsent seeds counts only for queries that have none stored, so a
	// concurrent increment from another request is never overwritten.
	SetQueryResultsCountsIfAbsent(counts map[uint]int) error
	// IncrQueryResultsCounts increments the query results counts by the given amounts.
	// Takes a map of query ID -> amount to increment.
	IncrQueryResultsCounts(queryIDsToAmounts map[uint]int) error
	// SetQueryResultsCount sets the query results count for a query to a specific value.
	// Used by the cleanup cron job after deleting excess rows to set the count to the max allowed.
	SetQueryResultsCount(queryID uint, count int) error
	// DeleteQueryResultsCount deletes the query results count for a query.
	// Used when deleting a query, to remove the Redis key.
	DeleteQueryResultsCount(queryID uint) error

	// SetQueryReportsHostCount stores the total number of hosts, used to raise
	// the query report cap so that reports are never capped below one row per host.
	// Refreshed by the query results cleanup cron job.
	SetQueryReportsHostCount(count int) error
	// GetQueryReportsHostCount returns the host count stored by SetQueryReportsHostCount. ok is
	// false when none is stored, so callers can fall back to the database.
	GetQueryReportsHostCount() (count int, ok bool, err error)
	// SetQueryReportsHostCountIfAbsent seeds the host count only when none is stored.
	SetQueryReportsHostCountIfAbsent(count int) error
	// IncrQueryReportsHostCount adjusts the stored host count by delta, so newly enrolled hosts
	// raise the cap before the cron refreshes it.
	IncrQueryReportsHostCount(delta int) error

	// MarkQueryReportsClipped records that a host's results for each query were rejected because of
	// the report cap. Each marker expires after its ttl so it clears itself once rejections stop.
	MarkQueryReportsClipped(ttlByQueryID map[uint]time.Duration) error
	// QueryReportsClipped returns which of the given queries have a clipped marker set.
	QueryReportsClipped(queryIDs []uint) (map[uint]bool, error)
	// ClearQueryReportsClipped removes the clipped marker for the given queries. Used when a
	// report admits a host it didn't cover yet, when its results are discarded, and when the query
	// is deleted.
	ClearQueryReportsClipped(queryIDs []uint) error
}
