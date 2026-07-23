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
	LabelsIncludeAny []string `json:"labels_include_any" premium:"true"`
	LabelsIncludeAll []string `json:"labels_include_all" premium:"true"`
	LabelsExcludeAny []string `json:"labels_exclude_any" premium:"true"`
	LabelsExcludeAll []string `json:"labels_exclude_all" premium:"true"`
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
	Opts     ListOptions `url:"list_options"`
	Platform string      `query:"platform,optional"`
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
	Platform    string      `query:"platform,optional"`
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
// Reset policy
/////////////////////////////////////////////////////////////////////////////////

type ResetPolicyRequest struct {
	PolicyID uint `url:"policy_id"`
}

type ResetPolicyResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ResetPolicyResponse) Error() error { return r.Err }

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
	TeamID                uint   `url:"fleet_id"`
	QueryID               *uint  `json:"query_id" renameto:"report_id"`
	Query                 string `json:"query"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	Resolution            string `json:"resolution"`
	Platform              string `json:"platform"`
	Critical              bool   `json:"critical" premium:"true"`
	CalendarEventsEnabled bool   `json:"calendar_events_enabled"`
	SoftwareTitleID       *uint  `json:"software_title_id"`
	// SoftwareInstallerID optionally selects which package of the title to install on failure.
	// When omitted, the policy defaults to the title's first-added package.
	SoftwareInstallerID          *uint    `json:"software_installer_id"`
	ScriptID                     *uint    `json:"script_id"`
	LabelsIncludeAny             []string `json:"labels_include_any" premium:"true"`
	LabelsIncludeAll             []string `json:"labels_include_all" premium:"true"`
	LabelsExcludeAny             []string `json:"labels_exclude_any" premium:"true"`
	LabelsExcludeAll             []string `json:"labels_exclude_all" premium:"true"`
	ConditionalAccessEnabled     bool     `json:"conditional_access_enabled"`
	ContinuousAutomationsEnabled bool     `json:"continuous_automations_enabled" premium:"true"`
	PatchWhenClosed              bool     `json:"patch_when_closed" premium:"true"`
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
	Platform                string         `query:"platform,optional"`
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
	Platform       string      `query:"platform,optional"`
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
// Policy Automation Activities - List
/////////////////////////////////////////////////////////////////////////////////

// PolicyAutomationActivity is a fleet.Activity enriched with the host it
// belongs to, as recorded in activity_host_past.
type PolicyAutomationActivity struct {
	Activity
	HostID          uint   `json:"host_id" db:"host_id"`
	HostDisplayName string `json:"host_display_name" db:"host_display_name"`
	// Status is the outcome of the activity: "error" or "success". It is set for
	// every activity, including the named automations (webhook/ticket/calendar/CA)
	// whose outcome is otherwise only encoded in the activity type.
	Status string `json:"status" db:"status"`
	// Output is the combined script output for ran_script activities and the
	// install-script output for installed_software activities. It is null for
	// named automation and VPP (installed_app_store_app) activities, which carry
	// no script output.
	Output *string `json:"output" db:"output"`
	// PreInstallOutput and PostInstallOutput are the pre-install query output and
	// post-install script output for installed_software activities (a software
	// install can fail at any of the three stages). They are null for every other
	// activity type.
	PreInstallOutput  *string `json:"pre_install_output" db:"pre_install_output"`
	PostInstallOutput *string `json:"post_install_output" db:"post_install_output"`
}

// ListPolicyAutomationActivitiesRequest is the request type for
// GET /api/_version_/fleet/policies/{policy_id}/automation_activities.
type ListPolicyAutomationActivitiesRequest struct {
	PolicyID uint        `url:"policy_id"`
	Opts     ListOptions `url:"list_options"`
	// Status filters by outcome: "error" (failed_* types), "success" (positive
	// types), or empty (all types). Any other value returns HTTP 422.
	Status string `query:"status,optional"`
}

type ListPolicyAutomationActivitiesResponse struct {
	Activities []*PolicyAutomationActivity `json:"activities"`
	Meta       *PaginationMetadata         `json:"meta"`
	Count      uint                        `json:"count"`
	Err        error                       `json:"error,omitempty"`
}

func (r ListPolicyAutomationActivitiesResponse) Error() error { return r.Err }
