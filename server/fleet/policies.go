package fleet

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
)

// PolicyPayload holds data for policy creation.
//
// If QueryID is not nil, then Name, Query and Description are ignored
// (such fields are fetched from the queries table).
type PolicyPayload struct {
	// QueryID allows creating a policy from an existing query.
	//
	// Using QueryID is the old way of creating policies.
	// Use Query, Name and Description instead.
	QueryID *uint
	// Name is the name of the policy (ignored if QueryID != nil).
	Name string
	// Query is the policy query (ignored if QueryID != nil).
	Query string
	// Critical marks the policy as high impact.
	Critical bool
	// Description is the policy description text (ignored if QueryID != nil).
	Description string
	// Resolution indicates the steps needed to solve a failing policy.
	Resolution string
	// Platform is a comma-separated string to indicate the target platforms.
	//
	// Empty string targets all platforms.
	Platform string
	// CalendarEventsEnabled indicates whether calendar events are enabled for the policy.
	//
	// Only applies to team policies.
	CalendarEventsEnabled bool
	// SoftwareInstallerID is the ID of the software installer that will be installed if the policy fails.
	//
	// Only applies to team policies.
	SoftwareInstallerID *uint
	// VPPAppsTeamsID is the team-specific PK of the VPP app that will be installed if the policy fails.
	//
	// Only applies to team policies.
	VPPAppsTeamsID *uint
	// ScriptID is the ID of the script that will be executed if the policy fails.
	//
	// Only applies to team policies.
	ScriptID *uint
	// LabelsIncludeAny scopes the policy to hosts that are members of ANY of the listed labels.
	LabelsIncludeAny []string
	// LabelsIncludeAll scopes the policy to hosts that are members of ALL of the listed labels.
	LabelsIncludeAll []string
	// LabelsExcludeAny scopes the policy to hosts that are NOT members of ANY of the listed labels.
	LabelsExcludeAny []string
	// LabelsExcludeAll scopes the policy to hosts that are NOT members of ALL of the listed labels.
	LabelsExcludeAll []string
	// ConditionalAccessEnabled indicates whether this is a policy used for Microsoft conditional access.
	//
	// Only applies to team policies.
	ConditionalAccessEnabled bool

	// Type is the policy type. It is 'dynamic' by default and 'patch' for patch policies.
	Type string
	// PatchSoftwareTitleID is the title id of the Fleet maintained app checked by a patch policy.
	//
	// Only applies to team policies with the patch type.
	PatchSoftwareTitleID *uint

	// ContinuousAutomationsEnabled indicates whether software/script automations
	// should run on every failing policy result, not just on pass→fail transitions.
	//
	// Only applies to team policies.
	ContinuousAutomationsEnabled bool
}

// NewTeamPolicyPayload holds data for team policy creation.
//
// If QueryID is not nil, then Name, Query and Description are ignored
// (such fields are fetched from the queries table).
type NewTeamPolicyPayload struct {
	// QueryID allows creating a policy from an existing query.
	//
	// Using QueryID is the old way of creating policies.
	// Use Query, Name and Description instead.
	QueryID *uint
	// Name is the name of the policy (ignored if QueryID != nil).
	Name string
	// Query is the policy query (ignored if QueryID != nil).
	Query string
	// Critical marks the policy as high impact.
	Critical bool
	// Description is the policy description text (ignored if QueryID != nil).
	Description string
	// Resolution indicates the steps needed to solve a failing policy.
	Resolution string
	// Platform is a comma-separated string to indicate the target platforms.
	//
	// Empty string targets all platforms.
	Platform string
	// CalendarEventsEnabled indicates whether calendar events are enabled for the policy.
	CalendarEventsEnabled bool
	// SoftwareTitleID is the ID of the software title that will be installed if the policy fails.
	SoftwareTitleID *uint
	// ScriptID is the ID of the script that will be executed if the policy fails.
	ScriptID *uint
	// LabelsIncludeAny scopes the policy to hosts that are members of ANY of the listed labels.
	LabelsIncludeAny []string
	// LabelsIncludeAll scopes the policy to hosts that are members of ALL of the listed labels.
	LabelsIncludeAll []string
	// LabelsExcludeAny scopes the policy to hosts that are NOT members of ANY of the listed labels.
	LabelsExcludeAny []string
	// LabelsExcludeAll scopes the policy to hosts that are NOT members of ALL of the listed labels.
	LabelsExcludeAll []string
	// ConditionalAccessEnabled indicates whether this is a policy used for Microsoft conditional access.
	ConditionalAccessEnabled bool

	// Type is the policy type. It is 'dynamic' by default and 'patch' for patch policies.
	Type *string
	// PatchSoftwareTitleID is the title id of the Fleet maintained app checked by a patch policy.
	PatchSoftwareTitleID *uint
	// ContinuousAutomationsEnabled indicates whether software/script automations
	// should run on every failing policy result, not just on pass→fail transitions.
	ContinuousAutomationsEnabled bool
}

