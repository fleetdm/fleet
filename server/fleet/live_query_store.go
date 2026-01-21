package fleet

import "context"

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
	// CleanupInactiveQueries removes any inactive queries. This is used via a
	// cron job to regularly cleanup any queries that may have failed to be
	// stopped properly in Redis.
	CleanupInactiveQueries(ctx context.Context, inactiveCampaignIDs []uint) error
	// LoadActiveQueryNames returns the names of all active queries.
	LoadActiveQueryNames() ([]string, error)

	// GetQueryResultsCounts returns the current count of query results for multiple queries.
	// Returns a map of query ID -> count. Missing keys are returned with a count of 0.
	GetQueryResultsCounts(queryIDs []uint) (map[uint]int, error)
	// IncrQueryResultsCount increments the query results count by the given amount.
	// Returns the new count after incrementing.
	IncrQueryResultsCount(queryID uint, amount int) (int, error)
	// SetQueryResultsCount sets the query results count for a query to a specific value.
	// Used by the cleanup cron job after deleting excess rows to set the count to the max allowed.
	SetQueryResultsCount(queryID uint, count int) error
	// DeleteQueryResultsCount deletes the query results count for a query.
	// Used when deleting a query, to remove the Redis key.
	DeleteQueryResultsCount(queryID uint) error
}
