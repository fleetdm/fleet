package fleet

type GlobalPolicyRequest struct {
	QueryID          *uint    `json:"query_id" renameto:"report_id"`
	Query            string   `json:"query"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Resolution       string   `json:"resolution"`
	Platform         string   `json:"platform"`
	Critical         bool     `json:"critical" premium:"true"`
	LabelsIncludeAny []string `json:"labels_include_any"`
	LabelsExcludeAny []string `json:"labels_exclude_any"`
}

type GlobalPolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r GlobalPolicyResponse) Error() error { return r.Err }

type ListGlobalPoliciesRequest struct {
	Opts ListOptions `url:"list_options"`
}

type ListGlobalPoliciesResponse struct {
	Policies []*Policy `json:"policies,omitempty"`
	Err      error     `json:"error,omitempty"`
}

func (r ListGlobalPoliciesResponse) Error() error { return r.Err }

type GetPolicyByIDRequest struct {
	PolicyID uint `url:"policy_id"`
}

type GetPolicyByIDResponse struct {
	Policy *Policy `json:"policy"`
	Err    error   `json:"error,omitempty"`
}

func (r GetPolicyByIDResponse) Error() error { return r.Err }

type CountGlobalPoliciesRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type CountGlobalPoliciesResponse struct {
	Count int   `json:"count"`
	Err   error `json:"error,omitempty"`
}

func (r CountGlobalPoliciesResponse) Error() error { return r.Err }

type DeleteGlobalPoliciesRequest struct {
	IDs []uint `json:"ids"`
}

type DeleteGlobalPoliciesResponse struct {
	Deleted []uint `json:"deleted,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r DeleteGlobalPoliciesResponse) Error() error { return r.Err }

type ModifyGlobalPolicyRequest struct {
	PolicyID uint `url:"policy_id"`
	ModifyPolicyPayload
}

type ModifyGlobalPolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r ModifyGlobalPolicyResponse) Error() error { return r.Err }

type ResetAutomationRequest struct {
	TeamIDs   []uint `json:"team_ids" premium:"true" renameto:"fleet_ids"`
	PolicyIDs []uint `json:"policy_ids"`
}

type ResetAutomationResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ResetAutomationResponse) Error() error { return r.Err }

type ApplyPolicySpecsRequest struct {
	Specs []*PolicySpec `json:"specs"`
}

type ApplyPolicySpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyPolicySpecsResponse) Error() error { return r.Err }

type AutofillPoliciesRequest struct {
	SQL string `json:"sql"`
}

type AutofillPoliciesResponse struct {
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
	Err         error  `json:"error,omitempty"`
}

func (a AutofillPoliciesResponse) Error() error {
	return a.Err
}
