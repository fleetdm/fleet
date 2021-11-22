package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

func TestAPIRoutes(t *testing.T) {
	ds := new(mock.Store)

	svc := newTestService(ds, nil, nil)

	r := mux.NewRouter()
	limitStore, _ := memstore.New(0)
	ke := MakeFleetServerEndpoints(svc, "", limitStore, kitlog.NewNopLogger())
	kh := makeKitHandlers(ke, nil)
	attachFleetAPIRoutes(r, kh)
	handler := mux.NewRouter()
	handler.PathPrefix("/").Handler(r)

	var routes = []struct {
		verb string
		uri  string
	}{
		{
			verb: "POST",
			uri:  "/api/v1/fleet/users",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/users",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/users/1",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/fleet/users/1",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/login",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/forgot_password",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/reset_password",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/me",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/config",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/fleet/config",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/invites",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/invites",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/fleet/invites/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/queries/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/queries",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/queries",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/fleet/queries/1",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/fleet/queries/1",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/queries/delete",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/queries/run",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/packs",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/packs",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/fleet/packs/1",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/fleet/packs/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/packs/1/scheduled",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/schedule",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/fleet/schedule/1",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/fleet/schedule/1",
		}, {
			verb: "POST",
			uri:  "/api/v1/osquery/enroll",
		},
		{
			verb: "POST",
			uri:  "/api/v1/osquery/config",
		},
		{
			verb: "POST",
			uri:  "/api/v1/osquery/distributed/read",
		},
		{
			verb: "POST",
			uri:  "/api/v1/osquery/distributed/write",
		},
		{
			verb: "POST",
			uri:  "/api/v1/osquery/log",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/labels/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/fleet/labels",
		},
		{
			verb: "POST",
			uri:  "/api/v1/fleet/labels",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/fleet/labels/1",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/fleet/hosts/1",
		},
	}

	for _, route := range routes {
		t.Run(fmt.Sprintf(": %v", route.uri), func(st *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(
				recorder,
				httptest.NewRequest(route.verb, route.uri, nil),
			)
			assert.NotEqual(st, 404, recorder.Code)
			assert.NotEqual(st, 405, recorder.Code, route.verb) // if it matches a path but with wrong verb
		})
	}
}

func TestAPIRoutesConflicts(t *testing.T) {
	ds := new(mock.Store)

	svc := newTestService(ds, nil, nil)
	limitStore, _ := memstore.New(0)
	h := MakeHandler(svc, config.TestConfig(), kitlog.NewNopLogger(), limitStore)
	router := h.(*mux.Router)

	type testCase struct {
		name string
		path string
		verb string
		want int
	}
	var cases []testCase

	// build the test cases: for each route, generate a request designed to match
	// it, and override its handler to return a unique status code. If the
	// request doesn't result in that status code, then some other route
	// conflicts with it and took precedence - a route conflict. The route's name
	// is used to name the sub-test for that route.
	status := 200
	reSimpleVar, reNumVar := regexp.MustCompile(`\{(\w+)\}`), regexp.MustCompile(`\{\w+:[^\}]+\}`)
	err := router.Walk(func(route *mux.Route, router *mux.Router, ancestores []*mux.Route) error {
		name := route.GetName()
		path, err := route.GetPathTemplate()
		if err != nil {
			// all our routes should have paths
			return fmt.Errorf("%s: %w", name, err)
		}
		meths, err := route.GetMethods()
		if err != nil || len(meths) == 0 {
			// only route without method is distributed_query_results (websocket)
			if name != "distributed_query_results" {
				return fmt.Errorf(name+" "+path+": %w", err)
			}
			return nil
		}
		path = reSimpleVar.ReplaceAllString(path, "$1")
		// for now at least, the only times we use regexp-constrained vars is
		// for numeric arguments.
		path = reNumVar.ReplaceAllString(path, "1")

		routeStatus := status
		route.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(routeStatus) })
		for _, meth := range meths {
			cases = append(cases, testCase{
				name: name,
				path: path,
				verb: meth,
				want: status,
			})
		}

		status++
		return nil
	})
	require.NoError(t, err)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Log(c.verb, c.path)
			req := httptest.NewRequest(c.verb, c.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			require.Equal(t, c.want, rr.Code)
		})
	}
}

