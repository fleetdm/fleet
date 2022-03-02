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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/throttled/throttled/v2"
	otmiddleware "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

// FleetEndpoints is a collection of RPC endpoints implemented by the Fleet API.
type FleetEndpoints struct {
	Login          endpoint.Endpoint
	Logout         endpoint.Endpoint
	ForgotPassword endpoint.Endpoint
	InitiateSSO    endpoint.Endpoint
	CallbackSSO    endpoint.Endpoint
	SSOSettings    endpoint.Endpoint
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
		InitiateSSO: logged(makeInitiateSSOEndpoint(svc)),
		CallbackSSO: logged(makeCallbackSSOEndpoint(svc, urlPrefix)),
		SSOSettings: logged(makeSSOSettingsEndpoint(svc)),
	}
}

type fleetHandlers struct {
	Login          http.Handler
	Logout         http.Handler
	ForgotPassword http.Handler
	InitiateSSO    http.Handler
	CallbackSSO    http.Handler
	SettingsSSO    http.Handler
}

func makeKitHandlers(e FleetEndpoints, opts []kithttp.ServerOption) *fleetHandlers {
	newServer := func(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc) http.Handler {
		e = authzcheck.NewMiddleware().AuthzCheck()(e)
		return kithttp.NewServer(e, decodeFn, encodeResponse, opts...)
	}
	return &fleetHandlers{
		Login:          newServer(e.Login, decodeLoginRequest),
		Logout:         newServer(e.Logout, decodeNoParamsRequest),
		ForgotPassword: newServer(e.ForgotPassword, decodeForgotPasswordRequest),
		InitiateSSO:    newServer(e.InitiateSSO, decodeInitiateSSORequest),
		CallbackSSO:    newServer(e.CallbackSSO, decodeCallbackSSORequest),
		SettingsSSO:    newServer(e.SSOSettings, decodeNoParamsRequest),
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
		kithttp.ServerErrorEncoder(encodeErrorAndTrySentry(config.Sentry.Dsn != "")),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
			logRequestEnd(logger),
			checkLicenseExpiration(svc),
		),
	}

	fleetEndpoints := MakeFleetServerEndpoints(svc, config.Server.URLPrefix, limitStore, logger)
	fleetHandlers := makeKitHandlers(fleetEndpoints, fleetAPIOptions)

	r := mux.NewRouter()
	if config.Logging.TracingEnabled && config.Logging.TracingType == "opentelemetry" {
		r.Use(otmiddleware.Middleware("fleet"))
	}

	attachFleetAPIRoutes(r, fleetHandlers)
	attachNewStyleFleetAPIRoutes(r, svc, logger, fleetAPIOptions)

	// Results endpoint is handled different due to websockets use
	// TODO: this would probably not work once v1 is deprecated
	r.PathPrefix("/api/v1/fleet/results/").
		Handler(makeStreamDistributedQueryCampaignResultsHandler(svc, logger)).
		Name("distributed_query_results")

	addMetrics(r)

	return r
}

