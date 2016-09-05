package server

import (
	"net/http"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func attachAPIRoutes(router *mux.Router, ctx context.Context, svc kolide.Service, opts []kithttp.ServerOption) {
	router.Handle("/api/v1/kolide/users",
		kithttp.NewServer(
			ctx,
			mustBeAdmin(makeCreateUserEndpoint(svc)),
			decodeCreateUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/users/{id}",
		kithttp.NewServer(
			ctx,
			canReadUser(makeGetUserEndpoint(svc)),
			decodeGetUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/users/{id}/password",
		kithttp.NewServer(
			ctx,
			canModifyUser(makeChangePasswordEndpoint(svc)),
			decodeChangePasswordRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/users/{id}/role",
		kithttp.NewServer(
			ctx,
			mustBeAdmin(makeUpdateAdminRoleEndpoint(svc)),
			decodeUpdateAdminRoleRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/users/{id}/status",
		kithttp.NewServer(
			ctx,
			canModifyUser(makeUpdateUserStatusEndpoint(svc)),
			decodeUpdateUserStatusRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/users/{id}/sessions",
		kithttp.NewServer(
			ctx,
			canReadUser(makeGetInfoAboutSessionsForUserEndpoint(svc)),
			decodeGetInfoAboutSessionsForUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/users/{id}/sessions",
		kithttp.NewServer(
			ctx,
			canModifyUser(makeDeleteSessionsForUserEndpoint(svc)),
			decodeDeleteSessionsForUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")

	router.Handle("/api/v1/kolide/sessions/{id}",
		kithttp.NewServer(
			ctx,
			mustBeAdmin(makeGetInfoAboutSessionEndpoint(svc)),
			decodeGetInfoAboutSessionRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/sessions/{id}",
		kithttp.NewServer(
			ctx,
			mustBeAdmin(makeDeleteSessionEndpoint(svc)),
			decodeDeleteSessionRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")

	router.Handle("/api/v1/kolide/queries/{id}",
		kithttp.NewServer(
			ctx,
			makeGetQueryEndpoint(svc),
			decodeGetQueryRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/queries",
		kithttp.NewServer(
			ctx,
			makeGetAllQueriesEndpoint(svc),
			decodeNoParamsRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/queries",
		kithttp.NewServer(
			ctx,
			makeCreateQueryEndpoint(svc),
			decodeCreateQueryRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/queries/{id}",
		kithttp.NewServer(
			ctx,
			makeModifyQueryEndpoint(svc),
			decodeModifyQueryRequest,
			encodeResponse,
			opts...,
		),
	).Methods("PATCH")

	router.Handle("/api/v1/kolide/queries/{id}",
		kithttp.NewServer(
			ctx,
			makeDeleteQueryEndpoint(svc),
			decodeDeleteQueryRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")

	router.Handle("/api/v1/kolide/packs/{id}",
		kithttp.NewServer(
			ctx,
			makeGetPackEndpoint(svc),
			decodeGetPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/packs",
		kithttp.NewServer(
			ctx,
			makeGetAllPacksEndpoint(svc),
			decodeNoParamsRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/packs",
		kithttp.NewServer(
			ctx,
			makeCreatePackEndpoint(svc),
			decodeCreatePackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/packs/{id}",
		kithttp.NewServer(
			ctx,
			makeModifyPackEndpoint(svc),
			decodeModifyPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("PATCH")

	router.Handle("/api/v1/kolide/packs/{id}",
		kithttp.NewServer(
			ctx,
			makeDeletePackEndpoint(svc),
			decodeDeletePackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")

	router.Handle("/api/v1/kolide/packs/{pid}/queries/{qid}",
		kithttp.NewServer(
			ctx,
			makeAddQueryToPackEndpoint(svc),
			decodeAddQueryToPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/packs/{id}/queries",
		kithttp.NewServer(
			ctx,
			makeGetQueriesInPackEndpoint(svc),
			decodeGetQueriesInPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/packs/{pid}/queries/{qid}",
		kithttp.NewServer(
			ctx,
			makeDeleteQueryFromPackEndpoint(svc),
			decodeDeleteQueryFromPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")
}

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

	api := mux.NewRouter()
	attachAPIRoutes(api, ctx, svc, opts)

	r := mux.NewRouter()
	r.PathPrefix("/api/v1/kolide").Handler(authMiddleware(svc, logger, api))
	r.Handle("/api/login", login(svc, logger)).Methods("POST")
	r.Handle("/api/logout", logout(svc, logger)).Methods("GET")
	r.PathPrefix("/assets").Handler(http.StripPrefix("/assets", http.FileServer(newBinaryFileSystem("/build"))))

	for _, route := range frontendRoutes {
		r.HandleFunc(route, serveReactApp)
	}

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
	id, _ := idFromRequest(r, "id")
	return context.WithValue(ctx, "request-id", id)
}
