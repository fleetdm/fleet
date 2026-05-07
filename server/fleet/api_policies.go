package fleet

/////////////////////////////////////////////////////////////////////////////////
// Global Policy - Add
/////////////////////////////////////////////////////////////////////////////////

type GlobalPolicyRequest struct {
	QueryID          *uint    `json:"query_id" renameto:"report_id"`
	Query            string   `json:"query"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Resolution       string   `json:"resolution"`
	Platform         string   `json:"platform"`
	Critical         bool     `json:"critical" premium:"true"`
	LabelsIncludeAny []string `json:"labels_include_any"`
	LabelsIncludeAll []string `json:"labels_include_all" premium:"true"`
	LabelsExcludeAny []string `json:"labels_exclude_any"`
}

type GlobalPolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r GlobalPolicyResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Global Policy - List
/////////////////////////////////////////////////////////////////////////////////

type ListGlobalPoliciesRequest struct {
	Opts ListOptions `url:"list_options"`
}

type ListGlobalPoliciesResponse struct {
	Policies []*Policy `json:"policies,omitempty"`
	Err      error     `json:"error,omitempty"`
}

func (r ListGlobalPoliciesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Global Policy - Get by id
/////////////////////////////////////////////////////////////////////////////////

type GetGlobalPolicyByIDRequest struct {
	PolicyID uint `url:"policy_id"`
}

type GetGlobalPolicyByIDResponse struct {
	Policy *Policy `json:"policy"`
	Err    error   `json:"error,omitempty"`
}

func (r GetGlobalPolicyByIDResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Global Policy - Count
/////////////////////////////////////////////////////////////////////////////////

type CountGlobalPoliciesRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type CountGlobalPoliciesResponse struct {
	Count int   `json:"count"`
	Err   error `json:"error,omitempty"`
}

func (r CountGlobalPoliciesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Global Policy - Delete
/////////////////////////////////////////////////////////////////////////////////

type DeleteGlobalPoliciesRequest struct {
	IDs []uint `json:"ids"`
}

type DeleteGlobalPoliciesResponse struct {
	Deleted []uint `json:"deleted,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r DeleteGlobalPoliciesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Global Policy - Modify
/////////////////////////////////////////////////////////////////////////////////

type ModifyGlobalPolicyRequest struct {
	PolicyID uint `url:"policy_id"`
	ModifyPolicyPayload
}

type ModifyGlobalPolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r ModifyGlobalPolicyResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Reset automation
/////////////////////////////////////////////////////////////////////////////////

type ResetAutomationRequest struct {
	TeamIDs   []uint `json:"team_ids" premium:"true" renameto:"fleet_ids"`
	PolicyIDs []uint `json:"policy_ids"`
}

type ResetAutomationResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ResetAutomationResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Apply Policy Spec
/////////////////////////////////////////////////////////////////////////////////

type ApplyPolicySpecsRequest struct {
	Specs []*PolicySpec `json:"specs"`
}

type ApplyPolicySpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyPolicySpecsResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Autofill Policies
/////////////////////////////////////////////////////////////////////////////////

type AutofillPoliciesRequest struct {
	SQL string `json:"sql"`
}

type AutofillPoliciesResponse struct {
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
	Err         error  `json:"error,omitempty"`
}

func (r AutofillPoliciesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Team Policy - Add
/////////////////////////////////////////////////////////////////////////////////

type TeamPolicyRequest struct {
	TeamID                   uint     `url:"fleet_id"`
	QueryID                  *uint    `json:"query_id" renameto:"report_id"`
	Query                    string   `json:"query"`
	Name                     string   `json:"name"`
	Description              string   `json:"description"`
	Resolution               string   `json:"resolution"`
	Platform                 string   `json:"platform"`
	Critical                 bool     `json:"critical" premium:"true"`
	CalendarEventsEnabled    bool     `json:"calendar_events_enabled"`
	SoftwareTitleID          *uint    `json:"software_title_id"`
	ScriptID                 *uint    `json:"script_id"`
	LabelsIncludeAny         []string `json:"labels_include_any"`
	LabelsIncludeAll         []string `json:"labels_include_all" premium:"true"`
	LabelsExcludeAny         []string `json:"labels_exclude_any"`
	ConditionalAccessEnabled bool     `json:"conditional_access_enabled"`
	Type                     *string  `json:"type"`
	PatchSoftwareTitleID     *uint    `json:"patch_software_title_id"`
}

type TeamPolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r TeamPolicyResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Team Policy - List
/////////////////////////////////////////////////////////////////////////////////

type ListTeamPoliciesRequest struct {
	TeamID                  uint           `url:"fleet_id"`
	Opts                    ListOptions    `url:"list_options"`
	InheritedPage           uint           `query:"inherited_page,optional"`
	InheritedPerPage        uint           `query:"inherited_per_page,optional"`
	InheritedOrderDirection OrderDirection `query:"inherited_order_direction,optional"`
	InheritedOrderKey       string         `query:"inherited_order_key,optional"`
	MergeInherited          bool           `query:"merge_inherited,optional"`
	AutomationType          string         `query:"automation_type,optional"`
}

type ListTeamPoliciesResponse struct {
	Policies          []*Policy `json:"policies,omitempty"`
	InheritedPolicies []*Policy `json:"inherited_policies,omitempty"`
	Err               error     `json:"error,omitempty"`
}

func (r ListTeamPoliciesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Team Policy - Count
/////////////////////////////////////////////////////////////////////////////////

type CountTeamPoliciesRequest struct {
	ListOptions    ListOptions `url:"list_options"`
	TeamID         uint        `url:"fleet_id"`
	MergeInherited bool        `query:"merge_inherited,optional"`
	AutomationType string      `query:"automation_type,optional"`
}

type CountTeamPoliciesResponse struct {
	Count                int   `json:"count"`
	InheritedPolicyCount int   `json:"inherited_policy_count"`
	Err                  error `json:"error,omitempty"`
}

func (r CountTeamPoliciesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Team Policy - Get by id
/////////////////////////////////////////////////////////////////////////////////

type GetTeamPolicyByIDRequest struct {
	TeamID   uint `url:"fleet_id"`
	PolicyID uint `url:"policy_id"`
}

type GetTeamPolicyByIDResponse struct {
	Policy *Policy `json:"policy"`
	Err    error   `json:"error,omitempty"`
}

func (r GetTeamPolicyByIDResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Team Policy - Delete
/////////////////////////////////////////////////////////////////////////////////

type DeleteTeamPoliciesRequest struct {
	TeamID uint   `url:"fleet_id"`
	IDs    []uint `json:"ids"`
}

type DeleteTeamPoliciesResponse struct {
	Deleted []uint `json:"deleted,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r DeleteTeamPoliciesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Team Policy - Modify
/////////////////////////////////////////////////////////////////////////////////

type ModifyTeamPolicyRequest struct {
	TeamID   uint `url:"fleet_id"`
	PolicyID uint `url:"policy_id"`
	ModifyPolicyPayload
}

type ModifyTeamPolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r ModifyTeamPolicyResponse) Error() error { return r.Err }
