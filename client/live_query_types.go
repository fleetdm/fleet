package client

// targetTotals contains aggregated target information for a live query campaign.
type targetTotals struct {
	Total           uint `json:"count"`
	Online          uint `json:"online"`
	Offline         uint `json:"offline"`
	MissingInAction uint `json:"missing_in_action"`
}

const (
	campaignStatusPending  = "pending"
	campaignStatusFinished = "finished"
)

// campaignStatus holds the current status of a live query campaign.
type campaignStatus struct {
	ExpectedResults           uint   `json:"expected_results"`
	ActualResults             uint   `json:"actual_results"`
	CountOfHostsWithResults   uint   `json:"count_of_hosts_with_results"`
	CountOfHostsWithNoResults uint   `json:"count_of_hosts_with_no_results"`
	Status                    string `json:"status"`
}
