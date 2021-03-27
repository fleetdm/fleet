package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/service/middleware/ratelimit"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/throttled/throttled/v2"
)

// KolideEndpoints is a collection of RPC endpoints implemented by the Kolide API.
type KolideEndpoints struct {
	Login                                 endpoint.Endpoint
	Logout                                endpoint.Endpoint
	ForgotPassword                        endpoint.Endpoint
	ResetPassword                         endpoint.Endpoint
	Me                                    endpoint.Endpoint
	ChangePassword                        endpoint.Endpoint
	CreateUserWithInvite                  endpoint.Endpoint
	CreateUser                            endpoint.Endpoint
	GetUser                               endpoint.Endpoint
	ListUsers                             endpoint.Endpoint
	ModifyUser                            endpoint.Endpoint
	AdminUser                             endpoint.Endpoint
	EnableUser                            endpoint.Endpoint
	RequirePasswordReset                  endpoint.Endpoint
	PerformRequiredPasswordReset          endpoint.Endpoint
	GetSessionsForUserInfo                endpoint.Endpoint
	DeleteSessionsForUser                 endpoint.Endpoint
	GetSessionInfo                        endpoint.Endpoint
	DeleteSession                         endpoint.Endpoint
	GetAppConfig                          endpoint.Endpoint
	ModifyAppConfig                       endpoint.Endpoint
	ApplyEnrollSecretSpec                 endpoint.Endpoint
	GetEnrollSecretSpec                   endpoint.Endpoint
	CreateInvite                          endpoint.Endpoint
	ListInvites                           endpoint.Endpoint
	DeleteInvite                          endpoint.Endpoint
	VerifyInvite                          endpoint.Endpoint
	GetQuery                              endpoint.Endpoint
	ListQueries                           endpoint.Endpoint
	CreateQuery                           endpoint.Endpoint
	ModifyQuery                           endpoint.Endpoint
	DeleteQuery                           endpoint.Endpoint
	DeleteQueryByID                       endpoint.Endpoint
	DeleteQueries                         endpoint.Endpoint
	ApplyQuerySpecs                       endpoint.Endpoint
	GetQuerySpecs                         endpoint.Endpoint
	GetQuerySpec                          endpoint.Endpoint
	CreateDistributedQueryCampaign        endpoint.Endpoint
	CreateDistributedQueryCampaignByNames endpoint.Endpoint
	CreatePack                            endpoint.Endpoint
	ModifyPack                            endpoint.Endpoint
	GetPack                               endpoint.Endpoint
	ListPacks                             endpoint.Endpoint
	DeletePack                            endpoint.Endpoint
	DeletePackByID                        endpoint.Endpoint
	GetScheduledQueriesInPack             endpoint.Endpoint
	ScheduleQuery                         endpoint.Endpoint
	GetScheduledQuery                     endpoint.Endpoint
	ModifyScheduledQuery                  endpoint.Endpoint
	DeleteScheduledQuery                  endpoint.Endpoint
	ApplyPackSpecs                        endpoint.Endpoint
	GetPackSpecs                          endpoint.Endpoint
	GetPackSpec                           endpoint.Endpoint
	EnrollAgent                           endpoint.Endpoint
	GetClientConfig                       endpoint.Endpoint
	GetDistributedQueries                 endpoint.Endpoint
	SubmitDistributedQueryResults         endpoint.Endpoint
	SubmitLogs                            endpoint.Endpoint
	CarveBegin                            endpoint.Endpoint
	CarveBlock                            endpoint.Endpoint
	CreateLabel                           endpoint.Endpoint
	ModifyLabel                           endpoint.Endpoint
	GetLabel                              endpoint.Endpoint
	ListLabels                            endpoint.Endpoint
	ListHostsInLabel                      endpoint.Endpoint
	DeleteLabel                           endpoint.Endpoint
	DeleteLabelByID                       endpoint.Endpoint
	ApplyLabelSpecs                       endpoint.Endpoint
	GetLabelSpecs                         endpoint.Endpoint
	GetLabelSpec                          endpoint.Endpoint
	GetHost                               endpoint.Endpoint
	HostByIdentifier                      endpoint.Endpoint
	DeleteHost                            endpoint.Endpoint
	ListHosts                             endpoint.Endpoint
	GetHostSummary                        endpoint.Endpoint
	SearchTargets                         endpoint.Endpoint
	ApplyOsqueryOptionsSpec               endpoint.Endpoint
	GetOsqueryOptionsSpec                 endpoint.Endpoint
	GetCertificate                        endpoint.Endpoint
	ChangeEmail                           endpoint.Endpoint
	InitiateSSO                           endpoint.Endpoint
	CallbackSSO                           endpoint.Endpoint
	SSOSettings                           endpoint.Endpoint
	StatusResultStore                     endpoint.Endpoint
	StatusLiveQuery                       endpoint.Endpoint
	ListCarves                            endpoint.Endpoint
	GetCarve                              endpoint.Endpoint
	GetCarveBlock                         endpoint.Endpoint
	Version                               endpoint.Endpoint
}

