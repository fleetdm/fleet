// Package googleworkspace implements pulling users and groups from a Google
// Workspace directory via the Admin SDK Directory API, using a service account
// with domain-wide delegation. It maps Google's data model onto Fleet's ScimUser
// so the sync engine can populate IdP host vitals (the scim_* tables).
package googleworkspace

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	directory "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
)

// Page sizes for the Directory API. Users.List allows up to 500; Groups.List and
// Members.List allow up to 200.
const (
	usersPageSize   = 500
	groupsPageSize  = 200
	membersPageSize = 200
)

// Field projections limiting each listing to what Fleet reads, so the API doesn't
// serialize and Fleet doesn't parse the rest of the resource (a user carries
// thumbnails, custom schemas, phones, addresses, aliases, SSH keys and more).
//
// nextPageToken must be present or Pages() cannot advance. name, organizations, and
// emails are requested whole rather than sub-selected: the Admin SDK types the latter
// two as `any`, and a sub-selection that doesn't match the response shape would
// silently yield empty departments or emails instead of failing. Keep in sync with
// mapUser, groupDisplayName, and the member filter in Directory.ListGroups.
const (
	usersFields   = "nextPageToken,users(id,primaryEmail,suspended,archived,name,organizations,emails)"
	groupsFields  = "nextPageToken,groups(id,name,email)"
	membersFields = "nextPageToken,members(id,type)"
)

// tokenURIKey is the key holding the OAuth2 token endpoint in a service-account
// JSON. Real Google service-account JSON always includes it; we honor it so a
// QA/load-test fake can route the token exchange to a stub endpoint.
const tokenURIKey = "token_uri"

// endpointOverrideEnv, when set, redirects the Admin SDK Directory API base URL
// (e.g. to the gw-directory-fake tool). For QA/load testing only — never set it
// in production.
const endpointOverrideEnv = "FLEET_TEST_GOOGLE_WORKSPACE_ENDPOINT"

// directoryScopes are the read-only Admin SDK Directory API scopes that must be
// authorized for the service account's client ID via domain-wide delegation in
// the Google Admin console.
var directoryScopes = []string{
	directory.AdminDirectoryUserReadonlyScope,
	directory.AdminDirectoryGroupReadonlyScope,
	directory.AdminDirectoryGroupMemberReadonlyScope,
}

// defaultMaxPagesPerListing bounds page iteration for every listing, regardless of
// the configured Limits. It catches what a record limit cannot: a malformed response
// or a proxy that keeps returning a next-page token with empty (or barely filled)
// pages never grows the result set, so only a page count stops it. It is deliberately
// far above what a real listing needs — the default user limit is reached in 1,000
// pages — so it never fails a sync that would otherwise complete.
const defaultMaxPagesPerListing = 10_000

// Limits bound how much of a directory a single sync pass pulls, guarding against
// holding an unbounded directory in memory. They are safety rails, not tuning knobs:
// no real Google Workspace tenant is expected to reach them. Defaults and their
// rationale live with the configuration in server/config. A limit of 0 is disabled
// (server config validation rejects negatives); the page cap always applies.
//
// Exceeding a limit fails the sync rather than truncating the pull: reconciliation
// treats the pull as authoritative and deletes any scim user/group missing from it,
// so a truncated directory would destroy IdP data.
type Limits struct {
	// MaxUsers bounds users returned by one users.list pass.
	MaxUsers int
	// MaxGroups bounds groups returned by one groups.list pass.
	MaxGroups int
	// MaxGroupMembers bounds members returned for a single group.
	MaxGroupMembers int
	// MaxGroupMemberships bounds the total memberships kept across all groups in
	// one pass, which is the dominant memory term for a large directory.
	MaxGroupMemberships int
	// maxPages replaces defaultMaxPagesPerListing when set. Unexported: it is not an
	// operator setting, only a way for tests to reach the page cap cheaply without
	// mutating package state.
	maxPages int
}

// pageCap is the page-iteration cap in force for a listing.
func (l Limits) pageCap() int {
	if l.maxPages > 0 {
		return l.maxPages
	}
	return defaultMaxPagesPerListing
}

// Config keys the limits come from, named in the errors so an operator hitting a
// limit knows what to raise.
const (
	maxUsersSetting            = "google_workspace.max_users"
	maxGroupsSetting           = "google_workspace.max_groups"
	maxGroupMembersSetting     = "google_workspace.max_group_members"
	maxGroupMembershipsSetting = "google_workspace.max_group_memberships"
)

