package fleet

import (
	"encoding/json"
)

//go:generate go run gen_activity_doc.go ../../docs/Using-Fleet/Audit-Activities.md

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

	ActivityTypeEditedAgentOptions{},

	ActivityTypeLiveQuery{},

	ActivityTypeUserAddedBySSO{},

	ActivityTypeUserLoggedIn{},

	ActivityTypeCreatedUser{},
	ActivityTypeDeletedUser{},
	ActivityTypeChangedUserGlobalRole{},
	ActivityTypeDeletedUserGlobalRole{},
	ActivityTypeChangedUserTeamRole{},
	ActivityTypeDeletedUserTeamRole{},
}

type ActivityDetails interface {
	// ActivityName is the name/type of the activity.
	ActivityName() string
	// Documentation is used by "go generate" to generate markdown docs.
	Documentation() (activity string, details string, detailsExample string)
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
	"team_name": "foo"
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
	"team_name": "foo"
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
- "team_id": unique ID of the team for which the agent options were updated (null if global is true).
- "team_name": the name of the team for which the agent options were updated (null if global is true).`, `{
	"team_id": 123, 
	"team_name": "foo",
	"global": false
}`
}

type ActivityTypeLiveQuery struct {
	TargetsCount uint    `json:"targets_count"`
	QuerySQL     string  `json:"query_sql"`
	QueryName    *string `json:"query_name,omitempty"`
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
	ID            uint             `json:"id" db:"id"`
	ActorFullName string           `json:"actor_full_name" db:"name"`
	ActorID       *uint            `json:"actor_id" db:"user_id"`
	ActorGravatar *string          `json:"actor_gravatar" db:"gravatar_url"`
	ActorEmail    *string          `json:"actor_email" db:"email"`
	Type          string           `json:"type" db:"activity_type"`
	Details       *json.RawMessage `json:"details" db:"details"`
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

// AuthzType implement AuthzTyper to be able to verify access to activities
func (*Activity) AuthzType() string {
	return "activity"
}