// MakeKolideServerEndpoints creates the Kolide API endpoints.
func MakeKolideServerEndpoints(svc kolide.Service, jwtKey, urlPrefix string, limitStore throttled.GCRAStore) KolideEndpoints {
	limiter := ratelimit.NewMiddleware(limitStore)

	return KolideEndpoints{
		Login: limiter.Limit(throttled.RateQuota{MaxRate: throttled.PerMin(10), MaxBurst: 9})(
			makeLoginEndpoint(svc),
		),
		Logout: makeLogoutEndpoint(svc),
		ForgotPassword: limiter.Limit(
			throttled.RateQuota{MaxRate: throttled.PerHour(10), MaxBurst: 9})(
			makeForgotPasswordEndpoint(svc),
		),
		ResetPassword:        makeResetPasswordEndpoint(svc),
		CreateUserWithInvite: makeCreateUserWithInviteEndpoint(svc),
		VerifyInvite:         makeVerifyInviteEndpoint(svc),
		InitiateSSO:          makeInitiateSSOEndpoint(svc),
		CallbackSSO:          makeCallbackSSOEndpoint(svc, urlPrefix),
		SSOSettings:          makeSSOSettingsEndpoint(svc),

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
		ModifyUser:           authenticatedUser(jwtKey, svc, canModifyUser(makeModifyUserEndpoint(svc))),
		AdminUser:            authenticatedUser(jwtKey, svc, mustBeAdmin(makeAdminUserEndpoint(svc))),
		EnableUser:           authenticatedUser(jwtKey, svc, mustBeAdmin(makeEnableUserEndpoint(svc))),
		RequirePasswordReset: authenticatedUser(jwtKey, svc, mustBeAdmin(makeRequirePasswordResetEndpoint(svc))),
		CreateUser:           authenticatedUser(jwtKey, svc, mustBeAdmin(makeCreateUserEndpoint(svc))),
		// PerformRequiredPasswordReset needs only to authenticate the
		// logged in user
		PerformRequiredPasswordReset:          authenticatedUser(jwtKey, svc, canPerformPasswordReset(makePerformRequiredPasswordResetEndpoint(svc))),
		GetSessionsForUserInfo:                authenticatedUser(jwtKey, svc, canReadUser(makeGetInfoAboutSessionsForUserEndpoint(svc))),
		DeleteSessionsForUser:                 authenticatedUser(jwtKey, svc, canModifyUser(makeDeleteSessionsForUserEndpoint(svc))),
		GetSessionInfo:                        authenticatedUser(jwtKey, svc, mustBeAdmin(makeGetInfoAboutSessionEndpoint(svc))),
		DeleteSession:                         authenticatedUser(jwtKey, svc, mustBeAdmin(makeDeleteSessionEndpoint(svc))),
		GetAppConfig:                          authenticatedUser(jwtKey, svc, canPerformActions(makeGetAppConfigEndpoint(svc))),
		ModifyAppConfig:                       authenticatedUser(jwtKey, svc, mustBeAdmin(makeModifyAppConfigEndpoint(svc))),
		ApplyEnrollSecretSpec:                 authenticatedUser(jwtKey, svc, mustBeAdmin(makeApplyEnrollSecretSpecEndpoint(svc))),
		GetEnrollSecretSpec:                   authenticatedUser(jwtKey, svc, canPerformActions(makeGetEnrollSecretSpecEndpoint(svc))),
		CreateInvite:                          authenticatedUser(jwtKey, svc, mustBeAdmin(makeCreateInviteEndpoint(svc))),
		ListInvites:                           authenticatedUser(jwtKey, svc, mustBeAdmin(makeListInvitesEndpoint(svc))),
		DeleteInvite:                          authenticatedUser(jwtKey, svc, mustBeAdmin(makeDeleteInviteEndpoint(svc))),
		GetQuery:                              authenticatedUser(jwtKey, svc, makeGetQueryEndpoint(svc)),
		ListQueries:                           authenticatedUser(jwtKey, svc, makeListQueriesEndpoint(svc)),
		CreateQuery:                           authenticatedUser(jwtKey, svc, makeCreateQueryEndpoint(svc)),
		ModifyQuery:                           authenticatedUser(jwtKey, svc, makeModifyQueryEndpoint(svc)),
		DeleteQuery:                           authenticatedUser(jwtKey, svc, makeDeleteQueryEndpoint(svc)),
		DeleteQueryByID:                       authenticatedUser(jwtKey, svc, makeDeleteQueryByIDEndpoint(svc)),
		DeleteQueries:                         authenticatedUser(jwtKey, svc, makeDeleteQueriesEndpoint(svc)),
		ApplyQuerySpecs:                       authenticatedUser(jwtKey, svc, makeApplyQuerySpecsEndpoint(svc)),
		GetQuerySpecs:                         authenticatedUser(jwtKey, svc, makeGetQuerySpecsEndpoint(svc)),
		GetQuerySpec:                          authenticatedUser(jwtKey, svc, makeGetQuerySpecEndpoint(svc)),
		CreateDistributedQueryCampaign:        authenticatedUser(jwtKey, svc, makeCreateDistributedQueryCampaignEndpoint(svc)),
		CreateDistributedQueryCampaignByNames: authenticatedUser(jwtKey, svc, makeCreateDistributedQueryCampaignByNamesEndpoint(svc)),
		CreatePack:                            authenticatedUser(jwtKey, svc, makeCreatePackEndpoint(svc)),
		ModifyPack:                            authenticatedUser(jwtKey, svc, makeModifyPackEndpoint(svc)),
		GetPack:                               authenticatedUser(jwtKey, svc, makeGetPackEndpoint(svc)),
		ListPacks:                             authenticatedUser(jwtKey, svc, makeListPacksEndpoint(svc)),
		DeletePack:                            authenticatedUser(jwtKey, svc, makeDeletePackEndpoint(svc)),
		DeletePackByID:                        authenticatedUser(jwtKey, svc, makeDeletePackByIDEndpoint(svc)),
		GetScheduledQueriesInPack:             authenticatedUser(jwtKey, svc, makeGetScheduledQueriesInPackEndpoint(svc)),
		ScheduleQuery:                         authenticatedUser(jwtKey, svc, makeScheduleQueryEndpoint(svc)),
		GetScheduledQuery:                     authenticatedUser(jwtKey, svc, makeGetScheduledQueryEndpoint(svc)),
		ModifyScheduledQuery:                  authenticatedUser(jwtKey, svc, makeModifyScheduledQueryEndpoint(svc)),
		DeleteScheduledQuery:                  authenticatedUser(jwtKey, svc, makeDeleteScheduledQueryEndpoint(svc)),
		ApplyPackSpecs:                        authenticatedUser(jwtKey, svc, makeApplyPackSpecsEndpoint(svc)),
		GetPackSpecs:                          authenticatedUser(jwtKey, svc, makeGetPackSpecsEndpoint(svc)),
		GetPackSpec:                           authenticatedUser(jwtKey, svc, makeGetPackSpecEndpoint(svc)),
		GetHost:                               authenticatedUser(jwtKey, svc, makeGetHostEndpoint(svc)),
		HostByIdentifier:                      authenticatedUser(jwtKey, svc, makeHostByIdentifierEndpoint(svc)),
		ListHosts:                             authenticatedUser(jwtKey, svc, makeListHostsEndpoint(svc)),
		GetHostSummary:                        authenticatedUser(jwtKey, svc, makeGetHostSummaryEndpoint(svc)),
		DeleteHost:                            authenticatedUser(jwtKey, svc, makeDeleteHostEndpoint(svc)),
		CreateLabel:                           authenticatedUser(jwtKey, svc, makeCreateLabelEndpoint(svc)),
		ModifyLabel:                           authenticatedUser(jwtKey, svc, makeModifyLabelEndpoint(svc)),
		GetLabel:                              authenticatedUser(jwtKey, svc, makeGetLabelEndpoint(svc)),
		ListLabels:                            authenticatedUser(jwtKey, svc, makeListLabelsEndpoint(svc)),
		ListHostsInLabel:                      authenticatedUser(jwtKey, svc, makeListHostsInLabelEndpoint(svc)),
		DeleteLabel:                           authenticatedUser(jwtKey, svc, makeDeleteLabelEndpoint(svc)),
		DeleteLabelByID:                       authenticatedUser(jwtKey, svc, makeDeleteLabelByIDEndpoint(svc)),
		ApplyLabelSpecs:                       authenticatedUser(jwtKey, svc, makeApplyLabelSpecsEndpoint(svc)),
		GetLabelSpecs:                         authenticatedUser(jwtKey, svc, makeGetLabelSpecsEndpoint(svc)),
		GetLabelSpec:                          authenticatedUser(jwtKey, svc, makeGetLabelSpecEndpoint(svc)),
		SearchTargets:                         authenticatedUser(jwtKey, svc, makeSearchTargetsEndpoint(svc)),
		ApplyOsqueryOptionsSpec:               authenticatedUser(jwtKey, svc, makeApplyOsqueryOptionsSpecEndpoint(svc)),
		GetOsqueryOptionsSpec:                 authenticatedUser(jwtKey, svc, makeGetOsqueryOptionsSpecEndpoint(svc)),
		GetCertificate:                        authenticatedUser(jwtKey, svc, makeCertificateEndpoint(svc)),
		ChangeEmail:                           authenticatedUser(jwtKey, svc, makeChangeEmailEndpoint(svc)),
		ListCarves:                            authenticatedUser(jwtKey, svc, makeListCarvesEndpoint(svc)),
		GetCarve:                              authenticatedUser(jwtKey, svc, makeGetCarveEndpoint(svc)),
		GetCarveBlock:                         authenticatedUser(jwtKey, svc, makeGetCarveBlockEndpoint(svc)),
		Version:                               authenticatedUser(jwtKey, svc, makeVersionEndpoint(svc)),

		// Authenticated status endpoints
		StatusResultStore: authenticatedUser(jwtKey, svc, makeStatusResultStoreEndpoint(svc)),
		StatusLiveQuery:   authenticatedUser(jwtKey, svc, makeStatusLiveQueryEndpoint(svc)),

		// Osquery endpoints
		EnrollAgent:                   makeEnrollAgentEndpoint(svc),
		GetClientConfig:               authenticatedHost(svc, makeGetClientConfigEndpoint(svc)),
		GetDistributedQueries:         authenticatedHost(svc, makeGetDistributedQueriesEndpoint(svc)),
		SubmitDistributedQueryResults: authenticatedHost(svc, makeSubmitDistributedQueryResultsEndpoint(svc)),
		SubmitLogs:                    authenticatedHost(svc, makeSubmitLogsEndpoint(svc)),
		CarveBegin:                    authenticatedHost(svc, makeCarveBeginEndpoint(svc)),
		// For some reason osquery does not provide a node key with the block
		// data. Instead the carve session ID should be verified in the service
		// method.
		CarveBlock: makeCarveBlockEndpoint(svc),
	}
}

