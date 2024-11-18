package fleet

import (
	"context"
	"encoding/json"
)

//go:generate go run gen_activity_doc.go "../../docs/Contributing/Audit-logs.md"

type ContextKey string

// ActivityWebhookContextKey is the context key to indicate that the activity webhook has been processed before saving the activity.
const ActivityWebhookContextKey = ContextKey("ActivityWebhook")

// ActivityDetailsList is used to generate documentation.
var ActivityDetailsList = []ActivityDetails{
	ActivityTypeCreatedPack{},
	ActivityTypeEditedPack{},
	ActivityTypeDeletedPack{},
	ActivityTypeAppliedSpecPack{},

	ActivityTypeCreatedPolicy{},
	ActivityTypeEditedPolicy{},
	ActivityTypeDeletedPolicy{},
	ActivityTypeAppliedSpecPolicy{},
	ActivityTypeCreatedSavedQuery{},
	ActivityTypeEditedSavedQuery{},
	ActivityTypeDeletedSavedQuery{},
	ActivityTypeDeletedMultipleSavedQuery{},
	ActivityTypeAppliedSpecSavedQuery{},

	ActivityTypeCreatedTeam{},
	ActivityTypeDeletedTeam{},
	ActivityTypeAppliedSpecTeam{},
	ActivityTypeTransferredHostsToTeam{},

	ActivityTypeEditedAgentOptions{},

	ActivityTypeLiveQuery{},

	ActivityTypeUserAddedBySSO{},

	ActivityTypeUserLoggedIn{},
	ActivityTypeUserFailedLogin{},

	ActivityTypeCreatedUser{},
	ActivityTypeDeletedUser{},
	ActivityTypeChangedUserGlobalRole{},
	ActivityTypeDeletedUserGlobalRole{},
	ActivityTypeChangedUserTeamRole{},
	ActivityTypeDeletedUserTeamRole{},

	ActivityTypeFleetEnrolled{},
	ActivityTypeMDMEnrolled{},
	ActivityTypeMDMUnenrolled{},

	ActivityTypeEditedMacOSMinVersion{},
	ActivityTypeEditedIOSMinVersion{},
	ActivityTypeEditedIPadOSMinVersion{},
	ActivityTypeEditedWindowsUpdates{},

	ActivityTypeReadHostDiskEncryptionKey{},

	ActivityTypeCreatedMacosProfile{},
	ActivityTypeDeletedMacosProfile{},
	ActivityTypeEditedMacosProfile{},

	ActivityTypeChangedMacosSetupAssistant{},
	ActivityTypeDeletedMacosSetupAssistant{},

	ActivityTypeEnabledMacosDiskEncryption{},
	ActivityTypeDisabledMacosDiskEncryption{},

	ActivityTypeAddedBootstrapPackage{},
	ActivityTypeDeletedBootstrapPackage{},

	ActivityTypeEnabledMacosSetupEndUserAuth{},
	ActivityTypeDisabledMacosSetupEndUserAuth{},

	ActivityTypeEnabledWindowsMDM{},
	ActivityTypeDisabledWindowsMDM{},

	ActivityTypeRanScript{},
	ActivityTypeAddedScript{},
	ActivityTypeDeletedScript{},
	ActivityTypeEditedScript{},

	ActivityTypeCreatedWindowsProfile{},
	ActivityTypeDeletedWindowsProfile{},
	ActivityTypeEditedWindowsProfile{},

	ActivityTypeLockedHost{},
	ActivityTypeUnlockedHost{},
	ActivityTypeWipedHost{},

	ActivityTypeCreatedDeclarationProfile{},
	ActivityTypeDeletedDeclarationProfile{},
	ActivityTypeEditedDeclarationProfile{},

	ActivityTypeResentConfigurationProfile{},

	ActivityTypeInstalledSoftware{},
	ActivityTypeUninstalledSoftware{},
	ActivityTypeAddedSoftware{},
	ActivityTypeEditedSoftware{},
	ActivityTypeDeletedSoftware{},
	ActivityEnabledVPP{},
	ActivityDisabledVPP{},
	ActivityAddedAppStoreApp{},
	ActivityDeletedAppStoreApp{},
	ActivityInstalledAppStoreApp{},

	ActivityAddedNDESSCEPProxy{},
	ActivityDeletedNDESSCEPProxy{},
	ActivityEditedNDESSCEPProxy{},

	ActivityTypeEnabledActivityAutomations{},
	ActivityTypeEditedActivityAutomations{},
	ActivityTypeDisabledActivityAutomations{},
}

type ActivityDetails interface {
	// ActivityName is the name/type of the activity.
	ActivityName() string
	// Documentation is used by "go generate" to generate markdown docs.
	Documentation() (activity string, details string, detailsExample string)
}

// ActivityHosts is the optional additional interface that can be implemented
// by activities that are related to hosts.
type ActivityHosts interface {
	ActivityDetails
	HostIDs() []uint
}

type ActivityTypeEnabledActivityAutomations struct {
	WebhookUrl string `json:"webhook_url"`
}

func (a ActivityTypeEnabledActivityAutomations) ActivityName() string {
	return "enabled_activity_automations"
}

func (a ActivityTypeEnabledActivityAutomations) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when activity automations are enabled`,
		`This activity contains the following field:
- "webhook_url": the URL to broadcast activities to.`, `{
	"webhook_url": "https://example.com/notify"
}`
}

type ActivityTypeEditedActivityAutomations struct {
	WebhookUrl string `json:"webhook_url"`
}

func (a ActivityTypeEditedActivityAutomations) ActivityName() string {
	return "edited_activity_automations"
}

func (a ActivityTypeEditedActivityAutomations) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when activity automations are edited while enabled`,
		`This activity contains the following field:
- "webhook_url": the URL to broadcast activities to, post-edit.`, `{
	"webhook_url": "https://example.com/notify"
}`
}

type ActivityTypeDisabledActivityAutomations struct{}

func (a ActivityTypeDisabledActivityAutomations) ActivityName() string {
	return "disabled_activity_automations"
}

func (a ActivityTypeDisabledActivityAutomations) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when activity automations are disabled`,
		`This activity does not contain any detail fields.`, ""
}

type ActivityTypeCreatedPack struct {
	ID   uint   `json:"pack_id"`
	Name string `json:"pack_name"`
}

func (a ActivityTypeCreatedPack) ActivityName() string {
	return "created_pack"
}

func (a ActivityTypeCreatedPack) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when creating scheduled query packs.`,
		`This activity contains the following fields:
- "pack_id": the id of the created pack.
- "pack_name": the name of the created pack.`, `{
	"pack_id": 123,
	"pack_name": "foo"
}`
}

type ActivityTypeEditedPack struct {
	ID   uint   `json:"pack_id"`
	Name string `json:"pack_name"`
}

func (a ActivityTypeEditedPack) ActivityName() string {
	return "edited_pack"
}

func (a ActivityTypeEditedPack) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when editing scheduled query packs.`,
		`This activity contains the following fields:
