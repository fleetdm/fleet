package fleet

import (
	"context"
	"encoding/json"
	"time"
)

type ContextKey string

// NewActivityFunc is the function signature for creating a new activity.
type NewActivityFunc func(ctx context.Context, user *User, activity ActivityDetails) error

type ActivityWebhookPayload struct {
	Timestamp     time.Time        `json:"timestamp"`
	ActorFullName *string          `json:"actor_full_name"`
	ActorID       *uint            `json:"actor_id"`
	ActorEmail    *string          `json:"actor_email"`
	Type          string           `json:"type"`
	Details       *json.RawMessage `json:"details"`
}

// ActivityWebhookContextKey is the context key to indicate that the activity webhook has been processed before saving the activity.
const ActivityWebhookContextKey = ContextKey("ActivityWebhook")

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

	ActorFullName  *string          `json:"actor_full_name,omitempty" db:"name"`
	ActorID        *uint            `json:"actor_id,omitempty" db:"user_id"`
	ActorGravatar  *string          `json:"actor_gravatar,omitempty" db:"gravatar_url"`
	ActorEmail     *string          `json:"actor_email,omitempty" db:"user_email"`
	ActorAPIOnly   *bool            `json:"actor_api_only,omitempty" db:"api_only"`
	Type           string           `json:"type" db:"activity_type"`
	Details        *json.RawMessage `json:"details" db:"details"`
	Streamed       *bool            `json:"-" db:"streamed"`
	FleetInitiated bool             `json:"fleet_initiated" db:"fleet_initiated"`
}

// AuthzType implement AuthzTyper to be able to verify access to activities
func (*Activity) AuthzType() string {
	return "activity"
}

// UpcomingActivity is the augmented activity type used to return the list of
// upcoming (pending) activities for a host.
type UpcomingActivity struct {
	Activity

	// this struct used to have an additional field for upcoming activities, but
	// it has since been removed. Keeping the distinct struct as a useful type
	// indication that the value is an upcoming, not past, activity.
}

// WellKnownActionType defines the special actions that an upcoming activity
// may correspond to, such as Lock, Wipe, etc.
type WellKnownActionType int

// List of well-known action types.
const (
	WellKnownActionNone WellKnownActionType = iota
	WellKnownActionLock
	WellKnownActionUnlock
	WellKnownActionWipe
)

// UpcomingActivityMeta is the metadata related to a host's upcoming
// activity.
type UpcomingActivityMeta struct {
	// ExecutionID is the unique identifier of the activity.
	ExecutionID string `db:"execution_id"`
	// ActivatedAt is the timestamp when the activity was "activated" (made ready
	// to process by the host). Nil if not activated yet (still waiting for
	// previous activities to complete).
	ActivatedAt *time.Time `db:"activated_at"`
	// UpcomingActivityType is the string value of the "activity_type" enum
	// column of the upcoming_activities table.
	UpcomingActivityType string `db:"activity_type"`
	// WellKnownAction is the special action that this activity corresponds to,
	// if any (default is WellKnownActionNone).
	WellKnownAction WellKnownActionType `db:"well_known_action"`
}

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
	ActivityTypeDeletedHost{},
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
	ActivityTypeEnabledMacosUpdateNewHosts{},
	ActivityTypeDisabledMacosUpdateNewHosts{},

	ActivityTypeReadHostDiskEncryptionKey{},

	ActivityTypeCreatedMacosProfile{},
	ActivityTypeDeletedMacosProfile{},
	ActivityTypeEditedMacosProfile{},

	ActivityTypeChangedMacosSetupAssistant{},
	ActivityTypeDeletedMacosSetupAssistant{},

	ActivityTypeEnabledMacosDiskEncryption{},
	ActivityTypeDisabledMacosDiskEncryption{},

	ActivityTypeEnabledRecoveryLockPassword{},
	ActivityTypeDisabledRecoveryLockPassword{},

	ActivityTypeEnabledGitOpsMode{},
	ActivityTypeDisabledGitOpsMode{},

	ActivityTypeAddedBootstrapPackage{},
	ActivityTypeDeletedBootstrapPackage{},

	ActivityTypeEnabledMacosSetupEndUserAuth{},
	ActivityTypeDisabledMacosSetupEndUserAuth{},

	ActivityTypeEnabledWindowsMDM{},
	ActivityTypeDisabledWindowsMDM{},
	ActivityTypeEnabledAndroidMDM{},
	ActivityTypeDisabledAndroidMDM{},
	ActivityTypeEnabledWindowsMDMMigration{},
	ActivityTypeDisabledWindowsMDMMigration{},

	ActivityTypeRanScript{},
	ActivityTypeAddedScript{},
	ActivityTypeDeletedScript{},
	ActivityTypeEditedScript{},  // via GitOps
	ActivityTypeUpdatedScript{}, // via individual script update endpoint

	ActivityTypeCreatedWindowsProfile{},
	ActivityTypeDeletedWindowsProfile{},
	ActivityTypeEditedWindowsProfile{},

	ActivityTypeLockedHost{},
	ActivityTypeUnlockedHost{},
	ActivityTypeWipedHost{},

	ActivityTypeCreatedDeclarationProfile{},
	ActivityTypeDeletedDeclarationProfile{},
	ActivityTypeEditedDeclarationProfile{},

	ActivityTypeCreatedAndroidProfile{},
	ActivityTypeDeletedAndroidProfile{},
	ActivityTypeEditedAndroidProfile{},
	ActivityTypeEditedAndroidCertificate{},

	ActivityTypeResentConfigurationProfile{},
	ActivityTypeResentConfigurationProfileBatch{},

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
	ActivityEditedAppStoreApp{},

	ActivityAddedNDESSCEPProxy{},
	ActivityDeletedNDESSCEPProxy{},
	ActivityEditedNDESSCEPProxy{},
	ActivityAddedCustomSCEPProxy{},
	ActivityDeletedCustomSCEPProxy{},
	ActivityEditedCustomSCEPProxy{},
	ActivityAddedDigiCert{},
	ActivityDeletedDigiCert{},
	ActivityEditedDigiCert{},
	ActivityAddedHydrant{},
	ActivityDeletedHydrant{},
	ActivityEditedHydrant{},
	ActivityAddedCustomESTProxy{},
	ActivityDeletedCustomESTProxy{},
	ActivityEditedCustomESTProxy{},
	ActivityAddedSmallstep{},
	ActivityDeletedSmallstep{},
	ActivityEditedSmallstep{},

	ActivityTypeEnabledActivityAutomations{},
	ActivityTypeEditedActivityAutomations{},
	ActivityTypeDisabledActivityAutomations{},

	ActivityTypeCanceledRunScript{},
	ActivityTypeCanceledInstallSoftware{},
	ActivityTypeCanceledUninstallSoftware{},
	ActivityTypeCanceledInstallAppStoreApp{},

	ActivityTypeRanScriptBatch{},
	ActivityTypeBatchScriptScheduled{},
	ActivityTypeBatchScriptCanceled{},

	ActivityTypeAddedConditionalAccessIntegrationMicrosoft{},
	ActivityTypeDeletedConditionalAccessIntegrationMicrosoft{},
	ActivityTypeAddedConditionalAccessOkta{},
	ActivityTypeDeletedConditionalAccessOkta{},
	ActivityTypeEnabledConditionalAccessAutomations{},
	ActivityTypeDisabledConditionalAccessAutomations{},

	ActivityTypeEscrowedDiskEncryptionKey{},

	ActivityCreatedCustomVariable{},
	ActivityDeletedCustomVariable{},

	ActivityEditedSetupExperienceSoftware{},

	ActivityTypeEditedHostIdpData{},

	ActivityTypeEditedEnrollSecrets{},
}