// InstrumentHandler wraps the provided handler with prometheus metrics
// middleware and returns the resulting handler that should be mounted for that
// route.
func InstrumentHandler(name string, handler http.Handler) http.Handler {
	reg := prometheus.DefaultRegisterer
	registerOrExisting := func(coll prometheus.Collector) prometheus.Collector {
		if err := reg.Register(coll); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				return are.ExistingCollector
			}
			panic(err)
		}
		return coll
	}

	// this configuration is to keep prometheus metrics as close as possible to
	// what the v0.9.3 (that we used to use) provided via the now-deprecated
	// prometheus.InstrumentHandler.

	reqCnt := registerOrExisting(prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem:   "http",
			Name:        "requests_total",
			Help:        "Total number of HTTP requests made.",
			ConstLabels: prometheus.Labels{"handler": name},
		},
		[]string{"method", "code"},
	)).(*prometheus.CounterVec)

	reqDur := registerOrExisting(prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem:   "http",
			Name:        "request_duration_seconds",
			Help:        "The HTTP request latencies in seconds.",
			ConstLabels: prometheus.Labels{"handler": name},
			// Use default buckets, as they are suited for durations.
		},
		nil,
	)).(*prometheus.HistogramVec)

	// 1KB, 100KB, 1MB, 100MB, 1GB
	sizeBuckets := []float64{1024, 100 * 1024, 1024 * 1024, 100 * 1024 * 1024, 1024 * 1024 * 1024}

	resSz := registerOrExisting(prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem:   "http",
			Name:        "response_size_bytes",
			Help:        "The HTTP response sizes in bytes.",
			ConstLabels: prometheus.Labels{"handler": name},
			Buckets:     sizeBuckets,
		},
		nil,
	)).(*prometheus.HistogramVec)

	reqSz := registerOrExisting(prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem:   "http",
			Name:        "request_size_bytes",
			Help:        "The HTTP request sizes in bytes.",
			ConstLabels: prometheus.Labels{"handler": name},
			Buckets:     sizeBuckets,
		},
		nil,
	)).(*prometheus.HistogramVec)

	return promhttp.InstrumentHandlerDuration(reqDur,
		promhttp.InstrumentHandlerCounter(reqCnt,
			promhttp.InstrumentHandlerResponseSize(resSz,
				promhttp.InstrumentHandlerRequestSize(reqSz, handler))))
}

// addMetrics decorates each handler with prometheus instrumentation
func addMetrics(r *mux.Router) {
	walkFn := func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		route.Handler(InstrumentHandler(route.GetName(), route.GetHandler()))
		return nil
	}
	r.Walk(walkFn)
}

func attachFleetAPIRoutes(r *mux.Router, h *fleetHandlers) {
	r.Handle("/api/v1/fleet/login", h.Login).Methods("POST").Name("login")
	r.Handle("/api/v1/fleet/logout", h.Logout).Methods("POST").Name("logout")
	r.Handle("/api/v1/fleet/forgot_password", h.ForgotPassword).Methods("POST").Name("forgot_password")
	r.Handle("/api/v1/fleet/sso", h.InitiateSSO).Methods("POST").Name("intiate_sso")
	r.Handle("/api/v1/fleet/sso", h.SettingsSSO).Methods("GET").Name("sso_config")
	r.Handle("/api/v1/fleet/sso/callback", h.CallbackSSO).Methods("POST").Name("callback_sso")
}

