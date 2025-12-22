package contract

type EnrollOsqueryAgentRequest struct {
	EnrollSecret   string                       `json:"enroll_secret"`
	HostIdentifier string                       `json:"host_identifier"`
	HostDetails    map[string]map[string]string `json:"host_details"`
}

type EnrollOsqueryAgentResponse struct {
	NodeKey string `json:"node_key,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r EnrollOsqueryAgentResponse) Error() error { return r.Err }