type ActivityDetails interface {
	// ActivityName is the name/type of the activity.
	ActivityName() string
}

// ActivityHosts is the optional additional interface that can be implemented
// by activities that are related to hosts.
type ActivityHosts interface {
	ActivityDetails
	HostIDs() []uint
}

// AutomatableActivity is the optional additional interface that can be implemented
// by activities that are sometimes the result of automation ("Fleet did X"), starting with
// install/script run policy automations
type AutomatableActivity interface {
	ActivityDetails
	WasFromAutomation() bool
}

// ActivityHostOnly is the optional additional interface that can be implemented by activities that
// we want to exclude from the global activity feed, and only show on the Hosts details page
type ActivityHostOnly interface {
	ActivityDetails
	HostOnly() bool
}

// ActivityActivator is the optional additional interface that can be implemented by activities that
// may require activating the next upcoming activity when it gets created. Most upcoming activities get
// activated when the result of the previous one completes (such as scripts and software installs), but
// some can only be activated when the activity gets recorded (such as VPP and in-house apps).
type ActivityActivator interface {
	ActivityDetails
	MustActivateNextUpcomingActivity() bool
	ActivateNextUpcomingActivityArgs() (hostID uint, cmdUUID string)
}

type ActivityTypeEnabledActivityAutomations struct {
	WebhookUrl string `json:"webhook_url"`
}

func (a ActivityTypeEnabledActivityAutomations) ActivityName() string {
	return "enabled_activity_automations"
}

type ActivityTypeEditedActivityAutomations struct {
	WebhookUrl string `json:"webhook_url"`
}

func (a ActivityTypeEditedActivityAutomations) ActivityName() string {
	return "edited_activity_automations"
}

type ActivityTypeDisabledActivityAutomations struct{}

func (a ActivityTypeDisabledActivityAutomations) ActivityName() string {
	return "disabled_activity_automations"
}

type ActivityTypeCreatedPack struct {
	ID   uint   `json:"pack_id"`
	Name string `json:"pack_name"`
}

func (a ActivityTypeCreatedPack) ActivityName() string {
	return "created_pack"
}

type ActivityTypeEditedPack struct {
	ID   uint   `json:"pack_id"`
	Name string `json:"pack_name"`
}

func (a ActivityTypeEditedPack) ActivityName() string {
	return "edited_pack"
}

type ActivityTypeDeletedPack struct {
	Name string `json:"pack_name"`
}

func (a ActivityTypeDeletedPack) ActivityName() string {
	return "deleted_pack"
}

type ActivityTypeAppliedSpecPack struct{}

func (a ActivityTypeAppliedSpecPack) ActivityName() string {
	return "applied_spec_pack"
}

type ActivityTypeCreatedPolicy struct {
	ID       uint    `json:"policy_id"`
	Name     string  `json:"policy_name"`
	TeamID   *int64  `json:"team_id,omitempty" renameto:"fleet_id"`
	TeamName *string `json:"team_name,omitempty" renameto:"fleet_name"`
}

func (a ActivityTypeCreatedPolicy) ActivityName() string {
	return "created_policy"
}

type ActivityTypeEditedPolicy struct {
	ID       uint    `json:"policy_id"`
	Name     string  `json:"policy_name"`
	TeamID   *int64  `json:"team_id,omitempty" renameto:"fleet_id"`
	TeamName *string `json:"team_name,omitempty" renameto:"fleet_name"`
}

func (a ActivityTypeEditedPolicy) ActivityName() string {
	return "edited_policy"
}

type ActivityTypeDeletedPolicy struct {
	ID       uint    `json:"policy_id"`
	Name     string  `json:"policy_name"`
	TeamID   *int64  `json:"team_id,omitempty" renameto:"fleet_id"`
	TeamName *string `json:"team_name,omitempty" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedPolicy) ActivityName() string {
	return "deleted_policy"
}

type ActivityTypeAppliedSpecPolicy struct {
	Policies []*PolicySpec `json:"policies"`
}

func (a ActivityTypeAppliedSpecPolicy) ActivityName() string {
	return "applied_spec_policy"
}

type ActivityTypeCreatedSavedQuery struct {
	ID       uint    `json:"query_id" renameto:"report_id"`
	Name     string  `json:"query_name" renameto:"report_name"`
	TeamID   int64   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name,omitempty" renameto:"fleet_name"`
}