- "pack_id": the id of the edited pack.
- "pack_name": the name of the edited pack.`, `{
	"pack_id": 123,
	"pack_name": "foo"
}`
}

type ActivityTypeDeletedPack struct {
	Name string `json:"pack_name"`
}

func (a ActivityTypeDeletedPack) ActivityName() string {
	return "deleted_pack"
}

func (a ActivityTypeDeletedPack) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when deleting scheduled query packs.`,
		`This activity contains the following fields:
- "pack_name": the name of the created pack.`, `{
	"pack_name": "foo"
}`
}

type ActivityTypeAppliedSpecPack struct{}

func (a ActivityTypeAppliedSpecPack) ActivityName() string {
	return "applied_spec_pack"
}

func (a ActivityTypeAppliedSpecPack) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when applying a scheduled query pack spec.`,
		`This activity does not contain any detail fields.`, ""
}

type ActivityTypeCreatedPolicy struct {
	ID   uint   `json:"policy_id"`
	Name string `json:"policy_name"`
}

func (a ActivityTypeCreatedPolicy) ActivityName() string {
	return "created_policy"
}

func (a ActivityTypeCreatedPolicy) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when creating policies.`,
		`This activity contains the following fields:
- "policy_id": the ID of the created policy.
- "policy_name": the name of the created policy.`, `{
	"policy_id": 123,
	"policy_name": "foo"
}`
}

type ActivityTypeEditedPolicy struct {
	ID   uint   `json:"policy_id"`
	Name string `json:"policy_name"`
}

func (a ActivityTypeEditedPolicy) ActivityName() string {
	return "edited_policy"
}

func (a ActivityTypeEditedPolicy) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when editing policies.`,
		`This activity contains the following fields:
- "policy_id": the ID of the edited policy.
- "policy_name": the name of the edited policy.`, `{
	"policy_id": 123,
	"policy_name": "foo"
}`
}

type ActivityTypeDeletedPolicy struct {
	ID   uint   `json:"policy_id"`
	Name string `json:"policy_name"`
}

func (a ActivityTypeDeletedPolicy) ActivityName() string {
	return "deleted_policy"
}

func (a ActivityTypeDeletedPolicy) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when deleting policies.`,
		`This activity contains the following fields:
- "policy_id": the ID of the deleted policy.
- "policy_name": the name of the deleted policy.`, `{
	"policy_id": 123,
	"policy_name": "foo"
}`
}

type ActivityTypeAppliedSpecPolicy struct {
	Policies []*PolicySpec `json:"policies"`
}

func (a ActivityTypeAppliedSpecPolicy) ActivityName() string {
	return "applied_spec_policy"
}

func (a ActivityTypeAppliedSpecPolicy) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when applying policy specs.`,
		`This activity contains a field "policies" where each item is a policy spec with the following fields:
- "name": Name of the applied policy.
- "query": SQL query of the policy.
- "description": Description of the policy.
- "critical": Marks the policy as high impact.
- "resolution": Describes how to solve a failing policy.
- "team": Name of the team this policy belongs to.
- "platform": Comma-separated string to indicate the target platforms.
`, `{
	"policies": [
		{
			"name":"Gatekeeper enabled (macOS)",
			"query":"SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
			"critical":false,
			"platform":"darwin",
			"resolution":"To enable Gatekeeper, on the failing device [...]",
			"description":"Checks to make sure that the Gatekeeper feature is [...]"
		},
		{
			"name":"Full disk encryption enabled (Windows)",
			"query":"SELECT 1 FROM bitlocker_info WHERE drive_letter='C:' AND protection_status=1;",
			"critical":false,
			"platform":"windows",
			"resolution":"To get additional information, run the following osquery [...]",
			"description":"Checks to make sure that full disk encryption is enabled on Windows devices."
		}
	]
}`
}

type ActivityTypeCreatedSavedQuery struct {
	ID   uint   `json:"query_id"`
	Name string `json:"query_name"`
}

func (a ActivityTypeCreatedSavedQuery) ActivityName() string {
	return "created_saved_query"
}

func (a ActivityTypeCreatedSavedQuery) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when creating a new query.`,
		`This activity contains the following fields:
- "query_id": the ID of the created query.
- "query_name": the name of the created query.`, `{
	"query_id": 123,
	"query_name": "foo"
}`
}

type ActivityTypeEditedSavedQuery struct {
	ID   uint   `json:"query_id"`
	Name string `json:"query_name"`
}

func (a ActivityTypeEditedSavedQuery) ActivityName() string {
	return "edited_saved_query"
}

func (a ActivityTypeEditedSavedQuery) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when editing a saved query.`,
		`This activity contains the following fields:
- "query_id": the ID of the query being edited.
- "query_name": the name of the query being edited.`, `{
	"query_id": 123,
	"query_name": "foo"
}`
}

type ActivityTypeDeletedSavedQuery struct {
	Name string `json:"query_name"`
}

func (a ActivityTypeDeletedSavedQuery) ActivityName() string {
	return "deleted_saved_query"
}

func (a ActivityTypeDeletedSavedQuery) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when deleting a saved query.`,
		`This activity contains the following fields:
- "query_name": the name of the query being deleted.`, `{
	"query_name": "foo"
}`
}

type ActivityTypeDeletedMultipleSavedQuery struct {
	IDs []uint `json:"query_ids"`
}

func (a ActivityTypeDeletedMultipleSavedQuery) ActivityName() string {
	return "deleted_multiple_saved_query"
}

func (a ActivityTypeDeletedMultipleSavedQuery) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when deleting multiple saved queries.`,
		`This activity contains the following fields:
- "query_ids": list of IDs of the deleted saved queries.`, `{
	"query_ids": [1, 42, 100]
}`
}

type ActivityTypeAppliedSpecSavedQuery struct {
	Specs []*QuerySpec `json:"specs"`
}

func (a ActivityTypeAppliedSpecSavedQuery) ActivityName() string {
	return "applied_spec_saved_query"
}

func (a ActivityTypeAppliedSpecSavedQuery) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when applying a query spec.`,
		`This activity contains a field "specs" where each item is a query spec with the following fields:
- "name": Name of the query.
- "description": Description of the query.
- "query": SQL query.`, `{
	"specs": [
		{
			"name":"Get OpenSSL versions",
			"query":"SELECT name AS name, version AS version, 'deb_packages' AS source FROM [...]",
			"description":"Retrieves the OpenSSL version."
		}
	]
}`
}

type ActivityTypeCreatedTeam struct {
	ID   uint   `json:"team_id"`
	Name string `json:"team_name"`
}

func (a ActivityTypeCreatedTeam) ActivityName() string {
	return "created_team"
}

func (a ActivityTypeCreatedTeam) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when creating teams.`,
		`This activity contains the following fields:
- "team_id": unique ID of the created team.
- "team_name": the name of the created team.`, `{
	"team_id": 123,
	"team_name": "Workstations"
}`
}

type ActivityTypeDeletedTeam struct {
	ID   uint   `json:"team_id"`
	Name string `json:"team_name"`
}

func (a ActivityTypeDeletedTeam) ActivityName() string {
	return "deleted_team"
}

func (a ActivityTypeDeletedTeam) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when deleting teams.`,
		`This activity contains the following fields:
- "team_id": unique ID of the deleted team.
- "team_name": the name of the deleted team.`, `{
	"team_id": 123,
	"team_name": "Workstations"
}`
}