var (
	errPolicyEmptyName                               = errors.New("policy name cannot be empty")
	errPolicyEmptyQuery                              = errors.New("policy query cannot be empty")
	errPolicyIDAndQuerySet                           = errors.New("both fields \"queryID\" and \"query\" cannot be set")
	errPolicyInvalidPlatform                         = errors.New("invalid policy platform")
	ErrPolicyConflictingIncludeLabels                = errors.New("policy can include at most one of labels_include_any or labels_include_all")
	ErrPolicyConflictingExcludeLabels                = errors.New("policy can include at most one of labels_exclude_any or labels_exclude_all")
	errPolicyPatchAndQuerySet                        = errors.New("If the \"type\" is \"patch\", the \"query\" field is not supported.")
	errPolicyPatchAndPlatformSet                     = errors.New("If the \"type\" is \"patch\", the \"platform\" field is not supported.")
	errPolicyPatchNoTitleID                          = errors.New("If the \"type\" is \"patch\", the \"patch_software_title_id\" field is required.")
	errPatchPolicyRequiresTeam                       = errors.New("If the \"type\" is \"patch\", the \"team\" field is required.")
	errPolicyQueryUpdated                            = errors.New("\"query\" can't be updated")
	errPolicyPlatformUpdated                         = errors.New("\"platform\" can't be updated")
	errPolicyConditionalAccessEnabledInvalidPlatform = errors.New("\"conditional_access_enabled\" is only valid on \"darwin\" and \"windows\" policies")
	errPolicyFMASlugRequiresPatch                    = errors.New("\"fleet_maintained_app_slug\" is only supported for patch policies")
)

// PolicyNoTeamID is the team ID of "No team" policies.
const PolicyNoTeamID = uint(0)

// Max times a policy automation will be retried on failure.
const MaxPolicyAutomationRetries = 3

// Verify verifies the policy payload is valid.
func (p PolicyPayload) Verify() error {
	if p.Type == PolicyTypePatch {
		if p.QueryID != nil {
			return errPolicyPatchAndQuerySet
		}
		if !emptyString(p.Query) {
			return errPolicyPatchAndQuerySet
		}
		if !emptyString(p.Platform) {
			return errPolicyPatchAndPlatformSet
		}
		if p.PatchSoftwareTitleID == nil {
			return errPolicyPatchNoTitleID
		}
		if err := verifyPolicyLabelScopes(p.LabelsIncludeAny, p.LabelsIncludeAll, p.LabelsExcludeAny, p.LabelsExcludeAll); err != nil {
			return err
		}
		return nil
	}

	if p.QueryID != nil {
		if p.Query != "" {
			return errPolicyIDAndQuerySet
		}
	} else {
		if err := verifyPolicyName(p.Name); err != nil {
			return err
		}
		if err := verifyPolicyQuery(p.Query, p.Type); err != nil {
			return err
		}
	}
	if err := verifyPolicyPlatforms(p.Platform); err != nil {
		return err
	}
	if err := PolicyVerifyConditionalAccess(p.ConditionalAccessEnabled, p.Platform); err != nil {
		return err
	}

	return verifyPolicyLabelScopes(p.LabelsIncludeAny, p.LabelsIncludeAll, p.LabelsExcludeAny, p.LabelsExcludeAll)
}

