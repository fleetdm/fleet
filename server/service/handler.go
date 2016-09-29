package service

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

// KolideEndpoints is a collection of RPC endpoints implemented by the Kolide API.
type KolideEndpoints struct {
	Login                         endpoint.Endpoint
	Logout                        endpoint.Endpoint
	ForgotPassword                endpoint.Endpoint
	ResetPassword                 endpoint.Endpoint
	Me                            endpoint.Endpoint
	CreateUser                    endpoint.Endpoint
	GetUser                       endpoint.Endpoint
	ListUsers                     endpoint.Endpoint
	ModifyUser                    endpoint.Endpoint
	GetSessionsForUserInfo        endpoint.Endpoint
	DeleteSessionsForUser         endpoint.Endpoint
	GetSessionInfo                endpoint.Endpoint
	DeleteSession                 endpoint.Endpoint
	GetAppConfig                  endpoint.Endpoint
	ModifyAppConfig               endpoint.Endpoint
	CreateInvite                  endpoint.Endpoint
	ListInvites                   endpoint.Endpoint
	DeleteInvite                  endpoint.Endpoint
	GetQuery                      endpoint.Endpoint
	GetAllQueries                 endpoint.Endpoint
	CreateQuery                   endpoint.Endpoint
	ModifyQuery                   endpoint.Endpoint
	DeleteQuery                   endpoint.Endpoint
	GetPack                       endpoint.Endpoint
	GetAllPacks                   endpoint.Endpoint
	CreatePack                    endpoint.Endpoint
	ModifyPack                    endpoint.Endpoint
	DeletePack                    endpoint.Endpoint
	AddQueryToPack                endpoint.Endpoint
	GetQueriesInPack              endpoint.Endpoint
	DeleteQueryFromPack           endpoint.Endpoint
	EnrollAgent                   endpoint.Endpoint
	GetClientConfig               endpoint.Endpoint
	GetDistributedQueries         endpoint.Endpoint
	SubmitDistributedQueryResults endpoint.Endpoint
	SubmitLogs                    endpoint.Endpoint
}

