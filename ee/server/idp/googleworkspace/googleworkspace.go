// Package googleworkspace syncs IdP host vitals (users and groups) directly
// from Google Workspace via the Admin SDK Directory API, using a service account
// with domain-wide delegation. Synced users/groups are written into Fleet's SCIM
// tables, so the rest of the IdP host vitals stack (host association, foreign
// vitals, host-vitals labels, profile variables) works unchanged.
//
// It mirrors the patterns used by the Google Calendar integration
// (ee/server/calendar): a low level DirectoryAPI interface that can be replaced
// by an in-memory mock for testing by setting the service account "client_email"
// to MockEmail.
package googleworkspace

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	directory "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
)

const (
	// MockEmail, when set as the service account "client_email", selects the
	// in-memory mock DirectoryAPI implementation (for tests).
	MockEmail = "workspace-mock@example.com"

	// usersPageSize and groupsPageSize bound the Directory API page sizes.
	usersPageSize  = 500
	groupsPageSize = 200

	// memberTypeGroup is the Directory API member type for nested groups, which
	// Fleet's SCIM model does not support; such members are skipped in v1.
	memberTypeGroup = "GROUP"

	// emailTypeWork is the SCIM email type used for the Google primary email.
	emailTypeWork = "work"
)

// directoryScopes are the read-only Admin SDK Directory API scopes Fleet
// requests via domain-wide delegation. These match the permissions documented
// for comparable integrations (e.g. Kandji). group.readonly is sufficient to
// list group members.
var directoryScopes = []string{
	directory.AdminDirectoryUserReadonlyScope,
	directory.AdminDirectoryGroupReadonlyScope,
}

// DirectoryAPI is the low level interface to the Google Workspace Admin SDK
// Directory API. The real implementation is DirectoryLowLevelAPI; tests use a
// mock (selected via MockEmail).
type DirectoryAPI interface {
	// Configure authenticates using the service account credentials and
	// impersonates adminEmail (a Google Workspace super-admin) via domain-wide
	// delegation.
	Configure(ctx context.Context, serviceAccountEmail, privateKey, adminEmail string) error
	// ListUsers returns a page of users for the customer and the next page token
	// ("" when there are no more pages).
	ListUsers(ctx context.Context, customerID, pageToken string) ([]*directory.User, string, error)
	// ListGroups returns a page of groups for the customer and the next page token.
	ListGroups(ctx context.Context, customerID, pageToken string) ([]*directory.Group, string, error)
	// ListGroupMembers returns a page of members for the group and the next page token.
	ListGroupMembers(ctx context.Context, groupKey, pageToken string) ([]*directory.Member, string, error)
}

// DirectoryLowLevelAPI is the production DirectoryAPI backed by the real Google
// Admin SDK Directory API.
type DirectoryLowLevelAPI struct {
	service *directory.Service
}

// Configure creates a Directory API service using the service account
// credentials, impersonating adminEmail via domain-wide delegation.
func (a *DirectoryLowLevelAPI) Configure(ctx context.Context, serviceAccountEmail, privateKey, adminEmail string) error {
	conf := &jwt.Config{
		Email:      serviceAccountEmail,
		Scopes:     directoryScopes,
		PrivateKey: []byte(privateKey),
		TokenURL:   google.JWTTokenURL,
		Subject:    adminEmail,
	}
	service, err := directory.NewService(ctx, option.WithHTTPClient(conf.Client(ctx)))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create google workspace directory service")
	}
	a.service = service
	return nil
}

func (a *DirectoryLowLevelAPI) ListUsers(ctx context.Context, customerID, pageToken string) ([]*directory.User, string, error) {
	call := a.service.Users.List().Customer(customerID).MaxResults(usersPageSize).Projection("full").OrderBy("email")
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}
	res, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "list google workspace users")
	}
	return res.Users, res.NextPageToken, nil
}

func (a *DirectoryLowLevelAPI) ListGroups(ctx context.Context, customerID, pageToken string) ([]*directory.Group, string, error) {
	call := a.service.Groups.List().Customer(customerID).MaxResults(groupsPageSize)
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}
	res, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "list google workspace groups")
	}
	return res.Groups, res.NextPageToken, nil
}

