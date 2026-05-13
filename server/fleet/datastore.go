package fleet

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	platform_errors "github.com/fleetdm/fleet/v4/server/platform/errors"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/jmoiron/sqlx"
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

type CarveBySessionIder interface {
	CarveBySessionId(ctx context.Context, sessionId string) (*CarveMetadata, error)
}

// InstallerStore is used to communicate to a blob storage containing pre-built
// fleet-osquery installers. This was originally implemented to support the
// Fleet Sandbox and is not expected to be used outside of this:
// https://fleetdm.com/docs/configuration/fleet-server-configuration#packaging
type InstallerStore interface {
	Get(ctx context.Context, installer Installer) (io.ReadCloser, int64, error)
	Put(ctx context.Context, installer Installer) (string, error)
	Exists(ctx context.Context, installer Installer) (bool, error)
}

// CleanupExcessQueryResultRowsOptions configures the behavior of CleanupExcessQueryResultRows.
type CleanupExcessQueryResultRowsOptions struct {
	// BatchSize is the number of rows to delete per batch. Defaults to 500 if not set.
	BatchSize int
}

// Datastore combines all the interfaces in the Fleet DAL
type Datastore interface {
	GetsAppConfig
	AccessesMDMConfigAssets
	health.Checker

	CarveStore

	///////////////////////////////////////////////////////////////////////////////
	// UserStore contains methods for managing users in a datastore

	NewUser(ctx context.Context, user *User) (*User, error)
	// HasUsers returns whether Fleet has any users registered
	HasUsers(ctx context.Context) (bool, error)
	ListUsers(ctx context.Context, opt UserListOptions) ([]*User, error)
	// UsersByIDs returns minimal user info matching the provided IDs.
	UsersByIDs(ctx context.Context, ids []uint) ([]*UserSummary, error)
	UserByEmail(ctx context.Context, email string) (*User, error)
	UserByID(ctx context.Context, id uint) (*User, error)
	UserOrDeletedUserByID(ctx context.Context, id uint) (*User, error)
	SaveUser(ctx context.Context, user *User) error
	SaveUsers(ctx context.Context, users []*User) error
	// DeleteUser permanently deletes the user identified by the provided ID.
	DeleteUser(ctx context.Context, id uint) error
	// DeleteUserIfNotLastAdmin atomically checks that the user being deleted
	// is not the last global admin before deleting. Returns ErrLastGlobalAdmin
	// if the user is the last global admin.
	DeleteUserIfNotLastAdmin(ctx context.Context, id uint) error
	// SaveUserIfNotLastAdmin atomically checks that there's more than one admin
	// before saving the user. Returns ErrLastGlobalAdmin if there's only one last global admin.
	SaveUserIfNotLastAdmin(ctx context.Context, user *User) error
	// PendingEmailChange creates a record with a pending email change for a user identified by uid. The change record
	// is keyed by a unique token. The token is emailed to the user with a link that they can use to confirm the change.
	PendingEmailChange(ctx context.Context, userID uint, newEmail, token string) error
	// ConfirmPendingEmailChange will confirm new email address identified by token is valid. The new email will be
	// written to user record. userID is the ID of the user whose e-mail is being changed.
	ConfirmPendingEmailChange(ctx context.Context, userID uint, token string) (string, error)

	UserSettings(ctx context.Context, userID uint) (*UserSettings, error)

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
	// ListQueries returns a list of queries filtered with the provided sorting and pagination
	// options, a count of total queries on all pages, a count of inherited (global) queries, and
	// pagination metadata. Associated packs should also be loaded.
	// The inherited count is only computed when TeamID is set and MergeInherited is true; otherwise it is 0.
	ListQueries(ctx context.Context, opt ListQueryOptions) ([]*Query, int, int, *PaginationMetadata, error)
	// ListScheduledQueriesForAgents returns a list of scheduled queries (without stats) for the
	// given teamID and hostID. If teamID is nil, then scheduled queries for the 'global' team are returned.
	ListScheduledQueriesForAgents(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*Query, error)
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

	// CleanupDistributedQueryCampaigns will clean and trim metadata for old
	// distributed query campaigns. Any campaign in the QueryWaiting state will
	// be moved to QueryComplete after one minute. Any campaign in the
	// QueryRunning state will be moved to QueryComplete after one day. Times are
	// from creation time. The now parameter makes this method easier to test.
	// The return values indicate how many campaigns were expired and any error.
	CleanupDistributedQueryCampaigns(ctx context.Context, now time.Time) (expired uint, err error)

	// CleanupCompletedCampaignTargets removes campaign targets for campaigns that have been
	// completed for more than the specified duration. This helps reduce database size by
	// cleaning up historical data that is no longer needed. Returns the number of
	// targets deleted.
	CleanupCompletedCampaignTargets(ctx context.Context, olderThan time.Time) (deleted uint, err error)

	// GetCompletedCampaigns returns the IDs of the campaigns that are in the fleet.QueryComplete state and that are in the
	// provided list of IDs. The return value is a slice of the IDs of the completed campaigns and any error.
	GetCompletedCampaigns(ctx context.Context, filter []uint) ([]uint, error)

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
	// ApplyLabelSpecs does the same as ApplyLabelSpecs, additionally allowing an author ID to be set for the labels.
	ApplyLabelSpecsWithAuthor(ctx context.Context, specs []*LabelSpec, authorId *uint) error
	// SetAsideLabels moves a set of labels out of the way if those labels *aren't* on the specified team and *are*
	// writable by the specified user
	SetAsideLabels(ctx context.Context, notOnTeamID *uint, names []string, user User) error
	// GetLabelSpecs returns all of the stored LabelSpecs that the user can see, optionally filtered to
	// a specific team (or global-only); in this case the team filter does *not* include global
	// labels if the user asks for a specific team
	GetLabelSpecs(ctx context.Context, filter TeamFilter) ([]*LabelSpec, error)
	// GetLabelSpec returns the spec for the named label, filtered by the provided team filter.
	GetLabelSpec(ctx context.Context, filter TeamFilter, name string) (*LabelSpec, error)

	// AddLabelsToHost adds the given label IDs membership to the host, with the assumption that the label
	// is available for the host (visibility checks are assumed to have been done prior to this call).
	// If a host is already a member of the label then this will update the row's updated_at.
	AddLabelsToHost(ctx context.Context, hostID uint, labelIDs []uint) error
	// RemoveLabelsFromHost removes the given label IDs membership from the host.
	// If a host is already not a member of a label then such label will be ignored.
	RemoveLabelsFromHost(ctx context.Context, hostID uint, labelIDs []uint) error

	// UpdateLabelMembershipByHostIDs updates the label membership for the given label with host
	// IDs, applied in batches, then returns the updated label
	UpdateLabelMembershipByHostIDs(ctx context.Context, label Label, hostIds []uint, teamFilter TeamFilter) (*Label, []uint, error)
	// UpdateLabelMembershipByHostCriteria updates the label membership for the given label
	// based on its host vitals criteria.
	UpdateLabelMembershipByHostCriteria(ctx context.Context, hvl HostVitalsLabel) (*Label, error)

	NewLabel(ctx context.Context, label *Label, opts ...OptionalArg) (*Label, error)
	// SaveLabel updates the label and returns the label and an array of host IDs
	// members of this label, or an error.
	SaveLabel(ctx context.Context, label *Label, teamFilter TeamFilter) (*LabelWithTeamName, []uint, error)
	DeleteLabel(ctx context.Context, name string, filter TeamFilter) error
	LabelByName(ctx context.Context, name string, filter TeamFilter) (*Label, error)
	// Label returns the label and an array of host IDs members of this label, or an error.
	Label(ctx context.Context, lid uint, teamFilter TeamFilter) (*LabelWithTeamName, []uint, error)
	// LabelMembershipHostIDs returns every host_id row in label_membership for
	// the given label ID, with no team-based filtering. Intended for internal
	// activity tracking where the unfiltered membership is needed.
	LabelMembershipHostIDs(ctx context.Context, labelID uint) ([]uint, error)
	ListLabels(ctx context.Context, filter TeamFilter, opt ListOptions, includeHostCounts bool) ([]*Label, error)
	LabelsSummary(ctx context.Context, filter TeamFilter) ([]*LabelSummary, error)

	GetEnrollmentIDsWithPendingMDMAppleCommands(ctx context.Context) ([]string, error)

	// LabelQueriesForHost returns the (dynamic) label queries that should be executed for the given host.
	// Results are returned in a map of label id -> query
	LabelQueriesForHost(ctx context.Context, host *Host) (map[string]string, error)

	// ListLabelsForHost returns the labels that the given host is in.
	ListLabelsForHost(ctx context.Context, hid uint) ([]*Label, error)

	// ListHostsInLabel returns a slice of hosts in the label with the given ID.
	ListHostsInLabel(ctx context.Context, filter TeamFilter, lid uint, opt HostListOptions) ([]*Host, error)

	SearchLabels(ctx context.Context, filter TeamFilter, query string, omit ...uint) ([]*Label, error)

	// LabelIDsByName retrieves the IDs associated with the given label names
	LabelIDsByName(ctx context.Context, labels []string, filter TeamFilter) (map[string]uint, error)
	// LabelsByName retrieves the labels associated with the given label names
	LabelsByName(ctx context.Context, names []string, filter TeamFilter) (map[string]*Label, error)

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
	ListBatchScriptHosts(ctx context.Context, batchScriptExecutionID string, batchScriptExecutionStatus BatchScriptExecutionStatus, opt ListOptions) ([]BatchScriptHost, *PaginationMetadata, uint, error)

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

	// ListHostUsers returns a list of users that are currently on the host
	ListHostUsers(ctx context.Context, hostID uint) ([]HostUser, error)

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
	// HostIDsByIdentifier retrieves the IDs associated with the given hostnames, UUIDs, hardware serials, node keys or osquery host IDs.
	HostIDsByIdentifier(ctx context.Context, filter TeamFilter, hostnames []string) ([]uint, error)

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
	// osquery_host_id, node_key, UUID, hardware_serial or hostname.
	HostByIdentifier(ctx context.Context, identifier string) (*Host, error)
	// HostLiteByIdentifier returns a host and a subset of its fields using an
	// "identifier" string. The identifier string will be matched against the
	// hostname, osquery_host_id, node_key, uuid and hardware_serial columns.
	HostLiteByIdentifier(ctx context.Context, identifier string) (*HostLite, error)
	// HostLiteByIdentifier returns a host and a subset of its fields from its id.
	HostLiteByID(ctx context.Context, id uint) (*HostLite, error)
	// AddHostsToTeam adds hosts to an existing team, clearing their team settings if params.TeamID is nil.
	AddHostsToTeam(ctx context.Context, params *AddHostsToTeamParams) error
	// HostnamesByIdentifiers returns the hostnames corresponding to the provided identifiers,
	// as understood by HostByIdentifier.
	HostnamesByIdentifiers(ctx context.Context, identifiers []string) ([]string, error)
	// UpdateHostIssuesFailingPolicies updates the failing policies count in host_issues table for the provided hosts.
	UpdateHostIssuesFailingPolicies(ctx context.Context, hostIDs []uint) error
	// UpdateHostIssuesFailingPoliciesForSingleHost updates the failing policies count in host_issues table for a single host.
	UpdateHostIssuesFailingPoliciesForSingleHost(ctx context.Context, hostID uint) error
	// Gets the last time the host's row in `host_issues` was updated
	GetHostIssuesLastUpdated(ctx context.Context, hostId uint) (time.Time, error)
	// UpdateHostIssuesVulnerabilities updates the critical vulnerabilities counts in host_issues.
	UpdateHostIssuesVulnerabilities(ctx context.Context) error
	// CleanupHostIssues deletes host issues that no longer belong to a host.
	CleanupHostIssues(ctx context.Context) error

	TotalAndUnseenHostsSince(ctx context.Context, teamID *uint, daysCount int) (total int, unseen []uint, err error)

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
	// SetOrUpdateIDPHostDeviceMapping creates or updates an IDP device mapping for a host.
	SetOrUpdateIDPHostDeviceMapping(ctx context.Context, hostID uint, email string) error
	// DeleteHostIDP deletes an existing host IDP device mapping.
	DeleteHostIDP(ctx context.Context, id uint) error
	// SetOrUpdateHostSCIMUserMapping associates a host with a SCIM user. If a
	// mapping already exists, it will be updated to the new SCIM user.
	SetOrUpdateHostSCIMUserMapping(ctx context.Context, hostID uint, scimUserID uint) error
	// DeleteHostSCIMUserMapping removes the association between a host and a SCIM user.
	DeleteHostSCIMUserMapping(ctx context.Context, hostID uint) error
	// ListHostBatteries returns the list of batteries for the given host ID.
	ListHostBatteries(ctx context.Context, id uint) ([]*HostBattery, error)
	ListUpcomingHostMaintenanceWindows(ctx context.Context, hid uint) ([]*HostMaintenanceWindow, error)

	// LoadHostByDeviceAuthToken loads the host identified by the device auth token.
	// If the token is invalid or expired it returns a NotFoundError.
	LoadHostByDeviceAuthToken(ctx context.Context, authToken string, tokenTTL time.Duration) (*Host, error)
	// SetOrUpdateDeviceAuthToken inserts or updates the auth token for a host.
	SetOrUpdateDeviceAuthToken(ctx context.Context, hostID uint, authToken string) error
	// GetDeviceAuthToken returns the current auth token for a given host
	GetDeviceAuthToken(ctx context.Context, hostID uint) (string, error)

	// FailingPoliciesCount returns the number of failling policies for 'host'
	FailingPoliciesCount(ctx context.Context, host *Host) (uint, error)

	// ListPoliciesForHost lists the policies that a host will check and whether they are passing
	ListPoliciesForHost(ctx context.Context, host *Host) ([]*HostPolicy, error)

	GetHostMunkiVersion(ctx context.Context, hostID uint) (string, error)
	GetHostMunkiIssues(ctx context.Context, hostID uint) ([]*HostMunkiIssue, error)
	GetHostMDM(ctx context.Context, hostID uint) (*HostMDM, error)
	GetHostMDMCheckinInfo(ctx context.Context, hostUUID string) (*HostMDMCheckinInfo, error)
	// GetHostMDMIdentifiers searches for hosts with identifiers matching the provided identifier.
	// It is intended as an optimization over existing host-by-identifier methods (e.g.,
	// HostLiteByIdentifier) that are prone to full-table scans. See the implementation for more details.
	GetHostMDMIdentifiers(ctx context.Context, identifer string, teamFilter TeamFilter) ([]*HostMDMIdentifiers, error)

	// ListIOSAndIPadOSToRefetch returns the UUIDs of iPhones/iPads that should be refetched (their details haven't been
	// updated in the given `interval`).
	ListIOSAndIPadOSToRefetch(ctx context.Context, refetchInterval time.Duration) (devices []AppleDevicesToRefetch, err error)
	// AddHostMDMCommands adds the provided MDM commands to the host to track which commands have been sent.
	AddHostMDMCommands(ctx context.Context, commands []HostMDMCommand) error
	// GetHostMDMCommands returns the MDM commands that have been sent to the host.
	GetHostMDMCommands(ctx context.Context, hostID uint) (commands []HostMDMCommand, err error)
	// RemoveHostMDMCommand removes the provided MDM command from the host, indicating that it has been processed.
	RemoveHostMDMCommand(ctx context.Context, command HostMDMCommand) error
	// CleanupHostMDMCommands removes invalid and stale MDM commands sent to hosts.
	CleanupHostMDMCommands(ctx context.Context) error
	// CleanupHostMDMAppleProfiles removes abandoned host MDM Apple profiles entries.
	CleanupHostMDMAppleProfiles(ctx context.Context) error
	// CleanupWindowsMDMCommandQueue removes ACKed entries from the Windows MDM command queue
	// whose corresponding result is older than 1 hour.
	CleanupWindowsMDMCommandQueue(ctx context.Context) error
	// CleanupAllHostMDMProfilesForPlatform deletes all host MDM profile rows for the given platform.
	// Used when MDM is toggled off globally to prevent stale pending profiles from persisting.
	CleanupAllHostMDMProfilesForPlatform(ctx context.Context, platform string) error

	// CleanupStaleNanoRefetchCommands deletes up to 3 nano_enrollment_queue and
	// their corresponding nano_command_results entries for the given enrollment ID
	// and REFETCH command prefix type that were sent and acknowledged/errored at
	// least 30 days ago. The current command UUID is excluded from deletion.
	CleanupStaleNanoRefetchCommands(ctx context.Context, enrollmentID string, commandUUIDPrefix string, currentCommandUUID string) error

	// CleanupOrphanedNanoRefetchCommands deletes up to 100 REFETCH-prefixed nano_commands
	// older than 30 days that have no remaining references in nano_enrollment_queue.
	CleanupOrphanedNanoRefetchCommands(ctx context.Context) error

	// IsHostConnectedToFleetMDM verifies if the host has an active Fleet MDM enrollment with this server
	IsHostConnectedToFleetMDM(ctx context.Context, host *Host) (bool, error)

	ListHostCertificates(ctx context.Context, hostID uint, opts ListOptions) ([]*HostCertificateRecord, *PaginationMetadata, error)
	// UpdateHostCertificates ingests certs reported by `origin`. Each call only
	// soft-deletes existing rows whose origin matches, so osquery and MDM
	// ingestion don't clobber each other's view.
	UpdateHostCertificates(ctx context.Context, hostID uint, hostUUID string, certs []*HostCertificateRecord, origin HostCertificateOrigin) error

	// ProfileHasACMEPayloadForCommand returns the host/profile gating data
	// needed to decide whether an InstallProfile ack should trigger a
	// CertificateList refetch: host platform, profile UUID, whether the
	// delivered profile contains a com.apple.security.acme payload, and
	// whether a refetch is already pending. All gates are computed
	// server-side in a single indexed lookup so the per-ack hot path stays
	// cheap. Substring-matched on the mobileconfig blob; bounded false-
	// positive risk (one redundant CertificateList per false match).
	ProfileHasACMEPayloadForCommand(ctx context.Context, hostUUID, commandUUID string) (ProfileACMECommandResult, error)

	// AreHostsConnectedToFleetMDM checks each host MDM enrollment with
	// this server and returns a map indexed by the host uuid and a boolean
	// indicating if the enrollment is active.
	//
	// This function exists to prevent n+1 queries when we need to check
	// the MDM status of a list of hosts.
	AreHostsConnectedToFleetMDM(ctx context.Context, hosts []*Host) (map[string]bool, error)

	AggregatedMunkiVersion(ctx context.Context, teamID *uint) ([]AggregatedMunkiVersion, time.Time, error)
	AggregatedMunkiIssues(ctx context.Context, teamID *uint) ([]AggregatedMunkiIssue, time.Time, error)
	AggregatedMDMStatus(ctx context.Context, teamID *uint, platform string) (AggregatedMDMStatus, time.Time, error)
	AggregatedMDMSolutions(ctx context.Context, teamID *uint, platform string) ([]AggregatedMDMSolutions, time.Time, error)
	GenerateAggregatedMunkiAndMDM(ctx context.Context) error

	GetMunkiIssue(ctx context.Context, munkiIssueID uint) (*MunkiIssue, error)
	GetMDMSolution(ctx context.Context, mdmID uint) (*MDMSolution, error)

	OSVersions(ctx context.Context, teamFilter *TeamFilter, platform *string, name *string, version *string) (*OSVersions, error)
	OSVersionsByCVE(ctx context.Context, cve string, teamID *uint) ([]*VulnerableOS, time.Time, error)
	SoftwareByCVE(ctx context.Context, cve string, teamID *uint) ([]*VulnerableSoftware, time.Time, error)
	// OSVersion returns the OSVersion with the provided ID. If teamFilter is not nil, then the OSVersion is filtered.
	// The returned OSVersion is accompanied by the time it was last updated.
	OSVersion(ctx context.Context, osVersionID uint, teamFilter *TeamFilter) (*OSVersion, *time.Time, error)
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

	// NewSession creates a new session for the given user and stores it
	NewSession(ctx context.Context, userID uint, sessionKeySize int) (*Session, error)

	// DestroySession destroys the currently tracked session
	DestroySession(ctx context.Context, session *Session) error

	// DestroyAllSessionsForUser destroys all of the sessions for a given user
	DestroyAllSessionsForUser(ctx context.Context, id uint) error

	// MarkSessionAccessed marks the currently tracked session as access to extend expiration
	MarkSessionAccessed(ctx context.Context, session *Session) error

	// SessionByMFAToken redeems an MFA token for a session, and returns the associated user, if that MFA token is valid
	SessionByMFAToken(ctx context.Context, token string, sessionKeySize int) (*Session, *User, error)

	// NewMFAToken creates a new MFA token for a given user and stores it
	NewMFAToken(ctx context.Context, userID uint) (string, error)

	///////////////////////////////////////////////////////////////////////////////
	// AppConfigStore contains method for saving and retrieving application configuration

	NewAppConfig(ctx context.Context, info *AppConfig) (*AppConfig, error)
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

	// Methods for getting and applying the stored yara rules.
	GetYaraRules(ctx context.Context) ([]YaraRule, error)
	ApplyYaraRules(ctx context.Context, rules []YaraRule) error
	YaraRuleByName(ctx context.Context, name string) (*YaraRule, error)

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
	CleanupExpiredHosts(ctx context.Context) ([]DeletedHostDetails, error)
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
	OverwriteQueryResultRows(ctx context.Context, rows []*ScheduledQueryResultRow, maxQueryReportRows int) (int, error)
	// CleanupDiscardedQueryResults deletes all query results for queries with DiscardData enabled.
	// Used in cleanups_then_aggregation cron to cleanup rows that were inserted immediately
	// after DiscardData was set to true due to query caching.
	CleanupDiscardedQueryResults(ctx context.Context) error
	// CleanupExcessQueryResultRows deletes query result rows that exceed the maximum allowed per query.
	// It keeps the most recent rows (by id, which correlates with insert order) up to the limit.
	// Deletes are batched to avoid large binlogs and long lock times. This runs as a cron job.
	// Returns a map of query IDs to their current row count after cleanup (for syncing Redis counters).
	CleanupExcessQueryResultRows(ctx context.Context, maxQueryReportRows int, opts ...CleanupExcessQueryResultRowsOptions) (map[uint]int, error)
	// ListHostReports returns the queries/reports associated with the given host, applying
	// the provided options for filtering, sorting, and pagination. teamID is the team of the
	// host (nil for global). maxQueryReportRows is the configured report cap used to determine
	// whether each query's report has been clipped. It returns the list of reports, the total
	// count (without pagination), optional pagination metadata, and any error.
	ListHostReports(ctx context.Context, hostID uint, teamID *uint, hostPlatform string, opts ListHostReportsOptions, maxQueryReportRows int) ([]*HostReport, int, *PaginationMetadata, error)

	///////////////////////////////////////////////////////////////////////////////
	// TeamStore

	// NewTeam creates a new Team object in the store.
	NewTeam(ctx context.Context, team *Team) (*Team, error)
	// SaveTeam saves any changes to the team.
	SaveTeam(ctx context.Context, team *Team) (*Team, error)
	// TeamWithExtras retrieves the Team by ID, including extra fields.
	TeamWithExtras(ctx context.Context, tid uint) (*Team, error)
	// TeamLite retrieves a Team by ID, including only id, created_at, name, filename, description, config fields.
	TeamLite(ctx context.Context, tid uint) (*TeamLite, error)
	// DeleteTeam deletes the Team by ID.
	DeleteTeam(ctx context.Context, tid uint) error
	// TeamByName retrieves the Team by Name (including extras).
	TeamByName(ctx context.Context, name string) (*Team, error)
	// TeamByFilename retrieves the Team by GitOps filename.
	TeamByFilename(ctx context.Context, filename string) (*Team, error)
	// TeamConflictsWithName returns a team whose collation-equal name conflicts
	// with the provided name and whose id != excludeID, or (nil, nil) when
	// no such team exists. Pass excludeID=0 to check against all teams.
	TeamConflictsWithName(ctx context.Context, name string, excludeID uint) (*Team, error)
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
	// DefaultTeamConfig returns the configuration for "No Team" hosts.
	DefaultTeamConfig(ctx context.Context) (*TeamConfig, error)
	// SaveDefaultTeamConfig saves the configuration for "No Team" hosts.
	SaveDefaultTeamConfig(ctx context.Context, config *TeamConfig) error
	// TeamExists returns true if a team with the given id exists.
	TeamExists(ctx context.Context, teamID uint) (bool, error)

	///////////////////////////////////////////////////////////////////////////////
	// Software Titles

	ListSoftwareTitles(ctx context.Context, opt SoftwareTitleListOptions, tmFilter TeamFilter) ([]SoftwareTitleListResult, int, *PaginationMetadata, error)
	SoftwareTitleByID(ctx context.Context, id uint, teamID *uint, tmFilter TeamFilter) (*SoftwareTitle, error)
	SoftwareTitleNameForHostFilter(ctx context.Context, id uint) (name, displayName string, err error)
	UpdateSoftwareTitleName(ctx context.Context, id uint, name string) error
	UpdateSoftwareTitleAutoUpdateConfig(ctx context.Context, titleID uint, teamID uint, config SoftwareAutoUpdateConfig) error
	ListSoftwareAutoUpdateSchedules(ctx context.Context, teamID uint, source string, optionalFilter ...SoftwareAutoUpdateScheduleFilter) ([]SoftwareAutoUpdateSchedule, error)

	// InsertSoftwareInstallRequest tracks a new request to install the provided
	// software installer in the host. It returns the auto-generated installation
	// uuid.
	InsertSoftwareInstallRequest(ctx context.Context, hostID uint, softwareInstallerID uint, opts HostSoftwareInstallOptions) (string, error)
	// InsertSoftwareUninstallRequest tracks a new request to uninstall the provided
	// software installer on the host. executionID is the script execution ID corresponding to uninstall script
	InsertSoftwareUninstallRequest(ctx context.Context, executionID string, hostID uint, softwareInstallerID uint, selfService bool) error
	// GetDetailsForUninstallFromExecutionID returns details from a software uninstall execution needed to create the corresponding activity
	// Non-error returns are software title name and whether the uninstall was self-service, respectively
	GetDetailsForUninstallFromExecutionID(ctx context.Context, executionID string) (string, bool, error)

	///////////////////////////////////////////////////////////////////////////////
	// SoftwareStore

	// ListSoftwareForVulnDetection returns all software for the given hostID with only the fields
	// used for vulnerability detection populated (id, name, version, cpe_id, cpe)
	ListSoftwareForVulnDetection(ctx context.Context, filter VulnSoftwareFilter) ([]Software, error)
	// ListSoftwareForVulnDetectionByOSVersion returns all distinct software installed on hosts
	// matching the given OS version.
	ListSoftwareForVulnDetectionByOSVersion(ctx context.Context, osVer OSVersion) ([]Software, error)
	ListSoftwareVulnerabilitiesByHostIDsSource(ctx context.Context, hostIDs []uint, source VulnerabilitySource) (map[uint][]SoftwareVulnerability, error)
	// ListSoftwareVulnerabilitiesBySoftwareIDs returns vulnerabilities for the given software IDs
	// filtered by source. Queries software_cve directly without joining through host_software.
	ListSoftwareVulnerabilitiesBySoftwareIDs(ctx context.Context, softwareIDs []uint, source VulnerabilitySource) ([]SoftwareVulnerability, error)
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
	// InsertSoftwareVulnerabilities inserts a batch of vulnerabilities into the datastore.
	// It checks which vulnerabilities are new (not already present) before inserting, and
	// returns only the newly inserted vulnerabilities.
	InsertSoftwareVulnerabilities(ctx context.Context, vulns []SoftwareVulnerability, source VulnerabilitySource) ([]SoftwareVulnerability, error)
	SoftwareByID(ctx context.Context, id uint, teamID *uint, includeCVEScores bool, tmFilter *TeamFilter) (*Software, error)
	// SoftwareLiteByID returns the name and version
	// of a software entry by ID without applying fleet(team)-scoped filtering.
	// Intentionally allows callers to discover the software name and version
	// even if the software is not present on their team.
	//
	// Only use for use cases where exposing the existence of a software version is acceptable.
	SoftwareLiteByID(ctx context.Context, id uint) (SoftwareLite, error)
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

	// CleanupSoftwareTitles cleans up any software titles (software_titles table)
	// that are no longer associated with any software version (software table).
	//
	// It is intended to be run after SyncHostsSoftware.
	CleanupSoftwareTitles(ctx context.Context) error

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

	ListHostSoftware(ctx context.Context, host *Host, opts HostSoftwareTitleListOptions) ([]*HostSoftwareWithInstaller, *PaginationMetadata, error)

	// IsSoftwareInstallerLabelScoped returns whether or not the given installerID is scoped to the
	// given host ID by labels.
	IsSoftwareInstallerLabelScoped(ctx context.Context, installerID, hostID uint) (bool, error)

	// IsVPPAppLabelScoped returns whether or not the given vppAppTeamID is scoped to the given hostID by labels.
	IsVPPAppLabelScoped(ctx context.Context, vppAppTeamID, hostID uint) (bool, error)

	// IsInHouseAppLabelScoped returns whether or not the given inHouseAppID is scoped to the given hostID by labels.
	IsInHouseAppLabelScoped(ctx context.Context, inHouseAppID, hostID uint) (bool, error)

	GetUnverifiedInHouseAppInstallsForHost(ctx context.Context, hostUUID string) ([]*HostVPPSoftwareInstall, error)
	SetInHouseAppInstallAsVerified(ctx context.Context, hostID uint, installUUID, verificationUUID string) error
	SetInHouseAppInstallAsFailed(ctx context.Context, hostID uint, installUUID, verificationUUID string) error
	ReplaceInHouseAppInstallVerificationUUID(ctx context.Context, oldVerifyUUID, verifyCommandUUID string) error
	GetPastActivityDataForInHouseAppInstall(ctx context.Context, commandResults *mdm.CommandResults) (*User, *ActivityTypeInstalledSoftware, error)

	// SetHostSoftwareInstallResult records the result of a software installation
	SetHostSoftwareInstallResult(ctx context.Context, result *HostSoftwareInstallResultPayload, attemptNumber *int) (wasCanceled bool, err error)

	// CreateIntermediateInstallFailureRecord creates a completed failure record for an
	// installation attempt that will be retried, while keeping the original pending.
	// It returns a deterministic execution ID for the failure record (unique per base
	// install UUID and retry ordinal) to support idempotency. This method is for
	// persistence/bookkeeping only and must not be used to trigger user-visible side effects.
	CreateIntermediateInstallFailureRecord(ctx context.Context, result *HostSoftwareInstallResultPayload) (string, error)

	// NewSoftwareCategory creates a new category for software.
	NewSoftwareCategory(ctx context.Context, name string) (*SoftwareCategory, error)
	// GetSoftwareCategoryIDs the list of IDs that correspond to the given list of software category names.
	GetSoftwareCategoryIDs(ctx context.Context, names []string) ([]uint, error)
	// GetSoftwareCategoryNameToIDMap returns a map of software category names to their IDs for the given names.
	// Only categories that exist in the database are included in the map.
	GetSoftwareCategoryNameToIDMap(ctx context.Context, names []string) (map[string]uint, error)
	// GetCategoriesForSoftwareTitles takes a set of software title IDs and returns a map
	// from the title IDs to the categories assigned to the installers for those titles.
	GetCategoriesForSoftwareTitles(ctx context.Context, softwareTitleIDs []uint, team_id *uint) (map[uint][]string, error)

	// AssociateMDMInstallToVerificationUUID updates the verification command UUID associated with the
	// given install attempt (InstallApplication command).
	// It will attempt to update both VPP and in-house app installs (only one will succeed since the command UUIDs are unique).
	AssociateMDMInstallToVerificationUUID(ctx context.Context, installUUID, verifyCommandUUID, hostUUID string) error
	// SetVPPInstallAsVerified marks the VPP app install attempt as "verified" (Fleet has validated
	// that it's installed on the device).
	SetVPPInstallAsVerified(ctx context.Context, hostID uint, installUUID, verificationUUID string) error
	// ReplaceVPPInstallVerificationUUID replaces the verification command UUID for all
	// VPP app install attempts were related to oldVerifyUUID.
	ReplaceVPPInstallVerificationUUID(ctx context.Context, oldVerifyUUID, verifyCommandUUID string) error
	// IsHostPendingMDMInstallVerification checks if a host has a pending VPP or in-house install verification command.
	IsHostPendingMDMInstallVerification(ctx context.Context, hostUUID string) (bool, error)
	// GetUnverifiedVPPInstallsForHost gets unverified HostVPPSoftwareInstalls by host UUID.
	GetUnverifiedVPPInstallsForHost(ctx context.Context, verificationUUID string) ([]*HostVPPSoftwareInstall, error)
	// SetVPPInstallAsFailed marks a VPP app install attempt as failed (Fleet couldn't validate that
	// it was installed on the host).
	SetVPPInstallAsFailed(ctx context.Context, hostID uint, installUUID, verificationUUID string) error
	MarkAllPendingAppleVPPAndInHouseInstallsAsFailed(ctx context.Context, jobName string) error

	CheckConflictingInstallerExists(ctx context.Context, teamID *uint, bundleIdentifier, platform string) (bool, error)
	CheckConflictingInHouseAppExists(ctx context.Context, teamID *uint, bundleIdentifier, platform string) (bool, error)

	// CheckAndroidWebAppNameExistsOnTeam checks if a different Android web app
	// with the given name already exists on the specified team (via vpp_apps_teams + vpp_apps).
	// The excludeAdamID param excludes the app being added/updated from the check.
	CheckAndroidWebAppNameExistsOnTeam(ctx context.Context, teamID *uint, name string, excludeAdamID string) (bool, error)

	///////////////////////////////////////////////////////////////////////////////
	// OperatingSystemsStore

	// GetHostOperatingSystem returns the operating system information
	// for a given host.
	GetHostOperatingSystem(ctx context.Context, hostID uint) (*OperatingSystem, error)
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

	// MDMTurnOff updates Fleet host information related to MDM when a host turns
	// off MDM. Anything related to the protocol itself is managed separately. It
	// returns the users and corresponding activities that may need to be created
	// as a result of turning off MDM.
	MDMTurnOff(ctx context.Context, uuid string) (users []*User, activities []ActivityDetails, err error)

	///////////////////////////////////////////////////////////////////////////////
	// ActivitiesStore

	ListHostUpcomingActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*UpcomingActivity, *PaginationMetadata, error)
	CancelHostUpcomingActivity(ctx context.Context, hostID uint, executionID string) (ActivityDetails, error)
	BatchCancelAllHostUpcomingActivities(ctx context.Context, hostID uint) ([]ActivityDetails, error)
	IsExecutionPendingForHost(ctx context.Context, hostID uint, scriptID uint) (bool, error)
	GetHostUpcomingActivityMeta(ctx context.Context, hostID uint, executionID string) (*UpcomingActivityMeta, error)
	UnblockHostsUpcomingActivityQueue(ctx context.Context, maxHosts int) (int, error)
	// ActivateNextUpcomingActivityForHost activates the next upcoming activity for the given host.
	// fromCompletedExecID is the execution ID of the activity that just completed (if any).
	ActivateNextUpcomingActivityForHost(ctx context.Context, hostID uint, fromCompletedExecID string) error

	///////////////////////////////////////////////////////////////////////////////
	// StatisticsStore

	ShouldSendStatistics(ctx context.Context, frequency time.Duration, config config.FleetConfig) (StatisticsPayload, bool, error)
	RecordStatisticsSent(ctx context.Context) error
	// CleanupStatistics executes cleanup tasks to be performed upon successful transmission of
	// statistics.
	CleanupStatistics(ctx context.Context) error
	// GetTableRowCounts returns approximate DB row counts for all tables in a map indexed by table name
	GetTableRowCounts(ctx context.Context) (map[string]uint, error)

	///////////////////////////////////////////////////////////////////////////////
	// GlobalPoliciesStore

	// ApplyPolicySpecs applies a list of policies (likely from a yaml file) to the datastore. Existing policies are updated,
	// and new policies are created.
	ApplyPolicySpecs(ctx context.Context, authorID uint, specs []*PolicySpec) error

	NewGlobalPolicy(ctx context.Context, authorID *uint, args PolicyPayload) (*Policy, error)
	Policy(ctx context.Context, id uint) (*Policy, error)
	PolicyLite(ctx context.Context, id uint) (*PolicyLite, error)

	// SavePolicy updates some fields of the given policy on the datastore.
	//
	// It is also used to update team policies.
	SavePolicy(ctx context.Context, p *Policy, shouldRemoveAllPolicyMemberships bool, removePolicyStats bool) error

	ListGlobalPolicies(ctx context.Context, opts ListOptions) ([]*Policy, error)
	PoliciesByID(ctx context.Context, ids []uint) (map[uint]*Policy, error)
	DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error)
	CountPolicies(ctx context.Context, teamID *uint, matchQuery string, automationType string) (int, error)
	CountMergedTeamPolicies(ctx context.Context, teamID uint, matchQuery string, automationType string) (int, error)
	UpdateHostPolicyCounts(ctx context.Context) error

	PolicyQueriesForHost(ctx context.Context, host *Host) (map[string]string, error)

	// GetTeamHostsPolicyMemberships returns the hosts that belong to the given team and their pass/fail statuses
	// around the provided policyIDs.
	// 	- Returns hosts of the team that are failing one or more of the provided policies.
	//	- Returns hosts of the team that are passing all the policies (or are not running any of the provided policies)
	//	  and have a calendar event scheduled.
	GetTeamHostsPolicyMemberships(ctx context.Context, domain string, teamID uint, policyIDs []uint,
		hostID *uint) ([]HostPolicyMembershipData, error)
	// GetPoliciesWithAssociatedInstaller returns team policies that have an associated installer.
	GetPoliciesWithAssociatedInstaller(ctx context.Context, teamID uint, policyIDs []uint) ([]PolicySoftwareInstallerData, error)
	// GetPoliciesWithAssociatedVPP returns team policies that have an associated VPP app
	GetPoliciesWithAssociatedVPP(ctx context.Context, teamID uint, policyIDs []uint) ([]PolicyVPPData, error)
	GetPoliciesWithAssociatedScript(ctx context.Context, teamID uint, policyIDs []uint) ([]PolicyScriptData, error)
	GetCalendarPolicies(ctx context.Context, teamID uint) ([]PolicyCalendarData, error)
	// GetPoliciesForConditionalAccess returns the team policies that are configured for "Conditional access".
	GetPoliciesForConditionalAccess(ctx context.Context, teamID uint, platform string) ([]uint, error)
	// GetPatchPolicy returns the patch policy associated with the title id
	GetPatchPolicy(ctx context.Context, teamID *uint, titleID uint) (*PatchPolicyData, error)

	// ConditionalAccessBypassDevice lets the host skip the conditional access check next time it fails
	ConditionalAccessBypassDevice(ctx context.Context, hostID uint) error
	// ConditionalAccessConsumeBypass consumes the bypass checks and consumes any conditional access
	// bypass a device has. If a bypass is present, it will return the time the bypass was enabled.
	// If a bypass is not present, it will return nil.
	ConditionalAccessConsumeBypass(ctx context.Context, hostID uint) (*time.Time, error)
	// ConditionalAccessClearBypasses clears all conditional access bypasses from the database
	ConditionalAccessClearBypasses(ctx context.Context) error
	// ConditionalAccessBypassedAt returns the time the bypass was enabled for a host, or nil if no bypass exists
	ConditionalAccessBypassedAt(ctx context.Context, hostID uint) (*time.Time, error)

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
	// DeleteSoftwareVulnerabilities deletes the given list of vulnerabilities identified by CPE+CVE.
	DeleteSoftwareVulnerabilities(ctx context.Context, vulnerabilities []SoftwareVulnerability) error
	// DeleteOutOfDateVulnerabilities deletes 'software_cve' entries from the provided source where
	// the updated_at timestamp is older than the provided timestamp
	DeleteOutOfDateVulnerabilities(ctx context.Context, source VulnerabilitySource, olderThan time.Time) error
	// DeleteOrphanedSoftwareVulnerabilities deletes 'software_cve' entries where the software_id
	// no longer has any associated hosts in 'host_software'.
	DeleteOrphanedSoftwareVulnerabilities(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// Calendar events

	CreateOrUpdateCalendarEvent(ctx context.Context, uuid string, email string, startTime time.Time, endTime time.Time, data []byte,
		timeZone *string, hostID uint, webhookStatus CalendarWebhookStatus) (*CalendarEvent, error)
	GetCalendarEvent(ctx context.Context, email string) (*CalendarEvent, error)
	GetCalendarEventDetailsByUUID(ctx context.Context, uuid string) (*CalendarEventDetails, error)
	DeleteCalendarEvent(ctx context.Context, calendarEventID uint) error
	UpdateCalendarEvent(ctx context.Context, calendarEventID uint, uuid string, startTime time.Time, endTime time.Time, data []byte,
		timeZone *string) error
	GetHostCalendarEvent(ctx context.Context, hostID uint) (*HostCalendarEvent, *CalendarEvent, error)
	GetHostCalendarEventByEmail(ctx context.Context, email string) (*HostCalendarEvent, *CalendarEvent, error)
	UpdateHostCalendarWebhookStatus(ctx context.Context, hostID uint, status CalendarWebhookStatus) error
	ListCalendarEvents(ctx context.Context, teamID *uint) ([]*CalendarEvent, error)
	ListOutOfDateCalendarEvents(ctx context.Context, t time.Time) ([]*CalendarEvent, error)

	///////////////////////////////////////////////////////////////////////////////
	// Team Policies

	NewTeamPolicy(ctx context.Context, teamID uint, authorID *uint, args PolicyPayload) (*Policy, error)
	ListTeamPolicies(ctx context.Context, teamID uint, opts ListOptions, iopts ListOptions, automationType string) (teamPolicies, inheritedPolicies []*Policy, err error)
	ListMergedTeamPolicies(ctx context.Context, teamID uint, opts ListOptions, automationType string) ([]*Policy, error)

	DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error)
	TeamPolicy(ctx context.Context, teamID uint, policyID uint) (*Policy, error)

	CleanupPolicyMembership(ctx context.Context, now time.Time) error
	// IsPolicyFailing checks if a policy is currently failing for a given host.
	IsPolicyFailing(ctx context.Context, policyID, hostID uint) (bool, error)
	// CountHostSoftwareInstallAttempts counts how many install attempts exist for a specific
	// host, software installer, and policy combination. Used to calculate attempt_number.
	CountHostSoftwareInstallAttempts(ctx context.Context, hostID, softwareInstallerID, policyID uint) (int, error)
	// ResetNonPolicyInstallAttempts resets the attempt_number for all non-policy install attempts
	// for a given host and software installer so that a new install starts fresh.
	ResetNonPolicyInstallAttempts(ctx context.Context, hostID, softwareInstallerID uint) error
	// CountHostScriptAttempts counts how many script execution attempts exist for a specific
	// host, script, and policy combination. Used to calculate attempt_number.
	CountHostScriptAttempts(ctx context.Context, hostID, scriptID, policyID uint) (int, error)
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
	UpdateCronStats(ctx context.Context, id int, status CronStatsStatus, cronErrors *CronScheduleErrors) error
	// ClaimCronStats transitions a queued cron stats record to the given status
	// and updates the instance to the worker that claimed it.
	ClaimCronStats(ctx context.Context, id int, instance string, status CronStatsStatus) error
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
	// 'installed_path\0team_identifier\0software.ToUniqueStr()' strings. 'mutationResults' contains the software inventory of
	// the host (pre-mutations) and the mutations performed after calling 'UpdateHostSoftware',
	// it is used as DB optimization.
	//
	// TODO(lucas): We should amend UpdateHostSoftwareInstalledPaths to just accept raw information
	// otherwise the caller has to assemble the reported set the same way in all places where it's used.
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
	// If newlyPassingPolicyIDs is non-nil, it contains the IDs of policies that flipped from failing to passing
	// and is used directly instead of calling FlippingPoliciesForHost internally. This allows callers that have
	// already computed flipping policies to avoid a redundant database query.
	RecordPolicyQueryExecutions(ctx context.Context, host *Host, results map[uint]*bool, updated time.Time, deferredSaveHost bool, newlyPassingPolicyIDs []uint) error

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
	SetOrUpdateMDMData(ctx context.Context, hostID uint, isServer, enrolled bool, serverURL string, installedFromDep bool, name string, fleetEnrollRef string, isPersonalEnrollment bool) error
	// UpdateMDMData updates the `enrolled` field of the host with the given ID.
	UpdateMDMData(ctx context.Context, hostID uint, enrolled bool) error
	// UpdateMDMInstalledFromDEP updates the `installed_from_dep` field of the host with the given ID.
	UpdateMDMInstalledFromDEP(ctx context.Context, hostID uint, installedFromDep bool) error
	// GetHostEmails returns the emails associated with the provided host for a given source, such as "google_chrome_profiles"
	GetHostEmails(ctx context.Context, hostUUID string, source string) ([]string, error)
	// SetOrUpdateHostDisksSpace sets or updates the gigs_total_disk_space and gigs_all_disk_space
	// fields for a host. gigs_all_disk_space should should only be non-nil for Linux hosts
	SetOrUpdateHostDisksSpace(ctx context.Context, hostID uint, gigsAvailable, percentAvailable, gigsTotal float64, gigsAll *float64) error

	GetConfigEnableDiskEncryption(ctx context.Context, teamID *uint) (DiskEncryptionConfig, error)
	SetOrUpdateHostDiskTpmPIN(ctx context.Context, hostID uint, pinSet bool) error
	SetOrUpdateHostDisksEncryption(ctx context.Context, hostID uint, encrypted bool, bitlockerProtectionStatus *int) error
	// SetOrUpdateHostDiskEncryptionKey sets the base64, encrypted key for
	// a host, returns whether the current key was archived or not due to the current one being updated/replaced.
	SetOrUpdateHostDiskEncryptionKey(ctx context.Context, host *Host, encryptedBase64Key, clientError string, decryptable *bool) (bool, error)
	// SaveLUKSData sets base64'd encrypted LUKS passphrase, key slot, and salt data for a host that has successfully
	// escrowed LUKS data, returns whether the current key was archived or not due to the current one being
	// updated/replaced.
	SaveLUKSData(ctx context.Context, host *Host, encryptedBase64Passphrase string, encryptedBase64Salt string, keySlot uint) (bool, error)
	// DeleteLUKSData deletes the LUKS encryption key associated with the provided host ID and key slot.
	DeleteLUKSData(ctx context.Context, hostID, keySlot uint) error

	// GetUnverifiedDiskEncryptionKeys returns all the encryption keys that
	// are collected but their decryptable status is not known yet (ie:
	// we're able to decrypt the key using a private key in the server)
	GetUnverifiedDiskEncryptionKeys(ctx context.Context) ([]HostDiskEncryptionKey, error)
	// SetHostsDiskEncryptionKeyStatus sets the encryptable status for the set
	// of encription keys provided
	SetHostsDiskEncryptionKeyStatus(ctx context.Context, hostIDs []uint, decryptable bool, threshold time.Time) error
	// GetHostDiskEncryptionKey returns the encryption key information for a given host
	GetHostDiskEncryptionKey(ctx context.Context, hostID uint) (*HostDiskEncryptionKey, error)
	// GetHostArchivedDiskEncryptionKey returns the archived disk encryption key for the given host ID.
	GetHostArchivedDiskEncryptionKey(ctx context.Context, host *Host) (*HostArchivedDiskEncryptionKey, error)
	// IsHostDiskEncryptionKeyArchived returns true if there is a disk encryption key archived
	// for the given host ID.
	IsHostDiskEncryptionKeyArchived(ctx context.Context, hostID uint) (bool, error)
	IsHostPendingEscrow(ctx context.Context, hostID uint) bool
	ClearPendingEscrow(ctx context.Context, hostID uint) error
	ReportEscrowError(ctx context.Context, hostID uint, err string) error
	QueueEscrow(ctx context.Context, hostID uint) error
	AssertHasNoEncryptionKeyStored(ctx context.Context, hostID uint) error

	// GetHostCertAssociationsToExpire retrieves host certificate
	// associations that are close to expire and don't have a renewal in
	// progress based on the provided arguments.
	GetHostCertAssociationsToExpire(ctx context.Context, expiryDays, limit int) ([]SCEPIdentityAssociation, error)

	// GetDeviceInfoForACMERenewal retrieves the device information for ACMERenewal based on the provided host UUIDs.
	GetDeviceInfoForACMERenewal(ctx context.Context, hostUUIDs []string) ([]DeviceInfoForACMERenewal, error)

	// SetCommandForPendingSCEPRenewal tracks the command used to renew a scep certificate
	SetCommandForPendingSCEPRenewal(ctx context.Context, assocs []SCEPIdentityAssociation, cmdUUID string) error

	// CleanSCEPRenewRefs cleans all references after a successful SCEP renewal.
	CleanSCEPRenewRefs(ctx context.Context, hostUUID string) error

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
	SetOrUpdateHostOrbitInfo(
		ctx context.Context, hostID uint, version string, desktopVersion sql.NullString, scriptsEnabled sql.NullBool,
	) error

	GetHostOrbitInfo(ctx context.Context, hostID uint) (*HostOrbitInfo, error)

	ReplaceHostDeviceMapping(ctx context.Context, id uint, mappings []*HostDeviceMapping, source string) error

	// ReplaceHostBatteries creates or updates the battery mappings of a host.
	ReplaceHostBatteries(ctx context.Context, id uint, mappings []*HostBattery) error

	// VerifyEnrollSecret checks that the provided secret matches an active enroll secret. If it is successfully
	// matched, that secret is returned. Otherwise, a NotFoundError is returned.
	VerifyEnrollSecret(ctx context.Context, secret string) (*EnrollSecret, error)

	// IsEnrollSecretAvailable checks if the provided secret is available for enrollment.
	IsEnrollSecretAvailable(ctx context.Context, secret string, isNew bool, teamID *uint) (bool, error)

	// EnrollOsquery will enroll a new host with the given identifier, setting the node key, and team. Implementations of
	// this method should respect the provided host enrollment cooldown, by returning an error if the host has enrolled
	// within the cooldown period.
	EnrollOsquery(ctx context.Context, opts ...DatastoreEnrollOsqueryOption) (*Host, error)

	// EnrollOrbit will enroll a new orbit instance.
	//	- If an entry for the host exists (osquery enrolled first) then it will update the host's orbit node key and team.
	//	- If an entry for the host doesn't exist (osquery enrolls later) then it will create a new entry in the hosts table.
	EnrollOrbit(ctx context.Context, opts ...DatastoreEnrollOrbitOption) (*Host, error)

	SerialUpdateHost(ctx context.Context, host *Host) error

	///////////////////////////////////////////////////////////////////////////////
	// JobStore

	// NewJob inserts a new job into the jobs table (queue).
	NewJob(ctx context.Context, job *Job) (*Job, error)

	// GetQueuedJobs gets queued jobs from the jobs table (queue) ready to be
	// processed. If now is the zero time, the current time will be used.
	GetQueuedJobs(ctx context.Context, maxNumJobs int, now time.Time) ([]*Job, error)

	// GetFilteredQueuedJobs gets queued jobs from the jobs table (queue) ready to be
	// processed, filtered by job names.
	GetFilteredQueuedJobs(ctx context.Context, maxNumJobs int, now time.Time, jobNames []string) ([]*Job, error)

	// UpdateJobs updates an existing job. Call this after processing a job.
	UpdateJob(ctx context.Context, id uint, job *Job) (*Job, error)

	// CleanupWorkerJobs deletes jobs in a final state that are older than the
	// provided durations. It returns the number of jobs deleted and an error.
	CleanupWorkerJobs(ctx context.Context, failedSince, completedSince time.Duration) (int64, error)

	// GetJob returns a job from the database
	GetJob(ctx context.Context, jobID uint) (*Job, error)

	// HasQueuedJobWithArgs reports whether a job with the given name and
	// args (compared as JSON values) currently exists in the jobs table in
	// state JobStateQueued. Used by callers that need at-most-one pending
	// job per (name, args) tuple — e.g. dedup of historical-data scrub
	// enqueues across rapid disable/enable toggles.
	HasQueuedJobWithArgs(ctx context.Context, name string, args json.RawMessage) (bool, error)

	///////////////////////////////////////////////////////////////////////////////
	// Debug

	InnoDBStatus(ctx context.Context) (string, error)
	ProcessList(ctx context.Context) ([]MySQLProcess, error)

	///////////////////////////////////////////////////////////////////////////////
	// OperatingSystemVulnerabilities Store
	ListOSVulnerabilitiesByOS(ctx context.Context, osID uint) ([]OSVulnerability, error)
	// ListVulnsByOsNameAndVersion fetches vulnerabilities for a single OS version. If maxVulnerabilities is provided,
	// limits the number of vulnerabilities returned while still providing the total count.
	ListVulnsByOsNameAndVersion(ctx context.Context, name, version string, includeCVSS bool, teamID *uint, maxVulnerabilities *int) (OSVulnerabilitiesWithCount, error)
	// ListVulnsByMultipleOSVersions is an optimized batch query that fetches vulnerabilities for multiple OS versions
	// in a single efficient operation. If maxVulnerabilities is provided, limits the number of vulnerabilities returned
	// per OS version while still providing the total count.
	ListVulnsByMultipleOSVersions(ctx context.Context, osVersions []OSVersion, includeCVSS bool, teamID *uint, maxVulnerabilities *int) (map[string]OSVulnerabilitiesWithCount, error)
	InsertOSVulnerabilities(ctx context.Context, vulnerabilities []OSVulnerability, source VulnerabilitySource) (int64, error)
	DeleteOSVulnerabilities(ctx context.Context, vulnerabilities []OSVulnerability) error
	// InsertOSVulnerability will either insert a new vulnerability in the datastore (in which
	// case it will return true) or if a matching record already exists it will update its
	// updated_at timestamp (in which case it will return false).
	InsertOSVulnerability(ctx context.Context, vuln OSVulnerability, source VulnerabilitySource) (bool, error)
	// DeleteOutOfDateOSVulnerabilities deletes 'operating_system_vulnerabilities' entries from the provided source where
	// the updated_at timestamp is older than the supplied timestamp
	DeleteOutOfDateOSVulnerabilities(ctx context.Context, source VulnerabilitySource, olderThan time.Time) error
	// DeleteOrphanedOSVulnerabilities deletes 'operating_system_vulnerabilities' entries where the operating_system_id
	// no longer has any associated hosts in 'host_operating_system'.
	DeleteOrphanedOSVulnerabilities(ctx context.Context) error

	ListKernelsByOS(ctx context.Context, osID uint, teamID *uint) ([]*Kernel, error)

	InsertKernelSoftwareMapping(ctx context.Context) error

	///////////////////////////////////////////////////////////////////////////////
	// Vulnerabilities

	// ListVulnerabilities returns a list of unique vulnerabilities based on the provided options.
	ListVulnerabilities(ctx context.Context, opt VulnListOptions) ([]VulnerabilityWithMetadata, *PaginationMetadata, error)
	// Vulnerability returns the vulnerability corresponding to the specified CVE ID
	Vulnerability(ctx context.Context, cve string, teamID *uint, includeCVEScores bool) (*VulnerabilityWithMetadata, error)
	// CountVulnerabilities returns the number of unique vulnerabilities based on the provided
	// options.
	CountVulnerabilities(ctx context.Context, opt VulnListOptions) (uint, error)
	// UpdateVulnerabilityHostCounts updates hosts counts for all vulnerabilities.  maxRoutines signifies the number of
	// goroutines to use for processing parallel database queries.
	UpdateVulnerabilityHostCounts(ctx context.Context, maxRoutines int) error
	// IsCVEKnownToFleet checks if the provided CVE is known to Fleet.
	IsCVEKnownToFleet(ctx context.Context, cve string) (bool, error)

	///////////////////////////////////////////////////////////////////////////////
	// Apple MDM

	// NewMDMAppleConfigProfile creates and returns a new configuration profile.
	NewMDMAppleConfigProfile(ctx context.Context, p MDMAppleConfigProfile, usesFleetVars []FleetVarName) (*MDMAppleConfigProfile, error)

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

	// GetMDMAppleDeclaration returns the declaration corresponding to the specified uuid.
	GetMDMAppleDeclaration(ctx context.Context, declUUID string) (*MDMAppleDeclaration, error)

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
	// DeleteMDMAppleDeclaration deletes the mdm declaration corresponding
	// to the specified declaration uuid.
	DeleteMDMAppleDeclaration(ctx context.Context, declUUID string) error

	// DeleteMDMAppleDeclartionByName deletes a DDM profile by its name for the
	// specified team (or no team).
	//
	// Returns nil, nil if the declaration with name on teamID doesn't exist.
	DeleteMDMAppleDeclarationByName(ctx context.Context, teamID *uint, name string) error

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

	// GetMDMAppleCommandResults returns the execution results of a command identified by a
	// CommandUUID. If a hostUUID is provided, it filters the results for that host.
	GetMDMAppleCommandResults(ctx context.Context, commandUUID string, hostUUID string) ([]*MDMCommandResult, error)

	// GetVPPCommandResults returns the execution results of a command identified by a CommandUUID,
	// only if the command corresponds to a VPP software (un)install for the specified host (else error notFound)
	GetVPPCommandResults(ctx context.Context, commandUUID string, hostUUID string) ([]*MDMCommandResult, error)

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
	// `host_dep_assignments` for all the provided hosts. mdmMigrationDeadlinesByHostID
	// should include migration deadlines from the DEP API for any hosts that had one set
	UpsertMDMAppleHostDEPAssignments(ctx context.Context, hosts []Host, abmTokenID uint, mdmMigrationDeadlinesByHostID map[uint]time.Time) error

	// IngestMDMAppleDevicesFromDEPSync creates new Fleet host records for MDM-enrolled devices that are
	// not already enrolled in Fleet. It returns the number of hosts created, and an error.
	IngestMDMAppleDevicesFromDEPSync(ctx context.Context, devices []godep.Device, abmTokenID uint, macOSTeam, iosTeam, ipadTeam *Team) (int64, error)

	// SetHostMDMMigrationCompleted sets a host's DEP record's migration as having completed by setting the
	// completed migration timestamp equal to the migration deadline timestamp. This is so that if we sync
	// the device from ABM again we know not to skip the migration.
	SetHostMDMMigrationCompleted(ctx context.Context, hostID uint) error

	// IngestMDMAppleDeviceFromOTAEnrollment creates new host records for
	// MDM-enrolled devices via OTA that are not already enrolled in Fleet.
	IngestMDMAppleDeviceFromOTAEnrollment(ctx context.Context, teamID *uint, idpUUID string, deviceInfo MDMAppleMachineInfo) error

	// MDMAppleUpsertHost creates or matches a Fleet host record for an
	// MDM-enrolled device.
	MDMAppleUpsertHost(ctx context.Context, mdmHost *Host, fromPersonalEnrollment bool) error

	// RestoreMDMApplePendingDEPHost restores a host that was previously deleted from Fleet.
	RestoreMDMApplePendingDEPHost(ctx context.Context, host *Host) error

	// MDMResetEnrollment resets all tables with enrollment-related
	// information if a matching row for the host exists.
	MDMResetEnrollment(ctx context.Context, hostUUID string, scepRenewalInProgress bool) error

	// ClearHostEnrolledFromMigration clears the enrolled from migration status of a host
	ClearHostEnrolledFromMigration(ctx context.Context, hostUUID string) error

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

	// GetHostDEPAssignmentsBySerial returns the DEP assignment for the host with the specified serial number.
	GetHostDEPAssignmentsBySerial(ctx context.Context, serial string) ([]*HostDEPAssignment, error)

	// GetNanoMDMEnrollment returns the nano enrollment information for the device id.
	GetNanoMDMEnrollment(ctx context.Context, id string) (*NanoEnrollment, error)

	// GetNanoMDMUserEnrollment returns the active nano user channel enrollment information for the device
	// id. Right now only one user channel enrollment is supported per device
	GetNanoMDMUserEnrollment(ctx context.Context, id string) (*NanoEnrollment, error)

	// GetNanoMDMUserEnrollmentUsernameAndUUID returns the short username and UUID of the user
	// channel enrollment for the device id. Right now only one user channel enrollment is
	// supported per device.
	GetNanoMDMUserEnrollmentUsernameAndUUID(ctx context.Context, deviceID string) (string, string, error)

	// UpdateNanoMDMUserEnrollmentUsername updates the username of the user channel with the given
	// userUUID for the specified deviceID. The actual ID of the nano_user to be updated will be
	// deviceID + ":" + userUUID because of the workings of nano. Note that this data can be
	// overriden by a TokenUpdate but that should provide the latest username
	UpdateNanoMDMUserEnrollmentUsername(ctx context.Context, deviceID string, userUUID string, username string) error

	// GetNanoMDMEnrollmentDetails returns the time of the most recent enrollment, the most recent
	// MDM protocol seen time, and whether the enrollment is hardware attested for the host with the given UUID
	GetNanoMDMEnrollmentDetails(ctx context.Context, hostUUID string) (*NanoMDMEnrollmentDetails, error)

	// IncreasePolicyAutomationIteration marks the policy to fire automation again.
	IncreasePolicyAutomationIteration(ctx context.Context, policyID uint) error

	// OutdatedAutomationBatch returns a batch of hosts that had a failing policy.
	OutdatedAutomationBatch(ctx context.Context) ([]PolicyFailure, error)

	// ListMDMAppleProfilesToInstall returns all the profiles that should
	// be installed based on diffing the ideal state vs the state we have
	// registered in `host_mdm_apple_profiles`, except if the optional argument `hostUUID` is passed.
	ListMDMAppleProfilesToInstall(ctx context.Context, hostUUID string) ([]*MDMAppleProfilePayload, error)

	// ListMDMAppleProfilesToRemove returns all the profiles that should
	// be removed based on diffing the ideal state vs the state we have
	// registered in `host_mdm_apple_profiles`
	ListMDMAppleProfilesToRemove(ctx context.Context) ([]*MDMAppleProfilePayload, error)

	// ListMDMAppleProfilesToInstallAndRemove returns the result of ListMDMAppleProfilesToInstall
	// and ListMDMAppleProfilesToRemove but queries for them in an isolated manner so that the two
	// lists reflect the same system state and no changes can be introduced between the queries.
	ListMDMAppleProfilesToInstallAndRemove(ctx context.Context) ([]*MDMAppleProfilePayload, []*MDMAppleProfilePayload, error)

	// BulkUpsertMDMAppleHostProfiles bulk-adds/updates records to track the
	// status of a profile in a host.
	BulkUpsertMDMAppleHostProfiles(ctx context.Context, payload []*MDMAppleBulkUpsertHostProfilePayload) error

	// BulkSetPendingMDMHostProfiles sets the status of profiles to install or to
	// remove for each affected host to pending for the provided criteria, which
	// may be either a list of hostIDs, teamIDs, profileUUIDs or hostUUIDs (only
	// one of those ID types can be provided).
	//
	// This reconciles Apple profiles, Apple declarations, and Android profiles
	// synchronously. Windows profile reconciliation is deferred: the
	// mdm_windows_profile_manager cron processes bounded host-window batches
	// using a persisted cursor (see ReconcileWindowsProfiles), so large host
	// populations may require multiple 30s ticks to converge. Callers must not
	// assume host_mdm_windows_profiles rows are written by the time this
	// function returns; if a caller needs immediate Windows state (e.g. a
	// test, or a synchronous UX flow), it must trigger reconciliation
	// explicitly. The returned MDMProfilesUpdates.WindowsConfigProfile is always false in
	// production: the BatchSetMDMProfiles flow that consumes this field
	// already gets an accurate transactional signal from BatchSetMDMProfiles
	// itself, and no other caller reads the field.
	BulkSetPendingMDMHostProfiles(ctx context.Context, hostIDs, teamIDs []uint,
		profileUUIDs, hostUUIDs []string) (updates MDMProfilesUpdates,
		err error)

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

	// AssociateHostMDMIdPAccountDB associates a host with an MDM IdP account
	AssociateHostMDMIdPAccountDB(ctx context.Context, hostUUID string, acctUUID string) error

	// GetMDMIdPAccountByUUID returns MDM IdP account that matches the given token.
	GetMDMIdPAccountByUUID(ctx context.Context, uuid string) (*MDMIdPAccount, error)

	// GetMDMIdPAccountByEmail returns MDM IdP account that matches the given email.
	GetMDMIdPAccountByEmail(ctx context.Context, email string) (*MDMIdPAccount, error)

	GetMDMIdPAccountsByHostUUIDs(ctx context.Context, hostUUIDs []string) (map[string]*MDMIdPAccount, error)

	// GetMDMAppleFileVaultSummary summarizes the current state of Apple disk encryption profiles on
	// each macOS host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetMDMAppleFileVaultSummary(ctx context.Context, teamID *uint) (*MDMAppleFileVaultSummary, error)

	///////////////////////////////////////////////////////////////////////////////
	// Apple MDM Recovery Lock Password

	// SetHostsRecoveryLockPasswords encrypts and stores recovery lock passwords for the given hosts.
	SetHostsRecoveryLockPasswords(ctx context.Context, passwords []HostRecoveryLockPasswordPayload) error

	// GetHostRecoveryLockPassword retrieves and decrypts the recovery lock password
	// for the given host UUID.
	GetHostRecoveryLockPassword(ctx context.Context, hostUUID string) (*HostRecoveryLockPassword, error)

	// GetHostRecoveryLockPasswordStatus returns the recovery lock password status for a given host.
	GetHostRecoveryLockPasswordStatus(ctx context.Context, hostUUID string) (*HostMDMRecoveryLockPassword, error)

	// GetHostsForRecoveryLockAction returns host UUIDs that need recovery lock password action:
	// - Teams with enable_recovery_lock_password = true
	// - macOS Apple Silicon hosts that are MDM enrolled
	// - No password saved or status is NULL (ready for command)
	GetHostsForRecoveryLockAction(ctx context.Context) ([]string, error)

	// RestoreRecoveryLockForReenabledHosts transitions hosts from "pending remove" back to
	// "verified install" when the recovery lock feature is re-enabled. This preserves the
	// existing password instead of trying to set a new one (which would fail).
	// Returns the number of hosts restored.
	RestoreRecoveryLockForReenabledHosts(ctx context.Context) (int64, error)

	// SetRecoveryLockVerified marks the recovery lock as verified.
	SetRecoveryLockVerified(ctx context.Context, hostUUID string) error

	// SetRecoveryLockFailed marks the recovery lock as failed with the given error message.
	SetRecoveryLockFailed(ctx context.Context, hostUUID string, errorMsg string) error

	// ClearRecoveryLockPendingStatus resets the recovery lock status to NULL for hosts
	// that failed to have their SetRecoveryLock commands enqueued. This allows them to
	// be picked up again on the next cron run.
	ClearRecoveryLockPendingStatus(ctx context.Context, hostUUIDs []string) error

	// ClaimHostsForRecoveryLockClear returns host UUIDs that need their recovery lock
	// cleared and marks them as pending. Returns hosts where the team/appconfig has
	// enable_recovery_lock_password=false and either:
	// - operation_type='install' and status='verified' (new clears), or
	// - operation_type='remove' and status=NULL (retries after failed enqueue)
	ClaimHostsForRecoveryLockClear(ctx context.Context) ([]string, error)

	// DeleteHostRecoveryLockPassword deletes the recovery lock password record for the given host.
	// Called after a successful clear operation.
	DeleteHostRecoveryLockPassword(ctx context.Context, hostUUID string) error

	// GetRecoveryLockOperationType returns the current operation type for the host's recovery lock.
	// Used by the result handler to determine if this was a set or clear operation.
	GetRecoveryLockOperationType(ctx context.Context, hostUUID string) (MDMOperationType, error)

	// InitiateRecoveryLockRotation stores a new pending password for rotation.
	// Validates: has verified/failed install password, no pending rotation, not in remove operation.
	InitiateRecoveryLockRotation(ctx context.Context, hostUUID string, newPassword string) error

	// CompleteRecoveryLockRotation moves pending password to active after MDM acknowledgment.
	// Sets: encrypted_password = pending_encrypted_password, clears pending columns, status = verified.
	CompleteRecoveryLockRotation(ctx context.Context, hostUUID string) error

	// FailRecoveryLockRotation marks rotation as failed, keeps pending password for potential retry.
	FailRecoveryLockRotation(ctx context.Context, hostUUID string, errorMsg string) error

	// ClearRecoveryLockRotation removes pending rotation (e.g., if command enqueue fails).
	ClearRecoveryLockRotation(ctx context.Context, hostUUID string) error

	// GetRecoveryLockRotationStatus returns current rotation state for API validation.
	GetRecoveryLockRotationStatus(ctx context.Context, hostUUID string) (*HostRecoveryLockRotationStatus, error)

	// HasPendingRecoveryLockRotation returns true if the host has a pending recovery lock rotation.
	HasPendingRecoveryLockRotation(ctx context.Context, hostUUID string) (bool, error)
	// ResetRecoveryLockForRetry resets a failed clear operation back to install/verified
	// so it will be picked up by ClaimHostsForRecoveryLockClear on the next cron cycle.
	// This is used when a clear command fails with a transient error (not password mismatch).
	ResetRecoveryLockForRetry(ctx context.Context, hostUUID string) error

	// MarkRecoveryLockPasswordViewed sets auto_rotate_at to 1 hour from now on
	// the host's install-state recovery lock row and returns the scheduled
	// rotation time. If the row is missing or in a state where rotation does
	// not apply (e.g., operation_type='remove'), returns a zero time and no
	// error so callers that have already retrieved the password do not 404.
	MarkRecoveryLockPasswordViewed(ctx context.Context, hostUUID string) (time.Time, error)

	// GetHostsForAutoRotation returns hosts where auto_rotate_at <= now
	// and are eligible for rotation (verified status, no pending rotation).
	// Returns host info needed for rotation and activity logging.
	// Limited to 100 hosts per batch.
	GetHostsForAutoRotation(ctx context.Context) ([]HostAutoRotationInfo, error)

	// SoftDeleteRecoveryLockPasswordsForUnenrolledHosts soft-deletes any live
	// recovery lock password rows whose host currently reports host_mdm.enrolled=0.
	// Apple wipes the device-side recovery lock whenever the MDM profile is removed,
	// so a row remaining for an unenrolled host is stale. Nulls rotation/view state
	// to prevent leakage into the re-animated row on re-enroll. Returns the number
	// of rows soft-deleted.
	SoftDeleteRecoveryLockPasswordsForUnenrolledHosts(ctx context.Context) (int64, error)

	///////////////////////////////////////////////////////////////////////////////
	// Managed local account

	// SaveHostManagedLocalAccount encrypts and stores the managed local account password
	// for a host. Uses INSERT ... ON DUPLICATE KEY UPDATE. Clears the existing account_uuid
	// if any since this is called on reenrollments
	SaveHostManagedLocalAccount(ctx context.Context, hostUUID, plaintextPassword, commandUUID string) error

	// GetHostManagedLocalAccountPassword retrieves and decrypts the managed local account
	// password for the given host UUID. Returns notFoundError if no record exists.
	GetHostManagedLocalAccountPassword(ctx context.Context, hostUUID string) (*HostManagedLocalAccountPassword, error)

	// GetHostManagedLocalAccountStatus returns the managed local account status for a host.
	// Translates DB NULL status to "pending". Returns notFoundError if no record exists.
	GetHostManagedLocalAccountStatus(ctx context.Context, hostUUID string) (*HostMDMManagedLocalAccount, error)

	// SetHostManagedLocalAccountStatus updates the status of the managed local account for a host.
	SetHostManagedLocalAccountStatus(ctx context.Context, hostUUID string, status MDMDeliveryStatus) error

	// GetManagedLocalAccountByCommandUUID looks up the host UUID associated with a managed
	// local account command UUID. Returns notFoundError if no matching record (i.e. SSO-only
	// AccountConfiguration).
	GetManagedLocalAccountByCommandUUID(ctx context.Context, commandUUID string) (host *Host, err error)

	// GetManagedLocalAccountUUID returns the account UUID captured from osquery for the
	// managed local account on the given host. Returns a NotFound error when no
	// managed_local_account row exists. A nil *string means the row exists but
	// account_uuid has not yet been captured. Reads from the read replica.
	GetManagedLocalAccountUUID(ctx context.Context, hostUUID string) (accountUUID *string, err error)

	// SetManagedLocalAccountUUID captures the osquery-reported account UUID on an existing
	// managed_local_account row. No-op if the row doesn't exist or account_uuid is already set
	// to the specified UUID.
	SetManagedLocalAccountUUID(ctx context.Context, hostUUID, accountUUID string) error

	// MarkManagedLocalAccountPasswordViewed records that the managed local account password
	// was viewed by a user (UI or API). On first view it sets status='pending',
	// auto_rotate_at = NOW(6) + 65 minutes, and initiated_by_fleet=1. Subsequent views
	// inside the window do NOT extend auto_rotate_at; the existing value is returned.
	// Returns notFound if the row doesn't exist, encrypted_password IS NULL, status='failed',
	// or a rotation is already pending.
	MarkManagedLocalAccountPasswordViewed(ctx context.Context, hostUUID string) (rotateAt time.Time, err error)

	// InitiateManagedLocalAccountRotation stores the (datastore-encrypted) pending
	// password and pending_command_uuid for an in-flight SetAutoAdminPassword command.
	// Eligibility: row exists, encrypted_password IS NOT NULL, status != 'failed',
	// account_uuid IS NOT NULL, pending_encrypted_password IS NULL. Does NOT modify
	// initiated_by_fleet — that flag is owned by the view path (sets to 1) and the
	// deferred-manual path (sets to 0). Returns ErrManagedLocalAccountRotationPending
	// or ErrManagedLocalAccountNotEligible when ineligible, or notFound when the row
	// is missing.
	InitiateManagedLocalAccountRotation(ctx context.Context, hostUUID, pendingPlaintextPassword, cmdUUID string) error

	// MarkManagedLocalAccountRotationDeferred records a manual rotation that the service
	// could not enqueue immediately because account_uuid is missing. Sets status='pending',
	// auto_rotate_at=NOW(6) (so the cron picks it up as soon as the UUID lands), and
	// initiated_by_fleet=0 so the cron skips re-logging the activity. Idempotent.
	MarkManagedLocalAccountRotationDeferred(ctx context.Context, hostUUID string) error

	// ClearManagedLocalAccountRotation unwinds pending rotation columns (used when the
	// commander returned a non-APNs persistence error after InitiateManagedLocalAccountRotation
	// already populated them).
	ClearManagedLocalAccountRotation(ctx context.Context, hostUUID string) error

	// CompleteManagedLocalAccountRotation finalizes a successful rotation acknowledgment.
	// Validates pending_command_uuid matches the acked command, swaps pending password into
	// encrypted_password, clears pending_*/auto_rotate_at, sets status='verified', and
	// resets initiated_by_fleet=0. Returns notFound when the command UUID does not match
	// the row's pending one.
	CompleteManagedLocalAccountRotation(ctx context.Context, hostUUID, cmdUUID string) error

	// FailManagedLocalAccountRotation marks the row's status='failed' and clears pending
	// columns; encrypted_password (the previous-known-good password) is left intact so
	// the password remains usable.
	FailManagedLocalAccountRotation(ctx context.Context, hostUUID, cmdUUID, errorMessage string) error

	// GetManagedLocalAccountsForAutoRotation returns up to 100 rows whose auto_rotate_at
	// has elapsed and which are eligible for an enqueue: account_uuid IS NOT NULL,
	// encrypted_password IS NOT NULL, pending_encrypted_password IS NULL, status != 'failed'.
	// status='pending' is intentionally allowed (a viewed row sits in pending while waiting).
	GetManagedLocalAccountsForAutoRotation(ctx context.Context) ([]HostManagedLocalAccountAutoRotationInfo, error)

	// GetManagedLocalAccountByPendingCommandUUID resolves a SetAutoAdminPassword ack back
	// to its host via pending_command_uuid. Returns notFound when no row matches.
	GetManagedLocalAccountByPendingCommandUUID(ctx context.Context, commandUUID string) (host *Host, err error)

	// InsertMDMAppleBootstrapPackage insterts a new bootstrap package in the
	// database (or S3 if configured).
	InsertMDMAppleBootstrapPackage(ctx context.Context, bp *MDMAppleBootstrapPackage, pkgStore MDMBootstrapPackageStore) error
	// CopyMDMAppleBootstrapPackage copies the bootstrap package specified in the app config (if any)
	// specified team (and a new token is assigned). It also updates the team config with the default bootstrap package URL.
	CopyDefaultMDMAppleBootstrapPackage(ctx context.Context, ac *AppConfig, toTeamID uint) error
	// DeleteMDMAppleBootstrapPackage deletes the bootstrap package for the given team id.
	DeleteMDMAppleBootstrapPackage(ctx context.Context, teamID uint) error
	// GetMDMAppleBootstrapPackageMeta returns metadata about the bootstrap
	// package for a team.
	GetMDMAppleBootstrapPackageMeta(ctx context.Context, teamID uint) (*MDMAppleBootstrapPackage, error)
	// GetMDMAppleBootstrapPackageBytes returns the bytes of a bootstrap package
	// with the given token.
	GetMDMAppleBootstrapPackageBytes(ctx context.Context, token string, pkgStore MDMBootstrapPackageStore) (*MDMAppleBootstrapPackage, error)
	// GetMDMAppleBootstrapPackageSummary returns an aggregated summary of the
	// status of the bootstrap package for hosts in a team.
	GetMDMAppleBootstrapPackageSummary(ctx context.Context, teamID uint) (*MDMAppleBootstrapPackageSummary, error)

	// RecordHostBootstrapPackage records a command used to install a
	// bootstrap package in a host.
	RecordHostBootstrapPackage(ctx context.Context, commandUUID string, hostUUID string) error
	// RecordSkippedHostBootstrapPackage records that a host skipped the
	// installation of a bootstrap package.
	RecordSkippedHostBootstrapPackage(ctx context.Context, hostUUID string) error
	// GetHostBootstrapPackageCommand returns the MDM command uuid used to
	// install a bootstrap package in a host.
	GetHostBootstrapPackageCommand(ctx context.Context, hostUUID string) (string, error)

	// CleanupUnusedBootstrapPackages will remove bootstrap packages that have no
	// references to them from the mdm_apple_bootstrap_packages table.
	CleanupUnusedBootstrapPackages(ctx context.Context, pkgStore MDMBootstrapPackageStore, removeCreatedBefore time.Time) error

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
	// Get the MDM Apple Setup Assistant profile uuid and timestamp for the
	// specified ABM token identified by organization name.
	GetMDMAppleSetupAssistantProfileForABMToken(ctx context.Context, teamID *uint, abmTokenOrgName string) (string, time.Time, error)
	// Delete the MDM Apple Setup Assistant for the provided team or no team.
	DeleteMDMAppleSetupAssistant(ctx context.Context, teamID *uint) error
	// Set the profile UUID generated by the call to Apple's DefineProfile API of
	// the setup assistant for a team or no team.
	//
	// With multi-ABM token support, this profileUUID is stored along with the
	// ABM token used to register it. The token is identified by its organization
	// name.
	SetMDMAppleSetupAssistantProfileUUID(ctx context.Context, teamID *uint, profileUUID, abmTokenOrgName string) error

	// Set the profile UUID generated by the call to Apple's DefineProfile API
	// of the default setup assistant for a team or no team. The default
	// profile is the same regardless of the team, except for the enabling of
	// the end-user authentication which may be configured per-team and affects
	// the JSON registered with Apple's API, possibly resulting in different
	// profile UUIDs for the same profile depending on the team.
	//
	// With multi-ABM token support, this profileUUID is stored along with the
	// ABM token used to register it. The token is identified by its organization
	// name.
	SetMDMAppleDefaultSetupAssistantProfileUUID(ctx context.Context, teamID *uint, profileUUID, abmTokenOrgName string) error

	// Get the profile UUID and last update timestamp for the default setup
	// assistant for a team or no team, as registered with the ABM token
	// represented by the organization name.
	GetMDMAppleDefaultSetupAssistant(ctx context.Context, teamID *uint, abmTokenOrgName string) (profileUUID string, updatedAt time.Time, err error)

	// GetMatchingHostSerials receives a list of serial numbers and returns
	// a map that only contains the serials that have a matching row in the `hosts` table.
	GetMatchingHostSerials(ctx context.Context, serials []string) (map[string]*Host, error)

	// GetMatchingHostSerialsMarkedDeleted takes a list of device serial numbers and returns a map
	// of only the ones that were found in the `hosts` table AND have a row in
	// `host_dep_assignments` that is marked as deleted.
	GetMatchingHostSerialsMarkedDeleted(ctx context.Context, serials []string) (map[string]struct{}, error)

	// DeleteHostDEPAssignmentsFromAnotherABM makes as deleted any DEP entry that matches one of the provided serials only if the entry is NOT associated to the provided ABM token.
	DeleteHostDEPAssignmentsFromAnotherABM(ctx context.Context, abmTokenID uint, serials []string) error

	// DeleteHostDEPAssignments marks as deleted entries in
	// host_dep_assignments for host with matching serials only if the entry is associated to the provided ABM token.
	DeleteHostDEPAssignments(ctx context.Context, abmTokenID uint, serials []string) error

	// UpdateHostDEPAssignProfileResponses receives a profile UUID and threes lists of serials, each representing
	// one of the three possible responses, and updates the host_dep_assignments table with the corresponding responses. For each response, it also sets the ABM token id in the table to the provided value.
	UpdateHostDEPAssignProfileResponses(ctx context.Context, resp *godep.ProfileResponse, abmTokenID uint) error

	// UpdateHostDEPAssignProfileResponsesSameABM receives a profile UUID and threes lists of serials, each representing
	// one of the three possible responses, and updates the host_dep_assignments table with the corresponding responses.
	// The ABM token ID remains unchanged.
	UpdateHostDEPAssignProfileResponsesSameABM(ctx context.Context, resp *godep.ProfileResponse) error

	// ScreenDEPAssignProfileSerialsForCooldown returns the serials that are still in cooldown and the
	// ones that are ready to be assigned a profile. If `screenRetryJobs` is true, it will also skip
	// any serials that have a non-zero `retry_job_id`.
	ScreenDEPAssignProfileSerialsForCooldown(ctx context.Context, serials []string) (skipSerialsByOrgName map[string][]string, serialsByOrgName map[string][]string, err error)
	// GetDEPAssignProfileExpiredCooldowns returns the serials of the hosts that have expired
	// cooldowns limited to the amount we sync in a single run, grouped by team.
	GetDEPAssignProfileExpiredCooldowns(ctx context.Context) (map[uint][]string, error)
	// UpdateDEPAssignProfileRetryPending sets the retry_pending flag for the hosts with the given
	// serials.
	UpdateDEPAssignProfileRetryPending(ctx context.Context, jobID uint, serials []string) error

	// InsertMDMAppleDDMRequest inserts a DDM request.
	InsertMDMAppleDDMRequest(ctx context.Context, hostUUID, messageType string, rawJSON json.RawMessage) error

	// MDMAppleDDMDeclarationsToken returns the token used to synchronize declarations for the
	// specified host UUID.
	MDMAppleDDMDeclarationsToken(ctx context.Context, hostUUID string) (*MDMAppleDDMDeclarationsToken, error)
	// MDMAppleDDMDeclarationItems returns the declaration items for the specified host UUID.
	MDMAppleDDMDeclarationItems(ctx context.Context, hostUUID string) ([]MDMAppleDDMDeclarationItem, error)
	// MDMAppleDDMDeclarationPayload returns the declaration payload for the specified identifier and team.
	MDMAppleDDMDeclarationsResponse(ctx context.Context, identifier string, hostUUID string) (*MDMAppleDeclaration, error)
	// MDMAppleBatchSetHostDeclarationState
	MDMAppleBatchSetHostDeclarationState(ctx context.Context) ([]string, error)
	// MDMAppleHostDeclarationsGetAndClearResync finds any hosts that requested a resync.
	// This is used to cover special cases where we're not 100% certain of the declarations on the device.
	MDMAppleHostDeclarationsGetAndClearResync(ctx context.Context) (hostUUIDs []string, err error)
	// MDMAppleStoreDDMStatusReport receives a host.uuid and a slice
	// of declarations, and updates the tracked host declaration status for
	// matching declarations.
	//
	// It also takes care of cleaning up all host declarations that are
	// pending removal.
	MDMAppleStoreDDMStatusReport(ctx context.Context, hostUUID string, updates []*MDMAppleHostDeclaration) error
	// SetHostMDMAppleDeclarationStatus updates the status and detail of a
	// single declaration for a host. If variablesUpdatedAt is non-nil, it also
	// sets the variables_updated_at timestamp.
	SetHostMDMAppleDeclarationStatus(ctx context.Context, hostUUID string, declarationUUID string, status *MDMDeliveryStatus, detail string, variablesUpdatedAt *time.Time) error
	// MDMAppleSetPendingDeclarationsAs updates all ("pending", "install")
	// declarations for a host to be ("verifying", status), where status is
	// the provided value.
	MDMAppleSetPendingDeclarationsAs(ctx context.Context, hostUUID string, status *MDMDeliveryStatus, detail string) error
	MDMAppleSetRemoveDeclarationsAsPending(ctx context.Context, hostUUID string, declarationUUIDs []string) error
	// GetMDMAppleOSUpdatesSettingsByHostSerial returns applicable Apple OS update settings (if any)
	// for the host with the given serial number alongside the host's platform. The host must be DEP assigned to Fleet.
	GetMDMAppleOSUpdatesSettingsByHostSerial(ctx context.Context, hostSerial string) (string, *AppleOSUpdateSettings, error)
	GetCAConfigAsset(ctx context.Context, name string, assetType CAConfigAssetType) (*CAConfigAsset, error)
	SaveCAConfigAssets(ctx context.Context, assets []CAConfigAsset) error
	DeleteCAConfigAssets(ctx context.Context, names []string) error

	// GetABMTokenByOrgName retrieves the Apple Business token identified by
	// its unique name (the organization name).
	GetABMTokenByOrgName(ctx context.Context, orgName string) (*ABMToken, error)

	// SaveABMToken updates the ABM token using the provided struct.
	SaveABMToken(ctx context.Context, tok *ABMToken) error

	InsertVPPToken(ctx context.Context, tok *VPPTokenData) (*VPPTokenDB, error)
	ListVPPTokens(ctx context.Context) ([]*VPPTokenDB, error)
	GetVPPToken(ctx context.Context, tokenID uint) (*VPPTokenDB, error)
	GetVPPTokenByTeamID(ctx context.Context, teamID *uint) (*VPPTokenDB, error)
	// UpdateVPPTokenTeams sets the teams associated with this token.
	// Note that updating the token's associations removes all
	// apps-team associations using this token
	UpdateVPPTokenTeams(ctx context.Context, id uint, teams []uint) (*VPPTokenDB, error)
	UpdateVPPToken(ctx context.Context, id uint, tok *VPPTokenData) (*VPPTokenDB, error)
	// UpdateVPPTokenCountryCode persists the lowercase ISO country code for a
	// VPP token. Used to lazy-backfill the column for tokens uploaded before
	// the country_code column existed.
	UpdateVPPTokenCountryCode(ctx context.Context, tokenID uint, countryCode string) error
	// UpdateVPPAppCountryCode persists the anchored storefront country for a
	// (adam_id, platform) row in vpp_apps. Used by the re-anchor self-heal
	// path when the original anchored country has no Fleet-known token left.
	UpdateVPPAppCountryCode(ctx context.Context, adamID string, platform InstallableDevicePlatform, countryCode string) error
	// BackfillVPPAppCountriesFromTokens populates `vpp_apps.country_code` for
	// any rows that are still NULL, by joining through `vpp_apps_teams` to
	// the `vpp_tokens` row whose `country_code` is set. Returns the number of
	// rows updated. Used by the one-shot legacy backfill that runs at server
	// startup. Becomes a no-op once all rows are populated.
	BackfillVPPAppCountriesFromTokens(ctx context.Context) (int64, error)
	// GetVPPAppByAdamIDPlatform returns the vpp_apps row for the given
	// (adam_id, platform), or a NotFound error if no row exists. Used by the
	// anchoring logic to decide whether the next add is a first-add or a
	// subsequent add.
	GetVPPAppByAdamIDPlatform(ctx context.Context, adamID string, platform InstallableDevicePlatform) (*VPPApp, error)
	// GetVPPTokenOwningAppInCountry returns a VPP token whose country_code
	// matches the given country and which owns the (adam_id, platform) app
	// via vpp_apps_teams. Returns NotFound when no eligible token exists —
	// callers may treat this as a re-anchor signal.
	GetVPPTokenOwningAppInCountry(ctx context.Context, adamID string, platform InstallableDevicePlatform, country string) (*VPPTokenDB, error)
	DeleteVPPToken(ctx context.Context, tokenID uint) error

	// SetABMTokenTermsExpiredForOrgName is a specialized method to set only the
	// terms_expired flag of the ABM token identified by the organization name.
	// It returns whether that flag was previously set for this token.
	SetABMTokenTermsExpiredForOrgName(ctx context.Context, orgName string, expired bool) (wasSet bool, err error)

	// CountABMTokensWithTermsExpired returns a count of ABM tokens that are
	// flagged with the Apple BM terms expired.
	CountABMTokensWithTermsExpired(ctx context.Context) (int, error)

	// InsertABMToken inserts a new ABM token into the datastore.
	InsertABMToken(ctx context.Context, tok *ABMToken) (*ABMToken, error)

	// ListABMTokens lists all of the ABM tokens.
	ListABMTokens(ctx context.Context) ([]*ABMToken, error)

	// DeleteABMToken deletes the given ABM token from the datastore.
	DeleteABMToken(ctx context.Context, tokenID uint) error

	// GetABMTokenByID retrieves the ABM token with the given ID.
	GetABMTokenByID(ctx context.Context, tokenID uint) (*ABMToken, error)

	// GetABMTokenCount returns the number of ABM tokens in the DB.
	GetABMTokenCount(ctx context.Context) (int, error)

	// GetABMTokenOrgNamesAssociatedWithTeam returns the set of ABM organization
	// names that correspond to the union of
	// - the tokens used to create each of the DEP hosts in that team.
	// - the tokens targeting that team as default for any platform.
	GetABMTokenOrgNamesAssociatedWithTeam(ctx context.Context, teamID *uint) ([]string, error)

	// ClearMDMUpcomingActivitiesDB clears the upcoming activities of the host that
	// require MDM to be processed, for when MDM is turned off for the host (or
	// when it turns on again, e.g. after removing the enrollment profile - it may
	// not necessarily report as "turned off" in that scenario).
	ClearMDMUpcomingActivitiesDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string) error

	// GetMDMAppleEnrolledDeviceDeletedFromFleet returns the information of a
	// device that is still enrolled in Fleet MDM but the corresponding host has
	// been deleted from Fleet.
	GetMDMAppleEnrolledDeviceDeletedFromFleet(ctx context.Context, hostUUID string) (*MDMAppleEnrolledDeviceInfo, error)

	// GetMDMAppleHostMDMEnrollRef returns the host mdm enrollment reference for
	// the given host ID.
	GetMDMAppleHostMDMEnrollRef(ctx context.Context, hostID uint) (string, error)
	// UpdateMDMAppleHostMDMEnrollRef updates the host mdm enrollment reference for
	// the given host ID. It returns a boolean indicating whether any row was
	// affected.
	UpdateMDMAppleHostMDMEnrollRef(ctx context.Context, hostID uint, enrollRef string) (bool, error)
	// DeactivateMDMAppleHostSCEPRenewCommands deactivates any pending SCEP renew
	// commands for the given host UUID in the nano_enrollment_queue. It also clears all renew
	// command uuid associated in nano_cert_auth_assocations for that host.
	DeactivateMDMAppleHostSCEPRenewCommands(ctx context.Context, hostUUID string) error

	// ListMDMAppleEnrolledIphoneIpadDeletedFromFleet returns a list of nano
	// device IDs (host UUIDs) of iPhone and iPad that are enrolled in Fleet MDM
	// but deleted from Fleet.
	ListMDMAppleEnrolledIPhoneIpadDeletedFromFleet(ctx context.Context, limit int) ([]string, error)

	// ReconcileMDMAppleEnrollRef returns the legacy enrollment reference for a
	// device with the given host UUID.
	ReconcileMDMAppleEnrollRef(ctx context.Context, enrollRef string, machineInfo *MDMAppleMachineInfo) (string, error)
	// GetMDMIdPAccountByHostUUID returns the MDM IdP account that associated with the given host UUID.
	GetMDMIdPAccountByHostUUID(ctx context.Context, hostUUID string) (*MDMIdPAccount, error)
	// AssociateHostMDMIdPAccount associates the given host UUID with the MDM IdP account UUID
	AssociateHostMDMIdPAccount(ctx context.Context, hostUUID string, accountUUID string) error

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

	// MDMWindowsDeleteEnrolledDeviceOnReenrollment deletes a given windows
	// device enrollment entry from the database using the HW device id.
	MDMWindowsDeleteEnrolledDeviceOnReenrollment(ctx context.Context, mdmDeviceHWID string) error

	// MDMWindowsGetEnrolledDeviceWithDeviceID receives a Windows MDM device id and returns the device information
	MDMWindowsGetEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) (*MDMWindowsEnrolledDevice, error)

	// MDMWindowsGetEnrolledDeviceWithHostUUID returns the MDMWindowsEnrolledDevice information for a given HostUUID
	MDMWindowsGetEnrolledDeviceWithHostUUID(ctx context.Context, hostUUID string) (*MDMWindowsEnrolledDevice, error)

	// MDMWindowsDeleteEnrolledDeviceWithDeviceID deletes a give MDMWindowsEnrolledDevice entry from the database using the device id
	MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx context.Context, mdmDeviceID string) error

	// MDMWindowsInsertCommandForHosts inserts a single command that may
	// target multiple hosts identified by their UUID, enqueuing one command
	// for each device.
	MDMWindowsInsertCommandForHosts(ctx context.Context, hostUUIDs []string, cmd *MDMWindowsCommand) error

	// MDMWindowsInsertCommandsForHost atomically inserts a batch of Windows MDM commands targeting a single host
	// (identified by host UUID or MDM device ID). All commands succeed or none do, in one transaction. Used by
	// the ESP finalize path so a partial-insert + fresh-UUID retry can't leave orphan rows in the queue.
	MDMWindowsInsertCommandsForHost(ctx context.Context, hostUUIDOrDeviceID string, cmds []*MDMWindowsCommand) error

	MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx context.Context, hostUUIDs []string, cmd *MDMWindowsCommand, profilePayloads []*MDMWindowsBulkUpsertHostProfilePayload) error

	// MDMWindowsGetPendingCommands returns all pending commands for the given enrollment.
	MDMWindowsGetPendingCommands(ctx context.Context, enrollmentID uint) ([]*MDMWindowsCommand, error)

	// MDMWindowsSaveResponse saves a full response for the given enrollment.
	MDMWindowsSaveResponse(ctx context.Context, enrolledDevice *MDMWindowsEnrolledDevice, enrichedSyncML EnrichedSyncML, commandIDsBeingResent []string) (*MDMWindowsSaveResponseResult, error)

	// GetMDMWindowsCommands returns the results of command
	GetMDMWindowsCommandResults(ctx context.Context, commandUUID string, hostUUID string) ([]*MDMCommandResult, error)

	// UpdateMDMWindowsEnrollmentsHostUUID updates the host UUID for a given MDM device ID.
	UpdateMDMWindowsEnrollmentsHostUUID(ctx context.Context, hostUUID string, mdmDeviceID string) (bool, error)

	// SetMDMWindowsAwaitingConfiguration performs a compare-and-swap update on the
	// awaiting_configuration status for a Windows MDM enrollment identified by
	// device ID. The update only applies if the current status matches expectFrom,
	// preventing races between concurrent management checkins. Returns true if the
	// transition occurred.
	SetMDMWindowsAwaitingConfiguration(ctx context.Context, mdmDeviceID string, expectFrom, to WindowsMDMAwaitingConfiguration) (bool, error)

	// GetMDMWindowsAwaitingConfigurationByHostUUID returns the awaiting
	// configuration value for the Windows MDM enrollment of the given host.
	// This is a lightweight read for the orbit config polling path.
	GetMDMWindowsAwaitingConfigurationByHostUUID(ctx context.Context, hostUUID string) (WindowsMDMAwaitingConfiguration, error)

	// HasWindowsSetupExperienceItemsForTeam returns true if any active Windows setup-experience software
	// installers (with install_during_setup) are configured for the given team. teamID=0 means "no team /
	// global". Used by the ESP release gate to disambiguate between "no setup configured" (safe to release)
	// and "setup configured but orbit hasn't initialized yet" (must wait) when
	// setup_experience_status_results is empty.
	HasWindowsSetupExperienceItemsForTeam(ctx context.Context, teamID uint) (bool, error)

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

	// ResendHostMDMProfile updates the host's profile status to NULL thereby triggering the profile
	// to be resent upon the next cron run.
	ResendHostMDMProfile(ctx context.Context, hostUUID string, profileUUID string) error

	// BatchResendMDMProfileToHosts updates the profile status to NULL for the
	// matching hosts that satisfy the filter, thereby triggering the profile to
	// be resent upon the next cron run.
	BatchResendMDMProfileToHosts(ctx context.Context, profileUUID string, filters BatchResendMDMProfileFilters) (int64, error)

	// GetMDMConfigProfileStatus returns the number of hosts per status for the
	// specified profile UUID.
	GetMDMConfigProfileStatus(ctx context.Context, profileUUID string) (MDMConfigProfileStatus, error)

	// GetHostMDMProfileInstallStatus returns the status of the profile for the host.
	GetHostMDMProfileInstallStatus(ctx context.Context, hostUUID string, profileUUID string) (MDMDeliveryStatus, error)

	///////////////////////////////////////////////////////////////////////////////
	// Linux MDM

	// GetLinuxDiskEncryptionSummary summarizes the current state of Linux disk encryption on
	// each Linux host in the specified team (or, if no team is specified, each host that is not assigned
	// to any team).
	GetLinuxDiskEncryptionSummary(ctx context.Context, teamID *uint) (MDMLinuxDiskEncryptionSummary, error)

	///////////////////////////////////////////////////////////////////////////////
	// MDM Commands

	// GetMDMCommandPlatform returns the platform (i.e. "darwin" or "windows") for the given command.
	GetMDMCommandPlatform(ctx context.Context, commandUUID string) (string, error)

	// ListMDMCommands returns a list of MDM Apple commands that have been
	// executed, based on the provided options.
	// returns a non-nil count if filtering by command_status = pending
	ListMDMCommands(ctx context.Context, tmFilter TeamFilter, listOpts *MDMCommandListOptions) ([]*MDMCommand, *int64, *PaginationMetadata, error)

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

	// ListMDMWindowsProfilesToInstallForHost returns the profiles that should
	// be installed for a specific host.
	ListMDMWindowsProfilesToInstallForHost(ctx context.Context, hostUUID string) ([]*MDMWindowsProfilePayload, error)

	// ListMDMWindowsProfilesToRemove returns all the profiles that should
	// be removed based on diffing the ideal state vs the state we have
	// registered in `host_mdm_windows_profiles`
	ListMDMWindowsProfilesToRemove(ctx context.Context) ([]*MDMWindowsProfilePayload, error)

	// ListMDMWindowsProfilesToInstallForHosts is the scoped variant of
	// ListMDMWindowsProfilesToInstall: it returns rows only for the given
	// host UUIDs. The cron uses this to bound per-tick work; see
	// ReconcileWindowsProfiles.
	ListMDMWindowsProfilesToInstallForHosts(ctx context.Context, hostUUIDs []string) ([]*MDMWindowsProfilePayload, error)

	// ListMDMWindowsProfilesToRemoveForHosts is the scoped variant of
	// ListMDMWindowsProfilesToRemove: it returns rows only for the given
	// host UUIDs. The cron uses this to bound per-tick work; see
	// ReconcileWindowsProfiles.
	ListMDMWindowsProfilesToRemoveForHosts(ctx context.Context, hostUUIDs []string) ([]*MDMWindowsProfilePayload, error)

	// ListNextPendingMDMWindowsHostUUIDs returns up to batchSize host UUIDs
	// (sorted ascending) where host_uuid > afterHostUUID and the host has
	// any pending Windows MDM profile reconciliation work. Used by the
	// cron's batched reconciliation path; see ReconcileWindowsProfiles.
	ListNextPendingMDMWindowsHostUUIDs(ctx context.Context, afterHostUUID string, batchSize int) ([]string, error)

	// GetMDMWindowsReconcileCursor returns the persisted host_uuid cursor
	// used by the Windows MDM reconciliation cron to bound per-tick work.
	// Returns "" if no cursor is set or if the implementation does not
	// support cursor persistence (the bare mysql.Datastore returns "" here;
	// the mysqlredis wrapper backs it with Redis). See
	// ReconcileWindowsProfiles.
	GetMDMWindowsReconcileCursor(ctx context.Context) (string, error)

	// SetMDMWindowsReconcileCursor persists the host_uuid cursor used by
	// the Windows MDM reconciliation cron. See GetMDMWindowsReconcileCursor.
	SetMDMWindowsReconcileCursor(ctx context.Context, cursor string) error

	// BulkUpsertMDMWindowsHostProfiles bulk-adds/updates records to track the
	// status of a profile in a host.
	BulkUpsertMDMWindowsHostProfiles(ctx context.Context, payload []*MDMWindowsBulkUpsertHostProfilePayload) error

	// GetMDMWindowsProfilesContents retrieves the XML contents of the
	// profiles requested.
	GetMDMWindowsProfilesContents(ctx context.Context, profileUUIDs []string) (map[string]MDMWindowsProfileContents, error)

	// GetExistingMDMWindowsProfileUUIDs returns the subset of the given
	// profile UUIDs that still exist in mdm_windows_configuration_profiles.
	// Callers use this to detect profiles deleted by a concurrent admin
	// action between listing and a downstream upsert (e.g. the
	// mdm_windows_profile_manager cron's reconciliation loop); installing a
	// profile that has since been deleted would create an unremovable zombie
	// row because the <Delete> builder needs the now-missing SyncML.
	GetExistingMDMWindowsProfileUUIDs(ctx context.Context, profileUUIDs []string) (map[string]struct{}, error)

	// BulkDeleteMDMWindowsHostsConfigProfiles deletes entries from
	// host_mdm_windows_profiles that match the given payload.
	BulkDeleteMDMWindowsHostsConfigProfiles(ctx context.Context, payload []*MDMWindowsProfilePayload) error

	// NewMDMWindowsConfigProfile creates and returns a new configuration profile.
	NewMDMWindowsConfigProfile(ctx context.Context, cp MDMWindowsConfigProfile, usesFleetVars []FleetVarName) (*MDMWindowsConfigProfile, error)

	// SetOrUpdateMDMWindowsConfigProfile creates or replaces a Windows profile.
	// The profile gets replaced if it already exists for the same team and name
	// combination.
	SetOrUpdateMDMWindowsConfigProfile(ctx context.Context, cp MDMWindowsConfigProfile) error

	// BatchSetMDMProfiles sets the MDM Apple or Windows profiles for the given team or
	// no team in a single transaction.
	BatchSetMDMProfiles(ctx context.Context, tmID *uint, macProfiles []*MDMAppleConfigProfile, winProfiles []*MDMWindowsConfigProfile,
		macDeclarations []*MDMAppleDeclaration, androidProfiles []*MDMAndroidConfigProfile, profilesVariables []MDMProfileIdentifierFleetVariables) (updates MDMProfilesUpdates, err error)

	// NewMDMAppleDeclaration creates and returns a new MDM Apple declaration.
	NewMDMAppleDeclaration(ctx context.Context, declaration *MDMAppleDeclaration, usesFleetVars []FleetVarName) (*MDMAppleDeclaration, error)

	// SetOrUpdateMDMAppleDeclaration upserts the MDM Apple declaration.
	SetOrUpdateMDMAppleDeclaration(ctx context.Context, declaration *MDMAppleDeclaration, usesFleetVars []FleetVarName) (*MDMAppleDeclaration, error)

	///////////////////////////////////////////////////////////////////////////////
	// Host Script Results

	// NewHostScriptExecutionRequest creates a new host script result entry with
	// just the script to run information (result is not yet available).
	NewHostScriptExecutionRequest(ctx context.Context, request *HostScriptRequestPayload) (*HostScriptResult, error)
	// SetHostScriptExecutionResult stores the result of a host script execution
	// return nil, "", nil. action is populated if this script was an MDM action (lock/unlock/wipe/uninstall).
	SetHostScriptExecutionResult(ctx context.Context, result *HostScriptResultPayload, attemptNumber *int) (hsr *HostScriptResult, action string, err error)
	// GetHostScriptExecutionResult returns the result of a host script
	// execution. It returns the host script results even if no results have been
	// received, it is the caller's responsibility to check if that was the case
	// (with ExitCode being null).
	GetHostScriptExecutionResult(ctx context.Context, execID string) (*HostScriptResult, error)
	// GetSelfServiceUninstallScriptExecutionResult returns the result of a host script
	// execution if it was for the specified host and the script was for a self-service uninstall.
	// It returns the host script results if no results have been received as long as the script
	// has been activated in the unified queue (will return Not Found if the execution is still in
	// upcoming_activities). It is the caller's responsibility to check if ExitCode is null.
	GetSelfServiceUninstallScriptExecutionResult(ctx context.Context, execID string, hostID uint) (*HostScriptResult, error)
	// ListPendingHostScriptExecutions returns all the pending host script executions, which are those that have yet
	// to record a result. Pass onlyShowInternal as true to return only scripts that execute when script execution is
	// globally disabled (uninstall/lock/unlock/wipe).
	ListPendingHostScriptExecutions(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*HostScriptResult, error)
	// ListReadyToExecuteScriptsForHost is like ListPendingHostScriptExecutions
	// except that it only returns those that are ready to execute ("activated" in
	// the upcoming activities queue, available for orbit to process).
	ListReadyToExecuteScriptsForHost(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*HostScriptResult, error)

	// NewScript creates a new saved script.
	NewScript(ctx context.Context, script *Script) (*Script, error)

	// UpdateScriptContents replaces the script contents of a script
	UpdateScriptContents(ctx context.Context, scriptID uint, scriptContents string) (*Script, error)

	// Script returns the saved script corresponding to id.
	Script(ctx context.Context, id uint) (*Script, error)

	// GetScriptContents returns the raw script contents of the corresponding
	// script.
	GetScriptContents(ctx context.Context, id uint) ([]byte, error)

	// GetAnyScriptContents returns the raw script contents of the corresponding
	// script, regardless whether it is present in the scripts table.
	GetAnyScriptContents(ctx context.Context, id uint) ([]byte, error)

	// DeleteScript deletes the script identified by its id.
	DeleteScript(ctx context.Context, id uint) error

	// ListScripts returns a paginated list of scripts corresponding to the
	// criteria.
	ListScripts(ctx context.Context, teamID *uint, opt ListOptions) ([]*Script, *PaginationMetadata, error)

	// GetScriptIDByName returns the id of the script with the given name and team id.
	GetScriptIDByName(ctx context.Context, name string, teamID *uint) (uint, error)

	// GetHostScriptDetails returns the list of host script details for saved scripts applicable to
	// a given host.
	GetHostScriptDetails(ctx context.Context, hostID uint, teamID *uint, opts ListOptions, hostPlatform string) ([]*HostScriptDetail, *PaginationMetadata, error)

	// BatchSetScripts sets the scripts for the given team or no team.
	BatchSetScripts(ctx context.Context, tmID *uint, scripts []*Script) ([]ScriptResponse, error)

	// BatchExecuteScript queues a script to run on a set of hosts and returns the batch script
	// execution ID.
	BatchExecuteScript(ctx context.Context, userID *uint, scriptID uint, hostIDs []uint) (string, error)

	// BatchExecuteScript queued a script to run on a set of hosts after notBefore and returns the
	// batch execution ID.
	BatchScheduleScript(ctx context.Context, userID *uint, scriptID uint, hostIDs []uint, notBefore time.Time) (string, error)

	// GetBatchActivity returns a batch activity with executionID
	GetBatchActivity(ctx context.Context, executionID string) (*BatchActivity, error)

	// GetBatchActivityHostResults returns all host results associated with batch executionID
	GetBatchActivityHostResults(ctx context.Context, executionID string) ([]*BatchActivityHostResult, error)

	// RunScheduledBatchActivity takes a scheduled batch script avtivity and executes it, queueing
	// the script on the hosts. Note that it does not know about the `not_before` column on the jobs
	// table and assumes it is being executed at the right time.
	RunScheduledBatchActivity(ctx context.Context, executionID string) error

	// BatchExecuteSummary returns the summary of a batch script execution
	BatchExecuteSummary(ctx context.Context, executionID string) (*BatchActivity, error)

	// CancelBatchScript cancels the execution of a batch script execution
	CancelBatchScript(ctx context.Context, executionID string) error

	// ListBatchScriptExecutions returns a filtered list of batch script executions, with summaries.
	ListBatchScriptExecutions(ctx context.Context, filter BatchExecutionStatusFilter) ([]BatchActivity, error)

	// CountBatchScriptExecutions returns the number of batch script executions matching the filter.
	CountBatchScriptExecutions(ctx context.Context, filter BatchExecutionStatusFilter) (int64, error)

	// MarkActivitiesAsCompleted updates the status of the specified activities to "completed".
	MarkActivitiesAsCompleted(ctx context.Context) error

	// GetHostLockWipeStatus gets the lock/unlock and wipe status for the host.
	GetHostLockWipeStatus(ctx context.Context, host *Host) (*HostLockWipeStatus, error)

	// GetHostsLockWipeStatusBatch gets the lock/unlock and wipe status for multiple hosts in a single operation.
	// Returns a map of host ID to HostLockWipeStatus.
	GetHostsLockWipeStatusBatch(ctx context.Context, hosts []*Host) (map[uint]*HostLockWipeStatus, error)

	// LockHostViaScript sends a script to lock a host and updates the
	// states in host_mdm_actions
	LockHostViaScript(ctx context.Context, request *HostScriptRequestPayload, hostFleetPlatform string) error

	// UnlockHostViaScript sends a script to unlock a host and updates the
	// states in host_mdm_actions
	UnlockHostViaScript(ctx context.Context, request *HostScriptRequestPayload, hostFleetPlatform string) error

	// UnlockHostmanually records a request to unlock a host that requires manual
	// intervention (such as for macOS). It indicates the an unlock request is
	// pending.
	UnlockHostManually(ctx context.Context, hostID uint, hostFleetPlatform string, ts time.Time) error

	// CleanAppleMDMLock cleans the lock status and pin for a macOS device
	// after it has been unlocked. 	CleanAppleMDMLock will be a no-op when
	// unlock_ref was set within the last 5 minutes, to prevent the trailing
	// Idle (sent right after the device acknowledges the lock command)
	// from prematurely clearing the lock state.
	CleanAppleMDMLock(ctx context.Context, hostUUID string) error

	InsertHostLocationData(ctx context.Context, locData HostLocationData) error
	// GetHostLocationData gets the given host's location data from the Fleet database, if it exists.
	GetHostLocationData(ctx context.Context, hostID uint) (*HostLocationData, error)
	DeleteHostLocationData(ctx context.Context, hostID uint) error

	// CleanupUnusedScriptContents will remove script contents that have no references to them from
	// the scripts or host_script_results tables.
	CleanupUnusedScriptContents(ctx context.Context) error
	// CleanupExpiredLiveQueries cleans up unsaved queries older than the given expiration window (in days),
	// orphaned distributed query campaigns that reference non-existing queries, and orphaned campaign targets that reference non-existing campaigns.
	CleanupExpiredLiveQueries(ctx context.Context, expiryWindowDays int) error
	// WipeHostViaScript sends a script to wipe a host and updates the
	// states in host_mdm_actions.
	WipeHostViaScript(ctx context.Context, request *HostScriptRequestPayload, hostFleetPlatform string) error

	// WipeHostViaWindowsMDM sends a Windows MDM command to wipe a host and
	// updates the states in host_mdm_actions.
	WipeHostViaWindowsMDM(ctx context.Context, host *Host, cmd *MDMWindowsCommand) error

	// UpdateHostLockWipeStatusFromAppleMDMResult updates the host_mdm_actions
	// table to reflect the result of the corresponding lock/wipe MDM command for
	// Apple hosts. It is optimized to update using only the information
	// available in the Apple MDM protocol.
	UpdateHostLockWipeStatusFromAppleMDMResult(ctx context.Context, hostUUID, cmdUUID, requestType string, succeeded bool) error

	///////////////////////////////////////////////////////////////////////////////
	// Software installers
	//

	// GetIncludedHostIDMapForSoftwareInstaller gets the set of hosts that are targeted/in scope for the
	// given software installer, based on label membership.
	GetIncludedHostIDMapForSoftwareInstaller(ctx context.Context, installerID uint) (map[uint]struct{}, error)

	// GetExcludedHostIDMapForSoftwareInstaller gets the set of hosts that are NOT targeted/in scope for the
	// given software installer, based on label membership.
	GetExcludedHostIDMapForSoftwareInstaller(ctx context.Context, installerID uint) (map[uint]struct{}, error)

	// ClearSoftwareInstallerAutoInstallPolicyStatusForHosts clears out the status of the policy related to the given
	// software installer for all the given hosts.
	ClearSoftwareInstallerAutoInstallPolicyStatusForHosts(ctx context.Context, installerID uint, hostIDs []uint) error

	// GetSoftwareInstallDetails returns details required to fetch and
	// run software installers
	GetSoftwareInstallDetails(ctx context.Context, executionId string) (*SoftwareInstallDetails, error)
	// ListPendingSoftwareInstalls returns a list of software
	// installer execution IDs that have not yet been run for a given host
	ListPendingSoftwareInstalls(ctx context.Context, hostID uint) ([]string, error)
	// ListReadyToExecuteSoftwareInstalls is like ListPendingSoftwareInstalls
	// except that it only returns software installs that are ready to execute
	// ("activated" in the upcoming activities queue, available for orbit to
	// process).
	ListReadyToExecuteSoftwareInstalls(ctx context.Context, hostID uint) ([]string, error)

	// GetHostLastInstallData returns the data for the last installation of a package on a host.
	GetHostLastInstallData(ctx context.Context, hostID, installerID uint) (*HostLastInstallData, error)

	// MatchOrCreateSoftwareInstaller matches or creates a new software installer.
	MatchOrCreateSoftwareInstaller(ctx context.Context, payload *UploadSoftwareInstallerPayload) (installerID, titleID uint, err error)

	// GetSoftwareInstallerMetadataByID returns the software installer corresponding to the installer id.
	GetSoftwareInstallerMetadataByID(ctx context.Context, id uint) (*SoftwareInstaller, error)

	// ValidateSoftwareInstallerAccess checks if a host has access to
	// an installer. Access is granted if there is currently an unfinished
	// install request present in host_software_installs
	ValidateOrbitSoftwareInstallerAccess(ctx context.Context, hostID uint, installerID uint) (bool, error)

	// GetSoftwareInstallerMetadataByTeamAndTitleID returns the software
	// installer corresponding to the specified team and title ids. If
	// withScriptContents is true, also returns the contents of the install and
	// (if set) post-install scripts, otherwise those fields are left empty.
	GetSoftwareInstallerMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*SoftwareInstaller, error)

	// GetFleetMaintainedVersionsByTitleID returns all cached versions of a
	// fleet-maintained app for the given title and team. If byVersion is true
	// the versions will be sorted by the version string.
	GetFleetMaintainedVersionsByTitleID(ctx context.Context, teamID *uint, titleID uint, byVersion bool) ([]FleetMaintainedVersion, error)

	// HasFMAInstallerVersion returns true if the given FMA version is already
	// cached as a software installer for the given team.
	HasFMAInstallerVersion(ctx context.Context, teamID *uint, fmaID uint, version string) (bool, error)

	// GetCachedFMAInstallerMetadata returns the cached metadata for a specific
	// FMA installer version, including install/uninstall scripts, URL, SHA256,
	// etc. Returns a NotFoundError if no cached installer exists for the given
	// version.
	GetCachedFMAInstallerMetadata(ctx context.Context, teamID *uint, fmaID uint, version string) (*MaintainedApp, error)

	InsertHostInHouseAppInstall(ctx context.Context, hostID uint, inHouseAppID, softwareTitleID uint, commandUUID string, opts HostSoftwareInstallOptions) error

	// GetSoftwareInstallersPendingUninstallScriptPopulation returns a map of software installers to storage IDs that:
	// 1. need uninstall scripts populated
	// 2. can have uninstall scripts auto-generated by Fleet
	GetSoftwareInstallersPendingUninstallScriptPopulation(ctx context.Context) (map[uint]string, error)

	// GetMSIInstallersWithoutUpgradeCode returns a map of MSI software installers to storage ids that do not have an upgrade code set.
	GetMSIInstallersWithoutUpgradeCode(ctx context.Context) (map[uint]string, error)

	// UpdateSoftwareInstallerWithoutPackageIDs updates the software installer corresponding to the id. Used to add uninstall scripts.
	UpdateSoftwareInstallerWithoutPackageIDs(ctx context.Context, id uint, payload UploadSoftwareInstallerPayload) error

	// UpdateInstallerUpgradeCode updates the software installer corresponding to the id. Used to add upgrade codes.
	UpdateInstallerUpgradeCode(ctx context.Context, id uint, upgradeCode string) error

	// ProcessInstallerUpdateSideEffects handles, in a transaction, the following based on whether metadata
	// or package are dirty:
	// 1. If metadata or package were updated, removes host_software_installer and queued script records for
	// pending non-VPP installs and uninstalls for an installer by its ID. See implementation for caveats.
	// 2. If package was updated, marks host software installer rows for the supplied installer
	// as removed, hiding them from stats calculations (note that this will null out installer statuses due
	// to how the virtual column works).
	ProcessInstallerUpdateSideEffects(ctx context.Context, installerID uint, wasMetadataUpdated bool, wasPackageUpdated bool) error

	// SaveInstallerUpdates persists new values to an existing installer. See comments in the payload struct
	// for which fields must be set.
	SaveInstallerUpdates(ctx context.Context, payload *UpdateSoftwareInstallerPayload) error

	// UpdateInstallerSelfServiceFlag sets an installer's self-service flag without modifying anything else
	UpdateInstallerSelfServiceFlag(ctx context.Context, selfService bool, id uint) error

	GetVPPAppByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*VPPApp, error)
	// GetVPPAppMetadataByTeamAndTitleID returns the VPP app corresponding to the
	// specified team and title ids.
	GetVPPAppMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*VPPAppStoreApp, error)

	// MapAdamIDsPendingInstall gets App Store IDs of VPP apps pending install for a host
	MapAdamIDsPendingInstall(ctx context.Context, hostID uint) (map[string]struct{}, error)

	// MapAdamIDsPendingInstallVerification gets Apps Store IDs of VPP apps pending verifications
	// on VPP installations for a host
	//
	// By pending verification it means that the installation command is not acknowledged, OR that the installation
	// is acknowledged but not yet verified (installation is still ongoing on the device and/or Fleet hasn't verified
	// the installation via InstalledApplicationList).
	MapAdamIDsPendingInstallVerification(ctx context.Context, hostID uint) (adamIDs map[string]struct{}, err error)

	// MapAdamIDsRecentInstalls returns a set of Adam IDs for the host that have been installed within the provided seconds.
	MapAdamIDsRecentInstalls(ctx context.Context, hostID uint, seconds int) (adamIDs map[string]struct{}, err error)

	// GetTitleInfoFromVPPAppsTeamsID returns title ID and VPP app name corresponding to the supplied team VPP app PK
	GetTitleInfoFromVPPAppsTeamsID(ctx context.Context, vppAppsTeamsID uint) (*PolicySoftwareTitle, error)

	// GetVPPAppMetadataByAdamIDPlatformTeamID returns the VPP app correspoding to the specified
	// ADAM ID, platform within the context of the specified team. It includes the vpp_app_team_id value.
	GetVPPAppMetadataByAdamIDPlatformTeamID(ctx context.Context, adamID string, platform InstallableDevicePlatform, teamID *uint) (*VPPApp, error)

	// DeleteSoftwareInstaller deletes the software installer corresponding to the id.
	DeleteSoftwareInstaller(ctx context.Context, id uint) error

	// DeleteVPPAppFromTeam deletes the VPP app corresponding to the adamID from
	// the provided team.
	DeleteVPPAppFromTeam(ctx context.Context, teamID *uint, appID VPPAppID) error

	GetAndroidAppsInScopeForHost(ctx context.Context, hostID uint) (applicationIDs []string, err error)

	// GetSummaryHostSoftwareInstalls returns the software install summary for
	// the given software installer id.
	GetSummaryHostSoftwareInstalls(ctx context.Context, installerID uint) (*SoftwareInstallerStatusSummary, error)

	// GetSummaryHostVPPAppInstalls returns the VPP app install summary for the
	// given team and VPP app adam_id.
	GetSummaryHostVPPAppInstalls(ctx context.Context, teamID *uint, appID VPPAppID) (*VPPAppStatusSummary, error)

	GetSoftwareInstallResults(ctx context.Context, resultsUUID string) (*HostSoftwareInstallerResult, error)

	// CleanupUnusedSoftwareInstallers will remove software installers that have
	// no references to them from the software_installers table.
	CleanupUnusedSoftwareInstallers(ctx context.Context, softwareInstallStore SoftwareInstallerStore, removeCreatedBefore time.Time) error

	// SaveInHouseAppUpdates persists new values to an existing in house app.
	SaveInHouseAppUpdates(ctx context.Context, payload *UpdateSoftwareInstallerPayload) error

	// GetInHouseAppMetadataByTeamAndTitleID returns the in house app corresponding to the specific team and title ids.
	GetInHouseAppMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*SoftwareInstaller, error)

	// Remove host inhouseapp installs and upcoming inhouseapp install activities
	RemovePendingInHouseAppInstalls(ctx context.Context, inHouseAppID uint) error

	// GetSummaryHostSoftwareInstalls returns the software install summary for the in house app ID
	GetSummaryHostInHouseAppInstalls(ctx context.Context, teamID *uint, inHouseAppID uint) (*VPPAppStatusSummary, error)

	// DeleteInHouseApp deletes an in house app and removes pending installs for it
	DeleteInHouseApp(ctx context.Context, id uint) error

	// CleanupUnusedSoftwareTitleIcons will remove software title icons that have
	// no references to them from the software_title_icons table.
	CleanupUnusedSoftwareTitleIcons(ctx context.Context, softwareTitleIconStore SoftwareTitleIconStore, removeCreatedBefore time.Time) error

	// BatchSetSoftwareInstallers sets the software installers for the given team or no team.
	BatchSetSoftwareInstallers(ctx context.Context, tmID *uint, installers []*UploadSoftwareInstallerPayload) error
	// BatchSetInHouseAppsInstallers sets the in-house apps installers for the given team or no team.
	BatchSetInHouseAppsInstallers(ctx context.Context, tmID *uint, installers []*UploadSoftwareInstallerPayload) error
	GetSoftwareInstallers(ctx context.Context, tmID uint) ([]SoftwarePackageResponse, error)

	// HasSelfServiceSoftwareInstallers returns true if self-service software installers are available for the team or globally.
	HasSelfServiceSoftwareInstallers(ctx context.Context, platform string, teamID *uint) (bool, error)

	CreateOrUpdateSoftwareTitleIcon(ctx context.Context, payload *UploadSoftwareTitleIconPayload) (*SoftwareTitleIcon, error)
	GetSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) (*SoftwareTitleIcon, error)
	GetTeamIdsForIconStorageId(ctx context.Context, storageID string) ([]uint, error)
	GetSoftwareIconsByTeamAndTitleIds(ctx context.Context, teamID uint, titleIDs []uint) (map[uint]SoftwareTitleIcon, error)
	DeleteSoftwareTitleIcon(ctx context.Context, teamID, titleID uint) error
	DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx context.Context, teamID uint) error
	ActivityDetailsForSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) (DetailsForSoftwareIconActivity, error)

	BatchInsertVPPApps(ctx context.Context, apps []*VPPApp) error
	GetAssignedVPPApps(ctx context.Context, teamID *uint) (map[VPPAppID]VPPAppTeam, error)
	GetVPPApps(ctx context.Context, teamID *uint) ([]VPPAppResponse, error)
	SetTeamVPPApps(ctx context.Context, teamID *uint, appIDs []VPPAppTeam, appStoreAppIDsToTitleIDs map[string]uint) (bool, error)
	InsertVPPAppWithTeam(ctx context.Context, app *VPPApp, teamID *uint) (*VPPApp, error)
	GetVPPAppsToInstallDuringSetupExperience(ctx context.Context, teamID *uint, platform string) ([]string, error)

	// GetAllVPPApps returns all the VPP apps in Fleet, across all teams.
	GetAllVPPApps(ctx context.Context) ([]*VPPApp, error)
	// InsertVPPApps inserts the given VPP apps in the database.
	InsertVPPApps(ctx context.Context, apps []*VPPApp) error

	// InsertHostVPPSoftwareInstall(ctx context.Context, hostID uint, appID VPPAppID, commandUUID, associatedEventID string, selfService bool, policyID *uint) error
	InsertHostVPPSoftwareInstall(ctx context.Context, hostID uint, appID VPPAppID, commandUUID, associatedEventID string, opts HostSoftwareInstallOptions) error
	GetPastActivityDataForVPPAppInstall(ctx context.Context, commandResults *mdm.CommandResults) (*User, *ActivityInstalledAppStoreApp, error)
	// GetVPPAppInstallStatusByCommandUUID returns whether the VPP app from the given install command
	// is currently installed. Returns false if the command doesn't exist or app is not installed.
	GetVPPAppInstallStatusByCommandUUID(ctx context.Context, commandUUID string) (bool, error)
	// IsAutoUpdateVPPInstall determines whether a VPP install command was triggered by auto-update config
	IsAutoUpdateVPPInstall(ctx context.Context, commandUUID string) (bool, error)

	GetVPPTokenByLocation(ctx context.Context, loc string) (*VPPTokenDB, error)

	// GetIncludedHostIDMapForVPPApp gets the set of hosts that are targeted/in scope for the
	// given VPP app, based on label membership.
	GetIncludedHostIDMapForVPPApp(ctx context.Context, vppAppTeamID uint) (map[uint]struct{}, error)

	GetIncludedHostUUIDMapForAppStoreApp(ctx context.Context, vppAppTeamID uint) (map[string]string, error)

	// GetExcludedHostIDMapForVPPApp gets the set of hosts that are NOT targeted/in scope for the
	// given VPP app, based on label membership.
	GetExcludedHostIDMapForVPPApp(ctx context.Context, vppAppTeamID uint) (map[uint]struct{}, error)

	// ClearVPPAppAutoInstallPolicyStatusForHosts clears out the status of the policy related to the given
	// VPP app for all the given hosts.
	ClearVPPAppAutoInstallPolicyStatusForHosts(ctx context.Context, vppAppTeamID uint, hostIDs []uint) error

	////////////////////////////////////////////////////////////////////////////////////
	// Setup Experience
	//

	SetSetupExperienceSoftwareTitles(ctx context.Context, platform string, teamID uint, titleIDs []uint) error
	ListSetupExperienceSoftwareTitles(ctx context.Context, platform string, teamID uint, opts ListOptions) ([]SoftwareTitleListResult, int, *PaginationMetadata, error)

	// SetHostAwaitingConfiguration sets a boolean indicating whether or not the given host is
	// in the setup experience flow (which runs during macOS Setup Assistant).
	SetHostAwaitingConfiguration(ctx context.Context, hostUUID string, inSetupExperience bool) error
	// GetHostAwaitingConfiguration returns a boolean indicating whether or not the given host is
	// in the setup experience flow (which runs during macOS Setup Assistant).
	GetHostAwaitingConfiguration(ctx context.Context, hostUUID string) (bool, error)

	// GetTeamsWithInstallerByHash gets a map of teamIDs (0 for No team) to software installers
	// metadata by the installer's hash.
	GetTeamsWithInstallerByHash(ctx context.Context, sha256, url string) (map[uint][]*ExistingSoftwareInstaller, error)

	// GetInstallerByTeamAndURL looks up an existing software installer by URL.
	// When teamID is non-nil, filters to that team. When nil, searches all teams
	// (cross-team fallback). Returns the most recently inserted active installer
	// matching the URL, including its storage_id and http_etag for conditional
	// downloads.
	GetInstallerByTeamAndURL(ctx context.Context, teamID *uint, url string) (*ExistingSoftwareInstaller, error)

	// TeamIDsWithSetupExperienceIdPEnabled returns the list of team IDs that
	// have the setup experience IdP (End user authentication) enabled. It uses
	// id 0 to represent "No team", should IdP be enabled for that team.
	TeamIDsWithSetupExperienceIdPEnabled(ctx context.Context) ([]uint, error)

	// ListSetupExperienceResultsByHostUUID lists the setup experience results for a host by its UUID.
	ListSetupExperienceResultsByHostUUID(ctx context.Context, hostUUID string, teamID uint) ([]*SetupExperienceStatusResult, error)

	// UpdateSetupExperienceStatusResult updates the given setup experience status result.
	UpdateSetupExperienceStatusResult(ctx context.Context, status *SetupExperienceStatusResult) error

	// EnqueueSetupExperienceItems enqueues the relevant setup experience items (software and
	// script) for a given host. It first clears out any pre-existing setup experience items that
	// were previously enqueued for the host (since the setup experience only happens once during
	// the initial device setup). It then adds any software and script that have been configured for
	// this team to the host's queue and sets their status to pending. If any items were enqueued,
	// it returns true, otherwise it returns false.
	//
	// It uses hostPlatformLike to cover scenarios where software items are not compatible with the target
	// platform. E.g. "deb" packages can only be queued for hosts with platform_like = "debian" (Ubuntu, Debian, etc.).
	// MacOS hosts have hosts.platform_like = 'darwin', Ubuntu and Debian hosts have hosts.platform_like = 'debian'
	// Fedora hosts have hosts.platform_like = 'rhel'. The hostPlatform argument (e.g. "darwin", "arch", "ubuntu", etc.)
	// is used for some validations, and to backfill hostPlatformLike if empty.
	EnqueueSetupExperienceItems(ctx context.Context, hostPlatform, hostPlatformLike, hostUUID string, teamID uint) (bool, error)

	// ResetSetupExperienceItemsAfterFailure resets any setup experience items that were canceled after
	// a software item failed to install on a host whose team was configured to stop setup experience on failure.
	ResetSetupExperienceItemsAfterFailure(ctx context.Context, hostPlatform, hostPlatformLike, hostUUID string, teamID uint) (bool, error)

	// CancelPendingSetupExperienceSteps cancels any setup experience items for the given host that aren't already completed.
	CancelPendingSetupExperienceSteps(ctx context.Context, hostUUID string) error

	// GetSetupExperienceScript gets the setup experience script for a team. There can only be 1
	// setup experience script per team.
	GetSetupExperienceScript(ctx context.Context, teamID *uint) (*Script, error)

	// GetSetupExperienceScriptByID gets the setup experience script by its ID.
	GetSetupExperienceScriptByID(ctx context.Context, scriptID uint) (*Script, error)

	// SetSetupExperienceScript sets the setup experience script to the given script.
	SetSetupExperienceScript(ctx context.Context, script *Script) error

	// DeleteSetupExperienceScript deletes the setup experience script for the given team.
	DeleteSetupExperienceScript(ctx context.Context, teamID *uint) error

	// MaybeUpdateSetupExperienceScriptStatus updates the status of the setup experience script for
	// the given host if the script result row exists. If there was an update, it returns true.
	// Otherwise, it returns false.
	MaybeUpdateSetupExperienceScriptStatus(ctx context.Context, hostUUID string, executionID string, status SetupExperienceStatusResultStatus) (bool, error)

	// MaybeUpdateSetupExperienceSoftwareInstallStatus updates the status of the setup experience
	// software installer for the given host if the software installer result row exists. If there
	// was an update, it returns true. Otherwise, it returns false.
	MaybeUpdateSetupExperienceSoftwareInstallStatus(ctx context.Context, hostUUID string, executionID string, status SetupExperienceStatusResultStatus) (bool, error)

	// MaybeUpdateSetupExperienceVPPStatus updates the status of the setup experience
	// VPP app for the given host if the VPP app installer row exists. If there was an update, it
	// returns true. Otherwise, it returns false.
	MaybeUpdateSetupExperienceVPPStatus(ctx context.Context, hostUUID string, commandUUID string, status SetupExperienceStatusResultStatus) (bool, error)

	// Fleet-maintained apps
	//

	// ListAvailableFleetMaintainedApps returns a list of Fleet-maintained apps, including software title ID if
	// either the maintained app or a custom package/VPP app for the same app is installed on the specified team,
	// if a team is specified.
	ListAvailableFleetMaintainedApps(ctx context.Context, teamID *uint, opt ListOptions) ([]MaintainedApp, *PaginationMetadata, error)

	// ClearRemovedFleetMaintainedApps deletes all Fleet-maintained apps that are not in the given
	// set of slugs.
	ClearRemovedFleetMaintainedApps(ctx context.Context, slugsToKeep []string) error

	// GetSetupExperienceCount returns the number of installers, vpp apps, and scritps available for
	// a team and platform
	GetSetupExperienceCount(ctx context.Context, platform string, teamID *uint) (*SetupExperienceCount, error)

	// GetMaintainedAppByID gets a Fleet-maintained app by its ID, including software title ID if
	// either the maintained app or a custom package/VPP app for the same app is installed on the specified team,
	// if a team is specified.
	GetMaintainedAppByID(ctx context.Context, appID uint, teamID *uint) (*MaintainedApp, error)

	// GetMaintainedAppBySlug gets a Fleet-maintained app by its slug
	GetMaintainedAppBySlug(ctx context.Context, slug string, teamID *uint) (*MaintainedApp, error)

	// UpsertMaintainedApp inserts or updates a maintained app using the updated
	// metadata provided via app.
	UpsertMaintainedApp(ctx context.Context, app *MaintainedApp) (*MaintainedApp, error)

	// GetFMANamesByIdentifier returns a map of unique_identifier -> canonical name
	// for all Fleet-maintained apps on macOS. This is used during software ingestion
	// to use the FMA name instead of the osquery-reported name.
	GetFMANamesByIdentifier(ctx context.Context) (map[string]string, error)

	// /////////////////////////////////////////////////////////////////////////////
	// Certificate management

	// BulkUpsertMDMManagedCertificates updates metadata regarding certificates on the host.
	BulkUpsertMDMManagedCertificates(ctx context.Context, payload []*MDMManagedCertificate) error

	// GetAppleHostMDMCertificateProfile returns the MDM profile information for the specified host UUID and profile UUID.
	// nil is returned if the profile is not found.
	GetAppleHostMDMCertificateProfile(ctx context.Context, hostUUID string, profileUUID string, caName string) (*HostMDMCertificateProfile, error)
	// GetWindowsHostMDMCertificateProfile returns the MDM profile information for the specified host UUID and profile UUID.
	// nil is returned if the profile is not found.
	GetWindowsHostMDMCertificateProfile(ctx context.Context, hostUUID string, profileUUID string, caName string) (*HostMDMCertificateProfile, error)

	// CleanUpMDMManagedCertificates removes all managed certificates that are not associated with any host+profile.
	CleanUpMDMManagedCertificates(ctx context.Context) error

	// RenewMDMManagedCertificates marks managed certificate profiles for resend when renewal is required
	RenewMDMManagedCertificates(ctx context.Context) error

	// ListHostMDMManagedCertificates returns the managed certificates for the given host UUID
	ListHostMDMManagedCertificates(ctx context.Context, hostUUID string) ([]*MDMManagedCertificate, error)

	// ResendHostCertificateProfile marks the given profile UUID to be resent to the host with the given UUID. It
	// also deactivates prior nano commands and resets the retry counter for the profile UUID and host UUID.
	ResendHostCertificateProfile(ctx context.Context, hostUUID string, profUUID string) error

	// /////////////////////////////////////////////////////////////////////////////
	// Secret variables

	// UpsertSecretVariables inserts or updates secret variables in the database.
	UpsertSecretVariables(ctx context.Context, secretVariables []SecretVariable) error

	// CreateSecretVariable inserts a secret variable (value encrypted) and returns its ID.
	// Returns an AlreadyExistsError error if there's already a secret variable with the same name.
	CreateSecretVariable(ctx context.Context, name string, value string) (id uint, err error)
	// ListSecretVariables returns a list of secret variable identifiers filtered with the provided sorting and pagination options.
	// Returns a count of total secret variable identifiers on all (filtered) pages, and pagination metadata if opt.IncludeMetadata is true.
	ListSecretVariables(ctx context.Context, opt ListOptions) (secretVariables []SecretVariableIdentifier, meta *PaginationMetadata, count int, err error)
	// DeleteSecretVariable deletes a secret variable by ID and returns the name of the deleted variable.
	// Returns a NotFoundError error if there's no secret variable with such ID.
	DeleteSecretVariable(ctx context.Context, id uint) (name string, err error)

	// GetSecretVariables retrieves secret variables from the database that match the given names.
	GetSecretVariables(ctx context.Context, names []string) ([]SecretVariable, error)

	// ValidateEmbeddedSecrets parses fleet secrets from a list of
	// documents and checks that they exist in the database.
	ValidateEmbeddedSecrets(ctx context.Context, documents []string) error

	// ExpandEmbeddedSecrets expands the fleet secrets in a
	// document using the secrets stored in the datastore.
	ExpandEmbeddedSecrets(ctx context.Context, document string) (string, error)

	// ExpandEmbeddedSecretsAndUpdatedAt is like ExpandEmbeddedSecrets but also
	// returns the latest updated_at time of the secrets used in the expansion.
	ExpandEmbeddedSecretsAndUpdatedAt(ctx context.Context, document string) (string, *time.Time, error)

	// ExpandHostSecrets expands host-scoped secrets ($FLEET_HOST_SECRET_*) in the document.
	// The enrollmentID (typically UDID) is used to look up host-specific secrets
	// like recovery lock passwords.
	ExpandHostSecrets(ctx context.Context, document string, enrollmentID string) (string, error)

	// /////////////////////////////////////////////////////////////////////////////
	// Android

	AndroidDatastore

	// NewMDMAndroidConfigProfile creates a new Android MDM config profile.
	NewMDMAndroidConfigProfile(ctx context.Context, cp MDMAndroidConfigProfile) (*MDMAndroidConfigProfile, error)

	// GetMDMAndroidConfigProfile returns the Android MDM profile corresponding
	// to the specified profile uuid.
	GetMDMAndroidConfigProfile(ctx context.Context, profileUUID string) (*MDMAndroidConfigProfile, error)

	// DeleteMDMAndroidConfigProfile deletes the Android MDM profile corresponding to
	// the specified profile uuid.
	DeleteMDMAndroidConfigProfile(ctx context.Context, profileUUID string) error

	// GetMDMAndroidProfilesSummary summarizes the current state of Android profiles on each
	// Android host in the specified team (or, if no team is specified, each host that is not
	// assigned to any team).
	GetMDMAndroidProfilesSummary(ctx context.Context, teamID *uint) (*MDMProfilesSummary, error)

	// GetHostCertificateTemplates returns what certificate templates are currently associated with the specified host.
	GetHostCertificateTemplates(ctx context.Context, hostUUID string) ([]HostCertificateTemplate, error)

	// CreatePendingCertificateTemplatesForExistingHosts creates pending certificate template records
	// for all enrolled Android hosts in the team when a new certificate template is added.
	CreatePendingCertificateTemplatesForExistingHosts(ctx context.Context, certificateTemplateID uint, teamID uint) (int64, error)

	// CreatePendingCertificateTemplatesForNewHost creates pending certificate template records
	// for a newly enrolled Android host based on their team's certificate templates.
	CreatePendingCertificateTemplatesForNewHost(ctx context.Context, hostUUID string, teamID uint) (int64, error)

	// RevertStaleCertificateTemplates reverts certificate templates stuck in 'delivering' status
	// for longer than the specified duration back to 'pending'.
	RevertStaleCertificateTemplates(ctx context.Context, staleDuration time.Duration) (int64, error)

	// GetHostMDMAndroidProfiles retrieves the Android MDM profiles for a specific host.
	GetHostMDMAndroidProfiles(ctx context.Context, hostUUID string) ([]HostMDMAndroidProfile, error)

	// NewAndroidPolicyRequest saves details about a new Android AMAPI request.
	NewAndroidPolicyRequest(ctx context.Context, req *android.MDMAndroidPolicyRequest) error

	// GetAndroidPolicyRequestByUUID retrieves an Android policy request by ID.
	GetAndroidPolicyRequestByUUID(ctx context.Context, requestUUID string) (*android.MDMAndroidPolicyRequest, error)

	// ListMDMAndroidProfilesToSend lists the Android hosts that need to have
	// their configuration profiles (Android policy) sent. It returns two lists,
	// the list of profiles to apply and the list of profiles to remove.
	ListMDMAndroidProfilesToSend(ctx context.Context) ([]*MDMAndroidProfilePayload, []*MDMAndroidProfilePayload, error)

	// GetMDMAndroidProfilesContents retrieves the contents of the Android
	// profiles with the specified UUIDs.
	GetMDMAndroidProfilesContents(ctx context.Context, uuids []string) (map[string]json.RawMessage, error)

	// ListAndroidEnrolledDevicesForReconcile returns the list of Android devices
	// that are currently marked as enrolled in Fleet (host_mdm.enrolled=1).
	// It returns a minimal device struct with host and device identifiers.
	ListAndroidEnrolledDevicesForReconcile(ctx context.Context) ([]*android.Device, error)

	// InsertAndroidSetupExperienceSoftwareInstall inserts a new Android
	// VPP app install record for the setup experience flow.
	InsertAndroidSetupExperienceSoftwareInstall(ctx context.Context, payload *HostAndroidVPPSoftwareInstall) error

	// GetAndroidAppConfiguration retrieves the configuration for an Android app by application ID and team
	GetAndroidAppConfiguration(ctx context.Context, applicationID string, teamID uint) ([]byte, error)
	GetAndroidAppConfigurationByAppTeamID(ctx context.Context, vppAppTeamID uint) ([]byte, error)
	HasAndroidAppConfigurationChanged(ctx context.Context, applicationID string, teamID uint, newConfig []byte) (bool, error)

	SetAndroidAppInstallPendingApplyConfig(ctx context.Context, hostUUID, applicationID string, policyVersion int64) error

	// BulkGetAndroidAppConfigurations retrieves Android app configurations for
	// all provided apps and returns them indexed by the app id.
	BulkGetAndroidAppConfigurations(ctx context.Context, appIDs []string, teamID uint) (map[string][]byte, error)

	// DeleteAndroidAppConfiguration removes an Android app configuration.
	DeleteAndroidAppConfiguration(ctx context.Context, adamID string, teamID uint) error

	ListMDMAndroidUUIDsToHostIDs(ctx context.Context, hostIDs []uint) (map[string]uint, error)

	// VPP App Configuration (iOS/iPadOS)
	GetVPPAppConfiguration(ctx context.Context, platform InstallableDevicePlatform, adamID string, teamID uint) ([]byte, error)
	HasVPPAppConfigurationChanged(ctx context.Context, platform InstallableDevicePlatform, adamID string, teamID uint, newConfig []byte) (bool, error)
	BulkGetVPPAppConfigurations(ctx context.Context, platform InstallableDevicePlatform, adamIDs []string, teamID uint) (map[string][]byte, error)
	DeleteVPPAppConfiguration(ctx context.Context, platform InstallableDevicePlatform, adamID string, teamID uint) error

	// In-House App Configuration (iOS/iPadOS).
	GetInHouseAppConfiguration(ctx context.Context, inHouseAppID uint) ([]byte, error)
	HasInHouseAppConfigurationChanged(ctx context.Context, inHouseAppID uint, newConfig []byte) (bool, error)
	BulkGetInHouseAppConfigurations(ctx context.Context, inHouseAppIDs []uint) (map[uint][]byte, error)
	DeleteInHouseAppConfiguration(ctx context.Context, inHouseAppID uint) error

	// /////////////////////////////////////////////////////////////////////////////
	// SCIM

	// CreateScimUser creates a new SCIM user in the database
	CreateScimUser(ctx context.Context, user *ScimUser) (uint, error)
	// ScimUserByID retrieves a SCIM user by ID
	ScimUserByID(ctx context.Context, id uint) (*ScimUser, error)
	// ScimUserByUserName retrieves a SCIM user by username
	ScimUserByUserName(ctx context.Context, userName string) (*ScimUser, error)
	// ScimUserByUserNameOrEmail finds a SCIM user by username. If it cannot find one, then it tries email, if set.
	// If multiple users are found with the same email, we log an error and return nil.
	// Emails and groups are NOT populated in this method.
	ScimUserByUserNameOrEmail(ctx context.Context, userName string, email string) (*ScimUser, error)
	// ScimUserByHostID retrieves a SCIM user associated with a host ID
	ScimUserByHostID(ctx context.Context, hostID uint) (*ScimUser, error)
	// ScimUsersExist checks if all the provided SCIM user IDs exist in the datastore
	// If the slice is empty, it returns true
	ScimUsersExist(ctx context.Context, ids []uint) (bool, error)
	// ReplaceScimUser replaces an existing SCIM user in the database
	ReplaceScimUser(ctx context.Context, user *ScimUser) error
	// DeleteScimUser deletes a SCIM user from the database
	DeleteScimUser(ctx context.Context, id uint) error
	// ListScimUsers retrieves a list of SCIM users with optional filtering
	ListScimUsers(ctx context.Context, opts ScimUsersListOptions) (users []ScimUser, totalResults uint, err error)
	// CreateScimGroup creates a new SCIM group in the database
	CreateScimGroup(ctx context.Context, group *ScimGroup) (uint, error)
	// ScimGroupByID retrieves a SCIM group by ID
	// If excludeUsers is true, the group's users will not be fetched
	ScimGroupByID(ctx context.Context, id uint, excludeUsers bool) (*ScimGroup, error)
	// ScimGroupByDisplayName retrieves a SCIM group by display name
	ScimGroupByDisplayName(ctx context.Context, displayName string) (*ScimGroup, error)
	// ReplaceScimGroup replaces an existing SCIM group in the database
	ReplaceScimGroup(ctx context.Context, group *ScimGroup) error
	// DeleteScimGroup deletes a SCIM group from the database
	DeleteScimGroup(ctx context.Context, id uint) error
	// ListScimGroups retrieves a list of SCIM groups with pagination
	ListScimGroups(ctx context.Context, opts ScimGroupsListOptions) (groups []ScimGroup, totalResults uint, err error)
	// ScimLastRequest retrieves the last SCIM request info
	ScimLastRequest(ctx context.Context) (*ScimLastRequest, error)
	// UpdateScimLastRequest updates the last SCIM request info
	UpdateScimLastRequest(ctx context.Context, lastRequest *ScimLastRequest) error
	// MaybeAssociateHostWithScimUser links a host with a SCIM user based on MDM IdP and IdP user ID
	MaybeAssociateHostWithScimUser(ctx context.Context, hostID uint) error

	// /////////////////////////////////////////////////////////////////////////////
	// Challenges

	// NewChallenge generates a random, base64-encoded challenge and inserts it into the challenges table.
	NewChallenge(ctx context.Context) (string, error)
	// ConsumeChallenge checks if a valid challenge exists in the challenges table
	// and deletes it if it does. The error will include sql.ErrNoRows if the challenge
	// is not found or is expired.
	ConsumeChallenge(ctx context.Context, challenge string) error
	// CleanupExpiredChallenges removes expired challenges from the challenges table,
	// intended to be run as a cron job.
	CleanupExpiredChallenges(ctx context.Context) (int64, error)

	// /////////////////////////////////////////////////////////////////////////////
	// Microsoft Compliance Partner

	// ConditionalAccessMicrosoftCreateIntegration creates the Conditional Access integration on the datastore.
	// The integration is created as "not done".
	// Currently only one integration can be configured, so this method replaces any existing integration.
	ConditionalAccessMicrosoftCreateIntegration(ctx context.Context, tenantID, proxyServerSecret string) error
	// ConditionalAccessMicrosoftGet returns the current Conditional Access integration.
	// Returns a NotFoundError error if there's none.
	ConditionalAccessMicrosoftGet(ctx context.Context) (*ConditionalAccessMicrosoftIntegration, error)
	// ConditionalAccessMicrosoftMarkSetupDone marks the configuration as done on the datastore.
	ConditionalAccessMicrosoftMarkSetupDone(ctx context.Context) error
	// ConditionalAccessMicrosoftDelete deletes the integration from the datastore.
	// It will also cleanup all recorded compliance status of all hosts from the datastore.
	ConditionalAccessMicrosoftDelete(ctx context.Context) error
	// LoadHostConditionalAccessStatus will load the current "Conditional Access" status of a host.
	// The status holds Entra's "Device ID", "User Principal Name", and last reported "managed" and "compliant" status.
	// Returns a NotFoundError error if there's no entry for the host.
	LoadHostConditionalAccessStatus(ctx context.Context, hostID uint) (*HostConditionalAccessStatus, error)
	// CreateHostConditionalAccessStatus creates the entry for the host on the datastore.
	// This does not set the "managed" or "compliant" status yet, this just creates the entry needed with Entra information.
	// If the host already has a different deviceID/userPrincipalName it will override them.
	CreateHostConditionalAccessStatus(ctx context.Context, hostID uint, deviceID string, userPrincipalName string) error
	// SetHostConditionalAccessStatus sets the "managed" and "compliant" statuses last set on Entra.
	// It does nothing if the host doesn't have a status entry created with CreateHostConditionalAccessStatus yet.
	SetHostConditionalAccessStatus(ctx context.Context, hostID uint, managed, compliant bool) error

	// /////////////////////////////////////////////////////////////////////////////
	// Host identity certificates

	// GetHostIdentityCertBySerialNumber gets the unrevoked valid cert corresponding to the provided serial number.
	GetHostIdentityCertBySerialNumber(ctx context.Context, serialNumber uint64) (*types.HostIdentityCertificate, error)
	// GetHostIdentityCertByName gets the unrevoked valid cert corresponding to the provided name (CN).
	GetHostIdentityCertByName(ctx context.Context, name string) (*types.HostIdentityCertificate, error)
	// UpdateHostIdentityCertHostIDBySerial updates the host ID associated with a certificate using its serial number.
	UpdateHostIdentityCertHostIDBySerial(ctx context.Context, serialNumber uint64, hostID uint) error
	// GetMDMSCEPCertBySerial looks up an MDM SCEP certificate by serial number and returns the device UUID.
	// This is used for iOS/iPadOS certificate-based authentication.
	GetMDMSCEPCertBySerial(ctx context.Context, serialNumber uint64) (deviceUUID string, err error)

	// /////////////////////////////////////////////////////////////////////////////
	// Conditional access certificates

	// GetConditionalAccessCertHostIDBySerialNumber retrieves the host_id for a valid certificate by serial number.
	GetConditionalAccessCertHostIDBySerialNumber(ctx context.Context, serial uint64) (uint, error)
	// GetConditionalAccessCertCreatedAtByHostID retrieves the created_at timestamp of the most recent certificate for a host.
	GetConditionalAccessCertCreatedAtByHostID(ctx context.Context, hostID uint) (*time.Time, error)
	// RevokeOldConditionalAccessCerts revokes old certificates for hosts that have a newer certificate.
	// Returns the number of certificates revoked.
	RevokeOldConditionalAccessCerts(ctx context.Context, gracePeriod time.Duration) (int64, error)

	// /////////////////////////////////////////////////////////////////////////////
	// Certificate Authorities
	// NewCertificateAuthority creates a new certificate authority.
	NewCertificateAuthority(ctx context.Context, ca *CertificateAuthority) (*CertificateAuthority, error)
	// GetCertificateAuthorityByID gets a certificate authority by its ID.
	GetCertificateAuthorityByID(ctx context.Context, id uint, includeSecrets bool) (*CertificateAuthority, error)
	// GetAllCertificateAuthorities returns all certificate authorities.
	GetAllCertificateAuthorities(ctx context.Context, includeSecrets bool) ([]*CertificateAuthority, error)
	// GetGroupedCertificateAuthorities returns all certificate authorities grouped by type
	GetGroupedCertificateAuthorities(ctx context.Context, includeSecrets bool) (*GroupedCertificateAuthorities, error)
	// ListCertificateAuthorities returns a summary of all certificate authorities.
	ListCertificateAuthorities(ctx context.Context) ([]*CertificateAuthoritySummary, error)
	// DeleteCertificateAuthority deletes the certificate authority of the provided ID, returns not found if it does not exist
	DeleteCertificateAuthority(ctx context.Context, certificateAuthorityID uint) (*CertificateAuthoritySummary, error)
	// UpdateCertificateAuthorityByID updates the certificate authority of the provided ID, returns not found if it does not exist
	UpdateCertificateAuthorityByID(ctx context.Context, id uint, certificateAuthority *CertificateAuthority) error
	// BatchApplyCertificateAuthorities applies a batch of certificate authority changes (add,
	// update, delete). Deletes are processed first based on name and type. Adds and updates are
	// processed together as upserts using INSERT...ON DUPLICATE KEY UPDATE.
	BatchApplyCertificateAuthorities(ctx context.Context, ops CertificateAuthoritiesBatchOperations) error
	// UpsertCertificateStatus allows a host to update the installation status of a certificate given its template.
	UpsertCertificateStatus(ctx context.Context, update *CertificateStatusUpdate) error

	// BatchUpsertCertificateTemplates upserts a batch of certificates.
	// Returns a map of team IDs that had certificates inserted or updated.
	BatchUpsertCertificateTemplates(ctx context.Context, certificates []*CertificateTemplate) ([]uint, error)
	// BatchDeleteCertificateTemplates deletes a batch of certificates.
	// Returns true if any rows were deleted.
	BatchDeleteCertificateTemplates(ctx context.Context, certificateTemplateIDs []uint) (bool, error)
	// CreateCertificateTemplate creates a new certificate template.
	CreateCertificateTemplate(ctx context.Context, certificateTemplate *CertificateTemplate) (*CertificateTemplateResponse, error)
	// DeleteCertificateTemplate deletes a certificate template by its ID.
	DeleteCertificateTemplate(ctx context.Context, id uint) error
	// GetCertificateTemplateById gets a certificate template by its ID (without host-specific data).
	GetCertificateTemplateById(ctx context.Context, id uint) (*CertificateTemplateResponse, error)
	// GetCertificateTemplateByIdForHost gets a certificate template by ID with host-specific status and challenge.
	GetCertificateTemplateByIdForHost(ctx context.Context, id uint, hostUUID string) (*CertificateTemplateResponseForHost, error)
	// GetCertificateTemplatesByTeamID gets all certificate templates for a team.
	GetCertificateTemplatesByTeamID(ctx context.Context, teamID uint, opts ListOptions) ([]*CertificateTemplateResponseSummary, *PaginationMetadata, error)
	// GetCertificateTemplatesByIdsAndTeam gets certificate templates by team ID and a list of certificate template IDs.
	GetCertificateTemplatesByIdsAndTeam(ctx context.Context, ids []uint, teamID uint) ([]*CertificateTemplateResponse, error)
	// GetCertificateTemplateByTeamIDAndName gets a certificate template by team ID and name.
	GetCertificateTemplateByTeamIDAndName(ctx context.Context, teamID uint, name string) (*CertificateTemplateResponse, error)
	// ListAndroidHostUUIDsWithDeliverableCertificateTemplates returns a paginated list of Android host UUIDs that have certificate templates.
	ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx context.Context, offset int, limit int) ([]string, error)
	// ListCertificateTemplatesForHosts returns ALL certificate templates for the given host UUIDs.
	ListCertificateTemplatesForHosts(ctx context.Context, hostUUIDs []string) ([]CertificateTemplateForHost, error)
	// GetCertificateTemplateForHost returns a certificate template for the given host UUID and certificate template ID.
	GetCertificateTemplateForHost(ctx context.Context, hostUUID string, certificateTemplateID uint) (*CertificateTemplateForHost, error)
	// GetHostCertificateTemplateRecord returns the host_certificate_templates record directly without
	// requiring the parent certificate_template to exist. Used for status updates on orphaned records.
	GetHostCertificateTemplateRecord(ctx context.Context, hostUUID string, certificateTemplateID uint) (*HostCertificateTemplate, error)
	// RetryHostCertificateTemplate resets a failed certificate to pending for automatic retry, increments
	// retry_count, preserves the error detail, and clears challenge/cert fields.
	RetryHostCertificateTemplate(ctx context.Context, hostUUID string, certificateTemplateID uint, detail string) error
	// GetCertificateTemplateStatusesByNameForHosts returns cert template statuses
	// keyed by host UUID and template name for all given hosts in a single query.
	// Only install records are considered; pending-remove rows are excluded.
	GetCertificateTemplateStatusesByNameForHosts(ctx context.Context, hostUUIDs []string) (map[string]map[string]CertificateTemplateStatus, error)
	// BulkInsertHostCertificateTemplates inserts multiple host_certificate_templates records.
	BulkInsertHostCertificateTemplates(ctx context.Context, hostCertTemplates []HostCertificateTemplate) error
	// DeleteHostCertificateTemplates deletes specific host_certificate_templates records
	// identified by (host_uuid, certificate_template_id) pairs.
	DeleteHostCertificateTemplates(ctx context.Context, hostCertTemplates []HostCertificateTemplate) error
	// DeleteHostCertificateTemplate deletes a single host_certificate_template record
	// identified by host_uuid and certificate_template_id.
	DeleteHostCertificateTemplate(ctx context.Context, hostUUID string, certificateTemplateID uint) error
	// DeleteAllHostCertificateTemplates deletes all host_certificate_templates records for a host.
	// Used during re-enrollment to clear stale cert records (including those from previous teams)
	// before creating fresh pending records for the host's current team.
	DeleteAllHostCertificateTemplates(ctx context.Context, hostUUID string) error
	// ResendHostCertificateTemplate queues a certificate template to be resent to a device
	ResendHostCertificateTemplate(ctx context.Context, hostID uint, templateID uint) error

	// ListAndroidHostUUIDsWithPendingCertificateTemplates returns hosts that have
	// certificate templates in 'pending' status ready for delivery.
	ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx context.Context, offset int, limit int) ([]string, error)

	// GetAndTransitionCertificateTemplatesToDelivering retrieves all certificate templates
	// with operation_type='install' for a host, transitions any pending ones to 'delivering' status.
	// If there are no pending certificate templates, then nothing is returned.
	GetAndTransitionCertificateTemplatesToDelivering(ctx context.Context, hostUUID string) (*HostCertificateTemplatesForDelivery, error)

	// TransitionCertificateTemplatesToDelivered transitions the specified templates from 'delivering' to 'delivered'.
	TransitionCertificateTemplatesToDelivered(ctx context.Context, hostUUID string, templateIDs []uint) error

	// RevertHostCertificateTemplatesToPending reverts specific host certificate templates from 'delivering' back to 'pending'.
	RevertHostCertificateTemplatesToPending(ctx context.Context, hostUUID string, certificateTemplateIDs []uint) error

	// SetHostCertificateTemplatesToPendingRemove prepares certificate templates for removal.
	// For a given certificate template ID, it deletes any rows with status=pending and
	// updates all other rows to status=pending, operation_type=remove.
	SetHostCertificateTemplatesToPendingRemove(ctx context.Context, certificateTemplateID uint) error

	// SetHostCertificateTemplatesToPendingRemoveForHost prepares all certificate templates
	// for a specific host for removal.
	SetHostCertificateTemplatesToPendingRemoveForHost(ctx context.Context, hostUUID string) error

	// GetAndroidCertificateTemplatesForRenewal returns certificate templates that are approaching
	// expiration and need to be renewed. Uses the same threshold logic as Apple/Windows:
	// - If validity period > 30 days: renew within 30 days of expiration
	// - If validity period <= 30 days: renew within half the validity period of expiration
	// Only returns certificates with status 'delivered' or 'verified' and operation_type 'install'.
	GetAndroidCertificateTemplatesForRenewal(ctx context.Context, now time.Time, limit int) ([]HostCertificateTemplateForRenewal, error)

	// SetAndroidCertificateTemplatesForRenewal marks the specified certificate templates for renewal
	// by setting status to 'pending', clearing validity fields, and generating a new UUID.
	// The new UUID signals to the Android agent that the certificate needs renewal.
	SetAndroidCertificateTemplatesForRenewal(ctx context.Context, templates []HostCertificateTemplateForRenewal) error

	// GetOrCreateFleetChallengeForCertificateTemplate ensures a fleet challenge exists for the given
	// host and certificate template. If a challenge already exists, it returns it. If not, it creates
	// a new one atomically. Only works for templates in 'delivered' status and with operation_type 'install'.
	GetOrCreateFleetChallengeForCertificateTemplate(ctx context.Context, hostUUID string, certificateTemplateID uint) (string, error)

	// GetCurrentTime gets the current time from the database
	GetCurrentTime(ctx context.Context) (time.Time, error)

	// GetWindowsMDMCommandsForResending retrieves Windows MDM commands that failed to be delivered
	// and need to be resent based on their command IDs.
	//
	// Returns a slice of MDMWindowsCommand pointers containing the commands to be resent.
	GetWindowsMDMCommandsForResending(ctx context.Context, deviceID string, failedCommandIds []string) ([]*MDMWindowsCommand, error)

	// ResendWindowsMDMCommand marks the specified Windows MDM command for resend
	// by inserting a new command entry, command queue, but also updates the host profile reference.
	ResendWindowsMDMCommand(ctx context.Context, mdmDeviceId string, newCmd *MDMWindowsCommand, oldCmd *MDMWindowsCommand) error

	// GetHostVPPInstallByCommandUUID retrieves the Apple VPP app install record
	// for the given command UUID.
	GetHostVPPInstallByCommandUUID(ctx context.Context, commandUUID string) (*HostVPPSoftwareInstallLite, error)

	// RetryVPPInstall retries a single VPP install that failed for the host.
	// It makes sure to queue a new nano command and update the command_uuid in the host_vpp_software_installs table, as well as the execution ID for the activity.
	RetryVPPInstall(ctx context.Context, vppInstall *HostVPPSoftwareInstallLite) error

	// MDMWindowsUpdateEnrolledDeviceCredentials updates the credentials hash for the enrolled Windows device.
	MDMWindowsUpdateEnrolledDeviceCredentials(ctx context.Context, deviceId string, credentialsHash []byte) error
	// MDMWindowsAcknowledgeEnrolledDeviceCredentials marks the enrolled Windows device credentials as acknowledged.
	MDMWindowsAcknowledgeEnrolledDeviceCredentials(ctx context.Context, deviceId string) error

	// IsAppleEnrollmentRenewalCommand checks if the given command UUID corresponds to an Apple enrollment renewal command (SCEP/ACME) for the host with the given UUID.
	IsAppleEnrollmentRenewalCommand(ctx context.Context, commandUUID, hostUUID string) (bool, error)

	// MDMAppleResetOnReenrollment performs necessary datastore operations to reset the state of a host that is re-enrolling in MDM,
	// resetting label membership, other host data, and optionally host activities (activities and mdm command queue).
	// Host activities will not be cleared if preserveHostActivities is true
	MDMAppleResetOnReenrollment(ctx context.Context, hostUUID string, preserveHostActivities bool) error

	// VerifyAppleConfigProfileScopesDoNotConflict checks scopes against existing profiles across the entire DB
	// to ensure there are no conflicts where an existing profile with the same identifier
	// has a different scope than the incoming profile. If we don't do this we must implement some sort of "move" semantics
	// to allow for scope changes when a host switches teams or when a profile is updated.
	VerifyAppleConfigProfileScopesDoNotConflict(ctx context.Context, cps []*MDMAppleConfigProfile) error
}