// TODO refactor this test to match new patterns
// func TestModifyUserPermissions(t *testing.T) {
// 	var (
// 		admin, enabled bool
// 		uid            uint
// 	)
// 	ms := new(mock.Store)
// 	ms.SessionByKeyFunc = func(key string) (*fleet.Session, error) {
// 		return &fleet.Session{AccessedAt: time.Now(), UserID: uid, ID: 1}, nil
// 	}
// 	ms.DestroySessionFunc = func(session *fleet.Session) error {
// 		return nil
// 	}
// 	ms.MarkSessionAccessedFunc = func(session *fleet.Session) error {
// 		return nil
// 	}
// 	ms.UserByIDFunc = func(id uint) (*fleet.User, error) {
// 		return &fleet.User{ID: id, Enabled: enabled, Admin: admin}, nil
// 	}
// 	ms.SaveUserFunc = func(u *fleet.User) error {
// 		// Return an error so that the endpoint returns
// 		return errors.New("foo")
// 	}

// 	svc, err := newTestService(ms, nil, nil)
// 	assert.Nil(t, err)
// 	limitStore, _ := memstore.New(0)

// 	handler := MakeHandler(
// 		svc,
// 		config.FleetConfig{},
// 		log.NewNopLogger(),
// 		limitStore,
// 	)

// 	testCases := []struct {
// 		ActingUserID      uint
// 		ActingUserAdmin   bool
// 		ActingUserEnabled bool
// 		TargetUserID      uint
// 		Authorized        bool
// 	}{
// 		// Disabled regular user
// 		{
// 			ActingUserID:      1,
// 			ActingUserAdmin:   false,
// 			ActingUserEnabled: false,
// 			TargetUserID:      1,
// 			Authorized:        false,
// 		},
// 		// Enabled regular user acting on self
// 		{
// 			ActingUserID:      1,
// 			ActingUserAdmin:   false,
// 			ActingUserEnabled: true,
// 			TargetUserID:      1,
// 			Authorized:        true,
// 		},
// 		// Enabled regular user acting on other
// 		{
// 			ActingUserID:      2,
// 			ActingUserAdmin:   false,
// 			ActingUserEnabled: true,
// 			TargetUserID:      1,
// 			Authorized:        false,
// 		},
// 		// Disabled admin user
// 		{
// 			ActingUserID:      1,
// 			ActingUserAdmin:   true,
// 			ActingUserEnabled: false,
// 			TargetUserID:      1,
// 			Authorized:        false,
// 		},
// 		// Enabled admin user acting on self
// 		{
// 			ActingUserID:      1,
// 			ActingUserAdmin:   true,
// 			ActingUserEnabled: true,
// 			TargetUserID:      1,
// 			Authorized:        true,
// 		},
// 		// Enabled admin user acting on other
// 		{
// 			ActingUserID:      2,
// 			ActingUserAdmin:   true,
// 			ActingUserEnabled: true,
// 			TargetUserID:      1,
// 			Authorized:        true,
// 		},
// 	}

// 	for _, tt := range testCases {
// 		t.Run("", func(t *testing.T) {
// 			// Set user params
// 			uid = tt.ActingUserID
// 			admin, enabled = tt.ActingUserAdmin, tt.ActingUserEnabled

// 			recorder := httptest.NewRecorder()
// 			path := fmt.Sprintf("/api/v1/fleet/users/%d", tt.TargetUserID)
// 			request := httptest.NewRequest("PATCH", path, bytes.NewBufferString("{}"))
// 			request.Header.Add("Authorization", "Bearer fake_session_token")

// 			handler.ServeHTTP(recorder, request)
// 			if tt.Authorized {
// 				assert.NotEqual(t, 403, recorder.Code)
// 			} else {
// 				assert.Equal(t, 403, recorder.Code)
// 			}

// 		})
// 	}

// }
