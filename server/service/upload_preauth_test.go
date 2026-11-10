package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type uploadPreAuthRoute struct {
	method, template, path, endpointPath string
	inCatalog                            bool
}

var uploadPreAuthRoutes = []uploadPreAuthRoute{
	{"POST", "/software/package", "/api/latest/fleet/software/package", "/api/v1/fleet/software/package", true},
	{"PATCH", "/software/titles/{id}/package", "/api/v1/fleet/software/titles/1/package", "/api/v1/fleet/software/titles/{id}/package", true},
	{"POST", "/bootstrap", "/api/2022-04/fleet/bootstrap", "/api/v1/fleet/bootstrap", true},
	{"POST", "/mdm/bootstrap", "/api/latest/fleet/mdm/bootstrap", "/api/v1/fleet/mdm/bootstrap", false},
	{"POST", "/mdm/apple/bootstrap", "/api/latest/fleet/mdm/apple/bootstrap", "/api/v1/fleet/mdm/apple/bootstrap", false},
}

type uploadPreAuthBody struct {
	io.Reader
	bytesRead int
}

func (b *uploadPreAuthBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.bytesRead += n
	return n, err
}

type uploadPreAuthRecorder struct {
	*httptest.ResponseRecorder
	readDeadlines []time.Time
}

func (w *uploadPreAuthRecorder) SetReadDeadline(deadline time.Time) error {
	w.readDeadlines = append(w.readDeadlines, deadline)
	return nil
}

func newUploadPreAuthTestService(t *testing.T, users map[string]*fleet.User) fleet.Service {
	t.Helper()
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	tokens := make(map[string]*fleet.User, len(users))
	for name, user := range users {
		tokens[strings.ReplaceAll(name, " ", "_")] = user
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		u, ok := tokens[key]
		if !ok {
			return nil, &notFoundError{}
		}
		return &fleet.Session{Key: key, UserID: u.ID, AccessedAt: time.Now(), APIOnly: &u.APIOnly}, nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, session *fleet.Session) error { return nil }
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		for _, u := range users {
			if u.ID == id {
				return u, nil
			}
		}
		return nil, &notFoundError{}
	}
	return svc
}

func serveUploadPreAuth(t *testing.T, svc fleet.Service, route uploadPreAuthRoute, bearer string) (*uploadPreAuthRecorder, int, bool) {
	t.Helper()
	body := &uploadPreAuthBody{Reader: strings.NewReader(strings.Repeat("x", 8192))}
	ran := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ran = true
		_, err := io.Copy(io.Discard, r.Body)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusNoContent)
	})
	router := mux.NewRouter()
	router.Handle("/api/{fleetversion:(?:v1|2022-04|latest)}/fleet"+route.template,
		uploadPreAuth(svc, slog.New(slog.DiscardHandler))(next)).Methods(route.method)
	req := httptest.NewRequest(route.method, route.path, body)
	req.Header.Set("Authorization", "Bearer "+strings.ReplaceAll(bearer, " ", "_"))
	w := &uploadPreAuthRecorder{ResponseRecorder: httptest.NewRecorder()}
	router.ServeHTTP(w, req)
	return w, body.bytesRead, ran
}

