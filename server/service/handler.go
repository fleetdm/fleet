package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kolide/kolide/server/kolide"
	"github.com/prometheus/client_golang/prometheus"
)

// KolideEndpoints is a collection of RPC endpoints implemented by the Kolide API.
type KolideEndpoints struct {
	Login                          endpoint.Endpoint
	Logout                         endpoint.Endpoint
	ForgotPassword                 endpoint.Endpoint
	ResetPassword                  endpoint.Endpoint
	Me                             endpoint.Endpoint
	ChangePassword                 endpoint.Endpoint
	CreateUser                     endpoint.Endpoint
	GetUser                        endpoint.Endpoint
	ListUsers                      endpoint.Endpoint
	ModifyUser                     endpoint.Endpoint
	AdminUser                      endpoint.Endpoint
	EnableUser                     endpoint.Endpoint
	RequirePasswordReset           endpoint.Endpoint
	PerformRequiredPasswordReset   endpoint.Endpoint
	GetSessionsForUserInfo         endpoint.Endpoint
	DeleteSessionsForUser          endpoint.Endpoint
	GetSessionInfo                 endpoint.Endpoint
	DeleteSession                  endpoint.Endpoint
	GetAppConfig                   endpoint.Endpoint
	ModifyAppConfig                endpoint.Endpoint
	CreateInvite                   endpoint.Endpoint
	ListInvites                    endpoint.Endpoint
	DeleteInvite                   endpoint.Endpoint
	VerifyInvite                   endpoint.Endpoint
	GetQuery                       endpoint.Endpoint
	ListQueries                    endpoint.Endpoint
	CreateQuery                    endpoint.Endpoint
	ModifyQuery                    endpoint.Endpoint
	DeleteQuery                    endpoint.Endpoint
	DeleteQueries                  endpoint.Endpoint
	CreateDistributedQueryCampaign endpoint.Endpoint
	GetPack                        endpoint.Endpoint
	ListPacks                      endpoint.Endpoint
	CreatePack                     endpoint.Endpoint
	ModifyPack                     endpoint.Endpoint
	DeletePack                     endpoint.Endpoint
	ScheduleQuery                  endpoint.Endpoint
	GetScheduledQueriesInPack      endpoint.Endpoint
	GetScheduledQuery              endpoint.Endpoint
	ModifyScheduledQuery           endpoint.Endpoint
	DeleteScheduledQuery           endpoint.Endpoint
	EnrollAgent                    endpoint.Endpoint
	GetClientConfig                endpoint.Endpoint
	GetDistributedQueries          endpoint.Endpoint
	SubmitDistributedQueryResults  endpoint.Endpoint
	SubmitLogs                     endpoint.Endpoint
	GetLabel                       endpoint.Endpoint
	ListLabels                     endpoint.Endpoint
	CreateLabel                    endpoint.Endpoint
	DeleteLabel                    endpoint.Endpoint
	ModifyLabel                    endpoint.Endpoint
	ListDecorators                 endpoint.Endpoint
	NewDecorator                   endpoint.Endpoint
	ModifyDecorator                endpoint.Endpoint
	DeleteDecorator                endpoint.Endpoint
	GetHost                        endpoint.Endpoint
	DeleteHost                     endpoint.Endpoint
	ListHosts                      endpoint.Endpoint
	GetHostSummary                 endpoint.Endpoint
	SearchTargets                  endpoint.Endpoint
	GetOptions                     endpoint.Endpoint
	ModifyOptions                  endpoint.Endpoint
	ResetOptions                   endpoint.Endpoint
	ImportConfig                   endpoint.Endpoint
	GetCertificate                 endpoint.Endpoint
	ChangeEmail                    endpoint.Endpoint
	UpdateLicense                  endpoint.Endpoint
	GetLicense                     endpoint.Endpoint
}

