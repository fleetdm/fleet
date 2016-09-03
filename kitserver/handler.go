package kitserver

import (
	"net/http"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

// MakeHandler creates an http handler for the Kolide API
func MakeHandler(ctx context.Context, svc kolide.Service, logger kitlog.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerBefore(
			setViewerContext(svc, logger),
		),
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}

	// make all the endpoints
	// the endpoints are wrapped in middleware with correct permissions
	// this is a bit simplistic, but so are the permissions
	// the reason's it's not a Service interface wrapper instead:
	// - the permissions are too simple to justify it. having 3-4 endpoint Middleware vs wrapping each service method individually.
	// - service API is still not stable yet
	var (
		createUserEndpoint       = mustBeAdmin(makeCreateUserEndpoint(svc))
		getUserEndpoint          = canReadUser(makeGetUserEndpoint(svc))
		changePasswordEndpoint   = canModifyUser(makeChangePasswordEndpoint(svc))
		updateAdminRoleEndpoint  = mustBeAdmin(makeUpdateAdminRoleEndpoint(svc))
		updateUserStatusEndpoint = canModifyUser(makeUpdateUserStatusEndpoint(svc))
	)

	createUserHandler := kithttp.NewServer(
		ctx,
		createUserEndpoint,
		decodeCreateUserRequest,
		encodeResponse,
		opts...,
	)

	getUserHandler := kithttp.NewServer(
		ctx,
		getUserEndpoint,
		decodeGetUserRequest,
		encodeResponse,
		opts...,
	)

	changePasswordHandler := kithttp.NewServer(
		ctx,
		changePasswordEndpoint,
		decodeChangePasswordRequest,
		encodeResponse,
		opts...,
	)

	updateAdminRoleHandler := kithttp.NewServer(
		ctx,
		updateAdminRoleEndpoint,
		decodeUpdateAdminRoleRequest,
		encodeResponse,
		opts...,
	)

	updateUserStatusHandler := kithttp.NewServer(
		ctx,
		updateUserStatusEndpoint,
		decodeUpdateUserStatusRequest,
		encodeResponse,
		opts...,
	)

	api := mux.NewRouter()
	api.Handle("/api/v1/kolide/users", createUserHandler).Methods("POST")
	api.Handle("/api/v1/kolide/users/{id}", getUserHandler).Methods("GET")
	api.Handle("/api/v1/kolide/users/{id}/password", changePasswordHandler).Methods("POST")
	api.Handle("/api/v1/kolide/users/{id}/role", updateAdminRoleHandler).Methods("POST")
	api.Handle("/api/v1/kolide/users/{id}/status", updateUserStatusHandler).Methods("POST")

	r := mux.NewRouter()

	r.PathPrefix("/api/v1/kolide").Handler(authMiddleware(svc, logger, api))
	r.Handle("/api/login", login(svc, logger)).Methods("POST")
	r.Handle("/api/logout", logout(svc, logger)).Methods("GET")
	return r
}

// setViewerContext updates the context with a viewerContext,
// which holds the currently logged in user
func setViewerContext(svc kolide.Service, logger kitlog.Logger) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		sm := svc.NewSessionManager(ctx, nil, r)
		session, err := sm.Session()
		if err != nil {
			logger.Log("err", err, "error-source", "setViewerContext")
			return ctx
		}

		user, err := svc.User(ctx, session.UserID)
		if err != nil {
			logger.Log("err", err, "error-source", "setViewerContext")
			return ctx
		}

		ctx = context.WithValue(ctx, "viewerContext", &viewerContext{
			user: user,
		})
		logger.Log("msg", "viewer context set", "user", user.ID)
		// get the user-id for request
		if strings.Contains(r.URL.Path, "users/") {
			ctx = withUserIDFromRequest(r, ctx)
		}
		return ctx
	}
}

func withUserIDFromRequest(r *http.Request, ctx context.Context) context.Context {
	uid, _ := userIDFromRequest(r)
	return context.WithValue(ctx, "request-id", uid)
}
