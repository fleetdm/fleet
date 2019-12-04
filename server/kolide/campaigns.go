package kolide

import (
	"context"
	"time"

	"github.com/kolide/fleet/server/websocket"
)

// CampaignStore defines the distributed query campaign related datastore
// methods
type CampaignStore interface {
	// NewDistributedQueryCampaign creates a new distributed query campaign
	NewDistributedQueryCampaign(camp *DistributedQueryCampaign) (*DistributedQueryCampaign, error)
	// DistributedQueryCampaign loads a distributed query campaign by ID
	DistributedQueryCampaign(id uint) (*DistributedQueryCampaign, error)
	// SaveDistributedQueryCampaign updates an existing distributed query
	// campaign
	SaveDistributedQueryCampaign(camp *DistributedQueryCampaign) error
	// DistributedQueryCampaignTargetIDs gets the IDs of the targets for
	// the query campaign of the provided ID
	DistributedQueryCampaignTargetIDs(id uint) (hostIDs []uint, labelIDs []uint, err error)

	// NewDistributedQueryCampaignTarget adds a new target to an existing
	// distributed query campaign
	NewDistributedQueryCampaignTarget(target *DistributedQueryCampaignTarget) (*DistributedQueryCampaignTarget, error)

	// NewDistributedQueryCampaignExecution records a new execution for a
	// distributed query campaign
	NewDistributedQueryExecution(exec *DistributedQueryExecution) (*DistributedQueryExecution, error)

	// CleanupDistributedQueryCampaigns will clean and trim metadata for
	// old distributed query campaigns. Any campaign in the QueryWaiting
	// state will be moved to QueryComplete after one minute. Any campaign
	// in the QueryRunning state will be moved to QueryComplete after one
	// day. Any campaign in the QueryComplete state will have the
	// associated executions deleted. All times are from creation time. The
	// now parameter makes this method easier to test. The return values
	// indicate how many campaigns were expired, how many executions were
	// deleted, and any error.
	CleanupDistributedQueryCampaigns(now time.Time) (expired uint, deleted uint, err error)
}

// CampaignService defines the distributed query campaign related service
// methods
type CampaignService interface {
	// NewDistributedQueryCampaign creates a new distributed query campaign
	// with the provided query and host/label targets (specified by name).
	NewDistributedQueryCampaignByNames(ctx context.Context, queryString string, hosts []string, labels []string) (*DistributedQueryCampaign, error)

	// NewDistributedQueryCampaign creates a new distributed query campaign
	// with the provided query and host/label targets
	NewDistributedQueryCampaign(ctx context.Context, queryString string, hosts []uint, labels []uint) (*DistributedQueryCampaign, error)

	// StreamCampaignResults streams updates with query results and
	// expected host totals over the provided websocket. Note that the type
	// signature is somewhat inconsistent due to this being a streaming API
	// and not the typical go-kit RPC style.
	StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint)
}

// DistributedQueryStatus is the lifecycle status of a distributed query
// campaign.
type DistributedQueryStatus int

const (
	QueryWaiting DistributedQueryStatus = iota
	QueryRunning
	QueryComplete
)

// DistributedQueryCampaign is the basic metadata associated with a distributed
// query.
type DistributedQueryCampaign struct {
	UpdateCreateTimestamps
	DeleteFields
	Metrics TargetMetrics
	ID      uint                   `json:"id"`
	QueryID uint                   `json:"query_id" db:"query_id"`
	Status  DistributedQueryStatus `json:"status"`
	UserID  uint                   `json:"user_id" db:"user_id"`
}

// DistributedQueryCampaignTarget stores a target (host or label) for a
// distributed query campaign. There is a one -> many mapping of campaigns to
// targets.
type DistributedQueryCampaignTarget struct {
	ID                         uint
	Type                       TargetType
	DistributedQueryCampaignID uint `db:"distributed_query_campaign_id"`
	TargetID                   uint `db:"target_id"`
}

// DistributedQueryExecutionStatus is the status of a distributed query
// execution on a single host.
type DistributedQueryExecutionStatus int

const (
	ExecutionWaiting DistributedQueryExecutionStatus = iota
	ExecutionRequested
	ExecutionSucceeded
	ExecutionFailed
)

// DistributedQueryResult is the result returned from the execution of a
// distributed query on a single host.
type DistributedQueryResult struct {
	DistributedQueryCampaignID uint                `json:"distributed_query_execution_id"`
	Host                       Host                `json:"host"`
	Rows                       []map[string]string `json:"rows"`
	// osquery currently doesn't return any helpful error information,
	// but we use string here instead of bool for future-proofing. Note also
	// that we can't use the error interface here because something
	// implementing that interface may not (un)marshal properly
	Error *string `json:"error"`
}

// DistributedQueryExecution is the metadata associated with a distributed
// query execution on a single host.
type DistributedQueryExecution struct {
	ID                         uint
	HostID                     uint `db:"host_id"`
	DistributedQueryCampaignID uint `db:"distributed_query_campaign_id"`
	Status                     DistributedQueryExecutionStatus
	Error                      string
	ExecutionDuration          time.Duration `db:"execution_duration"`
}
