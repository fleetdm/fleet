// Package googleworkspace implements pulling users and groups from a Google
// Workspace directory via the Admin SDK Directory API, using a service account
// with domain-wide delegation. It maps Google's data model onto Fleet's ScimUser
// so the sync engine can populate IdP host vitals (the scim_* tables).
package googleworkspace

import (
	"context"
	"encoding/json"
	"fmt"
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

// Limits bound how much of a directory a single sync pass pulls. They are safety
// rails, not tuning knobs: no real Google Workspace tenant is expected to reach
// them. They guard against runaway pagination (a malformed response or a proxy can
// keep handing out page tokens indefinitely) and against holding an unbounded
// directory in memory. A zero or negative value means unlimited.
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
type lowLevelAPI interface {
	ListUsers(ctx context.Context, domain string) ([]*directory.User, error)
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

// ListUsers returns every user in the configured domain mapped to a ScimUser.
func (d *Directory) ListUsers(ctx context.Context) ([]*fleet.ScimUser, error) {
	users, err := d.api.ListUsers(ctx, d.domain)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list google workspace users")
	}
	logger := d.log()
	out := make([]*fleet.ScimUser, 0, len(users))
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
			return nil, ctxerr.Errorf(ctx, "google workspace directory exceeded the limit of %d total group memberships; raise %s to sync this domain",
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
		ExternalID: new(u.Id),
		UserName:   u.PrimaryEmail,
		Active:     new(active),
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
	seen := make(map[string]int, len(raw)+1)
	out := make([]fleet.ScimUserEmail, 0, len(raw)+1)
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

// maxPages bounds page iteration independently of the record limit. A malformed
// response or a misbehaving proxy can return empty pages with a next-page token
// forever, which a record limit alone never catches because the result set never
// grows. Allowing twice the pages a completely full result set needs leaves room
// for the partially filled pages the Directory API may return.
func maxPages(recordLimit, pageSize int) int {
	return (recordLimit/pageSize + 1) * 2
}

// checkPageLimits aborts pagination once a list call has collected more than
// recordLimit records or iterated past the page limit derived from it. Returning an
// error from a Pages callback stops iteration and surfaces the error to the caller,
// which fails the sync — the alternative, a truncated pull, would make
// reconciliation delete every record past the limit.
func checkPageLimits(ctx context.Context, records, pages, recordLimit, pageSize int, resource, setting string) error {
	if recordLimit <= 0 {
		return nil
	}
	if records > recordLimit {
		return ctxerr.Errorf(ctx, "google workspace directory exceeded the limit of %d %s; raise %s to sync this domain",
			recordLimit, resource, setting)
	}
	if limit := maxPages(recordLimit, pageSize); pages > limit {
		return ctxerr.Errorf(ctx, "google workspace %s listing did not complete within %d pages; raise %s if the domain is that large",
			resource, limit, setting)
	}
	return nil
}

func (a *googleAPI) ListUsers(ctx context.Context, domain string) ([]*directory.User, error) {
	var users []*directory.User
	pages := 0
	err := a.service.Users.List().Domain(domain).MaxResults(usersPageSize).Pages(ctx, func(page *directory.Users) error {
		users = append(users, page.Users...)
		pages++
		return checkPageLimits(ctx, len(users), pages, a.limits.MaxUsers, usersPageSize, "users", maxUsersSetting)
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "google workspace users.list")
	}
	return users, nil
}

func (a *googleAPI) ListGroups(ctx context.Context, domain string) ([]*directory.Group, error) {
	var groups []*directory.Group
	pages := 0
	err := a.service.Groups.List().Domain(domain).MaxResults(groupsPageSize).Pages(ctx, func(page *directory.Groups) error {
		groups = append(groups, page.Groups...)
		pages++
		return checkPageLimits(ctx, len(groups), pages, a.limits.MaxGroups, groupsPageSize, "groups", maxGroupsSetting)
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "google workspace groups.list")
	}
	return groups, nil
}

func (a *googleAPI) ListGroupMembers(ctx context.Context, groupKey string) ([]*directory.Member, error) {
	var members []*directory.Member
	pages := 0
	resource := fmt.Sprintf("members of group %s", groupKey)
	err := a.service.Members.List(groupKey).MaxResults(membersPageSize).Pages(ctx, func(page *directory.Members) error {
		members = append(members, page.Members...)
		pages++
		return checkPageLimits(ctx, len(members), pages, a.limits.MaxGroupMembers, membersPageSize, resource, maxGroupMembersSetting)
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "google workspace members.list")
	}
	return members, nil
}
