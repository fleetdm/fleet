package service

import (
	"context"
	"errors"
	"net/http"
	"regexp"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
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

type extraHandlerOpts struct {
	loginRateLimit *throttled.Rate
}

// ExtraHandlerOption allows adding extra configuration to the HTTP handler.
type ExtraHandlerOption func(*extraHandlerOpts)

// WithLoginRateLimit configures the rate limit for the login endpoint.
func WithLoginRateLimit(r throttled.Rate) ExtraHandlerOption {
	return func(o *extraHandlerOpts) {
		o.loginRateLimit = &r
	}
}

// MakeHandler creates an HTTP handler for the Fleet server endpoints.
func MakeHandler(
	svc fleet.Service,
	config config.FleetConfig,
	logger kitlog.Logger,
	limitStore throttled.GCRAStore,
	extra ...ExtraHandlerOption,
) http.Handler {
	var eopts extraHandlerOpts
	for _, fn := range extra {
		fn(&eopts)
	}

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

	r := mux.NewRouter()
	if config.Logging.TracingEnabled && config.Logging.TracingType == "opentelemetry" {
		r.Use(otmiddleware.Middleware("fleet"))
	}

	r.Use(publicIP)

	attachFleetAPIRoutes(r, svc, config, logger, limitStore, fleetAPIOptions, eopts)
	addMetrics(r)

	return r
}

func publicIP(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if ip != "" {
			r.RemoteAddr = ip
		}
		handler.ServeHTTP(w, r.WithContext(publicip.NewContext(r.Context(), ip)))
	})
}

// PrometheusMetricsHandler wraps the provided handler with prometheus metrics
// middleware and returns the resulting handler that should be mounted for that
// route.
func PrometheusMetricsHandler(name string, handler http.Handler) http.Handler {
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
		route.Handler(PrometheusMetricsHandler(route.GetName(), route.GetHandler()))
		return nil
	}
	r.Walk(walkFn)
}

// desktopRateLimitMaxBurst is the max burst used for device request rate limiting.
//
// Defined as const to be used in tests.
const desktopRateLimitMaxBurst = 100