func (a *DirectoryLowLevelAPI) ListGroupMembers(ctx context.Context, groupKey, pageToken string) ([]*directory.Member, string, error) {
	call := a.service.Members.List(groupKey).MaxResults(groupsPageSize)
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}
	res, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "list google workspace group members")
	}
	return res.Members, res.NextPageToken, nil
}

// newDirectoryAPI selects the real or mock DirectoryAPI based on the service
// account email, mirroring NewGoogleCalendar.
func newDirectoryAPI(apiKeyEmail string) DirectoryAPI {
	if apiKeyEmail == MockEmail {
		return newMockDirectoryAPI()
	}
	return &DirectoryLowLevelAPI{}
}

// syncStats tracks counts surfaced in the integration's last-sync status.
type syncStats struct {
	usersUpserted   int
	usersSkipped    int
	usersDeleted    int
	groupsUpserted  int
	groupsDeleted   int
	membersSkipped  int
	membersResolved int
}

// Sync pulls all users and groups from Google Workspace and reconciles them into
// the SCIM tables. Google Workspace is treated as authoritative: SCIM users and
// groups that are no longer present in Workspace are deleted, but only when the
// full directory fetch succeeded (a fetch error aborts before any deletion so we
// never delete on partial data).
func Sync(ctx context.Context, ds fleet.Datastore, intg *fleet.GoogleWorkspaceIntegration, logger *slog.Logger) error {
	email := intg.ApiKey.Values[fleet.GoogleCalendarEmail]
	privateKey := intg.ApiKey.Values[fleet.GoogleCalendarPrivateKey]
	customerID := intg.CustomerID
	if customerID == "" {
		customerID = fleet.DefaultGoogleWorkspaceCustomerID
	}

	api := newDirectoryAPI(email)
	if err := api.Configure(ctx, email, privateKey, intg.AdminEmail); err != nil {
		return ctxerr.Wrap(ctx, err, "configure google workspace directory api")
	}

	var stats syncStats

	// 1. Sync users. googleIDToScimID maps the Google immutable user id to the
	// SCIM user id, used to resolve group memberships in step 2.
	seenUserExternalIDs := make(map[string]struct{})
	googleIDToScimID := make(map[string]uint)
	var pageToken string
	for {
		users, next, err := api.ListUsers(ctx, customerID, pageToken)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list users")
		}
		for _, u := range users {
			scimID, skipped, err := upsertUser(ctx, ds, u, logger)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "upsert user")
			}
			if skipped {
				stats.usersSkipped++
				continue
			}
			stats.usersUpserted++
			seenUserExternalIDs[u.Id] = struct{}{}
			googleIDToScimID[u.Id] = scimID
		}
		if next == "" {
			break
		}
		pageToken = next
	}

	// 2. Sync groups and their memberships.
	seenGroupExternalIDs := make(map[string]struct{})
	pageToken = ""
	for {
		groups, next, err := api.ListGroups(ctx, customerID, pageToken)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list groups")
		}
		for _, g := range groups {
			memberIDs, err := resolveGroupMemberScimIDs(ctx, api, g.Id, googleIDToScimID, &stats, logger)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "resolve group members")
			}
			if err := upsertGroup(ctx, ds, g, memberIDs, logger); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert group")
			}
			stats.groupsUpserted++
			seenGroupExternalIDs[g.Id] = struct{}{}
		}
		if next == "" {
			break
		}
		pageToken = next
	}

	// 3. Reconcile deletes. Only reached if every fetch above succeeded.
	usersDeleted, err := reconcileDeletedUsers(ctx, ds, seenUserExternalIDs, logger)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reconcile deleted users")
	}
	stats.usersDeleted = usersDeleted

	groupsDeleted, err := reconcileDeletedGroups(ctx, ds, seenGroupExternalIDs, logger)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reconcile deleted groups")
	}
	stats.groupsDeleted = groupsDeleted

	logger.InfoContext(ctx, "google workspace idp sync complete",
		"users_upserted", stats.usersUpserted,
		"users_skipped", stats.usersSkipped,
		"users_deleted", stats.usersDeleted,
		"groups_upserted", stats.groupsUpserted,
		"groups_deleted", stats.groupsDeleted,
		"members_resolved", stats.membersResolved,
		"members_skipped", stats.membersSkipped,
	)
	return nil
}

