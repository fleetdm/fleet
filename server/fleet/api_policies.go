package fleet

import "encoding/json"

/////////////////////////////////////////////////////////////////////////////////
// Policy - Get by id
/////////////////////////////////////////////////////////////////////////////////

type GetPolicyByIDRequest struct {
	PolicyID uint `url:"policy_id"`
}

type GetPolicyByIDResponse struct {
	Policy *Policy `json:"policy"`
	Err    error   `json:"error,omitempty"`
}

func (r GetPolicyByIDResponse) Error() error { return r.Err }

func (r GetPolicyByIDResponse) MarshalJSON() ([]byte, error) {
	type response GetPolicyByIDResponse
	return json.Marshal(struct {
		response
		Policy *apiPolicy `json:"policy"`
	}{response(r), newAPIPolicy(r.Policy)})
}

/////////////////////////////////////////////////////////////////////////////////
// Policy - Modify (any scope)
/////////////////////////////////////////////////////////////////////////////////

type ModifyPolicyRequest struct {
	PolicyID uint `url:"policy_id"`
	ModifyPolicyPayload
}

type ModifyPolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r ModifyPolicyResponse) Error() error { return r.Err }

func (r ModifyPolicyResponse) MarshalJSON() ([]byte, error) {
	type response ModifyPolicyResponse
	return json.Marshal(struct {
		response
		Policy *apiPolicy `json:"policy,omitempty"`
	}{response(r), newAPIPolicy(r.Policy)})
}

/////////////////////////////////////////////////////////////////////////////////
// Reset policy
/////////////////////////////////////////////////////////////////////////////////

type ResetPolicyRequest struct {
	PolicyID uint  `url:"policy_id"`
	HostID   *uint `query:"host_id,optional"`
}

type ResetPolicyResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ResetPolicyResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Reset automation
/////////////////////////////////////////////////////////////////////////////////

type ResetAutomationRequest struct {
	TeamIDs   []uint `json:"fleet_ids" premium:"true"`
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

/////////////////////////////////////////////////////////////////////////////////
// Policies in any scope
/////////////////////////////////////////////////////////////////////////////////

// ListPoliciesRequest lists the policies of a scope: no fleet_id is "All fleets", 0 is "Unassigned".
type ListPoliciesRequest struct {
	Opts           ListOptions      `url:"list_options"`
	TeamID         *uint            `query:"fleet_id,optional"`
	MergeInherited bool             `query:"merge_inherited,optional"`
	AutomationType PolicyAutomation `query:"automation_type,optional"`
	Type           string           `query:"type,optional"`
	Platform       string           `query:"platform,optional"`
}

type ListPoliciesResponse struct {
	Policies []*Policy `json:"policies"`
	Err      error     `json:"error,omitempty"`
}

func (r ListPoliciesResponse) Error() error { return r.Err }

func (r ListPoliciesResponse) MarshalJSON() ([]byte, error) {
	type response ListPoliciesResponse
	policies := make([]*apiPolicy, 0, len(r.Policies))
	for _, p := range r.Policies {
		policies = append(policies, newAPIPolicy(p))
	}
	if r.Policies == nil {
		policies = nil
	}
	return json.Marshal(struct {
		response
		Policies []*apiPolicy `json:"policies"`
	}{response(r), policies})
}

type CountPoliciesRequest struct {
	ListOptions    ListOptions      `url:"list_options"`
	TeamID         *uint            `query:"fleet_id,optional"`
	MergeInherited bool             `query:"merge_inherited,optional"`
	AutomationType PolicyAutomation `query:"automation_type,optional"`
	Type           string           `query:"type,optional"`
	Platform       string           `query:"platform,optional"`
}

type CountPoliciesResponse struct {
	Count int   `json:"count"`
	Err   error `json:"error,omitempty"`
}

func (r CountPoliciesResponse) Error() error { return r.Err }

// CreatePolicyRequest creates a policy in any scope: no fleet_id is "All fleets", 0 is "Unassigned".
type CreatePolicyRequest struct {
	TeamID *uint `json:"fleet_id"`
	// automation_type replaced the automation booleans, and software_title_id names a patch policy's app.
	_               struct{} `json:"-" removedkey:"query_id,calendar_events_enabled,conditional_access_enabled,patch_software_title_id"`
	Query           string   `json:"query"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Resolution      string   `json:"resolution"`
	Platform        string   `json:"platform"`
	Critical        bool     `json:"critical" premium:"true"`
	SoftwareTitleID *uint    `json:"software_title_id"`
	// SoftwareInstallerID optionally selects which package of the title to install on failure.
	// When omitted, the policy defaults to the title's first-added package. The wire
	// key is `software_package_id`; the endpointer's renameto layer accepts the
	// legacy `software_installer_id` alias and logs a deprecation warning.
	SoftwareInstallerID          *uint             `json:"software_installer_id" renameto:"software_package_id"`
	ScriptID                     *uint             `json:"script_id"`
	ProfileUUID                  *string           `json:"profile_uuid" premium:"true"`
	LabelsIncludeAny             []string          `json:"labels_include_any" premium:"true"`
	LabelsIncludeAll             []string          `json:"labels_include_all" premium:"true"`
	LabelsExcludeAny             []string          `json:"labels_exclude_any" premium:"true"`
	LabelsExcludeAll             []string          `json:"labels_exclude_all" premium:"true"`
	Hidden                       bool              `json:"hidden" premium:"true"`
	ContinuousAutomationsEnabled bool              `json:"continuous_automations_enabled" premium:"true"`
	Type                         *string           `json:"type"`
	PatchWhenClosed              bool              `json:"patch_when_closed" premium:"true"`
	NotifyBeforePatching         bool              `json:"notify_before_patching" premium:"true"`
	PatchDeadline                bool              `json:"patch_deadline" premium:"true"`
	ForcePatch                   *bool             `json:"force_patch" premium:"true"`
	AutomationType               *PolicyAutomation `json:"automation_type"`
}

type CreatePolicyResponse struct {
	Policy *Policy `json:"policy,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r CreatePolicyResponse) Error() error { return r.Err }

func (r CreatePolicyResponse) MarshalJSON() ([]byte, error) {
	type response CreatePolicyResponse
	return json.Marshal(struct {
		response
		Policy *apiPolicy `json:"policy,omitempty"`
	}{response(r), newAPIPolicy(r.Policy)})
}

type DeletePoliciesRequest struct {
	IDs []uint `json:"ids"`
}

type DeletePoliciesResponse struct {
	Deleted []uint `json:"deleted,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r DeletePoliciesResponse) Error() error { return r.Err }