type kolideHandlers struct {
	Login                                 http.Handler
	Logout                                http.Handler
	ForgotPassword                        http.Handler
	ResetPassword                         http.Handler
	Me                                    http.Handler
	ChangePassword                        http.Handler
	CreateUserWithInvite                  http.Handler
	CreateUser                            http.Handler
	GetUser                               http.Handler
	ListUsers                             http.Handler
	ModifyUser                            http.Handler
	AdminUser                             http.Handler
	EnableUser                            http.Handler
	RequirePasswordReset                  http.Handler
	PerformRequiredPasswordReset          http.Handler
	GetSessionsForUserInfo                http.Handler
	DeleteSessionsForUser                 http.Handler
	GetSessionInfo                        http.Handler
	DeleteSession                         http.Handler
	GetAppConfig                          http.Handler
	ModifyAppConfig                       http.Handler
	ApplyEnrollSecretSpec                 http.Handler
	GetEnrollSecretSpec                   http.Handler
	CreateInvite                          http.Handler
	ListInvites                           http.Handler
	DeleteInvite                          http.Handler
	VerifyInvite                          http.Handler
	GetQuery                              http.Handler
	ListQueries                           http.Handler
	CreateQuery                           http.Handler
	ModifyQuery                           http.Handler
	DeleteQuery                           http.Handler
	DeleteQueryByID                       http.Handler
	DeleteQueries                         http.Handler
	ApplyQuerySpecs                       http.Handler
	GetQuerySpecs                         http.Handler
	GetQuerySpec                          http.Handler
	CreateDistributedQueryCampaign        http.Handler
	CreateDistributedQueryCampaignByNames http.Handler
	CreatePack                            http.Handler
	ModifyPack                            http.Handler
	GetPack                               http.Handler
	ListPacks                             http.Handler
	DeletePack                            http.Handler
	DeletePackByID                        http.Handler
	GetScheduledQueriesInPack             http.Handler
	ScheduleQuery                         http.Handler
	GetScheduledQuery                     http.Handler
	ModifyScheduledQuery                  http.Handler
	DeleteScheduledQuery                  http.Handler
	ApplyPackSpecs                        http.Handler
	GetPackSpecs                          http.Handler
	GetPackSpec                           http.Handler
	EnrollAgent                           http.Handler
	GetClientConfig                       http.Handler
	GetDistributedQueries                 http.Handler
	SubmitDistributedQueryResults         http.Handler
	SubmitLogs                            http.Handler
	CarveBegin                            http.Handler
	CarveBlock                            http.Handler
	CreateLabel                           http.Handler
	ModifyLabel                           http.Handler
	GetLabel                              http.Handler
	ListLabels                            http.Handler
	ListHostsInLabel                      http.Handler
	DeleteLabel                           http.Handler
	DeleteLabelByID                       http.Handler
	ApplyLabelSpecs                       http.Handler
	GetLabelSpecs                         http.Handler
	GetLabelSpec                          http.Handler
	GetHost                               http.Handler
	HostByIdentifier                      http.Handler
	DeleteHost                            http.Handler
	ListHosts                             http.Handler
	GetHostSummary                        http.Handler
	SearchTargets                         http.Handler
	ApplyOsqueryOptionsSpec               http.Handler
	GetOsqueryOptionsSpec                 http.Handler
	GetCertificate                        http.Handler
	ChangeEmail                           http.Handler
	InitiateSSO                           http.Handler
	CallbackSSO                           http.Handler
	SettingsSSO                           http.Handler
	StatusResultStore                     http.Handler
	StatusLiveQuery                       http.Handler
	ListCarves                            http.Handler
	GetCarve                              http.Handler
	GetCarveBlock                         http.Handler
	Version                               http.Handler
}

