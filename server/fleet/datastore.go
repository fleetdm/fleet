package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/micromdm/nanodep/godep"
)

type CarveStore interface {
	NewCarve(ctx context.Context, metadata *CarveMetadata) (*CarveMetadata, error)
	UpdateCarve(ctx context.Context, metadata *CarveMetadata) error
	Carve(ctx context.Context, carveId int64) (*CarveMetadata, error)
	CarveBySessionId(ctx context.Context, sessionId string) (*CarveMetadata, error)
	CarveByName(ctx context.Context, name string) (*CarveMetadata, error)
	ListCarves(ctx context.Context, opt CarveListOptions) ([]*CarveMetadata, error)
	NewBlock(ctx context.Context, metadata *CarveMetadata, blockId int64, data []byte) error
	GetBlock(ctx context.Context, metadata *CarveMetadata, blockId int64) ([]byte, error)
	// CleanupCarves will mark carves older than 24 hours expired, and delete the associated data blocks. This behaves
	// differently for carves stored in S3 (check the implementation godoc comment for more details)
	CleanupCarves(ctx context.Context, now time.Time) (expired int, err error)
}

// InstallerStore is used to communicate to a blob storage containing pre-built
// fleet-osquery installers
type InstallerStore interface {
	Get(ctx context.Context, installer Installer) (io.ReadCloser, int64, error)
	Put(ctx context.Context, installer Installer) (string, error)
	Exists(ctx context.Context, installer Installer) (bool, error)
}

