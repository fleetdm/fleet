package fleet

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fleetdm/fleet/v4/server/websocket"
	"github.com/kolide/kit/version"
)

type OsqueryService interface {
	EnrollAgent(
		ctx context.Context, enrollSecret, hostIdentifier string, hostDetails map[string](map[string]string),
	) (nodeKey string, err error)
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
	GetDistributedQueries(ctx context.Context) (queries map[string]string, accelerate uint, err error)
	SubmitDistributedQueryResults(
		ctx context.Context,
		results OsqueryDistributedQueryResults,
		statuses map[string]OsqueryStatus,
		messages map[string]string,
	) (err error)
	SubmitStatusLogs(ctx context.Context, logs []json.RawMessage) (err error)
	SubmitResultLogs(ctx context.Context, logs []json.RawMessage) (err error)
}

type Service interface {
	OsqueryService

	///////////////////////////////////////////////////////////////////////////////
	// UserService contains methods for managing a Fleet User.

	// CreateUserFromInvite creates a new User from a request payload when there is already an existing invitation.
	CreateUserFromInvite(ctx context.Context, p UserPayload) (user *User, err error)

	// CreateUser allows an admin to create a new user without first creating  and validating invite tokens.
	CreateUser(ctx context.Context, p UserPayload) (user *User, err error)

	// CreateInitialUser creates the first user, skipping authorization checks.  If a user already exists this method
	// should fail.
	CreateInitialUser(ctx context.Context, p UserPayload) (user *User, err error)

	// User returns a valid User given a User ID.
	User(ctx context.Context, id uint) (user *User, err error)

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

	///////////////////////////////////////////////////////////////////////////////
	// Session

	// InitiateSSO is used to initiate an SSO session and returns a URL that can be used in a redirect to the IDP.
	// Arguments: redirectURL is the URL of the protected resource that the user was trying to access when they were
	// prompted to log in.
	InitiateSSO(ctx context.Context, redirectURL string) (string, error)

	// CallbackSSO handles the IDP response. The original URL the viewer attempted to access is returned from this
	// function, so we can redirect back to the front end and load the page the viewer originally attempted to access
	// when prompted for login.
	CallbackSSO(ctx context.Context, auth Auth) (*SSOSession, error)

	// SSOSettings returns non-sensitive single sign on information used before authentication
	SSOSettings(ctx context.Context) (*SessionSSOSettings, error)
	Login(ctx context.Context, email, password string) (user *User, sessionKey string, err error)
	Logout(ctx context.Context) (err error)
	DestroySession(ctx context.Context) (err error)
	GetInfoAboutSessionsForUser(ctx context.Context, id uint) (sessions []*Session, err error)
	DeleteSessionsForUser(ctx context.Context, id uint) (err error)
	GetInfoAboutSession(ctx context.Context, id uint) (session *Session, err error)
	GetSessionByKey(ctx context.Context, key string) (session *Session, err error)
	DeleteSession(ctx context.Context, id uint) (err error)

	///////////////////////////////////////////////////////////////////////////////
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

	///////////////////////////////////////////////////////////////////////////////
	// LabelService

	// ApplyLabelSpecs applies a list of LabelSpecs to the datastore, creating and updating labels as necessary.
	ApplyLabelSpecs(ctx context.Context, specs []*LabelSpec) error
	// GetLabelSpecs returns all of the stored LabelSpecs.
	GetLabelSpecs(ctx context.Context) ([]*LabelSpec, error)
	// GetLabelSpec gets the spec for the label with the given name.
	GetLabelSpec(ctx context.Context, name string) (*LabelSpec, error)

	NewLabel(ctx context.Context, p LabelPayload) (label *Label, err error)
	ModifyLabel(ctx context.Context, id uint, payload ModifyLabelPayload) (*Label, error)
	ListLabels(ctx context.Context, opt ListOptions) (labels []*Label, err error)
	GetLabel(ctx context.Context, id uint) (label *Label, err error)

	DeleteLabel(ctx context.Context, name string) (err error)
	// DeleteLabelByID is for backwards compatibility with the UI
	DeleteLabelByID(ctx context.Context, id uint) (err error)

	// ListHostsInLabel returns a slice of hosts in the label with the given ID.
	ListHostsInLabel(ctx context.Context, lid uint, opt HostListOptions) ([]*Host, error)

	// ListLabelsForHost returns the labels that the given host is in.
	ListLabelsForHost(ctx context.Context, hid uint) ([]*Label, error)

	///////////////////////////////////////////////////////////////////////////////
	// QueryService

	// ApplyQuerySpecs applies a list of queries (creating or updating them as necessary)
	ApplyQuerySpecs(ctx context.Context, specs []*QuerySpec) error
	// GetQuerySpecs gets the YAML file representing all the stored queries.
	GetQuerySpecs(ctx context.Context) ([]*QuerySpec, error)
	// GetQuerySpec gets the spec for the query with the given name.
	GetQuerySpec(ctx context.Context, name string) (*QuerySpec, error)

	// ListQueries returns a list of saved queries. Note only saved queries should be returned (those that are created
	// for distributed queries but not saved should not be returned).
	ListQueries(ctx context.Context, opt ListOptions) ([]*Query, error)
	GetQuery(ctx context.Context, id uint) (*Query, error)
	NewQuery(ctx context.Context, p QueryPayload) (*Query, error)
	ModifyQuery(ctx context.Context, id uint, p QueryPayload) (*Query, error)
	DeleteQuery(ctx context.Context, name string) error
	// DeleteQueryByID deletes a query by ID. For backwards compatibility with UI
	DeleteQueryByID(ctx context.Context, id uint) error
	// DeleteQueries deletes the existing query objects with the provided IDs. The number of deleted queries is returned
	// along with any error.
	DeleteQueries(ctx context.Context, ids []uint) (uint, error)

	///////////////////////////////////////////////////////////////////////////////
	// CampaignService defines the distributed query campaign related service methods

	// NewDistributedQueryCampaignByNames creates a new distributed query campaign with the provided query (or the query
	// referenced by ID) and host/label targets (specified by name).
	NewDistributedQueryCampaignByNames(
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
	RunLiveQueryDeadline(ctx context.Context, queryIDs []uint, hostIDs []uint, deadline time.Duration) ([]QueryCampaignResult, int)

	///////////////////////////////////////////////////////////////////////////////
	// AgentOptionsService

	// AgentOptionsForHost gets the agent options for the provided host. The host information should be used for
	// filtering based on team, platform, etc.
	AgentOptionsForHost(ctx context.Context, host *Host) (json.RawMessage, error)

	///////////////////////////////////////////////////////////////////////////////
	// HostService

	ListHosts(ctx context.Context, opt HostListOptions) (hosts []*Host, err error)
	GetHost(ctx context.Context, id uint) (host *HostDetail, err error)
	GetHostSummary(ctx context.Context, teamID *uint) (summary *HostSummary, err error)
	DeleteHost(ctx context.Context, id uint) (err error)
	// HostByIdentifier returns one host matching the provided identifier. Possible matches can be on
	// osquery_host_identifier, node_key, UUID, or hostname.
	HostByIdentifier(ctx context.Context, identifier string) (*HostDetail, error)
	// RefetchHost requests a refetch of host details for the provided host.
	RefetchHost(ctx context.Context, id uint) (err error)

	FlushSeenHosts(ctx context.Context) error
	// AddHostsToTeam adds hosts to an existing team, clearing their team settings if teamID is nil.
	AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint) error
	// AddHostsToTeamByFilter adds hosts to an existing team, clearing their team settings if teamID is nil. Hosts are
	// selected by the label and HostListOptions provided.
	AddHostsToTeamByFilter(ctx context.Context, teamID *uint, opt HostListOptions, lid *uint) error
	DeleteHosts(ctx context.Context, ids []uint, opt HostListOptions, lid *uint) error
	CountHosts(ctx context.Context, labelID *uint, opts HostListOptions) (int, error)

	///////////////////////////////////////////////////////////////////////////////
	// AppConfigService provides methods for configuring  the Fleet application

	NewAppConfig(ctx context.Context, p AppConfig) (info *AppConfig, err error)
	AppConfig(ctx context.Context) (info *AppConfig, err error)
	ModifyAppConfig(ctx context.Context, p []byte) (info *AppConfig, err error)

	// ApplyEnrollSecretSpec adds and updates the enroll secrets specified in the spec.
	ApplyEnrollSecretSpec(ctx context.Context, spec *EnrollSecretSpec) error
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

	// UpdateIntervalConfig returns the duration for different update intervals configured in osquery
	UpdateIntervalConfig(ctx context.Context) (*UpdateIntervalConfig, error)

	// VulnerabilitiesConfig returns the vulnerabilities checks configuration for
	// the fleet instance.
	VulnerabilitiesConfig(ctx context.Context) (*VulnerabilitiesConfig, error)

	///////////////////////////////////////////////////////////////////////////////
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

	///////////////////////////////////////////////////////////////////////////////
	// TargetService

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

	///////////////////////////////////////////////////////////////////////////////
	// ScheduledQueryService

	GetScheduledQueriesInPack(ctx context.Context, id uint, opts ListOptions) (queries []*ScheduledQuery, err error)
	GetScheduledQuery(ctx context.Context, id uint) (query *ScheduledQuery, err error)
	ScheduleQuery(ctx context.Context, sq *ScheduledQuery) (query *ScheduledQuery, err error)
	DeleteScheduledQuery(ctx context.Context, id uint) (err error)
	ModifyScheduledQuery(ctx context.Context, id uint, p ScheduledQueryPayload) (query *ScheduledQuery, err error)

	///////////////////////////////////////////////////////////////////////////////
	// StatusService

	// StatusResultStore returns nil if the result store is functioning correctly, or an error indicating the problem.
	StatusResultStore(ctx context.Context) error

	// StatusLiveQuery returns nil if live queries are enabled, or an
	// error indicating the problem.
	StatusLiveQuery(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// CarveService

	CarveBegin(ctx context.Context, payload CarveBeginPayload) (*CarveMetadata, error)
	CarveBlock(ctx context.Context, payload CarveBlockPayload) error
	GetCarve(ctx context.Context, id int64) (*CarveMetadata, error)
	ListCarves(ctx context.Context, opt CarveListOptions) ([]*CarveMetadata, error)
	GetBlock(ctx context.Context, carveId, blockId int64) ([]byte, error)

	///////////////////////////////////////////////////////////////////////////////
	// TeamService

	// NewTeam creates a new team.
	NewTeam(ctx context.Context, p TeamPayload) (*Team, error)
	// ModifyTeam modifies an existing team (besides agent options).
	ModifyTeam(ctx context.Context, id uint, payload TeamPayload) (*Team, error)
	// ModifyTeamAgentOptions modifies agent options for a team.
	ModifyTeamAgentOptions(ctx context.Context, id uint, options json.RawMessage) (*Team, error)
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
	// TeamEnrollSecrets lists the enroll secrets for the team.
	TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*EnrollSecret, error)
	// ModifyTeamEnrollSecrets modifies enroll secrets for a team.
	ModifyTeamEnrollSecrets(ctx context.Context, teamID uint, secrets []EnrollSecret) ([]*EnrollSecret, error)
	// ApplyTeamSpecs applies the changes for each team as defined in the specs.
	ApplyTeamSpecs(ctx context.Context, specs []*TeamSpec) error

	///////////////////////////////////////////////////////////////////////////////
	// ActivitiesService

	ListActivities(ctx context.Context, opt ListOptions) ([]*Activity, error)

	///////////////////////////////////////////////////////////////////////////////
	// UserRolesService

	// ApplyUserRolesSpecs applies a list of user global and team role changes
	ApplyUserRolesSpecs(ctx context.Context, specs UsersRoleSpec) error

	///////////////////////////////////////////////////////////////////////////////
	// GlobalScheduleService

	GlobalScheduleQuery(ctx context.Context, sq *ScheduledQuery) (*ScheduledQuery, error)
	GetGlobalScheduledQueries(ctx context.Context, opts ListOptions) ([]*ScheduledQuery, error)
	ModifyGlobalScheduledQueries(ctx context.Context, id uint, q ScheduledQueryPayload) (*ScheduledQuery, error)
	DeleteGlobalScheduledQueries(ctx context.Context, id uint) error

	///////////////////////////////////////////////////////////////////////////////
	// TranslatorService

	Translate(ctx context.Context, payloads []TranslatePayload) ([]TranslatePayload, error)

	///////////////////////////////////////////////////////////////////////////////
	// TeamScheduleService

	TeamScheduleQuery(ctx context.Context, teamID uint, sq *ScheduledQuery) (*ScheduledQuery, error)
	GetTeamScheduledQueries(ctx context.Context, teamID uint, opts ListOptions) ([]*ScheduledQuery, error)
	ModifyTeamScheduledQueries(
		ctx context.Context, teamID uint, scheduledQueryID uint, q ScheduledQueryPayload,
	) (*ScheduledQuery, error)
	DeleteTeamScheduledQueries(ctx context.Context, teamID uint, id uint) error

	///////////////////////////////////////////////////////////////////////////////
	// GlobalPolicyService

	NewGlobalPolicy(ctx context.Context, p PolicyPayload) (*Policy, error)
	ListGlobalPolicies(ctx context.Context) ([]*Policy, error)
	DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error)
	ModifyGlobalPolicy(ctx context.Context, id uint, p ModifyPolicyPayload) (*Policy, error)
	GetPolicyByIDQueries(ctx context.Context, policyID uint) (*Policy, error)
	ApplyPolicySpecs(ctx context.Context, policies []*PolicySpec) error

	///////////////////////////////////////////////////////////////////////////////
	// Software

	ListSoftware(ctx context.Context, opt SoftwareListOptions) ([]Software, error)
	SoftwareByID(ctx context.Context, id uint) (*Software, error)
	CountSoftware(ctx context.Context, opt SoftwareListOptions) (int, error)

	///////////////////////////////////////////////////////////////////////////////
	// Team Policies

	NewTeamPolicy(ctx context.Context, teamID uint, p PolicyPayload) (*Policy, error)
	ListTeamPolicies(ctx context.Context, teamID uint) ([]*Policy, error)
	DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error)
	ModifyTeamPolicy(ctx context.Context, teamID uint, id uint, p ModifyPolicyPayload) (*Policy, error)
	GetTeamPolicyByIDQueries(ctx context.Context, teamID uint, policyID uint) (*Policy, error)
}
