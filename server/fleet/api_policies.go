package fleet

import "time"

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

type GetPolicyByIDRequest struct {
	PolicyID uint `url:"policy_id"`
}

type GetPolicyByIDResponse struct {
	Policy *Policy `json:"policy"`
	Err    error   `json:"error,omitempty"`
}

func (r GetPolicyByIDResponse) Error() error { return r.Err }

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
	TeamID                       uint     `url:"fleet_id"`
	QueryID                      *uint    `json:"query_id" renameto:"report_id"`
	Query                        string   `json:"query"`
	Name                         string   `json:"name"`
	Description                  string   `json:"description"`
	Resolution                   string   `json:"resolution"`
	Platform                     string   `json:"platform"`
	Critical                     bool     `json:"critical" premium:"true"`
	CalendarEventsEnabled        bool     `json:"calendar_events_enabled"`
	SoftwareTitleID              *uint    `json:"software_title_id"`
	ScriptID                     *uint    `json:"script_id"`
	LabelsIncludeAny             []string `json:"labels_include_any"`
	LabelsIncludeAll             []string `json:"labels_include_all" premium:"true"`
	LabelsExcludeAny             []string `json:"labels_exclude_any"`
	ConditionalAccessEnabled     bool     `json:"conditional_access_enabled"`
	ContinuousAutomationsEnabled bool     `json:"continuous_automations_enabled" premium:"true"`
	Type                         *string  `json:"type"`
	PatchSoftwareTitleID         *uint    `json:"patch_software_title_id"`
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

/////////////////////////////////////////////////////////////////////////////////
// Policy Status - Get
/////////////////////////////////////////////////////////////////////////////////

type GetPolicyStatusRequest struct {
	PolicyID uint `url:"policy_id"`

	// HostNameQuery is a case-insensitive substring filter on the host hostname.
	HostNameQuery string `query:"hostname,optional"`

	// RunStatus, when set, must be one of:
	//   policy_failed     — host's current pass/fail state is failing
	//   automation_failed — at least one linked automation (webhook/jira/...
	//                       script/software) has a failure outcome
	RunStatus string `query:"run_status,optional"`

	ListOptions ListOptions `url:"list_options"`
}

type GetPolicyStatusResponse struct {
	Runs  []GetPolicyStatusPolicyRun `json:"runs,omitempty"`
	Count int                        `json:"count"`
	Meta  *PaginationMetadata        `json:"meta"`
	Err   error                      `json:"error,omitempty"`
}

func (r GetPolicyStatusResponse) Error() error { return r.Err }

type GetPolicyStatusPolicyRun struct {
	HostID               uint                                 `json:"host_id" db:"host_id"`
	HostName             string                               `json:"host_name" db:"host_name"`
	NewStatus            bool                                 `json:"new_status" db:"new_status"`
	ConsecutiveFailures  uint                                 `json:"consecutive_failures" db:"consecutive_failures"`
	CreatedAt            time.Time                            `json:"created_at" db:"created_at"`
	AutomationExecutions []GetPolicyStatusAutomationExecution `json:"automation_executions,omitempty" db:"-"`
}
type GetPolicyStatusAutomationExecution struct {
	// What type of automation this is:
	// - webook
	// - jira
	// - zendesk
	// - calendar_event
	// - software_installation
	// - script_run
	// - conditional_access
	Type string `json:"type"`

	// The status of the automation:
	// - success
	// - failed
	// - queued
	// - not_compatible: If the automation can't run because
	// it doesn't match the host's platform for example.
	// - not_in_target: If the software is scoped via a label
	//   and the host is not within that scope.
	Status       string `json:"status"`
	ErrorMessage string `json:"error"`
	// Name is the human-readable identifier for the automation resource.
	// Populated for script_run (script name) and software_installation (software title).
	// Empty for webhook/jira/zendesk/calendar/conditional_access automations.
	Name string `json:"name,omitempty"`
}