func makeKolideKitHandlers(e KolideEndpoints, opts []kithttp.ServerOption) *kolideHandlers {
	newServer := func(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc) http.Handler {
		return kithttp.NewServer(e, decodeFn, encodeResponse, opts...)
	}
	return &kolideHandlers{
		Login:                                 newServer(e.Login, decodeLoginRequest),
		Logout:                                newServer(e.Logout, decodeNoParamsRequest),
		ForgotPassword:                        newServer(e.ForgotPassword, decodeForgotPasswordRequest),
		ResetPassword:                         newServer(e.ResetPassword, decodeResetPasswordRequest),
		Me:                                    newServer(e.Me, decodeNoParamsRequest),
		ChangePassword:                        newServer(e.ChangePassword, decodeChangePasswordRequest),
		CreateUserWithInvite:                  newServer(e.CreateUserWithInvite, decodeCreateUserRequest),
		CreateUser:                            newServer(e.CreateUser, decodeCreateUserRequest),
		GetUser:                               newServer(e.GetUser, decodeGetUserRequest),
		ListUsers:                             newServer(e.ListUsers, decodeListUsersRequest),
		ModifyUser:                            newServer(e.ModifyUser, decodeModifyUserRequest),
		RequirePasswordReset:                  newServer(e.RequirePasswordReset, decodeRequirePasswordResetRequest),
		PerformRequiredPasswordReset:          newServer(e.PerformRequiredPasswordReset, decodePerformRequiredPasswordResetRequest),
		EnableUser:                            newServer(e.EnableUser, decodeEnableUserRequest),
		AdminUser:                             newServer(e.AdminUser, decodeAdminUserRequest),
		GetSessionsForUserInfo:                newServer(e.GetSessionsForUserInfo, decodeGetInfoAboutSessionsForUserRequest),
		DeleteSessionsForUser:                 newServer(e.DeleteSessionsForUser, decodeDeleteSessionsForUserRequest),
		GetSessionInfo:                        newServer(e.GetSessionInfo, decodeGetInfoAboutSessionRequest),
		DeleteSession:                         newServer(e.DeleteSession, decodeDeleteSessionRequest),
		GetAppConfig:                          newServer(e.GetAppConfig, decodeNoParamsRequest),
		ModifyAppConfig:                       newServer(e.ModifyAppConfig, decodeModifyAppConfigRequest),
		ApplyEnrollSecretSpec:                 newServer(e.ApplyEnrollSecretSpec, decodeApplyEnrollSecretSpecRequest),
		GetEnrollSecretSpec:                   newServer(e.GetEnrollSecretSpec, decodeNoParamsRequest),
		CreateInvite:                          newServer(e.CreateInvite, decodeCreateInviteRequest),
		ListInvites:                           newServer(e.ListInvites, decodeListInvitesRequest),
		DeleteInvite:                          newServer(e.DeleteInvite, decodeDeleteInviteRequest),
		VerifyInvite:                          newServer(e.VerifyInvite, decodeVerifyInviteRequest),
		GetQuery:                              newServer(e.GetQuery, decodeGetQueryRequest),
		ListQueries:                           newServer(e.ListQueries, decodeListQueriesRequest),
		CreateQuery:                           newServer(e.CreateQuery, decodeCreateQueryRequest),
		ModifyQuery:                           newServer(e.ModifyQuery, decodeModifyQueryRequest),
		DeleteQuery:                           newServer(e.DeleteQuery, decodeDeleteQueryRequest),
		DeleteQueryByID:                       newServer(e.DeleteQueryByID, decodeDeleteQueryByIDRequest),
		DeleteQueries:                         newServer(e.DeleteQueries, decodeDeleteQueriesRequest),
		ApplyQuerySpecs:                       newServer(e.ApplyQuerySpecs, decodeApplyQuerySpecsRequest),
		GetQuerySpecs:                         newServer(e.GetQuerySpecs, decodeNoParamsRequest),
		GetQuerySpec:                          newServer(e.GetQuerySpec, decodeGetGenericSpecRequest),
		CreateDistributedQueryCampaign:        newServer(e.CreateDistributedQueryCampaign, decodeCreateDistributedQueryCampaignRequest),
		CreateDistributedQueryCampaignByNames: newServer(e.CreateDistributedQueryCampaignByNames, decodeCreateDistributedQueryCampaignByNamesRequest),
		CreatePack:                            newServer(e.CreatePack, decodeCreatePackRequest),
		ModifyPack:                            newServer(e.ModifyPack, decodeModifyPackRequest),
		GetPack:                               newServer(e.GetPack, decodeGetPackRequest),
		ListPacks:                             newServer(e.ListPacks, decodeListPacksRequest),
		DeletePack:                            newServer(e.DeletePack, decodeDeletePackRequest),
		DeletePackByID:                        newServer(e.DeletePackByID, decodeDeletePackByIDRequest),
		GetScheduledQueriesInPack:             newServer(e.GetScheduledQueriesInPack, decodeGetScheduledQueriesInPackRequest),
		ScheduleQuery:                         newServer(e.ScheduleQuery, decodeScheduleQueryRequest),
		GetScheduledQuery:                     newServer(e.GetScheduledQuery, decodeGetScheduledQueryRequest),
		ModifyScheduledQuery:                  newServer(e.ModifyScheduledQuery, decodeModifyScheduledQueryRequest),
		DeleteScheduledQuery:                  newServer(e.DeleteScheduledQuery, decodeDeleteScheduledQueryRequest),
		ApplyPackSpecs:                        newServer(e.ApplyPackSpecs, decodeApplyPackSpecsRequest),
		GetPackSpecs:                          newServer(e.GetPackSpecs, decodeNoParamsRequest),
		GetPackSpec:                           newServer(e.GetPackSpec, decodeGetGenericSpecRequest),
		EnrollAgent:                           newServer(e.EnrollAgent, decodeEnrollAgentRequest),
		GetClientConfig:                       newServer(e.GetClientConfig, decodeGetClientConfigRequest),
		GetDistributedQueries:                 newServer(e.GetDistributedQueries, decodeGetDistributedQueriesRequest),
		SubmitDistributedQueryResults:         newServer(e.SubmitDistributedQueryResults, decodeSubmitDistributedQueryResultsRequest),
		SubmitLogs:                            newServer(e.SubmitLogs, decodeSubmitLogsRequest),
		CarveBegin:                            newServer(e.CarveBegin, decodeCarveBeginRequest),
		CarveBlock:                            newServer(e.CarveBlock, decodeCarveBlockRequest),
		CreateLabel:                           newServer(e.CreateLabel, decodeCreateLabelRequest),
		ModifyLabel:                           newServer(e.ModifyLabel, decodeModifyLabelRequest),
		GetLabel:                              newServer(e.GetLabel, decodeGetLabelRequest),
		ListLabels:                            newServer(e.ListLabels, decodeListLabelsRequest),
		ListHostsInLabel:                      newServer(e.ListHostsInLabel, decodeListHostsInLabelRequest),
		DeleteLabel:                           newServer(e.DeleteLabel, decodeDeleteLabelRequest),
		DeleteLabelByID:                       newServer(e.DeleteLabelByID, decodeDeleteLabelByIDRequest),
		ApplyLabelSpecs:                       newServer(e.ApplyLabelSpecs, decodeApplyLabelSpecsRequest),
		GetLabelSpecs:                         newServer(e.GetLabelSpecs, decodeNoParamsRequest),
		GetLabelSpec:                          newServer(e.GetLabelSpec, decodeGetGenericSpecRequest),
		GetHost:                               newServer(e.GetHost, decodeGetHostRequest),
		HostByIdentifier:                      newServer(e.HostByIdentifier, decodeHostByIdentifierRequest),
		DeleteHost:                            newServer(e.DeleteHost, decodeDeleteHostRequest),
		ListHosts:                             newServer(e.ListHosts, decodeListHostsRequest),
		GetHostSummary:                        newServer(e.GetHostSummary, decodeNoParamsRequest),
		SearchTargets:                         newServer(e.SearchTargets, decodeSearchTargetsRequest),
		ApplyOsqueryOptionsSpec:               newServer(e.ApplyOsqueryOptionsSpec, decodeApplyOsqueryOptionsSpecRequest),
		GetOsqueryOptionsSpec:                 newServer(e.GetOsqueryOptionsSpec, decodeNoParamsRequest),
		GetCertificate:                        newServer(e.GetCertificate, decodeNoParamsRequest),
		ChangeEmail:                           newServer(e.ChangeEmail, decodeChangeEmailRequest),
		InitiateSSO:                           newServer(e.InitiateSSO, decodeInitiateSSORequest),
		CallbackSSO:                           newServer(e.CallbackSSO, decodeCallbackSSORequest),
		SettingsSSO:                           newServer(e.SSOSettings, decodeNoParamsRequest),
		StatusResultStore:                     newServer(e.StatusResultStore, decodeNoParamsRequest),
		StatusLiveQuery:                       newServer(e.StatusLiveQuery, decodeNoParamsRequest),
		ListCarves:                            newServer(e.ListCarves, decodeListCarvesRequest),
		GetCarve:                              newServer(e.GetCarve, decodeGetCarveRequest),
		GetCarveBlock:                         newServer(e.GetCarveBlock, decodeGetCarveBlockRequest),
		Version:                               newServer(e.Version, decodeNoParamsRequest),
	}
}