// Datastore combines all the interfaces in the Fleet DAL
type Datastore interface {
	health.Checker

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
	ListQueries(ctx context.Context, opt ListQueryOptions) ([]*Query, error)
	// QueryByName looks up a query by name.
	QueryByName(ctx context.Context, name string, opts ...OptionalArg) (*Query, error)
	// ObserverCanRunQuery returns whether a user with an observer role is permitted to run the
	// identified query
	ObserverCanRunQuery(ctx context.Context, queryID uint) (bool, error)

	///////////////////////////////////////////////////////////////////////////////
	// CampaignStore defines the distributed query campaign related datastore methods

	// NewDistributedQueryCampaign creates a new distributed query campaign
	NewDistributedQueryCampaign(ctx context.Context, camp *DistributedQueryCampaign) (*DistributedQueryCampaign, error)
	// DistributedQueryCampaign loads a distributed query campaign by ID
	DistributedQueryCampaign(ctx context.Context, id uint) (*DistributedQueryCampaign, error)
	// SaveDistributedQueryCampaign updates an existing distributed query campaign
	SaveDistributedQueryCampaign(ctx context.Context, camp *DistributedQueryCampaign) error
	// DistributedQueryCampaignTargetIDs gets the IDs of the targets for the query campaign of the provided ID
	DistributedQueryCampaignTargetIDs(ctx context.Context, id uint) (targets *HostTargets, err error)

	// NewDistributedQueryCampaignTarget adds a new target to an existing distributed query campaign
	NewDistributedQueryCampaignTarget(ctx context.Context, target *DistributedQueryCampaignTarget) (*DistributedQueryCampaignTarget, error)

	// CleanupDistributedQueryCampaigns will clean and trim metadata for old distributed query campaigns. Any campaign
	// in the QueryWaiting state will be moved to QueryComplete after one minute. Any campaign in the QueryRunning state
	// will be moved to QueryComplete after one day. Times are from creation time. The now parameter makes this method
	// easier to test. The return values indicate how many campaigns were expired and any error.
	CleanupDistributedQueryCampaigns(ctx context.Context, now time.Time) (expired uint, err error)

	DistributedQueryCampaignsForQuery(ctx context.Context, queryID uint) ([]*DistributedQueryCampaign, error)

	///////////////////////////////////////////////////////////////////////////////
	// PackStore is the datastore interface for managing query packs.

	// ApplyPackSpecs applies a list of PackSpecs to the datastore, creating and updating packs as necessary.
	ApplyPackSpecs(ctx context.Context, specs []*PackSpec) error
	// GetPackSpecs returns all of the stored PackSpecs.
	GetPackSpecs(ctx context.Context) ([]*PackSpec, error)
	// GetPackSpec returns the spec for the named pack.
	GetPackSpec(ctx context.Context, name string) (*PackSpec, error)

	// NewPack creates a new pack in the datastore.
	NewPack(ctx context.Context, pack *Pack, opts ...OptionalArg) (*Pack, error)

	// SavePack updates an existing pack in the datastore.
	SavePack(ctx context.Context, pack *Pack) error

	// DeletePack deletes a pack record from the datastore.
	DeletePack(ctx context.Context, name string) error

	// Pack retrieves a pack from the datastore by ID.
	Pack(ctx context.Context, pid uint) (*Pack, error)

	// ListPacks lists all packs in the datastore.
	ListPacks(ctx context.Context, opt PackListOptions) ([]*Pack, error)

	// PackByName fetches pack if it exists, if the pack exists the bool return value is true
	PackByName(ctx context.Context, name string, opts ...OptionalArg) (*Pack, bool, error)

	// ListPacksForHost lists the packs that a host should execute.
	ListPacksForHost(ctx context.Context, hid uint) (packs []*Pack, err error)

	// EnsureGlobalPack gets or inserts a pack with type global
	EnsureGlobalPack(ctx context.Context) (*Pack, error)

	// EnsureTeamPack gets or inserts a pack with type global
	EnsureTeamPack(ctx context.Context, teamID uint) (*Pack, error)

	///////////////////////////////////////////////////////////////////////////////
	// LabelStore

	// ApplyLabelSpecs applies a list of LabelSpecs to the datastore, creating and updating labels as necessary.
	ApplyLabelSpecs(ctx context.Context, specs []*LabelSpec) error
	// GetLabelSpecs returns all of the stored LabelSpecs.
	GetLabelSpecs(ctx context.Context) ([]*LabelSpec, error)
	// GetLabelSpec returns the spec for the named label.
	GetLabelSpec(ctx context.Context, name string) (*LabelSpec, error)

	NewLabel(ctx context.Context, Label *Label, opts ...OptionalArg) (*Label, error)
	SaveLabel(ctx context.Context, label *Label) (*Label, error)
	DeleteLabel(ctx context.Context, name string) error
	Label(ctx context.Context, lid uint) (*Label, error)
	ListLabels(ctx context.Context, filter TeamFilter, opt ListOptions) ([]*Label, error)
	LabelsSummary(ctx context.Context) ([]*LabelSummary, error)

	// LabelQueriesForHost returns the label queries that should be executed for the given host.
	// Results are returned in a map of label id -> query
	LabelQueriesForHost(ctx context.Context, host *Host) (map[string]string, error)

	// ListLabelsForHost returns the labels that the given host is in.
	ListLabelsForHost(ctx context.Context, hid uint) ([]*Label, error)

	// ListHostsInLabel returns a slice of hosts in the label with the given ID.
	ListHostsInLabel(ctx context.Context, filter TeamFilter, lid uint, opt HostListOptions) ([]*Host, error)

	// ListUniqueHostsInLabels returns a slice of all of the hosts in the given label IDs. A host will only appear once
	// in the results even if it is in multiple of the provided labels.
	ListUniqueHostsInLabels(ctx context.Context, filter TeamFilter, labels []uint) ([]*Host, error)

	SearchLabels(ctx context.Context, filter TeamFilter, query string, omit ...uint) ([]*Label, error)

	// LabelIDsByName Retrieve the IDs associated with the given labels
	LabelIDsByName(ctx context.Context, labels []string) ([]uint, error)

	// Methods used for async processing of host label query results.
	AsyncBatchInsertLabelMembership(ctx context.Context, batch [][2]uint) error
	AsyncBatchDeleteLabelMembership(ctx context.Context, batch [][2]uint) error
	AsyncBatchUpdateLabelTimestamp(ctx context.Context, ids []uint, ts time.Time) error

	///////////////////////////////////////////////////////////////////////////////
	// HostStore

	// NewHost is deprecated and will be removed. Hosts should always be enrolled via EnrollHost.
	NewHost(ctx context.Context, host *Host) (*Host, error)
	DeleteHost(ctx context.Context, hid uint) error
	Host(ctx context.Context, id uint) (*Host, error)
	ListHosts(ctx context.Context, filter TeamFilter, opt HostListOptions) ([]*Host, error)
	ListHostsLiteByUUIDs(ctx context.Context, filter TeamFilter, uuids []string) ([]*Host, error)

	MarkHostsSeen(ctx context.Context, hostIDs []uint, t time.Time) error
	SearchHosts(ctx context.Context, filter TeamFilter, query string, omit ...uint) ([]*Host, error)
	// EnrolledHostIDs returns the full list of enrolled host IDs.
	EnrolledHostIDs(ctx context.Context) ([]uint, error)
	CountEnrolledHosts(ctx context.Context) (int, error)

	// TODO(sarah): Reconcile pending mdm hosts feature with original motivation to cleanup "dead incoming host"

	// CleanupIncomingHosts deletes hosts that have enrolled but never updated their status details. This clears dead
	// "incoming hosts" that never complete their registration.
	// A host is considered incoming if each of the hostname and osquery_version and hardware_serial
	// fields are empty. This means that multiple different osquery queries failed to populate details.
	CleanupIncomingHosts(ctx context.Context, now time.Time) ([]uint, error)
	// GenerateHostStatusStatistics retrieves the count of online, offline, MIA and new hosts.
	GenerateHostStatusStatistics(ctx context.Context, filter TeamFilter, now time.Time, platform *string, lowDiskSpace *int) (*HostSummary, error)
	// HostIDsByName Retrieve the IDs associated with the given hostnames
	HostIDsByName(ctx context.Context, filter TeamFilter, hostnames []string) ([]uint, error)

	// HostIDsByOSID retrieves the IDs of all host for the given OS ID
	HostIDsByOSID(ctx context.Context, osID uint, offset int, limit int) ([]uint, error)

	// TODO JUAN: Refactor this to use the Operating System type instead.
	// HostIDsByOSVersion retrieves the IDs of all host matching osVersion
	HostIDsByOSVersion(ctx context.Context, osVersion OSVersion, offset int, limit int) ([]uint, error)
	// HostByIdentifier returns one host matching the provided identifier. Possible matches can be on
	// osquery_host_identifier, node_key, UUID, or hostname.
	HostByIdentifier(ctx context.Context, identifier string) (*Host, error)
	// AddHostsToTeam adds hosts to an existing team, clearing their team settings if teamID is nil.
	AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint) error

	TotalAndUnseenHostsSince(ctx context.Context, daysCount int) (total int, unseen int, err error)

	// DeleteHosts deletes associated tables for multiple hosts.
	//
	// It atomically deletes each host but if it returns an error, some of the hosts may be
	// deleted and others not.
	DeleteHosts(ctx context.Context, ids []uint) error

	CountHosts(ctx context.Context, filter TeamFilter, opt HostListOptions) (int, error)
	CountHostsInLabel(ctx context.Context, filter TeamFilter, lid uint, opt HostListOptions) (int, error)
	ListHostDeviceMapping(ctx context.Context, id uint) ([]*HostDeviceMapping, error)
	// ListHostBatteries returns the list of batteries for the given host ID.
	ListHostBatteries(ctx context.Context, id uint) ([]*HostBattery, error)

	// LoadHostByDeviceAuthToken loads the host identified by the device auth token.
	// If the token is invalid or expired it returns a NotFoundError.
	LoadHostByDeviceAuthToken(ctx context.Context, authToken string, tokenTTL time.Duration) (*Host, error)
	// SetOrUpdateDeviceAuthToken inserts or updates the auth token for a host.
	SetOrUpdateDeviceAuthToken(ctx context.Context, hostID uint, authToken string) error

	// FailingPoliciesCount returns the number of failling policies for 'host'
	FailingPoliciesCount(ctx context.Context, host *Host) (uint, error)

	// ListPoliciesForHost lists the policies that a host will check and whether they are passing
	ListPoliciesForHost(ctx context.Context, host *Host) ([]*HostPolicy, error)

	GetHostMunkiVersion(ctx context.Context, hostID uint) (string, error)
	GetHostMunkiIssues(ctx context.Context, hostID uint) ([]*HostMunkiIssue, error)
	GetHostMDM(ctx context.Context, hostID uint) (*HostMDM, error)
	GetHostMDMCheckinInfo(ctx context.Context, hostUUID string) (*HostMDMCheckinInfo, error)

	AggregatedMunkiVersion(ctx context.Context, teamID *uint) ([]AggregatedMunkiVersion, time.Time, error)
	AggregatedMunkiIssues(ctx context.Context, teamID *uint) ([]AggregatedMunkiIssue, time.Time, error)
	AggregatedMDMStatus(ctx context.Context, teamID *uint, platform string) (AggregatedMDMStatus, time.Time, error)
	AggregatedMDMSolutions(ctx context.Context, teamID *uint, platform string) ([]AggregatedMDMSolutions, time.Time, error)
	GenerateAggregatedMunkiAndMDM(ctx context.Context) error

	GetMunkiIssue(ctx context.Context, munkiIssueID uint) (*MunkiIssue, error)
	GetMDMSolution(ctx context.Context, mdmID uint) (*MDMSolution, error)

	OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string) (*OSVersions, error)
	UpdateOSVersions(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// TargetStore

	// CountHostsInTargets returns the metrics of the hosts in the provided labels, teams, and explicit host IDs.
	CountHostsInTargets(ctx context.Context, filter TeamFilter, targets HostTargets, now time.Time) (TargetMetrics, error)
	// HostIDsInTargets returns the host IDs of the hosts in the provided labels, teams, and explicit host IDs. The
	// returned host IDs should be sorted in ascending order.
	HostIDsInTargets(ctx context.Context, filter TeamFilter, targets HostTargets) ([]uint, error)

	///////////////////////////////////////////////////////////////////////////////
	// PasswordResetStore manages password resets in the Datastore

	NewPasswordResetRequest(ctx context.Context, req *PasswordResetRequest) (*PasswordResetRequest, error)
	DeletePasswordResetRequestsForUser(ctx context.Context, userID uint) error
	FindPasswordResetByToken(ctx context.Context, token string) (*PasswordResetRequest, error)
	// CleanupExpiredPasswordResetRequests deletes any password reset requests that have expired.
	CleanupExpiredPasswordResetRequests(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// SessionStore is the abstract interface that all session backends must conform to.

	// SessionByKey returns, given a session key, a session object or an error if one could not be found for the given
	// key
	SessionByKey(ctx context.Context, key string) (*Session, error)

	// SessionByID returns, given a session id, find and return a session object or an error if one could not be found
	// for the given id
	SessionByID(ctx context.Context, id uint) (*Session, error)

	// ListSessionsForUser finds all the active sessions for a given user
	ListSessionsForUser(ctx context.Context, id uint) ([]*Session, error)

	// NewSession stores a new session struct
	NewSession(ctx context.Context, userID uint, sessionKey string) (*Session, error)

	// DestroySession destroys the currently tracked session
	DestroySession(ctx context.Context, session *Session) error

	// DestroyAllSessionsForUser destroys all of the sessions for a given user
	DestroyAllSessionsForUser(ctx context.Context, id uint) error

	// MarkSessionAccessed marks the currently tracked session as access to extend expiration
	MarkSessionAccessed(ctx context.Context, session *Session) error

	///////////////////////////////////////////////////////////////////////////////
	// AppConfigStore contains method for saving and retrieving application configuration

	NewAppConfig(ctx context.Context, info *AppConfig) (*AppConfig, error)
	AppConfig(ctx context.Context) (*AppConfig, error)
	SaveAppConfig(ctx context.Context, info *AppConfig) error

	// GetEnrollSecrets gets the enroll secrets for a team (or global if teamID is nil).
	GetEnrollSecrets(ctx context.Context, teamID *uint) ([]*EnrollSecret, error)
	// ApplyEnrollSecrets replaces the current enroll secrets for a team with the provided secrets.
	ApplyEnrollSecrets(ctx context.Context, teamID *uint, secrets []*EnrollSecret) error

	// AggregateEnrollSecretPerTeam returns a slice containing one
	// EnrollSecret per team, plus an EnrollSecret for "no team"
	//
	// If any of the teams doesn't have any enroll secrets, then the
	// corresponcing EnrollSecret.Secret entry will have an empty string
	// value.
	AggregateEnrollSecretPerTeam(ctx context.Context) ([]*EnrollSecret, error)

	///////////////////////////////////////////////////////////////////////////////
	// InviteStore contains the methods for managing user invites in a datastore.

	// NewInvite creates and stores a new invitation in a DB.
	NewInvite(ctx context.Context, i *Invite) (*Invite, error)

	// ListInvites lists all invites in the datastore.
	ListInvites(ctx context.Context, opt ListOptions) ([]*Invite, error)

	// Invite retrieves an invite by its ID.
	Invite(ctx context.Context, id uint) (*Invite, error)

	// InviteByEmail retrieves an invite for a specific email address.
	InviteByEmail(ctx context.Context, email string) (*Invite, error)

	// InviteByToken retrieves and invite using the token string.
	InviteByToken(ctx context.Context, token string) (*Invite, error)

	// DeleteInvite deletes an invitation.
	DeleteInvite(ctx context.Context, id uint) error

	UpdateInvite(ctx context.Context, id uint, i *Invite) (*Invite, error)

	///////////////////////////////////////////////////////////////////////////////
	// ScheduledQueryStore

	// ListScheduledQueriesInPackWithStats loads a pack's scheduled queries and its aggregated stats.
	ListScheduledQueriesInPackWithStats(ctx context.Context, id uint, opts ListOptions) ([]*ScheduledQuery, error)
	NewScheduledQuery(ctx context.Context, sq *ScheduledQuery, opts ...OptionalArg) (*ScheduledQuery, error)
	SaveScheduledQuery(ctx context.Context, sq *ScheduledQuery) (*ScheduledQuery, error)
	DeleteScheduledQuery(ctx context.Context, id uint) error
	ScheduledQuery(ctx context.Context, id uint) (*ScheduledQuery, error)
	CleanupExpiredHosts(ctx context.Context) ([]uint, error)
	// ScheduledQueryIDsByName loads the IDs associated with the given pack and
	// query names. It returns a slice of IDs in the same order as
	// packAndSchedQueryNames, with the ID set to 0 if the corresponding
	// scheduled query did not exist.
	ScheduledQueryIDsByName(ctx context.Context, batchSize int, packAndSchedQueryNames ...[2]string) ([]uint, error)

	///////////////////////////////////////////////////////////////////////////////
	// TeamStore

	// NewTeam creates a new Team object in the store.
	NewTeam(ctx context.Context, team *Team) (*Team, error)
	// SaveTeam saves any changes to the team.
	SaveTeam(ctx context.Context, team *Team) (*Team, error)
	// Team retrieves the Team by ID.
	Team(ctx context.Context, tid uint) (*Team, error)
	// Team deletes the Team by ID.
	DeleteTeam(ctx context.Context, tid uint) error
	// TeamByName retrieves the Team by Name.
	TeamByName(ctx context.Context, name string) (*Team, error)
	// ListTeams lists teams with the ordering and filters in the provided options.
	ListTeams(ctx context.Context, filter TeamFilter, opt ListOptions) ([]*Team, error)
	// TeamsSummary lists id, name and description for all teams.
	TeamsSummary(ctx context.Context) ([]*TeamSummary, error)
	// SearchTeams searches teams using the provided query and ommitting the provided existing selection.
	SearchTeams(ctx context.Context, filter TeamFilter, matchQuery string, omit ...uint) ([]*Team, error)
	// TeamEnrollSecrets lists the enroll secrets for the team.
	TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*EnrollSecret, error)
	// DeleteIntegrationsFromTeams deletes integrations used by teams, as they
	// are being deleted from the global configuration.
	DeleteIntegrationsFromTeams(ctx context.Context, deletedIntgs Integrations) error

	///////////////////////////////////////////////////////////////////////////////
	// SoftwareStore

	// ListSoftwareForVulnDetection returns all software for the given hostID with only the fields
	// used for vulnerability detection populated (id, name, version, cpe_id, cpe)
	ListSoftwareForVulnDetection(ctx context.Context, hostID uint) ([]Software, error)
	ListSoftwareVulnerabilitiesByHostIDsSource(ctx context.Context, hostIDs []uint, source VulnerabilitySource) (map[uint][]SoftwareVulnerability, error)
	LoadHostSoftware(ctx context.Context, host *Host, includeCVEScores bool) error

	AllSoftwareIterator(ctx context.Context, query SoftwareIterQueryOptions) (SoftwareIterator, error)
	// UpsertSoftwareCPEs either inserts new 'software_cpe' entries, or if a now with the same CPE
	// already exists, performs an update operation. Returns the number of rows affected.
	UpsertSoftwareCPEs(ctx context.Context, cpes []SoftwareCPE) (int64, error)
	// DeleteSoftwareCPEs removes entries from 'software_cpe' by matching the software_id in the
	// provided cpes. Returns the number of rows affected.
	DeleteSoftwareCPEs(ctx context.Context, cpes []SoftwareCPE) (int64, error)
	ListSoftwareCPEs(ctx context.Context) ([]SoftwareCPE, error)
	// InsertSoftwareVulnerability will either insert a new vulnerability in the datastore (in which
	// case it will return true) or if a matching record already exists it will update its
	// updated_at timestamp (in which case it will return false).
	InsertSoftwareVulnerability(ctx context.Context, vuln SoftwareVulnerability, source VulnerabilitySource) (bool, error)
	SoftwareByID(ctx context.Context, id uint, includeCVEScores bool) (*Software, error)
	// ListSoftwareByHostIDShort lists software by host ID, but does not include CPEs or vulnerabilites.
	// It is meant to be used when only minimal software fields are required eg when updating host software.
	ListSoftwareByHostIDShort(ctx context.Context, hostID uint) ([]Software, error)
	// SyncHostsSoftware calculates the number of hosts having each
	// software installed and stores that information in the software_host_counts
	// table.
	//
	// After aggregation, it cleans up unused software (e.g. software installed
	// on removed hosts, software uninstalled on hosts, etc.)
	SyncHostsSoftware(ctx context.Context, updatedAt time.Time) error
	HostsBySoftwareIDs(ctx context.Context, softwareIDs []uint) ([]*HostShort, error)
	HostsByCVE(ctx context.Context, cve string) ([]*HostShort, error)
	InsertCVEMeta(ctx context.Context, cveMeta []CVEMeta) error
	ListCVEs(ctx context.Context, maxAge time.Duration) ([]CVEMeta, error)

	///////////////////////////////////////////////////////////////////////////////
	// OperatingSystemsStore

	// ListOperationsSystems returns all operating systems (id, name, version)
	ListOperatingSystems(ctx context.Context) ([]OperatingSystem, error)
	// UpdateHostOperatingSystem updates the `host_operating_system` table
	// for the given host ID with the ID of the operating system associated
	// with the given name, version, arch, and kernel version in the
	// `operating_systems` table.
	//
	// If the `operating_systems` table does not already include a record
	// associated with the given name, version, arch, and kernel version,
	// a new record is also created.
	UpdateHostOperatingSystem(ctx context.Context, hostID uint, hostOS OperatingSystem) error
	// CleanupHostOperatingSystems removes records from the host_operating_system table that are
	// associated with any non-existent host (e.g., expired hosts) and removes records from the
	// operating_systems table that no longer associated with any host (e.g., all hosts have
	// upgraded from a prior version).
	CleanupHostOperatingSystems(ctx context.Context) error

	UpdateHostTablesOnMDMUnenroll(ctx context.Context, uuid string) error

	///////////////////////////////////////////////////////////////////////////////
	// ActivitiesStore

	NewActivity(ctx context.Context, user *User, activity ActivityDetails) error
	ListActivities(ctx context.Context, opt ListActivitiesOptions) ([]*Activity, *PaginationMetadata, error)
	MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error

	///////////////////////////////////////////////////////////////////////////////
	// StatisticsStore

	ShouldSendStatistics(ctx context.Context, frequency time.Duration, config config.FleetConfig) (StatisticsPayload, bool, error)
	RecordStatisticsSent(ctx context.Context) error
	// CleanupStatistics executes cleanup tasks to be performed upon successful transmission of
	// statistics.
	CleanupStatistics(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// GlobalPoliciesStore

	// ApplyPolicySpecs applies a list of policies (likely from a yaml file) to the datastore. Existing policies are updated,
	// and new policies are created.
	ApplyPolicySpecs(ctx context.Context, authorID uint, specs []*PolicySpec) error

	NewGlobalPolicy(ctx context.Context, authorID *uint, args PolicyPayload) (*Policy, error)
	Policy(ctx context.Context, id uint) (*Policy, error)
	// SavePolicy updates some fields of the given policy on the datastore.
	//
	// It is also used to update team policies.
	SavePolicy(ctx context.Context, p *Policy) error

	ListGlobalPolicies(ctx context.Context) ([]*Policy, error)
	PoliciesByID(ctx context.Context, ids []uint) (map[uint]*Policy, error)
	DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error)

	PolicyQueriesForHost(ctx context.Context, host *Host) (map[string]string, error)

	// Methods used for async processing of host policy query results.
	AsyncBatchInsertPolicyMembership(ctx context.Context, batch []PolicyMembershipResult) error
	AsyncBatchUpdatePolicyTimestamp(ctx context.Context, ids []uint, ts time.Time) error

	// MigrateTables creates and migrates the table schemas
	MigrateTables(ctx context.Context) error
	// MigrateData populates built-in data
	MigrateData(ctx context.Context) error
	// MigrationStatus returns nil if migrations are complete, and an error if migrations need to be run.
	MigrationStatus(ctx context.Context) (*MigrationStatus, error)

	ListSoftware(ctx context.Context, opt SoftwareListOptions) ([]Software, error)
	CountSoftware(ctx context.Context, opt SoftwareListOptions) (int, error)
	// DeleteVulnerabilities deletes the given list of vulnerabilities identified by CPE+CVE.
	DeleteSoftwareVulnerabilities(ctx context.Context, vulnerabilities []SoftwareVulnerability) error
	// DeleteOutOfDateVulnerabilities deletes 'software_cve' entries from the provided source where
	// the updated_at timestamp is older than the provided duration
	DeleteOutOfDateVulnerabilities(ctx context.Context, source VulnerabilitySource, duration time.Duration) error

	///////////////////////////////////////////////////////////////////////////////
	// Team Policies

	NewTeamPolicy(ctx context.Context, teamID uint, authorID *uint, args PolicyPayload) (*Policy, error)
	ListTeamPolicies(ctx context.Context, teamID uint) (teamPolicies, inheritedPolicies []*Policy, err error)
	DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error)
	TeamPolicy(ctx context.Context, teamID uint, policyID uint) (*Policy, error)

	CleanupPolicyMembership(ctx context.Context, now time.Time) error
	// IncrementPolicyViolationDays increments the aggregate count of policy violation days. One
	// policy violation day is added for each policy that a host is failing as of the time the count
	// is incremented. The count only increments once per 24-hour interval. If the interval has not
	// elapsed, IncrementPolicyViolationDays returns nil without incrementing the count.
	IncrementPolicyViolationDays(ctx context.Context) error
	// InitializePolicyViolationDays sets the aggregated count of policy violation days to zero. If
	// a record of the count already exists, its `created_at` timestamp is updated to the current timestamp.
	InitializePolicyViolationDays(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// Locking

	// Lock tries to get an atomic lock on an instance named with `name`
	// and an `owner` identified by a random string per instance.
	// Subsequently locking the same resource name for the same owner
	// renews the lock expiration.
	// It returns true, nil if it managed to obtain a lock on the instance.
	// false and potentially an error otherwise.
	// This must not be blocking.
	Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error)
	// Unlock tries to unlock the lock by that `name` for the specified
	// `owner`. Unlocking when not holding the lock shouldn't error
	Unlock(ctx context.Context, name string, owner string) error
	// DBLocks returns the current database transaction lock waits information.
	DBLocks(ctx context.Context) ([]*DBLock, error)

	///////////////////////////////////////////////////////////////////////////////
	// Cron Stats

	// GetLatestCronStats returns a slice of no more than two cron stats records, where index 0 (if
	// present) is the most recently created scheduled run, and index 1 (if present) represents a
	// triggered run that is currently pending.
	GetLatestCronStats(ctx context.Context, name string) ([]CronStats, error)
	// InsertCronStats inserts cron stats for the named cron schedule.
	InsertCronStats(ctx context.Context, statsType CronStatsType, name string, instance string, status CronStatsStatus) (int, error)
	// UpdateCronStats updates the status of the identified cron stats record.
	UpdateCronStats(ctx context.Context, id int, status CronStatsStatus) error
	// UpdateAllCronStatsForInstance updates all records for the identified instance with the
	// specified statuses
	UpdateAllCronStatsForInstance(ctx context.Context, instance string, fromStatus CronStatsStatus, toStatus CronStatsStatus) error
	// CleanupCronStats cleans up expired cron stats.
	CleanupCronStats(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// Aggregated Stats

	UpdateScheduledQueryAggregatedStats(ctx context.Context) error
	UpdateQueryAggregatedStats(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// Following are the set of APIs used by osquery hosts:
	// TODO(lucas): Move them to a separate datastore or abstraction, to avoid
	// future developers trying to use user-datastore APIs on the host requests.
	// The user-datastore APIs (all the above) are generally more expensive and
	// load data that is not necessary for osquery hosts.
	///////////////////////////////////////////////////////////////////////////////

	// LoadHostByNodeKey loads the whole host identified by the node key.
	// If the node key is invalid it returns a NotFoundError.
	LoadHostByNodeKey(ctx context.Context, nodeKey string) (*Host, error)

	// LoadHostByOrbitNodeKey loads the whole host identified by the node key.
	// If the node key is invalid it returns a NotFoundError.
	LoadHostByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error)

	// HostLite will load the primary data of the host with the given id.
	// We define "primary data" as all host information except the
	// details (like cpu, memory, gigs_disk_space_available, etc.).
	//
	// If the host doesn't exist, a NotFoundError is returned.
	HostLite(ctx context.Context, hostID uint) (*Host, error)

	// UpdateHostOsqueryIntervals updates the osquery intervals of a host.
	UpdateHostOsqueryIntervals(ctx context.Context, hostID uint, intervals HostOsqueryIntervals) error

	// TeamAgentOptions loads the agents options of a team.
	TeamAgentOptions(ctx context.Context, teamID uint) (*json.RawMessage, error)

	// TeamFeatures loads the features enabled for a team.
	TeamFeatures(ctx context.Context, teamID uint) (*Features, error)

	// TeamMDMConfig loads the MDM config for a team.
	TeamMDMConfig(ctx context.Context, teamID uint) (*TeamMDM, error)

	// SaveHostPackStats stores (and updates) the pack's scheduled queries stats of a host.
	SaveHostPackStats(ctx context.Context, hostID uint, stats []PackStats) error
	// AsyncBatchSaveHostsScheduledQueryStats efficiently saves a batch of hosts'
	// pack stats of scheduled queries. It is the async and batch version of
	// SaveHostPackStats. It returns the number of INSERT-ON DUPLICATE UPDATE
	// statements that were executed (for reporting purpose) or an error.
	AsyncBatchSaveHostsScheduledQueryStats(ctx context.Context, stats map[uint][]ScheduledQueryStats, batchSize int) (int, error)

	// UpdateHostSoftware updates the software list of a host.
	// The update consists of deleting existing entries that are not in the given `software`
	// slice, updating existing entries and inserting new entries.
	UpdateHostSoftware(ctx context.Context, hostID uint, software []Software) error

	// UpdateHost updates a host.
	UpdateHost(ctx context.Context, host *Host) error

	// ListScheduledQueriesInPack lists all the scheduled queries of a pack.
	ListScheduledQueriesInPack(ctx context.Context, packID uint) (ScheduledQueryList, error)

	// UpdateHostRefetchRequested updates a host's refetch requested field.
	UpdateHostRefetchRequested(ctx context.Context, hostID uint, value bool) error

	// FlippingPoliciesForHost fetches the policies with incoming results and returns:
	//	- a list of "new" failing policies; "new" here means those that fail on their first
	//	run, and those that were passing on the previous run and are failing on the incoming execution.
	//	- a list of "new" passing policies; "new" here means those that failed on a previous
	//	run and are passing now.
	//
	// "Failure" here means the policy query executed successfully but didn't return any rows,
	// so policies that did not execute (incomingResults with nil bool) are ignored.
	FlippingPoliciesForHost(ctx context.Context, hostID uint, incomingResults map[uint]*bool) (newFailing []uint, newPassing []uint, err error)

	// RecordPolicyQueryExecutions records the execution results of the policies for the given host.
	RecordPolicyQueryExecutions(ctx context.Context, host *Host, results map[uint]*bool, updated time.Time, deferredSaveHost bool) error

	// RecordLabelQueryExecutions saves the results of label queries. The results map is a map of label id -> whether or
	// not the label matches. The time parameter is the timestamp to save with the query execution.
	RecordLabelQueryExecutions(ctx context.Context, host *Host, results map[uint]*bool, t time.Time, deferredSaveHost bool) error

	// SaveHostUsers updates the user list of a host.
	// The update consists of deleting existing entries that are not in the given `users`
	// slice, updating existing entries and inserting new entries.
	SaveHostUsers(ctx context.Context, hostID uint, users []HostUser) error

	// SaveHostAdditional updates the additional queries results of a host.
	SaveHostAdditional(ctx context.Context, hostID uint, additional *json.RawMessage) error

	SetOrUpdateMunkiInfo(ctx context.Context, hostID uint, version string, errors, warnings []string) error
	SetOrUpdateMDMData(ctx context.Context, hostID uint, isServer, enrolled bool, serverURL string, installedFromDep bool, name string) error
	SetOrUpdateHostDisksSpace(ctx context.Context, hostID uint, gigsAvailable, percentAvailable float64) error
	SetOrUpdateHostDisksEncryption(ctx context.Context, hostID uint, encrypted bool) error
	// SetOrUpdateHostDiskEncryptionKey sets the base64, encrypted key for
	// a host
	SetOrUpdateHostDiskEncryptionKey(ctx context.Context, hostID uint, encryptedBase64Key string) error
	// GetUnverifiedDiskEncryptionKeys returns all the encryption keys that
	// are collected but their decryptable status is not known yet (ie:
	// we're able to decrypt the key using a private key in the server)
	GetUnverifiedDiskEncryptionKeys(ctx context.Context) ([]HostDiskEncryptionKey, error)
	// SetHostDiskEncryptionKeyStatus sets the encryptable status for the set
	// of encription keys provided
	SetHostsDiskEncryptionKeyStatus(ctx context.Context, hostIDs []uint, encryptable bool, threshold time.Time) error
	// GetHostDiskEncryptionKey returns the encryption key information for a given host
	GetHostDiskEncryptionKey(ctx context.Context, hostID uint) (*HostDiskEncryptionKey, error)

	SetDiskEncryptionResetStatus(ctx context.Context, hostID uint, status bool) error
	// SetOrUpdateHostOrbitInfo inserts of updates the orbit info for a host
	SetOrUpdateHostOrbitInfo(ctx context.Context, hostID uint, version string) error

	ReplaceHostDeviceMapping(ctx context.Context, id uint, mappings []*HostDeviceMapping) error

	// ReplaceHostBatteries creates or updates the battery mappings of a host.
	ReplaceHostBatteries(ctx context.Context, id uint, mappings []*HostBattery) error

	// VerifyEnrollSecret checks that the provided secret matches an active enroll secret. If it is successfully
	// matched, that secret is returned. Otherwise, an error is returned.
	VerifyEnrollSecret(ctx context.Context, secret string) (*EnrollSecret, error)

	// EnrollHost will enroll a new host with the given identifier, setting the node key, and team. Implementations of
	// this method should respect the provided host enrollment cooldown, by returning an error if the host has enrolled
	// within the cooldown period.
	EnrollHost(ctx context.Context, isMDMEnabled bool, osqueryHostId, hardwareUUID, hardwareSerial, nodeKey string, teamID *uint, cooldown time.Duration) (*Host, error)

	// EnrollOrbit will enroll a new orbit instance.
	//	- If an entry for the host exists (osquery enrolled first) then it will update the host's orbit node key and team.
	//	- If an entry for the host doesn't exist (osquery enrolls later) then it will create a new entry in the hosts table.
	EnrollOrbit(ctx context.Context, isMDMEnabled bool, hostInfo OrbitHostInfo, orbitNodeKey string, teamID *uint) (*Host, error)

	SerialUpdateHost(ctx context.Context, host *Host) error

	///////////////////////////////////////////////////////////////////////////////
	// JobStore

	// NewJob inserts a new job into the jobs table (queue).
	NewJob(ctx context.Context, job *Job) (*Job, error)

	// GetQueuedJobs gets queued jobs from the jobs table (queue).
	GetQueuedJobs(ctx context.Context, maxNumJobs int) ([]*Job, error)

	// UpdateJobs updates an existing job. Call this after processing a job.
	UpdateJob(ctx context.Context, id uint, job *Job) (*Job, error)

	///////////////////////////////////////////////////////////////////////////////
	// Debug

	InnoDBStatus(ctx context.Context) (string, error)
	ProcessList(ctx context.Context) ([]MySQLProcess, error)

	// WindowsUpdates Store
	ListWindowsUpdatesByHostID(ctx context.Context, hostID uint) ([]WindowsUpdate, error)
	InsertWindowsUpdates(ctx context.Context, hostID uint, updates []WindowsUpdate) error

	///////////////////////////////////////////////////////////////////////////////
	// OperatingSystemVulnerabilities Store
	ListOSVulnerabilities(ctx context.Context, hostID []uint) ([]OSVulnerability, error)
	InsertOSVulnerabilities(ctx context.Context, vulnerabilities []OSVulnerability, source VulnerabilitySource) (int64, error)
	DeleteOSVulnerabilities(ctx context.Context, vulnerabilities []OSVulnerability) error

	///////////////////////////////////////////////////////////////////////////////
	// Apple MDM

	// NewMDMAppleConfigProfile creates and returns a new configuration profile.
	NewMDMAppleConfigProfile(ctx context.Context, p MDMAppleConfigProfile) (*MDMAppleConfigProfile, error)

	// BulkUpsertMDMAppleConfigProfiles inserts or updates a configuration
	// profiles in bulk with the current payload.
	//
	// Be careful when using this for user actions, you generally want to
	// use NewMDMAppleConfigProfile/DeleteMDMAppleConfigProfile or the
	// batch insert/delete counterparts. With the current product vision,
	// this is mainly aimed to internal usage within the Fleet server.
	BulkUpsertMDMAppleConfigProfiles(ctx context.Context, payload []*MDMAppleConfigProfile) error

	// GetMDMAppleConfigProfile returns the mdm config profile corresponding to the specified
	// profile id.
	GetMDMAppleConfigProfile(ctx context.Context, profileID uint) (*MDMAppleConfigProfile, error)

	// ListMDMAppleConfigProfiles lists mdm config profiles associated with the specified team id.
	// For global config profiles, specify nil as the team id.
	ListMDMAppleConfigProfiles(ctx context.Context, teamID *uint) ([]*MDMAppleConfigProfile, error)

	// DeleteMDMAppleConfigProfile deletes the mdm config profile corresponding
	// to the specified profile id.
	DeleteMDMAppleConfigProfile(ctx context.Context, profileID uint) error

	// DeleteMDMAppleConfigProfileByTeamAndIdentifier deletes a configuration
	// profile using the unique key defined by `team_id` and `identifier`
	DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx context.Context, teamID *uint, profileIdentifier string) error

	// GetHostMDMProfiles returns the MDM profile information for the specified host UUID.
	GetHostMDMProfiles(ctx context.Context, hostUUID string) ([]HostMDMAppleProfile, error)

	CleanupDiskEncryptionKeysOnTeamChange(ctx context.Context, hostIDs []uint, newTeamID *uint) error

	// NewMDMAppleEnrollmentProfile creates and returns new enrollment profile.
	// Such enrollment profiles allow devices to enroll to Fleet MDM.
	NewMDMAppleEnrollmentProfile(ctx context.Context, enrollmentPayload MDMAppleEnrollmentProfilePayload) (*MDMAppleEnrollmentProfile, error)

	// GetMDMAppleEnrollmentProfileByToken loads the enrollment profile from its secret token.
	GetMDMAppleEnrollmentProfileByToken(ctx context.Context, token string) (*MDMAppleEnrollmentProfile, error)

	// GetMDMAppleEnrollmentProfileByType loads the enrollment profile from its type (e.g. manual, automatic).
	GetMDMAppleEnrollmentProfileByType(ctx context.Context, typ MDMAppleEnrollmentType) (*MDMAppleEnrollmentProfile, error)

	// ListMDMAppleEnrollmentProfiles returns the list of all the enrollment profiles.
	ListMDMAppleEnrollmentProfiles(ctx context.Context) ([]*MDMAppleEnrollmentProfile, error)

	// GetMDMAppleCommandResults returns the execution results of a command identified by a CommandUUID.
	GetMDMAppleCommandResults(ctx context.Context, commandUUID string) ([]*MDMAppleCommandResult, error)

	// ListMDMAppleCommands returns a list of MDM Apple commands that have been
	// executed, based on the provided options.
	ListMDMAppleCommands(ctx context.Context, tmFilter TeamFilter, listOpts *MDMAppleCommandListOptions) ([]*MDMAppleCommand, error)

	// NewMDMAppleInstaller creates and stores an Apple installer to Fleet.
	NewMDMAppleInstaller(ctx context.Context, name string, size int64, manifest string, installer []byte, urlToken string) (*MDMAppleInstaller, error)

	// MDMAppleInstaller returns the installer with its contents included (MDMAppleInstaller.Installer) from its token.
	MDMAppleInstaller(ctx context.Context, token string) (*MDMAppleInstaller, error)

	// MDMAppleInstallerDetailsByID returns the installer details of an installer, all fields except its content,
	// (MDMAppleInstaller.Installer is nil).
	MDMAppleInstallerDetailsByID(ctx context.Context, id uint) (*MDMAppleInstaller, error)

	// DeleteMDMAppleInstaller deletes an installer.
	DeleteMDMAppleInstaller(ctx context.Context, id uint) error

	// MDMAppleInstallerDetailsByToken loads the installer details, all fields except its content,
	// (MDMAppleInstaller.Installer is nil) from its secret token.
	MDMAppleInstallerDetailsByToken(ctx context.Context, token string) (*MDMAppleInstaller, error)

	// ListMDMAppleInstallers list all the uploaded installers.
	ListMDMAppleInstallers(ctx context.Context) ([]MDMAppleInstaller, error)

	// BatchSetMDMAppleProfiles sets the MDM Apple profiles for the given team or
	// no team.
	BatchSetMDMAppleProfiles(ctx context.Context, tmID *uint, profiles []*MDMAppleConfigProfile) error

	// MDMAppleListDevices lists all the MDM enrolled devices.
	MDMAppleListDevices(ctx context.Context) ([]MDMAppleDevice, error)

	// IngestMDMAppleDevicesFromDEPSync creates new Fleet host records for MDM-enrolled devices that are
	// not already enrolled in Fleet. It returns the number of hosts created, the team id that they
	// joined (nil for no team), and an error.
	IngestMDMAppleDevicesFromDEPSync(ctx context.Context, devices []godep.Device) (int64, *uint, error)

	// IngestMDMAppleDeviceFromCheckin creates a new Fleet host record for an MDM-enrolled device that is
	// not already enrolled in Fleet.
	IngestMDMAppleDeviceFromCheckin(ctx context.Context, mdmHost MDMAppleHostDetails) error

	// ListMDMAppleDEPSerialsInTeam returns a list of serial numbers of hosts
	// that are enrolled or pending enrollment in Fleet's MDM via DEP for the
	// specified team (or no team if teamID is nil).
	ListMDMAppleDEPSerialsInTeam(ctx context.Context, teamID *uint) ([]string, error)

	// ListMDMAppleDEPSerialsInHostIDs returns a list of serial numbers of hosts
	// that are enrolled or pending enrollment in Fleet's MDM via DEP in the
	// specified list of host IDs.
	ListMDMAppleDEPSerialsInHostIDs(ctx context.Context, hostIDs []uint) ([]string, error)

	// GetNanoMDMEnrollment returns the nano enrollment information for the device id.
	GetNanoMDMEnrollment(ctx context.Context, id string) (*NanoEnrollment, error)

	// IncreasePolicyAutomationIteration marks the policy to fire automation again.
	IncreasePolicyAutomationIteration(ctx context.Context, policyID uint) error

	// OutdatedAutomationBatch returns a batch of hosts that had a failing policy.
	OutdatedAutomationBatch(ctx context.Context) ([]PolicyFailure, error)

	// ListMDMAppleProfilesToInstall returns all the profiles that should
	// be installed based on diffing the ideal state vs the state we have
	// registered in `host_mdm_apple_profiles`
	ListMDMAppleProfilesToInstall(ctx context.Context) ([]*MDMAppleProfilePayload, error)

	// ListMDMAppleProfilesToRemove returns all the profiles that should
	// be removed based on diffing the ideal state vs the state we have
	// registered in `host_mdm_apple_profiles`
	ListMDMAppleProfilesToRemove(ctx context.Context) ([]*MDMAppleProfilePayload, error)

	// BulkUpsertMDMAppleHostProfiles bulk-adds/updates records to track the
	// status of a profile in a host.
	BulkUpsertMDMAppleHostProfiles(ctx context.Context, payload []*MDMAppleBulkUpsertHostProfilePayload) error

	// BulkSetPendingMDMAppleHostProfiles sets the status of profiles to install
	// or to remove for each affected host to pending for the provided criteria,
	// which may be either a list of hostIDs, teamIDs, profileIDs or hostUUIDs
	// (only one of those ID types can be provided).
	BulkSetPendingMDMAppleHostProfiles(ctx context.Context, hostIDs, teamIDs, profileIDs []uint, hostUUIDs []string) error

	// GetMDMAppleProfilesContents retrieves the XML contents of the
	// profiles requested.
	GetMDMAppleProfilesContents(ctx context.Context, profileIDs []uint) (map[uint]mobileconfig.Mobileconfig, error)

	// UpdateOrDeleteHostMDMAppleProfile updates information about a single
	// profile status. It deletes the row if the profile operation is "remove"
	// and the status is "verifying" (i.e. successfully removed).
	UpdateOrDeleteHostMDMAppleProfile(ctx context.Context, profile *HostMDMAppleProfile) error

	// GetMDMAppleCommandRequest type returns the request type for the given command
	GetMDMAppleCommandRequestType(ctx context.Context, commandUUID string) (string, error)

	// GetMDMAppleHostsProfilesSummary summarizes the current state of MDM configuration profiles on
	// each host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetMDMAppleHostsProfilesSummary(ctx context.Context, teamID *uint) (*MDMAppleConfigProfilesSummary, error)

	// InsertMDMIdPAccount inserts a new MDM IdP account
	InsertMDMIdPAccount(ctx context.Context, account *MDMIdPAccount) error

	// GetMDMAppleFileVaultSummary summarizes the current state of Apple disk encryption profiles on
	// each macOS host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetMDMAppleFileVaultSummary(ctx context.Context, teamID *uint) (*MDMAppleFileVaultSummary, error)

	// InsertMDMAppleBootstrapPackage insterts a new bootstrap package in the database
	InsertMDMAppleBootstrapPackage(ctx context.Context, bp *MDMAppleBootstrapPackage) error
	// DeleteMDMAppleBootstrapPackage deletes the bootstrap package for the given team id
	DeleteMDMAppleBootstrapPackage(ctx context.Context, teamID uint) error
	// GetMDMAppleBootstrapPackageMeta returns metadata about the bootstrap package for a team
	GetMDMAppleBootstrapPackageMeta(ctx context.Context, teamID uint) (*MDMAppleBootstrapPackage, error)
	// GetMDMAppleBootstrapPackageBytes returns the bytes of a bootstrap package with the given token
	GetMDMAppleBootstrapPackageBytes(ctx context.Context, token string) (*MDMAppleBootstrapPackage, error)
	// GetMDMAppleBootstrapPackageSummary returns an aggregated summary of
	// the status of the bootstrap package for hosts in a team.
	GetMDMAppleBootstrapPackageSummary(ctx context.Context, teamID uint) (*MDMAppleBootstrapPackageSummary, error)

	// RecordHostBootstrapPackage records a command used to install a
	// bootstrap package in a host.
	RecordHostBootstrapPackage(ctx context.Context, commandUUID string, hostUUID string) error

	// GetHostMDMMacOSSetup returns the MDM macOS setup information for the specified host id.
	GetHostMDMMacOSSetup(ctx context.Context, hostID uint) (*HostMDMMacOSSetup, error)

	// MDMAppleGetEULAMetadata returns metadata information about the EULA
	// filed stored in the database.
	MDMAppleGetEULAMetadata(ctx context.Context) (*MDMAppleEULA, error)
	// MDMAppleGetEULABytes returns the bytes of the EULA file stored in
	// the database. A token is required since this file is publicly
	// accessible by anyone with the token.
	MDMAppleGetEULABytes(ctx context.Context, token string) (*MDMAppleEULA, error)
	// MDMAppleInsertEULA inserts a new EULA in the database
	MDMAppleInsertEULA(ctx context.Context, eula *MDMAppleEULA) error
	// MDMAppleDeleteEULA deletes the EULA file from the database
	MDMAppleDeleteEULA(ctx context.Context, token string) error

	// Create or update the MDM Apple Setup Assistant for a team or no team.
	SetOrUpdateMDMAppleSetupAssistant(ctx context.Context, asst *MDMAppleSetupAssistant) (*MDMAppleSetupAssistant, error)
	// Get the MDM Apple Setup Assistant for the provided team or no team.
	GetMDMAppleSetupAssistant(ctx context.Context, teamID *uint) (*MDMAppleSetupAssistant, error)
	// Delete the MDM Apple Setup Assistant for the provided team or no team.
	DeleteMDMAppleSetupAssistant(ctx context.Context, teamID *uint) error
	// Set the profile UUID generated by the call to Apple's DefineProfile API of
	// the setup assistant for a team or no team.
	SetMDMAppleSetupAssistantProfileUUID(ctx context.Context, teamID *uint, profileUUID string) error
}

