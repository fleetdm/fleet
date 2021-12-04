package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/authzcheck"
	"github.com/fleetdm/fleet/v4/server/service/middleware/ratelimit"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/throttled/throttled/v2"
)

// FleetEndpoints is a collection of RPC endpoints implemented by the Fleet API.
type FleetEndpoints struct {
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
	DeleteUser                            endpoint.Endpoint
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
	GlobalScheduleQuery                   endpoint.Endpoint
	GetGlobalSchedule                     endpoint.Endpoint
	ModifyGlobalSchedule                  endpoint.Endpoint
	DeleteGlobalSchedule                  endpoint.Endpoint
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
	HostByIdentifier                      endpoint.Endpoint
	DeleteHost                            endpoint.Endpoint
	RefetchHost                           endpoint.Endpoint
	AddHostsToTeam                        endpoint.Endpoint
	AddHostsToTeamByFilter                endpoint.Endpoint
	SearchTargets                         endpoint.Endpoint
	GetCertificate                        endpoint.Endpoint
	ChangeEmail                           endpoint.Endpoint
	InitiateSSO                           endpoint.Endpoint
	CallbackSSO                           endpoint.Endpoint
	SSOSettings                           endpoint.Endpoint
	StatusResultStore                     endpoint.Endpoint
	StatusLiveQuery                       endpoint.Endpoint
	Version                               endpoint.Endpoint
	CreateTeam                            endpoint.Endpoint
	ModifyTeam                            endpoint.Endpoint
	ModifyTeamAgentOptions                endpoint.Endpoint
	DeleteTeam                            endpoint.Endpoint
	ListTeams                             endpoint.Endpoint
	ListTeamUsers                         endpoint.Endpoint
	AddTeamUsers                          endpoint.Endpoint
	DeleteTeamUsers                       endpoint.Endpoint
	TeamEnrollSecrets                     endpoint.Endpoint
}

