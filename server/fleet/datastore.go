package fleet

import (
	"context"
	"time"
)

type CarveStore interface {
	NewCarve(metadata *CarveMetadata) (*CarveMetadata, error)
	UpdateCarve(metadata *CarveMetadata) error
	Carve(carveId int64) (*CarveMetadata, error)
	CarveBySessionId(sessionId string) (*CarveMetadata, error)
	CarveByName(name string) (*CarveMetadata, error)
	ListCarves(opt CarveListOptions) ([]*CarveMetadata, error)
	NewBlock(metadata *CarveMetadata, blockId int64, data []byte) error
	GetBlock(metadata *CarveMetadata, blockId int64) ([]byte, error)
	// CleanupCarves will mark carves older than 24 hours expired, and delete the associated data blocks. This behaves
	// differently for carves stored in S3 (check the implementation godoc comment for more details)
	CleanupCarves(now time.Time) (expired int, err error)
}

// Datastore combines all the interfaces in the Fleet DAL
type Datastore interface {
	CarveStore

	///////////////////////////////////////////////////////////////////////////////
	// UserStore contains methods for managing users in a datastore

	NewUser(ctx context.Context, user *User) (*User, error)
	ListUsers(ctx context.Context, opt UserListOptions) ([]*User, error)
	UserByEmail(ctx context.Context, email string) (*User, error)
	UserByID(ctx context.Context, id uint) (*User, error)
	SaveUser(ctx context.Context, user *User) error
	SaveUsers(ctx context.Context, users []*User) error
	// DeleteUser permanently deletes the user identified by the provided ID.
	DeleteUser(ctx context.Context, id uint) error
	// PendingEmailChange creates a record with a pending email change for a user identified by uid. The change record
	// is keyed by a unique token. The token is emailed to the user with a link that they can use to confirm the change.
	PendingEmailChange(ctx context.Context, userID uint, newEmail, token string) error
	// ConfirmPendingEmailChange will confirm new email address identified by token is valid. The new email will be
	// written to user record. userID is the ID of the user whose e-mail is being changed.
	ConfirmPendingEmailChange(ctx context.Context, userID uint, token string) (string, error)

	///////////////////////////////////////////////////////////////////////////////
	// QueryStore

	// ApplyQueries applies a list of queries (likely from a yaml file) to the datastore. Existing queries are updated,
	// and new queries are created.
	ApplyQueries(ctx context.Context, authorID uint, queries []*Query) error

	// NewQuery creates a new query object in thie datastore. The returned query should have the ID updated.
	NewQuery(ctx context.Context, query *Query, opts ...OptionalArg) (*Query, error)
	// SaveQuery saves changes to an existing query object.
	SaveQuery(ctx context.Context, query *Query) error
	// DeleteQuery deletes an existing query object.
	DeleteQuery(ctx context.Context, name string) error
	// DeleteQueries deletes the existing query objects with the provided IDs. The number of deleted queries is returned
	// along with any error.
	DeleteQueries(ctx context.Context, ids []uint) (uint, error)
	// Query returns the query associated with the provided ID. Associated packs should also be loaded.
	Query(ctx context.Context, id uint) (*Query, error)
	// ListQueries returns a list of queries with the provided sorting and paging options. Associated packs should also
	// be loaded.
	ListQueries(ctx context.Context, opt ListOptions) ([]*Query, error)
	// QueryByName looks up a query by name.
	QueryByName(ctx context.Context, name string, opts ...OptionalArg) (*Query, error)

	///////////////////////////////////////////////////////////////////////////////
	// CampaignStore defines the distributed query campaign related datastore methods

	// NewDistributedQueryCampaign creates a new distributed query campaign
	NewDistributedQueryCampaign(camp *DistributedQueryCampaign) (*DistributedQueryCampaign, error)
	// DistributedQueryCampaign loads a distributed query campaign by ID
	DistributedQueryCampaign(id uint) (*DistributedQueryCampaign, error)
	// SaveDistributedQueryCampaign updates an existing distributed query campaign
	SaveDistributedQueryCampaign(camp *DistributedQueryCampaign) error
	// DistributedQueryCampaignTargetIDs gets the IDs of the targets for the query campaign of the provided ID
	DistributedQueryCampaignTargetIDs(id uint) (targets *HostTargets, err error)

	// NewDistributedQueryCampaignTarget adds a new target to an existing distributed query campaign
	NewDistributedQueryCampaignTarget(target *DistributedQueryCampaignTarget) (*DistributedQueryCampaignTarget, error)

	// CleanupDistributedQueryCampaigns will clean and trim metadata for old distributed query campaigns. Any campaign
	// in the QueryWaiting state will be moved to QueryComplete after one minute. Any campaign in the QueryRunning state
	// will be moved to QueryComplete after one day. Times are from creation time. The now parameter makes this method
	// easier to test. The return values indicate how many campaigns were expired and any error.
	CleanupDistributedQueryCampaigns(now time.Time) (expired uint, err error)

	///////////////////////////////////////////////////////////////////////////////
	// PackStore is the datastore interface for managing query packs.

	// ApplyPackSpecs applies a list of PackSpecs to the datastore, creating and updating packs as necessary.
	ApplyPackSpecs(specs []*PackSpec) error
	// GetPackSpecs returns all of the stored PackSpecs.
	GetPackSpecs() ([]*PackSpec, error)
	// GetPackSpec returns the spec for the named pack.
	GetPackSpec(name string) (*PackSpec, error)

	// NewPack creates a new pack in the datastore.
	NewPack(pack *Pack, opts ...OptionalArg) (*Pack, error)

	// SavePack updates an existing pack in the datastore.
	SavePack(pack *Pack) error

	// DeletePack deletes a pack record from the datastore.
	DeletePack(name string) error

	// Pack retrieves a pack from the datastore by ID.
	Pack(pid uint) (*Pack, error)

	// ListPacks lists all packs in the datastore.
	ListPacks(opt PackListOptions) ([]*Pack, error)

	// PackByName fetches pack if it exists, if the pack exists the bool return value is true
	PackByName(name string, opts ...OptionalArg) (*Pack, bool, error)

	// ListPacksForHost lists the packs that a host should execute.
	ListPacksForHost(hid uint) (packs []*Pack, err error)

	// EnsureGlobalPack gets or inserts a pack with type global
	EnsureGlobalPack() (*Pack, error)

	// EnsureTeamPack gets or inserts a pack with type global
	EnsureTeamPack(teamID uint) (*Pack, error)

	///////////////////////////////////////////////////////////////////////////////
	// LabelStore

	// ApplyLabelSpecs applies a list of LabelSpecs to the datastore, creating and updating labels as necessary.
	ApplyLabelSpecs(specs []*LabelSpec) error
	// GetLabelSpecs returns all of the stored LabelSpecs.
	GetLabelSpecs() ([]*LabelSpec, error)
	// GetLabelSpec returns the spec for the named label.
	GetLabelSpec(name string) (*LabelSpec, error)

	NewLabel(Label *Label, opts ...OptionalArg) (*Label, error)
	SaveLabel(label *Label) (*Label, error)
	DeleteLabel(name string) error
	Label(lid uint) (*Label, error)
	ListLabels(filter TeamFilter, opt ListOptions) ([]*Label, error)

	// LabelQueriesForHost returns the label queries that should be executed for the given host. The cutoff is the
	// minimum timestamp a query execution should have to be considered "fresh". Executions that are not fresh will be
	// repeated. Results are returned in a map of label id -> query
	LabelQueriesForHost(host *Host, cutoff time.Time) (map[string]string, error)

	// RecordLabelQueryExecutions saves the results of label queries. The results map is a map of label id -> whether or
	// not the label matches. The time parameter is the timestamp to save with the query execution.
	RecordLabelQueryExecutions(host *Host, results map[uint]*bool, t time.Time) error

	// ListLabelsForHost returns the labels that the given host is in.
	ListLabelsForHost(hid uint) ([]*Label, error)

	// ListHostsInLabel returns a slice of hosts in the label with the given ID.
	ListHostsInLabel(filter TeamFilter, lid uint, opt HostListOptions) ([]*Host, error)

	// ListUniqueHostsInLabels returns a slice of all of the hosts in the given label IDs. A host will only appear once
	// in the results even if it is in multiple of the provided labels.
	ListUniqueHostsInLabels(filter TeamFilter, labels []uint) ([]*Host, error)

	SearchLabels(filter TeamFilter, query string, omit ...uint) ([]*Label, error)

	// LabelIDsByName Retrieve the IDs associated with the given labels
	LabelIDsByName(labels []string) ([]uint, error)

	///////////////////////////////////////////////////////////////////////////////
	// HostStore

	// NewHost is deprecated and will be removed. Hosts should always be enrolled via EnrollHost.
	NewHost(host *Host) (*Host, error)
	SaveHost(host *Host) error
	DeleteHost(hid uint) error
	Host(id uint) (*Host, error)
	// EnrollHost will enroll a new host with the given identifier, setting the node key, and team. Implementations of
	// this method should respect the provided host enrollment cooldown, by returning an error if the host has enrolled
	// within the cooldown period.
	EnrollHost(osqueryHostId, nodeKey string, teamID *uint, cooldown time.Duration) (*Host, error)
	ListHosts(filter TeamFilter, opt HostListOptions) ([]*Host, error)
	// AuthenticateHost authenticates and returns host metadata by node key. This method should not return the host
	// "additional" information as this is not typically necessary for the operations performed by the osquery
	// endpoints.
	AuthenticateHost(nodeKey string) (*Host, error)
	MarkHostSeen(host *Host, t time.Time) error
	MarkHostsSeen(hostIDs []uint, t time.Time) error
	SearchHosts(filter TeamFilter, query string, omit ...uint) ([]*Host, error)
	// CleanupIncomingHosts deletes hosts that have enrolled but never updated their status details. This clears dead
	// "incoming hosts" that never complete their registration.
	// A host is considered incoming if both the hostname and osquery_version fields are empty. This means that multiple
	// different osquery queries failed to populate details.
	CleanupIncomingHosts(now time.Time) error
	// GenerateHostStatusStatistics retrieves the count of online, offline, MIA and new hosts.
	GenerateHostStatusStatistics(filter TeamFilter, now time.Time) (online, offline, mia, new uint, err error)
	// HostIDsByName Retrieve the IDs associated with the given hostnames
	HostIDsByName(filter TeamFilter, hostnames []string) ([]uint, error)
	// HostByIdentifier returns one host matching the provided identifier. Possible matches can be on
	// osquery_host_identifier, node_key, UUID, or hostname.
	HostByIdentifier(identifier string) (*Host, error)
	// AddHostsToTeam adds hosts to an existing team, clearing their team settings if teamID is nil.
	AddHostsToTeam(teamID *uint, hostIDs []uint) error

	TotalAndUnseenHostsSince(daysCount int) (int, int, error)

	///////////////////////////////////////////////////////////////////////////////
	// TargetStore

	// CountHostsInTargets returns the metrics of the hosts in the provided labels, teams, and explicit host IDs.
	CountHostsInTargets(filter TeamFilter, targets HostTargets, now time.Time) (TargetMetrics, error)
	// HostIDsInTargets returns the host IDs of the hosts in the provided labels, teams, and explicit host IDs. The
	// returned host IDs should be sorted in ascending order.
	HostIDsInTargets(filter TeamFilter, targets HostTargets) ([]uint, error)

	///////////////////////////////////////////////////////////////////////////////
	// PasswordResetStore manages password resets in the Datastore

	NewPasswordResetRequest(req *PasswordResetRequest) (*PasswordResetRequest, error)
	DeletePasswordResetRequestsForUser(userID uint) error
	FindPassswordResetByToken(token string) (*PasswordResetRequest, error)

	///////////////////////////////////////////////////////////////////////////////
	// SessionStore is the abstract interface that all session backends must conform to.

	// SessionByKey returns, given a session key, a session object or an error if one could not be found for the given
	// key
	SessionByKey(key string) (*Session, error)

	// SessionByID returns, given a session id, find and return a session object or an error if one could not be found
	// for the given id
	SessionByID(id uint) (*Session, error)

	// ListSessionsForUser finds all the active sessions for a given user
	ListSessionsForUser(id uint) ([]*Session, error)

	// NewSession stores a new session struct
	NewSession(session *Session) (*Session, error)

	// DestroySession destroys the currently tracked session
	DestroySession(session *Session) error

	// DestroyAllSessionsForUser destroys all of the sessions for a given user
	DestroyAllSessionsForUser(id uint) error

	// MarkSessionAccessed marks the currently tracked session as access to extend expiration
	MarkSessionAccessed(session *Session) error

	///////////////////////////////////////////////////////////////////////////////
	// AppConfigStore contains method for saving and retrieving application configuration

	NewAppConfig(info *AppConfig) (*AppConfig, error)
	AppConfig() (*AppConfig, error)
	SaveAppConfig(info *AppConfig) error

	// VerifyEnrollSecret checks that the provided secret matches an active enroll secret. If it is successfully
	// matched, that secret is returned. Otherwise, an error is returned.
	VerifyEnrollSecret(secret string) (*EnrollSecret, error)
	// GetEnrollSecrets gets the enroll secrets for a team (or global if teamID is nil).
	GetEnrollSecrets(teamID *uint) ([]*EnrollSecret, error)
	// ApplyEnrollSecrets replaces the current enroll secrets for a team with the provided secrets.
	ApplyEnrollSecrets(teamID *uint, secrets []*EnrollSecret) error

	///////////////////////////////////////////////////////////////////////////////
	// InviteStore contains the methods for managing user invites in a datastore.

	// NewInvite creates and stores a new invitation in a DB.
	NewInvite(i *Invite) (*Invite, error)

	// ListInvites lists all invites in the datastore.
	ListInvites(opt ListOptions) ([]*Invite, error)

	// Invite retrieves an invite by its ID.
	Invite(id uint) (*Invite, error)

	// InviteByEmail retrieves an invite for a specific email address.
	InviteByEmail(email string) (*Invite, error)

	// InviteByToken retrieves and invite using the token string.
	InviteByToken(token string) (*Invite, error)

	// DeleteInvite deletes an invitation.
	DeleteInvite(id uint) error

	///////////////////////////////////////////////////////////////////////////////
	// ScheduledQueryStore

	ListScheduledQueriesInPack(id uint, opts ListOptions) ([]*ScheduledQuery, error)
	NewScheduledQuery(sq *ScheduledQuery, opts ...OptionalArg) (*ScheduledQuery, error)
	SaveScheduledQuery(sq *ScheduledQuery) (*ScheduledQuery, error)
	DeleteScheduledQuery(id uint) error
	ScheduledQuery(id uint) (*ScheduledQuery, error)
	CleanupOrphanScheduledQueryStats() error

	///////////////////////////////////////////////////////////////////////////////
	// TeamStore

	// NewTeam creates a new Team object in the store.
	NewTeam(team *Team) (*Team, error)
	// SaveTeam saves any changes to the team.
	SaveTeam(team *Team) (*Team, error)
	// Team retrieves the Team by ID.
	Team(tid uint) (*Team, error)
	// Team deletes the Team by ID.
	DeleteTeam(tid uint) error
	// TeamByName retrieves the Team by Name.
	TeamByName(name string) (*Team, error)
	// ListTeams lists teams with the ordering and filters in the provided options.
	ListTeams(filter TeamFilter, opt ListOptions) ([]*Team, error)
	// SearchTeams searches teams using the provided query and ommitting the provided existing selection.
	SearchTeams(filter TeamFilter, matchQuery string, omit ...uint) ([]*Team, error)
	// TeamEnrollSecrets lists the enroll secrets for the team.
	TeamEnrollSecrets(teamID uint) ([]*EnrollSecret, error)

	///////////////////////////////////////////////////////////////////////////////
	// SoftwareStore

	SaveHostSoftware(host *Host) error
	LoadHostSoftware(host *Host) error
	AllSoftwareWithoutCPEIterator() (SoftwareIterator, error)
	AddCPEForSoftware(software Software, cpe string) error
	AllCPEs() ([]string, error)
	InsertCVEForCPE(cve string, cpes []string) error

	///////////////////////////////////////////////////////////////////////////////
	// ActivitiesStore

	NewActivity(user *User, activityType string, details *map[string]interface{}) error
	ListActivities(opt ListOptions) ([]*Activity, error)

	///////////////////////////////////////////////////////////////////////////////
	// StatisticsStore

	ShouldSendStatistics(frequency time.Duration) (StatisticsPayload, bool, error)
	RecordStatisticsSent() error

	///////////////////////////////////////////////////////////////////////////////
	// GlobalPoliciesStore interface {
	NewGlobalPolicy(queryID uint) (*Policy, error)
	Policy(id uint) (*Policy, error)
	RecordPolicyQueryExecutions(host *Host, results map[uint]*bool, updated time.Time) error

	ListGlobalPolicies() ([]*Policy, error)
	DeleteGlobalPolicies(ids []uint) ([]uint, error)

	PolicyQueriesForHost(host *Host) (map[string]string, error)

	// MigrateTables creates and migrates the table schemas
	MigrateTables() error
	// MigrateData populates built-in data
	MigrateData() error
	// MigrationStatus returns nil if migrations are complete, and an error if migrations need to be run.
	MigrationStatus() (MigrationStatus, error)
}

type MigrationStatus int

const (
	NoMigrationsCompleted = iota
	SomeMigrationsCompleted
	AllMigrationsCompleted
)

// NotFoundError is returned when the datastore resource cannot be found.
type NotFoundError interface {
	error
	IsNotFound() bool
}

func IsNotFound(err error) bool {
	e, ok := err.(NotFoundError)
	if !ok {
		return false
	}
	return e.IsNotFound()
}

// AlreadyExistsError is returned when creating a datastore resource that already exists.
type AlreadyExistsError interface {
	error
	IsExists() bool
}

// ForeignKeyError is returned when the operation fails due to foreign key constraints.
type ForeignKeyError interface {
	error
	IsForeignKey() bool
}

func IsForeignKey(err error) bool {
	e, ok := err.(ForeignKeyError)
	if !ok {
		return false
	}
	return e.IsForeignKey()
}

type OptionalArg func() interface{}
