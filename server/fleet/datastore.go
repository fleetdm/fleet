package fleet

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"

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
	ApplyQueries(ctx context.Context, authorID uint, queries []*Query, queriesToDiscardResults map[uint]struct{}) error
	// NewQuery creates a new query object in thie datastore. The returned query should have the ID updated.
	NewQuery(ctx context.Context, query *Query, opts ...OptionalArg) (*Query, error)
	// SaveQuery saves changes to an existing query object.
	SaveQuery(ctx context.Context, query *Query, shouldDiscardResults bool, shouldDeleteStats bool) error
	// DeleteQuery deletes an existing query object on a team. If teamID is nil, then the query is
	// looked up in the 'global' team.
	DeleteQuery(ctx context.Context, teamID *uint, name string) error
	// DeleteQueries deletes the existing query objects with the provided IDs. The number of deleted queries is returned
	// along with any error.
	DeleteQueries(ctx context.Context, ids []uint) (uint, error)
	// Query returns the query associated with the provided ID. Associated packs should also be loaded.
	Query(ctx context.Context, id uint) (*Query, error)
	// ListQueries returns a list of queries with the provided sorting and paging options. Associated packs should also
	// be loaded.
	ListQueries(ctx context.Context, opt ListQueryOptions) ([]*Query, error)
	// ListScheduledQueriesForAgents returns a list of scheduled queries (without stats) for the
	// given teamID. If teamID is nil, then all scheduled queries for the 'global' team are returned.
	ListScheduledQueriesForAgents(ctx context.Context, teamID *uint, queryReportsDisabled bool) ([]*Query, error)
	// QueryByName looks up a query by name on a team. If teamID is nil, then the query is looked up in
	// the 'global' team.
	QueryByName(ctx context.Context, teamID *uint, name string) (*Query, error)
	// ObserverCanRunQuery returns whether a user with an observer role is permitted to run the
	// identified query
	ObserverCanRunQuery(ctx context.Context, queryID uint) (bool, error)
	// CleanupGlobalDiscardQueryResults deletes all cached query results. Used in cleanups_then_aggregation cron.
	CleanupGlobalDiscardQueryResults(ctx context.Context) error
	// IsSavedQuery returns true if the given query is a saved query.
	IsSavedQuery(ctx context.Context, queryID uint) (bool, error)
	// GetLiveQueryStats returns the live query stats for the given query and hosts.
	GetLiveQueryStats(ctx context.Context, queryID uint, hostIDs []uint) ([]*LiveQueryStats, error)
	// UpdateLiveQueryStats writes new live query stats as a single operation.
	UpdateLiveQueryStats(ctx context.Context, queryID uint, stats []*LiveQueryStats) error
	// CalculateAggregatedPerfStatsPercentiles calculates the aggregated user/system time performance statistics for the given query.
	CalculateAggregatedPerfStatsPercentiles(ctx context.Context, aggregate AggregatedStatsType, queryID uint) error

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

	// ListPacksForHost lists the "user packs" that a host should execute.
	ListPacksForHost(ctx context.Context, hid uint) (packs []*Pack, err error)

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

	// LabelIDsByName retrieves the IDs associated with the given label names
	LabelIDsByName(ctx context.Context, labels []string) (map[string]uint, error)

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
	GetHostHealth(ctx context.Context, id uint) (*HostHealth, error)
	ListHosts(ctx context.Context, filter TeamFilter, opt HostListOptions) ([]*Host, error)

	// ListHostsLiteByUUIDs returns the "lite" version of hosts corresponding to
	// the provided uuids and filtered according to the provided team filters. It
	// does include the MDMInfo information (unlike HostLite and
	// ListHostsLiteByIDs) because listing hosts by UUIDs is commonly used to
	// support MDM-related operations, where the UUID is often the only available
	// identifier. The "lite" version is a subset of the fields related to the
	// host. See the implementation for the exact list.
	ListHostsLiteByUUIDs(ctx context.Context, filter TeamFilter, uuids []string) ([]*Host, error)

	// ListHostsLiteByIDs returns the "lite" version of hosts corresponding to
	// the provided ids. The "lite" version is a subset of the fields related to
	// the host. See documentation of Datastore.HostLite for more information, or
	// the implementation for the exact list.
	ListHostsLiteByIDs(ctx context.Context, ids []uint) ([]*Host, error)

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

	// HostMemberOfAllLabels returns whether the given host is a member of all the provided labels.
	// If a label name does not exist, then the host is considered not a member of the provided label.
	// A host will always be a member of an empty label set, so this method returns (true, nil)
	// if labelNames is empty.
	HostMemberOfAllLabels(ctx context.Context, hostID uint, labelNames []string) (bool, error)

	// TODO JUAN: Refactor this to use the Operating System type instead.
	// HostIDsByOSVersion retrieves the IDs of all host matching osVersion
	HostIDsByOSVersion(ctx context.Context, osVersion OSVersion, offset int, limit int) ([]uint, error)
	// HostByIdentifier returns one host matching the provided identifier. Possible matches can be on
	// osquery_host_identifier, node_key, UUID, or hostname.
	HostByIdentifier(ctx context.Context, identifier string) (*Host, error)
	// HostLiteByIdentifier returns a host and a subset of its fields using an "identifier" string.
	// The identifier string will be matched against the hostname, osquery_host_id, node_key, uuid and hardware_serial columns.
	HostLiteByIdentifier(ctx context.Context, identifier string) (*HostLite, error)
	// HostLiteByIdentifier returns a host and a subset of its fields from its id.
	HostLiteByID(ctx context.Context, id uint) (*HostLite, error)
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
	// SetOrUpdateCustomHostDeviceMapping replaces the custom email address
	// associated with the host with the provided one.
	SetOrUpdateCustomHostDeviceMapping(ctx context.Context, hostID uint, email, source string) ([]*HostDeviceMapping, error)
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
	OSVersionsByCVE(ctx context.Context, cve string, teamID *uint) ([]*VulnerableOS, time.Time, error)
	SoftwareByCVE(ctx context.Context, cve string, teamID *uint) ([]*VulnerableSoftware, time.Time, error)
	OSVersion(ctx context.Context, osVersionID uint, teamID *uint) (*OSVersion, *time.Time, error)
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
	// QueryResultsStore

	// QueryResultRows returns stored results of a query
	QueryResultRows(ctx context.Context, queryID uint, filter TeamFilter) ([]*ScheduledQueryResultRow, error)
	QueryResultRowsForHost(ctx context.Context, queryID, hostID uint) ([]*ScheduledQueryResultRow, error)
	ResultCountForQuery(ctx context.Context, queryID uint) (int, error)
	ResultCountForQueryAndHost(ctx context.Context, queryID, hostID uint) (int, error)
	OverwriteQueryResultRows(ctx context.Context, rows []*ScheduledQueryResultRow) error
	// CleanupDiscardedQueryResults deletes all query results for queries with DiscardData enabled.
	// Used in cleanups_then_aggregation cron to cleanup rows that were inserted immediately
	// after DiscardData was set to true due to query caching.
	CleanupDiscardedQueryResults(ctx context.Context) error

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
	// TeamExists returns true if a team with the given id exists.
	TeamExists(ctx context.Context, teamID uint) (bool, error)

	///////////////////////////////////////////////////////////////////////////////
	// Software Titles

	ListSoftwareTitles(ctx context.Context, opt SoftwareTitleListOptions, tmFilter TeamFilter) ([]SoftwareTitle, int, *PaginationMetadata, error)
	SoftwareTitleByID(ctx context.Context, id uint, teamID *uint, tmFilter TeamFilter) (*SoftwareTitle, error)

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
	SoftwareByID(ctx context.Context, id uint, teamID *uint, includeCVEScores bool, tmFilter *TeamFilter) (*Software, error)
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

	// ReconcileSoftwareTitles ensures the software_titles and software tables are in sync.
	// It inserts new software titles and updates the software table with the title_id.
	// It also cleans up any software titles that are no longer associated with any software.
	// It is intended to be run after SyncHostsSoftware.
	ReconcileSoftwareTitles(ctx context.Context) error

	// SyncHostsSoftwareTitles calculates the number of hosts having each
	// software_title installed and stores that information in the
	// software_titles_host_counts table.
	SyncHostsSoftwareTitles(ctx context.Context, updatedAt time.Time) error

	// HostVulnSummariesBySoftwareIDs returns a list of all hosts that have at least one of the
	// specified Software installed. Includes the path were the software was installed.
	HostVulnSummariesBySoftwareIDs(ctx context.Context, softwareIDs []uint) ([]HostVulnerabilitySummary, error)
	// *DEPRECATED use HostVulnSummariesBySoftwareIDs instead* HostsByCVE
	// returns a list of all hosts that have at least one software suceptible to the provided CVE.
	// Includes the path were the software was installed.
	HostsByCVE(ctx context.Context, cve string) ([]HostVulnerabilitySummary, error)
	InsertCVEMeta(ctx context.Context, cveMeta []CVEMeta) error
	ListCVEs(ctx context.Context, maxAge time.Duration) ([]CVEMeta, error)

	///////////////////////////////////////////////////////////////////////////////
	// OperatingSystemsStore

	// ListOperationsSystems returns all operating systems (id, name, version)
	ListOperatingSystems(ctx context.Context) ([]OperatingSystem, error)
	// ListOperatingSystemsForPlatform returns all operating systems for the given platform.
	// Supported values for platform are: "darwin" and "windows"
	ListOperatingSystemsForPlatform(ctx context.Context, platform string) ([]OperatingSystem, error)
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
	ListHostUpcomingActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
	ListHostPastActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
	IsExecutionPendingForHost(ctx context.Context, hostID uint, scriptID uint) ([]*uint, error)

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
	PolicyByName(ctx context.Context, name string) (*Policy, error)

	// SavePolicy updates some fields of the given policy on the datastore.
	//
	// It is also used to update team policies.
	SavePolicy(ctx context.Context, p *Policy, shouldRemoveAllPolicyMemberships bool) error

	ListGlobalPolicies(ctx context.Context, opts ListOptions) ([]*Policy, error)
	PoliciesByID(ctx context.Context, ids []uint) (map[uint]*Policy, error)
	DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error)
	CountPolicies(ctx context.Context, teamID *uint, matchQuery string) (int, error)
	UpdateHostPolicyCounts(ctx context.Context) error

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

	ListSoftware(ctx context.Context, opt SoftwareListOptions) ([]Software, *PaginationMetadata, error)
	CountSoftware(ctx context.Context, opt SoftwareListOptions) (int, error)
	// DeleteVulnerabilities deletes the given list of vulnerabilities identified by CPE+CVE.
	DeleteSoftwareVulnerabilities(ctx context.Context, vulnerabilities []SoftwareVulnerability) error
	// DeleteOutOfDateVulnerabilities deletes 'software_cve' entries from the provided source where
	// the updated_at timestamp is older than the provided duration
	DeleteOutOfDateVulnerabilities(ctx context.Context, source VulnerabilitySource, duration time.Duration) error

	///////////////////////////////////////////////////////////////////////////////
	// Team Policies

	NewTeamPolicy(ctx context.Context, teamID uint, authorID *uint, args PolicyPayload) (*Policy, error)
	ListTeamPolicies(ctx context.Context, teamID uint, opts ListOptions, iopts ListOptions) (teamPolicies, inheritedPolicies []*Policy, err error)
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
	SaveHostPackStats(ctx context.Context, teamID *uint, hostID uint, stats []PackStats) error
	// AsyncBatchSaveHostsScheduledQueryStats efficiently saves a batch of hosts'
	// pack stats of scheduled queries. It is the async and batch version of
	// SaveHostPackStats. It returns the number of INSERT-ON DUPLICATE UPDATE
	// statements that were executed (for reporting purpose) or an error.
	AsyncBatchSaveHostsScheduledQueryStats(ctx context.Context, stats map[uint][]ScheduledQueryStats, batchSize int) (int, error)

	// UpdateHostSoftware updates the software list of a host.
	// The update consists of deleting existing entries that are not in the given `software`
	// slice, updating existing entries and inserting new entries.
	// Returns a struct with the current installed software on the host (pre-mutations) plus all
	// mutations performed: what was inserted and what was removed.
	UpdateHostSoftware(ctx context.Context, hostID uint, software []Software) (*UpdateHostSoftwareDBResult, error)

	// UpdateHostSoftwareInstalledPaths looks at all software for 'hostID' and based on the contents of
	// 'reported', either inserts or deletes the corresponding entries in the
	// 'host_software_installed_paths' table. 'reported' is a set of
	// 'software.ToUniqueStr()--installed_path' strings. 'mutationResults' contains the software inventory of
	// the host (pre-mutations) and the mutations performed after calling 'UpdateHostSoftware',
	// it is used as DB optimization.
	UpdateHostSoftwareInstalledPaths(ctx context.Context, hostID uint, reported map[string]struct{}, mutationResults *UpdateHostSoftwareDBResult) error

	// UpdateHost updates a host.
	UpdateHost(ctx context.Context, host *Host) error

	// ListScheduledQueriesInPack lists all the scheduled queries of a pack.
	ListScheduledQueriesInPack(ctx context.Context, packID uint) (ScheduledQueryList, error)

	// UpdateHostRefetchRequested updates a host's refetch requested field.
	UpdateHostRefetchRequested(ctx context.Context, hostID uint, value bool) error

	// UpdateHostRefetchCriticalQueriesUntil updates a host's refetch critical queries until field.
	UpdateHostRefetchCriticalQueriesUntil(ctx context.Context, hostID uint, until *time.Time) error

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
	// Even if `results` is empty, the host's `policy_updated_at` will be updated.
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
	SetOrUpdateMDMData(ctx context.Context, hostID uint, isServer, enrolled bool, serverURL string, installedFromDep bool, name string, fleetEnrollRef string) error
	// SetOrUpdateHostEmailsFromMdmIdpAccounts sets or updates the host emails associated with the provided
	// host based on the MDM IdP account information associated with the provided fleet enrollment reference.
	SetOrUpdateHostEmailsFromMdmIdpAccounts(ctx context.Context, hostID uint, fleetEnrollmentRef string) error
	SetOrUpdateHostDisksSpace(ctx context.Context, hostID uint, gigsAvailable, percentAvailable, gigsTotal float64) error
	SetOrUpdateHostDisksEncryption(ctx context.Context, hostID uint, encrypted bool) error
	// SetOrUpdateHostDiskEncryptionKey sets the base64, encrypted key for
	// a host
	SetOrUpdateHostDiskEncryptionKey(ctx context.Context, hostID uint, encryptedBase64Key, clientError string, decryptable *bool) error
	// GetUnverifiedDiskEncryptionKeys returns all the encryption keys that
	// are collected but their decryptable status is not known yet (ie:
	// we're able to decrypt the key using a private key in the server)
	GetUnverifiedDiskEncryptionKeys(ctx context.Context) ([]HostDiskEncryptionKey, error)
	// SetHostsDiskEncryptionKeyStatus sets the encryptable status for the set
	// of encription keys provided
	SetHostsDiskEncryptionKeyStatus(ctx context.Context, hostIDs []uint, encryptable bool, threshold time.Time) error
	// GetHostDiskEncryptionKey returns the encryption key information for a given host
	GetHostDiskEncryptionKey(ctx context.Context, hostID uint) (*HostDiskEncryptionKey, error)

	SetDiskEncryptionResetStatus(ctx context.Context, hostID uint, status bool) error

	// GetHostCertAssociationsToExpire retrieves host certificate
	// associations that are close to expire and don't have a renewal in
	// progress based on the provided arguments.
	GetHostCertAssociationsToExpire(ctx context.Context, expiryDays, limit int) ([]SCEPIdentityAssociation, error)

	// SetCommandForPendingSCEPRenewal tracks the command used to renew a scep certificate
	SetCommandForPendingSCEPRenewal(ctx context.Context, assocs []SCEPIdentityAssociation, cmdUUID string) error

	// UpdateVerificationHostMacOSProfiles updates status of macOS profiles installed on a given
	// host. The toVerify, toFail, and toRetry slices contain the identifiers of the profiles that
	// should be verified, failed, and retried, respectively. For each profile in the toRetry slice,
	// the retries count is incremented by 1 and the status is set to null so that an install
	// profile command is enqueued the next time the profile manager cron runs.
	UpdateHostMDMProfilesVerification(ctx context.Context, host *Host, toVerify, toFail, toRetry []string) error
	// GetHostMDMProfilesExpected returns the expected MDM profiles for a given host. The map is
	// keyed by the profile identifier.
	GetHostMDMProfilesExpectedForVerification(ctx context.Context, host *Host) (map[string]*ExpectedMDMProfile, error)
	// GetHostMDMProfilesRetryCounts returns a list of MDM profile retry counts for a given host.
	GetHostMDMProfilesRetryCounts(ctx context.Context, host *Host) ([]HostMDMProfileRetryCount, error)
	// GetHostMDMProfileRetryCountByCommandUUID returns the retry count for the specified
	// host UUID and command UUID.
	GetHostMDMProfileRetryCountByCommandUUID(ctx context.Context, host *Host, cmdUUID string) (HostMDMProfileRetryCount, error)

	// SetOrUpdateHostOrbitInfo inserts of updates the orbit info for a host
	SetOrUpdateHostOrbitInfo(ctx context.Context, hostID uint, version string) error

	ReplaceHostDeviceMapping(ctx context.Context, id uint, mappings []*HostDeviceMapping, source string) error

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
	ListOSVulnerabilitiesByOS(ctx context.Context, osID uint) ([]OSVulnerability, error)
	ListVulnsByOsNameAndVersion(ctx context.Context, name, version string, includeCVSS bool) (Vulnerabilities, error)
	InsertOSVulnerabilities(ctx context.Context, vulnerabilities []OSVulnerability, source VulnerabilitySource) (int64, error)
	DeleteOSVulnerabilities(ctx context.Context, vulnerabilities []OSVulnerability) error
	// InsertOSVulnerability will either insert a new vulnerability in the datastore (in which
	// case it will return true) or if a matching record already exists it will update its
	// updated_at timestamp (in which case it will return false).
	InsertOSVulnerability(ctx context.Context, vuln OSVulnerability, source VulnerabilitySource) (bool, error)
	// DeleteOutOfDateVulnerabilities deletes 'operating_system_vulnerabilities' entries from the provided source where
	// the updated_at timestamp is older than the provided duration
	DeleteOutOfDateOSVulnerabilities(ctx context.Context, source VulnerabilitySource, duration time.Duration) error

	///////////////////////////////////////////////////////////////////////////////
	// Vulnerabilities

	// ListVulnerabilities returns a list of unique vulnerabilities based on the provided options.
	ListVulnerabilities(ctx context.Context, opt VulnListOptions) ([]VulnerabilityWithMetadata, *PaginationMetadata, error)
	// Vulnerability returns the vulnerability corresponding to the specified CVE ID
	Vulnerability(ctx context.Context, cve string, teamID *uint, includeCVEScores bool) (*VulnerabilityWithMetadata, error)
	// CountVulnerabilities returns the number of unique vulnerabilities based on the provided
	// options.
	CountVulnerabilities(ctx context.Context, opt VulnListOptions) (uint, error)
	// UpdateVulnerabilityHostCounts updates hosts counts for all vulnerabilities.
	UpdateVulnerabilityHostCounts(ctx context.Context) error

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

	// GetMDMAppleConfigProfileByDeprecatedID returns the mdm config profile
	// corresponding to the specified numeric profile id. This is deprecated and
	// should not be used for new endpoints.
	GetMDMAppleConfigProfileByDeprecatedID(ctx context.Context, profileID uint) (*MDMAppleConfigProfile, error)
	// GetMDMAppleConfigProfile returns the mdm config profile corresponding to the specified
	// profile uuid.
	GetMDMAppleConfigProfile(ctx context.Context, profileUUID string) (*MDMAppleConfigProfile, error)

	// ListMDMAppleConfigProfiles lists mdm config profiles associated with the specified team id.
	// For global config profiles, specify nil as the team id.
	ListMDMAppleConfigProfiles(ctx context.Context, teamID *uint) ([]*MDMAppleConfigProfile, error)

	// DeleteMDMAppleConfigProfileByDeprecatedID deletes the mdm config profile
	// corresponding to the specified numeric profile id. This is deprecated and
	// should not be used for new endpoints.
	DeleteMDMAppleConfigProfileByDeprecatedID(ctx context.Context, profileID uint) error
	// DeleteMDMAppleConfigProfile deletes the mdm config profile corresponding
	// to the specified profile uuid.
	DeleteMDMAppleConfigProfile(ctx context.Context, profileUUID string) error

	BulkDeleteMDMAppleHostsConfigProfiles(ctx context.Context, payload []*MDMAppleProfilePayload) error

	// DeleteMDMAppleConfigProfileByTeamAndIdentifier deletes a configuration
	// profile using the unique key defined by `team_id` and `identifier`
	DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx context.Context, teamID *uint, profileIdentifier string) error

	// GetHostMDMAppleProfiles returns the MDM profile information for the specified host UUID.
	GetHostMDMAppleProfiles(ctx context.Context, hostUUID string) ([]HostMDMAppleProfile, error)

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
	GetMDMAppleCommandResults(ctx context.Context, commandUUID string) ([]*MDMCommandResult, error)

	// ListMDMAppleCommands returns a list of MDM Apple commands that have been
	// executed, based on the provided options.
	ListMDMAppleCommands(ctx context.Context, tmFilter TeamFilter, listOpts *MDMCommandListOptions) ([]*MDMAppleCommand, error)

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

	// UpsertMDMAppleHostDEPAssignments ensures there's an entry in
	// `host_dep_assignments` for all the provided hosts.
	UpsertMDMAppleHostDEPAssignments(ctx context.Context, hosts []Host) error

	// IngestMDMAppleDevicesFromDEPSync creates new Fleet host records for MDM-enrolled devices that are
	// not already enrolled in Fleet. It returns the number of hosts created, the team id that they
	// joined (nil for no team), and an error.
	IngestMDMAppleDevicesFromDEPSync(ctx context.Context, devices []godep.Device) (int64, *uint, error)

	// IngestMDMAppleDeviceFromCheckin creates a new Fleet host record for an MDM-enrolled device that is
	// not already enrolled in Fleet.
	IngestMDMAppleDeviceFromCheckin(ctx context.Context, mdmHost MDMAppleHostDetails) error

	// RestoreMDMApplePendingDEPHost restores a host that was previously deleted from Fleet.
	RestoreMDMApplePendingDEPHost(ctx context.Context, host *Host) error

	// ResetMDMAppleEnrollment resets all tables with enrollment-related
	// information if a matching row for the host exists.
	ResetMDMAppleEnrollment(ctx context.Context, hostUUID string) error

	// ListMDMAppleDEPSerialsInTeam returns a list of serial numbers of hosts
	// that are enrolled or pending enrollment in Fleet's MDM via DEP for the
	// specified team (or no team if teamID is nil).
	ListMDMAppleDEPSerialsInTeam(ctx context.Context, teamID *uint) ([]string, error)

	// ListMDMAppleDEPSerialsInHostIDs returns a list of serial numbers of hosts
	// that are enrolled or pending enrollment in Fleet's MDM via DEP in the
	// specified list of host IDs.
	ListMDMAppleDEPSerialsInHostIDs(ctx context.Context, hostIDs []uint) ([]string, error)

	// GetHostDEPAssignment returns the DEP assignment for the host.
	GetHostDEPAssignment(ctx context.Context, hostID uint) (*HostDEPAssignment, error)

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

	// BulkSetPendingMDMHostProfiles sets the status of profiles to install or to
	// remove for each affected host to pending for the provided criteria, which
	// may be either a list of hostIDs, teamIDs, profileUUIDs or hostUUIDs (only
	// one of those ID types can be provided).
	BulkSetPendingMDMHostProfiles(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, hostUUIDs []string) error

	// GetMDMAppleProfilesContents retrieves the XML contents of the
	// profiles requested.
	GetMDMAppleProfilesContents(ctx context.Context, profileUUIDs []string) (map[string]mobileconfig.Mobileconfig, error)

	// UpdateOrDeleteHostMDMAppleProfile updates information about a single
	// profile status. It deletes the row if the profile operation is "remove"
	// and the status is "verifying" (i.e. successfully removed).
	UpdateOrDeleteHostMDMAppleProfile(ctx context.Context, profile *HostMDMAppleProfile) error

	// GetMDMAppleCommandRequest type returns the request type for the given command
	GetMDMAppleCommandRequestType(ctx context.Context, commandUUID string) (string, error)

	// GetMDMAppleProfilesSummary summarizes the current state of MDM configuration profiles on
	// each host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetMDMAppleProfilesSummary(ctx context.Context, teamID *uint) (*MDMProfilesSummary, error)

	// InsertMDMIdPAccount inserts a new MDM IdP account
	InsertMDMIdPAccount(ctx context.Context, account *MDMIdPAccount) error

	// GetMDMIdPAccountByUUID returns MDM IdP account that matches the given token.
	GetMDMIdPAccountByUUID(ctx context.Context, uuid string) (*MDMIdPAccount, error)

	// GetMDMIdPAccountByEmail returns MDM IdP account that matches the given email.
	GetMDMIdPAccountByEmail(ctx context.Context, email string) (*MDMIdPAccount, error)

	// GetMDMAppleFileVaultSummary summarizes the current state of Apple disk encryption profiles on
	// each macOS host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetMDMAppleFileVaultSummary(ctx context.Context, teamID *uint) (*MDMAppleFileVaultSummary, error)

	// InsertMDMAppleBootstrapPackage insterts a new bootstrap package in the database
	InsertMDMAppleBootstrapPackage(ctx context.Context, bp *MDMAppleBootstrapPackage) error
	// CopyMDMAppleBootstrapPackage copies the bootstrap package specified in the app config (if any)
	// specified team (and a new token is assigned). It also updates the team config with the default bootstrap package URL.
	CopyDefaultMDMAppleBootstrapPackage(ctx context.Context, ac *AppConfig, toTeamID uint) error
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

	// MDMGetEULAMetadata returns metadata information about the EULA
	// filed stored in the database.
	MDMGetEULAMetadata(ctx context.Context) (*MDMEULA, error)
	// MDMGetEULABytes returns the bytes of the EULA file stored in
	// the database. A token is required since this file is publicly
	// accessible by anyone with the token.
	MDMGetEULABytes(ctx context.Context, token string) (*MDMEULA, error)
	// MDMInsertEULA inserts a new EULA in the database
	MDMInsertEULA(ctx context.Context, eula *MDMEULA) error
	// MDMDeleteEULA deletes the EULA file from the database
	MDMDeleteEULA(ctx context.Context, token string) error

	// Create or update the MDM Apple Setup Assistant for a team or no team.
	SetOrUpdateMDMAppleSetupAssistant(ctx context.Context, asst *MDMAppleSetupAssistant) (*MDMAppleSetupAssistant, error)
	// Get the MDM Apple Setup Assistant for the provided team or no team.
	GetMDMAppleSetupAssistant(ctx context.Context, teamID *uint) (*MDMAppleSetupAssistant, error)
	// Delete the MDM Apple Setup Assistant for the provided team or no team.
	DeleteMDMAppleSetupAssistant(ctx context.Context, teamID *uint) error
	// Set the profile UUID generated by the call to Apple's DefineProfile API of
	// the setup assistant for a team or no team.
	SetMDMAppleSetupAssistantProfileUUID(ctx context.Context, teamID *uint, profileUUID string) error

	// Set the profile UUID generated by the call to Apple's DefineProfile API
	// of the default setup assistant for a team or no team. The default
	// profile is the same regardless of the team, except for the enabling of
	// the end-user authentication which may be configured per-team and affects
	// the JSON registered with Apple's API, possibly resulting in different
	// profile UUIDs for the same profile depending on the team.
	SetMDMAppleDefaultSetupAssistantProfileUUID(ctx context.Context, teamID *uint, profileUUID string) error

	// Get the profile UUID and last update timestamp for the default setup
	// assistant for a team or no team.
	GetMDMAppleDefaultSetupAssistant(ctx context.Context, teamID *uint) (profileUUID string, updatedAt time.Time, err error)

	// GetMatchingHostSerials receives a list of serial numbers and returns
	// a map that only contains the serials that have a matching row in the `hosts` table.
	GetMatchingHostSerials(ctx context.Context, serials []string) (map[string]*Host, error)

	// DeleteHostDEPAssignments marks as deleted entries in
	// host_dep_assignments for host with matching serials.
	DeleteHostDEPAssignments(ctx context.Context, serials []string) error

	///////////////////////////////////////////////////////////////////////////////
	// Microsoft MDM

	// WSTEPStoreCertificate stores a certificate in the database.
	WSTEPStoreCertificate(ctx context.Context, name string, crt *x509.Certificate) error
	// WSTEPNewSerial returns a new serial number for a certificate.
	WSTEPNewSerial(ctx context.Context) (*big.Int, error)
	// WSTEPAssociateCertHash associates a certificate hash with a device.
	WSTEPAssociateCertHash(ctx context.Context, deviceUUID string, hash string) error

	// MDMWindowsInsertEnrolledDevice inserts a new MDMWindowsEnrolledDevice in the database
	MDMWindowsInsertEnrolledDevice(ctx context.Context, device *MDMWindowsEnrolledDevice) error

	// MDMWindowsDeleteEnrolledDevice deletes a give MDMWindowsEnrolledDevice entry from the database using the HW device id.
	MDMWindowsDeleteEnrolledDevice(ctx context.Context, mdmDeviceHWID string) error

	// MDMWindowsGetEnrolledDeviceWithDeviceID receives a Windows MDM device id and returns the device information
	MDMWindowsGetEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) (*MDMWindowsEnrolledDevice, error)

	// MDMWindowsDeleteEnrolledDeviceWithDeviceID deletes a give MDMWindowsEnrolledDevice entry from the database using the device id
	MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) error

	// MDMWindowsInsertCommandForHosts inserts a single command that may
	// target multiple hosts identified by their UUID, enqueuing one command
	// for each device.
	MDMWindowsInsertCommandForHosts(ctx context.Context, hostUUIDs []string, cmd *MDMWindowsCommand) error

	// MDMWindowsGetPendingCommands returns all the pending commands for a device
	MDMWindowsGetPendingCommands(ctx context.Context, deviceID string) ([]*MDMWindowsCommand, error)

	// MDMWindowsSaveResponse saves a full response
	MDMWindowsSaveResponse(ctx context.Context, deviceID string, fullResponse *SyncML) error

	// GetMDMWindowsCommands returns the results of command
	GetMDMWindowsCommandResults(ctx context.Context, commandUUID string) ([]*MDMCommandResult, error)

	// UpdateMDMWindowsEnrollmentsHostUUID updates the host UUID for a given MDM device ID.
	UpdateMDMWindowsEnrollmentsHostUUID(ctx context.Context, hostUUID string, mdmDeviceID string) error

	// GetMDMWindowsConfigProfile returns the Windows MDM profile corresponding
	// to the specified profile uuid.
	GetMDMWindowsConfigProfile(ctx context.Context, profileUUID string) (*MDMWindowsConfigProfile, error)

	// DeleteMDMWindowsConfigProfile deletes the Windows MDM profile corresponding to
	// the specified profile uuid.
	DeleteMDMWindowsConfigProfile(ctx context.Context, profileUUID string) error

	// DeleteMDMWindowsConfigProfileByTeamAndName deletes the Windows MDM profile corresponding to
	// the specified team ID (or no team if nil) and profile name.
	DeleteMDMWindowsConfigProfileByTeamAndName(ctx context.Context, teamID *uint, profileName string) error

	// GetHostMDMWindowsProfiles returns the MDM profile information for the specified Windows host UUID.
	GetHostMDMWindowsProfiles(ctx context.Context, hostUUID string) ([]HostMDMWindowsProfile, error)

	// ListMDMConfigProfiles returns a paginated list of configuration profiles
	// corresponding to the criteria.
	ListMDMConfigProfiles(ctx context.Context, teamID *uint, opt ListOptions) ([]*MDMConfigProfilePayload, *PaginationMetadata, error)

	///////////////////////////////////////////////////////////////////////////////
	// MDM Commands

	// GetMDMCommandPlatform returns the platform (i.e. "darwin" or "windows") for the given command.
	GetMDMCommandPlatform(ctx context.Context, commandUUID string) (string, error)

	// ListMDMAppleCommands returns a list of MDM Apple commands that have been
	// executed, based on the provided options.
	ListMDMCommands(ctx context.Context, tmFilter TeamFilter, listOpts *MDMCommandListOptions) ([]*MDMCommand, error)

	// GetMDMWindowsBitLockerSummary summarizes the current state of Windows disk encryption on
	// each Windows host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetMDMWindowsBitLockerSummary(ctx context.Context, teamID *uint) (*MDMWindowsBitLockerSummary, error)
	// GetMDMWindowsBitLockerStatus returns the disk encryption status for a given host
	//
	// Note that the returned status will be nil if the host is reported to be a Windows
	// server or if disk encryption is disabled for the host's team (or no team, as applicable).
	GetMDMWindowsBitLockerStatus(ctx context.Context, host *Host) (*HostMDMDiskEncryption, error)

	// GetMDMWindowsProfilesSummary summarizes the current state of Windows profiles on
	// each Windows host in the specified team (or, if no team is specified, each host that is not
	// assigned to any team).
	GetMDMWindowsProfilesSummary(ctx context.Context, teamID *uint) (*MDMProfilesSummary, error)

	///////////////////////////////////////////////////////////////////////////////
	// Windows MDM Profiles

	// ListMDMWindowsProfilesToInstall returns all the profiles that should
	// be installed based on diffing the ideal state vs the state we have
	// registered in `host_mdm_windows_profiles`
	ListMDMWindowsProfilesToInstall(ctx context.Context) ([]*MDMWindowsProfilePayload, error)

	// ListMDMWindowsProfilesToRemove returns all the profiles that should
	// be removed based on diffing the ideal state vs the state we have
	// registered in `host_mdm_windows_profiles`
	ListMDMWindowsProfilesToRemove(ctx context.Context) ([]*MDMWindowsProfilePayload, error)

	// BulkUpsertMDMWindowsHostProfiles bulk-adds/updates records to track the
	// status of a profile in a host.
	BulkUpsertMDMWindowsHostProfiles(ctx context.Context, payload []*MDMWindowsBulkUpsertHostProfilePayload) error

	// GetMDMWindowsProfilesContents retrieves the XML contents of the
	// profiles requested.
	GetMDMWindowsProfilesContents(ctx context.Context, profileUUIDs []string) (map[string][]byte, error)

	// BulkDeleteMDMWindowsHostsConfigProfiles deletes entries from
	// host_mdm_windows_profiles that match the given payload.
	BulkDeleteMDMWindowsHostsConfigProfiles(ctx context.Context, payload []*MDMWindowsProfilePayload) error

	// NewMDMWindowsConfigProfile creates and returns a new configuration profile.
	NewMDMWindowsConfigProfile(ctx context.Context, cp MDMWindowsConfigProfile) (*MDMWindowsConfigProfile, error)

	// SetOrUpdateMDMWindowsConfigProfile creates or replaces a Windows profile.
	// The profile gets replaced if it already exists for the same team and name
	// combination.
	SetOrUpdateMDMWindowsConfigProfile(ctx context.Context, cp MDMWindowsConfigProfile) error

	// BatchSetMDMProfiles sets the MDM Apple or Windows profiles for the given team or
	// no team in a single transaction.
	BatchSetMDMProfiles(ctx context.Context, tmID *uint, macProfiles []*MDMAppleConfigProfile, winProfiles []*MDMWindowsConfigProfile) error

	///////////////////////////////////////////////////////////////////////////////
	// Host Script Results

	// NewHostScriptExecutionRequest creates a new host script result entry with
	// just the script to run information (result is not yet available).
	NewHostScriptExecutionRequest(ctx context.Context, request *HostScriptRequestPayload) (*HostScriptResult, error)
	// SetHostScriptExecutionResult stores the result of a host script execution
	// and returns the updated host script result record. Note that it does not
	// fail if the script execution request does not exist, in this case it will
	// return nil, nil.
	SetHostScriptExecutionResult(ctx context.Context, result *HostScriptResultPayload) (*HostScriptResult, error)
	// GetHostScriptExecutionResult returns the result of a host script
	// execution. It returns the host script results even if no results have been
	// received, it is the caller's responsibility to check if that was the case
	// (with ExitCode being null).
	GetHostScriptExecutionResult(ctx context.Context, execID string) (*HostScriptResult, error)
	// ListPendingHostScriptExecutions returns all the pending host script
	// executions, which are those that have yet to record a result.
	ListPendingHostScriptExecutions(ctx context.Context, hostID uint) ([]*HostScriptResult, error)

	// NewScript creates a new saved script.
	NewScript(ctx context.Context, script *Script) (*Script, error)

	// Script returns the saved script corresponding to id.
	Script(ctx context.Context, id uint) (*Script, error)

	// GetScriptContents returns the raw script contents of the corresponding
	// script.
	GetScriptContents(ctx context.Context, id uint) ([]byte, error)

	// DeleteScript deletes the script identified by its id.
	DeleteScript(ctx context.Context, id uint) error

	// ListScripts returns a paginated list of scripts corresponding to the
	// criteria.
	ListScripts(ctx context.Context, teamID *uint, opt ListOptions) ([]*Script, *PaginationMetadata, error)

	// GetHostScriptDetails returns the list of host script details for saved scripts applicable to
	// a given host.
	GetHostScriptDetails(ctx context.Context, hostID uint, teamID *uint, opts ListOptions, hostPlatform string) ([]*HostScriptDetail, *PaginationMetadata, error)

	// BatchSetScripts sets the scripts for the given team or no team.
	BatchSetScripts(ctx context.Context, tmID *uint, scripts []*Script) error

	// GetHostLockWipeStatus gets the lock/unlock and wipe status for the host.
	GetHostLockWipeStatus(ctx context.Context, hostID uint, fleetPlatform string) (*HostLockWipeStatus, error)

	// LockHostViaScript sends a script to lock a host and updates the
	// states in host_mdm_actions
	LockHostViaScript(ctx context.Context, request *HostScriptRequestPayload) error

	// UnlockHostViaScript sends a script to unlock a host and updates the
	// states in host_mdm_actions
	UnlockHostViaScript(ctx context.Context, request *HostScriptRequestPayload) error

	// UnlockHostmanually records a request to unlock a host that requires manual
	// intervention (such as for macOS). It indicates the an unlock request is
	// pending.
	UnlockHostManually(ctx context.Context, hostID uint, ts time.Time) error

	// CleanMacOSMDMLock cleans the lock status and pin for a macOS device
	// after it has been unlocked.
	CleanMacOSMDMLock(ctx context.Context, hostUUID string) error
}