type AndroidDatastore interface {
	android.Datastore
	AndroidHostLite(ctx context.Context, enterpriseSpecificID string) (*AndroidHost, error)
	AndroidHostLiteByHostUUID(ctx context.Context, hostUUID string) (*AndroidHost, error)
	AppConfig(ctx context.Context) (*AppConfig, error)
	BulkSetAndroidHostsUnenrolled(ctx context.Context) error
	SetAndroidHostUnenrolled(ctx context.Context, hostID uint) (bool, error)
	DeleteMDMConfigAssetsByName(ctx context.Context, assetNames []MDMAssetName) error
	GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []MDMAssetName,
		queryerContext sqlx.QueryerContext) (map[MDMAssetName]MDMConfigAsset, error)
	InsertOrReplaceMDMConfigAsset(ctx context.Context, asset MDMConfigAsset) error
	NewAndroidHost(ctx context.Context, host *AndroidHost, companyOwned bool) (*AndroidHost, error)
	SetAndroidEnabledAndConfigured(ctx context.Context, configured bool) error
	UpdateAndroidHost(ctx context.Context, host *AndroidHost, fromEnroll, companyOwned bool) error
	UserOrDeletedUserByID(ctx context.Context, id uint) (*User, error)
	VerifyEnrollSecret(ctx context.Context, secret string) (*EnrollSecret, error)
	GetMDMIdPAccountByUUID(ctx context.Context, uuid string) (*MDMIdPAccount, error)
	AssociateHostMDMIdPAccount(ctx context.Context, hostUUID, idpAcctUUID string) error
	TeamIDsWithSetupExperienceIdPEnabled(ctx context.Context) ([]uint, error)
	// TeamLite retrieves a Team by ID, including only id, created_at, name, filename, description, config fields.
	TeamLite(ctx context.Context, tid uint) (*TeamLite, error)
	// BulkUpsertMDMAndroidHostProfiles bulk-adds/updates records to track the
	// status of a profile in a host.
	BulkUpsertMDMAndroidHostProfiles(ctx context.Context, payload []*MDMAndroidProfilePayload) error
	// BulkDeleteMDMAndroidHostProfiles bulk removes records from the host's profile, that is pending or failed remove and less than or equals to the policy version.
	BulkDeleteMDMAndroidHostProfiles(ctx context.Context, hostUUID string, policyVersionID int64) error
	// ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersion returns a list of all android profiles that are pending or failed install, and where version is less than or equals to the policyVersion.
	ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersion(ctx context.Context, hostUUID string, policyVersion int64) ([]*MDMAndroidProfilePayload, error)
	GetAndroidPolicyRequestByUUID(ctx context.Context, requestUUID string) (*android.MDMAndroidPolicyRequest, error)
	// UpdateHostSoftware updates the software list of a host.
	// The update consists of deleting existing entries that are not in the given `software`
	// slice, updating existing entries and inserting new entries.
	// Returns a struct with the current installed software on the host (pre-mutations) plus all
	// mutations performed: what was inserted and what was removed.
	UpdateHostSoftware(ctx context.Context, hostID uint, software []Software) (*UpdateHostSoftwareDBResult, error)

	// GetLatestAppleMDMCommandOfType retrieves the latest command of the given type for the host with the given UUID.
	// If no such command exists, not found error is returned
	//
	// Returns a subset of fields in the MDMCommand struct.
	GetLatestAppleMDMCommandOfType(ctx context.Context, hostUUID string, commandType string) (*MDMCommand, error)

	// SetLockCommandForLostModeCheckin sets the lock reference for a lost mode check-in.
	// This is used when an iphone or ipados checks in after being deleted, with lost mode enabled.
	SetLockCommandForLostModeCheckin(ctx context.Context, hostID uint, commandUUID string) error

	// ListHostMDMAndroidVPPAppsPendingInstallWithVersion lists the Android
	// VPP apps pending install for a host that were requested in a policy
	// version <= the provided policy version.
	ListHostMDMAndroidVPPAppsPendingInstallWithVersion(ctx context.Context, hostUUID string, policyVersion int64) ([]*HostAndroidVPPSoftwareInstall, error)

	// BulkSetVPPInstallsAsVerified marks all VPP apps identified by the command UUIDs
	// as verified. This is for Android hosts, where the verification uuid is not important,
	// so the implementation generates a random one.
	BulkSetVPPInstallsAsVerified(ctx context.Context, hostID uint, commandUUIDs []string) error

	// BulkSetVPPInstallsAsFailed marks all VPP apps identified by the command UUIDs
	// as failed. This is for Android hosts, where the verification uuid is not important,
	// so the implementation generates a random one.
	BulkSetVPPInstallsAsFailed(ctx context.Context, hostID uint, commandUUIDs []string) error

	// GetPastActivityDataForAndroidVPPAppInstall is like GetPastActivityDataForVPPAppInstall
	// but available to the android datastore and without the Apple-based args.
	GetPastActivityDataForAndroidVPPAppInstall(ctx context.Context, cmdUUID string, status SoftwareInstallerStatus) (*User, *ActivityInstalledAppStoreApp, error)

	MarkAllPendingAndroidVPPInstallsAsFailed(ctx context.Context) error
	MarkAllPendingVPPInstallsAsFailedForAndroidHost(ctx context.Context, hostID uint) (users []*User, activities []ActivityDetails, err error)
}