// lowLevelAPI is the minimal Admin SDK Directory API surface the Directory needs.
// It exists so tests can supply a fake implementation without hitting Google.
//
// ListUsers hands over one page at a time so the caller can map each page and let it
// go: a raw *directory.User costs around 2 KB, mostly map[string]any overhead for the
// fields the Admin SDK types as `any`, which is four times what Fleet's mapped
// ScimUser costs. Accumulating them all was the largest term in a sync's memory use.
// Groups and members are returned whole — their raw objects are a tenth the size, and
// streaming groups would mean issuing a members.list for every group in the middle of
// an in-flight groups.list pagination.
type lowLevelAPI interface {
	ListUsers(ctx context.Context, domain string, forEachPage func(users []*directory.User) error) error
	ListGroups(ctx context.Context, domain string) ([]*directory.Group, error)
	ListGroupMembers(ctx context.Context, groupKey string) ([]*directory.Member, error)
}

// Directory implements fleet.GoogleWorkspaceDirectory.
type Directory struct {
	api    lowLevelAPI
	domain string
	logger *slog.Logger
	limits Limits
}

// NewDirectory builds a Directory that talks to the real Admin SDK Directory API
// using the integration's service account and impersonated admin user.
func NewDirectory(ctx context.Context, intg *fleet.GoogleWorkspaceIntegration, logger *slog.Logger, limits Limits) (fleet.GoogleWorkspaceDirectory, error) {
	api, err := newGoogleAPI(ctx, intg, logger, limits)
	if err != nil {
		return nil, err
	}
	return &Directory{api: api, domain: intg.Domain, logger: logger, limits: limits}, nil
}

// NewDirectoryFactory returns a NewDirectory bound to limits, for injecting into
// the sync cron (it matches cron.GoogleWorkspaceDirectoryFactory).
func NewDirectoryFactory(limits Limits) func(context.Context, *fleet.GoogleWorkspaceIntegration, *slog.Logger) (fleet.GoogleWorkspaceDirectory, error) {
	return func(ctx context.Context, intg *fleet.GoogleWorkspaceIntegration, logger *slog.Logger) (fleet.GoogleWorkspaceDirectory, error) {
		return NewDirectory(ctx, intg, logger, limits)
	}
}

func (d *Directory) log() *slog.Logger {
	if d.logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return d.logger
}

// ListUsers returns every user in the configured domain mapped to a ScimUser. Each
// page is mapped as it arrives so the raw Directory objects can be collected.
func (d *Directory) ListUsers(ctx context.Context) ([]*fleet.ScimUser, error) {
	logger := d.log()
	var out []*fleet.ScimUser
	err := d.api.ListUsers(ctx, d.domain, func(users []*directory.User) error {
		for _, u := range users {
			// A user with no ID or primary email cannot be linked to a host, so skip it.
			if u.Id == "" || u.PrimaryEmail == "" {
				logger.DebugContext(ctx, "skipping google workspace user with missing id or primary email",
					"id", u.Id, "primary_email", u.PrimaryEmail)
				continue
			}
			su := mapUser(u)
			logger.DebugContext(ctx, "ingested google workspace user",
				"external_id", u.Id,
				"user_name", su.UserName,
				"active", derefBool(su.Active),
				"department", derefString(su.Department),
				"num_emails", len(su.Emails),
				// Raw organizations as returned by the Directory API, to diagnose
				// missing department values (empty/absent means the API returned none).
				"raw_organizations", rawJSON(u.Organizations),
			)
			out = append(out, su)
		}
		return nil
	})
	if err != nil {
		// Never return what was mapped before the failure: the sync deletes every
		// scim user missing from the pull.
		return nil, ctxerr.Wrap(ctx, err, "list google workspace users")
	}
	return out, nil
}

// ListGroups returns every group in the configured domain with its members'
// external IDs (Google user IDs).
func (d *Directory) ListGroups(ctx context.Context) ([]*fleet.GoogleWorkspaceGroup, error) {
	groups, err := d.api.ListGroups(ctx, d.domain)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list google workspace groups")
	}
	out := make([]*fleet.GoogleWorkspaceGroup, 0, len(groups))
	memberships := 0
	for _, g := range groups {
		if g.Id == "" {
			continue
		}
		members, err := d.api.ListGroupMembers(ctx, g.Id)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "list members of google workspace group %s", g.Id)
		}
		memberIDs := make([]string, 0, len(members))
		for _, m := range members {
			if m.Id == "" {
				continue
			}
			// Only direct user members are mapped; nested groups are not expanded in v1.
			if m.Type != "" && m.Type != "USER" {
				continue
			}
			memberIDs = append(memberIDs, m.Id)
		}
		// Every group's members are held until the whole pull is reconciled, so cap
		// the running total, not just each group.
		memberships += len(memberIDs)
		if d.limits.MaxGroupMemberships > 0 && memberships > d.limits.MaxGroupMemberships {
			return nil, ctxerr.Errorf(ctx, "exceeded the limit of %d total group memberships; raise %s to sync this domain",
				d.limits.MaxGroupMemberships, maxGroupMembershipsSetting)
		}
		d.log().DebugContext(ctx, "ingested google workspace group",
			"external_id", g.Id,
			"display_name", groupDisplayName(g),
			"num_members", len(memberIDs),
		)
		out = append(out, &fleet.GoogleWorkspaceGroup{
			ExternalID:        g.Id,
			DisplayName:       groupDisplayName(g),
			MemberExternalIDs: memberIDs,
		})
	}
	return out, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	return b != nil && *b
}

