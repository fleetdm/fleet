package fleet

type TeamPolicyRequest struct {
	TeamID                         uint     `url:"fleet_id"`
	QueryID                        *uint    `json:"query_id" renameto:"report_id"`
	Query                          string   `json:"query"`
	Name                           string   `json:"name"`
	Description                    string   `json:"description"`
	Resolution                     string   `json:"resolution"`
	Platform                       string   `json:"platform"`
	Critical                       bool     `json:"critical" premium:"true"`
	CalendarEventsEnabled          bool     `json:"calendar_events_enabled"`
	SoftwareTitleID                *uint    `json:"software_title_id"`
	ScriptID                       *uint    `json:"script_id"`
	LabelsIncludeAny               []string `json:"labels_include_any"`
	LabelsExcludeAny               []string `json:"labels_exclude_any"`
	ConditionalAccessEnabled       bool     `json:"conditional_access_enabled"`
	ConditionalAccessBypassEnabled *bool    `json:"conditional_access_bypass_enabled"`
}

type TeamPolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r TeamPolicyResponse) Error() error { return r.Err }

type ListTeamPoliciesRequest struct {
	TeamID                  uint           `url:"fleet_id"`
	Opts                    ListOptions    `url:"list_options"`
	InheritedPage           uint           `query:"inherited_page,optional"`
	InheritedPerPage        uint           `query:"inherited_per_page,optional"`
	InheritedOrderDirection OrderDirection `query:"inherited_order_direction,optional"`
	InheritedOrderKey       string         `query:"inherited_order_key,optional"`
	MergeInherited          bool           `query:"merge_inherited,optional"`
}

type ListTeamPoliciesResponse struct {
	Policies          []*Policy `json:"policies,omitempty"`
	InheritedPolicies []*Policy `json:"inherited_policies,omitempty"`
	Err               error     `json:"error,omitempty"`
}

func (r ListTeamPoliciesResponse) Error() error { return r.Err }

type CountTeamPoliciesRequest struct {
	ListOptions    ListOptions `url:"list_options"`
	TeamID         uint        `url:"fleet_id"`
	MergeInherited bool        `query:"merge_inherited,optional"`
}

type CountTeamPoliciesResponse struct {
	Count                int   `json:"count"`
	InheritedPolicyCount int   `json:"inherited_policy_count"`
	Err                  error `json:"error,omitempty"`
}

func (r CountTeamPoliciesResponse) Error() error { return r.Err }

type GetTeamPolicyByIDRequest struct {
	TeamID   uint `url:"fleet_id"`
	PolicyID uint `url:"policy_id"`
}

type GetTeamPolicyByIDResponse struct {
	Policy *Policy `json:"policy"`
	Err    error   `json:"error,omitempty"`
}

func (r GetTeamPolicyByIDResponse) Error() error { return r.Err }

type DeleteTeamPoliciesRequest struct {
	TeamID uint   `url:"fleet_id"`
	IDs    []uint `json:"ids"`
}

type DeleteTeamPoliciesResponse struct {
	Deleted []uint `json:"deleted,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r DeleteTeamPoliciesResponse) Error() error { return r.Err }

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