// MakeFleetServerEndpoints creates the Fleet API endpoints.
func MakeFleetServerEndpoints(svc fleet.Service, urlPrefix string, limitStore throttled.GCRAStore, logger kitlog.Logger) FleetEndpoints {
	limiter := ratelimit.NewMiddleware(limitStore)

	return FleetEndpoints{
		Login: limiter.Limit(
			throttled.RateQuota{MaxRate: throttled.PerMin(10), MaxBurst: 9})(
			makeLoginEndpoint(svc),
		),
		Logout: logged(makeLogoutEndpoint(svc)),
		ForgotPassword: limiter.Limit(
			throttled.RateQuota{MaxRate: throttled.PerHour(10), MaxBurst: 9})(
			logged(makeForgotPasswordEndpoint(svc)),
		),
		ResetPassword:        logged(makeResetPasswordEndpoint(svc)),
		CreateUserWithInvite: logged(makeCreateUserFromInviteEndpoint(svc)),
		VerifyInvite:         logged(makeVerifyInviteEndpoint(svc)),
		InitiateSSO:          logged(makeInitiateSSOEndpoint(svc)),
		CallbackSSO:          logged(makeCallbackSSOEndpoint(svc, urlPrefix)),
		SSOSettings:          logged(makeSSOSettingsEndpoint(svc)),

		// PerformRequiredPasswordReset needs only to authenticate the
		// logged in user
		PerformRequiredPasswordReset: logged(canPerformPasswordReset(makePerformRequiredPasswordResetEndpoint(svc))),

		// Standard user authentication routes
		Me:                                    authenticatedUser(svc, makeGetSessionUserEndpoint(svc)),
		ChangePassword:                        authenticatedUser(svc, makeChangePasswordEndpoint(svc)),
		GetUser:                               authenticatedUser(svc, makeGetUserEndpoint(svc)),
		ListUsers:                             authenticatedUser(svc, makeListUsersEndpoint(svc)),
		ModifyUser:                            authenticatedUser(svc, makeModifyUserEndpoint(svc)),
		DeleteUser:                            authenticatedUser(svc, makeDeleteUserEndpoint(svc)),
		RequirePasswordReset:                  authenticatedUser(svc, makeRequirePasswordResetEndpoint(svc)),
		CreateUser:                            authenticatedUser(svc, makeCreateUserEndpoint(svc)),
		GetSessionsForUserInfo:                authenticatedUser(svc, makeGetInfoAboutSessionsForUserEndpoint(svc)),
		DeleteSessionsForUser:                 authenticatedUser(svc, makeDeleteSessionsForUserEndpoint(svc)),
		GetSessionInfo:                        authenticatedUser(svc, makeGetInfoAboutSessionEndpoint(svc)),
		DeleteSession:                         authenticatedUser(svc, makeDeleteSessionEndpoint(svc)),
		GetAppConfig:                          authenticatedUser(svc, makeGetAppConfigEndpoint(svc)),
		ModifyAppConfig:                       authenticatedUser(svc, makeModifyAppConfigEndpoint(svc)),
		ApplyEnrollSecretSpec:                 authenticatedUser(svc, makeApplyEnrollSecretSpecEndpoint(svc)),
		GetEnrollSecretSpec:                   authenticatedUser(svc, makeGetEnrollSecretSpecEndpoint(svc)),
		CreateInvite:                          authenticatedUser(svc, makeCreateInviteEndpoint(svc)),
		ListInvites:                           authenticatedUser(svc, makeListInvitesEndpoint(svc)),
		DeleteInvite:                          authenticatedUser(svc, makeDeleteInviteEndpoint(svc)),
		GetQuery:                              authenticatedUser(svc, makeGetQueryEndpoint(svc)),
		ListQueries:                           authenticatedUser(svc, makeListQueriesEndpoint(svc)),
		CreateQuery:                           authenticatedUser(svc, makeCreateQueryEndpoint(svc)),
		ModifyQuery:                           authenticatedUser(svc, makeModifyQueryEndpoint(svc)),
		DeleteQuery:                           authenticatedUser(svc, makeDeleteQueryEndpoint(svc)),
		DeleteQueryByID:                       authenticatedUser(svc, makeDeleteQueryByIDEndpoint(svc)),
		DeleteQueries:                         authenticatedUser(svc, makeDeleteQueriesEndpoint(svc)),
		ApplyQuerySpecs:                       authenticatedUser(svc, makeApplyQuerySpecsEndpoint(svc)),
		GetQuerySpecs:                         authenticatedUser(svc, makeGetQuerySpecsEndpoint(svc)),
		GetQuerySpec:                          authenticatedUser(svc, makeGetQuerySpecEndpoint(svc)),
		CreateDistributedQueryCampaign:        authenticatedUser(svc, makeCreateDistributedQueryCampaignEndpoint(svc)),
		CreateDistributedQueryCampaignByNames: authenticatedUser(svc, makeCreateDistributedQueryCampaignByNamesEndpoint(svc)),
		CreatePack:                            authenticatedUser(svc, makeCreatePackEndpoint(svc)),
		ModifyPack:                            authenticatedUser(svc, makeModifyPackEndpoint(svc)),
		ListPacks:                             authenticatedUser(svc, makeListPacksEndpoint(svc)),
		DeletePack:                            authenticatedUser(svc, makeDeletePackEndpoint(svc)),
		DeletePackByID:                        authenticatedUser(svc, makeDeletePackByIDEndpoint(svc)),
		GetScheduledQueriesInPack:             authenticatedUser(svc, makeGetScheduledQueriesInPackEndpoint(svc)),
		ScheduleQuery:                         authenticatedUser(svc, makeScheduleQueryEndpoint(svc)),
		GetScheduledQuery:                     authenticatedUser(svc, makeGetScheduledQueryEndpoint(svc)),
		ModifyScheduledQuery:                  authenticatedUser(svc, makeModifyScheduledQueryEndpoint(svc)),
		DeleteScheduledQuery:                  authenticatedUser(svc, makeDeleteScheduledQueryEndpoint(svc)),
		ApplyPackSpecs:                        authenticatedUser(svc, makeApplyPackSpecsEndpoint(svc)),
		GetPackSpecs:                          authenticatedUser(svc, makeGetPackSpecsEndpoint(svc)),
		GetPackSpec:                           authenticatedUser(svc, makeGetPackSpecEndpoint(svc)),
		GlobalScheduleQuery:                   authenticatedUser(svc, makeGlobalScheduleQueryEndpoint(svc)),
		GetGlobalSchedule:                     authenticatedUser(svc, makeGetGlobalScheduleEndpoint(svc)),
		ModifyGlobalSchedule:                  authenticatedUser(svc, makeModifyGlobalScheduleEndpoint(svc)),
		DeleteGlobalSchedule:                  authenticatedUser(svc, makeDeleteGlobalScheduleEndpoint(svc)),
		HostByIdentifier:                      authenticatedUser(svc, makeHostByIdentifierEndpoint(svc)),
		DeleteHost:                            authenticatedUser(svc, makeDeleteHostEndpoint(svc)),
		AddHostsToTeam:                        authenticatedUser(svc, makeAddHostsToTeamEndpoint(svc)),
		AddHostsToTeamByFilter:                authenticatedUser(svc, makeAddHostsToTeamByFilterEndpoint(svc)),
		RefetchHost:                           authenticatedUser(svc, makeRefetchHostEndpoint(svc)),
		CreateLabel:                           authenticatedUser(svc, makeCreateLabelEndpoint(svc)),
		ModifyLabel:                           authenticatedUser(svc, makeModifyLabelEndpoint(svc)),
		GetLabel:                              authenticatedUser(svc, makeGetLabelEndpoint(svc)),
		ListLabels:                            authenticatedUser(svc, makeListLabelsEndpoint(svc)),
		ListHostsInLabel:                      authenticatedUser(svc, makeListHostsInLabelEndpoint(svc)),
		DeleteLabel:                           authenticatedUser(svc, makeDeleteLabelEndpoint(svc)),
		DeleteLabelByID:                       authenticatedUser(svc, makeDeleteLabelByIDEndpoint(svc)),
		ApplyLabelSpecs:                       authenticatedUser(svc, makeApplyLabelSpecsEndpoint(svc)),
		GetLabelSpecs:                         authenticatedUser(svc, makeGetLabelSpecsEndpoint(svc)),
		GetLabelSpec:                          authenticatedUser(svc, makeGetLabelSpecEndpoint(svc)),
		SearchTargets:                         authenticatedUser(svc, makeSearchTargetsEndpoint(svc)),
		GetCertificate:                        authenticatedUser(svc, makeCertificateEndpoint(svc)),
		ChangeEmail:                           authenticatedUser(svc, makeChangeEmailEndpoint(svc)),
		Version:                               authenticatedUser(svc, makeVersionEndpoint(svc)),
		CreateTeam:                            authenticatedUser(svc, makeCreateTeamEndpoint(svc)),
		ModifyTeam:                            authenticatedUser(svc, makeModifyTeamEndpoint(svc)),
		ModifyTeamAgentOptions:                authenticatedUser(svc, makeModifyTeamAgentOptionsEndpoint(svc)),
		DeleteTeam:                            authenticatedUser(svc, makeDeleteTeamEndpoint(svc)),
		ListTeams:                             authenticatedUser(svc, makeListTeamsEndpoint(svc)),
		ListTeamUsers:                         authenticatedUser(svc, makeListTeamUsersEndpoint(svc)),
		AddTeamUsers:                          authenticatedUser(svc, makeAddTeamUsersEndpoint(svc)),
		DeleteTeamUsers:                       authenticatedUser(svc, makeDeleteTeamUsersEndpoint(svc)),
		TeamEnrollSecrets:                     authenticatedUser(svc, makeTeamEnrollSecretsEndpoint(svc)),

		// Authenticated status endpoints
		StatusResultStore: authenticatedUser(svc, makeStatusResultStoreEndpoint(svc)),
		StatusLiveQuery:   authenticatedUser(svc, makeStatusLiveQueryEndpoint(svc)),

		// Osquery endpoints
		EnrollAgent: logged(makeEnrollAgentEndpoint(svc)),
		// Authenticated osquery endpoints
		GetClientConfig:               authenticatedHost(svc, logger, makeGetClientConfigEndpoint(svc)),
		GetDistributedQueries:         authenticatedHost(svc, logger, makeGetDistributedQueriesEndpoint(svc)),
		SubmitDistributedQueryResults: authenticatedHost(svc, logger, makeSubmitDistributedQueryResultsEndpoint(svc)),
		SubmitLogs:                    authenticatedHost(svc, logger, makeSubmitLogsEndpoint(svc)),
		CarveBegin:                    authenticatedHost(svc, logger, makeCarveBeginEndpoint(svc)),
		// For some reason osquery does not provide a node key with the block
		// data. Instead the carve session ID should be verified in the service
		// method.
		CarveBlock: logged(makeCarveBlockEndpoint(svc)),
	}
}

