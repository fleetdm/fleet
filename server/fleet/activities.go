package fleet

import (
	"context"
	"encoding/json"
	"time"
)

type ContextKey string

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

// ActivityAutomationAuthor is the name used for the actor when an activity
// is performed by Fleet automation (cron jobs, system operations, etc.)
// rather than by a human user.
const ActivityAutomationAuthor = "Fleet"

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
	// Documentation is used by "go generate" to generate markdown docs.
	Documentation() (activity string, details string, detailsExample string)
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


type ActivityTypeEditedActivityAutomations struct {
	WebhookUrl string `json:"webhook_url"`
}


type ActivityTypeDisabledActivityAutomations struct{}


type ActivityTypeCreatedPack struct {
	ID   uint   `json:"pack_id"`
	Name string `json:"pack_name"`
}


type ActivityTypeEditedPack struct {
	ID   uint   `json:"pack_id"`
	Name string `json:"pack_name"`
}


type ActivityTypeDeletedPack struct {
	Name string `json:"pack_name"`
}


type ActivityTypeAppliedSpecPack struct{}


type ActivityTypeCreatedPolicy struct {
	ID       uint    `json:"policy_id"`
	Name     string  `json:"policy_name"`
	TeamID   *int64  `json:"team_id,omitempty"`
	TeamName *string `json:"team_name,omitempty"`
}


type ActivityTypeEditedPolicy struct {
	ID       uint    `json:"policy_id"`
	Name     string  `json:"policy_name"`
	TeamID   *int64  `json:"team_id,omitempty"`
	TeamName *string `json:"team_name,omitempty"`
}


type ActivityTypeDeletedPolicy struct {
	ID       uint    `json:"policy_id"`
	Name     string  `json:"policy_name"`
	TeamID   *int64  `json:"team_id,omitempty"`
	TeamName *string `json:"team_name,omitempty"`
}


type ActivityTypeAppliedSpecPolicy struct {
	Policies []*PolicySpec `json:"policies"`
}


type ActivityTypeCreatedSavedQuery struct {
	ID       uint    `json:"query_id"`
	Name     string  `json:"query_name"`
	TeamID   int64   `json:"team_id"`
	TeamName *string `json:"team_name,omitempty"`
}


type ActivityTypeEditedSavedQuery struct {
	ID       uint    `json:"query_id"`
	Name     string  `json:"query_name"`
	TeamID   int64   `json:"team_id"`
	TeamName *string `json:"team_name,omitempty"`
}


type ActivityTypeDeletedSavedQuery struct {
	Name     string  `json:"query_name"`
	TeamID   int64   `json:"team_id"`
	TeamName *string `json:"team_name,omitempty"`
}


type ActivityTypeDeletedMultipleSavedQuery struct {
	IDs      []uint  `json:"query_ids"`
	Teamid   int64   `json:"team_id"`
	TeamName *string `json:"team_name,omitempty"`
}


type ActivityTypeAppliedSpecSavedQuery struct {
	Specs []*QuerySpec `json:"specs"`
}


type ActivityTypeCreatedTeam struct {
	ID   uint   `json:"team_id"`
	Name string `json:"team_name"`
}


type ActivityTypeDeletedTeam struct {
	ID   uint   `json:"team_id"`
	Name string `json:"team_name"`
}


type TeamActivityDetail struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ActivityTypeAppliedSpecTeam struct {
	Teams []TeamActivityDetail `json:"teams"`
}


type ActivityTypeTransferredHostsToTeam struct {
	TeamID           *uint    `json:"team_id"`
	TeamName         *string  `json:"team_name"`
	HostIDs          []uint   `json:"host_ids"`
	HostDisplayNames []string `json:"host_display_names"`
}


type ActivityTypeEditedAgentOptions struct {
	Global   bool    `json:"global"`
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeLiveQuery struct {
	TargetsCount uint             `json:"targets_count"`
	QuerySQL     string           `json:"query_sql"`
	QueryName    *string          `json:"query_name,omitempty"`
	Stats        *AggregatedStats `json:"stats,omitempty"`
}


type ActivityTypeUserAddedBySSO struct{}


type ActivityTypeUserLoggedIn struct {
	PublicIP string `json:"public_ip"`
}


type ActivityTypeUserFailedLogin struct {
	Email    string `json:"email"`
	PublicIP string `json:"public_ip"`
}


type ActivityTypeCreatedUser struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}


type ActivityTypeDeletedUser struct {
	UserID               uint   `json:"user_id"`
	UserName             string `json:"user_name"`
	UserEmail            string `json:"user_email"`
	FromScimUserDeletion bool   `json:"-"`
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


func (a ActivityTypeDeletedHost) WasFromAutomation() bool {
	return a.TriggeredBy == DeletedHostTriggeredByExpiration
}

type ActivityTypeChangedUserGlobalRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
}


type ActivityTypeDeletedUserGlobalRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	OldRole   string `json:"role"`
}


type ActivityTypeChangedUserTeamRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
	TeamID    uint   `json:"team_id"`
	TeamName  string `json:"team_name"`
}


type ActivityTypeDeletedUserTeamRole struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
	TeamID    uint   `json:"team_id"`
	TeamName  string `json:"team_name"`
}


type ActivityTypeFleetEnrolled struct {
	HostID          uint   `json:"host_id"`
	HostSerial      string `json:"host_serial"`
	HostDisplayName string `json:"host_display_name"`
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


// TODO(BMAA): Should we add enrollment_id for BYOD unenrollments?
type ActivityTypeMDMUnenrolled struct {
	HostSerial       string  `json:"host_serial"`
	EnrollmentID     *string `json:"enrollment_id"`
	HostDisplayName  string  `json:"host_display_name"`
	InstalledFromDEP bool    `json:"installed_from_dep"`
	Platform         string  `json:"platform"`
}


type ActivityTypeEditedMacOSMinVersion struct {
	TeamID         *uint   `json:"team_id"`
	TeamName       *string `json:"team_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}


type ActivityTypeEnabledMacosUpdateNewHosts struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeDisabledMacosUpdateNewHosts struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeEditedWindowsUpdates struct {
	TeamID          *uint   `json:"team_id"`
	TeamName        *string `json:"team_name"`
	DeadlineDays    *int    `json:"deadline_days"`
	GracePeriodDays *int    `json:"grace_period_days"`
}


type ActivityTypeEditedIOSMinVersion struct {
	TeamID         *uint   `json:"team_id"`
	TeamName       *string `json:"team_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}


type ActivityTypeEditedIPadOSMinVersion struct {
	TeamID         *uint   `json:"team_id"`
	TeamName       *string `json:"team_name"`
	MinimumVersion string  `json:"minimum_version"`
	Deadline       string  `json:"deadline"`
}


type ActivityTypeReadHostDiskEncryptionKey struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
}


func (a ActivityTypeReadHostDiskEncryptionKey) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeCreatedMacosProfile struct {
	ProfileName       string  `json:"profile_name"`
	ProfileIdentifier string  `json:"profile_identifier"`
	TeamID            *uint   `json:"team_id"`
	TeamName          *string `json:"team_name"`
}


type ActivityTypeDeletedMacosProfile struct {
	ProfileName       string  `json:"profile_name"`
	ProfileIdentifier string  `json:"profile_identifier"`
	TeamID            *uint   `json:"team_id"`
	TeamName          *string `json:"team_name"`
}

type ActivityTypeEditedMacosProfile struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeChangedMacosSetupAssistant struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeDeletedMacosSetupAssistant struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeEnabledMacosDiskEncryption struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeDisabledMacosDiskEncryption struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeEnabledGitOpsMode struct{}


type ActivityTypeDisabledGitOpsMode struct{}


type ActivityTypeAddedBootstrapPackage struct {
	BootstrapPackageName string  `json:"bootstrap_package_name"`
	TeamID               *uint   `json:"team_id"`
	TeamName             *string `json:"team_name"`
}


type ActivityTypeDeletedBootstrapPackage struct {
	BootstrapPackageName string  `json:"bootstrap_package_name"`
	TeamID               *uint   `json:"team_id"`
	TeamName             *string `json:"team_name"`
}


type ActivityTypeEnabledMacosSetupEndUserAuth struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeDisabledMacosSetupEndUserAuth struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeEnabledWindowsMDM struct{}


type ActivityTypeDisabledWindowsMDM struct{}


type ActivityTypeEnabledWindowsMDMMigration struct{}


type ActivityTypeDisabledWindowsMDMMigration struct{}


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
	TeamID     *uint   `json:"team_id"`
	TeamName   *string `json:"team_name"`
}


type ActivityTypeUpdatedScript struct {
	ScriptName string  `json:"script_name"`
	TeamID     *uint   `json:"team_id"`
	TeamName   *string `json:"team_name"`
}


type ActivityTypeDeletedScript struct {
	ScriptName string  `json:"script_name"`
	TeamID     *uint   `json:"team_id"`
	TeamName   *string `json:"team_name"`
}


type ActivityTypeEditedScript struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeCreatedWindowsProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}


type ActivityTypeDeletedWindowsProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}


type ActivityTypeEditedWindowsProfile struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeLockedHost struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	ViewPIN         bool   `json:"view_pin"`
}


func (a ActivityTypeLockedHost) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeUnlockedHost struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	HostPlatform    string `json:"host_platform"`
}


