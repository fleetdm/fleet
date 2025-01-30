package fleet

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"io"
	"time"

	"github.com/fleetdm/fleet/v4/server/version"
	"github.com/fleetdm/fleet/v4/server/websocket"
)

// EnterpriseOverrides contains the methods that can be overriden by the
// enterprise service
//
// TODO: find if there's a better way to accomplish this and standardize.
type EnterpriseOverrides struct {
	HostFeatures   func(context context.Context, host *Host) (*Features, error)
	TeamByIDOrName func(ctx context.Context, id *uint, name *string) (*Team, error)
	// UpdateTeamMDMDiskEncryption is the team-specific service method for when
	// a team ID is provided to the UpdateMDMDiskEncryption method.
	UpdateTeamMDMDiskEncryption func(ctx context.Context, tm *Team, enable *bool) error

	// The next two functions are implemented by the ee/service, and called
	// properly when called from an ee/service method (e.g. Modify Team), but
	// they also need to be called from the standard server/service method (e.g.
	// Modify AppConfig), so in this case we need to use the enterprise
	// overrides.
	MDMAppleEnableFileVaultAndEscrow  func(ctx context.Context, teamID *uint) error
	MDMAppleDisableFileVaultAndEscrow func(ctx context.Context, teamID *uint) error
	DeleteMDMAppleSetupAssistant      func(ctx context.Context, teamID *uint) error
	MDMAppleSyncDEPProfiles           func(ctx context.Context) error
	DeleteMDMAppleBootstrapPackage    func(ctx context.Context, teamID *uint) error
	MDMWindowsEnableOSUpdates         func(ctx context.Context, teamID *uint, updates WindowsUpdates) error
	MDMWindowsDisableOSUpdates        func(ctx context.Context, teamID *uint) error
	MDMAppleEditedAppleOSUpdates      func(ctx context.Context, teamID *uint, appleDevice AppleDevice, updates AppleOSUpdateSettings) error
	SetupExperienceNextStep           func(ctx context.Context, hostUUID string) (bool, error)
	GetVPPTokenIfCanInstallVPPApps    func(ctx context.Context, appleDevice bool, host *Host) (string, error)
	InstallVPPAppPostValidation       func(ctx context.Context, host *Host, vppApp *VPPApp, token string, selfService bool, policyID *uint) (string, error)
}

type OsqueryService interface {
	EnrollAgent(
		ctx context.Context, enrollSecret, hostIdentifier string, hostDetails map[string](map[string]string),
	) (nodeKey string, err error)
	// AuthenticateHost loads host identified by nodeKey. Returns an error if the nodeKey doesn't exist.
	AuthenticateHost(ctx context.Context, nodeKey string) (host *Host, debug bool, err error)
	GetClientConfig(ctx context.Context) (config map[string]interface{}, err error)
	// GetDistributedQueries retrieves the distributed queries to run for the host in
	// the provided context. These may be (depending on update intervals):
	//	- detail queries (including additional queries, if any),
	//	- label queries,
	//	- user-initiated distributed queries (aka live queries),
	//	- policy queries.
	//
	// A map from query name to query is returned.
	//
	// To enable the osquery "accelerated checkins" feature, a positive integer (number of seconds to activate for)
	// should be returned. Returning 0 for this will not activate the feature.
	GetDistributedQueries(ctx context.Context) (queries map[string]string, discovery map[string]string, accelerate uint, err error)
	SubmitDistributedQueryResults(
		ctx context.Context,
		results OsqueryDistributedQueryResults,
		statuses map[string]OsqueryStatus,
		messages map[string]string,
		stats map[string]*Stats,
	) (err error)
	SubmitStatusLogs(ctx context.Context, logs []json.RawMessage) (err error)
	SubmitResultLogs(ctx context.Context, logs []json.RawMessage) (err error)
	YaraRuleByName(ctx context.Context, name string) (*YaraRule, error)
}