// verifyPolicyLabelScopes enforces the policy label-targeting rules: at most one
// include scope (labels_include_any or labels_include_all) and at most one
// exclude scope (labels_exclude_any or labels_exclude_all) may carry values, and
// no label may appear in both an include and an exclude list. An include scope
// and an exclude scope may be combined. Empty slices ([]) are treated as "no
// value", so e.g. {LabelsIncludeAny: [], LabelsIncludeAll: [A]} is valid.
func verifyPolicyLabelScopes(includeAny, includeAll, excludeAny, excludeAll []string) error {
	includeScopes := 0
	if len(includeAny) > 0 {
		includeScopes++
	}
	if len(includeAll) > 0 {
		includeScopes++
	}
	if includeScopes > 1 {
		return ErrPolicyConflictingIncludeLabels
	}

	excludeScopes := 0
	if len(excludeAny) > 0 {
		excludeScopes++
	}
	if len(excludeAll) > 0 {
		excludeScopes++
	}
	if excludeScopes > 1 {
		return ErrPolicyConflictingExcludeLabels
	}

	include := slices.Concat(includeAny, includeAll)
	exclude := slices.Concat(excludeAny, excludeAll)
	if overlap := LabelOverlap(include, exclude); overlap != "" {
		return fmt.Errorf("label %q cannot appear in both an include and an exclude list", overlap)
	}
	return nil
}

func verifyPolicyName(name string) error {
	if emptyString(name) {
		return errPolicyEmptyName
	}
	return nil
}

