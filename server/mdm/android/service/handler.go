package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func GetRoutes(fleetSvc fleet.Service, svc android.Service) endpoint_utils.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, fleetSvc, svc, opts)
	}
}

const pubSubPushEndpoint = "/api/v1/fleet/android_enterprise/pubsub"

func attachFleetAPIRoutes(r *mux.Router, fleetSvc fleet.Service, svc android.Service, opts []kithttp.ServerOption) {

	// user-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(fleetSvc, svc, opts, r, apiVersions()...)

	ue.GET("/api/_version_/fleet/android_enterprise/signup_url", androidEnterpriseSignupEndpoint, nil)
	ue.DELETE("/api/_version_/fleet/android_enterprise", androidDeleteEnterpriseEndpoint, nil)

	// unauthenticated endpoints
	// They typically do one-time authentication by verifying that a valid secret token is provided with the request.
	ne := newNoAuthEndpointer(fleetSvc, svc, opts, r, apiVersions()...)

	ne.GET("/api/_version_/fleet/android_enterprise/{id:[0-9]+}/connect", androidEnterpriseSignupCallbackEndpoint,
		androidEnterpriseSignupCallbackRequest{})
	ne.GET("/api/_version_/fleet/android_enterprise/enrollment_token", androidEnrollmentTokenEndpoint,
		androidEnrollmentTokenRequest{})
	ne.POST(pubSubPushEndpoint, androidPubSubPushEndpoint, androidPubSubPushRequest{})

}

func apiVersions() []string {
	return []string{"v1"}
}
