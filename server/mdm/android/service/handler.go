package service

import (
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/authzcheck"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func GetRoutes(fleetSvc fleet.Service, svc android.Service) endpoint_utils.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, fleetSvc, svc, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption) {

	// user-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(fleetSvc, svc, opts, r, apiVersions()...)

	ue.GET("/api/_version_/fleet/android_enterprise/signup_url", androidEnterpriseSignupEndpoint, nil)
	ue.DELETE("/api/_version_/fleet/android_enterprise", androidDeleteEnterpriseEndpoint, nil)

	ue.GET("/api/_version_/fleet/android_enterprise/{id:[0-9]+}/enrollment_token", androidEnrollmentTokenEndpoint,
		androidEnrollmentTokenRequest{})

	// unauthenticated endpoints - most of those are either login-related,
	// invite-related or host-enrolling. So they typically do some kind of
	// one-time authentication by verifying that a valid secret token is provided
	// with the request.
	ne := newNoAuthEndpointer(svc, opts, r, apiVersions()...)

	// Android management
	ne.GET("/api/_version_/fleet/android_enterprise/{id:[0-9]+}/connect", androidEnterpriseSignupCallbackEndpoint,
		androidEnterpriseSignupCallbackRequest{})

}

func apiVersions() []string {
	return []string{"v1"}
}

func newServer(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc, opts []kithttp.ServerOption) http.Handler {
	e = authzcheck.NewMiddleware().AuthzCheck()(e)
	return kithttp.NewServer(e, decodeFn, encodeResponse, opts...)
}
