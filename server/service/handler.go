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
	Login                  endpoint.Endpoint
	Logout                 endpoint.Endpoint
	ForgotPassword         endpoint.Endpoint
	ResetPassword          endpoint.Endpoint
	Me                     endpoint.Endpoint
	CreateUser             endpoint.Endpoint
	GetUser                endpoint.Endpoint
	ListUsers              endpoint.Endpoint
	ModifyUser             endpoint.Endpoint
	GetSessionsForUserInfo endpoint.Endpoint
	DeleteSessionsForUser  endpoint.Endpoint
	GetSessionInfo         endpoint.Endpoint
	DeleteSession          endpoint.Endpoint
	GetAppConfig           endpoint.Endpoint
	ModifyAppConfig        endpoint.Endpoint
	GetQuery               endpoint.Endpoint
	GetAllQueries          endpoint.Endpoint
	CreateQuery            endpoint.Endpoint
	ModifyQuery            endpoint.Endpoint
	DeleteQuery            endpoint.Endpoint
	GetPack                endpoint.Endpoint
	GetAllPacks            endpoint.Endpoint
	CreatePack             endpoint.Endpoint
	ModifyPack             endpoint.Endpoint
	DeletePack             endpoint.Endpoint
	AddQueryToPack         endpoint.Endpoint
	GetQueriesInPack       endpoint.Endpoint
	DeleteQueryFromPack    endpoint.Endpoint
}

// MakeKolideServerEndpoints creates the Kolide API endpoints.
func MakeKolideServerEndpoints(svc kolide.Service, jwtKey string) KolideEndpoints {
	return KolideEndpoints{
		Login:                  makeLoginEndpoint(svc),
		Logout:                 makeLogoutEndpoint(svc),
		ForgotPassword:         makeForgotPasswordEndpoint(svc),
		ResetPassword:          makeResetPasswordEndpoint(svc),
		Me:                     authenticated(jwtKey, svc, makeGetSessionUserEndpoint(svc)),
		CreateUser:             authenticated(jwtKey, svc, mustBeAdmin(makeCreateUserEndpoint(svc))),
		GetUser:                authenticated(jwtKey, svc, canReadUser(makeGetUserEndpoint(svc))),
		ListUsers:              authenticated(jwtKey, svc, canPerformActions(makeListUsersEndpoint(svc))),
		ModifyUser:             authenticated(jwtKey, svc, validateModifyUserRequest(makeModifyUserEndpoint(svc))),
		GetSessionsForUserInfo: authenticated(jwtKey, svc, canReadUser(makeGetInfoAboutSessionsForUserEndpoint(svc))),
		DeleteSessionsForUser:  authenticated(jwtKey, svc, canModifyUser(makeDeleteSessionsForUserEndpoint(svc))),
		GetSessionInfo:         authenticated(jwtKey, svc, mustBeAdmin(makeGetInfoAboutSessionEndpoint(svc))),
		DeleteSession:          authenticated(jwtKey, svc, mustBeAdmin(makeDeleteSessionEndpoint(svc))),
		GetAppConfig:           authenticated(jwtKey, svc, makeGetAppConfigEndpoint(svc)),
		ModifyAppConfig:        authenticated(jwtKey, svc, mustBeAdmin(makeModifyAppConfigRequest(svc))),
		GetQuery:               authenticated(jwtKey, svc, makeGetQueryEndpoint(svc)),
		GetAllQueries:          authenticated(jwtKey, svc, makeGetAllQueriesEndpoint(svc)),
		CreateQuery:            authenticated(jwtKey, svc, makeCreateQueryEndpoint(svc)),
		ModifyQuery:            authenticated(jwtKey, svc, makeModifyQueryEndpoint(svc)),
		DeleteQuery:            authenticated(jwtKey, svc, makeDeleteQueryEndpoint(svc)),
		GetPack:                authenticated(jwtKey, svc, makeGetPackEndpoint(svc)),
		GetAllPacks:            authenticated(jwtKey, svc, makeGetAllPacksEndpoint(svc)),
		CreatePack:             authenticated(jwtKey, svc, makeCreatePackEndpoint(svc)),
		ModifyPack:             authenticated(jwtKey, svc, makeModifyPackEndpoint(svc)),
		DeletePack:             authenticated(jwtKey, svc, makeDeletePackEndpoint(svc)),
		AddQueryToPack:         authenticated(jwtKey, svc, makeAddQueryToPackEndpoint(svc)),
		GetQueriesInPack:       authenticated(jwtKey, svc, makeGetQueriesInPackEndpoint(svc)),
		DeleteQueryFromPack:    authenticated(jwtKey, svc, makeDeleteQueryFromPackEndpoint(svc)),
	}
}