// MakeKolideServerEndpoints creates the Kolide API endpoints.
func MakeKolideServerEndpoints(svc kolide.Service, jwtKey string) KolideEndpoints {
	return KolideEndpoints{
		Login:          makeLoginEndpoint(svc),
		Logout:         makeLogoutEndpoint(svc),
		ForgotPassword: makeForgotPasswordEndpoint(svc),
		ResetPassword:  makeResetPasswordEndpoint(svc),
		CreateUser:     makeCreateUserEndpoint(svc),

		// Authenticated user endpoints
		Me:                     authenticatedUser(jwtKey, svc, makeGetSessionUserEndpoint(svc)),
		GetUser:                authenticatedUser(jwtKey, svc, canReadUser(makeGetUserEndpoint(svc))),
		ListUsers:              authenticatedUser(jwtKey, svc, canPerformActions(makeListUsersEndpoint(svc))),
		ModifyUser:             authenticatedUser(jwtKey, svc, validateModifyUserRequest(makeModifyUserEndpoint(svc))),
		GetSessionsForUserInfo: authenticatedUser(jwtKey, svc, canReadUser(makeGetInfoAboutSessionsForUserEndpoint(svc))),
		DeleteSessionsForUser:  authenticatedUser(jwtKey, svc, canModifyUser(makeDeleteSessionsForUserEndpoint(svc))),
		GetSessionInfo:         authenticatedUser(jwtKey, svc, mustBeAdmin(makeGetInfoAboutSessionEndpoint(svc))),
		DeleteSession:          authenticatedUser(jwtKey, svc, mustBeAdmin(makeDeleteSessionEndpoint(svc))),
		GetAppConfig:           authenticatedUser(jwtKey, svc, makeGetAppConfigEndpoint(svc)),
		ModifyAppConfig:        authenticatedUser(jwtKey, svc, mustBeAdmin(makeModifyAppConfigRequest(svc))),
		CreateInvite:           authenticatedUser(jwtKey, svc, mustBeAdmin(makeCreateInviteEndpoint(svc))),
		ListInvites:            authenticatedUser(jwtKey, svc, mustBeAdmin(makeListInvitesEndpoint(svc))),
		DeleteInvite:           authenticatedUser(jwtKey, svc, mustBeAdmin(makeDeleteInviteEndpoint(svc))),
		GetQuery:               authenticatedUser(jwtKey, svc, makeGetQueryEndpoint(svc)),
		GetAllQueries:          authenticatedUser(jwtKey, svc, makeGetAllQueriesEndpoint(svc)),
		CreateQuery:            authenticatedUser(jwtKey, svc, makeCreateQueryEndpoint(svc)),
		ModifyQuery:            authenticatedUser(jwtKey, svc, makeModifyQueryEndpoint(svc)),
		DeleteQuery:            authenticatedUser(jwtKey, svc, makeDeleteQueryEndpoint(svc)),
		GetPack:                authenticatedUser(jwtKey, svc, makeGetPackEndpoint(svc)),
		GetAllPacks:            authenticatedUser(jwtKey, svc, makeGetAllPacksEndpoint(svc)),
		CreatePack:             authenticatedUser(jwtKey, svc, makeCreatePackEndpoint(svc)),
		ModifyPack:             authenticatedUser(jwtKey, svc, makeModifyPackEndpoint(svc)),
		DeletePack:             authenticatedUser(jwtKey, svc, makeDeletePackEndpoint(svc)),
		AddQueryToPack:         authenticatedUser(jwtKey, svc, makeAddQueryToPackEndpoint(svc)),
		GetQueriesInPack:       authenticatedUser(jwtKey, svc, makeGetQueriesInPackEndpoint(svc)),
		DeleteQueryFromPack:    authenticatedUser(jwtKey, svc, makeDeleteQueryFromPackEndpoint(svc)),

		// Osquery endpoints
		EnrollAgent:                   makeEnrollAgentEndpoint(svc),
		GetClientConfig:               authenticatedHost(svc, makeGetClientConfigEndpoint(svc)),
		GetDistributedQueries:         authenticatedHost(svc, makeGetDistributedQueriesEndpoint(svc)),
		SubmitDistributedQueryResults: authenticatedHost(svc, makeSubmitDistributedQueryResultsEndpoint(svc)),
		SubmitLogs:                    authenticatedHost(svc, makeSubmitLogsEndpoint(svc)),
	}
}

type kolideHandlers struct {
	Login                         *kithttp.Server
	Logout                        *kithttp.Server
	ForgotPassword                *kithttp.Server
	ResetPassword                 *kithttp.Server
	Me                            *kithttp.Server
	CreateUser                    *kithttp.Server
	GetUser                       *kithttp.Server
	ListUsers                     *kithttp.Server
	ModifyUser                    *kithttp.Server
	GetSessionsForUserInfo        *kithttp.Server
	DeleteSessionsForUser         *kithttp.Server
	GetSessionInfo                *kithttp.Server
	DeleteSession                 *kithttp.Server
	GetAppConfig                  *kithttp.Server
	ModifyAppConfig               *kithttp.Server
	CreateInvite                  *kithttp.Server
	ListInvites                   *kithttp.Server
	DeleteInvite                  *kithttp.Server
	GetQuery                      *kithttp.Server
	GetAllQueries                 *kithttp.Server
	CreateQuery                   *kithttp.Server
	ModifyQuery                   *kithttp.Server
	DeleteQuery                   *kithttp.Server
	GetPack                       *kithttp.Server
	GetAllPacks                   *kithttp.Server
	CreatePack                    *kithttp.Server
	ModifyPack                    *kithttp.Server
	DeletePack                    *kithttp.Server
	AddQueryToPack                *kithttp.Server
	GetQueriesInPack              *kithttp.Server
	DeleteQueryFromPack           *kithttp.Server
	EnrollAgent                   *kithttp.Server
	GetClientConfig               *kithttp.Server
	GetDistributedQueries         *kithttp.Server
	SubmitDistributedQueryResults *kithttp.Server
	SubmitLogs                    *kithttp.Server
}