// MDMAppleStore wraps nanomdm's storage and adds methods to deal with
// Fleet-specific use cases.
type MDMAppleStore interface {
	storage.AllStorage
	MDMAssetRetriever
	GetPendingLockCommand(ctx context.Context, hostUUID string) (*mdm.Command, string, error)
	EnqueueDeviceLockCommand(ctx context.Context, host *Host, cmd *mdm.Command, pin string) error
	EnqueueDeviceUnlockCommand(ctx context.Context, host *Host, cmd *mdm.Command) error
	EnqueueDeviceWipeCommand(ctx context.Context, host *Host, cmd *mdm.Command) error
}

type MDMAssetRetriever interface {
	GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []MDMAssetName,
		queryerContext sqlx.QueryerContext) (map[MDMAssetName]MDMConfigAsset, error)
	GetABMTokenByOrgName(ctx context.Context, orgName string) (*ABMToken, error)
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
	// ExpandEmbeddedSecrets expands the fleet secrets in a
	// document using the secrets stored in the datastore.
	ExpandEmbeddedSecrets(ctx context.Context, document string) (string, error)
	// GetHostMDMWindowsProfiles returns the current MDM profile status for the given
	// Windows host
	GetHostMDMWindowsProfiles(ctx context.Context, hostUUID string) ([]HostMDMWindowsProfile, error)

	HostLiteByIdentifier(ctx context.Context, identifier string) (*HostLite, error)
	// IsAppleEnrollmentRenewalCommand checks if the given command UUID corresponds to an Apple enrollment renewal command (SCEP/ACME) for the host with the given UUID.
	IsAppleEnrollmentRenewalCommand(ctx context.Context, commandUUID, hostUUID string) (bool, error)
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
	// NeedsFleetv4732Fix means the database needs the special fix migration for fleet v4.73.2
	NeedsFleetv4732Fix
	// UnknownFleetv4732State means the database has the broken migrations from fleet v4.73.2 however
	// it is not in the expected state and needs manual intervention.
	UnknownFleetv4732State
)