func (a ActivityTypeCreatedSavedQuery) ActivityName() string {
	return "created_saved_query"
}

type ActivityTypeEditedSavedQuery struct {
	ID       uint    `json:"query_id" renameto:"report_id"`
	Name     string  `json:"query_name" renameto:"report_name"`
	TeamID   int64   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name,omitempty" renameto:"fleet_name"`
}

func (a ActivityTypeEditedSavedQuery) ActivityName() string {
	return "edited_saved_query"
}

type ActivityTypeDeletedSavedQuery struct {
	Name     string  `json:"query_name" renameto:"report_name"`
	TeamID   int64   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name,omitempty" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedSavedQuery) ActivityName() string {
	return "deleted_saved_query"
}

type ActivityTypeDeletedMultipleSavedQuery struct {
	IDs      []uint  `json:"query_ids" renameto:"report_ids"`
	Teamid   int64   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name,omitempty" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedMultipleSavedQuery) ActivityName() string {
	return "deleted_multiple_saved_query"
}

type ActivityTypeAppliedSpecSavedQuery struct {
	Specs []*QuerySpec `json:"specs"`
}

func (a ActivityTypeAppliedSpecSavedQuery) ActivityName() string {
	return "applied_spec_saved_query"
}

type ActivityTypeCreatedTeam struct {
	ID   uint   `json:"team_id" renameto:"fleet_id"`
	Name string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeCreatedTeam) ActivityName() string {
	return "created_team"
}

type ActivityTypeDeletedTeam struct {
	ID   uint   `json:"team_id" renameto:"fleet_id"`
	Name string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedTeam) ActivityName() string {
	return "deleted_team"
}

type TeamActivityDetail struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ActivityTypeAppliedSpecTeam struct {
	Teams []TeamActivityDetail `json:"teams" renameto:"fleets"`
}

func (a ActivityTypeAppliedSpecTeam) ActivityName() string {
	return "applied_spec_team"
}

type ActivityTypeTransferredHostsToTeam struct {
	TeamID           *uint    `json:"team_id" renameto:"fleet_id"`
	TeamName         *string  `json:"team_name" renameto:"fleet_name"`
	HostIDs          []uint   `json:"host_ids"`
	HostDisplayNames []string `json:"host_display_names"`
}

func (a ActivityTypeTransferredHostsToTeam) ActivityName() string {
	return "transferred_hosts"
}

type ActivityTypeEditedAgentOptions struct {
	Global   bool    `json:"global"`
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEditedAgentOptions) ActivityName() string {
	return "edited_agent_options"
}

type ActivityTypeLiveQuery struct {
	TargetsCount uint             `json:"targets_count"`
	QuerySQL     string           `json:"query_sql" renameto:"query"`
	QueryName    *string          `json:"query_name,omitempty" renameto:"report_name"`
	Stats        *AggregatedStats `json:"stats,omitempty"`
}

func (a ActivityTypeLiveQuery) ActivityName() string {
	return "live_query"
}

type ActivityTypeUserAddedBySSO struct{}

func (a ActivityTypeUserAddedBySSO) ActivityName() string {
	return "user_added_by_sso"
}

type ActivityTypeUserLoggedIn struct {
	PublicIP string `json:"public_ip"`
}

func (a ActivityTypeUserLoggedIn) ActivityName() string {
	return "user_logged_in"
}

type ActivityTypeUserFailedLogin struct {
	Email    string `json:"email"`
	PublicIP string `json:"public_ip"`
}

func (a ActivityTypeUserFailedLogin) ActivityName() string {
	return "user_failed_login"
}

type ActivityTypeCreatedUser struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

func (a ActivityTypeCreatedUser) ActivityName() string {
	return "created_user"
}

type ActivityTypeDeletedUser struct {
	UserID               uint   `json:"user_id"`
	UserName             string `json:"user_name"`
	UserEmail            string `json:"user_email"`
	FromScimUserDeletion bool   `json:"-"`
}

func (a ActivityTypeDeletedUser) ActivityName() string {
	return "deleted_user"
}

func (a ActivityTypeDeletedUser) WasFromAutomation() bool {
	return a.FromScimUserDeletion
}

type ActivityTypeDeletedHost struct {
	HostID           uint                   `json:"host_id"`
	HostDisplayName  string                 `json:"host_display_name"`
	HostSerial       string                 `json:"host_serial"`
	TriggeredBy      DeletedHostTriggeredBy `json:"triggered_by"`
	HostExpiryWindow *int                   `json:"host_expiry_window,omitempty"`
}

type DeletedHostTriggeredBy string

const (
	DeletedHostTriggeredByManual     DeletedHostTriggeredBy = "manual"
	DeletedHostTriggeredByExpiration DeletedHostTriggeredBy = "expiration"
)

func (a ActivityTypeDeletedHost) ActivityName() string {
	return "deleted_host"
}

func (a ActivityTypeDeletedHost) WasFromAutomation() bool {
	return a.TriggeredBy == DeletedHostTriggeredByExpiration
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

type ActivityTypeDeletedUserGlobalRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	OldRole   string `json:"role"`
}

func (a ActivityTypeDeletedUserGlobalRole) ActivityName() string {
	return "deleted_user_global_role"
}

type ActivityTypeChangedUserTeamRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
	TeamID    uint   `json:"team_id" renameto:"fleet_id"`
	TeamName  string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeChangedUserTeamRole) ActivityName() string {
	return "changed_user_team_role"
}

type ActivityTypeDeletedUserTeamRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
	TeamID    uint   `json:"team_id" renameto:"fleet_id"`
	TeamName  string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedUserTeamRole) ActivityName() string {
	return "deleted_user_team_role"
}

type ActivityTypeFleetEnrolled struct {
	HostID          uint   `json:"host_id"`
	HostSerial      string `json:"host_serial"`
	HostDisplayName string `json:"host_display_name"`
}

func (a ActivityTypeFleetEnrolled) ActivityName() string {
	return "fleet_enrolled"
}