func makeKolideKitHandlers(ctx context.Context, e KolideEndpoints, opts []kithttp.ServerOption) kolideHandlers {
	newServer := func(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc) *kithttp.Server {
		return kithttp.NewServer(ctx, e, decodeFn, encodeResponse, opts...)
	}
	return kolideHandlers{
		Login:                         newServer(e.Login, decodeLoginRequest),
		Logout:                        newServer(e.Logout, decodeNoParamsRequest),
		ForgotPassword:                newServer(e.ForgotPassword, decodeForgotPasswordRequest),
		ResetPassword:                 newServer(e.ResetPassword, decodeResetPasswordRequest),
		Me:                            newServer(e.Me, decodeNoParamsRequest),
		CreateUser:                    newServer(e.CreateUser, decodeCreateUserRequest),
		GetUser:                       newServer(e.GetUser, decodeGetUserRequest),
		ListUsers:                     newServer(e.ListUsers, decodeNoParamsRequest),
		ModifyUser:                    newServer(e.ModifyUser, decodeModifyUserRequest),
		GetSessionsForUserInfo:        newServer(e.GetSessionsForUserInfo, decodeGetInfoAboutSessionsForUserRequest),
		DeleteSessionsForUser:         newServer(e.DeleteSessionsForUser, decodeDeleteSessionsForUserRequest),
		GetSessionInfo:                newServer(e.GetSessionInfo, decodeGetInfoAboutSessionRequest),
		DeleteSession:                 newServer(e.DeleteSession, decodeDeleteSessionRequest),
		GetAppConfig:                  newServer(e.GetAppConfig, decodeNoParamsRequest),
		ModifyAppConfig:               newServer(e.ModifyAppConfig, decodeModifyAppConfigRequest),
		CreateInvite:                  newServer(e.CreateInvite, decodeCreateInviteRequest),
		ListInvites:                   newServer(e.ListInvites, decodeNoParamsRequest),
		DeleteInvite:                  newServer(e.DeleteInvite, decodeDeleteInviteRequest),
		GetQuery:                      newServer(e.GetQuery, decodeGetQueryRequest),
		GetAllQueries:                 newServer(e.GetAllQueries, decodeGetQueryRequest),
		CreateQuery:                   newServer(e.CreateQuery, decodeCreateQueryRequest),
		ModifyQuery:                   newServer(e.ModifyQuery, decodeModifyQueryRequest),
		DeleteQuery:                   newServer(e.DeleteQuery, decodeDeleteQueryRequest),
		GetPack:                       newServer(e.GetPack, decodeGetPackRequest),
		GetAllPacks:                   newServer(e.GetAllPacks, decodeNoParamsRequest),
		CreatePack:                    newServer(e.CreatePack, decodeCreatePackRequest),
		ModifyPack:                    newServer(e.ModifyPack, decodeModifyPackRequest),
		DeletePack:                    newServer(e.DeletePack, decodeDeletePackRequest),
		AddQueryToPack:                newServer(e.AddQueryToPack, decodeAddQueryToPackRequest),
		GetQueriesInPack:              newServer(e.GetQueriesInPack, decodeGetQueriesInPackRequest),
		DeleteQueryFromPack:           newServer(e.DeleteQueryFromPack, decodeDeleteQueryFromPackRequest),
		EnrollAgent:                   newServer(e.EnrollAgent, decodeEnrollAgentRequest),
		GetClientConfig:               newServer(e.GetClientConfig, decodeGetClientConfigRequest),
		GetDistributedQueries:         newServer(e.GetDistributedQueries, decodeGetDistributedQueriesRequest),
		SubmitDistributedQueryResults: newServer(e.SubmitDistributedQueryResults, decodeSubmitDistributedQueryResultsRequest),
		SubmitLogs:                    newServer(e.SubmitLogs, decodeSubmitLogsRequest),
	}
}