// rawJSON marshals a value to a compact JSON string for debug logging. It returns
// an empty string for nil and "<unmarshalable>" if marshaling fails.
func rawJSON(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "<unmarshalable>"
	}
	return string(b)
}

// mapUser maps a Google Directory user onto a Fleet ScimUser. ExternalID is the
// Google user ID; group membership is resolved separately from ListGroups.
func mapUser(u *directory.User) *fleet.ScimUser {
	active := !u.Suspended && !u.Archived
	su := &fleet.ScimUser{
		ExternalID: &u.Id,
		UserName:   u.PrimaryEmail,
		Active:     &active,
	}
	if u.Name != nil {
		if gn := strings.TrimSpace(u.Name.GivenName); gn != "" {
			su.GivenName = new(gn)
		}
		if fn := strings.TrimSpace(u.Name.FamilyName); fn != "" {
			su.FamilyName = new(fn)
		}
	}
	if dept := primaryDepartment(parseOrganizations(u.Organizations)); dept != "" {
		su.Department = new(dept)
	}
	su.Emails = mapEmails(u.PrimaryEmail, parseEmails(u.Emails))
	return su
}

func groupDisplayName(g *directory.Group) string {
	if name := strings.TrimSpace(g.Name); name != "" {
		return name
	}
	return strings.TrimSpace(g.Email)
}

// mapEmails maps Google's emails onto ScimUserEmail, de-duplicating by address
// (case-insensitive) and guaranteeing the primary email is present and flagged
// primary — the host↔user linking matches on the primary email.
func mapEmails(primaryEmail string, raw []directoryEmail) []fleet.ScimUserEmail {
	seen := make(map[string]int, len(raw))
	out := make([]fleet.ScimUserEmail, 0, len(raw))
	for _, e := range raw {
		addr := strings.TrimSpace(e.Address)
		if addr == "" {
			continue
		}
		if _, dup := seen[strings.ToLower(addr)]; dup {
			continue
		}
		em := fleet.ScimUserEmail{Email: addr, Primary: new(e.Primary)}
		if e.Type != "" {
			em.Type = new(e.Type)
		}
		seen[strings.ToLower(addr)] = len(out)
		out = append(out, em)
	}

	primaryEmail = strings.TrimSpace(primaryEmail)
	if primaryEmail == "" {
		return out
	}
	if idx, ok := seen[strings.ToLower(primaryEmail)]; ok {
		out[idx].Primary = new(true)
		return out
	}
	// Primary email wasn't in the emails array; prepend it.
	return append([]fleet.ScimUserEmail{{Email: primaryEmail, Primary: new(true)}}, out...)
}

// primaryDepartment returns the department of the primary organization, falling
// back to the first organization with a non-empty department.
func primaryDepartment(orgs []directoryOrganization) string {
	var fallback string
	for _, o := range orgs {
		dept := strings.TrimSpace(o.Department)
		if dept == "" {
			continue
		}
		if o.Primary {
			return dept
		}
		if fallback == "" {
			fallback = dept
		}
	}
	return fallback
}

// Google's directory.User exposes Emails and Organizations as untyped JSON
// (any), so we parse the slices we need via a JSON round-trip.

type directoryEmail struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Primary bool   `json:"primary"`
}

type directoryOrganization struct {
	Department string `json:"department"`
	Primary    bool   `json:"primary"`
}

func parseEmails(raw any) []directoryEmail {
	var out []directoryEmail
	jsonRoundTrip(raw, &out)
	return out
}

func parseOrganizations(raw any) []directoryOrganization {
	var out []directoryOrganization
	jsonRoundTrip(raw, &out)
	return out
}

func jsonRoundTrip(raw any, dst any) {
	if raw == nil {
		return
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return
	}
	// Best effort: malformed shapes simply yield no values.
	_ = json.Unmarshal(b, dst)
}

// googleAPI is the production lowLevelAPI backed by the Admin SDK Directory API.
type googleAPI struct {
	service *directory.Service
	limits  Limits
}