type errorHandler struct {
	logger kitlog.Logger
}

func (h *errorHandler) Handle(ctx context.Context, err error) {
	// get the request path
	path, _ := ctx.Value(kithttp.ContextKeyRequestPath).(string)
	logger := level.Info(kitlog.With(h.logger, "path", path))

	switch e := err.(type) {
	case ratelimit.Error:
		res := e.Result()
		logger.Log("err", "limit exceeded", "retry_after", res.RetryAfter)

	default:
		logger.Log("err", err)
	}
}

// MakeHandler creates an HTTP handler for the Fleet server endpoints.
func MakeHandler(svc kolide.Service, config config.KolideConfig, logger kitlog.Logger, limitStore throttled.GCRAStore) http.Handler {
	kolideAPIOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(
			kithttp.PopulateRequestContext, // populate the request context with common fields
			setRequestsContexts(svc, config.Auth.JwtKey),
		),
		//kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorHandler(&errorHandler{logger}),
		kithttp.ServerErrorEncoder(encodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}

	kolideEndpoints := MakeKolideServerEndpoints(svc, config.Auth.JwtKey, config.Server.URLPrefix, limitStore)
	kolideHandlers := makeKolideKitHandlers(kolideEndpoints, kolideAPIOptions)

	r := mux.NewRouter()

	attachKolideAPIRoutes(r, kolideHandlers)

	// TODO #42 remove the shims
	shimRoutes(r, logger)

	// Results endpoint is handled different due to websockets use
	r.PathPrefix("/api/v1/fleet/results/").
		Handler(makeStreamDistributedQueryCampaignResultsHandler(svc, config.Auth.JwtKey, logger)).
		Name("distributed_query_results")
	r.PathPrefix("/api/v1/fleet/results/").
		Handler(makeStreamDistributedQueryCampaignResultsHandler(svc, config.Auth.JwtKey, logger)).
		Name("distributed_query_results")

	addMetrics(r)

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

func shimRoutes(r *mux.Router, logger kitlog.Logger) {
	if err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return errors.Wrap(err, "get path template")
		}

		if !strings.HasPrefix(path, "/api/v1/fleet") {
			return nil
		}

		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{}
		}

		path = strings.Replace(path, "fleet", "kolide", 1)
		router.Handle(path, route.GetHandler()).Methods(methods...).Name(route.GetName())

		return nil
	}); err != nil {
		panic(err)
	}

	// Shim the routes to allow them to be referred as /api/v1/fleet or
	// /api/v1/kolide.
	var lastShimLogTime time.Time
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if time.Now().After(lastShimLogTime.Add(1 * time.Hour)) {
				if strings.HasPrefix(r.URL.Path, "/api/v1/kolide") {
					// There's a race condition with this timestamp but it's not
					// a big deal if we accidentally log this message more than
					// once. The point is to not overwhelm the logs with these
					// messages and this code will be killed soon anyway.
					lastShimLogTime = time.Now()
					level.Info(logger).Log(
						"msg", "client used deprecated route",
						"deprecated", r.URL.Path,
					)
				}
			}
			next.ServeHTTP(w, r)
		})
	})
}