// MakeKolideServerEndpoints creates the Kolide API endpoints.
func MakeKolideServerEndpoints(svc kolide.Service, jwtKey string) KolideEndpoints {
	return KolideEndpoints{
		Login:          makeLoginEndpoint(svc),
		Logout:         makeLogoutEndpoint(svc),
		ForgotPassword: makeForgotPasswordEndpoint(svc),
		ResetPassword:  makeResetPasswordEndpoint(svc),
		CreateUser:     makeCreateUserEndpoint(svc),
		VerifyInvite:   makeVerifyInviteEndpoint(svc),

		// Authenticated user endpoints
		// Each of these endpoints should have exactly one
		// authorization check around the make.*Endpoint method. At a
		// minimum, canPerformActions. Some endpoints use
		// stricter/different checks and should NOT also use
		// canPerformActions (these other checks should also call
		// canPerformActions if that is appropriate).
		Me:                   authenticatedUser(jwtKey, svc, canPerformActions(makeGetSessionUserEndpoint(svc))),
		ChangePassword:       authenticatedUser(jwtKey, svc, canPerformActions(makeChangePasswordEndpoint(svc))),
		GetUser:              authenticatedUser(jwtKey, svc, canReadUser(makeGetUserEndpoint(svc))),
		ListUsers:            authenticatedUser(jwtKey, svc, canPerformActions(makeListUsersEndpoint(svc))),
		ModifyUser:           authenticatedUser(jwtKey, svc, canPerformActions(makeModifyUserEndpoint(svc))),
		AdminUser:            authenticatedUser(jwtKey, svc, mustBeAdmin(makeAdminUserEndpoint(svc))),
		EnableUser:           authenticatedUser(jwtKey, svc, mustBeAdmin(makeEnableUserEndpoint(svc))),
		RequirePasswordReset: authenticatedUser(jwtKey, svc, mustBeAdmin(makeRequirePasswordResetEndpoint(svc))),
		// PerformRequiredPasswordReset needs only to authenticate the
		// logged in user
		PerformRequiredPasswordReset:   authenticatedUser(jwtKey, svc, makePerformRequiredPasswordResetEndpoint(svc)),
		GetSessionsForUserInfo:         authenticatedUser(jwtKey, svc, canReadUser(makeGetInfoAboutSessionsForUserEndpoint(svc))),
		DeleteSessionsForUser:          authenticatedUser(jwtKey, svc, canModifyUser(makeDeleteSessionsForUserEndpoint(svc))),
		GetSessionInfo:                 authenticatedUser(jwtKey, svc, mustBeAdmin(makeGetInfoAboutSessionEndpoint(svc))),
		DeleteSession:                  authenticatedUser(jwtKey, svc, mustBeAdmin(makeDeleteSessionEndpoint(svc))),
		GetAppConfig:                   authenticatedUser(jwtKey, svc, canPerformActions(makeGetAppConfigEndpoint(svc))),
		ModifyAppConfig:                authenticatedUser(jwtKey, svc, mustBeAdmin(makeModifyAppConfigEndpoint(svc))),
		CreateInvite:                   authenticatedUser(jwtKey, svc, mustBeAdmin(makeCreateInviteEndpoint(svc))),
		ListInvites:                    authenticatedUser(jwtKey, svc, mustBeAdmin(makeListInvitesEndpoint(svc))),
		DeleteInvite:                   authenticatedUser(jwtKey, svc, mustBeAdmin(makeDeleteInviteEndpoint(svc))),
		GetQuery:                       authenticatedUser(jwtKey, svc, makeGetQueryEndpoint(svc)),
		ListQueries:                    authenticatedUser(jwtKey, svc, makeListQueriesEndpoint(svc)),
		CreateQuery:                    authenticatedUser(jwtKey, svc, makeCreateQueryEndpoint(svc)),
		ModifyQuery:                    authenticatedUser(jwtKey, svc, makeModifyQueryEndpoint(svc)),
		DeleteQuery:                    authenticatedUser(jwtKey, svc, makeDeleteQueryEndpoint(svc)),
		DeleteQueries:                  authenticatedUser(jwtKey, svc, makeDeleteQueriesEndpoint(svc)),
		CreateDistributedQueryCampaign: authenticatedUser(jwtKey, svc, makeCreateDistributedQueryCampaignEndpoint(svc)),
		GetPack:                   authenticatedUser(jwtKey, svc, makeGetPackEndpoint(svc)),
		ListPacks:                 authenticatedUser(jwtKey, svc, makeListPacksEndpoint(svc)),
		CreatePack:                authenticatedUser(jwtKey, svc, makeCreatePackEndpoint(svc)),
		ModifyPack:                authenticatedUser(jwtKey, svc, makeModifyPackEndpoint(svc)),
		DeletePack:                authenticatedUser(jwtKey, svc, makeDeletePackEndpoint(svc)),
		ScheduleQuery:             authenticatedUser(jwtKey, svc, makeScheduleQueryEndpoint(svc)),
		GetScheduledQueriesInPack: authenticatedUser(jwtKey, svc, makeGetScheduledQueriesInPackEndpoint(svc)),
		GetScheduledQuery:         authenticatedUser(jwtKey, svc, makeGetScheduledQueryEndpoint(svc)),
		ModifyScheduledQuery:      authenticatedUser(jwtKey, svc, makeModifyScheduledQueryEndpoint(svc)),
		DeleteScheduledQuery:      authenticatedUser(jwtKey, svc, makeDeleteScheduledQueryEndpoint(svc)),
		GetHost:                   authenticatedUser(jwtKey, svc, makeGetHostEndpoint(svc)),
		ListHosts:                 authenticatedUser(jwtKey, svc, makeListHostsEndpoint(svc)),
		GetHostSummary:            authenticatedUser(jwtKey, svc, makeGetHostSummaryEndpoint(svc)),
		DeleteHost:                authenticatedUser(jwtKey, svc, makeDeleteHostEndpoint(svc)),
		GetLabel:                  authenticatedUser(jwtKey, svc, makeGetLabelEndpoint(svc)),
		ListLabels:                authenticatedUser(jwtKey, svc, makeListLabelsEndpoint(svc)),
		CreateLabel:               authenticatedUser(jwtKey, svc, makeCreateLabelEndpoint(svc)),
		DeleteLabel:               authenticatedUser(jwtKey, svc, makeDeleteLabelEndpoint(svc)),
		ModifyLabel:               authenticatedUser(jwtKey, svc, makeModifyLabelEndpoint(svc)),
		ListDecorators:            authenticatedUser(jwtKey, svc, makeListDecoratorsEndpoint(svc)),
		NewDecorator:              authenticatedUser(jwtKey, svc, makeNewDecoratorEndpoint(svc)),
		ModifyDecorator:           authenticatedUser(jwtKey, svc, makeModifyDecoratorEndpoint(svc)),
		DeleteDecorator:           authenticatedUser(jwtKey, svc, makeDeleteDecoratorEndpoint(svc)),
		SearchTargets:             authenticatedUser(jwtKey, svc, makeSearchTargetsEndpoint(svc)),
		GetOptions:                authenticatedUser(jwtKey, svc, mustBeAdmin(makeGetOptionsEndpoint(svc))),
		ModifyOptions:             authenticatedUser(jwtKey, svc, mustBeAdmin(makeModifyOptionsEndpoint(svc))),
		ResetOptions:              authenticatedUser(jwtKey, svc, mustBeAdmin(makeResetOptionsEndpoint(svc))),
		ImportConfig:              authenticatedUser(jwtKey, svc, makeImportConfigEndpoint(svc)),
		GetCertificate:            authenticatedUser(jwtKey, svc, makeCertificateEndpoint(svc)),
		ChangeEmail:               authenticatedUser(jwtKey, svc, makeChangeEmailEndpoint(svc)),
		UpdateLicense:             authenticatedUser(jwtKey, svc, mustBeAdmin(makeUpdateLicenseEndpoint(svc))),
		GetLicense:                authenticatedUser(jwtKey, svc, makeGetLicenseEndpoint(svc)),

		// Osquery endpoints
		EnrollAgent:                   makeEnrollAgentEndpoint(svc),
		GetClientConfig:               authenticatedHost(svc, makeGetClientConfigEndpoint(svc)),
		GetDistributedQueries:         authenticatedHost(svc, makeGetDistributedQueriesEndpoint(svc)),
		SubmitDistributedQueryResults: authenticatedHost(svc, makeSubmitDistributedQueryResultsEndpoint(svc)),
		SubmitLogs:                    authenticatedHost(svc, makeSubmitLogsEndpoint(svc)),
	}
}

