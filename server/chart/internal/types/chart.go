// Package types provides internal types and interfaces for the chart bounded context.
package types

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
)

// Datastore is the internal datastore interface for the chart bounded context.
type Datastore interface {
	// GetBlobData fetches raw host bitmap blobs from host_hourly_data_blobs for a given
	// dataset and date range.
	GetBlobData(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []string) ([]chart.BlobDataPoint, error)

	// GetHostIDsForFilter returns the host IDs that match the given host filter.
	// Used to build a filter bitmask for blob-based datasets.
	GetHostIDsForFilter(ctx context.Context, hostFilter *chart.HostFilter) ([]uint, error)

	// CountHostsForChartFilter returns the total number of hosts matching the chart host filters.
	CountHostsForChartFilter(ctx context.Context, hostFilter *chart.HostFilter) (int, error)

	// CollectUptimeChartData bulk-inserts uptime blob data for all recently seen hosts.
	CollectUptimeChartData(ctx context.Context, now time.Time) error

	// CollectPolicyFailingChartData builds per-policy failing-host bitmaps from
	// policy_membership and reconciles them into the SCD table.
	CollectPolicyFailingChartData(ctx context.Context, now time.Time) error

	// CleanupBlobData deletes blob rows older than the specified number of days.
	CleanupBlobData(ctx context.Context, days int) error

	// RecordSCDData reconciles the current per-entity host bitmaps for a dataset
	// against the table's open rows. Entities whose bitmap has not changed are left
	// alone. Entities with a new bitmap get their open row closed (when valid_from
	// is from a previous day) and a new row inserted for today; same-day bitmap
	// changes overwrite today's row in place. Entities absent from the input
	// that have open rows are closed.
	RecordSCDData(ctx context.Context, dataset string, entityBitmaps map[string][]byte, now time.Time) error

	// GetSCDData returns per-day distinct-host counts for an SCD dataset over the
	// given range. Applies the optional host filter via bitmap AND and the optional
	// entity filter via entity_id IN.
	GetSCDData(ctx context.Context, dataset string, startDate, endDate time.Time, hostFilter *chart.HostFilter, entityIDs []string) ([]chart.DataPoint, error)

	// CleanupSCDData deletes closed SCD rows whose valid_to is older than the
	// retention cutoff. Open rows (valid_to = sentinel) are never deleted.
	CleanupSCDData(ctx context.Context, days int) error

	// GetPolicyFailingSnapshot returns the currently-open per-policy failing-host
	// bitmaps from the policy_failing SCD dataset.
	GetPolicyFailingSnapshot(ctx context.Context) ([]chart.PolicyFailingSnapshot, error)

	// GetPoliciesMetadata returns id/name/team_id for every policy in the policies table.
	GetPoliciesMetadata(ctx context.Context) ([]chart.PolicyMeta, error)

	// GetTeamsMetadata returns id/name for every team in the teams table.
	GetTeamsMetadata(ctx context.Context) ([]chart.TeamMeta, error)

	// GetHostTeamAssignments returns the team assignment for every host.
	// Hosts with NULL team_id have TeamID == nil in the returned rows.
	GetHostTeamAssignments(ctx context.Context) ([]chart.HostTeam, error)

	// GetPolicyFailingByTeamTrend returns per-day, per-team counts of hosts
	// failing ≥1 policy over the given date range, for the stacked-bar chart.
	GetPolicyFailingByTeamTrend(ctx context.Context, startDate, endDate time.Time) ([]chart.TeamTrendPoint, error)

	// GetTopNonCompliantHosts returns the N hosts with the most currently-failing
	// policies, hydrated with hostname / team metadata for display.
	GetTopNonCompliantHosts(ctx context.Context, limit int) ([]chart.HostFailingSummary, error)
}