type fleetHandlers struct {
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
	DeleteUser                            http.Handler
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
	GlobalScheduleQuery                   http.Handler
	GetGlobalSchedule                     http.Handler
	ModifyGlobalSchedule                  http.Handler
	DeleteGlobalSchedule                  http.Handler
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
	HostByIdentifier                      http.Handler
	DeleteHost                            http.Handler
	RefetchHost                           http.Handler
	AddHostsToTeam                        http.Handler
	AddHostsToTeamByFilter                http.Handler
	SearchTargets                         http.Handler
	GetCertificate                        http.Handler
	ChangeEmail                           http.Handler
	InitiateSSO                           http.Handler
	CallbackSSO                           http.Handler
	SettingsSSO                           http.Handler
	StatusResultStore                     http.Handler
	StatusLiveQuery                       http.Handler
	Version                               http.Handler
	CreateTeam                            http.Handler
	ModifyTeam                            http.Handler
	ModifyTeamAgentOptions                http.Handler
	DeleteTeam                            http.Handler
	ListTeams                             http.Handler
	ListTeamUsers                         http.Handler
	AddTeamUsers                          http.Handler
	DeleteTeamUsers                       http.Handler
	TeamEnrollSecrets                     http.Handler
}

func makeKitHandlers(e FleetEndpoints, opts []kithttp.ServerOption) *fleetHandlers {
	newServer := func(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc) http.Handler {
		e = authzcheck.NewMiddleware().AuthzCheck()(e)
		return kithttp.NewServer(e, decodeFn, encodeResponse, opts...)
	}
	return &fleetHandlers{
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
		DeleteUser:                            newServer(e.DeleteUser, decodeDeleteUserRequest),
		RequirePasswordReset:                  newServer(e.RequirePasswordReset, decodeRequirePasswordResetRequest),
		PerformRequiredPasswordReset:          newServer(e.PerformRequiredPasswordReset, decodePerformRequiredPasswordResetRequest),
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
		GlobalScheduleQuery:                   newServer(e.GlobalScheduleQuery, decodeGlobalScheduleQueryRequest),
		GetGlobalSchedule:                     newServer(e.GetGlobalSchedule, decodeGetGlobalScheduleRequest),
		ModifyGlobalSchedule:                  newServer(e.ModifyGlobalSchedule, decodeModifyGlobalScheduleRequest),
		DeleteGlobalSchedule:                  newServer(e.DeleteGlobalSchedule, decodeDeleteGlobalScheduleRequest),
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
		HostByIdentifier:                      newServer(e.HostByIdentifier, decodeHostByIdentifierRequest),
		DeleteHost:                            newServer(e.DeleteHost, decodeDeleteHostRequest),
		RefetchHost:                           newServer(e.RefetchHost, decodeRefetchHostRequest),
		AddHostsToTeam:                        newServer(e.AddHostsToTeam, decodeAddHostsToTeamRequest),
		AddHostsToTeamByFilter:                newServer(e.AddHostsToTeamByFilter, decodeAddHostsToTeamByFilterRequest),
		SearchTargets:                         newServer(e.SearchTargets, decodeSearchTargetsRequest),
		GetCertificate:                        newServer(e.GetCertificate, decodeNoParamsRequest),
		ChangeEmail:                           newServer(e.ChangeEmail, decodeChangeEmailRequest),
		InitiateSSO:                           newServer(e.InitiateSSO, decodeInitiateSSORequest),
		CallbackSSO:                           newServer(e.CallbackSSO, decodeCallbackSSORequest),
		SettingsSSO:                           newServer(e.SSOSettings, decodeNoParamsRequest),
		StatusResultStore:                     newServer(e.StatusResultStore, decodeNoParamsRequest),
		StatusLiveQuery:                       newServer(e.StatusLiveQuery, decodeNoParamsRequest),
		Version:                               newServer(e.Version, decodeNoParamsRequest),
		CreateTeam:                            newServer(e.CreateTeam, decodeCreateTeamRequest),
		ModifyTeam:                            newServer(e.ModifyTeam, decodeModifyTeamRequest),
		ModifyTeamAgentOptions:                newServer(e.ModifyTeamAgentOptions, decodeModifyTeamAgentOptionsRequest),
		DeleteTeam:                            newServer(e.DeleteTeam, decodeDeleteTeamRequest),
		ListTeams:                             newServer(e.ListTeams, decodeListTeamsRequest),
		ListTeamUsers:                         newServer(e.ListTeamUsers, decodeListTeamUsersRequest),
		AddTeamUsers:                          newServer(e.AddTeamUsers, decodeModifyTeamUsersRequest),
		DeleteTeamUsers:                       newServer(e.DeleteTeamUsers, decodeModifyTeamUsersRequest),
		TeamEnrollSecrets:                     newServer(e.TeamEnrollSecrets, decodeTeamEnrollSecretsRequest),
	}
}

