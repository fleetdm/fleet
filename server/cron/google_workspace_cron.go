package cron

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
)

const (
	// googleWorkspaceSyncInterval is how often Fleet pulls the Google Workspace directory.
	googleWorkspaceSyncInterval = 5 * time.Minute

	// scimSyncPageSize is the page size used when loading the current scim_* state
	// from the database during reconciliation.
	scimSyncPageSize = 1000
)

// GoogleWorkspaceDirectoryFactory builds a directory client for the given
// integration. It is injected so cron can run without importing the EE client
// package directly and so tests can supply a fake directory. The logger is
// passed through so the directory client can emit per-user/group debug logs.
type GoogleWorkspaceDirectoryFactory func(ctx context.Context, intg *fleet.GoogleWorkspaceIntegration, logger *slog.Logger) (fleet.GoogleWorkspaceDirectory, error)

// NewGoogleWorkspaceSchedule registers the periodic Google Workspace directory
// sync. The job no-ops when no Google Workspace integration is configured.
func NewGoogleWorkspaceSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	factory GoogleWorkspaceDirectoryFactory,
	logger *slog.Logger,
) (*schedule.Schedule, error) {
	name := string(fleet.CronGoogleWorkspaceSync)
	logger = logger.With("cron", name)
	s := schedule.New(
		ctx, name, instanceID, googleWorkspaceSyncInterval, ds, ds,
		schedule.WithLogger(logger),
		schedule.WithJob(
			"google_workspace_sync",
			func(ctx context.Context) error {
				return cronGoogleWorkspaceSync(ctx, ds, factory, logger)
			},
		),
	)
	return s, nil
}

// cronGoogleWorkspaceSync runs one sync pass and records the result so the IdP
// settings UI can surface the last sync status. It reuses scim_last_request:
// because Google Workspace and SCIM are mutually exclusive (SCIM is ignored while
// a Google Workspace integration is configured), that row represents the last IdP
// ingest regardless of source.
func cronGoogleWorkspaceSync(ctx context.Context, ds fleet.Datastore, factory GoogleWorkspaceDirectoryFactory, logger *slog.Logger) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config")
	}
	if len(appConfig.Integrations.GoogleWorkspace) == 0 {
		// Not configured; nothing to do.
		return nil
	}
	intg := appConfig.Integrations.GoogleWorkspace[0]

	syncErr := syncGoogleWorkspaceDirectory(ctx, ds, factory, intg, logger)

	lastRequest := &fleet.ScimLastRequest{Status: "success"}
	if syncErr != nil {
		lastRequest.Status = "error"
		// The scim_last_request.details column is VARCHAR(255). Sync errors can wrap
		// arbitrarily long messages from the Google API, so truncate before writing or
		// UpdateScimLastRequest rejects the row and the failure goes unrecorded.
		lastRequest.Details = truncateRunes(syncErr.Error(), fleet.SCIMMaxFieldLength)
		logger.ErrorContext(ctx, "google workspace sync failed", "err", syncErr)
	}
	if err := ds.UpdateScimLastRequest(ctx, lastRequest); err != nil {
		// Don't mask the sync error with a status-write error, but do surface it.
		logger.ErrorContext(ctx, "update google workspace last sync status", "err", err)
	}

	return syncErr
}