type kolideHandlers struct {
	Login                  *kithttp.Server
	Logout                 *kithttp.Server
	ForgotPassword         *kithttp.Server
	ResetPassword          *kithttp.Server
	Me                     *kithttp.Server
	CreateUser             *kithttp.Server
	GetUser                *kithttp.Server
	ListUsers              *kithttp.Server
	ModifyUser             *kithttp.Server
	GetSessionsForUserInfo *kithttp.Server
	DeleteSessionsForUser  *kithttp.Server
	GetSessionInfo         *kithttp.Server
	DeleteSession          *kithttp.Server
	GetAppConfig           *kithttp.Server
	ModifyAppConfig        *kithttp.Server
	GetQuery               *kithttp.Server
	GetAllQueries          *kithttp.Server
	CreateQuery            *kithttp.Server
	ModifyQuery            *kithttp.Server
	DeleteQuery            *kithttp.Server
	GetPack                *kithttp.Server
	GetAllPacks            *kithttp.Server
	CreatePack             *kithttp.Server
	ModifyPack             *kithttp.Server
	DeletePack             *kithttp.Server
	AddQueryToPack         *kithttp.Server
	GetQueriesInPack       *kithttp.Server
	DeleteQueryFromPack    *kithttp.Server
}

func makeKolideKitHandlers(ctx context.Context, e KolideEndpoints, opts []kithttp.ServerOption) kolideHandlers {
	newServer := func(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc) *kithttp.Server {
		return kithttp.NewServer(ctx, e, decodeFn, encodeResponse, opts...)
	}
	return kolideHandlers{
		Login:                  newServer(e.Login, decodeLoginRequest),
		Logout:                 newServer(e.Logout, decodeNoParamsRequest),
		ForgotPassword:         newServer(e.ForgotPassword, decodeForgotPasswordRequest),
		ResetPassword:          newServer(e.ResetPassword, decodeResetPasswordRequest),
		Me:                     newServer(e.Me, decodeNoParamsRequest),
		CreateUser:             newServer(e.CreateUser, decodeCreateUserRequest),
		GetUser:                newServer(e.GetUser, decodeGetUserRequest),
		ListUsers:              newServer(e.ListUsers, decodeNoParamsRequest),
		ModifyUser:             newServer(e.ModifyUser, decodeModifyUserRequest),
		GetSessionsForUserInfo: newServer(e.GetSessionsForUserInfo, decodeGetInfoAboutSessionsForUserRequest),
		DeleteSessionsForUser:  newServer(e.DeleteSessionsForUser, decodeDeleteSessionsForUserRequest),
		GetSessionInfo:         newServer(e.GetSessionInfo, decodeGetInfoAboutSessionRequest),
		DeleteSession:          newServer(e.DeleteSession, decodeDeleteSessionRequest),
		GetAppConfig:           newServer(e.GetAppConfig, decodeNoParamsRequest),
		ModifyAppConfig:        newServer(e.ModifyAppConfig, decodeModifyAppConfigRequest),
		GetQuery:               newServer(e.GetQuery, decodeGetQueryRequest),
		GetAllQueries:          newServer(e.GetAllQueries, decodeGetQueryRequest),
		CreateQuery:            newServer(e.CreateQuery, decodeCreateQueryRequest),
		ModifyQuery:            newServer(e.ModifyQuery, decodeModifyQueryRequest),
		DeleteQuery:            newServer(e.DeleteQuery, decodeDeleteQueryRequest),
		GetPack:                newServer(e.GetPack, decodeGetPackRequest),
		GetAllPacks:            newServer(e.GetAllPacks, decodeNoParamsRequest),
		CreatePack:             newServer(e.CreatePack, decodeCreatePackRequest),
		ModifyPack:             newServer(e.ModifyPack, decodeModifyPackRequest),
		DeletePack:             newServer(e.DeletePack, decodeDeletePackRequest),
		AddQueryToPack:         newServer(e.AddQueryToPack, decodeAddQueryToPackRequest),
		GetQueriesInPack:       newServer(e.GetQueriesInPack, decodeGetQueriesInPackRequest),
		DeleteQueryFromPack:    newServer(e.DeleteQueryFromPack, decodeDeleteQueryFromPackRequest),
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
}
