package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/notifications"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	api_http "github.com/fleetdm/fleet/v4/server/notifications/api/http"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/platform/tracing"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// errMissingHost means the request reached the handler without the host ID
// the device auth middleware should have set. A Fleet programmer error, not
// something a caller can trigger.
var errMissingHost = errors.New("internal error: missing host from request context")

func GetRoutes(svc api.Service, authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, svc, authMiddleware, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, svc api.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption) {
	de := newDeviceAuthenticatedEndpointer(svc, authMiddleware, opts, r, apiVersions()...)

	de.GET("/api/_version_/fleet/device/{token}/notifications/{uuid}", getNotificationEndpoint, api_http.GetNotificationRequest{})
	de.POST("/api/_version_/fleet/device/{token}/notifications/{uuid}/actions", notificationActionEndpoint, api_http.NotificationActionRequest{})
}

// RegisterTracingTiers classifies this context's routes for trace sampling.
func RegisterTracingTiers(registry *tracing.Registry) {
	registry.Register(http.MethodGet, "/api/_version_/fleet/device/{token}/notifications/{uuid}", tracing.TierStandard)
	registry.Register(http.MethodPost, "/api/_version_/fleet/device/{token}/notifications/{uuid}/actions", tracing.TierStandard)
}

func apiVersions() []string {
	return []string{"v1", "latest"}
}

func getNotificationEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetNotificationRequest)

	hostID, ok := notifications.HostIDFromContext(ctx)
	if !ok {
		return api_http.GetNotificationResponse{Err: errMissingHost}
	}

	notification, err := svc.GetNotificationForHost(ctx, hostID, req.UUID)
	if err != nil {
		return api_http.GetNotificationResponse{Err: err}
	}
	return api_http.GetNotificationResponse{Payload: notification.Payload}
}

func notificationActionEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.NotificationActionRequest)

	hostID, ok := notifications.HostIDFromContext(ctx)
	if !ok {
		return api_http.NotificationActionResponse{Err: errMissingHost}
	}

	if err := svc.ApplyAction(ctx, hostID, req.UUID, req.EndUserNotificationAction); err != nil {
		return api_http.NotificationActionResponse{Err: err}
	}
	return api_http.NotificationActionResponse{}
}