// truncateRunes returns s shortened to at most maxRunes characters, preserving the
// start of the string. utf8mb4 VARCHAR(N) in MySQL counts characters (runes), not
// bytes, so we slice on runes to align with the column constraint.
func truncateRunes(s string, maxRunes int) string {
	if len(s) <= maxRunes {
		// Fast path: ASCII fits in maxRunes bytes -> maxRunes characters max.
		return s
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
}

// syncGoogleWorkspaceDirectory pulls the full directory and reconciles it into the
// scim_* tables. Everything downstream of those tables (host linking, IdP host
// vitals, host-vitals labels, Fleet variables) is source-agnostic and works
// unchanged.
func syncGoogleWorkspaceDirectory(
	ctx context.Context,
	ds fleet.Datastore,
	factory GoogleWorkspaceDirectoryFactory,
	intg *fleet.GoogleWorkspaceIntegration,
	logger *slog.Logger,
) error {
	dir, err := factory(ctx, intg, logger)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create google workspace directory client")
	}

	gwUsers, err := dir.ListUsers(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list users from google workspace")
	}
	gwGroups, err := dir.ListGroups(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list groups from google workspace")
	}

	extIDToScimUserID, usersFailed, err := syncGoogleWorkspaceUsers(ctx, ds, gwUsers, logger)
	if err != nil {
		return err
	}

	groupsFailed, err := syncGoogleWorkspaceGroups(ctx, ds, gwGroups, extIDToScimUserID, logger)
	if err != nil {
		return err
	}

	logger.InfoContext(ctx, "google workspace sync complete",
		"users", len(gwUsers), "users_failed", usersFailed,
		"groups", len(gwGroups), "groups_failed", groupsFailed)

	// Per-record failures don't abort the sync (one bad user/group shouldn't block
	// the rest of the directory), but we surface them as an error so the sync status
	// reflects the partial failure. Specifics are in the logs above.
	if usersFailed > 0 || groupsFailed > 0 {
		return ctxerr.Errorf(ctx, "partial sync: %d of %d users and %d of %d groups failed to ingest; see server logs for details",
			usersFailed, len(gwUsers), groupsFailed, len(gwGroups))
	}
	return nil
}

// syncGoogleWorkspaceUsers reconciles users and returns a map of Google user ID
// (external_id) -> scim_users.id, used to resolve group membership.
func syncGoogleWorkspaceUsers(ctx context.Context, ds fleet.Datastore, gwUsers []*fleet.ScimUser, logger *slog.Logger) (map[string]uint, int, error) {
	existing, err := listAllScimUsers(ctx, ds)
	if err != nil {
		// Failing to load the current state is fatal: without it every user would
		// look new and we'd attempt to recreate the entire directory.
		return nil, 0, ctxerr.Wrap(ctx, err, "list existing scim users")
	}
	existingByExtID := make(map[string]*fleet.ScimUser, len(existing))
	for i := range existing {
		u := &existing[i]
		if u.ExternalID != nil {
			existingByExtID[*u.ExternalID] = u
		}
	}

	seen := make(map[string]struct{}, len(gwUsers))
	extIDToScimUserID := make(map[string]uint, len(gwUsers))
	failed := 0

	for _, gu := range gwUsers {
		if gu.ExternalID == nil || *gu.ExternalID == "" {
			continue
		}
		extID := *gu.ExternalID
		seen[extID] = struct{}{}

		if ex, ok := existingByExtID[extID]; ok {
			gu.ID = ex.ID
			extIDToScimUserID[extID] = ex.ID
			if scimUserNeedsUpdate(ex, gu) {
				if _, err := ds.ReplaceScimUser(ctx, gu); err != nil {
					// Best-effort: log and skip this user so one bad record doesn't
					// abort the whole sync.
					failed++
					logger.ErrorContext(ctx, "google workspace sync: skipping user that failed to update",
						"user_name", gu.UserName, "external_id", extID, "err", err)
				}
			}
			continue
		}

		id, err := ds.CreateScimUser(ctx, gu)
		if err != nil {
			failed++
			logger.ErrorContext(ctx, "google workspace sync: skipping user that failed to create",
				"user_name", gu.UserName, "external_id", extID, "err", err)
			continue
		}
		extIDToScimUserID[extID] = id
	}

	// Delete users that are no longer in Google Workspace. Google Workspace is the
	// source of truth while configured, so any scim user not present in the pull is
	// removed (cascading to host_scim_user). Guard against a misconfiguration that
	// returns zero users, which would otherwise wipe all IdP data.
	if len(gwUsers) == 0 {
		logger.WarnContext(ctx, "google workspace returned no users; skipping user deletion to avoid data loss")
		return extIDToScimUserID, failed, nil
	}
	for extID, ex := range existingByExtID {
		if _, ok := seen[extID]; ok {
			continue
		}
		if _, err := ds.DeleteScimUser(ctx, ex.ID); err != nil {
			failed++
			logger.ErrorContext(ctx, "google workspace sync: failed to delete user no longer in directory",
				"scim_user_id", ex.ID, "external_id", extID, "err", err)
		}
	}

	return extIDToScimUserID, failed, nil
}