// MakeHandler creates an HTTP handler for the Kolide server endpoints.
func MakeHandler(ctx context.Context, svc kolide.Service, jwtKey string, logger kitlog.Logger) http.Handler {
	kolideAPIOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(
			setRequestsContexts(svc, jwtKey),
		),
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorEncoder(encodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}

	kolideEndpoints := MakeKolideServerEndpoints(svc, jwtKey)
	kolideHandlers := makeKolideKitHandlers(ctx, kolideEndpoints, kolideAPIOptions)

	r := mux.NewRouter()
	attachKolideAPIRoutes(r, kolideHandlers)

	return r
}

func attachKolideAPIRoutes(r *mux.Router, h kolideHandlers) {
	r.Handle("/api/v1/kolide/login", h.Login).Methods("POST")
	r.Handle("/api/v1/kolide/logout", h.Logout).Methods("POST")
	r.Handle("/api/v1/kolide/forgot_password", h.ForgotPassword).Methods("POST")
	r.Handle("/api/v1/kolide/reset_password", h.ResetPassword).Methods("POST")
	r.Handle("/api/v1/kolide/me", h.Me).Methods("GET")

	r.Handle("/api/v1/kolide/users", h.ListUsers).Methods("GET")
	r.Handle("/api/v1/kolide/users", h.CreateUser).Methods("POST")
	r.Handle("/api/v1/kolide/users/{id}", h.GetUser).Methods("GET")
	r.Handle("/api/v1/kolide/users/{id}", h.ModifyUser).Methods("PATCH")
	r.Handle("/api/v1/kolide/users/{id}/sessions", h.GetSessionsForUserInfo).Methods("GET")
	r.Handle("/api/v1/kolide/users/{id}/sessions", h.DeleteSessionsForUser).Methods("DELETE")

	r.Handle("/api/v1/kolide/sessions/{id}", h.GetSessionInfo).Methods("GET")
	r.Handle("/api/v1/kolide/sessions/{id}", h.DeleteSession).Methods("DELETE")

	r.Handle("/api/v1/kolide/config", h.GetAppConfig).Methods("GET")
	r.Handle("/api/v1/kolide/config", h.ModifyAppConfig).Methods("PATCH")
	r.Handle("/api/v1/kolide/invites", h.CreateInvite).Methods("POST")
	r.Handle("/api/v1/kolide/invites", h.ListInvites).Methods("GET")
	r.Handle("/api/v1/kolide/invites/{id}", h.DeleteInvite).Methods("DELETE")

	r.Handle("/api/v1/kolide/queries/{id}", h.GetQuery).Methods("GET")
	r.Handle("/api/v1/kolide/queries", h.GetAllQueries).Methods("GET")
	r.Handle("/api/v1/kolide/queries", h.CreateQuery).Methods("POST")
	r.Handle("/api/v1/kolide/queries/{id}", h.ModifyQuery).Methods("PATCH")
	r.Handle("/api/v1/kolide/queries/{id}", h.DeleteQuery).Methods("DELETE")

	r.Handle("/api/v1/kolide/packs/{id}", h.GetPack).Methods("GET")
	r.Handle("/api/v1/kolide/packs", h.GetAllPacks).Methods("GET")
	r.Handle("/api/v1/kolide/packs", h.CreatePack).Methods("POST")
	r.Handle("/api/v1/kolide/packs/{id}", h.ModifyPack).Methods("PATCH")
	r.Handle("/api/v1/kolide/packs/{id}", h.DeletePack).Methods("DELETE")
	r.Handle("/api/v1/kolide/packs/{pid}/queries/{qid}", h.AddQueryToPack).Methods("POST")
	r.Handle("/api/v1/kolide/packs/{id}/queries", h.GetQueriesInPack).Methods("GET")
	r.Handle("/api/v1/kolide/packs/{pid}/queries/{qid}", h.DeleteQueryFromPack).Methods("DELETE")

	r.Handle("/api/v1/osquery/enroll", h.EnrollAgent).Methods("POST")
	r.Handle("/api/v1/osquery/config", h.GetClientConfig).Methods("POST")
	r.Handle("/api/v1/osquery/distributed/read", h.GetDistributedQueries).Methods("POST")
	r.Handle("/api/v1/osquery/distributed/write", h.SubmitDistributedQueryResults).Methods("POST")
	r.Handle("/api/v1/osquery/log", h.SubmitLogs).Methods("POST")
}
