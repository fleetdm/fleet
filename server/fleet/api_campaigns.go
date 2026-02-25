package fleet

type CreateDistributedQueryCampaignRequest struct {
	QuerySQL string      `json:"query"`
	QueryID  *uint       `json:"query_id" renameto:"report_id"`
	Selected HostTargets `json:"selected"`
}

type CreateDistributedQueryCampaignResponse struct {
	Campaign *DistributedQueryCampaign `json:"campaign,omitempty"`
	Err      error                     `json:"error,omitempty"`
}

func (r CreateDistributedQueryCampaignResponse) Error() error { return r.Err }

// DistributedQueryCampaignTargetsByIdentifiers holds campaign targets specified by string identifiers.
type DistributedQueryCampaignTargetsByIdentifiers struct {
	Labels []string `json:"labels"`
	// list of hostnames, UUIDs, and/or hardware serials
	Hosts []string `json:"hosts"`
}

type CreateDistributedQueryCampaignByIdentifierRequest struct {
	QuerySQL string                                           `json:"query"`
	QueryID  *uint                                            `json:"query_id" renameto:"report_id"`
	Selected DistributedQueryCampaignTargetsByIdentifiers `json:"selected"`
}
