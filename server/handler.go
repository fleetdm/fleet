package server

import (
	"net/http"

	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func attachAPIRoutes(router *mux.Router, ctx context.Context, svc kolide.Service, opts []kithttp.ServerOption) {
	router.Handle("/api/v1/kolide/login",
		kithttp.NewServer(
			ctx,
			makeLoginEndpoint(svc),
			decodeLoginRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/logout",
		kithttp.NewServer(
			ctx,
			makeLogoutEndpoint(svc),
			decodeNoParamsRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/forgot_password",
		kithttp.NewServer(
			ctx,
			makeForgotPasswordEndpoint(svc),
			decodeForgotPasswordRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/reset_password",
		kithttp.NewServer(
			ctx,
			makeResetPasswordEndpoint(svc),
			decodeResetPasswordRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/me",
		kithttp.NewServer(
			ctx,
			authenticated(makeGetSessionUserEndpoint(svc)),
			decodeNoParamsRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/users",
		kithttp.NewServer(
			ctx,
			authenticated(canPerformActions(makeListUsersEndpoint(svc))),
			decodeNoParamsRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/users",
		kithttp.NewServer(
			ctx,
			authenticated(mustBeAdmin(makeCreateUserEndpoint(svc))),
			decodeCreateUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/users/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(canReadUser(makeGetUserEndpoint(svc))),
			decodeGetUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/users/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(validateModifyUserRequest(makeModifyUserEndpoint(svc))),
			decodeModifyUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("PATCH")

	router.Handle("/api/v1/kolide/users/{id}/sessions",
		kithttp.NewServer(
			ctx,
			authenticated(canReadUser(makeGetInfoAboutSessionsForUserEndpoint(svc))),
			decodeGetInfoAboutSessionsForUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/users/{id}/sessions",
		kithttp.NewServer(
			ctx,
			authenticated(canModifyUser(makeDeleteSessionsForUserEndpoint(svc))),
			decodeDeleteSessionsForUserRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")

	router.Handle("/api/v1/kolide/sessions/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(mustBeAdmin(makeGetInfoAboutSessionEndpoint(svc))),
			decodeGetInfoAboutSessionRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/sessions/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(mustBeAdmin(makeDeleteSessionEndpoint(svc))),
			decodeDeleteSessionRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")

	router.Handle("/api/v1/kolide/config",
		kithttp.NewServer(
			ctx,
			authenticated(makeGetAppConfigEndpoint(svc)),
			decodeNoParamsRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/config",
		kithttp.NewServer(
			ctx,
			authenticated(mustBeAdmin(makeModifyAppConfigRequest(svc))),
			decodeModifyAppConfigRequest,
			encodeResponse,
			opts...,
		),
	).Methods("PATCH")

	router.Handle("/api/v1/kolide/queries/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(makeGetQueryEndpoint(svc)),
			decodeGetQueryRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/queries",
		kithttp.NewServer(
			ctx,
			authenticated(makeGetAllQueriesEndpoint(svc)),
			decodeNoParamsRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/queries",
		kithttp.NewServer(
			ctx,
			authenticated(makeCreateQueryEndpoint(svc)),
			decodeCreateQueryRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/queries/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(makeModifyQueryEndpoint(svc)),
			decodeModifyQueryRequest,
			encodeResponse,
			opts...,
		),
	).Methods("PATCH")

	router.Handle("/api/v1/kolide/queries/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(makeDeleteQueryEndpoint(svc)),
			decodeDeleteQueryRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")

	router.Handle("/api/v1/kolide/packs/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(makeGetPackEndpoint(svc)),
			decodeGetPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/packs",
		kithttp.NewServer(
			ctx,
			authenticated(makeGetAllPacksEndpoint(svc)),
			decodeNoParamsRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/packs",
		kithttp.NewServer(
			ctx,
			authenticated(makeCreatePackEndpoint(svc)),
			decodeCreatePackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("POST")

	router.Handle("/api/v1/kolide/packs/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(makeModifyPackEndpoint(svc)),
			decodeModifyPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("PATCH")

	router.Handle("/api/v1/kolide/packs/{id}",
		kithttp.NewServer(
			ctx,
			authenticated(makeDeletePackEndpoint(svc)),
			decodeDeletePackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")

	router.Handle("/api/v1/kolide/packs/{pid}/queries/{qid}",
		kithttp.NewServer(
			ctx,
			authenticated(makeAddQueryToPackEndpoint(svc)),
			decodeAddQueryToPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/packs/{id}/queries",
		kithttp.NewServer(
			ctx,
			authenticated(makeGetQueriesInPackEndpoint(svc)),
			decodeGetQueriesInPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("GET")

	router.Handle("/api/v1/kolide/packs/{pid}/queries/{qid}",
		kithttp.NewServer(
			ctx,
			authenticated(makeDeleteQueryFromPackEndpoint(svc)),
			decodeDeleteQueryFromPackRequest,
			encodeResponse,
			opts...,
		),
	).Methods("DELETE")
}

// MakeHandler creates an http handler for the Kolide API
func MakeHandler(ctx context.Context, svc kolide.Service, jwtKey string, ds kolide.Datastore, logger kitlog.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerBefore(
			setViewerContext(svc, ds, jwtKey, logger),
		),
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorEncoder(encodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}

	r := mux.NewRouter()
	attachAPIRoutes(r, ctx, svc, opts)

	return r
}