func (a ActivityTypeUnlockedHost) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeWipedHost struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
}


func (a ActivityTypeWipedHost) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeCreatedDeclarationProfile struct {
	ProfileName string  `json:"profile_name"`
	Identifier  string  `json:"identifier"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}


type ActivityTypeDeletedDeclarationProfile struct {
	ProfileName string  `json:"profile_name"`
	Identifier  string  `json:"identifier"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}


type ActivityTypeEditedDeclarationProfile struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeResentConfigurationProfile struct {
	HostID          *uint   `json:"host_id"`
	HostDisplayName *string `json:"host_display_name"`
	ProfileName     string  `json:"profile_name"`
}


type ActivityTypeResentConfigurationProfileBatch struct {
	ProfileName string `json:"profile_name"`
	HostCount   int64  `json:"host_count"`
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
	TeamName         *string                 `json:"team_name"`
	TeamID           *uint                   `json:"team_id"`
	SelfService      bool                    `json:"self_service"`
	SoftwareTitleID  uint                    `json:"software_title_id"`
	LabelsIncludeAny []ActivitySoftwareLabel `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ActivitySoftwareLabel `json:"labels_exclude_any,omitempty"`
}


type ActivityTypeEditedSoftware struct {
	SoftwareTitle       string                  `json:"software_title"`
	SoftwarePackage     *string                 `json:"software_package"`
	TeamName            *string                 `json:"team_name"`
	TeamID              *uint                   `json:"team_id"`
	SelfService         bool                    `json:"self_service"`
	SoftwareIconURL     *string                 `json:"software_icon_url"`
	LabelsIncludeAny    []ActivitySoftwareLabel `json:"labels_include_any,omitempty"`
	LabelsExcludeAny    []ActivitySoftwareLabel `json:"labels_exclude_any,omitempty"`
	SoftwareTitleID     uint                    `json:"software_title_id"`
	SoftwareDisplayName string                  `json:"software_display_name"`
}


type ActivityTypeDeletedSoftware struct {
	SoftwareTitle    string                  `json:"software_title"`
	SoftwarePackage  string                  `json:"software_package"`
	TeamName         *string                 `json:"team_name"`
	TeamID           *uint                   `json:"team_id"`
	SelfService      bool                    `json:"self_service"`
	SoftwareIconURL  *string                 `json:"software_icon_url"`
	LabelsIncludeAny []ActivitySoftwareLabel `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ActivitySoftwareLabel `json:"labels_exclude_any,omitempty"`
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


type ActivityDisabledVPP struct {
	Location string `json:"location"`
}


type ActivityAddedAppStoreApp struct {
	SoftwareTitle    string                    `json:"software_title"`
	SoftwareTitleId  uint                      `json:"software_title_id"`
	AppStoreID       string                    `json:"app_store_id"`
	TeamName         *string                   `json:"team_name"`
	TeamID           *uint                     `json:"team_id"`
	Platform         InstallableDevicePlatform `json:"platform"`
	SelfService      bool                      `json:"self_service"`
	LabelsIncludeAny []ActivitySoftwareLabel   `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ActivitySoftwareLabel   `json:"labels_exclude_any,omitempty"`
	Configuration    json.RawMessage           `json:"configuration,omitempty"`
}


type ActivityDeletedAppStoreApp struct {
	SoftwareTitle    string                    `json:"software_title"`
	AppStoreID       string                    `json:"app_store_id"`
	TeamName         *string                   `json:"team_name"`
	TeamID           *uint                     `json:"team_id"`
	Platform         InstallableDevicePlatform `json:"platform"`
	SoftwareIconURL  *string                   `json:"software_icon_url"`
	LabelsIncludeAny []ActivitySoftwareLabel   `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ActivitySoftwareLabel   `json:"labels_exclude_any,omitempty"`
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
	TeamName            *string                   `json:"team_name"`
	TeamID              *uint                     `json:"team_id"`
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


type ActivityAddedNDESSCEPProxy struct{}


type ActivityDeletedNDESSCEPProxy struct{}


type ActivityEditedNDESSCEPProxy struct{}


type ActivityAddedCustomSCEPProxy struct {
	Name string `json:"name"`
}


type ActivityDeletedCustomSCEPProxy struct {
	Name string `json:"name"`
}


type ActivityEditedCustomSCEPProxy struct {
	Name string `json:"name"`
}


type ActivityAddedDigiCert struct {
	Name string `json:"name"`
}


type ActivityDeletedDigiCert struct {
	Name string `json:"name"`
}


type ActivityEditedDigiCert struct {
	Name string `json:"name"`
}


type ActivityAddedHydrant struct {
	Name string `json:"name"`
}


type ActivityDeletedHydrant struct {
	Name string `json:"name"`
}


type ActivityEditedHydrant struct {
	Name string `json:"name"`
}


type ActivityAddedCustomESTProxy struct {
	Name string `json:"name"`
}


type ActivityDeletedCustomESTProxy struct {
	Name string `json:"name"`
}


type ActivityEditedCustomESTProxy struct {
	Name string `json:"name"`
}


type ActivityAddedSmallstep struct {
	Name string `json:"name"`
}


type ActivityDeletedSmallstep struct {
	Name string `json:"name"`
}


type ActivityEditedSmallstep struct {
	Name string `json:"name"`
}


type ActivityTypeEnabledAndroidMDM struct{}

type ActivityTypeDisabledAndroidMDM struct{}

type ActivityTypeCanceledRunScript struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	ScriptName      string `json:"script_name"`
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


func (a ActivityTypeCanceledInstallSoftware) HostIDs() []uint {
	return []uint{a.HostID}
}

type ActivityTypeCanceledUninstallSoftware struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	SoftwareTitle   string `json:"software_title"`
	SoftwareTitleID uint   `json:"software_title_id"`
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


type ActivityTypeRanScriptBatch struct {
	ScriptName       string `json:"script_name"`
	BatchExecutionID string `json:"batch_execution_id"`
	HostCount        uint   `json:"host_count"`
	TeamID           *uint  `json:"team_id"`
}


type ActivityTypeBatchScriptScheduled struct {
	BatchExecutionID string     `json:"batch_execution_id"`
	ScriptName       *string    `json:"script_name,omitempty"`
	HostCount        uint       `json:"host_count"`
	TeamID           *uint      `json:"team_id"`
	NotBefore        *time.Time `json:"not_before"`
}


type ActivityTypeBatchScriptCanceled struct {
	BatchExecutionID string `json:"batch_execution_id"`
	ScriptName       string `json:"script_name"`
	HostCount        uint   `json:"host_count"`
	CanceledCount    uint   `json:"canceled_count"`
}


type ActivityTypeAddedConditionalAccessIntegrationMicrosoft struct{}


type ActivityTypeDeletedConditionalAccessIntegrationMicrosoft struct{}


type ActivityTypeAddedConditionalAccessOkta struct{}


type ActivityTypeDeletedConditionalAccessOkta struct{}


type ActivityTypeEnabledConditionalAccessAutomations struct {
	TeamID   *uint  `json:"team_id"`
	TeamName string `json:"team_name"`
}


type ActivityTypeDisabledConditionalAccessAutomations struct {
	TeamID   *uint  `json:"team_id"`
	TeamName string `json:"team_name"`
}


type ActivityTypeUpdateConditionalAccessBypass struct {
	BypassDisabled bool `json:"bypass_disabled"`
}


type ActivityTypeHostBypassedConditionalAccess struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	IdPFullName     string `json:"idp_full_name"`
}


type ActivityTypeEscrowedDiskEncryptionKey struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
}