type TeamActivityDetail struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ActivityTypeAppliedSpecTeam struct {
	Teams []TeamActivityDetail `json:"teams"`
}

func (a ActivityTypeAppliedSpecTeam) ActivityName() string {
	return "applied_spec_team"
}

func (a ActivityTypeAppliedSpecTeam) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when applying team specs.`,
		`This activity contains a field "teams" where each item contains the team details with the following fields:
- "id": Unique ID of the team.
- "name": Name of the team.`, `{
	"teams": [
		{
			"id": 123,
			"name": "foo"
		}
	]
}`
}

type ActivityTypeTransferredHostsToTeam struct {
	TeamID           *uint    `json:"team_id"`
	TeamName         *string  `json:"team_name"`
	HostIDs          []uint   `json:"host_ids"`
	HostDisplayNames []string `json:"host_display_names"`
}

func (a ActivityTypeTransferredHostsToTeam) ActivityName() string {
	return "transferred_hosts"
}

func (a ActivityTypeTransferredHostsToTeam) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user transfers a host (or multiple hosts) to a team (or no team).`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the hosts were transferred to, ` + "`null`" + ` if transferred to no team.
- "team_name": The name of the team that the hosts were transferred to, ` + "`null`" + ` if transferred to no team.
- "host_ids": The list of identifiers of the hosts that were transferred.
- "host_display_names": The list of display names of the hosts that were transferred (in the same order as the "host_ids").`, `{
  "team_id": 123,
  "team_name": "Workstations",
  "host_ids": [1, 2, 3],
  "host_display_names": ["alice-macbook-air", "bob-macbook-pro", "linux-server"]
}`
}

type ActivityTypeEditedAgentOptions struct {
	Global   bool    `json:"global"`
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeEditedAgentOptions) ActivityName() string {
	return "edited_agent_options"
}

func (a ActivityTypeEditedAgentOptions) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when agent options are edited (either globally or for a team).`,
		`This activity contains the following fields:
- "global": "true" if the user updated the global agent options, "false" if the agent options of a team were updated.
- "team_id": unique ID of the team for which the agent options were updated (` + "`null`" + ` if global is true).
- "team_name": the name of the team for which the agent options were updated (` + "`null`" + ` if global is true).`, `{
	"team_id": 123,
	"team_name": "Workstations",
	"global": false
}`
}

type ActivityTypeLiveQuery struct {
	TargetsCount uint             `json:"targets_count"`
	QuerySQL     string           `json:"query_sql"`
	QueryName    *string          `json:"query_name,omitempty"`
	Stats        *AggregatedStats `json:"stats,omitempty"`
}

func (a ActivityTypeLiveQuery) ActivityName() string {
	return "live_query"
}

func (a ActivityTypeLiveQuery) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when running live queries.`,
		`This activity contains the following fields:
- "targets_count": Number of hosts where the live query was targeted to run.
- "query_sql": The SQL query to run on hosts.
- "query_name": Name of the query (this field is not set if this was not a saved query).`, `{
	"targets_count": 5000,
	"query_sql": "SELECT * from osquery_info;",
	"query_name": "foo"
}`
}

type ActivityTypeUserAddedBySSO struct{}

func (a ActivityTypeUserAddedBySSO) ActivityName() string {
	return "user_added_by_sso"
}

func (a ActivityTypeUserAddedBySSO) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when new users are added via SSO JIT provisioning`,
		`This activity does not contain any detail fields.`, ""
}

type Activity struct {
	CreateTimestamp

	// ID is the activity id in the activities table, it is omitted for upcoming
	// activities as those are "virtual activities" generated from entries in
	// queues (e.g. pending host_script_results).
	ID uint `json:"id,omitempty" db:"id"`

	// UUID is the activity UUID for the upcoming activities, as identified in
	// the relevant queue (e.g. pending host_script_results). It is omitted for
	// past activities as those are "real activities" with an activity id.
	UUID string `json:"uuid,omitempty" db:"uuid"`

	ActorFullName *string          `json:"actor_full_name,omitempty" db:"name"`
	ActorID       *uint            `json:"actor_id,omitempty" db:"user_id"`
	ActorGravatar *string          `json:"actor_gravatar,omitempty" db:"gravatar_url"`
	ActorEmail    *string          `json:"actor_email,omitempty" db:"user_email"`
	Type          string           `json:"type" db:"activity_type"`
	Details       *json.RawMessage `json:"details" db:"details"`
	Streamed      *bool            `json:"-" db:"streamed"`
}

// AuthzType implement AuthzTyper to be able to verify access to activities
func (*Activity) AuthzType() string {
	return "activity"
}

type ActivityTypeUserLoggedIn struct {
	PublicIP string `json:"public_ip"`
}

func (a ActivityTypeUserLoggedIn) ActivityName() string {
	return "user_logged_in"
}

func (a ActivityTypeUserLoggedIn) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when users successfully log in to Fleet.`,
		`This activity contains the following fields:
- "public_ip": Public IP of the login request.`, `{
	"public_ip": "168.226.215.82"
}`
}

type ActivityTypeUserFailedLogin struct {
	Email    string `json:"email"`
	PublicIP string `json:"public_ip"`
}

func (a ActivityTypeUserFailedLogin) ActivityName() string {
	return "user_failed_login"
}

func (a ActivityTypeUserFailedLogin) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when users try to log in to Fleet and fail.`,
		`This activity contains the following fields:
- "email": The email used in the login request.
- "public_ip": Public IP of the login request.`, `{
	"email": "foo@example.com",
	"public_ip": "168.226.215.82"
}`
}

type ActivityTypeCreatedUser struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

func (a ActivityTypeCreatedUser) ActivityName() string {
	return "created_user"
}

func (a ActivityTypeCreatedUser) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a user is created.`,
		`This activity contains the following fields:
- "user_id": Unique ID of the created user in Fleet.
- "user_name": Name of the created user.
- "user_email": E-mail of the created user.`, `{
	"user_id": 42,
	"user_name": "Foo",
	"user_email": "foo@example.com"
}`
}

type ActivityTypeDeletedUser struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

func (a ActivityTypeDeletedUser) ActivityName() string {
	return "deleted_user"
}

func (a ActivityTypeDeletedUser) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a user is deleted.`,
		`This activity contains the following fields:
- "user_id": Unique ID of the deleted user in Fleet.
- "user_name": Name of the deleted user.
- "user_email": E-mail of the deleted user.`, `{
	"user_id": 42,
	"user_name": "Foo",
	"user_email": "foo@example.com"
}`
}

type ActivityTypeChangedUserGlobalRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
}

func (a ActivityTypeChangedUserGlobalRole) ActivityName() string {
	return "changed_user_global_role"
}