func attachNewStyleFleetAPIRoutes(r *mux.Router, svc fleet.Service, logger kitlog.Logger, opts []kithttp.ServerOption) {
	// user-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(svc, opts, r, "v1")

	ue.GET("/api/_version_/fleet/me", meEndpoint, nil)
	ue.GET("/api/_version_/fleet/sessions/{id:[0-9]+}", getInfoAboutSessionEndpoint, getInfoAboutSessionRequest{})
	ue.DELETE("/api/_version_/fleet/sessions/{id:[0-9]+}", deleteSessionEndpoint, deleteSessionRequest{})

	ue.GET("/api/_version_/fleet/config/certificate", getCertificateEndpoint, nil)
	ue.GET("/api/_version_/fleet/config", getAppConfigEndpoint, nil)
	ue.PATCH("/api/_version_/fleet/config", modifyAppConfigEndpoint, modifyAppConfigRequest{})
	ue.POST("/api/_version_/fleet/spec/enroll_secret", applyEnrollSecretSpecEndpoint, applyEnrollSecretSpecRequest{})
	ue.GET("/api/_version_/fleet/spec/enroll_secret", getEnrollSecretSpecEndpoint, nil)
	ue.GET("/api/_version_/fleet/version", versionEndpoint, nil)

	ue.POST("/api/_version_/fleet/users/roles/spec", applyUserRoleSpecsEndpoint, applyUserRoleSpecsRequest{})
	ue.POST("/api/_version_/fleet/translate", translatorEndpoint, translatorRequest{})
	ue.POST("/api/_version_/fleet/spec/teams", applyTeamSpecsEndpoint, applyTeamSpecsRequest{})
	ue.PATCH("/api/_version_/fleet/teams/{team_id:[0-9]+}/secrets", modifyTeamEnrollSecretsEndpoint, modifyTeamEnrollSecretsRequest{})
	ue.POST("/api/_version_/fleet/teams", createTeamEndpoint, createTeamRequest{})
	ue.GET("/api/_version_/fleet/teams", listTeamsEndpoint, listTeamsRequest{})
	ue.GET("/api/_version_/fleet/teams/{id:[0-9]+}", getTeamEndpoint, getTeamRequest{})
	ue.PATCH("/api/_version_/fleet/teams/{id:[0-9]+}", modifyTeamEndpoint, modifyTeamRequest{})
	ue.DELETE("/api/_version_/fleet/teams/{id:[0-9]+}", deleteTeamEndpoint, deleteTeamRequest{})
	ue.POST("/api/_version_/fleet/teams/{id:[0-9]+}/agent_options", modifyTeamAgentOptionsEndpoint, modifyTeamAgentOptionsRequest{})
	ue.GET("/api/_version_/fleet/teams/{id:[0-9]+}/users", listTeamUsersEndpoint, listTeamUsersRequest{})
	ue.PATCH("/api/_version_/fleet/teams/{id:[0-9]+}/users", addTeamUsersEndpoint, modifyTeamUsersRequest{})
	ue.DELETE("/api/_version_/fleet/teams/{id:[0-9]+}/users", deleteTeamUsersEndpoint, modifyTeamUsersRequest{})
	ue.GET("/api/_version_/fleet/teams/{id:[0-9]+}/secrets", teamEnrollSecretsEndpoint, teamEnrollSecretsRequest{})

	// Alias /api/_version_/fleet/team/ -> /api/_version_/fleet/teams/
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/schedule").GET("/api/_version_/fleet/teams/{team_id}/schedule", getTeamScheduleEndpoint, getTeamScheduleRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/schedule").POST("/api/_version_/fleet/teams/{team_id}/schedule", teamScheduleQueryEndpoint, teamScheduleQueryRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/schedule/{scheduled_query_id}").PATCH("/api/_version_/fleet/teams/{team_id}/schedule/{scheduled_query_id}", modifyTeamScheduleEndpoint, modifyTeamScheduleRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/schedule/{scheduled_query_id}").DELETE("/api/_version_/fleet/teams/{team_id}/schedule/{scheduled_query_id}", deleteTeamScheduleEndpoint, deleteTeamScheduleRequest{})

	ue.GET("/api/_version_/fleet/users", listUsersEndpoint, listUsersRequest{})
	ue.POST("/api/_version_/fleet/users/admin", createUserEndpoint, createUserRequest{})
	ue.GET("/api/_version_/fleet/users/{id:[0-9]+}", getUserEndpoint, getUserRequest{})
	ue.PATCH("/api/_version_/fleet/users/{id:[0-9]+}", modifyUserEndpoint, modifyUserRequest{})
	ue.DELETE("/api/_version_/fleet/users/{id:[0-9]+}", deleteUserEndpoint, deleteUserRequest{})
	ue.POST("/api/_version_/fleet/users/{id:[0-9]+}/require_password_reset", requirePasswordResetEndpoint, requirePasswordResetRequest{})
	ue.GET("/api/_version_/fleet/users/{id:[0-9]+}/sessions", getInfoAboutSessionsForUserEndpoint, getInfoAboutSessionsForUserRequest{})
	ue.DELETE("/api/_version_/fleet/users/{id:[0-9]+}/sessions", deleteSessionsForUserEndpoint, deleteSessionsForUserRequest{})
	ue.POST("/api/_version_/fleet/change_password", changePasswordEndpoint, changePasswordRequest{})

	ue.GET("/api/_version_/fleet/email/change/{token}", changeEmailEndpoint, changeEmailRequest{})
	ue.POST("/api/_version_/fleet/targets", searchTargetsEndpoint, searchTargetsRequest{})

	ue.POST("/api/_version_/fleet/invites", createInviteEndpoint, createInviteRequest{})
	ue.GET("/api/_version_/fleet/invites", listInvitesEndpoint, listInvitesRequest{})
	ue.DELETE("/api/_version_/fleet/invites/{id:[0-9]+}", deleteInviteEndpoint, deleteInviteRequest{})
	ue.PATCH("/api/_version_/fleet/invites/{id:[0-9]+}", updateInviteEndpoint, updateInviteRequest{})

	ue.POST("/api/_version_/fleet/global/policies", globalPolicyEndpoint, globalPolicyRequest{})
	ue.GET("/api/_version_/fleet/global/policies", listGlobalPoliciesEndpoint, nil)
	ue.GET("/api/_version_/fleet/global/policies/{policy_id}", getPolicyByIDEndpoint, getPolicyByIDRequest{})
	ue.POST("/api/_version_/fleet/global/policies/delete", deleteGlobalPoliciesEndpoint, deleteGlobalPoliciesRequest{})
	ue.PATCH("/api/_version_/fleet/global/policies/{policy_id}", modifyGlobalPolicyEndpoint, modifyGlobalPolicyRequest{})

	// Alias /api/_version_/fleet/team/ -> /api/_version_/fleet/teams/
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies").POST("/api/_version_/fleet/teams/{team_id}/policies", teamPolicyEndpoint, teamPolicyRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies").GET("/api/_version_/fleet/teams/{team_id}/policies", listTeamPoliciesEndpoint, listTeamPoliciesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies/{policy_id}").GET("/api/_version_/fleet/teams/{team_id}/policies/{policy_id}", getTeamPolicyByIDEndpoint, getTeamPolicyByIDRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies/delete").POST("/api/_version_/fleet/teams/{team_id}/policies/delete", deleteTeamPoliciesEndpoint, deleteTeamPoliciesRequest{})
	ue.PATCH("/api/_version_/fleet/teams/{team_id}/policies/{policy_id}", modifyTeamPolicyEndpoint, modifyTeamPolicyRequest{})
	ue.POST("/api/_version_/fleet/spec/policies", applyPolicySpecsEndpoint, applyPolicySpecsRequest{})

	ue.GET("/api/_version_/fleet/queries/{id:[0-9]+}", getQueryEndpoint, getQueryRequest{})
	ue.GET("/api/_version_/fleet/queries", listQueriesEndpoint, listQueriesRequest{})
	ue.POST("/api/_version_/fleet/queries", createQueryEndpoint, createQueryRequest{})
	ue.PATCH("/api/_version_/fleet/queries/{id:[0-9]+}", modifyQueryEndpoint, modifyQueryRequest{})
	ue.DELETE("/api/_version_/fleet/queries/{name}", deleteQueryEndpoint, deleteQueryRequest{})
	ue.DELETE("/api/_version_/fleet/queries/id/{id:[0-9]+}", deleteQueryByIDEndpoint, deleteQueryByIDRequest{})
	ue.POST("/api/_version_/fleet/queries/delete", deleteQueriesEndpoint, deleteQueriesRequest{})
	ue.POST("/api/_version_/fleet/spec/queries", applyQuerySpecsEndpoint, applyQuerySpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/queries", getQuerySpecsEndpoint, nil)
	ue.GET("/api/_version_/fleet/spec/queries/{name}", getQuerySpecEndpoint, getGenericSpecRequest{})

	ue.GET("/api/_version_/fleet/packs/{id:[0-9]+}/scheduled", getScheduledQueriesInPackEndpoint, getScheduledQueriesInPackRequest{})
	ue.POST("/api/_version_/fleet/schedule", scheduleQueryEndpoint, scheduleQueryRequest{})
	ue.GET("/api/_version_/fleet/schedule/{id:[0-9]+}", getScheduledQueryEndpoint, getScheduledQueryRequest{})
	ue.PATCH("/api/_version_/fleet/schedule/{id:[0-9]+}", modifyScheduledQueryEndpoint, modifyScheduledQueryRequest{})
	ue.DELETE("/api/_version_/fleet/schedule/{id:[0-9]+}", deleteScheduledQueryEndpoint, deleteScheduledQueryRequest{})

	ue.GET("/api/_version_/fleet/packs/{id:[0-9]+}", getPackEndpoint, getPackRequest{})
	ue.POST("/api/_version_/fleet/packs", createPackEndpoint, createPackRequest{})
	ue.PATCH("/api/_version_/fleet/packs/{id:[0-9]+}", modifyPackEndpoint, modifyPackRequest{})
	ue.GET("/api/_version_/fleet/packs", listPacksEndpoint, listPacksRequest{})
	ue.DELETE("/api/_version_/fleet/packs/{name}", deletePackEndpoint, deletePackRequest{})
	ue.DELETE("/api/_version_/fleet/packs/id/{id:[0-9]+}", deletePackByIDEndpoint, deletePackByIDRequest{})
	ue.POST("/api/_version_/fleet/spec/packs", applyPackSpecsEndpoint, applyPackSpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/packs", getPackSpecsEndpoint, nil)
	ue.GET("/api/_version_/fleet/spec/packs/{name}", getPackSpecEndpoint, getGenericSpecRequest{})

	ue.GET("/api/_version_/fleet/software", listSoftwareEndpoint, listSoftwareRequest{})
	ue.GET("/api/_version_/fleet/software/count", countSoftwareEndpoint, countSoftwareRequest{})

	ue.GET("/api/_version_/fleet/host_summary", getHostSummaryEndpoint, getHostSummaryRequest{})
	ue.GET("/api/_version_/fleet/hosts", listHostsEndpoint, listHostsRequest{})
	ue.POST("/api/_version_/fleet/hosts/delete", deleteHostsEndpoint, deleteHostsRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}", getHostEndpoint, getHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/count", countHostsEndpoint, countHostsRequest{})
	ue.GET("/api/_version_/fleet/hosts/identifier/{identifier}", hostByIdentifierEndpoint, hostByIdentifierRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}", deleteHostEndpoint, deleteHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/transfer", addHostsToTeamEndpoint, addHostsToTeamRequest{})
	ue.POST("/api/_version_/fleet/hosts/transfer/filter", addHostsToTeamByFilterEndpoint, addHostsToTeamByFilterRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/refetch", refetchHostEndpoint, refetchHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/device_mapping", listHostDeviceMappingEndpoint, listHostDeviceMappingRequest{})

	ue.POST("/api/_version_/fleet/labels", createLabelEndpoint, createLabelRequest{})
	ue.PATCH("/api/_version_/fleet/labels/{id:[0-9]+}", modifyLabelEndpoint, modifyLabelRequest{})
	ue.GET("/api/_version_/fleet/labels/{id:[0-9]+}", getLabelEndpoint, getLabelRequest{})
	ue.GET("/api/_version_/fleet/labels", listLabelsEndpoint, listLabelsRequest{})
	ue.GET("/api/_version_/fleet/labels/{id:[0-9]+}/hosts", listHostsInLabelEndpoint, listHostsInLabelRequest{})
	ue.DELETE("/api/_version_/fleet/labels/{name}", deleteLabelEndpoint, deleteLabelRequest{})
	ue.DELETE("/api/_version_/fleet/labels/id/{id:[0-9]+}", deleteLabelByIDEndpoint, deleteLabelByIDRequest{})
	ue.POST("/api/_version_/fleet/spec/labels", applyLabelSpecsEndpoint, applyLabelSpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/labels", getLabelSpecsEndpoint, nil)
	ue.GET("/api/_version_/fleet/spec/labels/{name}", getLabelSpecEndpoint, getGenericSpecRequest{})

	ue.GET("/api/_version_/fleet/queries/run", runLiveQueryEndpoint, runLiveQueryRequest{})
	ue.POST("/api/_version_/fleet/queries/run", createDistributedQueryCampaignEndpoint, createDistributedQueryCampaignRequest{})
	ue.POST("/api/_version_/fleet/queries/run_by_names", createDistributedQueryCampaignByNamesEndpoint, createDistributedQueryCampaignByNamesRequest{})

	ue.GET("/api/_version_/fleet/activities", listActivitiesEndpoint, listActivitiesRequest{})

	ue.GET("/api/_version_/fleet/global/schedule", getGlobalScheduleEndpoint, getGlobalScheduleRequest{})
	ue.POST("/api/_version_/fleet/global/schedule", globalScheduleQueryEndpoint, globalScheduleQueryRequest{})
	ue.PATCH("/api/_version_/fleet/global/schedule/{id:[0-9]+}", modifyGlobalScheduleEndpoint, modifyGlobalScheduleRequest{})
	ue.DELETE("/api/_version_/fleet/global/schedule/{id:[0-9]+}", deleteGlobalScheduleEndpoint, deleteGlobalScheduleRequest{})

	ue.GET("/api/_version_/fleet/carves", listCarvesEndpoint, listCarvesRequest{})
	ue.GET("/api/_version_/fleet/carves/{id:[0-9]+}", getCarveEndpoint, getCarveRequest{})
	ue.GET("/api/_version_/fleet/carves/{id:[0-9]+}/block/{block_id}", getCarveBlockEndpoint, getCarveBlockRequest{})

	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/macadmins", getMacadminsDataEndpoint, getMacadminsDataRequest{})
	ue.GET("/api/_version_/fleet/macadmins", getAggregatedMacadminsDataEndpoint, getAggregatedMacadminsDataRequest{})

	ue.GET("/api/_version_/fleet/status/result_store", statusResultStoreEndpoint, nil)
	ue.GET("/api/_version_/fleet/status/live_query", statusLiveQueryEndpoint, nil)

	// host-authenticated endpoints
	he := newHostAuthenticatedEndpointer(svc, logger, opts, r, "v1")
	he.POST("/api/_version_/osquery/config", getClientConfigEndpoint, getClientConfigRequest{})
	he.POST("/api/_version_/osquery/distributed/read", getDistributedQueriesEndpoint, getDistributedQueriesRequest{})
	he.POST("/api/_version_/osquery/distributed/write", submitDistributedQueryResultsEndpoint, submitDistributedQueryResultsRequestShim{})
	he.POST("/api/_version_/osquery/carve/begin", carveBeginEndpoint, carveBeginRequest{})
	he.POST("/api/_version_/osquery/log", submitLogsEndpoint, submitLogsRequest{})

	// unauthenticated endpoints - most of those are either login-related,
	// invite-related or host-enrolling. So they typically do some kind of
	// one-time authentication by verifying that a valid secret token is provided
	// with the request.
	ne := newNoAuthEndpointer(svc, opts, r, "v1")
	ne.POST("/api/_version_/osquery/enroll", enrollAgentEndpoint, enrollAgentRequest{})

	// For some reason osquery does not provide a node key with the block data.
	// Instead the carve session ID should be verified in the service method.
	ne.POST("/api/_version_/osquery/carve/block", carveBlockEndpoint, carveBlockRequest{})

	ne.POST("/api/_version_/fleet/perform_required_password_reset", performRequiredPasswordResetEndpoint, performRequiredPasswordResetRequest{})
	ne.POST("/api/_version_/fleet/users", createUserFromInviteEndpoint, createUserRequest{})
	ne.GET("/api/_version_/fleet/invites/{token}", verifyInviteEndpoint, verifyInviteRequest{})
	ne.POST("/api/v1/fleet/reset_password", resetPasswordEndpoint, resetPasswordRequest{})
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

	// TODO: hard-codes v1 as a path fragment, which would probably not work once we
	// deprecate it for newer versions.

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