type kolideHandlers struct {
	Login                          http.Handler
	Logout                         http.Handler
	ForgotPassword                 http.Handler
	ResetPassword                  http.Handler
	Me                             http.Handler
	ChangePassword                 http.Handler
	CreateUser                     http.Handler
	GetUser                        http.Handler
	ListUsers                      http.Handler
	ModifyUser                     http.Handler
	AdminUser                      http.Handler
	EnableUser                     http.Handler
	RequirePasswordReset           http.Handler
	PerformRequiredPasswordReset   http.Handler
	GetSessionsForUserInfo         http.Handler
	DeleteSessionsForUser          http.Handler
	GetSessionInfo                 http.Handler
	DeleteSession                  http.Handler
	GetAppConfig                   http.Handler
	ModifyAppConfig                http.Handler
	CreateInvite                   http.Handler
	ListInvites                    http.Handler
	DeleteInvite                   http.Handler
	VerifyInvite                   http.Handler
	GetQuery                       http.Handler
	ListQueries                    http.Handler
	CreateQuery                    http.Handler
	ModifyQuery                    http.Handler
	DeleteQuery                    http.Handler
	DeleteQueries                  http.Handler
	CreateDistributedQueryCampaign http.Handler
	GetPack                        http.Handler
	ListPacks                      http.Handler
	CreatePack                     http.Handler
	ModifyPack                     http.Handler
	DeletePack                     http.Handler
	ScheduleQuery                  http.Handler
	GetScheduledQueriesInPack      http.Handler
	GetScheduledQuery              http.Handler
	ModifyScheduledQuery           http.Handler
	DeleteScheduledQuery           http.Handler
	EnrollAgent                    http.Handler
	GetClientConfig                http.Handler
	GetDistributedQueries          http.Handler
	SubmitDistributedQueryResults  http.Handler
	SubmitLogs                     http.Handler
	GetLabel                       http.Handler
	ListLabels                     http.Handler
	CreateLabel                    http.Handler
	DeleteLabel                    http.Handler
	ModifyLabel                    http.Handler
	ListDecorators                 http.Handler
	NewDecorator                   http.Handler
	ModifyDecorator                http.Handler
	DeleteDecorator                http.Handler
	GetHost                        http.Handler
	DeleteHost                     http.Handler
	ListHosts                      http.Handler
	GetHostSummary                 http.Handler
	SearchTargets                  http.Handler
	GetOptions                     http.Handler
	ModifyOptions                  http.Handler
	ResetOptions                   http.Handler
	ImportConfig                   http.Handler
	GetCertificate                 http.Handler
	ChangeEmail                    http.Handler
	UpdateLicense                  http.Handler
	GetLicense                     http.Handler
}