func attachFleetAPIRoutes(r *mux.Router, svc fleet.Service, config config.FleetConfig,
	logger kitlog.Logger, limitStore throttled.GCRAStore, opts []kithttp.ServerOption,
	extra extraHandlerOpts,
) {
	apiVersions := []string{"v1", "2022-04"}

	// user-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(svc, opts, r, apiVersions...)

	ue.POST("/api/_version_/fleet/trigger", triggerEndpoint, triggerRequest{})

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
	// TODO: searchTargetsEndpoint will be removed in Fleet 5.0
	ue.POST("/api/_version_/fleet/targets", searchTargetsEndpoint, searchTargetsRequest{})
	ue.POST("/api/_version_/fleet/targets/count", countTargetsEndpoint, countTargetsRequest{})

	ue.POST("/api/_version_/fleet/invites", createInviteEndpoint, createInviteRequest{})
	ue.GET("/api/_version_/fleet/invites", listInvitesEndpoint, listInvitesRequest{})
	ue.DELETE("/api/_version_/fleet/invites/{id:[0-9]+}", deleteInviteEndpoint, deleteInviteRequest{})
	ue.PATCH("/api/_version_/fleet/invites/{id:[0-9]+}", updateInviteEndpoint, updateInviteRequest{})

	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/global/policies", globalPolicyEndpoint, globalPolicyRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/policies", globalPolicyEndpoint, globalPolicyRequest{})
	ue.EndingAtVersion("v1").GET("/api/_version_/fleet/global/policies", listGlobalPoliciesEndpoint, nil)
	ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/policies", listGlobalPoliciesEndpoint, nil)
	ue.EndingAtVersion("v1").GET("/api/_version_/fleet/global/policies/{policy_id}", getPolicyByIDEndpoint, getPolicyByIDRequest{})
	ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/policies/{policy_id}", getPolicyByIDEndpoint, getPolicyByIDRequest{})
	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/global/policies/delete", deleteGlobalPoliciesEndpoint, deleteGlobalPoliciesRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/policies/delete", deleteGlobalPoliciesEndpoint, deleteGlobalPoliciesRequest{})
	ue.EndingAtVersion("v1").PATCH("/api/_version_/fleet/global/policies/{policy_id}", modifyGlobalPolicyEndpoint, modifyGlobalPolicyRequest{})
	ue.StartingAtVersion("2022-04").PATCH("/api/_version_/fleet/policies/{policy_id}", modifyGlobalPolicyEndpoint, modifyGlobalPolicyRequest{})

	// Alias /api/_version_/fleet/team/ -> /api/_version_/fleet/teams/
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies").
		POST("/api/_version_/fleet/teams/{team_id}/policies", teamPolicyEndpoint, teamPolicyRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies").
		GET("/api/_version_/fleet/teams/{team_id}/policies", listTeamPoliciesEndpoint, listTeamPoliciesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies/{policy_id}").
		GET("/api/_version_/fleet/teams/{team_id}/policies/{policy_id}", getTeamPolicyByIDEndpoint, getTeamPolicyByIDRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies/delete").
		POST("/api/_version_/fleet/teams/{team_id}/policies/delete", deleteTeamPoliciesEndpoint, deleteTeamPoliciesRequest{})
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
	ue.GET("/api/_version_/fleet/software/{id:[0-9]+}", getSoftwareEndpoint, getSoftwareRequest{})
	ue.GET("/api/_version_/fleet/software/count", countSoftwareEndpoint, countSoftwareRequest{})

	ue.GET("/api/_version_/fleet/host_summary", getHostSummaryEndpoint, getHostSummaryRequest{})
	ue.GET("/api/_version_/fleet/hosts", listHostsEndpoint, listHostsRequest{})
	ue.POST("/api/_version_/fleet/hosts/delete", deleteHostsEndpoint, deleteHostsRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}", getHostEndpoint, getHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/count", countHostsEndpoint, countHostsRequest{})
	ue.POST("/api/_version_/fleet/hosts/search", searchHostsEndpoint, searchHostsRequest{})
	ue.GET("/api/_version_/fleet/hosts/identifier/{identifier}", hostByIdentifierEndpoint, hostByIdentifierRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}", deleteHostEndpoint, deleteHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/transfer", addHostsToTeamEndpoint, addHostsToTeamRequest{})
	ue.POST("/api/_version_/fleet/hosts/transfer/filter", addHostsToTeamByFilterEndpoint, addHostsToTeamByFilterRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/refetch", refetchHostEndpoint, refetchHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/device_mapping", listHostDeviceMappingEndpoint, listHostDeviceMappingRequest{})
	ue.GET("/api/_version_/fleet/hosts/report", hostsReportEndpoint, hostsReportRequest{})
	ue.GET("/api/_version_/fleet/os_versions", osVersionsEndpoint, osVersionsRequest{})

	ue.GET("/api/_version_/fleet/hosts/summary/mdm", getHostMDMSummary, getHostMDMSummaryRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/mdm", getHostMDM, getHostMDMRequest{})

	ue.POST("/api/_version_/fleet/labels", createLabelEndpoint, createLabelRequest{})
	ue.PATCH("/api/_version_/fleet/labels/{id:[0-9]+}", modifyLabelEndpoint, modifyLabelRequest{})
	ue.GET("/api/_version_/fleet/labels/{id:[0-9]+}", getLabelEndpoint, getLabelRequest{})
	ue.GET("/api/_version_/fleet/labels", listLabelsEndpoint, listLabelsRequest{})
	ue.GET("/api/_version_/fleet/labels/summary", getLabelsSummaryEndpoint, nil)
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

	ue.POST("/api/_version_/fleet/download_installer/{kind}", getInstallerEndpoint, getInstallerRequest{})
	ue.HEAD("/api/_version_/fleet/download_installer/{kind}", checkInstallerEndpoint, checkInstallerRequest{})

	ue.GET("/api/_version_/fleet/packs/{id:[0-9]+}/scheduled", getScheduledQueriesInPackEndpoint, getScheduledQueriesInPackRequest{})
	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/schedule", scheduleQueryEndpoint, scheduleQueryRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/packs/schedule", scheduleQueryEndpoint, scheduleQueryRequest{})
	ue.GET("/api/_version_/fleet/schedule/{id:[0-9]+}", getScheduledQueryEndpoint, getScheduledQueryRequest{})
	ue.EndingAtVersion("v1").PATCH("/api/_version_/fleet/schedule/{id:[0-9]+}", modifyScheduledQueryEndpoint, modifyScheduledQueryRequest{})
	ue.StartingAtVersion("2022-04").PATCH("/api/_version_/fleet/packs/schedule/{id:[0-9]+}", modifyScheduledQueryEndpoint, modifyScheduledQueryRequest{})
	ue.EndingAtVersion("v1").DELETE("/api/_version_/fleet/schedule/{id:[0-9]+}", deleteScheduledQueryEndpoint, deleteScheduledQueryRequest{})
	ue.StartingAtVersion("2022-04").DELETE("/api/_version_/fleet/packs/schedule/{id:[0-9]+}", deleteScheduledQueryEndpoint, deleteScheduledQueryRequest{})

	ue.EndingAtVersion("v1").GET("/api/_version_/fleet/global/schedule", getGlobalScheduleEndpoint, getGlobalScheduleRequest{})
	ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/schedule", getGlobalScheduleEndpoint, getGlobalScheduleRequest{})
	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/global/schedule", globalScheduleQueryEndpoint, globalScheduleQueryRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/schedule", globalScheduleQueryEndpoint, globalScheduleQueryRequest{})
	ue.EndingAtVersion("v1").PATCH("/api/_version_/fleet/global/schedule/{id:[0-9]+}", modifyGlobalScheduleEndpoint, modifyGlobalScheduleRequest{})
	ue.StartingAtVersion("2022-04").PATCH("/api/_version_/fleet/schedule/{id:[0-9]+}", modifyGlobalScheduleEndpoint, modifyGlobalScheduleRequest{})
	ue.EndingAtVersion("v1").DELETE("/api/_version_/fleet/global/schedule/{id:[0-9]+}", deleteGlobalScheduleEndpoint, deleteGlobalScheduleRequest{})
	ue.StartingAtVersion("2022-04").DELETE("/api/_version_/fleet/schedule/{id:[0-9]+}", deleteGlobalScheduleEndpoint, deleteGlobalScheduleRequest{})

	// Alias /api/_version_/fleet/team/ -> /api/_version_/fleet/teams/
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/schedule").
		GET("/api/_version_/fleet/teams/{team_id}/schedule", getTeamScheduleEndpoint, getTeamScheduleRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/schedule").
		POST("/api/_version_/fleet/teams/{team_id}/schedule", teamScheduleQueryEndpoint, teamScheduleQueryRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/schedule/{scheduled_query_id}").
		PATCH("/api/_version_/fleet/teams/{team_id}/schedule/{scheduled_query_id}", modifyTeamScheduleEndpoint, modifyTeamScheduleRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/schedule/{scheduled_query_id}").
		DELETE("/api/_version_/fleet/teams/{team_id}/schedule/{scheduled_query_id}", deleteTeamScheduleEndpoint, deleteTeamScheduleRequest{})

	ue.GET("/api/_version_/fleet/carves", listCarvesEndpoint, listCarvesRequest{})
	ue.GET("/api/_version_/fleet/carves/{id:[0-9]+}", getCarveEndpoint, getCarveRequest{})
	ue.GET("/api/_version_/fleet/carves/{id:[0-9]+}/block/{block_id}", getCarveBlockEndpoint, getCarveBlockRequest{})

	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/macadmins", getMacadminsDataEndpoint, getMacadminsDataRequest{})
	ue.GET("/api/_version_/fleet/macadmins", getAggregatedMacadminsDataEndpoint, getAggregatedMacadminsDataRequest{})

	ue.GET("/api/_version_/fleet/status/result_store", statusResultStoreEndpoint, nil)
	ue.GET("/api/_version_/fleet/status/live_query", statusLiveQueryEndpoint, nil)

	if config.MDMApple.Enable {
		ue.POST("/api/_version_/fleet/mdm/apple/enrollmentprofiles", createMDMAppleEnrollmentProfilesEndpoint, createMDMAppleEnrollmentProfileRequest{})
		ue.GET("/api/_version_/fleet/mdm/apple/enrollmentprofiles", listMDMAppleEnrollmentsEndpoint, listMDMAppleEnrollmentProfilesRequest{})
		ue.POST("/api/_version_/fleet/mdm/apple/enqueue", enqueueMDMAppleCommandEndpoint, enqueueMDMAppleCommandRequest{})
		ue.GET("/api/_version_/fleet/mdm/apple/commandresults", getMDMAppleCommandResultsEndpoint, getMDMAppleCommandResultsRequest{})
		ue.POST("/api/_version_/fleet/mdm/apple/installers", uploadAppleInstallerEndpoint, uploadAppleInstallerRequest{})
		ue.GET("/api/_version_/fleet/mdm/apple/installers/{installer_id:[0-9]+}", getAppleInstallerEndpoint, getAppleInstallerDetailsRequest{})
		ue.DELETE("/api/_version_/fleet/mdm/apple/installers/{installer_id:[0-9]+}", deleteAppleInstallerEndpoint, deleteAppleInstallerDetailsRequest{})
		ue.GET("/api/_version_/fleet/mdm/apple/installers", listMDMAppleInstallersEndpoint, listMDMAppleInstallersRequest{})
		ue.GET("/api/_version_/fleet/mdm/apple/devices", listMDMAppleDevicesEndpoint, listMDMAppleDevicesRequest{})
		ue.GET("/api/_version_/fleet/mdm/apple/dep/devices", listMDMAppleDEPDevicesEndpoint, listMDMAppleDEPDevicesRequest{})
	}

	errorLimiter := ratelimit.NewErrorMiddleware(limitStore)

	// device-authenticated endpoints
	de := newDeviceAuthenticatedEndpointer(svc, logger, opts, r, apiVersions...)
	// We allow a quota of 720 because in the onboarding of a Fleet Desktop takes a few tries until it authenticates
	// properly
	desktopQuota := throttled.RateQuota{MaxRate: throttled.PerHour(720), MaxBurst: desktopRateLimitMaxBurst}
	de.WithCustomMiddleware(
		errorLimiter.Limit("get_device_host", desktopQuota),
	).GET("/api/_version_/fleet/device/{token}", getDeviceHostEndpoint, getDeviceHostRequest{})
	de.WithCustomMiddleware(
		errorLimiter.Limit("get_fleet_desktop", desktopQuota),
	).GET("/api/_version_/fleet/device/{token}/desktop", getFleetDesktopEndpoint, getFleetDesktopRequest{})
	de.WithCustomMiddleware(
		errorLimiter.Limit("refetch_device_host", desktopQuota),
	).POST("/api/_version_/fleet/device/{token}/refetch", refetchDeviceHostEndpoint, refetchDeviceHostRequest{})
	de.WithCustomMiddleware(
		errorLimiter.Limit("get_device_mapping", desktopQuota),
	).GET("/api/_version_/fleet/device/{token}/device_mapping", listDeviceHostDeviceMappingEndpoint, listDeviceHostDeviceMappingRequest{})
	de.WithCustomMiddleware(
		errorLimiter.Limit("get_device_macadmins", desktopQuota),
	).GET("/api/_version_/fleet/device/{token}/macadmins", getDeviceMacadminsDataEndpoint, getDeviceMacadminsDataRequest{})
	de.WithCustomMiddleware(
		errorLimiter.Limit("get_device_policies", desktopQuota),
	).GET("/api/_version_/fleet/device/{token}/policies", listDevicePoliciesEndpoint, listDevicePoliciesRequest{})
	de.WithCustomMiddleware(
		errorLimiter.Limit("get_device_transparency", desktopQuota),
	).GET("/api/_version_/fleet/device/{token}/transparency", transparencyURL, transparencyURLRequest{})

	// host-authenticated endpoints
	he := newHostAuthenticatedEndpointer(svc, logger, opts, r, apiVersions...)

	// Note that the /osquery/ endpoints are *not* versioned, i.e. there is no
	// `_version_` placeholder in the path. This is deliberate, see
	// https://github.com/fleetdm/fleet/pull/4731#discussion_r838931732 For now
	// we add an alias to `/api/v1/osquery` so that it is backwards compatible,
	// but even that `v1` is *not* part of the standard versioning, it will still
	// work even after we remove support for the `v1` version for the rest of the
	// API. This allows us to deprecate osquery endpoints separately.
	he.WithAltPaths("/api/v1/osquery/config").
		POST("/api/osquery/config", getClientConfigEndpoint, getClientConfigRequest{})
	he.WithAltPaths("/api/v1/osquery/distributed/read").
		POST("/api/osquery/distributed/read", getDistributedQueriesEndpoint, getDistributedQueriesRequest{})
	he.WithAltPaths("/api/v1/osquery/distributed/write").
		POST("/api/osquery/distributed/write", submitDistributedQueryResultsEndpoint, submitDistributedQueryResultsRequestShim{})
	he.WithAltPaths("/api/v1/osquery/carve/begin").
		POST("/api/osquery/carve/begin", carveBeginEndpoint, carveBeginRequest{})
	he.WithAltPaths("/api/v1/osquery/log").
		POST("/api/osquery/log", submitLogsEndpoint, submitLogsRequest{})

	// orbit authenticated endpoints
	oe := newOrbitAuthenticatedEndpointer(svc, logger, opts, r, apiVersions...)
	oe.POST("/api/fleet/orbit/device_token", setOrUpdateDeviceTokenEndpoint, setOrUpdateDeviceTokenRequest{})
	oe.POST("/api/fleet/orbit/config", getOrbitConfigEndpoint, orbitGetConfigRequest{})

	// unauthenticated endpoints - most of those are either login-related,
	// invite-related or host-enrolling. So they typically do some kind of
	// one-time authentication by verifying that a valid secret token is provided
	// with the request.
	ne := newNoAuthEndpointer(svc, opts, r, apiVersions...)
	ne.WithAltPaths("/api/v1/osquery/enroll").
		POST("/api/osquery/enroll", enrollAgentEndpoint, enrollAgentRequest{})

	if config.MDMApple.Enable {
		// These endpoint are token authenticated.
		ne.GET(apple_mdm.EnrollPath, mdmAppleEnrollEndpoint, mdmAppleEnrollRequest{})
		ne.GET(apple_mdm.InstallerPath, mdmAppleGetInstallerEndpoint, mdmAppleGetInstallerRequest{})
		ne.HEAD(apple_mdm.InstallerPath, mdmAppleHeadInstallerEndpoint, mdmAppleHeadInstallerRequest{})
	}

	ne.POST("/api/fleet/orbit/enroll", enrollOrbitEndpoint, EnrollOrbitRequest{})

	// For some reason osquery does not provide a node key with the block data.
	// Instead the carve session ID should be verified in the service method.
	ne.WithAltPaths("/api/v1/osquery/carve/block").
		POST("/api/osquery/carve/block", carveBlockEndpoint, carveBlockRequest{})

	ne.POST("/api/_version_/fleet/perform_required_password_reset", performRequiredPasswordResetEndpoint, performRequiredPasswordResetRequest{})
	ne.POST("/api/_version_/fleet/users", createUserFromInviteEndpoint, createUserRequest{})
	ne.GET("/api/_version_/fleet/invites/{token}", verifyInviteEndpoint, verifyInviteRequest{})
	ne.POST("/api/_version_/fleet/reset_password", resetPasswordEndpoint, resetPasswordRequest{})
	ne.POST("/api/_version_/fleet/logout", logoutEndpoint, nil)
	ne.POST("/api/v1/fleet/sso", initiateSSOEndpoint, initiateSSORequest{})
	ne.POST("/api/v1/fleet/sso/callback", makeCallbackSSOEndpoint(config.Server.URLPrefix), callbackSSORequest{})
	ne.GET("/api/v1/fleet/sso", settingsSSOEndpoint, nil)

	// the websocket distributed query results endpoint is a bit different - the
	// provided path is a prefix, not an exact match, and it is not a go-kit
	// endpoint but a raw http.Handler. It uses the NoAuthEndpointer because
	// authentication is done when the websocket session is established, inside
	// the handler.
	ne.UsePathPrefix().PathHandler("GET", "/api/_version_/fleet/results/", makeStreamDistributedQueryCampaignResultsHandler(svc, logger))

	quota := throttled.RateQuota{MaxRate: throttled.PerHour(10), MaxBurst: 90}
	limiter := ratelimit.NewMiddleware(limitStore)
	ne.
		WithCustomMiddleware(limiter.Limit("forgot_password", quota)).
		POST("/api/_version_/fleet/forgot_password", forgotPasswordEndpoint, forgotPasswordRequest{})

	loginRateLimit := throttled.PerMin(10)
	if extra.loginRateLimit != nil {
		loginRateLimit = *extra.loginRateLimit
	}

	ne.WithCustomMiddleware(limiter.Limit("login", throttled.RateQuota{MaxRate: loginRateLimit, MaxBurst: 9})).
		POST("/api/_version_/fleet/login", loginEndpoint, loginRequest{})

	// Fleet Sandbox demo login (always errors unless config.server.sandbox_enabled is set)
	ne.WithCustomMiddleware(limiter.Limit("login", throttled.RateQuota{MaxRate: loginRateLimit, MaxBurst: 9})).
		POST("/api/_version_/fleet/demologin", makeDemologinEndpoint(config.Server.URLPrefix), demologinRequest{})

	ne.WithCustomMiddleware(
		errorLimiter.Limit("ping_device", desktopQuota),
	).HEAD("/api/fleet/device/ping", devicePingEndpoint, devicePingRequest{})

	ne.WithCustomMiddleware(
		errorLimiter.Limit("ping_orbit", desktopQuota),
	).HEAD("/api/fleet/orbit/ping", orbitPingEndpoint, orbitPingRequest{})
}

func newServer(e endpoint.Endpoint, decodeFn kithttp.DecodeRequestFunc, opts []kithttp.ServerOption) http.Handler {
	// TODO: some handlers don't have authz checks, and because the SkipAuth call is done only in the
	// endpoint handler, any middleware that raises errors before the handler is reached will end up
	// returning authz check missing instead of the more relevant error. Should be addressed as part
	// of #4406.
	e = authzcheck.NewMiddleware().AuthzCheck()(e)
	return kithttp.NewServer(e, decodeFn, encodeResponse, opts...)
}

// WithSetup is an http middleware that checks if setup procedures have been completed.
// If setup hasn't been completed it serves the API with a setup middleware.
// If the server is already configured, the default API handler is exposed.
func WithSetup(svc fleet.Service, logger kitlog.Logger, next http.Handler) http.HandlerFunc {
	rxOsquery := regexp.MustCompile(`^/api/[^/]+/osquery`)
	return func(w http.ResponseWriter, r *http.Request) {
		configRouter := http.NewServeMux()
		srv := kithttp.NewServer(
			makeSetupEndpoint(svc, logger),
			decodeSetupRequest,
			encodeResponse,
		)
		// NOTE: support setup on both /v1/ and version-less, in the future /v1/
		// will be dropped.
		configRouter.Handle("/api/v1/setup", srv)
		configRouter.Handle("/api/setup", srv)

		// whitelist osqueryd endpoints
		if rxOsquery.MatchString(r.URL.Path) {
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