// upsertUser maps a Google Workspace user to a SCIM user and creates or replaces
// it. It returns the SCIM user id, or skipped=true if the user was skipped
// (e.g. no primary email, which Fleet uses as the SCIM userName).
func upsertUser(ctx context.Context, ds fleet.Datastore, u *directory.User, logger *slog.Logger) (id uint, skipped bool, err error) {
	if u.PrimaryEmail == "" {
		logger.WarnContext(ctx, "skipping google workspace user without primary email", "google_user_id", u.Id)
		return 0, true, nil
	}

	user := &fleet.ScimUser{
		UserName:   u.PrimaryEmail,
		ExternalID: new(u.Id),
		Active:     new(!(u.Suspended || u.Archived)),
		Emails: []fleet.ScimUserEmail{{
			Email:   u.PrimaryEmail,
			Primary: new(true),
			Type:    new(emailTypeWork),
		}},
	}
	if u.Name != nil {
		if u.Name.GivenName != "" {
			user.GivenName = new(u.Name.GivenName)
		}
		if u.Name.FamilyName != "" {
			user.FamilyName = new(u.Name.FamilyName)
		}
	}
	if dept := primaryDepartment(u.Organizations); dept != "" {
		user.Department = new(dept)
	}

	existing, err := ds.ScimUserByUserName(ctx, u.PrimaryEmail)
	switch {
	case err == nil:
		user.ID = existing.ID
		if err := ds.ReplaceScimUser(ctx, user); err != nil {
			return 0, false, ctxerr.Wrap(ctx, err, "replace scim user")
		}
		return existing.ID, false, nil
	case fleet.IsNotFound(err):
		newID, err := ds.CreateScimUser(ctx, user)
		if err != nil {
			return 0, false, ctxerr.Wrap(ctx, err, "create scim user")
		}
		return newID, false, nil
	default:
		return 0, false, ctxerr.Wrap(ctx, err, "lookup scim user by user name")
	}
}

// upsertGroup maps a Google Workspace group to a SCIM group (with its resolved
// member SCIM user ids) and creates or replaces it.
func upsertGroup(ctx context.Context, ds fleet.Datastore, g *directory.Group, memberIDs []uint, logger *slog.Logger) error {
	displayName := g.Name
	if displayName == "" {
		displayName = g.Email
	}
	if displayName == "" {
		logger.WarnContext(ctx, "skipping google workspace group without name or email", "google_group_id", g.Id)
		return nil
	}

	group := &fleet.ScimGroup{
		DisplayName: displayName,
		ExternalID:  new(g.Id),
		ScimUsers:   memberIDs,
	}

	existing, err := ds.ScimGroupByDisplayName(ctx, displayName)
	switch {
	case err == nil:
		group.ID = existing.ID
		return ctxerr.Wrap(ctx, ds.ReplaceScimGroup(ctx, group), "replace scim group")
	case fleet.IsNotFound(err):
		_, err := ds.CreateScimGroup(ctx, group)
		return ctxerr.Wrap(ctx, err, "create scim group")
	default:
		return ctxerr.Wrap(ctx, err, "lookup scim group by display name")
	}
}