func (a ActivityTypeChangedUserGlobalRole) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when user global roles are changed.`,
		`This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": New global role of the edited user.`, `{
	"user_id": 42,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Observer"
}`
}

type ActivityTypeDeletedUserGlobalRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	OldRole   string `json:"role"`
}

func (a ActivityTypeDeletedUserGlobalRole) ActivityName() string {
	return "deleted_user_global_role"
}

func (a ActivityTypeDeletedUserGlobalRole) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when user global roles are deleted.`,
		`This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Deleted global role of the edited user.`, `{
	"user_id": 43,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Maintainer"
}`
}

type ActivityTypeChangedUserTeamRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
	TeamID    uint   `json:"team_id"`
	TeamName  string `json:"team_name"`
}

func (a ActivityTypeChangedUserTeamRole) ActivityName() string {
	return "changed_user_team_role"
}

func (a ActivityTypeChangedUserTeamRole) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when user team roles are changed.`,
		`This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Team role set to the edited user.
- "team_id": Unique ID of the team of the changed role.
- "team_name": Name of the team of the changed role.`, `{
	"user_id": 43,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Maintainer",
	"team_id": 5,
	"team_name": "Bar"
}`
}

type ActivityTypeDeletedUserTeamRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
	TeamID    uint   `json:"team_id"`
	TeamName  string `json:"team_name"`
}

func (a ActivityTypeDeletedUserTeamRole) ActivityName() string {
	return "deleted_user_team_role"
}

func (a ActivityTypeDeletedUserTeamRole) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when user team roles are deleted.`,
		`This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Team role deleted from the edited user.
- "team_id": Unique ID of the team of the deleted role.
- "team_name": Name of the team of the deleted role.`, `{
	"user_id": 44,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Observer",
	"team_id": 2,
	"team_name": "Zoo"
}`
}

type ActivityTypeFleetEnrolled struct {
	HostSerial      string `json:"host_serial"`
	HostDisplayName string `json:"host_display_name"`
}

func (a ActivityTypeFleetEnrolled) ActivityName() string {
	return "fleet_enrolled"
}

func (a ActivityTypeFleetEnrolled) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a host is enrolled to Fleet (Fleet's agent fleetd is installed).`,
		`This activity contains the following fields:
- "host_serial": Serial number of the host.
- "host_display_name": Display name of the host.`, `{
  "host_serial": "B04FL3ALPT21",
  "host_display_name": "WIN-DESKTOP-JGS78KJ7C"
}`
}

type ActivityTypeMDMEnrolled struct {
	HostSerial       string `json:"host_serial"`
	HostDisplayName  string `json:"host_display_name"`
	InstalledFromDEP bool   `json:"installed_from_dep"`
	MDMPlatform      string `json:"mdm_platform"`
}

func (a ActivityTypeMDMEnrolled) ActivityName() string {
	return "mdm_enrolled"
}

func (a ActivityTypeMDMEnrolled) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a host is enrolled in Fleet's MDM.`,
		`This activity contains the following fields:
- "host_serial": Serial number of the host (Apple enrollments only, always empty for Microsoft).
- "host_display_name": Display name of the host.
- "installed_from_dep": Whether the host was enrolled via DEP (Apple enrollments only, always false for Microsoft).
- "mdm_platform": Used to distinguish between Apple and Microsoft enrollments. Can be "apple", "microsoft" or not present. If missing, this value is treated as "apple" for backwards compatibility.`, `{
  "host_serial": "C08VQ2AXHT96",
  "host_display_name": "MacBookPro16,1 (C08VQ2AXHT96)",
  "installed_from_dep": true,
  "mdm_platform": "apple"
}`
}

type ActivityTypeMDMUnenrolled struct {
	HostSerial       string `json:"host_serial"`
	HostDisplayName  string `json:"host_display_name"`
	InstalledFromDEP bool   `json:"installed_from_dep"`
}

func (a ActivityTypeMDMUnenrolled) ActivityName() string {
	return "mdm_unenrolled"
}

func (a ActivityTypeMDMUnenrolled) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a host is unenrolled from Fleet's MDM.`,
		`This activity contains the following fields:
- "host_serial": Serial number of the host.
- "host_display_name": Display name of the host.
- "installed_from_dep": Whether the host was enrolled via DEP.`, `{
  "host_serial": "C08VQ2AXHT96",
  "host_display_name": "MacBookPro16,1 (C08VQ2AXHT96)",
  "installed_from_dep": true
}`
}

type ActivityTypeEditedMacOSMinVersion struct {
	TeamID         *uint   `json:"team_id"`
	TeamName       *string `json:"team_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}

func (a ActivityTypeEditedMacOSMinVersion) ActivityName() string {
	return "edited_macos_min_version"
}

func (a ActivityTypeEditedMacOSMinVersion) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when the minimum required macOS version or deadline is modified.`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the minimum macOS version applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the minimum macOS version applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "minimum_version": The minimum macOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.`, `{
  "team_id": 3,
  "team_name": "Workstations",
  "minimum_version": "13.0.1",
  "deadline": "2023-06-01"
}`
}

type ActivityTypeEditedWindowsUpdates struct {
	TeamID          *uint   `json:"team_id"`
	TeamName        *string `json:"team_name"`
	DeadlineDays    *int    `json:"deadline_days"`
	GracePeriodDays *int    `json:"grace_period_days"`
}

func (a ActivityTypeEditedWindowsUpdates) ActivityName() string {
	return "edited_windows_updates"
}

func (a ActivityTypeEditedWindowsUpdates) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when the Windows OS updates deadline or grace period is modified.`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the Windows OS updates settings applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the Windows OS updates settings applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "deadline_days": The number of days before updates are installed, ` + "`null`" + ` if the requirement was removed.
- "grace_period_days": The number of days after the deadline before the host is forced to restart, ` + "`null`" + ` if the requirement was removed.`, `{
  "team_id": 3,
  "team_name": "Workstations",
  "deadline_days": 5,
  "grace_period_days": 2
}`
}

type ActivityTypeEditedIOSMinVersion struct {
	TeamID         *uint   `json:"team_id"`
	TeamName       *string `json:"team_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}

func (a ActivityTypeEditedIOSMinVersion) ActivityName() string {
	return "edited_ios_min_version"
}

func (a ActivityTypeEditedIOSMinVersion) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when the minimum required iOS version or deadline is modified.`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the minimum iOS version applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the minimum iOS version applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "minimum_version": The minimum iOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.`, `{
  "team_id": 3,
  "team_name": "iPhones",
  "minimum_version": "17.5.1",
  "deadline": "2023-06-01"
}`
}