// syncGoogleWorkspaceGroups reconciles groups and their memberships, resolving
// member Google user IDs to scim_users.id via extIDToScimUserID.
func syncGoogleWorkspaceGroups(
	ctx context.Context,
	ds fleet.Datastore,
	gwGroups []*fleet.GoogleWorkspaceGroup,
	extIDToScimUserID map[string]uint,
	logger *slog.Logger,
) (int, error) {
	existing, err := listAllScimGroups(ctx, ds)
	if err != nil {
		// Failing to load the current state is fatal (see syncGoogleWorkspaceUsers).
		return 0, ctxerr.Wrap(ctx, err, "list existing scim groups")
	}
	existingByExtID := make(map[string]*fleet.ScimGroup, len(existing))
	for i := range existing {
		g := &existing[i]
		if g.ExternalID != nil {
			existingByExtID[*g.ExternalID] = g
		}
	}

	seen := make(map[string]struct{}, len(gwGroups))
	// scim_groups.display_name is UNIQUE under a case-insensitive collation, but
	// Google only requires group emails to be unique. A group already stored
	// under its own name keeps it whatever order Google lists groups in, since
	// labels and profile variables key on the name; any other group whose name
	// is taken gets its external ID appended.
	usedDisplayNames := make(map[string]struct{}, len(gwGroups))
	for _, gg := range gwGroups {
		if ex, ok := existingByExtID[gg.ExternalID]; ok && gg.ExternalID != "" && holdsDisplayName(ex, gg) {
			usedDisplayNames[displayNameKey(baseDisplayName(gg))] = struct{}{}
		}
	}
	failed := 0

	for _, gg := range gwGroups {
		if gg.ExternalID == "" {
			continue
		}
		seen[gg.ExternalID] = struct{}{}

		memberIDs := make([]uint, 0, len(gg.MemberExternalIDs))
		for _, memberExtID := range gg.MemberExternalIDs {
			if scimID, ok := extIDToScimUserID[memberExtID]; ok {
				memberIDs = append(memberIDs, scimID)
			}
		}

		ex := existingByExtID[gg.ExternalID]
		suffixed := fmt.Sprintf("%s (%s)", baseDisplayName(gg), gg.ExternalID)
		displayName := baseDisplayName(gg)
		if ex == nil || !holdsDisplayName(ex, gg) {
			displayName = uniqueDisplayName(displayName, suffixed, usedDisplayNames)
		}
		desired := &fleet.ScimGroup{
			ExternalID:  new(gg.ExternalID),
			DisplayName: displayName,
			ScimUsers:   memberIDs,
		}
		if ex != nil {
			desired.ID = ex.ID
		}

		if err := writeGoogleWorkspaceGroup(ctx, ds, ex, desired, suffixed, usedDisplayNames, logger); err != nil {
			// Best-effort: log and skip so one bad group doesn't abort the sync.
			failed++
			msg := "google workspace sync: skipping group that failed to create"
			if ex != nil {
				msg = "google workspace sync: skipping group that failed to update"
			}
			logger.ErrorContext(ctx, msg, "display_name", desired.DisplayName, "external_id", gg.ExternalID, "err", err)
		}
	}

	// Delete groups no longer in Google Workspace (guard against an empty pull).
	if len(gwGroups) == 0 {
		logger.WarnContext(ctx, "google workspace returned no groups; skipping group deletion to avoid data loss")
		return failed, nil
	}
	for extID, ex := range existingByExtID {
		if _, ok := seen[extID]; ok {
			continue
		}
		if err := ds.DeleteScimGroup(ctx, ex.ID); err != nil {
			failed++
			logger.ErrorContext(ctx, "google workspace sync: failed to delete group no longer in directory",
				"scim_group_id", ex.ID, "external_id", extID, "err", err)
		}
	}

	return failed, nil
}