// resolveGroupMemberScimIDs pages through a group's members and resolves each
// (user) member to its SCIM user id via googleIDToScimID. Nested group members
// and members not synced as SCIM users (e.g. external members) are skipped.
func resolveGroupMemberScimIDs(
	ctx context.Context, api DirectoryAPI, groupID string, googleIDToScimID map[string]uint, stats *syncStats, logger *slog.Logger,
) ([]uint, error) {
	var memberIDs []uint
	seen := make(map[uint]struct{})
	var pageToken string
	for {
		members, next, err := api.ListGroupMembers(ctx, groupID, pageToken)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "list group members")
		}
		for _, m := range members {
			if m.Type == memberTypeGroup {
				// Fleet's SCIM model does not support nested groups.
				stats.membersSkipped++
				logger.DebugContext(ctx, "skipping nested group member", "google_group_id", groupID, "member_id", m.Id)
				continue
			}
			scimID, ok := googleIDToScimID[m.Id]
			if !ok {
				stats.membersSkipped++
				continue
			}
			if _, dup := seen[scimID]; dup {
				continue
			}
			seen[scimID] = struct{}{}
			memberIDs = append(memberIDs, scimID)
			stats.membersResolved++
		}
		if next == "" {
			break
		}
		pageToken = next
	}
	return memberIDs, nil
}

// reconcileDeletedUsers deletes SCIM users whose Google id (external_id) was not
// seen during this sync. Returns the number of users deleted.
func reconcileDeletedUsers(ctx context.Context, ds fleet.Datastore, seenExternalIDs map[string]struct{}, logger *slog.Logger) (int, error) {
	const perPage = 500
	deleted := 0
	startIndex := uint(1)
	for {
		users, total, err := ds.ListScimUsers(ctx, fleet.ScimUsersListOptions{
			ScimListOptions: fleet.ScimListOptions{StartIndex: startIndex, PerPage: perPage},
		})
		if err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "list scim users")
		}
		for _, u := range users {
			// Only delete rows we own (have a Google external id we manage).
			if u.ExternalID == nil {
				continue
			}
			if _, ok := seenExternalIDs[*u.ExternalID]; ok {
				continue
			}
			if err := ds.DeleteScimUser(ctx, u.ID); err != nil {
				return deleted, ctxerr.Wrap(ctx, err, "delete scim user")
			}
			deleted++
			logger.InfoContext(ctx, "deleted scim user no longer in google workspace", "scim_user_id", u.ID, "user_name", u.UserName)
		}
		startIndex += perPage
		if uint64(startIndex) > uint64(total) {
			break
		}
	}
	return deleted, nil
}

// reconcileDeletedGroups deletes SCIM groups whose Google id (external_id) was
// not seen during this sync. Returns the number of groups deleted.
func reconcileDeletedGroups(ctx context.Context, ds fleet.Datastore, seenExternalIDs map[string]struct{}, logger *slog.Logger) (int, error) {
	const perPage = 500
	deleted := 0
	startIndex := uint(1)
	for {
		groups, total, err := ds.ListScimGroups(ctx, fleet.ScimGroupsListOptions{
			ScimListOptions: fleet.ScimListOptions{StartIndex: startIndex, PerPage: perPage},
			ExcludeUsers:    true,
		})
		if err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "list scim groups")
		}
		for _, g := range groups {
			if g.ExternalID == nil {
				continue
			}
			if _, ok := seenExternalIDs[*g.ExternalID]; ok {
				continue
			}
			if err := ds.DeleteScimGroup(ctx, g.ID); err != nil {
				return deleted, ctxerr.Wrap(ctx, err, "delete scim group")
			}
			deleted++
			logger.InfoContext(ctx, "deleted scim group no longer in google workspace", "scim_group_id", g.ID, "display_name", g.DisplayName)
		}
		startIndex += perPage
		if uint64(startIndex) > uint64(total) {
			break
		}
	}
	return deleted, nil
}

// primaryDepartment extracts the department from the Directory API user's
// Organizations field, which is returned as untyped JSON. It prefers the
// organization marked primary, falling back to the first one with a department.
func primaryDepartment(organizations any) string {
	if organizations == nil {
		return ""
	}
	raw, err := json.Marshal(organizations)
	if err != nil {
		return ""
	}
	var orgs []struct {
		Department string `json:"department"`
		Primary    bool   `json:"primary"`
	}
	if err := json.Unmarshal(raw, &orgs); err != nil {
		return ""
	}
	var fallback string
	for _, o := range orgs {
		if o.Department == "" {
			continue
		}
		if o.Primary {
			return o.Department
		}
		if fallback == "" {
			fallback = o.Department
		}
	}
	return fallback
}