const (
	// Default batch size to use for ScheduledQueryIDsByName.
	DefaultScheduledQueryIDsByNameBatchSize = 1000
	// Default batch size for loading IDs of or inserting new munki issues.
	DefaultMunkiIssuesBatchSize = 100
)

type PolicyFailure struct {
	PolicyID uint
	Host     PolicySetHost
}

type MySQLProcess struct {
	Id      int     `json:"id" db:"Id"`
	User    string  `json:"user" db:"User"`
	Host    string  `json:"host" db:"Host"`
	DB      *string `json:"db" db:"db"`
	Command string  `json:"command" db:"Command"`
	Time    int     `json:"time" db:"Time"`
	State   *string `json:"state" db:"State"`
	Info    *string `json:"info" db:"Info"`
}

// HostOsqueryIntervals holds an osquery host's osquery interval configurations.
type HostOsqueryIntervals struct {
	DistributedInterval uint `json:"distributed_interval" db:"distributed_interval"`
	ConfigTLSRefresh    uint `json:"config_tls_refresh" db:"config_tls_refresh"`
	LoggerTLSPeriod     uint `json:"logger_tls_period" db:"logger_tls_period"`
}

type MigrationStatus struct {
	// StatusCode holds the code for the migration status.
	//
	// If StatusCode is NoMigrationsCompleted or AllMigrationsCompleted
	// then all other fields are empty.
	//
	// If StatusCode is SomeMigrationsCompleted, then missing migrations
	// are available in MissingTable and MissingData.
	//
	// If StatusCode is UnknownMigrations, then unknown migrations
	// are available in UnknownTable and UnknownData.
	StatusCode MigrationStatusCode `json:"status_code"`
	// MissingTable holds the missing table migrations.
	MissingTable []int64 `json:"missing_table"`
	// MissingTable holds the missing data migrations.
	MissingData []int64 `json:"missing_data"`
	// UnknownTable holds unknown applied table migrations.
	UnknownTable []int64 `json:"unknown_table"`
	// UnknownTable holds unknown applied data migrations.
	UnknownData []int64 `json:"unknown_data"`
}

type MigrationStatusCode int

const (
	// NoMigrationsCompleted indicates the database has no migrations installed.
	NoMigrationsCompleted MigrationStatusCode = iota
	// SomeMigrationsCompleted indicates some (not all) migrations are missing.
	SomeMigrationsCompleted
	// AllMigrationsCompleted means all migrations have been installed successfully.
	AllMigrationsCompleted
	// UnknownMigrations means some unidentified migrations were detected on the database.
	UnknownMigrations
)

// TODO: we have a similar but different interface in the service package,
// service.NotFoundErr - at the very least, the IsNotFound method should be the
// same in both (the other is currently NotFound), and ideally we'd just have
// one of those interfaces.

// NotFoundError is returned when the datastore resource cannot be found.
type NotFoundError interface {
	error
	IsNotFound() bool
}

func IsNotFound(err error) bool {
	var nfe NotFoundError
	if errors.As(err, &nfe) {
		return nfe.IsNotFound()
	}
	return false
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
	var fke ForeignKeyError
	if errors.As(err, &fke) {
		return fke.IsForeignKey()
	}
	return false
}

type OptionalArg func() interface{}