func attachKolideAPIRoutes(r *mux.Router, h *kolideHandlers) {
	r.Handle("/api/v1/fleet/login", h.Login).Methods("POST").Name("login")
	r.Handle("/api/v1/fleet/logout", h.Logout).Methods("POST").Name("logout")
	r.Handle("/api/v1/fleet/forgot_password", h.ForgotPassword).Methods("POST").Name("forgot_password")
	r.Handle("/api/v1/fleet/reset_password", h.ResetPassword).Methods("POST").Name("reset_password")
	r.Handle("/api/v1/fleet/me", h.Me).Methods("GET").Name("me")
	r.Handle("/api/v1/fleet/change_password", h.ChangePassword).Methods("POST").Name("change_password")
	r.Handle("/api/v1/fleet/perform_required_password_reset", h.PerformRequiredPasswordReset).Methods("POST").Name("perform_required_password_reset")
	r.Handle("/api/v1/fleet/sso", h.InitiateSSO).Methods("POST").Name("intiate_sso")
	r.Handle("/api/v1/fleet/sso", h.SettingsSSO).Methods("GET").Name("sso_config")
	r.Handle("/api/v1/fleet/sso/callback", h.CallbackSSO).Methods("POST").Name("callback_sso")
	r.Handle("/api/v1/fleet/users", h.ListUsers).Methods("GET").Name("list_users")
	r.Handle("/api/v1/fleet/users", h.CreateUserWithInvite).Methods("POST").Name("create_user_with_invite")
	r.Handle("/api/v1/fleet/users/admin", h.CreateUser).Methods("POST").Name("create_user")
	r.Handle("/api/v1/fleet/users/{id}", h.GetUser).Methods("GET").Name("get_user")
	r.Handle("/api/v1/fleet/users/{id}", h.ModifyUser).Methods("PATCH").Name("modify_user")
	r.Handle("/api/v1/fleet/users/{id}/enable", h.EnableUser).Methods("POST").Name("enable_user")
	r.Handle("/api/v1/fleet/users/{id}/admin", h.AdminUser).Methods("POST").Name("admin_user")
	r.Handle("/api/v1/fleet/users/{id}/require_password_reset", h.RequirePasswordReset).Methods("POST").Name("require_password_reset")
	r.Handle("/api/v1/fleet/users/{id}/sessions", h.GetSessionsForUserInfo).Methods("GET").Name("get_session_for_user")
	r.Handle("/api/v1/fleet/users/{id}/sessions", h.DeleteSessionsForUser).Methods("DELETE").Name("delete_session_for_user")

	r.Handle("/api/v1/fleet/sessions/{id}", h.GetSessionInfo).Methods("GET").Name("get_session_info")
	r.Handle("/api/v1/fleet/sessions/{id}", h.DeleteSession).Methods("DELETE").Name("delete_session")

	r.Handle("/api/v1/fleet/config/certificate", h.GetCertificate).Methods("GET").Name("get_certificate")
	r.Handle("/api/v1/fleet/config", h.GetAppConfig).Methods("GET").Name("get_app_config")
	r.Handle("/api/v1/fleet/config", h.ModifyAppConfig).Methods("PATCH").Name("modify_app_config")
	r.Handle("/api/v1/fleet/spec/enroll_secret", h.ApplyEnrollSecretSpec).Methods("POST").Name("apply_enroll_secret_spec")
	r.Handle("/api/v1/fleet/spec/enroll_secret", h.GetEnrollSecretSpec).Methods("GET").Name("get_enroll_secret_spec")
	r.Handle("/api/v1/fleet/invites", h.CreateInvite).Methods("POST").Name("create_invite")
	r.Handle("/api/v1/fleet/invites", h.ListInvites).Methods("GET").Name("list_invites")
	r.Handle("/api/v1/fleet/invites/{id}", h.DeleteInvite).Methods("DELETE").Name("delete_invite")
	r.Handle("/api/v1/fleet/invites/{token}", h.VerifyInvite).Methods("GET").Name("verify_invite")

	r.Handle("/api/v1/fleet/email/change/{token}", h.ChangeEmail).Methods("GET").Name("change_email")

	r.Handle("/api/v1/fleet/queries/{id}", h.GetQuery).Methods("GET").Name("get_query")
	r.Handle("/api/v1/fleet/queries", h.ListQueries).Methods("GET").Name("list_queries")
	r.Handle("/api/v1/fleet/queries", h.CreateQuery).Methods("POST").Name("create_query")
	r.Handle("/api/v1/fleet/queries/{id}", h.ModifyQuery).Methods("PATCH").Name("modify_query")
	r.Handle("/api/v1/fleet/queries/{name}", h.DeleteQuery).Methods("DELETE").Name("delete_query")
	r.Handle("/api/v1/fleet/queries/id/{id}", h.DeleteQueryByID).Methods("DELETE").Name("delete_query_by_id")
	r.Handle("/api/v1/fleet/queries/delete", h.DeleteQueries).Methods("POST").Name("delete_queries")
	r.Handle("/api/v1/fleet/spec/queries", h.ApplyQuerySpecs).Methods("POST").Name("apply_query_specs")
	r.Handle("/api/v1/fleet/spec/queries", h.GetQuerySpecs).Methods("GET").Name("get_query_specs")
	r.Handle("/api/v1/fleet/spec/queries/{name}", h.GetQuerySpec).Methods("GET").Name("get_query_spec")
	r.Handle("/api/v1/fleet/queries/run", h.CreateDistributedQueryCampaign).Methods("POST").Name("create_distributed_query_campaign")
	r.Handle("/api/v1/fleet/queries/run_by_names", h.CreateDistributedQueryCampaignByNames).Methods("POST").Name("create_distributed_query_campaign_by_names")

	r.Handle("/api/v1/fleet/packs", h.CreatePack).Methods("POST").Name("create_pack")
	r.Handle("/api/v1/fleet/packs/{id}", h.ModifyPack).Methods("PATCH").Name("modify_pack")
	r.Handle("/api/v1/fleet/packs/{id}", h.GetPack).Methods("GET").Name("get_pack")
	r.Handle("/api/v1/fleet/packs", h.ListPacks).Methods("GET").Name("list_packs")
	r.Handle("/api/v1/fleet/packs/{name}", h.DeletePack).Methods("DELETE").Name("delete_pack")
	r.Handle("/api/v1/fleet/packs/id/{id}", h.DeletePackByID).Methods("DELETE").Name("delete_pack_by_id")
	r.Handle("/api/v1/fleet/packs/{id}/scheduled", h.GetScheduledQueriesInPack).Methods("GET").Name("get_scheduled_queries_in_pack")
	r.Handle("/api/v1/fleet/schedule", h.ScheduleQuery).Methods("POST").Name("schedule_query")
	r.Handle("/api/v1/fleet/schedule/{id}", h.GetScheduledQuery).Methods("GET").Name("get_scheduled_query")
	r.Handle("/api/v1/fleet/schedule/{id}", h.ModifyScheduledQuery).Methods("PATCH").Name("modify_scheduled_query")
	r.Handle("/api/v1/fleet/schedule/{id}", h.DeleteScheduledQuery).Methods("DELETE").Name("delete_scheduled_query")
	r.Handle("/api/v1/fleet/spec/packs", h.ApplyPackSpecs).Methods("POST").Name("apply_pack_specs")
	r.Handle("/api/v1/fleet/spec/packs", h.GetPackSpecs).Methods("GET").Name("get_pack_specs")
	r.Handle("/api/v1/fleet/spec/packs/{name}", h.GetPackSpec).Methods("GET").Name("get_pack_spec")

	r.Handle("/api/v1/fleet/labels", h.CreateLabel).Methods("POST").Name("create_label")
	r.Handle("/api/v1/fleet/labels/{id}", h.ModifyLabel).Methods("PATCH").Name("modify_label")
	r.Handle("/api/v1/fleet/labels/{id}", h.GetLabel).Methods("GET").Name("get_label")
	r.Handle("/api/v1/fleet/labels", h.ListLabels).Methods("GET").Name("list_labels")
	r.Handle("/api/v1/fleet/labels/{id}/hosts", h.ListHostsInLabel).Methods("GET").Name("list_hosts_in_label")
	r.Handle("/api/v1/fleet/labels/{name}", h.DeleteLabel).Methods("DELETE").Name("delete_label")
	r.Handle("/api/v1/fleet/labels/id/{id}", h.DeleteLabelByID).Methods("DELETE").Name("delete_label_by_id")
	r.Handle("/api/v1/fleet/spec/labels", h.ApplyLabelSpecs).Methods("POST").Name("apply_label_specs")
	r.Handle("/api/v1/fleet/spec/labels", h.GetLabelSpecs).Methods("GET").Name("get_label_specs")
	r.Handle("/api/v1/fleet/spec/labels/{name}", h.GetLabelSpec).Methods("GET").Name("get_label_spec")

	r.Handle("/api/v1/fleet/hosts", h.ListHosts).Methods("GET").Name("list_hosts")
	r.Handle("/api/v1/fleet/host_summary", h.GetHostSummary).Methods("GET").Name("get_host_summary")
	r.Handle("/api/v1/fleet/hosts/{id}", h.GetHost).Methods("GET").Name("get_host")
	r.Handle("/api/v1/fleet/hosts/identifier/{identifier}", h.HostByIdentifier).Methods("GET").Name("host_by_identifier")
	r.Handle("/api/v1/fleet/hosts/{id}", h.DeleteHost).Methods("DELETE").Name("delete_host")

	r.Handle("/api/v1/fleet/spec/osquery_options", h.ApplyOsqueryOptionsSpec).Methods("POST").Name("apply_osquery_options_spec")
	r.Handle("/api/v1/fleet/spec/osquery_options", h.GetOsqueryOptionsSpec).Methods("GET").Name("get_osquery_options_spec")

	r.Handle("/api/v1/fleet/targets", h.SearchTargets).Methods("POST").Name("search_targets")

	r.Handle("/api/v1/fleet/version", h.Version).Methods("GET").Name("version")

	r.Handle("/api/v1/fleet/status/result_store", h.StatusResultStore).Methods("GET").Name("status_result_store")
	r.Handle("/api/v1/fleet/status/live_query", h.StatusLiveQuery).Methods("GET").Name("status_live_query")

	r.Handle("/api/v1/fleet/carves", h.ListCarves).Methods("GET").Name("list_carves")
	r.Handle("/api/v1/fleet/carves/{id}", h.GetCarve).Methods("GET").Name("get_carve")
	r.Handle("/api/v1/fleet/carves/{id}/block/{block_id}", h.GetCarveBlock).Methods("GET").Name("get_carve_block")

	r.Handle("/api/v1/osquery/enroll", h.EnrollAgent).Methods("POST").Name("enroll_agent")
	r.Handle("/api/v1/osquery/config", h.GetClientConfig).Methods("POST").Name("get_client_config")
	r.Handle("/api/v1/osquery/distributed/read", h.GetDistributedQueries).Methods("POST").Name("get_distributed_queries")
	r.Handle("/api/v1/osquery/distributed/write", h.SubmitDistributedQueryResults).Methods("POST").Name("submit_distributed_query_results")
	r.Handle("/api/v1/osquery/log", h.SubmitLogs).Methods("POST").Name("submit_logs")
	r.Handle("/api/v1/osquery/carve/begin", h.CarveBegin).Methods("POST").Name("carve_begin")
	r.Handle("/api/v1/osquery/carve/block", h.CarveBlock).Methods("POST").Name("carve_block")
}

