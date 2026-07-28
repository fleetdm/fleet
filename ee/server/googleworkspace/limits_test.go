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

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// pagingFake is a fake Directory API whose listings keep handing out next-page
// tokens, so pagination stops only when a limit trips or when the listing's page
// count is reached. A perPage of 0 makes a listing return empty pages, which is the
// malformed-response case the page limit exists for: the record limit alone never
// catches it, because the result set never grows.
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
// requests it has served, so a test can show pagination actually stopped.
func (f pagingFake) server(t *testing.T) (*httptest.Server, *atomic.Int64) {
	requests := new(atomic.Int64)

	// servePage returns the requested page index and the token for the next page,
	// empty when this is the last of pages pages.
	servePage := func(r *http.Request, pages int) (int, string) {
		requests.Add(1)
		page, _ := strconv.Atoi(r.URL.Query().Get("pageToken"))
		if pages > 0 && page+1 >= pages {
			return page, ""
		}
		return page, strconv.Itoa(page + 1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "x", "token_type": "Bearer", "expires_in": 3600})
	})
	mux.HandleFunc("GET /admin/directory/v1/users", func(w http.ResponseWriter, r *http.Request) {
		page, next := servePage(r, f.userPages)
		users := make([]map[string]any, 0, f.usersPerPage)
		for i := range f.usersPerPage {
			id := fmt.Sprintf("u%d-%d", page, i)
			users = append(users, map[string]any{"id": id, "primaryEmail": id + "@b.com"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"users": users, "nextPageToken": next})
	})
	mux.HandleFunc("GET /admin/directory/v1/groups", func(w http.ResponseWriter, r *http.Request) {
		page, next := servePage(r, f.groupPages)
		groups := make([]map[string]any, 0, f.groupsPerPage)
		for i := range f.groupsPerPage {
			id := fmt.Sprintf("g%d-%d", page, i)
			groups = append(groups, map[string]any{"id": id, "name": id})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"groups": groups, "nextPageToken": next})
	})
	mux.HandleFunc("GET /admin/directory/v1/groups/{k}/members", func(w http.ResponseWriter, r *http.Request) {
		page, next := servePage(r, f.memberPages)
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
		// wantMaxRequests bounds the API calls the listing may make, so a
		// regression that keeps paginating fails instead of hanging.
		wantMaxRequests int64
	}{
		{
			name:            "users over record limit",
			fake:            pagingFake{usersPerPage: usersPageSize},
			limits:          Limits{MaxUsers: 1000},
			wantErrContains: "exceeded the limit of 1000 users",
			wantMaxRequests: 3,
		},
		{
			// The record limit can never trip here, only the page limit can.
			name:            "users empty pages forever",
			fake:            pagingFake{usersPerPage: 0},
			limits:          Limits{MaxUsers: 1000},
			wantErrContains: "users listing did not complete within 6 pages",
			wantMaxRequests: 7,
		},
		{
			name:            "users under limit",
			fake:            pagingFake{usersPerPage: 2, userPages: 2},
			limits:          Limits{MaxUsers: 1000},
			wantMaxRequests: 2,
		},
		{
			name:            "users unlimited",
			fake:            pagingFake{usersPerPage: 2, userPages: 3},
			limits:          Limits{MaxUsers: 0},
			wantMaxRequests: 3,
		},
		{
			name:            "groups over record limit",
			fake:            pagingFake{groupsPerPage: groupsPageSize, membersPerPage: 1, memberPages: 1},
			listGroups:      true,
			limits:          Limits{MaxGroups: 400},
			wantErrContains: "exceeded the limit of 400 groups",
			wantMaxRequests: 3,
		},
		{
			name:            "groups empty pages forever",
			fake:            pagingFake{groupsPerPage: 0},
			listGroups:      true,
			limits:          Limits{MaxGroups: 400},
			wantErrContains: "groups listing did not complete within 6 pages",
			wantMaxRequests: 7,
		},
		{
			name:            "members of one group over record limit",
			fake:            pagingFake{groupsPerPage: 1, groupPages: 1, membersPerPage: membersPageSize},
			listGroups:      true,
			limits:          Limits{MaxGroupMembers: 400},
			wantErrContains: "exceeded the limit of 400 members of group g0-0",
			// One groups page plus the member pages for the first group.
			wantMaxRequests: 4,
		},
		{
			name:            "members of one group empty pages forever",
			fake:            pagingFake{groupsPerPage: 1, groupPages: 1, membersPerPage: 0},
			listGroups:      true,
			limits:          Limits{MaxGroupMembers: 400},
			wantErrContains: "members of group g0-0 listing did not complete within 6 pages",
			wantMaxRequests: 8,
		},
		{
			name:            "total memberships over limit",
			fake:            pagingFake{groupsPerPage: 3, groupPages: 1, membersPerPage: 4, memberPages: 1},
			listGroups:      true,
			limits:          Limits{MaxGroupMemberships: 10},
			wantErrContains: "exceeded the limit of 10 total group memberships",
			// Groups page plus member listings for the first three groups.
			wantMaxRequests: 4,
		},
		{
			name:            "groups and memberships under limits",
			fake:            pagingFake{groupsPerPage: 3, groupPages: 1, membersPerPage: 4, memberPages: 1},
			listGroups:      true,
			limits:          Limits{MaxGroups: 400, MaxGroupMembers: 400, MaxGroupMemberships: 12},
			wantMaxRequests: 4,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, requests := tc.fake.server(t)
			dir := newTestDirectory(t, srv, pemKey, tc.limits)

			var err error
			if tc.listGroups {
				_, err = dir.ListGroups(t.Context())
			} else {
				_, err = dir.ListUsers(t.Context())
			}

			if tc.wantErrContains == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
			}
			require.LessOrEqual(t, requests.Load(), tc.wantMaxRequests)
		})
	}
}

func TestMaxPages(t *testing.T) {
	// Twice the pages a full result set needs, so partially filled pages don't trip
	// the guard before the record limit does.
	require.Equal(t, 4, maxPages(500, 500))
	require.Equal(t, 6, maxPages(1000, 500))
	require.Equal(t, 2, maxPages(1, 500))
	require.Equal(t, 2002, maxPages(500_000, 500))
}