func makeKolideKitHandlers(e KolideEndpoints, opts []kithttp.ServerOption) *kolideHandlers {
	newServer := func(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc) http.Handler {
		return kithttp.NewServer(e, decodeFn, encodeResponse, opts...)
	}
	return &kolideHandlers{
		Login:                          newServer(e.Login, decodeLoginRequest),
		Logout:                         newServer(e.Logout, decodeNoParamsRequest),
		ForgotPassword:                 newServer(e.ForgotPassword, decodeForgotPasswordRequest),
		ResetPassword:                  newServer(e.ResetPassword, decodeResetPasswordRequest),
		Me:                             newServer(e.Me, decodeNoParamsRequest),
		ChangePassword:                 newServer(e.ChangePassword, decodeChangePasswordRequest),
		CreateUser:                     newServer(e.CreateUser, decodeCreateUserRequest),
		GetUser:                        newServer(e.GetUser, decodeGetUserRequest),
		ListUsers:                      newServer(e.ListUsers, decodeListUsersRequest),
		ModifyUser:                     newServer(e.ModifyUser, decodeModifyUserRequest),
		RequirePasswordReset:           newServer(e.RequirePasswordReset, decodeRequirePasswordResetRequest),
		PerformRequiredPasswordReset:   newServer(e.PerformRequiredPasswordReset, decodePerformRequiredPasswordResetRequest),
		EnableUser:                     newServer(e.EnableUser, decodeEnableUserRequest),
		AdminUser:                      newServer(e.AdminUser, decodeAdminUserRequest),
		GetSessionsForUserInfo:         newServer(e.GetSessionsForUserInfo, decodeGetInfoAboutSessionsForUserRequest),
		DeleteSessionsForUser:          newServer(e.DeleteSessionsForUser, decodeDeleteSessionsForUserRequest),
		GetSessionInfo:                 newServer(e.GetSessionInfo, decodeGetInfoAboutSessionRequest),
		DeleteSession:                  newServer(e.DeleteSession, decodeDeleteSessionRequest),
		GetAppConfig:                   newServer(e.GetAppConfig, decodeNoParamsRequest),
		ModifyAppConfig:                newServer(e.ModifyAppConfig, decodeModifyAppConfigRequest),
		CreateInvite:                   newServer(e.CreateInvite, decodeCreateInviteRequest),
		ListInvites:                    newServer(e.ListInvites, decodeListInvitesRequest),
		DeleteInvite:                   newServer(e.DeleteInvite, decodeDeleteInviteRequest),
		VerifyInvite:                   newServer(e.VerifyInvite, decodeVerifyInviteRequest),
		GetQuery:                       newServer(e.GetQuery, decodeGetQueryRequest),
		ListQueries:                    newServer(e.ListQueries, decodeListQueriesRequest),
		CreateQuery:                    newServer(e.CreateQuery, decodeCreateQueryRequest),
		ModifyQuery:                    newServer(e.ModifyQuery, decodeModifyQueryRequest),
		DeleteQuery:                    newServer(e.DeleteQuery, decodeDeleteQueryRequest),
		DeleteQueries:                  newServer(e.DeleteQueries, decodeDeleteQueriesRequest),
		CreateDistributedQueryCampaign: newServer(e.CreateDistributedQueryCampaign, decodeCreateDistributedQueryCampaignRequest),
		GetPack:                       newServer(e.GetPack, decodeGetPackRequest),
		ListPacks:                     newServer(e.ListPacks, decodeListPacksRequest),
		CreatePack:                    newServer(e.CreatePack, decodeCreatePackRequest),
		ModifyPack:                    newServer(e.ModifyPack, decodeModifyPackRequest),
		DeletePack:                    newServer(e.DeletePack, decodeDeletePackRequest),
		ScheduleQuery:                 newServer(e.ScheduleQuery, decodeScheduleQueryRequest),
		GetScheduledQueriesInPack:     newServer(e.GetScheduledQueriesInPack, decodeGetScheduledQueriesInPackRequest),
		GetScheduledQuery:             newServer(e.GetScheduledQuery, decodeGetScheduledQueryRequest),
		ModifyScheduledQuery:          newServer(e.ModifyScheduledQuery, decodeModifyScheduledQueryRequest),
		DeleteScheduledQuery:          newServer(e.DeleteScheduledQuery, decodeDeleteScheduledQueryRequest),
		EnrollAgent:                   newServer(e.EnrollAgent, decodeEnrollAgentRequest),
		GetClientConfig:               newServer(e.GetClientConfig, decodeGetClientConfigRequest),
		GetDistributedQueries:         newServer(e.GetDistributedQueries, decodeGetDistributedQueriesRequest),
		SubmitDistributedQueryResults: newServer(e.SubmitDistributedQueryResults, decodeSubmitDistributedQueryResultsRequest),
		SubmitLogs:                    newServer(e.SubmitLogs, decodeSubmitLogsRequest),
		GetLabel:                      newServer(e.GetLabel, decodeGetLabelRequest),
		ListLabels:                    newServer(e.ListLabels, decodeListLabelsRequest),
		CreateLabel:                   newServer(e.CreateLabel, decodeCreateLabelRequest),
		DeleteLabel:                   newServer(e.DeleteLabel, decodeDeleteLabelRequest),
		ModifyLabel:                   newServer(e.ModifyLabel, decodeModifyLabelRequest),
		ListDecorators:                newServer(e.ListDecorators, decodeNoParamsRequest),
		NewDecorator:                  newServer(e.NewDecorator, decodeNewDecoratorRequest),
		ModifyDecorator:               newServer(e.ModifyDecorator, decodeModifyDecoratorRequest),
		DeleteDecorator:               newServer(e.DeleteDecorator, decodeDeleteDecoratorRequest),
		GetHost:                       newServer(e.GetHost, decodeGetHostRequest),
		DeleteHost:                    newServer(e.DeleteHost, decodeDeleteHostRequest),
		ListHosts:                     newServer(e.ListHosts, decodeListHostsRequest),
		GetHostSummary:                newServer(e.GetHostSummary, decodeNoParamsRequest),
		SearchTargets:                 newServer(e.SearchTargets, decodeSearchTargetsRequest),
		GetOptions:                    newServer(e.GetOptions, decodeNoParamsRequest),
		ModifyOptions:                 newServer(e.ModifyOptions, decodeModifyOptionsRequest),
		ResetOptions:                  newServer(e.ResetOptions, decodeNoParamsRequest),
		ImportConfig:                  newServer(e.ImportConfig, decodeImportConfigRequest),
		GetCertificate:                newServer(e.GetCertificate, decodeNoParamsRequest),
		ChangeEmail:                   newServer(e.ChangeEmail, decodeChangeEmailRequest),
		UpdateLicense:                 newServer(e.UpdateLicense, decodeLicenseRequest),
		GetLicense:                    newServer(e.GetLicense, decodeNoParamsRequest),
	}
}