func TestUploadPreAuthRoles(t *testing.T) {
	for _, route := range uploadPreAuthRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			cases := []struct {
				name            string
				user            *fleet.User
				status          int
				message, reason string
			}{
				{"global observer", test.UserObserver, http.StatusForbidden, "forbidden", ""},
				{"global observer_plus", test.UserObserverPlus, http.StatusForbidden, "forbidden", ""},
				{"global technician", test.UserTechnician, http.StatusForbidden, "forbidden", ""},
				{"no roles", test.UserNoRoles, http.StatusForbidden, "forbidden", ""},
				{"team1 observer", test.UserTeamObserverTeam1, http.StatusForbidden, "forbidden", ""},
				{"team1 technician", test.UserTeamTechnicianTeam1, http.StatusForbidden, "forbidden", ""},
				{"forced-reset global admin", &fleet.User{GlobalRole: new(fleet.RoleAdmin), AdminForcedPasswordReset: true}, http.StatusUnauthorized, "password reset required", ""},
				{"restricted API-only global admin", &fleet.User{GlobalRole: new(fleet.RoleAdmin), APIOnly: true,
					APIEndpoints: []fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/hosts"}}}, http.StatusForbidden, "Forbidden", auth.EndpointRestrictionDeniedMessage},
				{"global admin", test.UserAdmin, http.StatusNoContent, "", ""},
				{"global gitops", test.UserGitOps, http.StatusNoContent, "", ""},
				{"team1 maintainer", test.UserTeamMaintainerTeam1, http.StatusNoContent, "", ""},
				{"team1 observer and team2 admin", test.UserTeamObserverTeam1TeamAdminTeam2, http.StatusNoContent, "", ""},
				{"route API-only global admin", &fleet.User{GlobalRole: new(fleet.RoleAdmin), APIOnly: true,
					APIEndpoints: []fleet.APIEndpointRef{{Method: route.method, Path: route.endpointPath}}}, http.StatusNoContent, "", ""},
			}
			if !route.inCatalog {
				cases[len(cases)-1].status = http.StatusForbidden
				cases[len(cases)-1].message = "Forbidden"
				cases[len(cases)-1].reason = auth.EndpointRestrictionDeniedMessage
			}
			users := make(map[string]*fleet.User, len(cases))
			for i, tc := range cases {
				u := *tc.user
				u.ID = uint(i + 1)
				users[tc.name] = &u
			}
			svc := newUploadPreAuthTestService(t, users)
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					w, bytesRead, ran := serveUploadPreAuth(t, svc, route, tc.name)
					assert.Equal(t, tc.status, w.Code)
					allowed := tc.status == http.StatusNoContent
					assert.Equal(t, allowed, ran)
					if allowed {
						assert.Equal(t, 8192, bytesRead)
						require.Len(t, w.readDeadlines, 1)
						assert.True(t, w.readDeadlines[0].IsZero())
					} else {
						assert.Zero(t, bytesRead)
						assert.Empty(t, w.readDeadlines)
						var jsonErr endpointer.JsonError
						if assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &jsonErr)) {
							assert.Equal(t, tc.message, jsonErr.Message)
							if tc.reason != "" {
								assert.Contains(t, w.Body.String(), tc.reason)
							}
						}
					}
				})
			}
		})
	}
}

func TestUploadPreAuthDatastoreError(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: connection refused")}
	}
	for _, route := range uploadPreAuthRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w, bytesRead, ran := serveUploadPreAuth(t, svc, route, "token")
			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.False(t, ran)
			assert.Zero(t, bytesRead)
			var jsonErr endpointer.JsonError
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jsonErr))
			assert.Equal(t, platform_http.GenericErrorMessage, jsonErr.Message)
			assert.NotContains(t, w.Body.String(), "connection refused")
		})
	}
}

func TestUploadPreAuthMatchesAuthz(t *testing.T) {
	roles := []string{fleet.RoleAdmin, fleet.RoleMaintainer, fleet.RoleGitOps, fleet.RoleTechnician, fleet.RoleObserver, fleet.RoleObserverPlus}
	users := map[string]*fleet.User{"no roles": {}}
	for _, role := range roles {
		users["global "+role] = &fleet.User{GlobalRole: new(role)}
		users["team1 "+role] = &fleet.User{Teams: []fleet.UserTeam{{ID: 1, Role: role}}}
	}
	users["team1 observer and team2 admin"] = test.UserTeamObserverTeam1TeamAdminTeam2
	users["team1 maintainer and team2 observer"] = &fleet.User{Teams: []fleet.UserTeam{
		{ID: 1, Role: fleet.RoleMaintainer},
		{ID: 2, Role: fleet.RoleObserver},
	}}
	i := uint(1)
	for name, user := range users {
		u := *user
		u.ID = i
		i++
		users[name] = &u
	}
	svc := newUploadPreAuthTestService(t, users)
	authorizer := authz.Must()
	for name, user := range users {
		t.Run(name, func(t *testing.T) {
			ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: user})
			for _, objectType := range []string{"software", "bootstrap"} {
				t.Run(objectType, func(t *testing.T) {
					want := false
					var objects []any
					route := uploadPreAuthRoutes[0]
					if objectType == "software" {
						objects = append(objects, &fleet.SoftwareInstaller{})
					} else {
						route = uploadPreAuthRoutes[2]
					}
					for _, teamID := range []uint{0, 1, 2} {
						if objectType == "software" {
							objects = append(objects, &fleet.SoftwareInstaller{TeamID: new(teamID)})
						} else {
							objects = append(objects, &fleet.MDMAppleBootstrapPackage{TeamID: teamID})
						}
					}
					for _, object := range objects {
						if authorizer.Authorize(ctx, object, fleet.ActionWrite) == nil {
							want = true
						}
					}
					_, _, got := serveUploadPreAuth(t, svc, route, name)
					assert.Equal(t, want, got)
				})
			}
		})
	}
}
