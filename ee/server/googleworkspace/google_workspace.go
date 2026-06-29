// Package googleworkspace implements pulling users and groups from a Google
// Workspace directory via the Admin SDK Directory API, using a service account
// with domain-wide delegation. It maps Google's data model onto Fleet's ScimUser
// so the sync engine can populate IdP host vitals (the scim_* tables).
package googleworkspace

import (
	"context"
	"encoding/json"
	"log/slog"
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

// directoryScopes are the read-only Admin SDK Directory API scopes that must be
// authorized for the service account's client ID via domain-wide delegation in
// the Google Admin console.
var directoryScopes = []string{
	directory.AdminDirectoryUserReadonlyScope,
	directory.AdminDirectoryGroupReadonlyScope,
	directory.AdminDirectoryGroupMemberReadonlyScope,
}

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
}

// NewDirectory builds a Directory that talks to the real Admin SDK Directory API
// using the integration's service account and impersonated admin user.
func NewDirectory(ctx context.Context, intg *fleet.GoogleWorkspaceIntegration, logger *slog.Logger) (fleet.GoogleWorkspaceDirectory, error) {
	api, err := newGoogleAPI(ctx, intg)
	if err != nil {
		return nil, err
	}
	return &Directory{api: api, domain: intg.Domain, logger: logger}, nil
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
}

func newGoogleAPI(ctx context.Context, intg *fleet.GoogleWorkspaceIntegration) (*googleAPI, error) {
	conf := &jwt.Config{
		Email:      intg.ApiKey.Values[fleet.GoogleCalendarEmail],
		Scopes:     directoryScopes,
		PrivateKey: []byte(intg.ApiKey.Values[fleet.GoogleCalendarPrivateKey]),
		TokenURL:   google.JWTTokenURL,
		Subject:    intg.ImpersonatedUserEmail,
	}
	service, err := directory.NewService(ctx, option.WithHTTPClient(conf.Client(ctx)))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create google workspace directory service")
	}
	return &googleAPI{service: service}, nil
}

func (a *googleAPI) ListUsers(ctx context.Context, domain string) ([]*directory.User, error) {
	var users []*directory.User
	err := a.service.Users.List().Domain(domain).MaxResults(usersPageSize).Pages(ctx, func(page *directory.Users) error {
		users = append(users, page.Users...)
		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "google workspace users.list")
	}
	return users, nil
}

func (a *googleAPI) ListGroups(ctx context.Context, domain string) ([]*directory.Group, error) {
	var groups []*directory.Group
	err := a.service.Groups.List().Domain(domain).MaxResults(groupsPageSize).Pages(ctx, func(page *directory.Groups) error {
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
	err := a.service.Members.List(groupKey).MaxResults(membersPageSize).Pages(ctx, func(page *directory.Members) error {
		members = append(members, page.Members...)
		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "google workspace members.list")
	}
	return members, nil
}