// MakeHandler creates an HTTP handler for the Kolide server endpoints.
func MakeHandler(svc kolide.Service, jwtKey string, logger kitlog.Logger) http.Handler {
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
	kolideHandlers := makeKolideKitHandlers(kolideEndpoints, kolideAPIOptions)

	r := mux.NewRouter()
	attachKolideAPIRoutes(r, kolideHandlers)
	addMetrics(r)

	r.PathPrefix("/api/v1/kolide/results/").
		Handler(makeStreamDistributedQueryCampaignResultsHandler(svc, jwtKey, logger)).
		Name("distributed_query_results")

	return r
}

// addMetrics decorates each hander with prometheus instrumentation
func addMetrics(r *mux.Router) {
	walkFn := func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		route.Handler(prometheus.InstrumentHandler(route.GetName(), route.GetHandler()))
		return nil
	}
	r.Walk(walkFn)

}

func attachKolideAPIRoutes(r *mux.Router, h *kolideHandlers) {
	r.Handle("/api/v1/kolide/login", h.Login).Methods("POST").Name("login")
	r.Handle("/api/v1/kolide/logout", h.Logout).Methods("POST").Name("logout")
	r.Handle("/api/v1/kolide/forgot_password", h.ForgotPassword).Methods("POST").Name("forgot_password")
	r.Handle("/api/v1/kolide/reset_password", h.ResetPassword).Methods("POST").Name("reset_password")
	r.Handle("/api/v1/kolide/me", h.Me).Methods("GET").Name("me")
	r.Handle("/api/v1/kolide/change_password", h.ChangePassword).Methods("POST").Name("change_password")
	r.Handle("/api/v1/kolide/perform_required_password_reset", h.PerformRequiredPasswordReset).Methods("POST").Name("perform_required_password_reset")

	r.Handle("/api/v1/kolide/users", h.ListUsers).Methods("GET").Name("list_users")
	r.Handle("/api/v1/kolide/users", h.CreateUser).Methods("POST").Name("create_user")
	r.Handle("/api/v1/kolide/users/{id}", h.GetUser).Methods("GET").Name("get_user")
	r.Handle("/api/v1/kolide/users/{id}", h.ModifyUser).Methods("PATCH").Name("modify_user")
	r.Handle("/api/v1/kolide/users/{id}/enable", h.EnableUser).Methods("POST").Name("enable_user")
	r.Handle("/api/v1/kolide/users/{id}/admin", h.AdminUser).Methods("POST").Name("admin_user")
	r.Handle("/api/v1/kolide/users/{id}/require_password_reset", h.RequirePasswordReset).Methods("POST").Name("require_password_reset")
	r.Handle("/api/v1/kolide/users/{id}/sessions", h.GetSessionsForUserInfo).Methods("GET").Name("get_session_for_user")
	r.Handle("/api/v1/kolide/users/{id}/sessions", h.DeleteSessionsForUser).Methods("DELETE").Name("delete_session_for_user")

	r.Handle("/api/v1/kolide/sessions/{id}", h.GetSessionInfo).Methods("GET").Name("get_session_info")
	r.Handle("/api/v1/kolide/sessions/{id}", h.DeleteSession).Methods("DELETE").Name("delete_session")

	r.Handle("/api/v1/kolide/config/certificate", h.GetCertificate).Methods("GET").Name("get_certificate")
	r.Handle("/api/v1/kolide/config", h.GetAppConfig).Methods("GET").Name("get_app_config")
	r.Handle("/api/v1/kolide/config", h.ModifyAppConfig).Methods("PATCH").Name("modify_app_config")
	r.Handle("/api/v1/kolide/invites", h.CreateInvite).Methods("POST").Name("create_invite")
	r.Handle("/api/v1/kolide/invites", h.ListInvites).Methods("GET").Name("list_invites")
	r.Handle("/api/v1/kolide/invites/{id}", h.DeleteInvite).Methods("DELETE").Name("delete_invite")
	r.Handle("/api/v1/kolide/invites/{token}", h.VerifyInvite).Methods("GET").Name("verify_invite")

	r.Handle("/api/v1/kolide/email/change/{token}", h.ChangeEmail).Methods("GET").Name("change_email")

	r.Handle("/api/v1/kolide/queries/{id}", h.GetQuery).Methods("GET").Name("get_query")
	r.Handle("/api/v1/kolide/queries", h.ListQueries).Methods("GET").Name("list_queries")
	r.Handle("/api/v1/kolide/queries", h.CreateQuery).Methods("POST").Name("create_query")
	r.Handle("/api/v1/kolide/queries/{id}", h.ModifyQuery).Methods("PATCH").Name("modify_query")
	r.Handle("/api/v1/kolide/queries/{id}", h.DeleteQuery).Methods("DELETE").Name("delete_query")
	r.Handle("/api/v1/kolide/queries/delete", h.DeleteQueries).Methods("POST").Name("delete_queries")
	r.Handle("/api/v1/kolide/queries/run", h.CreateDistributedQueryCampaign).Methods("POST").Name("create_distributed_query_campaign")

	r.Handle("/api/v1/kolide/packs/{id}", h.GetPack).Methods("GET").Name("get_pack")
	r.Handle("/api/v1/kolide/packs", h.ListPacks).Methods("GET").Name("list_packs")
	r.Handle("/api/v1/kolide/packs", h.CreatePack).Methods("POST").Name("create_pack")
	r.Handle("/api/v1/kolide/packs/{id}", h.ModifyPack).Methods("PATCH").Name("modify_pack")
	r.Handle("/api/v1/kolide/packs/{id}", h.DeletePack).Methods("DELETE").Name("delete_pack")
	r.Handle("/api/v1/kolide/packs/{id}/scheduled", h.GetScheduledQueriesInPack).Methods("GET").Name("get_scheduled_queries_in_pack")
	r.Handle("/api/v1/kolide/schedule", h.ScheduleQuery).Methods("POST").Name("schedule_query")
	r.Handle("/api/v1/kolide/schedule/{id}", h.GetScheduledQuery).Methods("GET").Name("get_scheduled_query")
	r.Handle("/api/v1/kolide/schedule/{id}", h.ModifyScheduledQuery).Methods("PATCH").Name("modify_scheduled_query")
	r.Handle("/api/v1/kolide/schedule/{id}", h.DeleteScheduledQuery).Methods("DELETE").Name("delete_scheduled_query")
	r.Handle("/api/v1/kolide/labels/{id}", h.GetLabel).Methods("GET").Name("get_label")
	r.Handle("/api/v1/kolide/labels", h.ListLabels).Methods("GET").Name("list_labels")
	r.Handle("/api/v1/kolide/labels", h.CreateLabel).Methods("POST").Name("create_label")
	r.Handle("/api/v1/kolide/labels/{id}", h.DeleteLabel).Methods("DELETE").Name("delete_label")
	r.Handle("/api/v1/kolide/labels/{id}", h.ModifyLabel).Methods("PATCH").Name("modify_label")

	r.Handle("/api/v1/kolide/decorators", h.ListDecorators).Methods("GET").Name("list_decorators")
	r.Handle("/api/v1/kolide/decorators", h.NewDecorator).Methods("POST").Name("create_decorator")
	r.Handle("/api/v1/kolide/decorators/{id}", h.ModifyDecorator).Methods("PATCH").Name("modify_decorator")
	r.Handle("/api/v1/kolide/decorators/{id}", h.DeleteDecorator).Methods("DELETE").Name("delete_decorator")

	r.Handle("/api/v1/kolide/hosts", h.ListHosts).Methods("GET").Name("list_hosts")
	r.Handle("/api/v1/kolide/host_summary", h.GetHostSummary).Methods("GET").Name("get_host_summary")
	r.Handle("/api/v1/kolide/hosts/{id}", h.GetHost).Methods("GET").Name("get_host")
	r.Handle("/api/v1/kolide/hosts/{id}", h.DeleteHost).Methods("DELETE").Name("delete_host")

	r.Handle("/api/v1/kolide/options", h.GetOptions).Methods("GET").Name("get_options")
	r.Handle("/api/v1/kolide/options", h.ModifyOptions).Methods("PATCH").Name("modify_options")
	r.Handle("/api/v1/kolide/options/reset", h.ResetOptions).Methods("GET").Name("reset_options")

	r.Handle("/api/v1/kolide/targets", h.SearchTargets).Methods("POST").Name("search_targets")

	r.Handle("/api/v1/kolide/osquery/config/import", h.ImportConfig).Methods("POST").Name("import_config")

	r.Handle("/api/v1/kolide/license", h.UpdateLicense).Methods("POST").Name("update_license")
	r.Handle("/api/v1/kolide/license", h.GetLicense).Methods("GET").Name("get_license")

	r.Handle("/api/v1/osquery/enroll", h.EnrollAgent).Methods("POST").Name("enroll_agent")
	r.Handle("/api/v1/osquery/config", h.GetClientConfig).Methods("POST").Name("get_client_config")
	r.Handle("/api/v1/osquery/distributed/read", h.GetDistributedQueries).Methods("POST").Name("get_distributed_queries")
	r.Handle("/api/v1/osquery/distributed/write", h.SubmitDistributedQueryResults).Methods("POST").Name("submit_distributed_query_results")
	r.Handle("/api/v1/osquery/log", h.SubmitLogs).Methods("POST").Name("submit_logs")
}