type ActivityTypeEditedIPadOSMinVersion struct {
	TeamID         *uint   `json:"team_id"`
	TeamName       *string `json:"team_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}

func (a ActivityTypeEditedIPadOSMinVersion) ActivityName() string {
	return "edited_ipados_min_version"
}

func (a ActivityTypeEditedIPadOSMinVersion) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when the minimum required iPadOS version or deadline is modified.`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the minimum iPadOS version applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the minimum iPadOS version applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "minimum_version": The minimum iPadOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.`, `{
  "team_id": 3,
  "team_name": "iPads",
  "minimum_version": "17.5.1",
  "deadline": "2023-06-01"
}`
}

type ActivityTypeReadHostDiskEncryptionKey struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
}

func (a ActivityTypeReadHostDiskEncryptionKey) ActivityName() string {
	return "read_host_disk_encryption_key"
}

func (a ActivityTypeReadHostDiskEncryptionKey) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a user reads the disk encryption key for a host.`,
		`This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.`, `{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro"
}`
}

type ActivityTypeCreatedMacosProfile struct {
	ProfileName       string  `json:"profile_name"`
	ProfileIdentifier string  `json:"profile_identifier"`
	TeamID            *uint   `json:"team_id"`
	TeamName          *string `json:"team_name"`
}

func (a ActivityTypeCreatedMacosProfile) ActivityName() string {
	return "created_macos_profile"
}

func (a ActivityTypeCreatedMacosProfile) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user adds a new macOS profile to a team (or no team).`,
		`This activity contains the following fields:
- "profile_name": Name of the profile.
- "profile_identifier": Identifier of the profile.
- "team_id": The ID of the team that the profile applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the profile applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeDeletedMacosProfile struct {
	ProfileName       string  `json:"profile_name"`
	ProfileIdentifier string  `json:"profile_identifier"`
	TeamID            *uint   `json:"team_id"`
	TeamName          *string `json:"team_name"`
}

func (a ActivityTypeDeletedMacosProfile) ActivityName() string {
	return "deleted_macos_profile"
}

func (a ActivityTypeDeletedMacosProfile) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user deletes a macOS profile from a team (or no team).`,
		`This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "profile_identifier": Identifier of deleted the profile.
- "team_id": The ID of the team that the profile applied to, ` + "`null`" + ` if it applied to devices that are not in a team.
- "team_name": The name of the team that the profile applied to, ` + "`null`" + ` if it applied to devices that are not in a team.`, `{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeEditedMacosProfile struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeEditedMacosProfile) ActivityName() string {
	return "edited_macos_profile"
}

func (a ActivityTypeEditedMacosProfile) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user edits the macOS profiles of a team (or no team) via the fleetctl CLI.`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the profiles apply to, ` + "`null`" + ` if they apply to devices that are not in a team.
- "team_name": The name of the team that the profiles apply to, ` + "`null`" + ` if they apply to devices that are not in a team.`, `{
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeChangedMacosSetupAssistant struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeChangedMacosSetupAssistant) ActivityName() string {
	return "changed_macos_setup_assistant"
}

func (a ActivityTypeChangedMacosSetupAssistant) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user sets the macOS setup assistant for a team (or no team).`,
		`This activity contains the following fields:
- "name": Name of the macOS setup assistant file.
- "team_id": The ID of the team that the setup assistant applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the setup assistant applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "name": "dep_profile.json",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeDeletedMacosSetupAssistant struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeDeletedMacosSetupAssistant) ActivityName() string {
	return "deleted_macos_setup_assistant"
}

func (a ActivityTypeDeletedMacosSetupAssistant) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user deletes the macOS setup assistant for a team (or no team).`,
		`This activity contains the following fields:
- "name": Name of the deleted macOS setup assistant file.
- "team_id": The ID of the team that the setup assistant applied to, ` + "`null`" + ` if it applied to devices that are not in a team.
- "team_name": The name of the team that the setup assistant applied to, ` + "`null`" + ` if it applied to devices that are not in a team.`, `{
  "name": "dep_profile.json",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeEnabledMacosDiskEncryption struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeEnabledMacosDiskEncryption) ActivityName() string {
	return "enabled_macos_disk_encryption"
}

func (a ActivityTypeEnabledMacosDiskEncryption) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user turns on macOS disk encryption for a team (or no team).`,
		`This activity contains the following fields:
- "team_id": The ID of the team that disk encryption applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that disk encryption applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeDisabledMacosDiskEncryption struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeDisabledMacosDiskEncryption) ActivityName() string {
	return "disabled_macos_disk_encryption"
}

func (a ActivityTypeDisabledMacosDiskEncryption) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user turns off macOS disk encryption for a team (or no team).`,
		`This activity contains the following fields:
- "team_id": The ID of the team that disk encryption applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that disk encryption applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeAddedBootstrapPackage struct {
	BootstrapPackageName string  `json:"bootstrap_package_name"`
	TeamID               *uint   `json:"team_id"`
	TeamName             *string `json:"team_name"`
}

func (a ActivityTypeAddedBootstrapPackage) ActivityName() string {
	return "added_bootstrap_package"
}

func (a ActivityTypeAddedBootstrapPackage) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user adds a new bootstrap package to a team (or no team).`,
		`This activity contains the following fields:
- "package_name": Name of the package.
- "team_id": The ID of the team that the package applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the package applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "bootstrap_package_name": "bootstrap-package.pkg",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeDeletedBootstrapPackage struct {
	BootstrapPackageName string  `json:"bootstrap_package_name"`
	TeamID               *uint   `json:"team_id"`
	TeamName             *string `json:"team_name"`
}

func (a ActivityTypeDeletedBootstrapPackage) ActivityName() string {
	return "deleted_bootstrap_package"
}

func (a ActivityTypeDeletedBootstrapPackage) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user deletes a bootstrap package from a team (or no team).`,
		`This activity contains the following fields:
- "package_name": Name of the package.
- "team_id": The ID of the team that the package applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the package applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "package_name": "bootstrap-package.pkg",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeEnabledMacosSetupEndUserAuth struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeEnabledMacosSetupEndUserAuth) ActivityName() string {
	return "enabled_macos_setup_end_user_auth"
}

func (a ActivityTypeEnabledMacosSetupEndUserAuth) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user turns on end user authentication for macOS hosts that automatically enroll to a team (or no team).`,
		`This activity contains the following fields:
- "team_id": The ID of the team that end user authentication applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that end user authentication applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeDisabledMacosSetupEndUserAuth struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeDisabledMacosSetupEndUserAuth) ActivityName() string {
	return "disabled_macos_setup_end_user_auth"
}

func (a ActivityTypeDisabledMacosSetupEndUserAuth) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user turns off end user authentication for macOS hosts that automatically enroll to a team (or no team).`,
		`This activity contains the following fields:
- "team_id": The ID of the team that end user authentication applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that end user authentication applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeEnabledWindowsMDM struct{}

func (a ActivityTypeEnabledWindowsMDM) ActivityName() string {
	return "enabled_windows_mdm"
}

func (a ActivityTypeEnabledWindowsMDM) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user turns on MDM features for all Windows hosts (servers excluded).`,
		`This activity does not contain any detail fields.`, ``
}

type ActivityTypeDisabledWindowsMDM struct{}

func (a ActivityTypeDisabledWindowsMDM) ActivityName() string {
	return "disabled_windows_mdm"
}

