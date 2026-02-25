package fleet

type RunLiveQueryRequest struct {
	QueryIDs []uint `json:"query_ids" renameto:"report_ids"`
	HostIDs  []uint `json:"host_ids"`
}

type RunOneLiveQueryRequest struct {
	QueryID uint   `url:"id"`
	HostIDs []uint `json:"host_ids"`
}

type RunLiveQueryOnHostRequest struct {
	Identifier string `url:"identifier"`
	Query      string `json:"query"`
}

type RunLiveQueryOnHostByIDRequest struct {
	HostID uint   `url:"id"`
	Query  string `json:"query"`
}

type SummaryPayload struct {
	TargetedHostCount  int `json:"targeted_host_count"`
	RespondedHostCount int `json:"responded_host_count"`
}

type RunLiveQueryResponse struct {
	Summary SummaryPayload `json:"summary"`
	Err     error          `json:"error,omitempty"`

	Results []QueryCampaignResult `json:"live_query_results" renameto:"live_report_results"`
}

func (r RunLiveQueryResponse) Error() error { return r.Err }

type RunOneLiveQueryResponse struct {
	QueryID            uint          `json:"query_id" renameto:"report_id"`
	TargetedHostCount  int           `json:"targeted_host_count"`
	RespondedHostCount int           `json:"responded_host_count"`
	Results            []QueryResult `json:"results"`
	Err                error         `json:"error,omitempty"`
}

func (r RunOneLiveQueryResponse) Error() error { return r.Err }

type RunLiveQueryOnHostResponse struct {
	HostID uint                `json:"host_id"`
	Rows   []map[string]string `json:"rows"`
	Query  string              `json:"query"`
	Status HostStatus          `json:"status"`
	Err    string              `json:"error,omitempty"`
}

func (r RunLiveQueryOnHostResponse) Error() error { return nil }
