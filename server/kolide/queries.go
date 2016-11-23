package kolide

import (
	"time"

	"github.com/kolide/kolide-ose/server/websocket"
	"golang.org/x/net/context"
)

type QueryStore interface {
	// Query methods
	NewQuery(query *Query) (*Query, error)
	SaveQuery(query *Query) error
	DeleteQuery(query *Query) error
	Query(id uint) (*Query, error)
	ListQueries(opt ListOptions) ([]*Query, error)

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
}

type QueryService interface {
	ListQueries(ctx context.Context, opt ListOptions) ([]*Query, error)
	GetQuery(ctx context.Context, id uint) (*Query, error)
	NewQuery(ctx context.Context, p QueryPayload) (*Query, error)
	ModifyQuery(ctx context.Context, id uint, p QueryPayload) (*Query, error)
	DeleteQuery(ctx context.Context, id uint) error
	NewDistributedQueryCampaign(ctx context.Context, queryString string, hosts []uint, labels []uint) (*DistributedQueryCampaign, error)

	// StreamCampaignResults streams updates with query results and
	// expected host totals over the provided websocket. Note that the type
	// signature is somewhat inconsistent due to this being a streaming API
	// and not the typical go-kit RPC style.
	StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint)
}

type QueryPayload struct {
	Name         *string
	Description  *string
	Query        *string
	Interval     *uint
	Snapshot     *bool
	Differential *bool
	Platform     *string
	Version      *string
}

type Query struct {
	UpdateCreateTimestamps
	DeleteFields
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Query        string `json:"query"`
	Interval     uint   `json:"interval"`
	Snapshot     bool   `json:"snapshot"`
	Differential bool   `json:"differential"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
}

type DistributedQueryStatus int

const (
	QueryRunning  DistributedQueryStatus = iota
	QueryComplete DistributedQueryStatus = iota
	QueryError    DistributedQueryStatus = iota
)

type DistributedQueryCampaign struct {
	UpdateCreateTimestamps
	DeleteFields
	ID      uint                   `json:"id"`
	QueryID uint                   `json:"query_id" db:"query_id"`
	Status  DistributedQueryStatus `json:"status"`
	UserID  uint                   `json:"user_id" db:"user_id"`
}

type DistributedQueryCampaignTarget struct {
	ID                         uint
	Type                       TargetType
	DistributedQueryCampaignID uint `db:"distributed_query_campaign_id"`
	TargetID                   uint `db:"target_id"`
}

type DistributedQueryExecutionStatus int

const (
	ExecutionWaiting DistributedQueryExecutionStatus = iota
	ExecutionRequested
	ExecutionSucceeded
	ExecutionFailed
)

type DistributedQueryResult struct {
	DistributedQueryCampaignID uint                `json:"distributed_query_execution_id"`
	Host                       Host                `json:"host"`
	Rows                       []map[string]string `json:"rows"`
}

type DistributedQueryExecution struct {
	ID                         uint
	HostID                     uint `db:"host_id"`
	DistributedQueryCampaignID uint `db:"distributed_query_campaign_id"`
	Status                     DistributedQueryExecutionStatus
	Error                      string
	ExecutionDuration          time.Duration `db:"execution_duration"`
}

type Option struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	Key       string
	Value     string
	Platform  string
}

type DecoratorType int

const (
	DecoratorLoad DecoratorType = iota
	DecoratorAlways
	DecoratorInterval
)

type Decorator struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      DecoratorType
	Interval  int
	Query     string
}