func (a ActivityTypeDisabledWindowsMDM) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user turns off MDM features for all Windows hosts.`,
		`This activity does not contain any detail fields.`, ``
}

type ActivityTypeRanScript struct {
	HostID            uint    `json:"host_id"`
	HostDisplayName   string  `json:"host_display_name"`
	ScriptExecutionID string  `json:"script_execution_id"`
	ScriptName        string  `json:"script_name"`
	Async             bool    `json:"async"`
	PolicyID          *uint   `json:"policy_id"`
	PolicyName        *string `json:"policy_name"`
}

func (a ActivityTypeRanScript) ActivityName() string {
	return "ran_script"
}

func (a ActivityTypeRanScript) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityTypeRanScript) Documentation() (activity, details, detailsExample string) {
	return `Generated when a script is sent to be run for a host.`,
		`This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "script_execution_id": Execution ID of the script run.
- "script_name": Name of the script (empty if it was an anonymous script).
- "async": Whether the script was executed asynchronously.
- "policy_id": ID of the policy whose failure triggered the script run. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered the script run. Null if no associated policy.`, `{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "script_name": "set-timezones.sh",
  "script_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "async": false,
  "policy_id": 123,
  "policy_name": "Ensure photon torpedoes are primed"
}`
}

type ActivityTypeAddedScript struct {
	ScriptName string  `json:"script_name"`
	TeamID     *uint   `json:"team_id"`
	TeamName   *string `json:"team_name"`
}

func (a ActivityTypeAddedScript) ActivityName() string {
	return "added_script"
}

func (a ActivityTypeAddedScript) Documentation() (activity, details, detailsExample string) {
	return `Generated when a script is added to a team (or no team).`,
		`This activity contains the following fields:
- "script_name": Name of the script.
- "team_id": The ID of the team that the script applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the script applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "script_name": "set-timezones.sh",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeDeletedScript struct {
	ScriptName string  `json:"script_name"`
	TeamID     *uint   `json:"team_id"`
	TeamName   *string `json:"team_name"`
}

func (a ActivityTypeDeletedScript) ActivityName() string {
	return "deleted_script"
}

func (a ActivityTypeDeletedScript) Documentation() (activity, details, detailsExample string) {
	return `Generated when a script is deleted from a team (or no team).`,
		`This activity contains the following fields:
- "script_name": Name of the script.
- "team_id": The ID of the team that the script applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the script applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "script_name": "set-timezones.sh",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeEditedScript struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeEditedScript) ActivityName() string {
	return "edited_script"
}

func (a ActivityTypeEditedScript) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user edits the scripts of a team (or no team) via the fleetctl CLI.`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the scripts apply to, ` + "`null`" + ` if they apply to devices that are not in a team.
- "team_name": The name of the team that the scripts apply to, ` + "`null`" + ` if they apply to devices that are not in a team.`, `{
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeCreatedWindowsProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}

func (a ActivityTypeCreatedWindowsProfile) ActivityName() string {
	return "created_windows_profile"
}

func (a ActivityTypeCreatedWindowsProfile) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user adds a new Windows profile to a team (or no team).`,
		`This activity contains the following fields:
- "profile_name": Name of the profile.
- "team_id": The ID of the team that the profile applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the profile applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeDeletedWindowsProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}

func (a ActivityTypeDeletedWindowsProfile) ActivityName() string {
	return "deleted_windows_profile"
}

func (a ActivityTypeDeletedWindowsProfile) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user deletes a Windows profile from a team (or no team).`,
		`This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "team_id": The ID of the team that the profile applied to, ` + "`null`" + ` if it applied to devices that are not in a team.
- "team_name": The name of the team that the profile applied to, ` + "`null`" + ` if it applied to devices that are not in a team.`, `{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeEditedWindowsProfile struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeEditedWindowsProfile) ActivityName() string {
	return "edited_windows_profile"
}

func (a ActivityTypeEditedWindowsProfile) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user edits the Windows profiles of a team (or no team) via the fleetctl CLI.`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the profiles apply to, ` + "`null`" + ` if they apply to devices that are not in a team.
- "team_name": The name of the team that the profiles apply to, ` + "`null`" + ` if they apply to devices that are not in a team.`, `{
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeLockedHost struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	ViewPIN         bool   `json:"view_pin"`
}

func (a ActivityTypeLockedHost) ActivityName() string {
	return "locked_host"
}

func (a ActivityTypeLockedHost) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityTypeLockedHost) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user sends a request to lock a host.`,
		`This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "view_pin": Whether lock PIN was viewed (for Apple devices).`, `{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "view_pin": true
}`
}

type ActivityTypeUnlockedHost struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	HostPlatform    string `json:"host_platform"`
}

func (a ActivityTypeUnlockedHost) ActivityName() string {
	return "unlocked_host"
}

func (a ActivityTypeUnlockedHost) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityTypeUnlockedHost) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user sends a request to unlock a host.`,
		`This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "host_platform": Platform of the host.`, `{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "host_platform": "darwin"
}`
}

type ActivityTypeWipedHost struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
}

func (a ActivityTypeWipedHost) ActivityName() string {
	return "wiped_host"
}

func (a ActivityTypeWipedHost) Documentation() (activity, details, detailsExample string) {
	return `Generated when a user sends a request to wipe a host.`,
		`This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.`, `{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro"
}`
}

type ActivityTypeCreatedDeclarationProfile struct {
	ProfileName string  `json:"profile_name"`
	Identifier  string  `json:"identifier"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}

func (a ActivityTypeCreatedDeclarationProfile) ActivityName() string {
	return "created_declaration_profile"
}

func (a ActivityTypeCreatedDeclarationProfile) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a user adds a new macOS declaration to a team (or no team).`,
		`This activity contains the following fields:
- "profile_name": Name of the declaration.
- "identifier": Identifier of the declaration.
- "team_id": The ID of the team that the declaration applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the declaration applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeDeletedDeclarationProfile struct {
	ProfileName string  `json:"profile_name"`
	Identifier  string  `json:"identifier"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}

func (a ActivityTypeDeletedDeclarationProfile) ActivityName() string {
	return "deleted_declaration_profile"
}

func (a ActivityTypeDeletedDeclarationProfile) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a user removes a macOS declaration from a team (or no team).`,
		`This activity contains the following fields:
- "profile_name": Name of the declaration.
- "identifier": Identifier of the declaration.
- "team_id": The ID of the team that the declaration applies to, ` + "`null`" + ` if it applies to devices that are not in a team.
- "team_name": The name of the team that the declaration applies to, ` + "`null`" + ` if it applies to devices that are not in a team.`, `{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeEditedDeclarationProfile struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

func (a ActivityTypeEditedDeclarationProfile) ActivityName() string {
	return "edited_declaration_profile"
}

func (a ActivityTypeEditedDeclarationProfile) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a user edits the macOS declarations of a team (or no team) via the fleetctl CLI.`,
		`This activity contains the following fields:
- "team_id": The ID of the team that the declarations apply to, ` + "`null`" + ` if they apply to devices that are not in a team.
- "team_name": The name of the team that the declarations apply to, ` + "`null`" + ` if they apply to devices that are not in a team.`, `{
  "team_id": 123,
  "team_name": "Workstations"
}`
}

