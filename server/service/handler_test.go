package service

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/server/datastore"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestAPIRoutes(t *testing.T) {
	ds, err := datastore.New("inmem", "")
	assert.Nil(t, err)

	svc, err := newTestService(ds)
	assert.Nil(t, err)

	ctx := context.Background()

	r := mux.NewRouter()
	ke := MakeKolideServerEndpoints(svc, "CHANGEME")
	kh := makeKolideKitHandlers(ctx, ke, nil)
	attachKolideAPIRoutes(r, kh)
	handler := mux.NewRouter()
	handler.PathPrefix("/").Handler(r)

	var routes = []struct {
		verb string
		uri  string
	}{
		{
			verb: "POST",
			uri:  "/api/v1/kolide/users",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/users",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/users/1",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/users/1",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/login",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/forgot_password",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/reset_password",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/me",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/config",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/config",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/invites",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/invites",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/invites/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/queries/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/queries",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/queries",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/queries/1",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/queries/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/packs/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/packs",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/packs",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/packs/1",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/packs/1",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/packs/1/queries/2",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/packs/1/queries",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/packs/1/queries/2",
		},
		{
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
			uri:  "/api/v1/kolide/labels/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/labels",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/labels",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/labels/1",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/labels/1",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/packs/1/labels/2",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/packs/1/labels",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/packs/1/labels/2",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/hosts/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/hosts",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/hosts/1",
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
		})
	}
}