func newGoogleAPI(ctx context.Context, intg *fleet.GoogleWorkspaceIntegration, logger *slog.Logger, limits Limits) (*googleAPI, error) {
	// Honor token_uri from the service-account JSON (real GSA JSON always carries
	// it). Falls back to Google's endpoint when absent.
	tokenURL := google.JWTTokenURL
	if v := intg.ApiKey.Values[tokenURIKey]; v != "" {
		tokenURL = v
	}

	conf := &jwt.Config{
		Email:      intg.ApiKey.Values[fleet.GoogleCalendarEmail],
		Scopes:     directoryScopes,
		PrivateKey: []byte(intg.ApiKey.Values[fleet.GoogleCalendarPrivateKey]),
		TokenURL:   tokenURL,
		Subject:    intg.ImpersonatedUserEmail,
	}

	opts := []option.ClientOption{option.WithHTTPClient(conf.Client(ctx))}
	if endpoint := os.Getenv(endpointOverrideEnv); endpoint != "" {
		// QA/load-test only: redirect the Directory API to a fake server.
		if logger != nil {
			logger.WarnContext(ctx, "using Google Workspace Directory API endpoint override; do not use in production",
				"env", endpointOverrideEnv, "endpoint", endpoint)
		}
		opts = append(opts, option.WithEndpoint(endpoint))
	}

	service, err := directory.NewService(ctx, opts...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create google workspace directory service")
	}
	return &googleAPI{service: service, limits: limits}, nil
}

// checkListLimits aborts pagination once a listing has reached more than recordLimit
// records, or has served its last allowed page with more still to come. Returning an
// error from a Pages callback stops iteration and surfaces the error to the caller,
// which fails the sync — the alternative, a truncated pull, would make reconciliation
// delete every record past the limit.
//
// records counts the page just received, before it is retained, so a breach doesn't
// hold a page past the limit. hasMore reports whether the response carried a
// next-page token, so a listing that ends exactly on the last allowed page succeeds
// and no request is spent past the cap.
//
// The messages stay terse because they end up in the 255-character sync status
// column, and the setting name is the part an operator needs; the caller's error
// wrapping already says this is Google Workspace.
func (a *googleAPI) checkListLimits(ctx context.Context, records, pages int, hasMore bool, recordLimit int, resource, setting string) error {
	if recordLimit > 0 && records > recordLimit {
		return ctxerr.Errorf(ctx, "exceeded the limit of %d %s; raise %s to sync this domain",
			recordLimit, resource, setting)
	}
	if pageLimit := a.limits.pageCap(); hasMore && pages >= pageLimit {
		return ctxerr.Errorf(ctx, "%s listing exceeded the limit of %d pages; the API kept returning next-page tokens",
			resource, pageLimit)
	}
	return nil
}

func (a *googleAPI) ListUsers(ctx context.Context, domain string, forEachPage func(users []*directory.User) error) error {
	pages, records := 0, 0
	err := a.service.Users.List().Domain(domain).MaxResults(usersPageSize).Fields(usersFields).Pages(ctx, func(page *directory.Users) error {
		pages++
		records += len(page.Users)
		// Check before handing the page over, so an over-limit page isn't mapped.
		if err := a.checkListLimits(ctx, records, pages, page.NextPageToken != "", a.limits.MaxUsers, "users", maxUsersSetting); err != nil {
			return err
		}
		return forEachPage(page.Users)
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "google workspace users.list")
	}
	return nil
}

func (a *googleAPI) ListGroups(ctx context.Context, domain string) ([]*directory.Group, error) {
	var groups []*directory.Group
	pages := 0
	err := a.service.Groups.List().Domain(domain).MaxResults(groupsPageSize).Fields(groupsFields).Pages(ctx, func(page *directory.Groups) error {
		pages++
		// Check before retaining the page, so a breach doesn't hold groups past the limit.
		if err := a.checkListLimits(ctx, len(groups)+len(page.Groups), pages, page.NextPageToken != "", a.limits.MaxGroups, "groups", maxGroupsSetting); err != nil {
			return err
		}
		groups = append(groups, page.Groups...)
		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "google workspace groups.list")
	}
	return groups, nil
}

func (a *googleAPI) ListGroupMembers(ctx context.Context, groupKey string) ([]*directory.Member, error) {
	var members []*directory.Member
	pages := 0
	// The group is named by the caller's error wrapping, and the sync status column
	// this error lands in is only 255 characters, so don't repeat it here.
	err := a.service.Members.List(groupKey).MaxResults(membersPageSize).Fields(membersFields).Pages(ctx, func(page *directory.Members) error {
		pages++
		if err := a.checkListLimits(ctx, len(members)+len(page.Members), pages, page.NextPageToken != "", a.limits.MaxGroupMembers, "group members", maxGroupMembersSetting); err != nil {
			return err
		}
		members = append(members, page.Members...)
		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "google workspace members.list")
	}
	return members, nil
}