type ActivityTypeResentConfigurationProfile struct {
	HostID          *uint   `json:"host_id"`
	HostDisplayName *string `json:"host_display_name"`
	ProfileName     string  `json:"profile_name"`
}

func (a ActivityTypeResentConfigurationProfile) ActivityName() string {
	return "resent_configuration_profile"
}

func (a ActivityTypeResentConfigurationProfile) Documentation() (activity string, details string, detailsExample string) {
	return `Generated when a user resends an MDM configuration profile to a host.`,
		`This activity contains the following fields:
- "host_id": The ID of the host.
- "host_display_name": The display name of the host.
- "profile_name": The name of the configuration profile.`, `{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "profile_name": "Passcode requirements"
}`
}

type ActivityTypeInstalledSoftware struct {
	HostID          uint    `json:"host_id"`
	HostDisplayName string  `json:"host_display_name"`
	SoftwareTitle   string  `json:"software_title"`
	SoftwarePackage string  `json:"software_package"`
	SelfService     bool    `json:"self_service"`
	InstallUUID     string  `json:"install_uuid"`
	Status          string  `json:"status"`
	PolicyID        *uint   `json:"policy_id"`
	PolicyName      *string `json:"policy_name"`
}

func (a ActivityTypeInstalledSoftware) ActivityName() string {
	return "installed_software"
}

func (a ActivityTypeInstalledSoftware) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityTypeInstalledSoftware) Documentation() (activity, details, detailsExample string) {
	return `Generated when a software is installed on a host.`,
		`This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "install_uuid": ID of the software installation.
- "self_service": Whether the installation was initiated by the end user.
- "software_title": Name of the software.
- "software_package": Filename of the installer.
- "status": Status of the software installation.
- "policy_id": ID of the policy whose failure triggered the installation. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered installation. Null if no associated policy.
`, `{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "self_service": true,
  "install_uuid": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "status": "pending",
  "policy_id": 1337,
  "policy_name": "Ensure 1Password is installed and up to date"
}`
}

type ActivityTypeUninstalledSoftware struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	SoftwareTitle   string `json:"software_title"`
	ExecutionID     string `json:"script_execution_id"`
	Status          string `json:"status"`
}

func (a ActivityTypeUninstalledSoftware) ActivityName() string {
	return "uninstalled_software"
}

func (a ActivityTypeUninstalledSoftware) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityTypeUninstalledSoftware) Documentation() (activity, details, detailsExample string) {
	return `Generated when a software is uninstalled on a host.`,
		`This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "script_execution_id": ID of the software uninstall script.
- "status": Status of the software uninstallation.`, `{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Falcon.app",
  "script_execution_id": "ece8d99d-4313-446a-9af2-e152cd1bad1e",
  "status": "uninstalled"
}`
}

type ActivityTypeAddedSoftware struct {
	SoftwareTitle   string  `json:"software_title"`
	SoftwarePackage string  `json:"software_package"`
	TeamName        *string `json:"team_name"`
	TeamID          *uint   `json:"team_id"`
	SelfService     bool    `json:"self_service"`
}

func (a ActivityTypeAddedSoftware) ActivityName() string {
	return "added_software"
}

func (a ActivityTypeAddedSoftware) Documentation() (string, string, string) {
	return `Generated when a software installer is uploaded to Fleet.`, `This activity contains the following fields:
- "software_title": Name of the software.
- "software_package": Filename of the installer.
- "team_name": Name of the team to which this software was added.` + " `null` " + `if it was added to no team." +
- "team_id": The ID of the team to which this software was added.` + " `null` " + `if it was added to no team.
- "self_service": Whether the software is available for installation by the end user.`, `{
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "team_name": "Workstations",
  "team_id": 123,
  "self_service": true
}`
}

type ActivityTypeEditedSoftware struct {
	SoftwareTitle   string  `json:"software_title"`
	SoftwarePackage *string `json:"software_package"`
	TeamName        *string `json:"team_name"`
	TeamID          *uint   `json:"team_id"`
	SelfService     bool    `json:"self_service"`
}

func (a ActivityTypeEditedSoftware) ActivityName() string {
	return "edited_software"
}

func (a ActivityTypeEditedSoftware) Documentation() (string, string, string) {
	return `Generated when a software installer is updated in Fleet.`, `This activity contains the following fields:
- "software_title": Name of the software.
- "software_package": Filename of the installer as of this update (including if unchanged).
- "team_name": Name of the team on which this software was updated.` + " `null` " + `if it was updated on no team.
- "team_id": The ID of the team on which this software was updated.` + " `null` " + `if it was updated on no team.
- "self_service": Whether the software is available for installation by the end user.`, `{
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "team_name": "Workstations",
  "team_id": 123,
  "self_service": true
}`
}

type ActivityTypeDeletedSoftware struct {
	SoftwareTitle   string  `json:"software_title"`
	SoftwarePackage string  `json:"software_package"`
	TeamName        *string `json:"team_name"`
	TeamID          *uint   `json:"team_id"`
	SelfService     bool    `json:"self_service"`
}

func (a ActivityTypeDeletedSoftware) ActivityName() string {
	return "deleted_software"
}

func (a ActivityTypeDeletedSoftware) Documentation() (string, string, string) {
	return `Generated when a software installer is deleted from Fleet.`, `This activity contains the following fields:
- "software_title": Name of the software.
- "software_package": Filename of the installer.
- "team_name": Name of the team to which this software was added.` + " `null` " + `if it was added to no team.
- "team_id": The ID of the team to which this software was added.` + " `null` " + `if it was added to no team.
- "self_service": Whether the software was available for installation by the end user.`, `{
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "team_name": "Workstations",
  "team_id": 123,
  "self_service": true
}`
}