type ActivityTypeMDMEnrolled struct {
	HostSerial       *string `json:"host_serial"`
	HostDisplayName  string  `json:"host_display_name"`
	InstalledFromDEP bool    `json:"installed_from_dep"`
	MDMPlatform      string  `json:"mdm_platform"`
	// EnrollmentID is the unique identifier for the MDM BYOD enrollments. It is nil for other enrollments.
	EnrollmentID *string `json:"enrollment_id"`
	Platform     string  `json:"platform"`
}

func (a ActivityTypeMDMEnrolled) ActivityName() string {
	return "mdm_enrolled"
}

// TODO(BMAA): Should we add enrollment_id for BYOD unenrollments?
type ActivityTypeMDMUnenrolled struct {
	HostSerial       string  `json:"host_serial"`
	EnrollmentID     *string `json:"enrollment_id"`
	HostDisplayName  string  `json:"host_display_name"`
	InstalledFromDEP bool    `json:"installed_from_dep"`
	Platform         string  `json:"platform"`
}

func (a ActivityTypeMDMUnenrolled) ActivityName() string {
	return "mdm_unenrolled"
}

type ActivityTypeEditedMacOSMinVersion struct {
	TeamID         *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName       *string `json:"team_name" renameto:"fleet_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}

func (a ActivityTypeEditedMacOSMinVersion) ActivityName() string {
	return "edited_macos_min_version"
}

type ActivityTypeEnabledMacosUpdateNewHosts struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEnabledMacosUpdateNewHosts) ActivityName() string {
	return "enabled_macos_update_new_hosts"
}

type ActivityTypeDisabledMacosUpdateNewHosts struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDisabledMacosUpdateNewHosts) ActivityName() string {
	return "disabled_macos_update_new_hosts"
}

type ActivityTypeEditedWindowsUpdates struct {
	TeamID          *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName        *string `json:"team_name" renameto:"fleet_name"`
	DeadlineDays    *int    `json:"deadline_days"`
	GracePeriodDays *int    `json:"grace_period_days"`
}

func (a ActivityTypeEditedWindowsUpdates) ActivityName() string {
	return "edited_windows_updates"
}

type ActivityTypeEditedIOSMinVersion struct {
	TeamID         *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName       *string `json:"team_name" renameto:"fleet_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}

func (a ActivityTypeEditedIOSMinVersion) ActivityName() string {
	return "edited_ios_min_version"
}

type ActivityTypeEditedIPadOSMinVersion struct {
	TeamID         *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName       *string `json:"team_name" renameto:"fleet_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}

func (a ActivityTypeEditedIPadOSMinVersion) ActivityName() string {
	return "edited_ipados_min_version"
}

type ActivityTypeReadHostDiskEncryptionKey struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
}

func (a ActivityTypeReadHostDiskEncryptionKey) ActivityName() string {
	return "read_host_disk_encryption_key"
}

func (a ActivityTypeReadHostDiskEncryptionKey) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeCreatedMacosProfile struct {
	ProfileName       string  `json:"profile_name"`
	ProfileIdentifier string  `json:"profile_identifier"`
	TeamID            *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName          *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeCreatedMacosProfile) ActivityName() string {
	return "created_macos_profile"
}

type ActivityTypeDeletedMacosProfile struct {
	ProfileName       string  `json:"profile_name"`
	ProfileIdentifier string  `json:"profile_identifier"`
	TeamID            *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName          *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedMacosProfile) ActivityName() string {
	return "deleted_macos_profile"
}

type ActivityTypeEditedMacosProfile struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEditedMacosProfile) ActivityName() string {
	return "edited_macos_profile"
}

type ActivityTypeChangedMacosSetupAssistant struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeChangedMacosSetupAssistant) ActivityName() string {
	return "changed_macos_setup_assistant"
}

type ActivityTypeDeletedMacosSetupAssistant struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedMacosSetupAssistant) ActivityName() string {
	return "deleted_macos_setup_assistant"
}

type ActivityTypeEnabledMacosDiskEncryption struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEnabledMacosDiskEncryption) ActivityName() string {
	return "enabled_macos_disk_encryption"
}

type ActivityTypeDisabledMacosDiskEncryption struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDisabledMacosDiskEncryption) ActivityName() string {
	return "disabled_macos_disk_encryption"
}

type ActivityTypeEnabledRecoveryLockPassword struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEnabledRecoveryLockPassword) ActivityName() string {
	return "enabled_recovery_lock_password"
}

type ActivityTypeDisabledRecoveryLockPassword struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDisabledRecoveryLockPassword) ActivityName() string {
	return "disabled_recovery_lock_password"
}

type ActivityTypeEnabledGitOpsMode struct{}

func (a ActivityTypeEnabledGitOpsMode) ActivityName() string {
	return "enabled_gitops_mode"
}

type ActivityTypeDisabledGitOpsMode struct{}

func (a ActivityTypeDisabledGitOpsMode) ActivityName() string {
	return "disabled_gitops_mode"
}

type ActivityTypeAddedBootstrapPackage struct {
	BootstrapPackageName string  `json:"bootstrap_package_name"`
	TeamID               *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName             *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeAddedBootstrapPackage) ActivityName() string {
	return "added_bootstrap_package"
}

type ActivityTypeDeletedBootstrapPackage struct {
	BootstrapPackageName string  `json:"bootstrap_package_name"`
	TeamID               *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName             *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedBootstrapPackage) ActivityName() string {
	return "deleted_bootstrap_package"
}

type ActivityTypeEnabledMacosSetupEndUserAuth struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEnabledMacosSetupEndUserAuth) ActivityName() string {
	return "enabled_macos_setup_end_user_auth"
}

type ActivityTypeDisabledMacosSetupEndUserAuth struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDisabledMacosSetupEndUserAuth) ActivityName() string {
	return "disabled_macos_setup_end_user_auth"
}

type ActivityTypeEnabledWindowsMDM struct{}

func (a ActivityTypeEnabledWindowsMDM) ActivityName() string {
	return "enabled_windows_mdm"
}

type ActivityTypeDisabledWindowsMDM struct{}

func (a ActivityTypeDisabledWindowsMDM) ActivityName() string {
	return "disabled_windows_mdm"
}

type ActivityTypeEnabledWindowsMDMMigration struct{}

func (a ActivityTypeEnabledWindowsMDMMigration) ActivityName() string {
	return "enabled_windows_mdm_migration"
}

type ActivityTypeDisabledWindowsMDMMigration struct{}

func (a ActivityTypeDisabledWindowsMDMMigration) ActivityName() string {
	return "disabled_windows_mdm_migration"
}

type ActivityTypeRanScript struct {
	HostID              uint    `json:"host_id"`
	HostDisplayName     string  `json:"host_display_name"`
	ScriptExecutionID   string  `json:"script_execution_id"`
	BatchExecutionID    *string `json:"batch_execution_id"`
	ScriptName          string  `json:"script_name"`
	Async               bool    `json:"async"`
	PolicyID            *uint   `json:"policy_id"`
	PolicyName          *string `json:"policy_name"`
	FromSetupExperience bool    `json:"-"`
}

func (a ActivityTypeRanScript) ActivityName() string {
	return "ran_script"
}

func (a ActivityTypeRanScript) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityTypeRanScript) HostOnly() bool {
	return a.BatchExecutionID != nil
}

func (a ActivityTypeRanScript) WasFromAutomation() bool {
	return a.PolicyID != nil || a.FromSetupExperience
}

type ActivityTypeAddedScript struct {
	ScriptName string  `json:"script_name"`
	TeamID     *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName   *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeAddedScript) ActivityName() string {
	return "added_script"
}

type ActivityTypeUpdatedScript struct {
	ScriptName string  `json:"script_name"`
	TeamID     *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName   *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeUpdatedScript) ActivityName() string {
	return "updated_script"
}

type ActivityTypeDeletedScript struct {
	ScriptName string  `json:"script_name"`
	TeamID     *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName   *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedScript) ActivityName() string {
	return "deleted_script"
}

type ActivityTypeEditedScript struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEditedScript) ActivityName() string {
	return "edited_script"
}

type ActivityTypeCreatedWindowsProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName    *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeCreatedWindowsProfile) ActivityName() string {
	return "created_windows_profile"
}

type ActivityTypeDeletedWindowsProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName    *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedWindowsProfile) ActivityName() string {
	return "deleted_windows_profile"
}

type ActivityTypeEditedWindowsProfile struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEditedWindowsProfile) ActivityName() string {
	return "edited_windows_profile"
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

type ActivityTypeWipedHost struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
}

func (a ActivityTypeWipedHost) ActivityName() string {
	return "wiped_host"
}

func (a ActivityTypeWipedHost) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeCreatedDeclarationProfile struct {
	ProfileName string  `json:"profile_name"`
	Identifier  string  `json:"identifier"`
	TeamID      *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName    *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeCreatedDeclarationProfile) ActivityName() string {
	return "created_declaration_profile"
}

type ActivityTypeDeletedDeclarationProfile struct {
	ProfileName string  `json:"profile_name"`
	Identifier  string  `json:"identifier"`
	TeamID      *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName    *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedDeclarationProfile) ActivityName() string {
	return "deleted_declaration_profile"
}

type ActivityTypeEditedDeclarationProfile struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEditedDeclarationProfile) ActivityName() string {
	return "edited_declaration_profile"
}

type ActivityTypeResentConfigurationProfile struct {
	HostID          *uint   `json:"host_id"`
	HostDisplayName *string `json:"host_display_name"`
	ProfileName     string  `json:"profile_name"`
}

func (a ActivityTypeResentConfigurationProfile) ActivityName() string {
	return "resent_configuration_profile"
}

type ActivityTypeResentConfigurationProfileBatch struct {
	ProfileName string `json:"profile_name"`
	HostCount   int64  `json:"host_count"`
}

func (a ActivityTypeResentConfigurationProfileBatch) ActivityName() string {
	return "resent_configuration_profile_batch"
}

type ActivityTypeInstalledSoftware struct {
	HostID              uint    `json:"host_id"`
	HostDisplayName     string  `json:"host_display_name"`
	SoftwareTitle       string  `json:"software_title"`
	SoftwarePackage     string  `json:"software_package"`
	SelfService         bool    `json:"self_service"`
	InstallUUID         string  `json:"install_uuid"`
	Status              string  `json:"status"`
	Source              *string `json:"source,omitempty"`
	PolicyID            *uint   `json:"policy_id"`
	PolicyName          *string `json:"policy_name"`
	FromSetupExperience bool    `json:"-"`
	CommandUUID         string  `json:"command_uuid,omitempty"`
}

func (a ActivityTypeInstalledSoftware) ActivityName() string {
	return "installed_software"
}

func (a ActivityTypeInstalledSoftware) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityTypeInstalledSoftware) WasFromAutomation() bool {
	return a.PolicyID != nil || a.FromSetupExperience
}

func (a ActivityTypeInstalledSoftware) MustActivateNextUpcomingActivity() bool {
	// for in-house apps, we only activate the next upcoming activity if the
	// installation failed, because if it succeeded (and in this case, it only
	// means the command to install succeeded), we only activate the next
	// activity when we verify the app is actually installed.
	return a.CommandUUID != "" && a.Status != string(SoftwareInstalled)
}

func (a ActivityTypeInstalledSoftware) ActivateNextUpcomingActivityArgs() (uint, string) {
	return a.HostID, a.CommandUUID
}

type ActivityTypeUninstalledSoftware struct {
	HostID          uint    `json:"host_id"`
	HostDisplayName string  `json:"host_display_name"`
	SoftwareTitle   string  `json:"software_title"`
	ExecutionID     string  `json:"script_execution_id"`
	SelfService     bool    `json:"self_service"`
	Status          string  `json:"status"`
	Source          *string `json:"source,omitempty"`
}

func (a ActivityTypeUninstalledSoftware) ActivityName() string {
	return "uninstalled_software"
}

func (a ActivityTypeUninstalledSoftware) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivitySoftwareLabel struct {
	Name string `json:"name"`
	ID   uint   `json:"id"`
}

type ActivityTypeAddedSoftware struct {
	SoftwareTitle    string                  `json:"software_title"`
	SoftwarePackage  string                  `json:"software_package"`
	TeamName         *string                 `json:"team_name" renameto:"fleet_name"`
	TeamID           *uint                   `json:"team_id" renameto:"fleet_id"`
	SelfService      bool                    `json:"self_service"`
	SoftwareTitleID  uint                    `json:"software_title_id"`
	LabelsIncludeAny []ActivitySoftwareLabel `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ActivitySoftwareLabel `json:"labels_exclude_any,omitempty"`
}

func (a ActivityTypeAddedSoftware) ActivityName() string {
	return "added_software"
}

type ActivityTypeEditedSoftware struct {
	SoftwareTitle       string                  `json:"software_title"`
	SoftwarePackage     *string                 `json:"software_package"`
	TeamName            *string                 `json:"team_name" renameto:"fleet_name"`
	TeamID              *uint                   `json:"team_id" renameto:"fleet_id"`
	SelfService         bool                    `json:"self_service"`
	SoftwareIconURL     *string                 `json:"software_icon_url"`
	LabelsIncludeAny    []ActivitySoftwareLabel `json:"labels_include_any,omitempty"`
	LabelsExcludeAny    []ActivitySoftwareLabel `json:"labels_exclude_any,omitempty"`
	SoftwareTitleID     uint                    `json:"software_title_id"`
	SoftwareDisplayName string                  `json:"software_display_name"`
}

func (a ActivityTypeEditedSoftware) ActivityName() string {
	return "edited_software"
}

type ActivityTypeDeletedSoftware struct {
	SoftwareTitle    string                  `json:"software_title"`
	SoftwarePackage  string                  `json:"software_package"`
	TeamName         *string                 `json:"team_name" renameto:"fleet_name"`
	TeamID           *uint                   `json:"team_id" renameto:"fleet_id"`
	SelfService      bool                    `json:"self_service"`
	SoftwareIconURL  *string                 `json:"software_icon_url"`
	LabelsIncludeAny []ActivitySoftwareLabel `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ActivitySoftwareLabel `json:"labels_exclude_any,omitempty"`
}

func (a ActivityTypeDeletedSoftware) ActivityName() string {
	return "deleted_software"
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

type ActivityDisabledVPP struct {
	Location string `json:"location"`
}

func (a ActivityDisabledVPP) ActivityName() string {
	return "disabled_vpp"
}

type ActivityAddedAppStoreApp struct {
	SoftwareTitle    string                    `json:"software_title"`
	SoftwareTitleId  uint                      `json:"software_title_id"`
	AppStoreID       string                    `json:"app_store_id"`
	TeamName         *string                   `json:"team_name" renameto:"fleet_name"`
	TeamID           *uint                     `json:"team_id" renameto:"fleet_id"`
	Platform         InstallableDevicePlatform `json:"platform"`
	SelfService      bool                      `json:"self_service"`
	LabelsIncludeAny []ActivitySoftwareLabel   `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ActivitySoftwareLabel   `json:"labels_exclude_any,omitempty"`
	Configuration    json.RawMessage           `json:"configuration,omitempty"`
}

func (a ActivityAddedAppStoreApp) ActivityName() string {
	return "added_app_store_app"
}

type ActivityDeletedAppStoreApp struct {
	SoftwareTitle    string                    `json:"software_title"`
	AppStoreID       string                    `json:"app_store_id"`
	TeamName         *string                   `json:"team_name" renameto:"fleet_name"`
	TeamID           *uint                     `json:"team_id" renameto:"fleet_id"`
	Platform         InstallableDevicePlatform `json:"platform"`
	SoftwareIconURL  *string                   `json:"software_icon_url"`
	LabelsIncludeAny []ActivitySoftwareLabel   `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ActivitySoftwareLabel   `json:"labels_exclude_any,omitempty"`
}

func (a ActivityDeletedAppStoreApp) ActivityName() string {
	return "deleted_app_store_app"
}

type ActivityInstalledAppStoreApp struct {
	HostID              uint    `json:"host_id"`
	HostDisplayName     string  `json:"host_display_name"`
	SoftwareTitle       string  `json:"software_title"`
	AppStoreID          string  `json:"app_store_id"`
	CommandUUID         string  `json:"command_uuid"`
	Status              string  `json:"status,omitempty"`
	SelfService         bool    `json:"self_service"`
	PolicyID            *uint   `json:"policy_id"`
	PolicyName          *string `json:"policy_name"`
	HostPlatform        string  `json:"host_platform"`
	FromSetupExperience bool    `json:"-"`
	FromAutoUpdate      bool    `json:"from_auto_update"`
}

func (a ActivityInstalledAppStoreApp) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityInstalledAppStoreApp) ActivityName() string {
	return "installed_app_store_app"
}

func (a ActivityInstalledAppStoreApp) WasFromAutomation() bool {
	return a.PolicyID != nil || a.FromSetupExperience || a.FromAutoUpdate
}

func (a ActivityInstalledAppStoreApp) MustActivateNextUpcomingActivity() bool {
	// for VPP apps, we only activate the next upcoming activity if the installation
	// failed, because if it succeeded (and in this case, it only means the command to
	// install succeeded), we only activate the next activity when we verify the
	// app is actually installed.
	return a.Status != string(SoftwareInstalled)
}

func (a ActivityInstalledAppStoreApp) ActivateNextUpcomingActivityArgs() (uint, string) {
	return a.HostID, a.CommandUUID
}

type ActivityEditedAppStoreApp struct {
	SoftwareTitle       string                    `json:"software_title"`
	SoftwareTitleID     uint                      `json:"software_title_id"`
	AppStoreID          string                    `json:"app_store_id"`
	TeamName            *string                   `json:"team_name" renameto:"fleet_name"`
	TeamID              *uint                     `json:"team_id" renameto:"fleet_id"`
	Platform            InstallableDevicePlatform `json:"platform"`
	SelfService         bool                      `json:"self_service"`
	SoftwareIconURL     *string                   `json:"software_icon_url"`
	LabelsIncludeAny    []ActivitySoftwareLabel   `json:"labels_include_any,omitempty"`
	LabelsExcludeAny    []ActivitySoftwareLabel   `json:"labels_exclude_any,omitempty"`
	SoftwareDisplayName string                    `json:"software_display_name"`
	Configuration       json.RawMessage           `json:"configuration,omitempty"`
	AutoUpdateEnabled   *bool                     `json:"auto_update_enabled,omitempty"`
	AutoUpdateStartTime *string                   `json:"auto_update_window_start,omitempty"`
	AutoUpdateEndTime   *string                   `json:"auto_update_window_end,omitempty"`
}

func (a ActivityEditedAppStoreApp) ActivityName() string {
	return "edited_app_store_app"
}

type ActivityAddedNDESSCEPProxy struct{}

func (a ActivityAddedNDESSCEPProxy) ActivityName() string {
	return "added_ndes_scep_proxy"
}

type ActivityDeletedNDESSCEPProxy struct{}

func (a ActivityDeletedNDESSCEPProxy) ActivityName() string {
	return "deleted_ndes_scep_proxy"
}

type ActivityEditedNDESSCEPProxy struct{}

func (a ActivityEditedNDESSCEPProxy) ActivityName() string {
	return "edited_ndes_scep_proxy"
}

type ActivityAddedCustomSCEPProxy struct {
	Name string `json:"name"`
}

func (a ActivityAddedCustomSCEPProxy) ActivityName() string {
	return "added_custom_scep_proxy"
}

type ActivityDeletedCustomSCEPProxy struct {
	Name string `json:"name"`
}

func (a ActivityDeletedCustomSCEPProxy) ActivityName() string {
	return "deleted_custom_scep_proxy"
}

type ActivityEditedCustomSCEPProxy struct {
	Name string `json:"name"`
}

func (a ActivityEditedCustomSCEPProxy) ActivityName() string {
	return "edited_custom_scep_proxy"
}

type ActivityAddedDigiCert struct {
	Name string `json:"name"`
}

func (a ActivityAddedDigiCert) ActivityName() string {
	return "added_digicert"
}

type ActivityDeletedDigiCert struct {
	Name string `json:"name"`
}

func (a ActivityDeletedDigiCert) ActivityName() string {
	return "deleted_digicert"
}

type ActivityEditedDigiCert struct {
	Name string `json:"name"`
}

func (a ActivityEditedDigiCert) ActivityName() string {
	return "edited_digicert"
}

type ActivityAddedHydrant struct {
	Name string `json:"name"`
}

func (a ActivityAddedHydrant) ActivityName() string {
	return "added_hydrant"
}

type ActivityDeletedHydrant struct {
	Name string `json:"name"`
}

func (a ActivityDeletedHydrant) ActivityName() string {
	return "deleted_hydrant"
}

type ActivityEditedHydrant struct {
	Name string `json:"name"`
}

func (a ActivityEditedHydrant) ActivityName() string {
	return "edited_hydrant"
}

type ActivityAddedCustomESTProxy struct {
	Name string `json:"name"`
}

func (a ActivityAddedCustomESTProxy) ActivityName() string {
	return "added_custom_est_proxy"
}

type ActivityDeletedCustomESTProxy struct {
	Name string `json:"name"`
}

func (a ActivityDeletedCustomESTProxy) ActivityName() string {
	return "deleted_custom_est_proxy"
}

type ActivityEditedCustomESTProxy struct {
	Name string `json:"name"`
}

func (a ActivityEditedCustomESTProxy) ActivityName() string {
	return "edited_custom_est_proxy"
}

type ActivityAddedSmallstep struct {
	Name string `json:"name"`
}

func (a ActivityAddedSmallstep) ActivityName() string {
	return "added_smallstep"
}

type ActivityDeletedSmallstep struct {
	Name string `json:"name"`
}

func (a ActivityDeletedSmallstep) ActivityName() string {
	return "deleted_smallstep"
}

type ActivityEditedSmallstep struct {
	Name string `json:"name"`
}

func (a ActivityEditedSmallstep) ActivityName() string {
	return "edited_smallstep"
}

type ActivityTypeEnabledAndroidMDM struct{}

func (a ActivityTypeEnabledAndroidMDM) ActivityName() string { return "enabled_android_mdm" }

type ActivityTypeDisabledAndroidMDM struct{}

func (a ActivityTypeDisabledAndroidMDM) ActivityName() string { return "disabled_android_mdm" }

type ActivityTypeCanceledRunScript struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	ScriptName      string `json:"script_name"`
}

func (a ActivityTypeCanceledRunScript) ActivityName() string {
	return "canceled_run_script"
}

func (a ActivityTypeCanceledRunScript) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeCanceledInstallSoftware struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	SoftwareTitle   string `json:"software_title"`
	SoftwareTitleID uint   `json:"software_title_id"`
}

func (a ActivityTypeCanceledInstallSoftware) ActivityName() string {
	return "canceled_install_software"
}

func (a ActivityTypeCanceledInstallSoftware) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeCanceledUninstallSoftware struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	SoftwareTitle   string `json:"software_title"`
	SoftwareTitleID uint   `json:"software_title_id"`
}

func (a ActivityTypeCanceledUninstallSoftware) ActivityName() string {
	return "canceled_uninstall_software"
}

func (a ActivityTypeCanceledUninstallSoftware) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeCanceledInstallAppStoreApp struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	SoftwareTitle   string `json:"software_title"`
	SoftwareTitleID uint   `json:"software_title_id"`
}

func (a ActivityTypeCanceledInstallAppStoreApp) HostIDs() []uint {
	return []uint{a.HostID}
}

func (a ActivityTypeCanceledInstallAppStoreApp) ActivityName() string {
	return "canceled_install_app_store_app"
}

type ActivityTypeRanScriptBatch struct {
	ScriptName       string `json:"script_name"`
	BatchExecutionID string `json:"batch_execution_id"`
	HostCount        uint   `json:"host_count"`
	TeamID           *uint  `json:"team_id" renameto:"fleet_id"`
}

func (a ActivityTypeRanScriptBatch) ActivityName() string {
	return "ran_script_batch"
}

type ActivityTypeBatchScriptScheduled struct {
	BatchExecutionID string     `json:"batch_execution_id"`
	ScriptName       *string    `json:"script_name,omitempty"`
	HostCount        uint       `json:"host_count"`
	TeamID           *uint      `json:"team_id" renameto:"fleet_id"`
	NotBefore        *time.Time `json:"not_before"`
}

func (a ActivityTypeBatchScriptScheduled) ActivityName() string {
	return "scheduled_script_batch"
}

type ActivityTypeBatchScriptCanceled struct {
	BatchExecutionID string `json:"batch_execution_id"`
	ScriptName       string `json:"script_name"`
	HostCount        uint   `json:"host_count"`
	CanceledCount    uint   `json:"canceled_count"`
}

func (a ActivityTypeBatchScriptCanceled) ActivityName() string {
	return "canceled_script_batch"
}

type ActivityTypeAddedConditionalAccessIntegrationMicrosoft struct{}

func (a ActivityTypeAddedConditionalAccessIntegrationMicrosoft) ActivityName() string {
	return "added_conditional_access_integration_microsoft"
}

type ActivityTypeDeletedConditionalAccessIntegrationMicrosoft struct{}

func (a ActivityTypeDeletedConditionalAccessIntegrationMicrosoft) ActivityName() string {
	return "deleted_conditional_access_integration_microsoft"
}

type ActivityTypeAddedConditionalAccessOkta struct{}

func (a ActivityTypeAddedConditionalAccessOkta) ActivityName() string {
	return "added_conditional_access_okta"
}

type ActivityTypeDeletedConditionalAccessOkta struct{}

func (a ActivityTypeDeletedConditionalAccessOkta) ActivityName() string {
	return "deleted_conditional_access_okta"
}

type ActivityTypeEnabledConditionalAccessAutomations struct {
	TeamID   *uint  `json:"team_id" renameto:"fleet_id"`
	TeamName string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEnabledConditionalAccessAutomations) ActivityName() string {
	return "enabled_conditional_access_automations"
}

type ActivityTypeDisabledConditionalAccessAutomations struct {
	TeamID   *uint  `json:"team_id" renameto:"fleet_id"`
	TeamName string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDisabledConditionalAccessAutomations) ActivityName() string {
	return "disabled_conditional_access_automations"
}

type ActivityTypeUpdateConditionalAccessBypass struct {
	BypassDisabled bool `json:"bypass_disabled"`
}

func (a ActivityTypeUpdateConditionalAccessBypass) ActivityName() string {
	return "update_conditional_access_bypass"
}

type ActivityTypeHostBypassedConditionalAccess struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	IdPFullName     string `json:"idp_full_name"`
}

func (a ActivityTypeHostBypassedConditionalAccess) ActivityName() string {
	return "host_bypassed_conditional_access"
}

type ActivityTypeEscrowedDiskEncryptionKey struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
}

func (a ActivityTypeEscrowedDiskEncryptionKey) ActivityName() string {
	return "escrowed_disk_encryption_key"
}

func (a ActivityTypeEscrowedDiskEncryptionKey) WasFromAutomation() bool {
	return true
}

type ActivityCreatedCustomVariable struct {
	CustomVariableID   uint   `json:"custom_variable_id"`
	CustomVariableName string `json:"custom_variable_name"`
}

func (a ActivityCreatedCustomVariable) ActivityName() string {
	return "created_custom_variable"
}

type ActivityDeletedCustomVariable struct {
	CustomVariableID   uint   `json:"custom_variable_id"`
	CustomVariableName string `json:"custom_variable_name"`
}

func (a ActivityDeletedCustomVariable) ActivityName() string {
	return "deleted_custom_variable"
}

type ActivityEditedSetupExperienceSoftware struct {
	Platform string `json:"platform"`
	TeamID   uint   `json:"team_id" renameto:"fleet_id"`
	TeamName string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityEditedSetupExperienceSoftware) ActivityName() string {
	return "edited_setup_experience_software"
}

type ActivityTypeCreatedAndroidProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName    *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeCreatedAndroidProfile) ActivityName() string {
	return "created_android_profile"
}

type ActivityTypeDeletedAndroidProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName    *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedAndroidProfile) ActivityName() string {
	return "deleted_android_profile"
}

type ActivityTypeEditedAndroidProfile struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEditedAndroidProfile) ActivityName() string {
	return "edited_android_profile"
}

type ActivityTypeEditedAndroidCertificate struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEditedAndroidCertificate) ActivityName() string {
	return "edited_android_certificate"
}

type ActivityTypeEditedHostIdpData struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	HostIdPUsername string `json:"host_idp_username"`
}

func (a ActivityTypeEditedHostIdpData) ActivityName() string {
	return "edited_host_idp_data"
}

type ActivityTypeAddedCertificate struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeAddedCertificate) ActivityName() string {
	return "added_certificate"
}

type ActivityTypeDeletedCertificate struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeDeletedCertificate) ActivityName() string {
	return "deleted_certificate"
}

type ActivityTypeAddedMicrosoftEntraTenant struct {
	TenantID string `json:"tenant_id"`
}

func (a ActivityTypeAddedMicrosoftEntraTenant) ActivityName() string {
	return "added_microsoft_entra_tenant"
}

type ActivityTypeDeletedMicrosoftEntraTenant struct {
	TenantID string `json:"tenant_id"`
}

func (a ActivityTypeDeletedMicrosoftEntraTenant) ActivityName() string {
	return "deleted_microsoft_entra_tenant"
}

type ActivityTypeEditedEnrollSecrets struct {
	TeamID   *uint   `json:"team_id" renameto:"fleet_id"`
	TeamName *string `json:"team_name" renameto:"fleet_name"`
}

func (a ActivityTypeEditedEnrollSecrets) ActivityName() string {
	return "edited_enroll_secrets"
}