// MDMAppleStore wraps nanomdm's storage and adds methods to deal with
// Fleet-specific use cases.
type MDMAppleStore interface {
	storage.AllStorage
	EnqueueDeviceLockCommand(ctx context.Context, host *Host, cmd *mdm.Command, pin string) error
}

// Cloner represents any type that can clone itself. Used for the cached_mysql
// caching layer.
type Cloner interface {
	Clone() (Cloner, error)
}

const (
	// Default batch size to use for ScheduledQueryIDsByName.
	DefaultScheduledQueryIDsByNameBatchSize = 1000
	// Default batch size for loading IDs of or inserting new munki issues.
	DefaultMunkiIssuesBatchSize = 100
)

// ProfileVerificationStore is the minimal interface required to get and update the verification
// status of a host's MDM profiles. The Fleet Datastore satisfies this interface.
type ProfileVerificationStore interface {
	// GetHostMDMProfilesExpectedForVerification returns the expected MDM profiles for a given host. The map is
	// keyed by the profile identifier.
	GetHostMDMProfilesExpectedForVerification(ctx context.Context, host *Host) (map[string]*ExpectedMDMProfile, error)
	// GetHostMDMProfilesRetryCounts returns the retry counts for the specified host.
	GetHostMDMProfilesRetryCounts(ctx context.Context, host *Host) ([]HostMDMProfileRetryCount, error)
	// GetHostMDMProfileRetryCountByCommandUUID returns the retry count for the specified
	// host UUID and command UUID.
	GetHostMDMProfileRetryCountByCommandUUID(ctx context.Context, host *Host, commandUUID string) (HostMDMProfileRetryCount, error)
	// UpdateHostMDMProfilesVerification updates status of macOS profiles installed on a given
	// host. The toVerify, toFail, and toRetry slices contain the identifiers of the profiles that
	// should be verified, failed, and retried, respectively. For each profile in the toRetry slice,
	// the retries count is incremented by 1 and the status is set to null so that an install
	// profile command is enqueued the next time the profile manager cron runs.
	UpdateHostMDMProfilesVerification(ctx context.Context, host *Host, toVerify, toFail, toRetry []string) error
	// UpdateOrDeleteHostMDMAppleProfile updates information about a single
	// profile status. It deletes the row if the profile operation is "remove"
	// and the status is "verifying" (i.e. successfully removed).
	UpdateOrDeleteHostMDMAppleProfile(ctx context.Context, profile *HostMDMAppleProfile) error
}

var _ ProfileVerificationStore = (Datastore)(nil)

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
