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