type errorHandler struct {
	logger kitlog.Logger
}

func (h *errorHandler) Handle(ctx context.Context, err error) {
	// get the request path
	path, _ := ctx.Value(kithttp.ContextKeyRequestPath).(string)
	logger := level.Info(kitlog.With(h.logger, "path", path))

	var ewi fleet.ErrWithInternal
	if errors.As(err, &ewi) {
		logger = kitlog.With(logger, "internal", ewi.Internal())
	}

	var ewlf fleet.ErrWithLogFields
	if errors.As(err, &ewlf) {
		logger = kitlog.With(logger, ewlf.LogFields()...)
	}

	var rle ratelimit.Error
	if errors.As(err, &rle) {
		res := rle.Result()
		logger.Log("err", "limit exceeded", "retry_after", res.RetryAfter)
	} else {
		logger.Log("err", err)
	}
}

func logRequestEnd(logger kitlog.Logger) func(context.Context, http.ResponseWriter) context.Context {
	return func(ctx context.Context, w http.ResponseWriter) context.Context {
		logCtx, ok := logging.FromContext(ctx)
		if !ok {
			return ctx
		}
		logCtx.Log(ctx, logger)
		return ctx
	}
}

func checkLicenseExpiration(svc fleet.Service) func(context.Context, http.ResponseWriter) context.Context {
	return func(ctx context.Context, w http.ResponseWriter) context.Context {
		license, err := svc.License(ctx)
		if err != nil || license == nil {
			return ctx
		}
		if license.IsPremium() && license.IsExpired() {
			w.Header().Set(fleet.HeaderLicenseKey, fleet.HeaderLicenseValueExpired)
		}
		return ctx
	}
}