type Service interface {
	OsqueryService

	// AuthenticateOrbitHost loads host identified by orbit's nodeKey. Returns an error if that nodeKey doesn't exist
	AuthenticateOrbitHost(ctx context.Context, nodeKey string) (host *Host, debug bool, err error)
	// EnrollOrbit enrolls an orbit instance to Fleet by using the host information + enroll secret
	// and returns the orbit node key if successful.
	//
	//	- If an entry for the host exists (osquery enrolled first) then it will update the host's orbit node key and team.
	//	- If an entry for the host doesn't exist (osquery enrolls later) then it will create a new entry in the hosts table.
	EnrollOrbit(ctx context.Context, hostInfo OrbitHostInfo, enrollSecret string) (orbitNodeKey string, err error)
	// GetOrbitConfig returns team specific flags and extensions in agent options
	// if the team id is not nil for host, otherwise it returns flags from global
	// agent options. It also returns any notifications that fleet wants to surface
	// to fleetd (formerly orbit).
	GetOrbitConfig(ctx context.Context) (OrbitConfig, error)

	// LogFleetdError logs an error report from a `fleetd` component
	LogFleetdError(ctx context.Context, errData FleetdError) error

	// SetOrUpdateDeviceAuthToken creates or updates a device auth token for the given host.
	SetOrUpdateDeviceAuthToken(ctx context.Context, authToken string) error

	// GetFleetDesktopSummary returns a summary of the host used by Fleet Desktop to operate.
	GetFleetDesktopSummary(ctx context.Context) (DesktopSummary, error)

	// SetEnterpriseOverrides allows the enterprise service to override specific methods
	// that can't be easily overridden via embedding.
	//
	// TODO: find if there's a better way to accomplish this and standardize.
	SetEnterpriseOverrides(overrides EnterpriseOverrides)

	// /////////////////////////////////////////////////////////////////////////////
	// UserService contains methods for managing a Fleet User.

	// CreateUserFromInvite creates a new User from a request payload when there is already an existing invitation.
	CreateUserFromInvite(ctx context.Context, p UserPayload) (user *User, err error)

	// CreateUser allows an admin to create a new user without first creating  and validating invite tokens.
	// The sessionKey is only returned (not-nil) when creating API-only (non-SSO) users.
	CreateUser(ctx context.Context, p UserPayload) (user *User, sessionKey *string, err error)

	// CreateInitialUser creates the first user, skipping authorization checks.  If a user already exists this method
	// should fail.
	CreateInitialUser(ctx context.Context, p UserPayload) (user *User, err error)

	// User returns a valid User given a User ID.
	User(ctx context.Context, id uint) (user *User, err error)

	// NewUser creates a new user with the given payload
	NewUser(ctx context.Context, p UserPayload) (*User, error)

	// UserUnauthorized returns a valid User given a User ID, *skipping authorization checks*
	// This method should only be used in middleware where there is not yet a viewer context and we need to load up a
	// user to create that context.
	UserUnauthorized(ctx context.Context, id uint) (user *User, err error)

	// AuthenticatedUser returns the current user from the viewer context.
	AuthenticatedUser(ctx context.Context) (user *User, err error)

	// ListUsers returns all users.
	ListUsers(ctx context.Context, opt UserListOptions) (users []*User, err error)

	// ChangePassword validates the existing password, and sets the new  password. User is retrieved from the viewer
	// context.
	ChangePassword(ctx context.Context, oldPass, newPass string) error

	// RequestPasswordReset generates a password reset request for the user  specified by email. The request results
	// in a token emailed to the user.
	RequestPasswordReset(ctx context.Context, email string) (err error)

	// RequirePasswordReset requires a password reset for the user  specified by ID (if require is true). It deletes
	// all the user's sessions, and requires that their password be reset upon the next login. Setting require to
	// false will take a user out of this state. The updated user is returned.
	RequirePasswordReset(ctx context.Context, uid uint, require bool) (*User, error)

	// PerformRequiredPasswordReset resets a password for a user that is in the required reset state. It must be called
	// with the logged in viewer context of that user.
	PerformRequiredPasswordReset(ctx context.Context, password string) (*User, error)

	// ResetPassword validates the provided password reset token and updates the user's password.
	ResetPassword(ctx context.Context, token, password string) (err error)

	// ModifyUser updates a user's parameters given a UserPayload.
	ModifyUser(ctx context.Context, userID uint, p UserPayload) (user *User, err error)

	// DeleteUser permanently deletes the user identified by the provided ID.
	DeleteUser(ctx context.Context, id uint) error

	// ChangeUserEmail is used to confirm new email address and if confirmed,
	// write the new email address to user.
	ChangeUserEmail(ctx context.Context, token string) (string, error)

	GetUserSettings(ctx context.Context, id uint) (settings *UserSettings, err error)

	// /////////////////////////////////////////////////////////////////////////////
	// Session

	// InitiateSSO is used to initiate an SSO session and returns a URL that can be used in a redirect to the IDP.
	// Arguments: redirectURL is the URL of the protected resource that the user was trying to access when they were
	// prompted to log in.
	InitiateSSO(ctx context.Context, redirectURL string) (string, error)

	// InitiateMDMAppleSSO initiates SSO for MDM flows, this method is
	// different from InitiateSSO because it receives a different
	// configuration and only supports a subset of the features (eg: we
	// don't want to allow IdP initiated authentications)
	InitiateMDMAppleSSO(ctx context.Context) (string, error)

	// InitSSOCallback handles the IDP response and ensures the credentials
	// are valid
	InitSSOCallback(ctx context.Context, auth Auth) (string, error)

	// InitiateMDMAppleSSOCallback handles the IDP response and ensures the
	// credentials are valid, then responds with an URL to the Fleet UI to
	// handle next steps based on the query parameters provided.
	InitiateMDMAppleSSOCallback(ctx context.Context, auth Auth) string

	// GetSSOUser handles retrieval of an user that is trying to authenticate
	// via SSO
	GetSSOUser(ctx context.Context, auth Auth) (*User, error)
	// LoginSSOUser logs-in the given SSO user
	LoginSSOUser(ctx context.Context, user *User, redirectURL string) (*SSOSession, error)

	// SSOSettings returns non-sensitive single sign on information used before authentication
	SSOSettings(ctx context.Context) (*SessionSSOSettings, error)
	Login(ctx context.Context, email, password string, supportsEmailVerification bool) (user *User, session *Session, err error)
	Logout(ctx context.Context) (err error)
	CompleteMFA(ctx context.Context, token string) (*Session, *User, error)
	DestroySession(ctx context.Context) (err error)
	GetInfoAboutSessionsForUser(ctx context.Context, id uint) (sessions []*Session, err error)
	DeleteSessionsForUser(ctx context.Context, id uint) (err error)
	GetInfoAboutSession(ctx context.Context, id uint) (session *Session, err error)
	GetSessionByKey(ctx context.Context, key string) (session *Session, err error)
	DeleteSession(ctx context.Context, id uint) (err error)

	// /////////////////////////////////////////////////////////////////////////////
	// PackService is the service interface for managing query packs.

	// ApplyPackSpecs applies a list of PackSpecs to the datastore, creating and updating packs as necessary.
	ApplyPackSpecs(ctx context.Context, specs []*PackSpec) ([]*PackSpec, error)

	// GetPackSpecs returns all of the stored PackSpecs.
	GetPackSpecs(ctx context.Context) ([]*PackSpec, error)

	// GetPackSpec gets the spec for the pack with the given name.
	GetPackSpec(ctx context.Context, name string) (*PackSpec, error)

	// NewPack creates a new pack in the datastore.
	NewPack(ctx context.Context, p PackPayload) (pack *Pack, err error)

	// ModifyPack modifies an existing pack in the datastore.
	ModifyPack(ctx context.Context, id uint, p PackPayload) (pack *Pack, err error)

	// ListPacks lists all packs in the application.
	ListPacks(ctx context.Context, opt PackListOptions) (packs []*Pack, err error)

	// GetPack retrieves a pack by ID.
	GetPack(ctx context.Context, id uint) (pack *Pack, err error)

	// DeletePack deletes a pack record from the datastore.
	DeletePack(ctx context.Context, name string) (err error)

	// DeletePackByID is for backwards compatibility with the UI
	DeletePackByID(ctx context.Context, id uint) (err error)

	// ListPacksForHost lists the packs that a host should execute.
	ListPacksForHost(ctx context.Context, hid uint) (packs []*Pack, err error)

	// /////////////////////////////////////////////////////////////////////////////
	// LabelService

	// ApplyLabelSpecs applies a list of LabelSpecs to the datastore, creating and updating labels as necessary.
	ApplyLabelSpecs(ctx context.Context, specs []*LabelSpec) error
	// GetLabelSpecs returns all of the stored LabelSpecs.
	GetLabelSpecs(ctx context.Context) ([]*LabelSpec, error)
	// GetLabelSpec gets the spec for the label with the given name.
	GetLabelSpec(ctx context.Context, name string) (*LabelSpec, error)

	NewLabel(ctx context.Context, p LabelPayload) (label *Label, hostIDs []uint, err error)
	ModifyLabel(ctx context.Context, id uint, payload ModifyLabelPayload) (*Label, []uint, error)
	ListLabels(ctx context.Context, opt ListOptions) (labels []*Label, err error)
	LabelsSummary(ctx context.Context) (labels []*LabelSummary, err error)
	GetLabel(ctx context.Context, id uint) (label *Label, hostIDs []uint, err error)

	DeleteLabel(ctx context.Context, name string) (err error)
	// DeleteLabelByID is for backwards compatibility with the UI
	DeleteLabelByID(ctx context.Context, id uint) (err error)

	// ListHostsInLabel returns a slice of hosts in the label with the given ID.
	ListHostsInLabel(ctx context.Context, lid uint, opt HostListOptions) ([]*Host, error)

	// ListLabelsForHost returns a slice of labels for a given host
	ListLabelsForHost(ctx context.Context, hostID uint) ([]*Label, error)

	// BatchValidateLabels validates that each of the provided label names exists. The returned map
	// is keyed by label name. Caller must ensure that appropirate authorization checks are
	// performed prior to calling this method.
	BatchValidateLabels(ctx context.Context, labelNames []string) (map[string]LabelIdent, error)

	// /////////////////////////////////////////////////////////////////////////////
	// QueryService

	// ApplyQuerySpecs applies a list of queries (creating or updating them as necessary)
	ApplyQuerySpecs(ctx context.Context, specs []*QuerySpec) error
	// GetQuerySpecs gets the YAML file representing all the stored queries.
	GetQuerySpecs(ctx context.Context, teamID *uint) ([]*QuerySpec, error)
	// GetQuerySpec gets the spec for the query with the given name on a team.
	// A nil or 0 teamID means the query is looked for in the global domain.
	GetQuerySpec(ctx context.Context, teamID *uint, name string) (*QuerySpec, error)

	// ListQueries returns a list of saved queries. Note only saved queries should be returned (those that are created
	// for distributed queries but not saved should not be returned).
	// When is set to scheduled != nil, then only scheduled queries will be returned if `*scheduled == true`
	// and only non-scheduled queries will be returned if `*scheduled == false`.
	// If mergeInherited is true and a teamID is provided, then queries from the global team will be
	// included in the results.
	ListQueries(ctx context.Context, opt ListOptions, teamID *uint, scheduled *bool, mergeInherited bool, platform *string) ([]*Query, int, *PaginationMetadata, error)
	GetQuery(ctx context.Context, id uint) (*Query, error)
	// GetQueryReportResults returns all the stored results of a query for hosts the requestor has access to.
	// Returns a boolean indicating whether the report is clipped.
	GetQueryReportResults(ctx context.Context, id uint, teamID *uint) ([]HostQueryResultRow, bool, error)
	// GetHostQueryReportResults returns all stored results of a query for a specific host
	GetHostQueryReportResults(ctx context.Context, hid uint, queryID uint) (rows []HostQueryReportResult, lastFetched *time.Time, err error)
	// QueryReportIsClipped returns true if the number of query report rows exceeds the maximum
	QueryReportIsClipped(ctx context.Context, queryID uint, maxQueryReportRows int) (bool, error)
	NewQuery(ctx context.Context, p QueryPayload) (*Query, error)
	ModifyQuery(ctx context.Context, id uint, p QueryPayload) (*Query, error)
	DeleteQuery(ctx context.Context, teamID *uint, name string) error
	// DeleteQueryByID deletes a query by ID. For backwards compatibility with UI
	DeleteQueryByID(ctx context.Context, id uint) error
	// DeleteQueries deletes the existing query objects with the provided IDs. The number of deleted queries is returned
	// along with any error.
	DeleteQueries(ctx context.Context, ids []uint) (uint, error)

	// /////////////////////////////////////////////////////////////////////////////
	// CampaignService defines the distributed query campaign related service methods

	// NewDistributedQueryCampaignByIdentifiers creates a new distributed query campaign with the provided query (or the query
	// referenced by ID) and host/label targets (specified by hostname, UUID, or hardware serial).
	NewDistributedQueryCampaignByIdentifiers(
		ctx context.Context, queryString string, queryID *uint, hosts []string, labels []string,
	) (*DistributedQueryCampaign, error)

	// NewDistributedQueryCampaign creates a new distributed query campaign with the provided query (or the query
	// referenced by ID) and host/label targets
	NewDistributedQueryCampaign(
		ctx context.Context, queryString string, queryID *uint, targets HostTargets,
	) (*DistributedQueryCampaign, error)

	// StreamCampaignResults streams updates with query results and expected host totals over the provided websocket.
	// Note that the type signature is somewhat inconsistent due to this being a streaming API and not the typical
	// go-kit RPC style.
	StreamCampaignResults(ctx context.Context, conn *websocket.Conn, campaignID uint)

	GetCampaignReader(ctx context.Context, campaign *DistributedQueryCampaign) (<-chan interface{}, context.CancelFunc, error)
	CompleteCampaign(ctx context.Context, campaign *DistributedQueryCampaign) error
	RunLiveQueryDeadline(ctx context.Context, queryIDs []uint, query string, hostIDs []uint, deadline time.Duration) (
		[]QueryCampaignResult, int, error,
	)

	// /////////////////////////////////////////////////////////////////////////////
	// AgentOptionsService

	// AgentOptionsForHost gets the agent options for the provided host. The host information should be used for
	// filtering based on team, platform, etc.
	AgentOptionsForHost(ctx context.Context, hostTeamID *uint, hostPlatform string) (json.RawMessage, error)

	// /////////////////////////////////////////////////////////////////////////////
	// HostService

	// AuthenticateDevice loads host identified by the device's auth token.
	// Returns an error if the auth token doesn't exist.
	AuthenticateDevice(ctx context.Context, authToken string) (host *Host, debug bool, err error)

	ListHosts(ctx context.Context, opt HostListOptions) (hosts []*Host, err error)
	// GetHost returns the host with the provided ID.
	//
	// The return value can also include policy information and CVE scores based
	// on the values provided to `opts`
	GetHost(ctx context.Context, id uint, opts HostDetailOptions) (host *HostDetail, err error)
	// GetHostLite returns basic host information not requiring table joins
	GetHostLite(ctx context.Context, id uint) (host *Host, err error)
	GetHostHealth(ctx context.Context, id uint) (hostHealth *HostHealth, err error)
	GetHostSummary(ctx context.Context, teamID *uint, platform *string, lowDiskSpace *int) (summary *HostSummary, err error)
	DeleteHost(ctx context.Context, id uint) (err error)
	// HostByIdentifier returns one host matching the provided identifier.
	// Possible matches can be on osquery_host_identifier, node_key, UUID, or
	// hostname.
	//
	// The return value can also include policy information and CVE scores based
	// on the values provided to `opts`
	HostByIdentifier(ctx context.Context, identifier string, opts HostDetailOptions) (*HostDetail, error)
	// RefetchHost requests a refetch of host details for the provided host.
	RefetchHost(ctx context.Context, id uint) (err error)
	// AddHostsToTeam adds hosts to an existing team, clearing their team settings if teamID is nil.
	AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint, skipBulkPending bool) error
	// AddHostsToTeamByFilter adds hosts to an existing team, clearing their team settings if teamID is nil. Hosts are
	// selected by the label and HostListOptions provided.
	AddHostsToTeamByFilter(ctx context.Context, teamID *uint, filter *map[string]interface{}) error
	DeleteHosts(ctx context.Context, ids []uint, filters *map[string]interface{}) error
	CountHosts(ctx context.Context, labelID *uint, opts HostListOptions) (int, error)
	// SearchHosts performs a search on the hosts table using the following criteria:
	//	- matchQuery is the query SQL
	//	- queryID is the ID of a saved query to run (used to determine whether this is a query that observers can run)
	//	- excludedHostIDs is an optional list of IDs to omit from the search
	SearchHosts(ctx context.Context, matchQuery string, queryID *uint, excludedHostIDs []uint) ([]*Host, error)
	// ListHostDeviceMapping returns the list of device-mapping of user's email address
	// for the host.
	ListHostDeviceMapping(ctx context.Context, id uint) ([]*HostDeviceMapping, error)
	// SetCustomHostDeviceMapping sets the custom email address associated with
	// the host, which is either set by the fleetd installer at startup (via a
	// device-authenticated API), or manually by the user (via the
	// user-authenticated API).
	SetCustomHostDeviceMapping(ctx context.Context, hostID uint, email string) ([]*HostDeviceMapping, error)
	// HostLiteByIdentifier returns a host and a subset of its fields using an "identifier" string.
	// The identifier string will be matched against the Hostname, OsqueryHostID, NodeKey, UUID and HardwareSerial fields.
	HostLiteByIdentifier(ctx context.Context, identifier string) (*HostLite, error)
	// HostLiteByIdentifier returns a host and a subset of its fields from its id.
	HostLiteByID(ctx context.Context, id uint) (*HostLite, error)

	// ListDevicePolicies lists all policies for the given host, including passing / failing summaries
	ListDevicePolicies(ctx context.Context, host *Host) ([]*HostPolicy, error)

	// DisableAuthForPing is used by the /orbit/ping and /device/ping endpoints
	// to bypass authentication, as they are public
	DisableAuthForPing(ctx context.Context)

	MacadminsData(ctx context.Context, id uint) (*MacadminsData, error)
	MDMData(ctx context.Context, id uint) (*HostMDM, error)
	AggregatedMacadminsData(ctx context.Context, teamID *uint) (*AggregatedMacadminsData, error)
	AggregatedMDMData(ctx context.Context, id *uint, platform string) (AggregatedMDMData, error)
	GetMDMSolution(ctx context.Context, mdmID uint) (*MDMSolution, error)
	GetMunkiIssue(ctx context.Context, munkiIssueID uint) (*MunkiIssue, error)

	HostEncryptionKey(ctx context.Context, id uint) (*HostDiskEncryptionKey, error)
	EscrowLUKSData(ctx context.Context, passphrase string, salt string, keySlot *uint, clientError string) error

	// AddLabelsToHost adds the given label names to the host's label membership.
	//
	// If a host is already a member of one of the labels then this operation will only
	// update the membership row update time.
	//
	// Returns an error if any of the labels does not exist or if any of the labels
	// are not manual.
	AddLabelsToHost(ctx context.Context, id uint, labels []string) error
	// RemoveLabelsFromHost removes the given label names from the host's label membership.
	// Labels that the host are already not a member of are ignored.
	//
	// Returns an error if any of the labels does not exist or if any of the labels
	// are not manual.
	RemoveLabelsFromHost(ctx context.Context, id uint, labels []string) error

	// OSVersions returns a list of operating systems and associated host counts, which may be
	// filtered using the following optional criteria: team id, platform, or name and version.
	// Name cannot be used without version, and conversely, version cannot be used without name.
	OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string, opts ListOptions, includeCVSS bool) (*OSVersions, int, *PaginationMetadata, error)
	// OSVersion returns an operating system and associated host counts
	OSVersion(ctx context.Context, osVersionID uint, teamID *uint, includeCVSS bool) (*OSVersion, *time.Time, error)

	// ListHostSoftware lists the software installed or available for install on
	// the specified host.
	ListHostSoftware(ctx context.Context, hostID uint, opts HostSoftwareTitleListOptions) ([]*HostSoftwareWithInstaller, *PaginationMetadata, error)

	// /////////////////////////////////////////////////////////////////////////////
	// AppConfigService provides methods for configuring  the Fleet application

	NewAppConfig(ctx context.Context, p AppConfig) (info *AppConfig, err error)
	// AppConfigObfuscated returns the global application config with obfuscated credentials.
	AppConfigObfuscated(ctx context.Context) (info *AppConfig, err error)
	ModifyAppConfig(ctx context.Context, p []byte, applyOpts ApplySpecOptions) (info *AppConfig, err error)
	SandboxEnabled() bool

	// ApplyEnrollSecretSpec adds and updates the enroll secrets specified in the spec.
	ApplyEnrollSecretSpec(ctx context.Context, spec *EnrollSecretSpec, applyOpts ApplySpecOptions) error
	// GetEnrollSecretSpec gets the spec for the current enroll secrets.
	GetEnrollSecretSpec(ctx context.Context) (*EnrollSecretSpec, error)

	// CertificateChain returns the PEM encoded certificate chain for osqueryd TLS termination. For cases where the
	// connection is self-signed, the server will attempt to connect using the InsecureSkipVerify option in tls.Config.
	CertificateChain(ctx context.Context) (cert []byte, err error)

	// SetupRequired returns whether the app config setup needs to be performed (only when first initializing a Fleet
	// server).
	SetupRequired(ctx context.Context) (bool, error)

	// Version returns version and build information.
	Version(ctx context.Context) (*version.Info, error)

	// License returns the licensing information.
	License(ctx context.Context) (*LicenseInfo, error)

	// LoggingConfig parses config.FleetConfig instance and returns a Logging.
	LoggingConfig(ctx context.Context) (*Logging, error)

	// EmailConfig parses config.FleetConfig and returns an EmailConfig
	EmailConfig(ctx context.Context) (*EmailConfig, error)

	// UpdateIntervalConfig returns the duration for different update intervals configured in osquery
	UpdateIntervalConfig(ctx context.Context) (*UpdateIntervalConfig, error)

	// VulnerabilitiesConfig returns the vulnerabilities checks configuration for
	// the fleet instance.
	VulnerabilitiesConfig(ctx context.Context) (*VulnerabilitiesConfig, error)

	// /////////////////////////////////////////////////////////////////////////////
	// InviteService contains methods for a service which deals with user invites.

	// InviteNewUser creates an invite for a new user to join Fleet.
	InviteNewUser(ctx context.Context, payload InvitePayload) (invite *Invite, err error)

	// DeleteInvite removes an invite.
	DeleteInvite(ctx context.Context, id uint) (err error)

	// ListInvites returns a list of all invites.
	ListInvites(ctx context.Context, opt ListOptions) (invites []*Invite, err error)

	// VerifyInvite verifies that an invite exists and that it matches the invite token.
	VerifyInvite(ctx context.Context, token string) (invite *Invite, err error)

	UpdateInvite(ctx context.Context, id uint, payload InvitePayload) (*Invite, error)

	// /////////////////////////////////////////////////////////////////////////////
	// TargetService **NOTE: SearchTargets will be removed in Fleet 5.0**

	// SearchTargets will accept a search query, a slice of IDs of hosts to omit, and a slice of IDs of labels to omit,
	// and it will return a set of targets (hosts and label) which match the supplied search query. If the query ID is
	// provided and the referenced query allows observers to run, targets will include hosts that the user has observer
	// role for.
	SearchTargets(
		ctx context.Context, searchQuery string, queryID *uint, targets HostTargets,
	) (*TargetSearchResults, error)

	// CountHostsInTargets returns the metrics of the hosts in the provided label and explicit host IDs. If the query ID
	// is provided and the referenced query allows observers to run, targets will include hosts that the user has
	// observer role for.
	CountHostsInTargets(ctx context.Context, queryID *uint, targets HostTargets) (*TargetMetrics, error)

	// /////////////////////////////////////////////////////////////////////////////
	// ScheduledQueryService

	GetScheduledQueriesInPack(ctx context.Context, id uint, opts ListOptions) (queries []*ScheduledQuery, err error)
	GetScheduledQuery(ctx context.Context, id uint) (query *ScheduledQuery, err error)
	ScheduleQuery(ctx context.Context, sq *ScheduledQuery) (query *ScheduledQuery, err error)
	DeleteScheduledQuery(ctx context.Context, id uint) (err error)
	ModifyScheduledQuery(ctx context.Context, id uint, p ScheduledQueryPayload) (query *ScheduledQuery, err error)

	// /////////////////////////////////////////////////////////////////////////////
	// StatusService

	// StatusResultStore returns nil if the result store is functioning correctly, or an error indicating the problem.
	StatusResultStore(ctx context.Context) error

	// StatusLiveQuery returns nil if live queries are enabled, or an
	// error indicating the problem.
	StatusLiveQuery(ctx context.Context) error

	// /////////////////////////////////////////////////////////////////////////////
	// CarveService

	CarveBegin(ctx context.Context, payload CarveBeginPayload) (*CarveMetadata, error)
	CarveBlock(ctx context.Context, payload CarveBlockPayload) error
	GetCarve(ctx context.Context, id int64) (*CarveMetadata, error)
	ListCarves(ctx context.Context, opt CarveListOptions) ([]*CarveMetadata, error)
	GetBlock(ctx context.Context, carveId, blockId int64) ([]byte, error)

	// /////////////////////////////////////////////////////////////////////////////
	// TeamService

	// NewTeam creates a new team.
	NewTeam(ctx context.Context, p TeamPayload) (*Team, error)
	// GetTeam returns a existing team.
	GetTeam(ctx context.Context, id uint) (*Team, error)
	// ModifyTeam modifies an existing team (besides agent options).
	ModifyTeam(ctx context.Context, id uint, payload TeamPayload) (*Team, error)
	// ModifyTeamAgentOptions modifies agent options for a team.
	ModifyTeamAgentOptions(ctx context.Context, id uint, teamOptions json.RawMessage, applyOptions ApplySpecOptions) (*Team, error)
	// AddTeamUsers adds users to an existing team.
	AddTeamUsers(ctx context.Context, teamID uint, users []TeamUser) (*Team, error)
	// DeleteTeamUsers deletes users from an existing team.
	DeleteTeamUsers(ctx context.Context, teamID uint, users []TeamUser) (*Team, error)
	// DeleteTeam deletes an existing team.
	DeleteTeam(ctx context.Context, id uint) error
	// ListTeams lists teams with the ordering and filters in the provided options.
	ListTeams(ctx context.Context, opt ListOptions) ([]*Team, error)
	// ListTeamUsers lists users on the team with the provided list options.
	ListTeamUsers(ctx context.Context, teamID uint, opt ListOptions) ([]*User, error)
	// ListAvailableTeamsForUser lists the teams the user is permitted to view
	ListAvailableTeamsForUser(ctx context.Context, user *User) ([]*TeamSummary, error)
	// TeamEnrollSecrets lists the enroll secrets for the team.
	TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*EnrollSecret, error)
	// ModifyTeamEnrollSecrets modifies enroll secrets for a team.
	ModifyTeamEnrollSecrets(ctx context.Context, teamID uint, secrets []EnrollSecret) ([]*EnrollSecret, error)
	// ApplyTeamSpecs applies the changes for each team as defined in the specs.
	// On success, it returns the mapping of team names to team ids.
	ApplyTeamSpecs(ctx context.Context, specs []*TeamSpec, applyOpts ApplyTeamSpecOptions) (map[string]uint, error)

	// /////////////////////////////////////////////////////////////////////////////
	// ActivitiesService

	// NewActivity creates the given activity on the datastore.
	//
	// What we call "Activities" are administrative operations,
	// logins, running a live query, etc.
	NewActivity(ctx context.Context, user *User, activity ActivityDetails) error

	// ListActivities lists the activities stored in the datastore.
	//
	// What we call "Activities" are administrative operations,
	// logins, running a live query, etc.
	ListActivities(ctx context.Context, opt ListActivitiesOptions) ([]*Activity, *PaginationMetadata, error)

	// ListHostUpcomingActivities lists the upcoming activities for the specified
	// host. Those are activities that are queued or scheduled to run on the host
	// but haven't run yet. It also returns the total (unpaginated) count of upcoming
	// activities.
	ListHostUpcomingActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)

	// ListHostPastActivities lists the activities that have already happened for the specified host.
	ListHostPastActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)

	// /////////////////////////////////////////////////////////////////////////////
	// UserRolesService

	// ApplyUserRolesSpecs applies a list of user global and team role changes
	ApplyUserRolesSpecs(ctx context.Context, specs UsersRoleSpec) error

	// /////////////////////////////////////////////////////////////////////////////
	// GlobalScheduleService

	GlobalScheduleQuery(ctx context.Context, sq *ScheduledQuery) (*ScheduledQuery, error)
	GetGlobalScheduledQueries(ctx context.Context, opts ListOptions) ([]*ScheduledQuery, error)
	ModifyGlobalScheduledQueries(ctx context.Context, id uint, q ScheduledQueryPayload) (*ScheduledQuery, error)
	DeleteGlobalScheduledQueries(ctx context.Context, id uint) error

	// /////////////////////////////////////////////////////////////////////////////
	// TranslatorService

	Translate(ctx context.Context, payloads []TranslatePayload) ([]TranslatePayload, error)

	// /////////////////////////////////////////////////////////////////////////////
	// TeamScheduleService

	TeamScheduleQuery(ctx context.Context, teamID uint, sq *ScheduledQuery) (*ScheduledQuery, error)
	GetTeamScheduledQueries(ctx context.Context, teamID uint, opts ListOptions) ([]*ScheduledQuery, error)
	ModifyTeamScheduledQueries(
		ctx context.Context, teamID uint, scheduledQueryID uint, q ScheduledQueryPayload,
	) (*ScheduledQuery, error)
	DeleteTeamScheduledQueries(ctx context.Context, teamID uint, id uint) error

	// /////////////////////////////////////////////////////////////////////////////
	// GlobalPolicyService

	NewGlobalPolicy(ctx context.Context, p PolicyPayload) (*Policy, error)
	ListGlobalPolicies(ctx context.Context, opts ListOptions) ([]*Policy, error)
	DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error)
	ModifyGlobalPolicy(ctx context.Context, id uint, p ModifyPolicyPayload) (*Policy, error)
	GetPolicyByIDQueries(ctx context.Context, policyID uint) (*Policy, error)
	ApplyPolicySpecs(ctx context.Context, policies []*PolicySpec) error
	CountGlobalPolicies(ctx context.Context, matchQuery string) (int, error)
	AutofillPolicySql(ctx context.Context, sql string) (description string, resolution string, err error)

	// /////////////////////////////////////////////////////////////////////////////
	// Software

	ListSoftware(ctx context.Context, opt SoftwareListOptions) ([]Software, *PaginationMetadata, error)
	SoftwareByID(ctx context.Context, id uint, teamID *uint, includeCVEScores bool) (*Software, error)
	CountSoftware(ctx context.Context, opt SoftwareListOptions) (int, error)

	// SaveHostSoftwareInstallResult saves information about execution of a
	// software installation on a host.
	SaveHostSoftwareInstallResult(ctx context.Context, result *HostSoftwareInstallResultPayload) error

	// /////////////////////////////////////////////////////////////////////////////
	// Software Titles

	ListSoftwareTitles(ctx context.Context, opt SoftwareTitleListOptions) ([]SoftwareTitleListResult, int, *PaginationMetadata, error)
	SoftwareTitleByID(ctx context.Context, id uint, teamID *uint) (*SoftwareTitle, error)

	// InstallSoftwareTitle installs a software title in the given host.
	InstallSoftwareTitle(ctx context.Context, hostID uint, softwareTitleID uint) error

	// GetVPPTokenIfCanInstallVPPApps returns the host team's VPP token if the host can be a target for VPP apps
	GetVPPTokenIfCanInstallVPPApps(ctx context.Context, appleDevice bool, host *Host) (string, error)

	// InstallVPPAppPostValidation installs a VPP app, assuming that GetVPPTokenIfCanInstallVPPApps has passed and provided a VPP token
	InstallVPPAppPostValidation(ctx context.Context, host *Host, vppApp *VPPApp, token string, selfService bool, policyID *uint) (string, error)

	// UninstallSoftwareTitle uninstalls a software title in the given host.
	UninstallSoftwareTitle(ctx context.Context, hostID uint, softwareTitleID uint) error

	// GetSoftwareInstallResults gets the results for a particular software install attempt.
	GetSoftwareInstallResults(ctx context.Context, installUUID string) (*HostSoftwareInstallerResult, error)

	// BatchSetSoftwareInstallers asynchronously replaces the software installers for a specified team.
	// Returns a request UUID that can be used to track an ongoing batch request (with GetBatchSetSoftwareInstallersResult).
	BatchSetSoftwareInstallers(ctx context.Context, tmName string, payloads []*SoftwareInstallerPayload, dryRun bool) (string, error)
	// GetBatchSetSoftwareInstallersResult polls for the status of a batch-apply started by BatchSetSoftwareInstallers.
	// Return values:
	//	- 'status': status of the batch-apply which can be "processing", "completed" or "failed".
	//	- 'message': which contains error information when the status is "failed".
	//	- 'packages': Contains the list of the applied software packages (when status is "completed"). This is always empty for a dry run.
	GetBatchSetSoftwareInstallersResult(ctx context.Context, tmName string, requestUUID string, dryRun bool) (status string, message string, packages []SoftwarePackageResponse, err error)

	// SelfServiceInstallSoftwareTitle installs a software title
	// initiated by the user
	SelfServiceInstallSoftwareTitle(ctx context.Context, host *Host, softwareTitleID uint) error

	// HasSelfServiceSoftwareInstallers returns whether the host has self-service software installers
	HasSelfServiceSoftwareInstallers(ctx context.Context, host *Host) (bool, error)

	GetAppStoreApps(ctx context.Context, teamID *uint) ([]*VPPApp, error)

	AddAppStoreApp(ctx context.Context, teamID *uint, appTeam VPPAppTeam) error

	// MDMAppleProcessOTAEnrollment handles OTA enrollment requests.
	//
	// Per the [spec][1] OTA enrollment is composed of two phases, each
	// phase is a request sent by the host to the same endpoint, but it
	// must be handled differently depending on the signatures of the
	// request body:
	//
	// 1. First request has a certificate signed by Apple's CA as the root
	// certificate. The server must return a SCEP payload that the device
	// will use to get a keypair. Note that this keypair is _different_
	// from the "SCEP identity certificate" that will be generated during
	// MDM enrollment, and only used for OTA.
	//
	// 2. Second request has the SCEP certificate generated in `1` as the
	// root certificate, the server responds with a "classic" enrollment
	// profile and the device starts the regular enrollment process from there.
	//
	// The extra steps allows us to grab device information like the serial
	// number and hardware uuid to perform operations before the host even
	// enrolls in MDM. Currently, this method creates a host records and
	// assigns a pre-defined team (based on the enrollSecret provided) to
	// the host.
	//
	// [1]: https://developer.apple.com/library/archive/documentation/NetworkingInternet/Conceptual/iPhoneOTAConfiguration/Introduction/Introduction.html#//apple_ref/doc/uid/TP40009505-CH1-SW1
	MDMAppleProcessOTAEnrollment(ctx context.Context, certificates []*x509.Certificate, rootSigner *x509.Certificate, enrollSecret string, deviceInfo MDMAppleMachineInfo) ([]byte, error)

	// /////////////////////////////////////////////////////////////////////////////
	// Vulnerabilities

	// ListVulnerabilities returns a list of vulnerabilities based on the provided options.
	ListVulnerabilities(ctx context.Context, opt VulnListOptions) ([]VulnerabilityWithMetadata, *PaginationMetadata, error)
	// ListVulnerability returns a vulnerability based on the provided CVE.
	Vulnerability(ctx context.Context, cve string, teamID *uint, useCVSScores bool) (vuln *VulnerabilityWithMetadata, known bool, err error)
	// CountVulnerabilities returns the number of vulnerabilities based on the provided options.
	CountVulnerabilities(ctx context.Context, opt VulnListOptions) (uint, error)
	// ListOSVersionsByCVE returns a list of OS versions affected by the provided CVE.
	ListOSVersionsByCVE(ctx context.Context, cve string, teamID *uint) (result []*VulnerableOS, updatedAt time.Time, err error)
	// ListSoftwareByCVE returns a list of software affected by the provided CVE.
	ListSoftwareByCVE(ctx context.Context, cve string, teamID *uint) (result []*VulnerableSoftware, updatedAt time.Time, err error)

	// /////////////////////////////////////////////////////////////////////////////
	// Team Policies

	NewTeamPolicy(ctx context.Context, teamID uint, p NewTeamPolicyPayload) (*Policy, error)
	ListTeamPolicies(ctx context.Context, teamID uint, opts ListOptions, iopts ListOptions, mergeInherited bool) (teamPolicies, inheritedPolicies []*Policy, err error)
	DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error)
	ModifyTeamPolicy(ctx context.Context, teamID uint, id uint, p ModifyPolicyPayload) (*Policy, error)
	GetTeamPolicyByIDQueries(ctx context.Context, teamID uint, policyID uint) (*Policy, error)
	CountTeamPolicies(ctx context.Context, teamID uint, matchQuery string, mergeInherited bool) (int, error)

	// /////////////////////////////////////////////////////////////////////////////
	// Geolocation

	LookupGeoIP(ctx context.Context, ip string) *GeoLocation

	// /////////////////////////////////////////////////////////////////////////////
	// Installers

	GetInstaller(ctx context.Context, installer Installer) (io.ReadCloser, int64, error)
	CheckInstallerExistence(ctx context.Context, installer Installer) error

	////////////////////////////////////////////////////////////////////////////////
	// Software Installers

	GetSoftwareInstallDetails(ctx context.Context, installUUID string) (*SoftwareInstallDetails, error)

	// /////////////////////////////////////////////////////////////////////////////
	// Apple MDM

	GetAppleMDM(ctx context.Context) (*AppleMDM, error)
	GetAppleBM(ctx context.Context) (*AppleBM, error)
	RequestMDMAppleCSR(ctx context.Context, email, org string) (*AppleCSR, error)

	// GetMDMAppleCSR returns a signed CSR as base64 encoded bytes for Apple MDM. The first time
	// this method is called, it will create a SCEP certificate, a SCEP key, and an APNS key and
	// write these to the DB. On subsequent calls, it will use the saved APNS key for generating the CSR.
	GetMDMAppleCSR(ctx context.Context) ([]byte, error)

	UploadMDMAppleAPNSCert(ctx context.Context, cert io.ReadSeeker) error
	DeleteMDMAppleAPNSCert(ctx context.Context) error

	UploadVPPToken(ctx context.Context, token io.ReadSeeker) (*VPPTokenDB, error)
	UpdateVPPToken(ctx context.Context, id uint, token io.ReadSeeker) (*VPPTokenDB, error)
	UpdateVPPTokenTeams(ctx context.Context, tokenID uint, teamIDs []uint) (*VPPTokenDB, error)
	GetVPPTokens(ctx context.Context) ([]*VPPTokenDB, error)
	DeleteVPPToken(ctx context.Context, tokenID uint) error

	BatchAssociateVPPApps(ctx context.Context, teamName string, payloads []VPPBatchPayload, dryRun bool) ([]VPPAppResponse, error)

	// GetHostDEPAssignment retrieves the host DEP assignment for the specified host.
	GetHostDEPAssignment(ctx context.Context, host *Host) (*HostDEPAssignment, error)

	// NewMDMAppleConfigProfile creates a new configuration profile for the specified team.
	NewMDMAppleConfigProfile(ctx context.Context, teamID uint, r io.Reader, labels []string, labelsMembershipMode MDMLabelsMode) (*MDMAppleConfigProfile, error)
	// NewMDMAppleConfigProfileWithPayload creates a new declaration for the specified team.
	NewMDMAppleDeclaration(ctx context.Context, teamID uint, r io.Reader, labels []string, name string, labelsMembershipMode MDMLabelsMode) (*MDMAppleDeclaration, error)

	// GetMDMAppleConfigProfileByDeprecatedID retrieves the specified Apple
	// configuration profile via its numeric ID. This method is deprecated and
	// should not be used for new endpoints.
	GetMDMAppleConfigProfileByDeprecatedID(ctx context.Context, profileID uint) (*MDMAppleConfigProfile, error)
	// GetMDMAppleConfigProfile retrieves the specified configuration profile.
	GetMDMAppleConfigProfile(ctx context.Context, profileUUID string) (*MDMAppleConfigProfile, error)

	// GetMDMAppleDeclaration retrieves the specified declaration.
	GetMDMAppleDeclaration(ctx context.Context, declarationUUID string) (*MDMAppleDeclaration, error)

	// DeleteMDMAppleConfigProfileByDeprecatedID deletes the specified Apple
	// configuration profile via its numeric ID. This method is deprecated and
	// should not be used for new endpoints.
	DeleteMDMAppleConfigProfileByDeprecatedID(ctx context.Context, profileID uint) error
	// DeleteMDMAppleConfigProfile deletes the specified configuration profile.
	DeleteMDMAppleConfigProfile(ctx context.Context, profileUUID string) error

	// DeleteMDMAppleDeclaration deletes the specified declaration.
	DeleteMDMAppleDeclaration(ctx context.Context, declarationUUID string) error

	// ListMDMAppleConfigProfiles returns the list of all the configuration profiles for the
	// specified team.
	ListMDMAppleConfigProfiles(ctx context.Context, teamID uint) ([]*MDMAppleConfigProfile, error)

	// GetMDMAppleFileVaultSummary summarizes the current state of Apple disk encryption profiles on
	// each macOS host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetMDMAppleFileVaultSummary(ctx context.Context, teamID *uint) (*MDMAppleFileVaultSummary, error)

	// GetMDMAppleProfilesSummary summarizes the current state of MDM configuration profiles on
	// each host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetMDMAppleProfilesSummary(ctx context.Context, teamID *uint) (*MDMProfilesSummary, error)

	// GetMDMAppleEnrollmentProfileByToken returns the Apple enrollment from its secret token.
	GetMDMAppleEnrollmentProfileByToken(ctx context.Context, enrollmentToken string, enrollmentRef string) (profile []byte, err error)

	// GetDeviceMDMAppleEnrollmentProfile loads the raw (PList-format) enrollment
	// profile for the currently authenticated device.
	GetDeviceMDMAppleEnrollmentProfile(ctx context.Context) ([]byte, error)

	// GetMDMAppleCommandResults returns the execution results of a command identified by a CommandUUID.
	GetMDMAppleCommandResults(ctx context.Context, commandUUID string) ([]*MDMCommandResult, error)

	// ListMDMAppleCommands returns a list of MDM Apple commands corresponding to
	// the specified options.
	ListMDMAppleCommands(ctx context.Context, opts *MDMCommandListOptions) ([]*MDMAppleCommand, error)

	// UploadMDMAppleInstaller uploads an Apple installer to Fleet.
	UploadMDMAppleInstaller(ctx context.Context, name string, size int64, installer io.Reader) (*MDMAppleInstaller, error)

	// GetMDMAppleInstallerByID returns the installer details of an installer, all fields except its content,
	// (MDMAppleInstaller.Installer is nil).
	GetMDMAppleInstallerByID(ctx context.Context, id uint) (*MDMAppleInstaller, error)

	// DeleteMDMAppleInstaller deletes an Apple installer from Fleet.
	DeleteMDMAppleInstaller(ctx context.Context, id uint) error

	// GetMDMAppleInstallerByToken returns the installer with its contents included (MDMAppleInstaller.Installer) from its secret token.
	GetMDMAppleInstallerByToken(ctx context.Context, token string) (*MDMAppleInstaller, error)

	// GetMDMAppleInstallerDetailsByToken loads the installer details, all fields except its content,
	// (MDMAppleInstaller.Installer is nil) from its secret token.
	GetMDMAppleInstallerDetailsByToken(ctx context.Context, token string) (*MDMAppleInstaller, error)

	// ListMDMAppleInstallers lists all the uploaded installers.
	ListMDMAppleInstallers(ctx context.Context) ([]MDMAppleInstaller, error)

	// ListMDMAppleDevices lists all the MDM enrolled Apple devices.
	ListMDMAppleDevices(ctx context.Context) ([]MDMAppleDevice, error)

	// NewMDMAppleDEPKeyPair creates a public private key pair for use with the Apple MDM DEP token.
	//
	// Deprecated: NewMDMAppleDEPKeyPair exists only to support a deprecated endpoint.
	NewMDMAppleDEPKeyPair(ctx context.Context) (*MDMAppleDEPKeyPair, error)

	// GenerateABMKeyPair generates and stores in the database public and
	// private keys to use in ABM to generate an encrypted auth token.
	GenerateABMKeyPair(ctx context.Context) (*MDMAppleDEPKeyPair, error)

	// UploadABMToken reads and validates if the provided token can be
	// decrypted using the keys stored in the database, then saves the token.
	UploadABMToken(ctx context.Context, token io.Reader) (*ABMToken, error)

	// ListABMTokens lists all the ABM tokens in Fleet.
	ListABMTokens(ctx context.Context) ([]*ABMToken, error)

	// CountABMTokens counts the ABM tokens in Fleet.
	CountABMTokens(ctx context.Context) (int, error)

	// UpdateABMTokenTeams updates the default macOS, iOS, and iPadOS team IDs for a given ABM token.
	UpdateABMTokenTeams(ctx context.Context, tokenID uint, macOSTeamID, iOSTeamID, iPadOSTeamID *uint) (*ABMToken, error)

	// DeleteABMToken deletes the given ABM token.
	DeleteABMToken(ctx context.Context, tokenID uint) error

	// RenewABMToken replaces the contents of the given ABM token with the given bytes.
	RenewABMToken(ctx context.Context, token io.Reader, tokenID uint) (*ABMToken, error)

	// EnqueueMDMAppleCommand enqueues a command for execution on the given
	// devices. Note that a deviceID is the same as a host's UUID.
	EnqueueMDMAppleCommand(ctx context.Context, rawBase64Cmd string, deviceIDs []string) (result *CommandEnqueueResult, err error)

	// EnqueueMDMAppleCommandRemoveEnrollmentProfile enqueues a command to remove the
	// profile used for Fleet MDM enrollment from the specified device.
	EnqueueMDMAppleCommandRemoveEnrollmentProfile(ctx context.Context, hostID uint) error

	// BatchSetMDMAppleProfiles replaces the custom macOS profiles for a specified
	// team or for hosts with no team.
	BatchSetMDMAppleProfiles(ctx context.Context, teamID *uint, teamName *string, profiles [][]byte, dryRun bool, skipBulkPending bool) error

	// MDMApplePreassignProfile preassigns a profile to a host, pending the match
	// request that will match the profiles to a team (or create one if needed),
	// assign the host to that team and assign the profiles to the host.
	MDMApplePreassignProfile(ctx context.Context, payload MDMApplePreassignProfilePayload) error

	// MDMAppleMatchPreassignment matches the existing preassigned profiles to a
	// team, creating one if none match, assigns the corresponding host to that
	// team and assigns the matched team's profiles to the host.
	MDMAppleMatchPreassignment(ctx context.Context, externalHostIdentifier string) error

	// MDMAppleDeviceLock remote locks a host
	MDMAppleDeviceLock(ctx context.Context, hostID uint) error

	// MMDAppleEraseDevice erases a host
	MDMAppleEraseDevice(ctx context.Context, hostID uint) error

	// MDMListHostConfigurationProfiles returns configuration profiles for a given host
	MDMListHostConfigurationProfiles(ctx context.Context, hostID uint) ([]*MDMAppleConfigProfile, error)

	// MDMAppleEnableFileVaultAndEscrow adds a configuration profile for the
	// given team that enables FileVault with a config that allows Fleet to
	// escrow the recovery key.
	MDMAppleEnableFileVaultAndEscrow(ctx context.Context, teamID *uint) error

	// MDMAppleDisableFileVaultAndEscrow removes the FileVault configuration
	// profile for the given team.
	MDMAppleDisableFileVaultAndEscrow(ctx context.Context, teamID *uint) error

	// UpdateMDMDiskEncryption updates the disk encryption setting for a
	// specified team or for hosts with no team.
	UpdateMDMDiskEncryption(ctx context.Context, teamID *uint, enableDiskEncryption *bool) error

	// VerifyMDMAppleConfigured verifies that the server is configured for
	// Apple MDM. If an error is returned, authorization is skipped so the
	// error can be raised to the user.
	VerifyMDMAppleConfigured(ctx context.Context) error

	// VerifyMDMWindowsConfigured verifies that the server is configured for
	// Windows MDM. If an error is returned, authorization is skipped so the
	// error can be raised to the user.
	VerifyMDMWindowsConfigured(ctx context.Context) error

	// VerifyMDMAppleOrWindowsConfigured verifies that the server is configured
	// for either Apple or Windows MDM. If an error is returned, authorization is
	// skipped so the error can be raised to the user.
	VerifyMDMAppleOrWindowsConfigured(ctx context.Context) error

	MDMAppleUploadBootstrapPackage(ctx context.Context, name string, pkg io.Reader, teamID uint) error

	GetMDMAppleBootstrapPackageBytes(ctx context.Context, token string) (*MDMAppleBootstrapPackage, error)

	GetMDMAppleBootstrapPackageMetadata(ctx context.Context, teamID uint, forUpdate bool) (*MDMAppleBootstrapPackage, error)

	DeleteMDMAppleBootstrapPackage(ctx context.Context, teamID *uint) error

	GetMDMAppleBootstrapPackageSummary(ctx context.Context, teamID *uint) (*MDMAppleBootstrapPackageSummary, error)

	// MDMGetEULABytes returns the contents of the EULA that matches
	// the given token.
	//
	// A token is required as the means of authentication for this resource
	// since it can be publicly accessed with anyone with a valid token.
	MDMGetEULABytes(ctx context.Context, token string) (*MDMEULA, error)
	// MDMGetEULAMetadata returns metadata about the EULA file that can
	// be used by clients to display information.
	MDMGetEULAMetadata(ctx context.Context) (*MDMEULA, error)
	// MDMCreateEULA adds a new EULA file.
	MDMCreateEULA(ctx context.Context, name string, file io.ReadSeeker) error
	// MDMAppleDelete EULA removes an EULA entry.
	MDMDeleteEULA(ctx context.Context, token string) error

	// Create or update the MDM Apple Setup Assistant for a team or no team.
	SetOrUpdateMDMAppleSetupAssistant(ctx context.Context, asst *MDMAppleSetupAssistant) (*MDMAppleSetupAssistant, error)
	// Get the MDM Apple Setup Assistant for the provided team or no team.
	GetMDMAppleSetupAssistant(ctx context.Context, teamID *uint) (*MDMAppleSetupAssistant, error)
	// Delete the MDM Apple Setup Assistant for the provided team or no team.
	DeleteMDMAppleSetupAssistant(ctx context.Context, teamID *uint) error

	// HasCustomSetupAssistantConfigurationWebURL checks if the team/global
	// config has a custom setup assistant defined, and if the JSON content
	// defines a custom `configuration_web_url`.
	HasCustomSetupAssistantConfigurationWebURL(ctx context.Context, teamID *uint) (bool, error)

	// UpdateMDMAppleSetup updates the specified MDM Apple setup values for a
	// specified team or for hosts with no team.
	UpdateMDMAppleSetup(ctx context.Context, payload MDMAppleSetupPayload) error

	// TriggerMigrateMDMDevice posts a webhook request to the URL configured
	// for MDM macOS migration.
	TriggerMigrateMDMDevice(ctx context.Context, host *Host) error

	GetMDMManualEnrollmentProfile(ctx context.Context) ([]byte, error)

	TriggerLinuxDiskEncryptionEscrow(ctx context.Context, host *Host) error

	// CheckMDMAppleEnrollmentWithMinimumOSVersion checks if the minimum OS version is met for a MDM enrollment
	CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx context.Context, m *MDMAppleMachineInfo) (*MDMAppleSoftwareUpdateRequired, error)

	// GetOTAProfile gets the OTA (over-the-air) profile for a given team based on the enroll secret provided.
	GetOTAProfile(ctx context.Context, enrollSecret string) ([]byte, error)

	///////////////////////////////////////////////////////////////////////////////
	// CronSchedulesService

	// TriggerCronSchedule attempts to trigger an ad-hoc run of the named cron schedule.
	TriggerCronSchedule(ctx context.Context, name string) error

	// ResetAutomation sets the policies and all policies of the listed teams to fire again
	// for all hosts that are already marked as failing.
	ResetAutomation(ctx context.Context, teamIDs, policyIDs []uint) error

	///////////////////////////////////////////////////////////////////////////////
	// Windows MDM

	// GetMDMMicrosoftDiscoveryResponse returns a valid DiscoveryResponse message
	GetMDMMicrosoftDiscoveryResponse(ctx context.Context, upnEmail string) (*DiscoverResponse, error)

	// GetMDMMicrosoftSTSAuthResponse returns a valid STS auth page
	GetMDMMicrosoftSTSAuthResponse(ctx context.Context, appru string, loginHint string) (string, error)

	// GetMDMWindowsPolicyResponse returns a valid GetPoliciesResponse message
	GetMDMWindowsPolicyResponse(ctx context.Context, authToken *HeaderBinarySecurityToken) (*GetPoliciesResponse, error)

	// GetMDMWindowsEnrollResponse returns a valid RequestSecurityTokenResponseCollection message
	GetMDMWindowsEnrollResponse(ctx context.Context, secTokenMsg *RequestSecurityToken, authToken *HeaderBinarySecurityToken) (*RequestSecurityTokenResponseCollection, error)

	// GetAuthorizedSoapFault authorize the request so SoapFault message can be returned
	GetAuthorizedSoapFault(ctx context.Context, eType string, origMsg int, errorMsg error) *SoapFault

	// SignMDMMicrosoftClientCSR returns a signed certificate from the client certificate signing request and the
	// certificate fingerprint. The certificate common name should be passed in the subject parameter.
	SignMDMMicrosoftClientCSR(ctx context.Context, subject string, csr *x509.CertificateRequest) ([]byte, string, error)

	// GetMDMWindowsManagementResponse returns a valid SyncML response message
	GetMDMWindowsManagementResponse(ctx context.Context, reqSyncML *SyncML, reqCerts []*x509.Certificate) (*SyncML, error)

	// GetMDMWindowsTOSContent returns TOS content
	GetMDMWindowsTOSContent(ctx context.Context, redirectUri string, reqID string) (string, error)

	// RunMDMCommand enqueues an MDM command for execution on the given devices.
	// Note that a deviceID is the same as a host's UUID.
	RunMDMCommand(ctx context.Context, rawBase64Cmd string, deviceIDs []string) (result *CommandEnqueueResult, err error)

	// GetMDMCommandResults returns the execution results of a command identified by a CommandUUID.
	GetMDMCommandResults(ctx context.Context, commandUUID string) ([]*MDMCommandResult, error)

	// ListMDMCommands returns MDM commands based on the provided options.
	ListMDMCommands(ctx context.Context, opts *MDMCommandListOptions) ([]*MDMCommand, error)

	// Set or update the disk encryption key for a host.
	SetOrUpdateDiskEncryptionKey(ctx context.Context, encryptionKey, clientError string) error

	// GetMDMWindowsConfigProfile retrieves the specified configuration profile.
	GetMDMWindowsConfigProfile(ctx context.Context, profileUUID string) (*MDMWindowsConfigProfile, error)

	// DeleteMDMWindowsConfigProfile deletes the specified windows profile.
	DeleteMDMWindowsConfigProfile(ctx context.Context, profileUUID string) error

	// GetMDMWindowsProfileSummary summarizes the current state of MDM configuration profiles on each host
	// in the specified team (or, if no team is specified, each host that is not assigned to any team).
	GetMDMWindowsProfilesSummary(ctx context.Context, teamID *uint) (*MDMProfilesSummary, error)

	// NewMDMWindowsConfigProfile creates a new Windows configuration profile for
	// the specified team.
	NewMDMWindowsConfigProfile(ctx context.Context, teamID uint, profileName string, r io.Reader, labels []string, labelsMembershipMode MDMLabelsMode) (*MDMWindowsConfigProfile, error)

	// NewMDMUnsupportedConfigProfile is called when a profile with an
	// unsupported extension is uploaded.
	NewMDMUnsupportedConfigProfile(ctx context.Context, teamID uint, filename string) error

	// ListMDMConfigProfiles returns a list of paginated configuration profiles.
	ListMDMConfigProfiles(ctx context.Context, teamID *uint, opt ListOptions) ([]*MDMConfigProfilePayload, *PaginationMetadata, error)

	// BatchSetMDMProfiles replaces the custom Windows/macOS profiles for a specified
	// team or for hosts with no team.
	BatchSetMDMProfiles(
		ctx context.Context, teamID *uint, teamName *string, profiles []MDMProfileBatchPayload, dryRun bool, skipBulkPending bool,
		assumeEnabled *bool,
	) error

	///////////////////////////////////////////////////////////////////////////////
	// Linux MDM

	// LinuxHostDiskEncryptionStatus returns the current disk encryption status of the specified Linux host
	// Returns empty status if the host is not a supported Linux host
	LinuxHostDiskEncryptionStatus(ctx context.Context, host Host) (HostMDMDiskEncryption, error)

	// GetMDMLinuxProfilesSummary summarizes the current status of Linux disk encryption for
	// the provided team (or hosts without a team if teamId is nil), or returns zeroes if disk
	// encryption is not enforced on the selected team
	GetMDMLinuxProfilesSummary(ctx context.Context, teamId *uint) (MDMProfilesSummary, error)

	///////////////////////////////////////////////////////////////////////////////
	// Common MDM

	// GetMDMDiskEncryptionSummary returns the current disk encryption status of all macOS, Windows, and
	// Linux hosts in the specified team (or, if no team is specified, each host that is not
	// assigned to any team).
	GetMDMDiskEncryptionSummary(ctx context.Context, teamID *uint) (*MDMDiskEncryptionSummary, error)

	// ResendHostMDMProfile resends the MDM profile to the host.
	ResendHostMDMProfile(ctx context.Context, hostID uint, profileUUID string) error

	///////////////////////////////////////////////////////////////////////////////
	// Host Script Execution

	// RunHostScript executes a script on a host and optionally waits for the
	// result if waitForResult is > 0. If it times out waiting for a result, it
	// fails with a 504 Gateway Timeout error.
	RunHostScript(ctx context.Context, request *HostScriptRequestPayload, waitForResult time.Duration) (*HostScriptResult, error)

	// GetScriptIDByName returns the ID of a script matching the provided name and team. If no team
	// is provided, it will return the ID of the script with the provided name that is not
	// associated with any team.
	GetScriptIDByName(ctx context.Context, name string, teamID *uint) (uint, error)

	// GetHostScript returns information about a host script execution.
	GetHostScript(ctx context.Context, execID string) (*HostScriptResult, error)

	// SaveHostScriptResult saves information about execution of a script on a host.
	SaveHostScriptResult(ctx context.Context, result *HostScriptResultPayload) error

	// GetScriptResult returns the result of a script run
	GetScriptResult(ctx context.Context, execID string) (*HostScriptResult, error)

	// NewScript creates a new (saved) script with its content provided by the
	// io.Reader r.
	NewScript(ctx context.Context, teamID *uint, name string, r io.Reader) (*Script, error)

	// UpdateScript updates a saved script with the contents of io.Reader r
	UpdateScript(ctx context.Context, scriptID uint, r io.Reader) (*Script, error)

	// DeleteScript deletes an existing (saved) script.
	DeleteScript(ctx context.Context, scriptID uint) error

	// ListScripts returns a list of paginated saved scripts.
	ListScripts(ctx context.Context, teamID *uint, opt ListOptions) ([]*Script, *PaginationMetadata, error)

	// GetScript returns the script corresponding to the provided id. If the
	// download is requested, it also returns the script's contents.
	GetScript(ctx context.Context, scriptID uint, downloadRequested bool) (*Script, []byte, error)

	// GetHostScriptDetails returns a list of scripts that apply to the provided host.
	GetHostScriptDetails(ctx context.Context, hostID uint, opt ListOptions) ([]*HostScriptDetail, *PaginationMetadata, error)

	// BatchSetScripts replaces the scripts for a specified team or for
	// hosts with no team.
	BatchSetScripts(ctx context.Context, maybeTmID *uint, maybeTmName *string, payloads []ScriptPayload, dryRun bool) ([]ScriptResponse, error)

	// Script-based methods (at least for some platforms, MDM-based for others)
	LockHost(ctx context.Context, hostID uint, viewPIN bool) (unlockPIN string, err error)
	UnlockHost(ctx context.Context, hostID uint) (unlockPIN string, err error)
	WipeHost(ctx context.Context, hostID uint) error

	///////////////////////////////////////////////////////////////////////////////
	// Software installers
	//

	UploadSoftwareInstaller(ctx context.Context, payload *UploadSoftwareInstallerPayload) error
	UpdateSoftwareInstaller(ctx context.Context, payload *UpdateSoftwareInstallerPayload) (*SoftwareInstaller, error)
	DeleteSoftwareInstaller(ctx context.Context, titleID uint, teamID *uint) error
	GenerateSoftwareInstallerToken(ctx context.Context, alt string, titleID uint, teamID *uint) (string, error)
	GetSoftwareInstallerTokenMetadata(ctx context.Context, token string, titleID uint) (*SoftwareInstallerTokenMetadata, error)
	GetSoftwareInstallerMetadata(ctx context.Context, skipAuthz bool, titleID uint, teamID *uint) (*SoftwareInstaller, error)
	DownloadSoftwareInstaller(ctx context.Context, skipAuthz bool, alt string, titleID uint,
		teamID *uint) (*DownloadSoftwareInstallerPayload, error)
	OrbitDownloadSoftwareInstaller(ctx context.Context, installerID uint) (*DownloadSoftwareInstallerPayload, error)

	////////////////////////////////////////////////////////////////////////////////
	// Setup Experience

	SetSetupExperienceSoftware(ctx context.Context, teamID uint, titleIDs []uint) error
	ListSetupExperienceSoftware(ctx context.Context, teamID uint, opts ListOptions) ([]SoftwareTitleListResult, int, *PaginationMetadata, error)
	// GetOrbitSetupExperienceStatus gets the current status of a macOS setup experience for the given host.
	GetOrbitSetupExperienceStatus(ctx context.Context, orbitNodeKey string, forceRelease bool) (*SetupExperienceStatusPayload, error)
	// GetSetupExperienceScript gets the current setup experience script for the given team.
	GetSetupExperienceScript(ctx context.Context, teamID *uint, downloadRequested bool) (*Script, []byte, error)
	// SetSetupExperienceScript sets the setup experience script for the given team.
	SetSetupExperienceScript(ctx context.Context, teamID *uint, name string, r io.Reader) error
	// DeleteSetupExperienceScript deletes the setup experience script for the given team.
	DeleteSetupExperienceScript(ctx context.Context, teamID *uint) error
	// SetupExperienceNextStep is a callback that processes the
	// setup experience status results table and enqueues the next
	// step. It returns true when there is nothing left to do (setup finished)
	SetupExperienceNextStep(ctx context.Context, hostUUID string) (bool, error)

	///////////////////////////////////////////////////////////////////////////////
	// Fleet-maintained apps

	// AddFleetMaintainedApp adds a Fleet-maintained app to the given team.
	AddFleetMaintainedApp(ctx context.Context, teamID *uint, appID uint, installScript, preInstallQuery, postInstallScript, uninstallScript string, selfService bool, labelsIncludeAny, labelsExcludeAny []string) (uint, error)
	// ListFleetMaintainedApps lists Fleet-maintained apps available to a specific team
	ListFleetMaintainedApps(ctx context.Context, teamID *uint, opts ListOptions) ([]MaintainedApp, *PaginationMetadata, error)
	// GetFleetMaintainedApp returns a Fleet-maintained app by ID
	GetFleetMaintainedApp(ctx context.Context, appID uint) (*MaintainedApp, error)

	// /////////////////////////////////////////////////////////////////////////////
	// Maintenance windows

	// CalendarWebhook handles incoming calendar callback requests.
	CalendarWebhook(ctx context.Context, eventUUID string, channelID string, resourceState string) error

	// /////////////////////////////////////////////////////////////////////////////
	// Secret variables

	// CreateSecretVariables creates secret variables for scripts and profiles.
	CreateSecretVariables(ctx context.Context, secretVariables []SecretVariable, dryRun bool) error
}

type KeyValueStore interface {
	Set(ctx context.Context, key string, value string, expireTime time.Duration) error
	Get(ctx context.Context, key string) (*string, error)
}

const (
	// BatchSetSoftwareInstallerStatusProcessing is the value returned for an ongoing BatchSetSoftwareInstallers operation.
	BatchSetSoftwareInstallersStatusProcessing = "processing"
	// BatchSetSoftwareInstallerStatusCompleted is the value returned for a completed BatchSetSoftwareInstallers operation.
	BatchSetSoftwareInstallersStatusCompleted = "completed"
	// BatchSetSoftwareInstallerStatusFailed is the value returned for a failed BatchSetSoftwareInstallers operation.
	BatchSetSoftwareInstallersStatusFailed = "failed"
	// MinOrbitLUKSVersion is the earliest version of Orbit that can escrow LUKS passphrases
	MinOrbitLUKSVersion = "1.36.0"
	// MFALinkTTL is how long MFA verification links stay active
	MFALinkTTL = time.Minute * 15
)