// TODO: we have a similar but different interface in the service package,
// service.NotFoundErr - at the very least, the IsNotFound method should be the
// same in both (the other is currently NotFound), and ideally we'd just have
// one of those interfaces.

// NotFoundError is an alias for platform_errors.NotFoundError.
type NotFoundError = platform_errors.NotFoundError

// IsNotFound is an alias for platform_errors.IsNotFound.
var IsNotFound = platform_errors.IsNotFound

// AlreadyExistsError is an alias for platform_http.AlreadyExistsError.
type AlreadyExistsError = platform_http.AlreadyExistsError

// ForeignKeyError is an alias for platform_http.ForeignKeyError.
type ForeignKeyError = platform_http.ForeignKeyError

// IsForeignKey is an alias for platform_http.IsForeignKey.
var IsForeignKey = platform_http.IsForeignKey

type OptionalArg func() interface{}

// SecretUsedError is returned when attempting to delete a variable that is in use in scripts or profiles.
type SecretUsedError struct {
	SecretName string
	Entity     EntityUsingSecret
}

// Error implements the error interface.
func (c *SecretUsedError) Error() string {
	if c.Entity.Type == "script" {
		return fmt.Sprintf(
			"%s is used by the %q script in the %q team. Please edit or delete the script and try again.",
			c.SecretName, c.Entity.Name, c.Entity.TeamName,
		)
	}
	return fmt.Sprintf(
		"%s is used by the %q configuration profile in the %q team. Please delete the configuration profile and try again.",
		c.SecretName, c.Entity.Name, c.Entity.TeamName,
	)
}