// MakeHandler creates an HTTP handler for the Fleet server endpoints.
func MakeHandler(svc fleet.Service, config config.FleetConfig, logger kitlog.Logger, limitStore throttled.GCRAStore) http.Handler {
	fleetAPIOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(
			kithttp.PopulateRequestContext, // populate the request context with common fields
			setRequestsContexts(svc),
		),
		kithttp.ServerErrorHandler(&errorHandler{logger}),
		kithttp.ServerErrorEncoder(encodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
			logRequestEnd(logger),
			checkLicenseExpiration(svc),
		),
	}

	fleetEndpoints := MakeFleetServerEndpoints(svc, config.Server.URLPrefix, limitStore, logger)
	fleetHandlers := makeKitHandlers(fleetEndpoints, fleetAPIOptions)

	r := mux.NewRouter()

	attachFleetAPIRoutes(r, fleetHandlers)
	attachNewStyleFleetAPIRoutes(r, svc, fleetAPIOptions)

	// Results endpoint is handled different due to websockets use
	r.PathPrefix("/api/v1/fleet/results/").
		Handler(makeStreamDistributedQueryCampaignResultsHandler(svc, logger)).
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

func attachFleetAPIRoutes(r *mux.Router, h *fleetHandlers) {
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
	r.Handle("/api/v1/fleet/users/{id:[0-9]+}", h.GetUser).Methods("GET").Name("get_user")
	r.Handle("/api/v1/fleet/users/{id:[0-9]+}", h.ModifyUser).Methods("PATCH").Name("modify_user")
	r.Handle("/api/v1/fleet/users/{id:[0-9]+}", h.DeleteUser).Methods("DELETE").Name("delete_user")
	r.Handle("/api/v1/fleet/users/{id:[0-9]+}/require_password_reset", h.RequirePasswordReset).Methods("POST").Name("require_password_reset")
	r.Handle("/api/v1/fleet/users/{id:[0-9]+}/sessions", h.GetSessionsForUserInfo).Methods("GET").Name("get_session_for_user")
	r.Handle("/api/v1/fleet/users/{id:[0-9]+}/sessions", h.DeleteSessionsForUser).Methods("DELETE").Name("delete_session_for_user")

	r.Handle("/api/v1/fleet/sessions/{id:[0-9]+}", h.GetSessionInfo).Methods("GET").Name("get_session_info")
	r.Handle("/api/v1/fleet/sessions/{id:[0-9]+}", h.DeleteSession).Methods("DELETE").Name("delete_session")

	r.Handle("/api/v1/fleet/config/certificate", h.GetCertificate).Methods("GET").Name("get_certificate")
	r.Handle("/api/v1/fleet/config", h.GetAppConfig).Methods("GET").Name("get_app_config")
	r.Handle("/api/v1/fleet/config", h.ModifyAppConfig).Methods("PATCH").Name("modify_app_config")
	r.Handle("/api/v1/fleet/spec/enroll_secret", h.ApplyEnrollSecretSpec).Methods("POST").Name("apply_enroll_secret_spec")
	r.Handle("/api/v1/fleet/spec/enroll_secret", h.GetEnrollSecretSpec).Methods("GET").Name("get_enroll_secret_spec")
	r.Handle("/api/v1/fleet/invites", h.CreateInvite).Methods("POST").Name("create_invite")
	r.Handle("/api/v1/fleet/invites", h.ListInvites).Methods("GET").Name("list_invites")
	r.Handle("/api/v1/fleet/invites/{id:[0-9]+}", h.DeleteInvite).Methods("DELETE").Name("delete_invite")
	r.Handle("/api/v1/fleet/invites/{token}", h.VerifyInvite).Methods("GET").Name("verify_invite")

	r.Handle("/api/v1/fleet/email/change/{token}", h.ChangeEmail).Methods("GET").Name("change_email")

	r.Handle("/api/v1/fleet/queries/{id:[0-9]+}", h.GetQuery).Methods("GET").Name("get_query")
	r.Handle("/api/v1/fleet/queries", h.ListQueries).Methods("GET").Name("list_queries")
	r.Handle("/api/v1/fleet/queries", h.CreateQuery).Methods("POST").Name("create_query")
	r.Handle("/api/v1/fleet/queries/{id:[0-9]+}", h.ModifyQuery).Methods("PATCH").Name("modify_query")
	r.Handle("/api/v1/fleet/queries/{name}", h.DeleteQuery).Methods("DELETE").Name("delete_query")
	r.Handle("/api/v1/fleet/queries/id/{id:[0-9]+}", h.DeleteQueryByID).Methods("DELETE").Name("delete_query_by_id")
	r.Handle("/api/v1/fleet/queries/delete", h.DeleteQueries).Methods("POST").Name("delete_queries")
	r.Handle("/api/v1/fleet/spec/queries", h.ApplyQuerySpecs).Methods("POST").Name("apply_query_specs")
	r.Handle("/api/v1/fleet/spec/queries", h.GetQuerySpecs).Methods("GET").Name("get_query_specs")
	r.Handle("/api/v1/fleet/spec/queries/{name}", h.GetQuerySpec).Methods("GET").Name("get_query_spec")
	r.Handle("/api/v1/fleet/queries/run", h.CreateDistributedQueryCampaign).Methods("POST").Name("create_distributed_query_campaign")
	r.Handle("/api/v1/fleet/queries/run_by_names", h.CreateDistributedQueryCampaignByNames).Methods("POST").Name("create_distributed_query_campaign_by_names")

	r.Handle("/api/v1/fleet/packs", h.CreatePack).Methods("POST").Name("create_pack")
	r.Handle("/api/v1/fleet/packs/{id:[0-9]+}", h.ModifyPack).Methods("PATCH").Name("modify_pack")
	r.Handle("/api/v1/fleet/packs", h.ListPacks).Methods("GET").Name("list_packs")
	r.Handle("/api/v1/fleet/packs/{name}", h.DeletePack).Methods("DELETE").Name("delete_pack")
	r.Handle("/api/v1/fleet/packs/id/{id:[0-9]+}", h.DeletePackByID).Methods("DELETE").Name("delete_pack_by_id")
	r.Handle("/api/v1/fleet/packs/{id:[0-9]+}/scheduled", h.GetScheduledQueriesInPack).Methods("GET").Name("get_scheduled_queries_in_pack")
	r.Handle("/api/v1/fleet/schedule", h.ScheduleQuery).Methods("POST").Name("schedule_query")
	r.Handle("/api/v1/fleet/schedule/{id:[0-9]+}", h.GetScheduledQuery).Methods("GET").Name("get_scheduled_query")
	r.Handle("/api/v1/fleet/schedule/{id:[0-9]+}", h.ModifyScheduledQuery).Methods("PATCH").Name("modify_scheduled_query")
	r.Handle("/api/v1/fleet/schedule/{id:[0-9]+}", h.DeleteScheduledQuery).Methods("DELETE").Name("delete_scheduled_query")
	r.Handle("/api/v1/fleet/spec/packs", h.ApplyPackSpecs).Methods("POST").Name("apply_pack_specs")
	r.Handle("/api/v1/fleet/spec/packs", h.GetPackSpecs).Methods("GET").Name("get_pack_specs")
	r.Handle("/api/v1/fleet/spec/packs/{name}", h.GetPackSpec).Methods("GET").Name("get_pack_spec")

	r.Handle("/api/v1/fleet/global/schedule", h.GetGlobalSchedule).Methods("GET").Name("set_global_schedule")
	r.Handle("/api/v1/fleet/global/schedule", h.GlobalScheduleQuery).Methods("POST").Name("add_to_global_schedule")
	r.Handle("/api/v1/fleet/global/schedule/{id:[0-9]+}", h.ModifyGlobalSchedule).Methods("PATCH").Name("modify_global_schedule")
	r.Handle("/api/v1/fleet/global/schedule/{id:[0-9]+}", h.DeleteGlobalSchedule).Methods("DELETE").Name("delete_global_schedule")

	r.Handle("/api/v1/fleet/labels", h.CreateLabel).Methods("POST").Name("create_label")
	r.Handle("/api/v1/fleet/labels/{id:[0-9]+}", h.ModifyLabel).Methods("PATCH").Name("modify_label")
	r.Handle("/api/v1/fleet/labels/{id:[0-9]+}", h.GetLabel).Methods("GET").Name("get_label")
	r.Handle("/api/v1/fleet/labels", h.ListLabels).Methods("GET").Name("list_labels")
	r.Handle("/api/v1/fleet/labels/{id:[0-9]+}/hosts", h.ListHostsInLabel).Methods("GET").Name("list_hosts_in_label")
	r.Handle("/api/v1/fleet/labels/{name}", h.DeleteLabel).Methods("DELETE").Name("delete_label")
	r.Handle("/api/v1/fleet/labels/id/{id:[0-9]+}", h.DeleteLabelByID).Methods("DELETE").Name("delete_label_by_id")
	r.Handle("/api/v1/fleet/spec/labels", h.ApplyLabelSpecs).Methods("POST").Name("apply_label_specs")
	r.Handle("/api/v1/fleet/spec/labels", h.GetLabelSpecs).Methods("GET").Name("get_label_specs")
	r.Handle("/api/v1/fleet/spec/labels/{name}", h.GetLabelSpec).Methods("GET").Name("get_label_spec")

	r.Handle("/api/v1/fleet/hosts/identifier/{identifier}", h.HostByIdentifier).Methods("GET").Name("host_by_identifier")
	r.Handle("/api/v1/fleet/hosts/{id:[0-9]+}", h.DeleteHost).Methods("DELETE").Name("delete_host")
	r.Handle("/api/v1/fleet/hosts/transfer", h.AddHostsToTeam).Methods("POST").Name("add_hosts_to_team")
	r.Handle("/api/v1/fleet/hosts/transfer/filter", h.AddHostsToTeamByFilter).Methods("POST").Name("add_hosts_to_team_by_filter")
	r.Handle("/api/v1/fleet/hosts/{id:[0-9]+}/refetch", h.RefetchHost).Methods("POST").Name("refetch_host")

	r.Handle("/api/v1/fleet/targets", h.SearchTargets).Methods("POST").Name("search_targets")

	r.Handle("/api/v1/fleet/version", h.Version).Methods("GET").Name("version")

	r.Handle("/api/v1/fleet/status/result_store", h.StatusResultStore).Methods("GET").Name("status_result_store")
	r.Handle("/api/v1/fleet/status/live_query", h.StatusLiveQuery).Methods("GET").Name("status_live_query")

	r.Handle("/api/v1/fleet/teams", h.CreateTeam).Methods("POST").Name("create_team")
	r.Handle("/api/v1/fleet/teams", h.ListTeams).Methods("GET").Name("list_teams")
	r.Handle("/api/v1/fleet/teams/{id:[0-9]+}", h.ModifyTeam).Methods("PATCH").Name("modify_team")
	r.Handle("/api/v1/fleet/teams/{id:[0-9]+}", h.DeleteTeam).Methods("DELETE").Name("delete_team")
	r.Handle("/api/v1/fleet/teams/{id:[0-9]+}/agent_options", h.ModifyTeamAgentOptions).Methods("POST").Name("modify_team_agent_options")
	r.Handle("/api/v1/fleet/teams/{id:[0-9]+}/users", h.ListTeamUsers).Methods("GET").Name("team_users")
	r.Handle("/api/v1/fleet/teams/{id:[0-9]+}/users", h.AddTeamUsers).Methods("PATCH").Name("add_team_users")
	r.Handle("/api/v1/fleet/teams/{id:[0-9]+}/users", h.DeleteTeamUsers).Methods("DELETE").Name("delete_team_users")
	r.Handle("/api/v1/fleet/teams/{id:[0-9]+}/secrets", h.TeamEnrollSecrets).Methods("GET").Name("get_team_enroll_secrets")
	r.Handle("/api/v1/osquery/enroll", h.EnrollAgent).Methods("POST").Name("enroll_agent")
	r.Handle("/api/v1/osquery/config", h.GetClientConfig).Methods("POST").Name("get_client_config")
	r.Handle("/api/v1/osquery/distributed/read", h.GetDistributedQueries).Methods("POST").Name("get_distributed_queries")
	r.Handle("/api/v1/osquery/distributed/write", h.SubmitDistributedQueryResults).Methods("POST").Name("submit_distributed_query_results")
	r.Handle("/api/v1/osquery/log", h.SubmitLogs).Methods("POST").Name("submit_logs")
	r.Handle("/api/v1/osquery/carve/begin", h.CarveBegin).Methods("POST").Name("carve_begin")
	r.Handle("/api/v1/osquery/carve/block", h.CarveBlock).Methods("POST").Name("carve_block")
}

