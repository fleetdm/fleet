package fleet

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

// DistributedQueryResult is the result returned from the execution of a
// distributed query on a single host.
//
// IMPORTANT: This struct is stored in the result store (e.g. Redis)
// and is streamed to a pubsub listener (browser or fleetctl user) via websockets,
// thus it should be kept as small as possible.
type DistributedQueryResult struct {
	// DistributedQueryCampaignID is the unique ID of the live query campaign.
	DistributedQueryCampaignID uint `json:"distributed_query_execution_id"`
	// Host holds the host's data from where the query result comes from.
	Host  ResultHostData      `json:"host"`
	Rows  []map[string]string `json:"rows"`
	Stats *Stats              `json:"stats"`
	// Error contains any error reported by osquery when running the query.
	// Note we can't use the error interface here because something
	// implementing that interface may not (un)marshal properly
	Error *string `json:"error,omitempty"`
}

// ResultHostData holds the host's data from where a query result comes from.
type ResultHostData struct {
	// ID is the unique ID of the host.
	ID uint `json:"id"`
	// Hostname is the host's hostname.
	Hostname string `json:"hostname"`
	// DisplayName holds the display name of the host.
	DisplayName string `json:"display_name"`
}

type QueryResult struct {
	HostID uint                `json:"host_id"`
	Rows   []map[string]string `json:"rows"`
	Error  *string             `json:"error"`
}

type QueryCampaignResult struct {
	QueryID uint          `json:"query_id"`
	Error   *string       `json:"error,omitempty"`
	Results []QueryResult `json:"results"`
	Err     error         `json:"-"`
}