// writeGoogleWorkspaceGroup creates desired, or replaces ex with it if anything
// changed. The collation also equates names that differ by accents, emoji or
// trailing spaces, which displayNameKey doesn't model, so a write rejected for
// a taken name is retried once with suffixed.
func writeGoogleWorkspaceGroup(
	ctx context.Context,
	ds fleet.Datastore,
	ex, desired *fleet.ScimGroup,
	suffixed string,
	usedDisplayNames map[string]struct{},
	logger *slog.Logger,
) error {
	write := func() error {
		if ex == nil {
			_, err := ds.CreateScimGroup(ctx, desired)
			return err
		}
		if !scimGroupNeedsUpdate(ex, desired) {
			return nil
		}
		return ds.ReplaceScimGroup(ctx, desired)
	}

	err := write()
	var existsErr fleet.AlreadyExistsError
	if !errors.As(err, &existsErr) || desired.DisplayName == suffixed {
		return err
	}
	logger.DebugContext(ctx, "google workspace sync: group display name taken, retrying with external ID suffix",
		"display_name", desired.DisplayName, "external_id", ptr.ValOrZero(desired.ExternalID))
	desired.DisplayName = suffixed
	usedDisplayNames[displayNameKey(suffixed)] = struct{}{}
	return write()
}

// baseDisplayName is the name a Google group is stored under when nothing
// else claims it.
func baseDisplayName(gg *fleet.GoogleWorkspaceGroup) string {
	if gg.DisplayName == "" {
		return gg.ExternalID
	}
	return gg.DisplayName
}

// holdsDisplayName reports whether the stored group already has gg's name,
// ignoring case so a case-only rename in Google keeps ownership.
func holdsDisplayName(ex *fleet.ScimGroup, gg *fleet.GoogleWorkspaceGroup) bool {
	return displayNameKey(ex.DisplayName) == displayNameKey(baseDisplayName(gg))
}

func displayNameKey(name string) string {
	return strings.ToLower(name)
}

// uniqueDisplayName returns name, or suffixed if a group in this sync pass
// already claimed name, and records the result as claimed.
func uniqueDisplayName(name, suffixed string, used map[string]struct{}) string {
	if _, taken := used[displayNameKey(name)]; taken {
		name = suffixed
	}
	used[displayNameKey(name)] = struct{}{}
	return name
}

func scimUserNeedsUpdate(existing, desired *fleet.ScimUser) bool {
	switch {
	case existing.UserName != desired.UserName:
		return true
	case !strPtrEqual(existing.GivenName, desired.GivenName):
		return true
	case !strPtrEqual(existing.FamilyName, desired.FamilyName):
		return true
	case !strPtrEqual(existing.Department, desired.Department):
		return true
	case !boolPtrEqual(existing.Active, desired.Active):
		return true
	case !scimEmailsEqual(existing.Emails, desired.Emails):
		return true
	default:
		return false
	}
}

func scimGroupNeedsUpdate(existing, desired *fleet.ScimGroup) bool {
	if existing.DisplayName != desired.DisplayName {
		return true
	}
	return !uintSetEqual(existing.ScimUsers, desired.ScimUsers)
}

func listAllScimUsers(ctx context.Context, ds fleet.Datastore) ([]fleet.ScimUser, error) {
	var all []fleet.ScimUser
	startIndex := uint(1)
	for {
		page, total, err := ds.ListScimUsers(ctx, fleet.ScimUsersListOptions{
			ScimListOptions: fleet.ScimListOptions{StartIndex: startIndex, PerPage: scimSyncPageSize},
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) == 0 || uint(len(all)) >= total {
			break
		}
		startIndex += uint(len(page))
	}
	return all, nil
}

func listAllScimGroups(ctx context.Context, ds fleet.Datastore) ([]fleet.ScimGroup, error) {
	var all []fleet.ScimGroup
	startIndex := uint(1)
	for {
		page, total, err := ds.ListScimGroups(ctx, fleet.ScimGroupsListOptions{
			ScimListOptions: fleet.ScimListOptions{StartIndex: startIndex, PerPage: scimSyncPageSize},
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) == 0 || uint(len(all)) >= total {
			break
		}
		startIndex += uint(len(page))
	}
	return all, nil
}

func strPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func scimEmailsEqual(a, b []fleet.ScimUserEmail) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, e := range a {
		counts[e.GenerateComparisonKey()]++
	}
	for _, e := range b {
		counts[e.GenerateComparisonKey()]--
	}
	for _, v := range counts {
		if v != 0 {
			return false
		}
	}
	return true
}

func uintSetEqual(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[uint]int, len(a))
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
	}
	for _, v := range counts {
		if v != 0 {
			return false
		}
	}
	return true
}