func (a ActivityTypeEscrowedDiskEncryptionKey) WasFromAutomation() bool {
	return true
}

type ActivityCreatedCustomVariable struct {
	CustomVariableID   uint   `json:"custom_variable_id"`
	CustomVariableName string `json:"custom_variable_name"`
}


type ActivityDeletedCustomVariable struct {
	CustomVariableID   uint   `json:"custom_variable_id"`
	CustomVariableName string `json:"custom_variable_name"`
}


type ActivityEditedSetupExperienceSoftware struct {
	Platform string `json:"platform"`
	TeamID   uint   `json:"team_id"`
	TeamName string `json:"team_name"`
}


type ActivityTypeCreatedAndroidProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}


type ActivityTypeDeletedAndroidProfile struct {
	ProfileName string  `json:"profile_name"`
	TeamID      *uint   `json:"team_id"`
	TeamName    *string `json:"team_name"`
}


type ActivityTypeEditedAndroidProfile struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeEditedAndroidCertificate struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeEditedHostIdpData struct {
	HostID          uint   `json:"host_id"`
	HostDisplayName string `json:"host_display_name"`
	HostIdPUsername string `json:"host_idp_username"`
}


type ActivityTypeAddedCertificate struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeDeletedCertificate struct {
	Name     string  `json:"name"`
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}


type ActivityTypeAddedMicrosoftEntraTenant struct {
	TenantID string `json:"tenant_id"`
}


type ActivityTypeDeletedMicrosoftEntraTenant struct {
	TenantID string `json:"tenant_id"`
}


type ActivityTypeEditedEnrollSecrets struct {
	TeamID   *uint   `json:"team_id"`
	TeamName *string `json:"team_name"`
}