func attachNewStyleFleetAPIRoutes(r *mux.Router, svc fleet.Service, opts []kithttp.ServerOption) {
	e := NewUserAuthenticatedEndpointer(svc, opts, r)
	e.POST("/api/v1/fleet/users/roles/spec", applyUserRoleSpecsEndpoint, applyUserRoleSpecsRequest{})
	e.POST("/api/v1/fleet/translate", translatorEndpoint, translatorRequest{})
	e.POST("/api/v1/fleet/spec/teams", applyTeamSpecsEndpoint, applyTeamSpecsRequest{})
	e.PATCH("/api/v1/fleet/teams/{team_id:[0-9]+}/secrets", modifyTeamEnrollSecretsEndpoint, modifyTeamEnrollSecretsRequest{})

	// Alias /api/v1/fleet/team/ -> /api/v1/fleet/teams/
	e.GET("/api/v1/fleet/team/{team_id}/schedule", getTeamScheduleEndpoint, getTeamScheduleRequest{})
	e.POST("/api/v1/fleet/team/{team_id}/schedule", teamScheduleQueryEndpoint, teamScheduleQueryRequest{})
	e.PATCH("/api/v1/fleet/team/{team_id}/schedule/{scheduled_query_id}", modifyTeamScheduleEndpoint, modifyTeamScheduleRequest{})
	e.DELETE("/api/v1/fleet/team/{team_id}/schedule/{scheduled_query_id}", deleteTeamScheduleEndpoint, deleteTeamScheduleRequest{})
	// End alias

	e.GET("/api/v1/fleet/teams/{team_id}/schedule", getTeamScheduleEndpoint, getTeamScheduleRequest{})
	e.POST("/api/v1/fleet/teams/{team_id}/schedule", teamScheduleQueryEndpoint, teamScheduleQueryRequest{})
	e.PATCH("/api/v1/fleet/teams/{team_id}/schedule/{scheduled_query_id}", modifyTeamScheduleEndpoint, modifyTeamScheduleRequest{})
	e.DELETE("/api/v1/fleet/teams/{team_id}/schedule/{scheduled_query_id}", deleteTeamScheduleEndpoint, deleteTeamScheduleRequest{})

	e.POST("/api/v1/fleet/global/policies", globalPolicyEndpoint, globalPolicyRequest{})
	e.GET("/api/v1/fleet/global/policies", listGlobalPoliciesEndpoint, nil)
	e.GET("/api/v1/fleet/global/policies/{policy_id}", getPolicyByIDEndpoint, getPolicyByIDRequest{})
	e.POST("/api/v1/fleet/global/policies/delete", deleteGlobalPoliciesEndpoint, deleteGlobalPoliciesRequest{})
	e.PATCH("/api/v1/fleet/global/policies/{policy_id}", modifyGlobalPolicyEndpoint, modifyGlobalPolicyRequest{})

	// Alias /api/v1/fleet/team/ -> /api/v1/fleet/teams/
	e.POST("/api/v1/fleet/team/{team_id}/policies", teamPolicyEndpoint, teamPolicyRequest{})
	e.GET("/api/v1/fleet/team/{team_id}/policies", listTeamPoliciesEndpoint, listTeamPoliciesRequest{})
	e.GET("/api/v1/fleet/team/{team_id}/policies/{policy_id}", getTeamPolicyByIDEndpoint, getTeamPolicyByIDRequest{})
	e.POST("/api/v1/fleet/team/{team_id}/policies/delete", deleteTeamPoliciesEndpoint, deleteTeamPoliciesRequest{})
	// End alias

	e.POST("/api/v1/fleet/teams/{team_id}/policies", teamPolicyEndpoint, teamPolicyRequest{})
	e.GET("/api/v1/fleet/teams/{team_id}/policies", listTeamPoliciesEndpoint, listTeamPoliciesRequest{})
	e.GET("/api/v1/fleet/teams/{team_id}/policies/{policy_id}", getTeamPolicyByIDEndpoint, getTeamPolicyByIDRequest{})
	e.POST("/api/v1/fleet/teams/{team_id}/policies/delete", deleteTeamPoliciesEndpoint, deleteTeamPoliciesRequest{})
	e.PATCH("/api/v1/fleet/teams/{team_id}/policies/{policy_id}", modifyTeamPolicyEndpoint, modifyTeamPolicyRequest{})

	e.POST("/api/v1/fleet/spec/policies", applyPolicySpecsEndpoint, applyPolicySpecsRequest{})

	e.GET("/api/v1/fleet/packs/{id:[0-9]+}", getPackEndpoint, getPackRequest{})

	e.GET("/api/v1/fleet/software", listSoftwareEndpoint, listSoftwareRequest{})
	e.GET("/api/v1/fleet/software/count", countSoftwareEndpoint, countSoftwareRequest{})

	e.GET("/api/v1/fleet/host_summary", getHostSummaryEndpoint, getHostSummaryRequest{})
	e.GET("/api/v1/fleet/hosts", listHostsEndpoint, listHostsRequest{})
	e.POST("/api/v1/fleet/hosts/delete", deleteHostsEndpoint, deleteHostsRequest{})
	e.GET("/api/v1/fleet/hosts/{id:[0-9]+}", getHostEndpoint, getHostRequest{})
	e.GET("/api/v1/fleet/hosts/count", countHostsEndpoint, countHostsRequest{})

	e.GET("/api/v1/fleet/queries/run", runLiveQueryEndpoint, runLiveQueryRequest{})

	e.PATCH("/api/v1/fleet/invites/{id:[0-9]+}", updateInviteEndpoint, updateInviteRequest{})

	e.GET("/api/v1/fleet/activities", listActivitiesEndpoint, listActivitiesRequest{})

	e.GET("/api/v1/fleet/carves", listCarvesEndpoint, listCarvesRequest{})
	e.GET("/api/v1/fleet/carves/{id:[0-9]+}", getCarveEndpoint, getCarveRequest{})
	e.GET("/api/v1/fleet/carves/{id:[0-9]+}/block/{block_id}", getCarveBlockEndpoint, getCarveBlockRequest{})
}

// TODO: this duplicates the one in makeKitHandler
func newServer(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc, opts []kithttp.ServerOption) http.Handler {
	e = authzcheck.NewMiddleware().AuthzCheck()(e)
	return kithttp.NewServer(e, decodeFn, encodeResponse, opts...)
}

// WithSetup is an http middleware that checks if setup procedures have been completed.
// If setup hasn't been completed it serves the API with a setup middleware.
// If the server is already configured, the default API handler is exposed.
func WithSetup(svc fleet.Service, logger kitlog.Logger, next http.Handler) http.HandlerFunc {
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
		requireSetup, err := svc.SetupRequired(context.Background())
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
func RedirectLoginToSetup(svc fleet.Service, logger kitlog.Logger, next http.Handler, urlPrefix string) http.HandlerFunc {
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

		setupRequired, err := svc.SetupRequired(context.Background())
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

// RedirectSetupToLogin forces the /setup path to be redirected to login. This middleware is used after
// the app has been setup.
func RedirectSetupToLogin(svc fleet.Service, logger kitlog.Logger, next http.Handler, urlPrefix string) http.HandlerFunc {
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