// LogRoleChangeActivities logs activities for each role change, globally and one for each change in teams.
func LogRoleChangeActivities(
	ctx context.Context, svc Service, adminUser *User, oldGlobalRole *string, oldTeamRoles []UserTeam, user *User,
) error {
	if user.GlobalRole != nil && (oldGlobalRole == nil || *oldGlobalRole != *user.GlobalRole) {
		if err := svc.NewActivity(
			ctx,
			adminUser,
			ActivityTypeChangedUserGlobalRole{
				UserID:    user.ID,
				UserName:  user.Name,
				UserEmail: user.Email,
				Role:      *user.GlobalRole,
			},
		); err != nil {
			return err
		}
	}
	if user.GlobalRole == nil && oldGlobalRole != nil {
		if err := svc.NewActivity(
			ctx,
			adminUser,
			ActivityTypeDeletedUserGlobalRole{
				UserID:    user.ID,
				UserName:  user.Name,
				UserEmail: user.Email,
				OldRole:   *oldGlobalRole,
			},
		); err != nil {
			return err
		}
	}
	oldTeamsLookup := make(map[uint]UserTeam, len(oldTeamRoles))
	for _, t := range oldTeamRoles {
		oldTeamsLookup[t.ID] = t
	}

	newTeamsLookup := make(map[uint]struct{}, len(user.Teams))
	for _, t := range user.Teams {
		newTeamsLookup[t.ID] = struct{}{}
		o, ok := oldTeamsLookup[t.ID]
		if ok && o.Role == t.Role {
			continue
		}
		if err := svc.NewActivity(
			ctx,
			adminUser,
			ActivityTypeChangedUserTeamRole{
				UserID:    user.ID,
				UserName:  user.Name,
				UserEmail: user.Email,
				Role:      t.Role,
				TeamID:    t.ID,
				TeamName:  t.Name,
			},
		); err != nil {
			return err
		}
	}
	for _, o := range oldTeamRoles {
		if _, ok := newTeamsLookup[o.ID]; ok {
			continue
		}
		if err := svc.NewActivity(
			ctx,
			adminUser,
			ActivityTypeDeletedUserTeamRole{
				UserID:    user.ID,
				UserName:  user.Name,
				UserEmail: user.Email,
				Role:      o.Role,
				TeamID:    o.ID,
				TeamName:  o.Name,
			},
		); err != nil {
			return err
		}
	}
	return nil
}

type ActivityEnabledVPP struct {
	Location string `json:"location"`
}

func (a ActivityEnabledVPP) ActivityName() string {
	return "enabled_vpp"
}

func (a ActivityEnabledVPP) Documentation() (activity string, details string, detailsExample string) {
	return "Generated when VPP features are enabled in Fleet.", `This activity contains the following fields:
- "location": Location associated with the VPP content token for the enabled VPP features.`, `{
  "location": "Acme Inc."
}`
}

type ActivityDisabledVPP struct {
	Location string `json:"location"`
}

func (a ActivityDisabledVPP) ActivityName() string {
	return "disabled_vpp"
}

func (a ActivityDisabledVPP) Documentation() (activity string, details string, detailsExample string) {
	return "Generated when VPP features are disabled in Fleet.", `This activity contains the following fields:
- "location": Location associated with the VPP content token for the disabled VPP features.`, `{
  "location": "Acme Inc."
}`
}

type ActivityAddedAppStoreApp struct {
	SoftwareTitle string              `json:"software_title"`
	AppStoreID    string              `json:"app_store_id"`
	TeamName      *string             `json:"team_name"`
	TeamID        *uint               `json:"team_id"`
	Platform      AppleDevicePlatform `json:"platform"`
	SelfService   bool                `json:"self_service"`
}

func (a ActivityAddedAppStoreApp) ActivityName() string {
	return "added_app_store_app"
}

func (a ActivityAddedAppStoreApp) Documentation() (activity string, details string, detailsExample string) {
	return "Generated when an App Store app is added to Fleet.", `This activity contains the following fields:
- "software_title": Name of the App Store app.
- "app_store_id": ID of the app on the Apple App Store.
- "platform": Platform of the app (` + "`darwin`, `ios`, or `ipados`" + `).
- "self_service": App installation can be initiated by device owner.
- "team_name": Name of the team to which this App Store app was added, or ` + "`null`" + ` if it was added to no team.
- "team_id": ID of the team to which this App Store app was added, or ` + "`null`" + `if it was added to no team.`, `{
  "software_title": "Logic Pro",
  "app_store_id": "1234567",
  "platform": "darwin",
  "self_service": false,
  "team_name": "Workstations",
  "team_id": 1
}`
}

type ActivityDeletedAppStoreApp struct {
	SoftwareTitle string              `json:"software_title"`
	AppStoreID    string              `json:"app_store_id"`
	TeamName      *string             `json:"team_name"`
	TeamID        *uint               `json:"team_id"`
	Platform      AppleDevicePlatform `json:"platform"`
}

func (a ActivityDeletedAppStoreApp) ActivityName() string {
	return "deleted_app_store_app"
}

func (a ActivityDeletedAppStoreApp) Documentation() (activity string, details string, detailsExample string) {
	return "Generated when an App Store app is deleted from Fleet.", `This activity contains the following fields:
- "software_title": Name of the App Store app.
- "app_store_id": ID of the app on the Apple App Store.
- "platform": Platform of the app (` + "`darwin`, `ios`, or `ipados`" + `).
- "team_name": Name of the team from which this App Store app was deleted, or ` + "`null`" + ` if it was deleted from no team.
- "team_id": ID of the team from which this App Store app was deleted, or ` + "`null`" + `if it was deleted from no team.`, `{
  "software_title": "Logic Pro",
  "app_store_id": "1234567",
  "platform": "darwin",
  "team_name": "Workstations",
  "team_id": 1
}`
}

type ActivityInstalledAppStoreApp struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	SoftwareTitle   string `json:"software_title"`
	AppStoreID      string `json:"app_store_id"`
	CommandUUID     string `json:"command_uuid"`
	Status          string `json:"status,omitempty"`
	SelfService     bool   `json:"self_service"`
}

func (a ActivityInstalledAppStoreApp) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityInstalledAppStoreApp) ActivityName() string {
	return "installed_app_store_app"
}

func (a ActivityInstalledAppStoreApp) Documentation() (string, string, string) {
	return "Generated when an App Store app is installed on a device.", `This activity contains the following fields:
- host_id: ID of the host on which the app was installed.
- self_service: App installation was initiated by device owner.
- host_display_name: Display name of the host.
- software_title: Name of the App Store app.
- app_store_id: ID of the app on the Apple App Store.
- command_uuid: UUID of the MDM command used to install the app.`, `{
  "host_id": 42,
  "self_service": true,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Logic Pro",
  "app_store_id": "1234567",
  "command_uuid": "98765432-1234-1234-1234-1234567890ab"
}`
}

type ActivityAddedNDESSCEPProxy struct{}

func (a ActivityAddedNDESSCEPProxy) ActivityName() string {
	return "added_ndes_scep_proxy"
}

func (a ActivityAddedNDESSCEPProxy) Documentation() (activity string, details string, detailsExample string) {
	return "Generated when NDES SCEP proxy is configured in Fleet.", `This activity does not contain any detail fields.`, ``
}

type ActivityDeletedNDESSCEPProxy struct{}

func (a ActivityDeletedNDESSCEPProxy) ActivityName() string {
	return "deleted_ndes_scep_proxy"
}

func (a ActivityDeletedNDESSCEPProxy) Documentation() (activity string, details string, detailsExample string) {
	return "Generated when NDES SCEP proxy configuration is deleted in Fleet.", `This activity does not contain any detail fields.`, ``
}

type ActivityEditedNDESSCEPProxy struct{}

func (a ActivityEditedNDESSCEPProxy) ActivityName() string {
	return "edited_ndes_scep_proxy"
}

func (a ActivityEditedNDESSCEPProxy) Documentation() (activity string, details string, detailsExample string) {
	return "Generated when NDES SCEP proxy configuration is edited in Fleet.", `This activity does not contain any detail fields.`, ``
}