// EntityUsingSecret describes the entity using a secret variable.
type EntityUsingSecret struct {
	// Type is the entity type, "script", "apple_profile", "apple_declaration", or "windows_profile".
	Type string
	// Name is the name of the entity.
	Name string
	// TeamName is the name of the team the entity belongs to.
	TeamName string
}

type AccessesMDMConfigAssets interface {
	// InsertMDMConfigAssets inserts MDM-related config assets, such as SCEP and APNS certs and keys.
	// tx is used to pass an existing transaction; if nil, a new transaction will be created inside the call
	InsertMDMConfigAssets(ctx context.Context, assets []MDMConfigAsset, tx sqlx.ExtContext) error
	// InsertOrReplaceMDMConfigAsset inserts or updates an encrypted asset.
	InsertOrReplaceMDMConfigAsset(ctx context.Context, asset MDMConfigAsset) error
	// GetAllMDMConfigAssetsByName returns the requested config assets.
	//
	// If it doesn't find all the assets requested, it returns a `mysql.ErrPartialResult` error.
	// The queryerContext is optional and can be used to pass a transaction.
	GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []MDMAssetName,
		queryerContext sqlx.QueryerContext) (map[MDMAssetName]MDMConfigAsset, error)
	// GetAllMDMConfigAssetsHashes behaves like
	// GetAllMDMConfigAssetsByName, but only returns a sha256 checksum of
	// each asset
	//
	// If it doesn't find all the assets requested, it returns a `mysql.ErrPartialResult`
	GetAllMDMConfigAssetsHashes(ctx context.Context, assetNames []MDMAssetName) (map[MDMAssetName]string, error)
	// DeleteMDMConfigAssetsByName soft deletes the given MDM config assets.
	DeleteMDMConfigAssetsByName(ctx context.Context, assetNames []MDMAssetName) error
	// HardDeleteMDMConfigAsset permanently deletes the given MDM config asset.
	HardDeleteMDMConfigAsset(ctx context.Context, assetName MDMAssetName) error
	// ReplaceMDMConfigAssets replaces (soft delete if they exist + insert) `MDMConfigAsset`s in a
	// single transaction. Useful for "renew" flows where users are updating the assets with newly
	// generated ones.
	// tx parameter is optional and can be used to pass an existing transaction.
	ReplaceMDMConfigAssets(ctx context.Context, assets []MDMConfigAsset, tx sqlx.ExtContext) error
	// GetAllCAConfigAssetsByType returns the config assets for DigiCert and custom SCEP CAs.
	GetAllCAConfigAssetsByType(ctx context.Context, assetType CAConfigAssetType) (map[string]CAConfigAsset, error)
}
type GetsAppConfig interface {
	AppConfig(ctx context.Context) (*AppConfig, error)
	AppConfigUrls(ctx context.Context) (*AppConfigUrls, error)
}