func emptyString(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func verifyPolicyQuery(query string, typ string) error {
	if emptyString(query) && typ != PolicyTypePatch {
		return errPolicyEmptyQuery
	}
	return nil
}

func verifyPolicyPlatforms(platforms string) error {
	if platforms == "" {
		return nil
	}
	for _, s := range strings.Split(platforms, ",") {
		switch strings.TrimSpace(s) {
		case "windows", "linux", "darwin", "chrome":
			// OK
		default:
			return errPolicyInvalidPlatform
		}
	}
	return nil
}

func verifyPatchPolicy(team string, typ string) error {
	if typ == PolicyTypePatch && emptyString(team) {
		return errPatchPolicyRequiresTeam
	}
	return nil
}

func PolicyVerifyConditionalAccess(conditionalAccessEnabled bool, platform string) error {
	if conditionalAccessEnabled && !strings.Contains(platform, "darwin") && !strings.Contains(platform, "windows") {
		return errPolicyConditionalAccessEnabledInvalidPlatform
	}
	return nil
}

// ModifyPolicyPayload holds data for policy modification.
type ModifyPolicyPayload struct {
	// Name is the name of the policy.
	Name *string `json:"name"`
	// Query is the policy query.
	Query *string `json:"query"`
	// Description is the policy description text.
	Description *string `json:"description"`
	// Resolution indicate the steps needed to solve a failing policy.
	Resolution *string `json:"resolution"`
	// Platform is a comma-separated string to indicate the target platforms.
	// If non-nil, empty string targets all platforms.
	Platform *string `json:"platform"`
	// Critical marks the policy as high impact.
	Critical *bool `json:"critical" premium:"true"`
	// CalendarEventsEnabled indicates whether calendar events are enabled for the policy.
	//
	// Only applies to team policies.
	CalendarEventsEnabled *bool `json:"calendar_events_enabled" premium:"true"`
	// SoftwareTitleID is the ID of the software title that will be installed if the policy fails.
	// Value 0 will unset the current installer from the policy.
	//
	// Only applies to team policies.
	SoftwareTitleID optjson.Any[uint] `json:"software_title_id" premium:"true"`
	// ScriptID is the ID of the script that will be executed if the policy fails.
	// Value 0 will unset the current script from the policy.
	//
	// Only applies to team policies.
	ScriptID optjson.Any[uint] `json:"script_id" premium:"true"`
	// LabelsIncludeAny scopes the policy to hosts that are members of ANY of the listed labels.
	LabelsIncludeAny []string `json:"labels_include_any" premium:"true"`
	// LabelsIncludeAll scopes the policy to hosts that are members of ALL of the listed labels.
	LabelsIncludeAll []string `json:"labels_include_all" premium:"true"`
	// LabelsExcludeAny scopes the policy to hosts that are NOT members of ANY of the listed labels.
	LabelsExcludeAny []string `json:"labels_exclude_any" premium:"true"`
	// LabelsExcludeAll scopes the policy to hosts that are NOT members of ALL of the listed labels.
	LabelsExcludeAll []string `json:"labels_exclude_all" premium:"true"`
	// ConditionalAccessEnabled indicates whether this is a policy used for Microsoft conditional access.
	//
	// Only applies to team policies.
	ConditionalAccessEnabled *bool `json:"conditional_access_enabled" premium:"true"`
	// ContinuousAutomationsEnabled indicates whether software/script automations
	// should run on every failing policy result, not just on pass→fail transitions.
	//
	// Only applies to team policies.
	ContinuousAutomationsEnabled *bool `json:"continuous_automations_enabled" premium:"true"`

	// Type is the policy type. It is 'dynamic' by default and 'patch' for patch policies.
	Type string `json:"-"`
}

// Verify verifies the policy payload is valid.
func (p ModifyPolicyPayload) Verify() error {
	if p.Type == PolicyTypePatch {
		if p.Name != nil {
			if err := verifyPolicyName(*p.Name); err != nil {
				return err
			}
		}
		if p.Query != nil {
			return errPolicyQueryUpdated
		}
		if p.Platform != nil {
			return errPolicyPlatformUpdated
		}
		return verifyPolicyLabelScopes(p.LabelsIncludeAny, p.LabelsIncludeAll, p.LabelsExcludeAny, p.LabelsExcludeAll)
	}

	if p.Name != nil {
		if err := verifyPolicyName(*p.Name); err != nil {
			return err
		}
	}
	if p.Query != nil {
		if err := verifyPolicyQuery(*p.Query, PolicyTypeDynamic); err != nil {
			return err
		}
	}
	if p.Platform != nil {
		if err := verifyPolicyPlatforms(*p.Platform); err != nil {
			return err
		}
	}
	return verifyPolicyLabelScopes(p.LabelsIncludeAny, p.LabelsIncludeAll, p.LabelsExcludeAny, p.LabelsExcludeAll)
}

// PolicyData holds data of a fleet policy.
type PolicyData struct {
	// ID is the unique ID of a policy.
	ID uint `json:"id"`
	// Name is the name of the policy query.
	Name string `json:"name" db:"name"`
	// Query is the actual query to run on the osquery agents.
	Query string `json:"query" db:"query"`
	// Critical marks the policy as high impact.
	Critical bool `json:"critical" db:"critical"`
	// Description describes the policy.
	Description string `json:"description" db:"description"`
	// AuthorID is the ID of the author of the policy.
	//
	// AuthorID is nil if the author is deleted from the system
	AuthorID *uint `json:"author_id" db:"author_id"`
	// AuthorName is retrieved with a join to the users table in the MySQL backend (using AuthorID).
	AuthorName string `json:"author_name" db:"author_name"`
	// AuthorEmail is retrieved with a join to the users table in the MySQL backend (using AuthorID).
	AuthorEmail string `json:"author_email" db:"author_email"`
	// TeamID is the ID of the team the policy belongs to.
	// If TeamID is nil, then this is a global policy.
	TeamID *uint `json:"team_id" renameto:"fleet_id" db:"team_id"`
	// Resolution describes how to solve a failing policy.
	Resolution *string `json:"resolution,omitempty" db:"resolution"`
	// Platform is a comma-separated string to indicate the target platforms.
	//
	// Empty string targets all platforms.
	Platform string `json:"platform" db:"platforms"`

	// LabelsIncludeAny scopes the policy to hosts that are members of ANY of the listed labels.
	LabelsIncludeAny []LabelIdent `json:"labels_include_any,omitempty"`
	// LabelsIncludeAll scopes the policy to hosts that are members of ALL of the listed labels.
	LabelsIncludeAll []LabelIdent `json:"labels_include_all,omitempty"`
	// LabelsExcludeAny scopes the policy to hosts that are NOT members of ANY of the listed labels.
	LabelsExcludeAny []LabelIdent `json:"labels_exclude_any,omitempty"`
	// LabelsExcludeAll scopes the policy to hosts that are NOT members of ALL of the listed labels.
	LabelsExcludeAll []LabelIdent `json:"labels_exclude_all,omitempty"`

	// CalendarEventsEnabled indicates whether calendar events are enabled for the policy.
	//
	// Only applies to team policies.
	CalendarEventsEnabled bool  `json:"calendar_events_enabled" db:"calendar_events_enabled"`
	SoftwareInstallerID   *uint `json:"-" db:"software_installer_id"`
	VPPAppsTeamsID        *uint `json:"-" db:"vpp_apps_teams_id"`
	ScriptID              *uint `json:"-" db:"script_id"`

	// ConditionalAccessEnabled indicates whether this is a policy used for Microsoft conditional access.
	//
	// Only applies to team policies.
	ConditionalAccessEnabled bool `json:"conditional_access_enabled" db:"conditional_access_enabled"`

	// Type is the policy type. It is 'dynamic' by default and 'patch' for patch policies.
	Type string `json:"type" db:"type"`
	// PatchSoftwareTitleID is the title id of the Fleet maintained app chcked by a patch policy.
	//
	// Only applies to team policies with the patch type.
	PatchSoftwareTitleID *uint `json:"-" db:"patch_software_title_id"`

	// ContinuousAutomationsEnabled indicates whether software/script automations
	// should run on every failing policy result, not just on pass→fail transitions.
	//
	// Only applies to team policies.
	ContinuousAutomationsEnabled bool `json:"continuous_automations_enabled" db:"continuous_automations_enabled"`

	UpdateCreateTimestamps
}

// VerifyLabelScopes checks that the policy's label scopes are valid: at most one
// include scope (any/all) combined with at most one exclude scope (any/all),
// with no label appearing in both an include and an exclude list.
func (p PolicyData) VerifyLabelScopes() error {
	return verifyPolicyLabelScopes(
		LabelIdentsToNames(p.LabelsIncludeAny),
		LabelIdentsToNames(p.LabelsIncludeAll),
		LabelIdentsToNames(p.LabelsExcludeAny),
		LabelIdentsToNames(p.LabelsExcludeAll),
	)
}

// Policy is a fleet's policy query.
type Policy struct {
	PolicyData

	// PassingHostCount is the number of hosts this policy passes on.
	PassingHostCount uint `json:"passing_host_count" db:"passing_host_count"`
	// FailingHostCount is the number of hosts this policy fails on.
	FailingHostCount   uint       `json:"failing_host_count" db:"failing_host_count"`
	HostCountUpdatedAt *time.Time `json:"host_count_updated_at" db:"host_count_updated_at"`

	// InstallSoftware is used to trigger installation of a software title
	// when this policy fails.
	//
	// Only applies to team policies.
	//
	// This field is populated from PolicyData.SoftwareInstallerID.
	InstallSoftware *PolicySoftwareTitle `json:"install_software,omitempty"`

	// RunScript is used to trigger script execution when this policy fails.
	//
	// Only applies to team policies.
	//
	// This field is populated from PolicyData.ScriptID
	RunScript *PolicyScript `json:"run_script,omitempty"`

	// PatchSoftware is used to check the installed version of a Fleet
	// maintaind app.
	//
	// Only applies to team policies with the patch type.
	//
	// This field is populated from PolicyData.PatchSoftwareTitleID
	PatchSoftware *PolicySoftwareTitle `json:"patch_software,omitempty"`
}

type PolicyCalendarData struct {
	ID   uint   `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

type PolicySoftwareInstallerData struct {
	ID                           uint `db:"id"`
	InstallerID                  uint `db:"software_installer_id"`
	ContinuousAutomationsEnabled bool `db:"continuous_automations_enabled"`
}

type PolicyVPPData struct {
	ID                           uint                      `db:"id"`
	AdamID                       string                    `db:"adam_id"`
	Platform                     InstallableDevicePlatform `db:"platform"`
	ContinuousAutomationsEnabled bool                      `db:"continuous_automations_enabled"`
}

type PolicyScriptData struct {
	ID                           uint `db:"id"`
	ScriptID                     uint `db:"script_id"`
	ContinuousAutomationsEnabled bool `db:"continuous_automations_enabled"`
}

// PolicyLite is a stripped down version of the policy.
type PolicyLite struct {
	ID uint `db:"id"`
	// Name is the name of the policy.
	Name string `db:"name"`
	// Description describes the policy.
	Description string `db:"description"`
	// Resolution describes how to solve a failing policy.
	Resolution *string `db:"resolution"`
}

func (p Policy) AuthzType() string {
	return "policy"
}

const (
	PolicyKind = "policy"
)

// HostPolicy is a fleet's policy query in the context of a host.
type HostPolicy struct {
	PolicyData

	// Response can be one of the following values:
	//	- "pass": if the policy was executed and passed.
	//	- "fail": if the policy was executed and did not pass.
	//	- "": if the policy did not run yet.
	Response string `json:"response" db:"response"`
}

// PolicySpec is used to hold policy data to apply policy specs.
//
// Policies are currently identified by name (unique).
type PolicySpec struct {
	// Name is the name of the policy.
	Name string `json:"name"`
	// Query is the policy's SQL query.
	Query string `json:"query"`
	// Description describes the policy.
	Description string `json:"description"`
	// Critical marks the policy as high impact.
	Critical bool `json:"critical"`
	// Resolution describes how to solve a failing policy.
	Resolution string `json:"resolution,omitempty"`
	// Team is the name of the team.
	Team string `json:"team,omitempty" renameto:"fleet"`
	// Platform is a comma-separated string to indicate the target platforms.
	//
	// Empty string targets all platforms.
	Platform string `json:"platform,omitempty"`
	// CalendarEventsEnabled indicates whether calendar events are enabled for the policy.
	//
	// Only applies to team policies.
	CalendarEventsEnabled bool `json:"calendar_events_enabled"`
	// SoftwareTitleID is the title ID of the installer associated with this policy (team policies only).
	// When editing a policy, if this is nil or 0 then the title ID is unset from the policy.
	SoftwareTitleID *uint `json:"software_title_id"`
	// ScriptID is the ID of the script associated with this policy (team policies only).
	// When editing a policy, if this is nil or 0 then the script ID is unset from the policy.
	ScriptID         *uint    `json:"script_id"`
	LabelsIncludeAny []string `json:"labels_include_any,omitempty"`
	LabelsIncludeAll []string `json:"labels_include_all,omitempty"`
	LabelsExcludeAny []string `json:"labels_exclude_any,omitempty"`
	LabelsExcludeAll []string `json:"labels_exclude_all,omitempty"`
	// ConditionalAccessEnabled indicates whether this is a policy used for Microsoft conditional access.
	//
	// Only applies to team policies.
	ConditionalAccessEnabled bool `json:"conditional_access_enabled"`
	// ContinuousAutomationsEnabled indicates whether software/script automations
	// should run on every failing policy result, not just on pass→fail transitions.
	//
	// Only applies to team policies.
	ContinuousAutomationsEnabled bool `json:"continuous_automations_enabled"`

	Type                   string `json:"type"`
	FleetMaintainedAppSlug string `json:"fleet_maintained_app_slug"`
	PatchSoftwareTitleID   uint   `json:"-"`
}

// PolicySoftwareTitle contains software title data for policies.
type PolicySoftwareTitle struct {
	// SoftwareTitleID is the ID of the title associated to the policy.
	SoftwareTitleID uint `json:"software_title_id" db:"title_id"`
	// Name is the associated installer title name
	// (not the package name, but the installed software title).
	Name        string `json:"name" db:"name"`
	DisplayName string `json:"display_name" db:"display_name"`
	// IconURL is the API path to this software title's icon in the policy's
	// team. It is set when a custom icon was uploaded for the title, or for VPP
	// apps (whose icon endpoint redirects to the App Store icon), and is nil
	// otherwise.
	IconURL *string `json:"icon_url,omitempty"`
}

// PolicyScript contains script data for policies.
type PolicyScript struct {
	// ID is the ID of the script associated with the policy
	ID uint `json:"id"`
	// Name is the script name
	Name string `json:"name"`
}

// Verify verifies the policy data is valid.
func (p PolicySpec) Verify() error {
	if err := verifyPolicyName(p.Name); err != nil {
		return err
	}
	if err := verifyPolicyQuery(p.Query, p.Type); err != nil {
		return err
	}
	if err := verifyPolicyPlatforms(p.Platform); err != nil {
		return err
	}
	if err := PolicyVerifyConditionalAccess(p.ConditionalAccessEnabled, p.Platform); err != nil {
		return err
	}
	if err := verifyPatchPolicy(p.Team, p.Type); err != nil {
		return err
	}
	if p.Type != PolicyTypePatch && p.FleetMaintainedAppSlug != "" {
		return errPolicyFMASlugRequiresPatch
	}
	return p.VerifyLabelScopes()
}

// VerifyLabelScopes checks that the spec's label scopes are valid: at most one
// include scope (any/all) combined with at most one exclude scope (any/all),
// with no label appearing in both an include and an exclude list.
func (p PolicySpec) VerifyLabelScopes() error {
	return verifyPolicyLabelScopes(p.LabelsIncludeAny, p.LabelsIncludeAll, p.LabelsExcludeAny, p.LabelsExcludeAll)
}

// FirstDuplicatePolicySpecName returns first duplicate name of policies (in a team) or empty string if no duplicates found
func FirstDuplicatePolicySpecName(specs []*PolicySpec) string {
	teams := make(map[string]map[string]struct{})
	for _, spec := range specs {
		if team, ok := teams[spec.Team]; ok {
			if _, ok = team[spec.Name]; ok {
				return spec.Name
			}
			team[spec.Name] = struct{}{}
		} else {
			teams[spec.Team] = map[string]struct{}{spec.Name: {}}
		}
	}
	return ""
}

// FailingPolicySet holds sets of hosts that failed policy executions.
type FailingPolicySet interface {
	// ListSets lists all the policy sets.
	ListSets() ([]uint, error)
	// AddHost adds the given host to the policy set.
	AddHost(policyID uint, host PolicySetHost) error
	// ListHosts returns the list of hosts present in the policy set.
	ListHosts(policyID uint) ([]PolicySetHost, error)
	// RemoveHosts removes the hosts from the policy set.
	RemoveHosts(policyID uint, hosts []PolicySetHost) error
	// RemoveSet removes a policy set.
	RemoveSet(policyID uint) error
}

// PolicySetHost is a host entry for a policy set.
type PolicySetHost struct {
	// ID is the identifier of the host.
	ID uint
	// Hostname is the host's name.
	Hostname string
	// DisplayName is the ComputerName if it exists, or the Hostname otherwise.
	DisplayName string
}

type PolicyMembershipResult struct {
	HostID   uint
	PolicyID uint
	Passes   *bool
}

const (
	PolicyTypeDynamic = "dynamic"
	PolicyTypePatch   = "patch"
)