// WithSetup is an http middleware that checks if setup procedures have been completed.
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
func RedirectLoginToSetup(svc kolide.Service, logger kitlog.Logger, next http.Handler, urlPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		redirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/setup" {
				next.ServeHTTP(w, r)
				return
			}
			newURL := r.URL
			newURL.Path = urlPrefix + "/setup"
			http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
		})

		setupRequired, err := RequireSetup(svc)
		if err != nil {
			logger.Log("msg", "fetching setupinfo from db", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if setupRequired {
			redirect.ServeHTTP(w, r)
			return
		}
		RedirectSetupToLogin(svc, logger, next, urlPrefix).ServeHTTP(w, r)
	}
}

// RequireSetup checks to see if the service has been setup.
func RequireSetup(svc kolide.Service) (bool, error) {
	ctx := context.Background()
	users, err := svc.ListUsers(ctx, kolide.ListOptions{Page: 0, PerPage: 1})
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		return true, nil
	}
	return false, nil
}

// RedirectSetupToLogin forces the /setup path to be redirected to login. This middleware is used after
// the app has been setup.
func RedirectSetupToLogin(svc kolide.Service, logger kitlog.Logger, next http.Handler, urlPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/setup" {
			newURL := r.URL
			newURL.Path = urlPrefix + "/login"
			http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	}
}
