package googleworkspace

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// pagingFake is a fake Directory API whose listings keep handing out next-page
// tokens, so pagination stops only when a limit trips or when the listing's page
// count is reached. A perPage of 0 makes a listing return empty pages, which is the
// malformed-response case the page limit exists for: a record limit never catches
// it, because the result set never grows.
type pagingFake struct {
	usersPerPage int
	// userPages is how many pages the users listing has; 0 means it never ends.
	userPages      int
	groupsPerPage  int
	groupPages     int
	membersPerPage int
	memberPages    int
}

// server starts the fake and returns it along with the number of Directory API
// requests it has served, so a test can show pagination actually stopped. Requests
// past ceiling fail: a regression that stops honoring a limit would otherwise
// paginate until the test binary times out instead of failing an assertion.
func (f pagingFake) server(t *testing.T, ceiling int64) (*httptest.Server, *atomic.Int64) {
	requests := new(atomic.Int64)

	// servePage returns the requested page index and the token for the next page
	// (empty when this is the last of pages pages), or false when the fake has
	// already served every request it was allowed.
	servePage := func(w http.ResponseWriter, r *http.Request, pages int) (int, string, bool) {
		if requests.Add(1) > ceiling {
			http.Error(w, "fake request ceiling exceeded", http.StatusBadRequest)
			return 0, "", false
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("pageToken"))
		if pages > 0 && page+1 >= pages {
			return page, "", true
		}
		return page, strconv.Itoa(page + 1), true
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "x", "token_type": "Bearer", "expires_in": 3600})
	})
	mux.HandleFunc("GET /admin/directory/v1/users", func(w http.ResponseWriter, r *http.Request) {
		page, next, ok := servePage(w, r, f.userPages)
		if !ok {
			return
		}
		users := make([]map[string]any, 0, f.usersPerPage)
		for i := range f.usersPerPage {
			id := fmt.Sprintf("u%d-%d", page, i)
			users = append(users, map[string]any{"id": id, "primaryEmail": id + "@b.com"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"users": users, "nextPageToken": next})
	})
	mux.HandleFunc("GET /admin/directory/v1/groups", func(w http.ResponseWriter, r *http.Request) {
		page, next, ok := servePage(w, r, f.groupPages)
		if !ok {
			return
		}
		groups := make([]map[string]any, 0, f.groupsPerPage)
		for i := range f.groupsPerPage {
			id := fmt.Sprintf("g%d-%d", page, i)
			groups = append(groups, map[string]any{"id": id, "name": id})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"groups": groups, "nextPageToken": next})
	})
	mux.HandleFunc("GET /admin/directory/v1/groups/{k}/members", func(w http.ResponseWriter, r *http.Request) {
		page, next, ok := servePage(w, r, f.memberPages)
		if !ok {
			return
		}
		members := make([]map[string]any, 0, f.membersPerPage)
		for i := range f.membersPerPage {
			members = append(members, map[string]any{"id": fmt.Sprintf("%s-m%d-%d", r.PathValue("k"), page, i), "type": "USER"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"members": members, "nextPageToken": next})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, requests
}

// newTestDirectory points a real Directory client at the fake server through the
// endpoint override seam, so the limits are exercised in the production code path.
func newTestDirectory(t *testing.T, srv *httptest.Server, pemKey []byte, limits Limits) fleet.GoogleWorkspaceDirectory {
	t.Setenv(endpointOverrideEnv, srv.URL)
	intg := &fleet.GoogleWorkspaceIntegration{
		Domain:                "b.com",
		ImpersonatedUserEmail: "admin@b.com",
		ApiKey: fleet.GoogleCalendarApiKey{Values: map[string]string{
			fleet.GoogleCalendarEmail:      "sa@b.com",
			fleet.GoogleCalendarPrivateKey: string(pemKey),
			tokenURIKey:                    srv.URL + "/token",
		}},
	}
	dir, err := NewDirectory(t.Context(), intg, slog.New(slog.DiscardHandler), limits)
	require.NoError(t, err)
	return dir
}

func testServiceAccountKey(t *testing.T) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

// errBudget is how long a limit error may be. cronGoogleWorkspaceSync truncates the
// sync status to fleet.SCIMMaxFieldLength runes after adding its own wrapping, and a
// real Google group ID is 21 characters where the fake's is 4, so a longer message
// loses the setting name that tells the operator what to raise.
const errBudget = fleet.SCIMMaxFieldLength - 55

func TestDirectoryPaginationLimits(t *testing.T) {
	pemKey := testServiceAccountKey(t)

	for _, tc := range []struct {
		name string
		fake pagingFake
		// listGroups lists groups (and their members) instead of users.
		listGroups bool
		limits     Limits
		// wantErrContains is empty when the listing is expected to succeed.
		wantErrContains string
		// wantRecords is the number of users or groups a successful listing returns.
		wantRecords int
		// wantRequests is the exact number of API calls the listing should make; the
		// fake fails anything past it, so a lost limit check fails fast.
		wantRequests int64
	}{
		{
			name:            "users over record limit",
			fake:            pagingFake{usersPerPage: usersPageSize},
			limits:          Limits{MaxUsers: 1000},
			wantErrContains: "exceeded the limit of 1000 users; raise google_workspace.max_users",
			wantRequests:    3,
		},
		{
			// The record limit can never trip here, only the page limit can.
			name:            "users empty pages forever",
			fake:            pagingFake{usersPerPage: 0},
			limits:          Limits{MaxUsers: 1000, maxPages: 5},
			wantErrContains: "users listing exceeded the limit of 5 pages",
			wantRequests:    5,
		},
		{
			// The page limit must hold with the record limit disabled too, or the
			// documented escape hatch would restore the unbounded loop.
			name:            "users empty pages forever with no record limit",
			fake:            pagingFake{usersPerPage: 0},
			limits:          Limits{MaxUsers: 0, maxPages: 5},
			wantErrContains: "users listing exceeded the limit of 5 pages",
			wantRequests:    5,
		},
		{
			name:         "users under limit",
			fake:         pagingFake{usersPerPage: 2, userPages: 2},
			limits:       Limits{MaxUsers: 1000},
			wantRecords:  4,
			wantRequests: 2,
		},
		{
			// A listing whose last page is the last allowed page has nothing more to
			// fetch, so it must succeed rather than trip the page limit.
			name:         "users ending on the last allowed page",
			fake:         pagingFake{usersPerPage: 2, userPages: 5},
			limits:       Limits{MaxUsers: 1000, maxPages: 5},
			wantRecords:  10,
			wantRequests: 5,
		},
		{
			// More users than the record limit in the cases above would allow.
			name:         "users unlimited",
			fake:         pagingFake{usersPerPage: usersPageSize, userPages: 4},
			limits:       Limits{MaxUsers: 0},
			wantRecords:  4 * usersPageSize,
			wantRequests: 4,
		},
		{
			name:            "groups over record limit",
			fake:            pagingFake{groupsPerPage: groupsPageSize, membersPerPage: 1, memberPages: 1},
			listGroups:      true,
			limits:          Limits{MaxGroups: 400},
			wantErrContains: "exceeded the limit of 400 groups; raise google_workspace.max_groups",
			wantRequests:    3,
		},
		{
			name:            "groups empty pages forever with no record limit",
			fake:            pagingFake{groupsPerPage: 0},
			listGroups:      true,
			limits:          Limits{MaxGroups: 0, maxPages: 5},
			wantErrContains: "groups listing exceeded the limit of 5 pages",
			wantRequests:    5,
		},
		{
			name:            "group members over record limit",
			fake:            pagingFake{groupsPerPage: 1, groupPages: 1, membersPerPage: membersPageSize},
			listGroups:      true,
			limits:          Limits{MaxGroupMembers: 400},
			wantErrContains: "exceeded the limit of 400 group members; raise google_workspace.max_group_members",
			// One groups page plus the member pages for the first group.
			wantRequests: 4,
		},
		{
			name:            "group members empty pages forever with no record limit",
			fake:            pagingFake{groupsPerPage: 1, groupPages: 1, membersPerPage: 0},
			listGroups:      true,
			limits:          Limits{MaxGroupMembers: 0, maxPages: 5},
			wantErrContains: "group members listing exceeded the limit of 5 pages",
			wantRequests:    6,
		},
		{
			name:            "total memberships over limit",
			fake:            pagingFake{groupsPerPage: 3, groupPages: 1, membersPerPage: 4, memberPages: 1},
			listGroups:      true,
			limits:          Limits{MaxGroupMemberships: 10},
			wantErrContains: "exceeded the limit of 10 total group memberships; raise google_workspace.max_group_memberships",
			// Groups page plus member listings for the first three groups.
			wantRequests: 4,
		},
		{
			name:         "groups and memberships under limits",
			fake:         pagingFake{groupsPerPage: 3, groupPages: 1, membersPerPage: 4, memberPages: 1},
			listGroups:   true,
			limits:       Limits{MaxGroups: 400, MaxGroupMembers: 400, MaxGroupMemberships: 12},
			wantRecords:  3,
			wantRequests: 4,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, requests := tc.fake.server(t, tc.wantRequests+1)
			dir := newTestDirectory(t, srv, pemKey, tc.limits)

			var (
				records int
				err     error
			)
			if tc.listGroups {
				var groups []*fleet.GoogleWorkspaceGroup
				groups, err = dir.ListGroups(t.Context())
				records = len(groups)
				if tc.wantErrContains != "" {
					// A limit must abort the pull, never return part of it: the sync
					// deletes every scim record missing from what it gets back.
					require.Nil(t, groups)
				}
			} else {
				var users []*fleet.ScimUser
				users, err = dir.ListUsers(t.Context())
				records = len(users)
				if tc.wantErrContains != "" {
					require.Nil(t, users)
				}
			}

			if tc.wantErrContains == "" {
				require.NoError(t, err)
				require.Equal(t, tc.wantRecords, records)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
				require.LessOrEqual(t, utf8.RuneCountInString(err.Error()), errBudget)
			}
			require.Equal(t, tc.wantRequests, requests.Load())
		})
	}
}