// WithSetup is an http middleware that checks is setup procedures have been completed.
// If setup hasn't been completed it serves the API with a setup middleware.
// If the server is already configured, the default API handler is exposed.
func WithSetup(svc kolide.Service, logger kitlog.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		configRouter := http.NewServeMux()
		configRouter.Handle("/api/v1/setup", kithttp.NewServer(
			makeSetupEndpoint(svc),
			decodeSetupRequest,
			encodeResponse,
		))
		configRouter.Handle("/api/v1/license", kithttp.NewServer(
			makeSetupLicenseEndpoint(svc),
			decodeLicenseRequest,
			encodeResponse,
		))
		// whitelist osqueryd endpoints
		if strings.HasPrefix(r.URL.Path, "/api/v1/osquery") {
			next.ServeHTTP(w, r)
			return
		}
		requireSetup, err := RequireSetup(svc)
		if err != nil {
			logger.Log("msg", "fetching setup info from db", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if requireSetup {
			configRouter.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// RedirectLoginToSetup detects if the setup endpoint should be used. If setup is required it redirect all
// frontend urls to /setup, otherwise the frontend router is used.
func RedirectLoginToSetup(svc kolide.Service, logger kitlog.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		redirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/setup" || r.URL.Path == "/license" {
				next.ServeHTTP(w, r)
				return
			}
			newURL := r.URL
			license, err := svc.License(context.Background())
			if err != nil {
				logger.Log("msg", "fetching license info from db", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if license.Token == nil {
				newURL.Path = "/license"
			} else {
				newURL.Path = "/setup"
			}
			http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
		})

		setupRequired, err := RequireSetup(svc)
		if err != nil {
			logger.Log("msg", "fetching license info from db", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if setupRequired {
			redirect.ServeHTTP(w, r)
			return
		}
		RedirectSetupToLogin(svc, logger, next).ServeHTTP(w, r)
	}
}

// RequireSetup checks to see if the service has a license and has been setup.
// if either of these things has not been done, return true
func RequireSetup(svc kolide.Service) (bool, error) {
	ctx := context.Background()
	license, err := svc.License(ctx)
	if err != nil {
		return false, err
	}
	if license.Token == nil {
		return true, nil
	}
	users, err := svc.ListUsers(ctx, kolide.ListOptions{Page: 0, PerPage: 1})
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		return true, nil
	}
	return false, nil
}

// RedirectSetupToLogin forces the /setup and /license path to be redirected to login. This middleware is used after
// the app has been setup.
func RedirectSetupToLogin(svc kolide.Service, logger kitlog.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/setup" {
			newURL := r.URL
			newURL.Path = "/login"
			http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
			return
		}
		if r.URL.Path == "/license" {
			newURL := r.URL
			newURL.Path = "/login"
			http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	}
}
