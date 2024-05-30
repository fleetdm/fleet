package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/certverify"
	httpmdm "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http/mdm"
	nanomdm_log "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log"
	nanomdm_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/multi"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/nanomdm"
	nanomdm_storage "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	scep_depot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/service/middleware/authzcheck"
	"github.com/fleetdm/fleet/v4/server/service/middleware/mdmconfigured"
	"github.com/fleetdm/fleet/v4/server/service/middleware/ratelimit"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/throttled/throttled/v2"
	"go.elastic.co/apm/module/apmgorilla/v2"
	otmiddleware "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
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

	var uuider fleet.ErrorUUIDer
	if errors.As(err, &uuider) {
		logger = kitlog.With(logger, "uuid", uuider.UUID())
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
		kithttp.ServerErrorEncoder(encodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
			logRequestEnd(logger),
			checkLicenseExpiration(svc),
		),
	}

	r := mux.NewRouter()
	if config.Logging.TracingEnabled {
		if config.Logging.TracingType == "opentelemetry" {
			r.Use(otmiddleware.Middleware("fleet"))
		} else {
			apmgorilla.Instrument(r)
		}
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
	r.Walk(walkFn) //nolint:errcheck
}

// These are defined as const so that they can be used in tests.
const (
	desktopRateLimitMaxBurst        = 100 // Max burst used for device request rate limiting.
	forgotPasswordRateLimitMaxBurst = 9   // Max burst used for rate limiting on the the forgot_password endpoint.
)

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
	ue.EndingAtVersion("v1").GET("/api/_version_/fleet/global/policies", listGlobalPoliciesEndpoint, listGlobalPoliciesRequest{})
	ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/policies", listGlobalPoliciesEndpoint, listGlobalPoliciesRequest{})
	ue.GET("/api/_version_/fleet/policies/count", countGlobalPoliciesEndpoint, countGlobalPoliciesRequest{})
	ue.EndingAtVersion("v1").GET("/api/_version_/fleet/global/policies/{policy_id}", getPolicyByIDEndpoint, getPolicyByIDRequest{})
	ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/policies/{policy_id}", getPolicyByIDEndpoint, getPolicyByIDRequest{})
	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/global/policies/delete", deleteGlobalPoliciesEndpoint, deleteGlobalPoliciesRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/policies/delete", deleteGlobalPoliciesEndpoint, deleteGlobalPoliciesRequest{})
	ue.EndingAtVersion("v1").PATCH("/api/_version_/fleet/global/policies/{policy_id}", modifyGlobalPolicyEndpoint, modifyGlobalPolicyRequest{})
	ue.StartingAtVersion("2022-04").PATCH("/api/_version_/fleet/policies/{policy_id}", modifyGlobalPolicyEndpoint, modifyGlobalPolicyRequest{})
	ue.POST("/api/_version_/fleet/automations/reset", resetAutomationEndpoint, resetAutomationRequest{})

	// Alias /api/_version_/fleet/team/ -> /api/_version_/fleet/teams/
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies").
		POST("/api/_version_/fleet/teams/{team_id}/policies", teamPolicyEndpoint, teamPolicyRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies").
		GET("/api/_version_/fleet/teams/{team_id}/policies", listTeamPoliciesEndpoint, listTeamPoliciesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies/count").
		GET("/api/_version_/fleet/teams/{team_id}/policies/count", countTeamPoliciesEndpoint, countTeamPoliciesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies/{policy_id}").
		GET("/api/_version_/fleet/teams/{team_id}/policies/{policy_id}", getTeamPolicyByIDEndpoint, getTeamPolicyByIDRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{team_id}/policies/delete").
		POST("/api/_version_/fleet/teams/{team_id}/policies/delete", deleteTeamPoliciesEndpoint, deleteTeamPoliciesRequest{})
	ue.PATCH("/api/_version_/fleet/teams/{team_id}/policies/{policy_id}", modifyTeamPolicyEndpoint, modifyTeamPolicyRequest{})
	ue.POST("/api/_version_/fleet/spec/policies", applyPolicySpecsEndpoint, applyPolicySpecsRequest{})

	ue.GET("/api/_version_/fleet/queries/{id:[0-9]+}", getQueryEndpoint, getQueryRequest{})
	ue.GET("/api/_version_/fleet/queries", listQueriesEndpoint, listQueriesRequest{})
	ue.GET("/api/_version_/fleet/queries/{id:[0-9]+}/report", getQueryReportEndpoint, getQueryReportRequest{})
	ue.POST("/api/_version_/fleet/queries", createQueryEndpoint, createQueryRequest{})
	ue.PATCH("/api/_version_/fleet/queries/{id:[0-9]+}", modifyQueryEndpoint, modifyQueryRequest{})
	ue.DELETE("/api/_version_/fleet/queries/{name}", deleteQueryEndpoint, deleteQueryRequest{})
	ue.DELETE("/api/_version_/fleet/queries/id/{id:[0-9]+}", deleteQueryByIDEndpoint, deleteQueryByIDRequest{})
	ue.POST("/api/_version_/fleet/queries/delete", deleteQueriesEndpoint, deleteQueriesRequest{})
	ue.POST("/api/_version_/fleet/spec/queries", applyQuerySpecsEndpoint, applyQuerySpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/queries", getQuerySpecsEndpoint, getQuerySpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/queries/{name}", getQuerySpecEndpoint, getQuerySpecRequest{})

	ue.GET("/api/_version_/fleet/packs/{id:[0-9]+}", getPackEndpoint, getPackRequest{})
	ue.POST("/api/_version_/fleet/packs", createPackEndpoint, createPackRequest{})
	ue.PATCH("/api/_version_/fleet/packs/{id:[0-9]+}", modifyPackEndpoint, modifyPackRequest{})
	ue.GET("/api/_version_/fleet/packs", listPacksEndpoint, listPacksRequest{})
	ue.DELETE("/api/_version_/fleet/packs/{name}", deletePackEndpoint, deletePackRequest{})
	ue.DELETE("/api/_version_/fleet/packs/id/{id:[0-9]+}", deletePackByIDEndpoint, deletePackByIDRequest{})
	ue.POST("/api/_version_/fleet/spec/packs", applyPackSpecsEndpoint, applyPackSpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/packs", getPackSpecsEndpoint, nil)
	ue.GET("/api/_version_/fleet/spec/packs/{name}", getPackSpecEndpoint, getGenericSpecRequest{})

	ue.GET("/api/_version_/fleet/software/versions", listSoftwareVersionsEndpoint, listSoftwareRequest{})
	ue.GET("/api/_version_/fleet/software/versions/{id:[0-9]+}", getSoftwareEndpoint, getSoftwareRequest{})

	// DEPRECATED: use /api/_version_/fleet/software/versions instead
	ue.GET("/api/_version_/fleet/software", listSoftwareEndpoint, listSoftwareRequest{})
	// DEPRECATED: use /api/_version_/fleet/software/versions{id:[0-9]+} instead
	ue.GET("/api/_version_/fleet/software/{id:[0-9]+}", getSoftwareEndpoint, getSoftwareRequest{})
	// DEPRECATED: software version counts are now included directly in the software version list
	ue.GET("/api/_version_/fleet/software/count", countSoftwareEndpoint, countSoftwareRequest{})

	ue.GET("/api/_version_/fleet/software/titles", listSoftwareTitlesEndpoint, listSoftwareTitlesRequest{})
	ue.GET("/api/_version_/fleet/software/titles/{id:[0-9]+}", getSoftwareTitleEndpoint, getSoftwareTitleRequest{})
	ue.POST("/api/_version_/fleet/hosts/{host_id:[0-9]+}/software/install/{software_title_id:[0-9]+}", installSoftwareTitleEndpoint, installSoftwareRequest{})

	// Sofware installers
	ue.GET("/api/_version_/fleet/software/{title_id:[0-9]+}/package", getSoftwareInstallerEndpoint, getSoftwareInstallerRequest{})
	ue.POST("/api/_version_/fleet/software/package", uploadSoftwareInstallerEndpoint, uploadSoftwareInstallerRequest{})
	ue.DELETE("/api/_version_/fleet/software/{title_id:[0-9]+}/package", deleteSoftwareInstallerEndpoint, deleteSoftwareInstallerRequest{})
	ue.GET("/api/_version_/fleet/software/install/results/{install_uuid}", getSoftwareInstallResultsEndpoint, getSoftwareInstallResultsRequest{})
	ue.POST("/api/_version_/fleet/software/batch", batchSetSoftwareInstallersEndpoint, batchSetSoftwareInstallersRequest{})

	// Vulnerabilities
	ue.GET("/api/_version_/fleet/vulnerabilities", listVulnerabilitiesEndpoint, listVulnerabilitiesRequest{})
	ue.GET("/api/_version_/fleet/vulnerabilities/{cve}", getVulnerabilityEndpoint, getVulnerabilityRequest{})

	// Hosts
	ue.GET("/api/_version_/fleet/host_summary", getHostSummaryEndpoint, getHostSummaryRequest{})
	ue.GET("/api/_version_/fleet/hosts", listHostsEndpoint, listHostsRequest{})
	ue.POST("/api/_version_/fleet/hosts/delete", deleteHostsEndpoint, deleteHostsRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}", getHostEndpoint, getHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/count", countHostsEndpoint, countHostsRequest{})
	ue.POST("/api/_version_/fleet/hosts/search", searchHostsEndpoint, searchHostsRequest{})
	ue.GET("/api/_version_/fleet/hosts/identifier/{identifier}", hostByIdentifierEndpoint, hostByIdentifierRequest{})
	ue.POST("/api/_version_/fleet/hosts/identifier/{identifier}/query", runLiveQueryOnHostEndpoint, runLiveQueryOnHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/query", runLiveQueryOnHostByIDEndpoint, runLiveQueryOnHostByIDRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}", deleteHostEndpoint, deleteHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/transfer", addHostsToTeamEndpoint, addHostsToTeamRequest{})
	ue.POST("/api/_version_/fleet/hosts/transfer/filter", addHostsToTeamByFilterEndpoint, addHostsToTeamByFilterRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/refetch", refetchHostEndpoint, refetchHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/device_mapping", listHostDeviceMappingEndpoint, listHostDeviceMappingRequest{})
	ue.PUT("/api/_version_/fleet/hosts/{id:[0-9]+}/device_mapping", putHostDeviceMappingEndpoint, putHostDeviceMappingRequest{})
	ue.GET("/api/_version_/fleet/hosts/report", hostsReportEndpoint, hostsReportRequest{})
	ue.GET("/api/_version_/fleet/os_versions", osVersionsEndpoint, osVersionsRequest{})
	ue.GET("/api/_version_/fleet/os_versions/{id:[0-9]+}", getOSVersionEndpoint, getOSVersionRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/queries/{query_id:[0-9]+}", getHostQueryReportEndpoint, getHostQueryReportRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/health", getHostHealthEndpoint, getHostHealthRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/labels", addLabelsToHostEndpoint, addLabelsToHostRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}/labels", removeLabelsFromHostEndpoint, removeLabelsFromHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/software", getHostSoftwareEndpoint, getHostSoftwareRequest{})

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

	// This endpoint runs live queries synchronously (with a configured timeout).
	ue.POST("/api/_version_/fleet/queries/{id:[0-9]+}/run", runOneLiveQueryEndpoint, runOneLiveQueryRequest{})
	// Old endpoint, removed from docs. This GET endpoint runs live queries synchronously (with a configured timeout).
	ue.GET("/api/_version_/fleet/queries/run", runLiveQueryEndpoint, runLiveQueryRequest{})
	// The following two POST APIs are the asynchronous way to run live queries.
	// The live queries are created with these two endpoints and their results can be queried via
	// websockets via the `GET /api/_version_/fleet/results/` endpoint.
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

	ue.POST("/api/_version_/fleet/scripts/run", runScriptEndpoint, runScriptRequest{})
	ue.POST("/api/_version_/fleet/scripts/run/sync", runScriptSyncEndpoint, runScriptSyncRequest{})
	ue.GET("/api/_version_/fleet/scripts/results/{execution_id}", getScriptResultEndpoint, getScriptResultRequest{})
	ue.POST("/api/_version_/fleet/scripts", createScriptEndpoint, createScriptRequest{})
	ue.GET("/api/_version_/fleet/scripts", listScriptsEndpoint, listScriptsRequest{})
	ue.GET("/api/_version_/fleet/scripts/{script_id:[0-9]+}", getScriptEndpoint, getScriptRequest{})
	ue.DELETE("/api/_version_/fleet/scripts/{script_id:[0-9]+}", deleteScriptEndpoint, deleteScriptRequest{})
	ue.POST("/api/_version_/fleet/scripts/batch", batchSetScriptsEndpoint, batchSetScriptsRequest{})

	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/scripts", getHostScriptDetailsEndpoint, getHostScriptDetailsRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/activities/upcoming", listHostUpcomingActivitiesEndpoint, listHostUpcomingActivitiesRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/activities", listHostPastActivitiesEndpoint, listHostPastActivitiesRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/lock", lockHostEndpoint, lockHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/unlock", unlockHostEndpoint, unlockHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/wipe", wipeHostEndpoint, wipeHostRequest{})

	// Generative AI
	ue.POST("/api/_version_/fleet/autofill/policy", autofillPoliciesEndpoint, autofillPoliciesRequest{})

	// Only Fleet MDM specific endpoints should be within the root /mdm/ path.
	// NOTE: remember to update
	// `service.mdmConfigurationRequiredEndpoints` when you add an
	// endpoint that's behind the mdmConfiguredMiddleware, this applies
	// both to this set of endpoints and to any public/token-authenticated
	// endpoints using `neMDM` below in this file.
	mdmConfiguredMiddleware := mdmconfigured.NewMDMConfigMiddleware(svc)
	mdmAppleMW := ue.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyAppleMDM())

	// Deprecated: POST /mdm/apple/enqueue is now deprecated, replaced by the
	// platform-agnostic POST /mdm/commands/run. It is still supported
	// indefinitely for backwards compatibility.
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/enqueue", enqueueMDMAppleCommandEndpoint, enqueueMDMAppleCommandRequest{})
	// Deprecated: POST /mdm/apple/commandresults is now deprecated, replaced by the
	// platform-agnostic POST /mdm/commands/commandresults. It is still supported
	// indefinitely for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/commandresults", getMDMAppleCommandResultsEndpoint, getMDMAppleCommandResultsRequest{})
	// Deprecated: POST /mdm/apple/commands is now deprecated, replaced by the
	// platform-agnostic POST /mdm/commands/commands. It is still supported
	// indefinitely for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/commands", listMDMAppleCommandsEndpoint, listMDMAppleCommandsRequest{})
	// Deprecated: those /mdm/apple/profiles/... endpoints are now deprecated,
	// replaced by the platform-agnostic /mdm/profiles/... It is still supported
	// indefinitely for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/profiles/{profile_id:[0-9]+}", getMDMAppleConfigProfileEndpoint, getMDMAppleConfigProfileRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/profiles/{profile_id:[0-9]+}", deleteMDMAppleConfigProfileEndpoint, deleteMDMAppleConfigProfileRequest{})
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/profiles", newMDMAppleConfigProfileEndpoint, newMDMAppleConfigProfileRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesEndpoint, listMDMAppleConfigProfilesRequest{})

	// Deprecated: GET /mdm/apple/filevault/summary is now deprecated, replaced by the
	// platform-agnostic GET /mdm/disk_encryption/summary. It is still supported indefinitely
	// for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/filevault/summary", getMdmAppleFileVaultSummaryEndpoint, getMDMAppleFileVaultSummaryRequest{})

	// Deprecated: GET /mdm/apple/profiles/summary is now deprecated, replaced by the
	// platform-agnostic GET /mdm/profiles/summary. It is still supported indefinitely
	// for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/profiles/summary", getMDMAppleProfilesSummaryEndpoint, getMDMAppleProfilesSummaryRequest{})

	// Deprecated: POST /mdm/apple/enrollment_profile is now deprecated, replaced by the
	// POST /enrollment_profiles/automatic endpoint.
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantEndpoint, createMDMAppleSetupAssistantRequest{})
	mdmAppleMW.POST("/api/_version_/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantEndpoint, createMDMAppleSetupAssistantRequest{})

	// Deprecated: GET /mdm/apple/enrollment_profile is now deprecated, replaced by the
	// GET /enrollment_profiles/automatic endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/enrollment_profile", getMDMAppleSetupAssistantEndpoint, getMDMAppleSetupAssistantRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/enrollment_profiles/automatic", getMDMAppleSetupAssistantEndpoint, getMDMAppleSetupAssistantRequest{})

	// Deprecated: DELETE /mdm/apple/enrollment_profile is now deprecated, replaced by the
	// DELETE /enrollment_profiles/automatic endpoint.
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/enrollment_profile", deleteMDMAppleSetupAssistantEndpoint, deleteMDMAppleSetupAssistantRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/enrollment_profiles/automatic", deleteMDMAppleSetupAssistantEndpoint, deleteMDMAppleSetupAssistantRequest{})

	// TODO: are those undocumented endpoints still needed? I think they were only used
	// by 'fleetctl apple-mdm' sub-commands.
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/installers", uploadAppleInstallerEndpoint, uploadAppleInstallerRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/installers/{installer_id:[0-9]+}", getAppleInstallerEndpoint, getAppleInstallerDetailsRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/installers/{installer_id:[0-9]+}", deleteAppleInstallerEndpoint, deleteAppleInstallerDetailsRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/installers", listMDMAppleInstallersEndpoint, listMDMAppleInstallersRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/devices", listMDMAppleDevicesEndpoint, listMDMAppleDevicesRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/dep/devices", listMDMAppleDEPDevicesEndpoint, listMDMAppleDEPDevicesRequest{})

	// Deprecated: GET /mdm/manual_enrollment_profile is now deprecated, replaced by the
	// GET /enrollment_profiles/manual endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/manual_enrollment_profile", getManualEnrollmentProfileEndpoint, getManualEnrollmentProfileRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/enrollment_profiles/manual", getManualEnrollmentProfileEndpoint, getManualEnrollmentProfileRequest{})

	// bootstrap-package routes

	// Deprecated: POST /mdm/bootstrap is now deprecated, replaced by the
	// POST /bootstrap endpoint.
	mdmAppleMW.POST("/api/_version_/fleet/mdm/bootstrap", uploadBootstrapPackageEndpoint, uploadBootstrapPackageRequest{})
	mdmAppleMW.POST("/api/_version_/fleet/bootstrap", uploadBootstrapPackageEndpoint, uploadBootstrapPackageRequest{})

	// Deprecated: GET /mdm/bootstrap/:team_id/metadata is now deprecated, replaced by the
	// GET /bootstrap/:team_id/metadata endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/bootstrap/{team_id:[0-9]+}/metadata", bootstrapPackageMetadataEndpoint, bootstrapPackageMetadataRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/bootstrap/{team_id:[0-9]+}/metadata", bootstrapPackageMetadataEndpoint, bootstrapPackageMetadataRequest{})

	// Deprecated: DELETE /mdm/bootstrap/:team_id is now deprecated, replaced by the
	// DELETE /bootstrap/:team_id endpoint.
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/bootstrap/{team_id:[0-9]+}", deleteBootstrapPackageEndpoint, deleteBootstrapPackageRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/bootstrap/{team_id:[0-9]+}", deleteBootstrapPackageEndpoint, deleteBootstrapPackageRequest{})

	// Deprecated: GET /mdm/bootstrap/summary is now deprecated, replaced by the
	// GET /bootstrap/summary endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/bootstrap/summary", getMDMAppleBootstrapPackageSummaryEndpoint, getMDMAppleBootstrapPackageSummaryRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/bootstrap/summary", getMDMAppleBootstrapPackageSummaryEndpoint, getMDMAppleBootstrapPackageSummaryRequest{})

	// Deprecated: POST /mdm/apple/bootstrap is now deprecated, replaced by the platform agnostic /mdm/bootstrap
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/bootstrap", uploadBootstrapPackageEndpoint, uploadBootstrapPackageRequest{})
	// Deprecated: GET /mdm/apple/bootstrap/:team_id/metadata is now deprecated, replaced by the platform agnostic /mdm/bootstrap/:team_id/metadata
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/bootstrap/{team_id:[0-9]+}/metadata", bootstrapPackageMetadataEndpoint, bootstrapPackageMetadataRequest{})
	// Deprecated: DELETE /mdm/apple/bootstrap/:team_id is now deprecated, replaced by the platform agnostic /mdm/bootstrap/:team_id
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/bootstrap/{team_id:[0-9]+}", deleteBootstrapPackageEndpoint, deleteBootstrapPackageRequest{})
	// Deprecated: GET /mdm/apple/bootstrap/summary is now deprecated, replaced by the platform agnostic /mdm/bootstrap/summary
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/bootstrap/summary", getMDMAppleBootstrapPackageSummaryEndpoint, getMDMAppleBootstrapPackageSummaryRequest{})

	// host-specific mdm routes

	// Deprecated: PATCH /mdm/hosts/:id/unenroll is now deprecated, replaced by
	// DELETE /hosts/:id/mdm.
	mdmAppleMW.PATCH("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/unenroll", mdmAppleCommandRemoveEnrollmentProfileEndpoint, mdmAppleCommandRemoveEnrollmentProfileRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}/mdm", mdmAppleCommandRemoveEnrollmentProfileEndpoint, mdmAppleCommandRemoveEnrollmentProfileRequest{})

	mdmAppleMW.POST("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/lock", deviceLockEndpoint, deviceLockRequest{})
	mdmAppleMW.POST("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/wipe", deviceWipeEndpoint, deviceWipeRequest{})

	// Deprecated: GET /mdm/hosts/:id/profiles is now deprecated, replaced by
	// GET /hosts/:id/configuration_profiles.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/profiles", getHostProfilesEndpoint, getHostProfilesRequest{})
	// TODO: Confirm if response should be updated to include Windows profiles and use mdmAnyMW
	mdmAppleMW.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/configuration_profiles", getHostProfilesEndpoint, getHostProfilesRequest{})

	// Deprecated: PATCH /mdm/apple/setup is now deprecated, replaced by the
	// PATCH /setup_experience endpoint.
	mdmAppleMW.PATCH("/api/_version_/fleet/mdm/apple/setup", updateMDMAppleSetupEndpoint, updateMDMAppleSetupRequest{})
	mdmAppleMW.PATCH("/api/_version_/fleet/setup_experience", updateMDMAppleSetupEndpoint, updateMDMAppleSetupRequest{})

	// Deprecated: GET /mdm/apple is now deprecated, replaced by the
	// GET /apns endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple", getAppleMDMEndpoint, nil)
	mdmAppleMW.GET("/api/_version_/fleet/apns", getAppleMDMEndpoint, nil)

	// EULA routes

	// Deprecated: POST /mdm/setup/eula is now deprecated, replaced by the
	// POST /setup_experience/eula endpoint.
	mdmAppleMW.POST("/api/_version_/fleet/mdm/setup/eula", createMDMEULAEndpoint, createMDMEULARequest{})
	mdmAppleMW.POST("/api/_version_/fleet/setup_experience/eula", createMDMEULAEndpoint, createMDMEULARequest{})

	// Deprecated: GET /mdm/setup/eula/metadata is now deprecated, replaced by the
	// GET /setup_experience/eula/metadata endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/setup/eula/metadata", getMDMEULAMetadataEndpoint, getMDMEULAMetadataRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/setup_experience/eula/metadata", getMDMEULAMetadataEndpoint, getMDMEULAMetadataRequest{})

	// Deprecated: DELETE /mdm/setup/eula/:token is now deprecated, replaced by the
	// DELETE /setup_experience/eula/:token endpoint.
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/setup/eula/{token}", deleteMDMEULAEndpoint, deleteMDMEULARequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/setup_experience/eula/{token}", deleteMDMEULAEndpoint, deleteMDMEULARequest{})

	// Deprecated: POST /mdm/apple/setup/eula is now deprecated, replaced by the platform agnostic /mdm/setup/eula
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/setup/eula", createMDMEULAEndpoint, createMDMEULARequest{})
	// Deprecated: GET /mdm/apple/setup/eula/metadata is now deprecated, replaced by the platform agnostic /mdm/setup/eula/metadata
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/setup/eula/metadata", getMDMEULAMetadataEndpoint, getMDMEULAMetadataRequest{})
	// Deprecated: DELETE /mdm/apple/setup/eula/:token is now deprecated, replaced by the platform agnostic /mdm/setup/eula/:token
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/setup/eula/{token}", deleteMDMEULAEndpoint, deleteMDMEULARequest{})

	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileEndpoint, preassignMDMAppleProfileRequest{})
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentEndpoint, matchMDMApplePreassignmentRequest{})

	mdmAnyMW := ue.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyAppleOrWindowsMDM())

	// Deprecated: POST /mdm/commands/run is now deprecated, replaced by the
	// POST /commands/run endpoint.
	mdmAnyMW.POST("/api/_version_/fleet/mdm/commands/run", runMDMCommandEndpoint, runMDMCommandRequest{})
	mdmAnyMW.POST("/api/_version_/fleet/commands/run", runMDMCommandEndpoint, runMDMCommandRequest{})

	// Deprecated: GET /mdm/commandresults is now deprecated, replaced by the
	// GET /commands/results endpoint.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/commandresults", getMDMCommandResultsEndpoint, getMDMCommandResultsRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/commands/results", getMDMCommandResultsEndpoint, getMDMCommandResultsRequest{})

	// Deprecated: GET /mdm/commands is now deprecated, replaced by the
	// GET /commands endpoint.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/commands", listMDMCommandsEndpoint, listMDMCommandsRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/commands", listMDMCommandsEndpoint, listMDMCommandsRequest{})

	// Deprecated: GET /mdm/disk_encryption/summary is now deprecated, replaced by the
	// GET /disk_encryption endpoint.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/disk_encryption/summary", getMDMDiskEncryptionSummaryEndpoint, getMDMDiskEncryptionSummaryRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/disk_encryption", getMDMDiskEncryptionSummaryEndpoint, getMDMDiskEncryptionSummaryRequest{})

	// Deprecated: GET /mdm/hosts/:id/encryption_key is now deprecated, replaced by
	// GET /hosts/:id/encryption_key.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/encryption_key", getHostEncryptionKey, getHostEncryptionKeyRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/encryption_key", getHostEncryptionKey, getHostEncryptionKeyRequest{})

	// Deprecated: GET /mdm/profiles/summary is now deprecated, replaced by the
	// GET /configuration_profiles/summary endpoint.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/profiles/summary", getMDMProfilesSummaryEndpoint, getMDMProfilesSummaryRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/configuration_profiles/summary", getMDMProfilesSummaryEndpoint, getMDMProfilesSummaryRequest{})

	// Deprecated: GET /mdm/profiles/:profile_uuid is now deprecated, replaced by
	// GET /configuration_profiles/:profile_uuid.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/profiles/{profile_uuid}", getMDMConfigProfileEndpoint, getMDMConfigProfileRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/configuration_profiles/{profile_uuid}", getMDMConfigProfileEndpoint, getMDMConfigProfileRequest{})

	// Deprecated: DELETE /mdm/profiles/:profile_uuid is now deprecated, replaced by
	// DELETE /configuration_profiles/:profile_uuid.
	mdmAnyMW.DELETE("/api/_version_/fleet/mdm/profiles/{profile_uuid}", deleteMDMConfigProfileEndpoint, deleteMDMConfigProfileRequest{})
	mdmAnyMW.DELETE("/api/_version_/fleet/configuration_profiles/{profile_uuid}", deleteMDMConfigProfileEndpoint, deleteMDMConfigProfileRequest{})

	// Deprecated: GET /mdm/profiles is now deprecated, replaced by the
	// GET /configuration_profiles endpoint.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/profiles", listMDMConfigProfilesEndpoint, listMDMConfigProfilesRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/configuration_profiles", listMDMConfigProfilesEndpoint, listMDMConfigProfilesRequest{})

	// Deprecated: POST /mdm/profiles is now deprecated, replaced by the
	// POST /configuration_profiles endpoint.
	mdmAnyMW.POST("/api/_version_/fleet/mdm/profiles", newMDMConfigProfileEndpoint, newMDMConfigProfileRequest{})
	mdmAnyMW.POST("/api/_version_/fleet/configuration_profiles", newMDMConfigProfileEndpoint, newMDMConfigProfileRequest{})

	mdmAnyMW.POST("/api/_version_/fleet/hosts/{host_id:[0-9]+}/configuration_profiles/resend/{profile_uuid}", resendHostMDMProfileEndpoint, resendHostMDMProfileRequest{})

	// Deprecated: PATCH /mdm/apple/settings is deprecated, replaced by POST /disk_encryption.
	// It was only used to set disk encryption.
	mdmAnyMW.PATCH("/api/_version_/fleet/mdm/apple/settings", updateMDMAppleSettingsEndpoint, updateMDMAppleSettingsRequest{})
	mdmAnyMW.POST("/api/_version_/fleet/disk_encryption", updateMDMDiskEncryptionEndpoint, updateMDMDiskEncryptionRequest{})

	// the following set of mdm endpoints must always be accessible (even
	// if MDM is not configured) as it bootstraps the setup of MDM
	// (generates CSR request for APNs, plus the SCEP and ABM keypairs).
	ue.POST("/api/_version_/fleet/mdm/apple/request_csr", requestMDMAppleCSREndpoint, requestMDMAppleCSRRequest{})
	ue.POST("/api/_version_/fleet/mdm/apple/dep/key_pair", newMDMAppleDEPKeyPairEndpoint, nil)

	// Deprecated: GET /mdm/apple_bm is now deprecated, replaced by the
	// GET /abm endpoint.
	ue.GET("/api/_version_/fleet/mdm/apple_bm", getAppleBMEndpoint, nil)
	ue.GET("/api/_version_/fleet/abm", getAppleBMEndpoint, nil)

	// Deprecated: POST /mdm/apple/profiles/batch is now deprecated, replaced by the
	// platform-agnostic POST /mdm/profiles/batch. It is still supported
	// indefinitely for backwards compatibility.
	//
	// batch-apply is accessible even though MDM is not enabled, it needs
	// to support the case where `fleetctl get config`'s output is used as
	// input to `fleetctl apply`
	ue.POST("/api/_version_/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesEndpoint, batchSetMDMAppleProfilesRequest{})

	// batch-apply is accessible even though MDM is not enabled, it needs
	// to support the case where `fleetctl get config`'s output is used as
	// input to `fleetctl apply`
	ue.POST("/api/_version_/fleet/mdm/profiles/batch", batchSetMDMProfilesEndpoint, batchSetMDMProfilesRequest{})

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
		errorLimiter.Limit("ping_device_auth", desktopQuota),
	).HEAD("/api/_version_/fleet/device/{token}/ping", devicePingEndpoint, deviceAuthPingRequest{})
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
	de.WithCustomMiddleware(
		errorLimiter.Limit("send_device_error", desktopQuota),
	).POST("/api/_version_/fleet/device/{token}/debug/errors", fleetdError, fleetdErrorRequest{})
	de.WithCustomMiddleware(
		errorLimiter.Limit("get_device_software", desktopQuota),
	).GET("/api/_version_/fleet/device/{token}/software", getDeviceSoftwareEndpoint, getDeviceSoftwareRequest{})

	// mdm-related endpoints available via device authentication
	demdm := de.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyAppleMDM())
	demdm.WithCustomMiddleware(
		errorLimiter.Limit("get_device_mdm", desktopQuota),
	).GET("/api/_version_/fleet/device/{token}/mdm/apple/manual_enrollment_profile", getDeviceMDMManualEnrollProfileEndpoint, getDeviceMDMManualEnrollProfileRequest{})

	demdm.WithCustomMiddleware(
		errorLimiter.Limit("post_device_rotate_encryption_key", desktopQuota),
	).POST("/api/_version_/fleet/device/{token}/rotate_encryption_key", rotateEncryptionKeyEndpoint, rotateEncryptionKeyRequest{})

	demdm.WithCustomMiddleware(
		errorLimiter.Limit("post_device_migrate_mdm", desktopQuota),
	).POST("/api/_version_/fleet/device/{token}/migrate_mdm", migrateMDMDeviceEndpoint, deviceMigrateMDMRequest{})

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
	// using POST to get a script execution request since all authenticated orbit
	// endpoints are POST due to passing the device token in the JSON body.
	oe.POST("/api/fleet/orbit/scripts/request", getOrbitScriptEndpoint, orbitGetScriptRequest{})
	oe.POST("/api/fleet/orbit/scripts/result", postOrbitScriptResultEndpoint, orbitPostScriptResultRequest{})
	oe.PUT("/api/fleet/orbit/device_mapping", putOrbitDeviceMappingEndpoint, orbitPutDeviceMappingRequest{})
	oe.POST("/api/fleet/orbit/software_install/result", postOrbitSoftwareInstallResultEndpoint, orbitPostSoftwareInstallResultRequest{})

	oe.POST("/api/fleet/orbit/software_install/package", orbitDownloadSoftwareInstallerEndpoint, orbitDownloadSoftwareInstallerRequest{})

	oe.POST("/api/fleet/orbit/software_install/details", getOrbitSoftwareInstallDetails, orbitGetSoftwareInstallRequest{})

	oeWindowsMDM := oe.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyWindowsMDM())
	oeWindowsMDM.POST("/api/fleet/orbit/disk_encryption_key", postOrbitDiskEncryptionKeyEndpoint, orbitPostDiskEncryptionKeyRequest{})

	// unauthenticated endpoints - most of those are either login-related,
	// invite-related or host-enrolling. So they typically do some kind of
	// one-time authentication by verifying that a valid secret token is provided
	// with the request.
	ne := newNoAuthEndpointer(svc, opts, r, apiVersions...)
	ne.WithAltPaths("/api/v1/osquery/enroll").
		POST("/api/osquery/enroll", enrollAgentEndpoint, enrollAgentRequest{})

	// These endpoint are token authenticated.
	// NOTE: remember to update
	// `service.mdmConfigurationRequiredEndpoints` when you add an
	// endpoint that's behind the mdmConfiguredMiddleware, this applies
	// both to this set of endpoints and to any user authenticated
	// endpoints using `mdmAppleMW.*` above in this file.
	neAppleMDM := ne.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyAppleMDM())
	neAppleMDM.GET(apple_mdm.EnrollPath, mdmAppleEnrollEndpoint, mdmAppleEnrollRequest{})
	neAppleMDM.GET(apple_mdm.InstallerPath, mdmAppleGetInstallerEndpoint, mdmAppleGetInstallerRequest{})
	neAppleMDM.HEAD(apple_mdm.InstallerPath, mdmAppleHeadInstallerEndpoint, mdmAppleHeadInstallerRequest{})

	// Deprecated: GET /mdm/bootstrap is now deprecated, replaced by the
	// GET /bootstrap endpoint.
	neAppleMDM.GET("/api/_version_/fleet/mdm/bootstrap", downloadBootstrapPackageEndpoint, downloadBootstrapPackageRequest{})
	neAppleMDM.GET("/api/_version_/fleet/bootstrap", downloadBootstrapPackageEndpoint, downloadBootstrapPackageRequest{})

	// Deprecated: GET /mdm/apple/bootstrap is now deprecated, replaced by the platform agnostic /mdm/bootstrap
	neAppleMDM.GET("/api/_version_/fleet/mdm/apple/bootstrap", downloadBootstrapPackageEndpoint, downloadBootstrapPackageRequest{})

	// Deprecated: GET /mdm/setup/eula/:token is now deprecated, replaced by the
	// GET /setup_experience/eula/:token endpoint.
	neAppleMDM.GET("/api/_version_/fleet/mdm/setup/eula/{token}", getMDMEULAEndpoint, getMDMEULARequest{})
	neAppleMDM.GET("/api/_version_/fleet/setup_experience/eula/{token}", getMDMEULAEndpoint, getMDMEULARequest{})

	// Deprecated: GET /mdm/apple/setup/eula/:token is now deprecated, replaced by the platform agnostic /mdm/setup/eula/:token
	neAppleMDM.GET("/api/_version_/fleet/mdm/apple/setup/eula/{token}", getMDMEULAEndpoint, getMDMEULARequest{})

	// These endpoint are used by Microsoft devices during MDM device enrollment phase
	neWindowsMDM := ne.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyWindowsMDM())

	// Microsoft MS-MDE2 Endpoints
	// This endpoint is unauthenticated and is used by Microsoft devices to discover the MDM server endpoints
	neWindowsMDM.POST(microsoft_mdm.MDE2DiscoveryPath, mdmMicrosoftDiscoveryEndpoint, SoapRequestContainer{})

	// This endpoint is unauthenticated and is used by Microsoft devices to retrieve the opaque STS auth token
	neWindowsMDM.GET(microsoft_mdm.MDE2AuthPath, mdmMicrosoftAuthEndpoint, SoapRequestContainer{})

	// This endpoint is authenticated using the BinarySecurityToken header field
	neWindowsMDM.POST(microsoft_mdm.MDE2PolicyPath, mdmMicrosoftPolicyEndpoint, SoapRequestContainer{})

	// This endpoint is authenticated using the BinarySecurityToken header field
	neWindowsMDM.POST(microsoft_mdm.MDE2EnrollPath, mdmMicrosoftEnrollEndpoint, SoapRequestContainer{})

	// This endpoint is unauthenticated for now
	// It should be authenticated through TLS headers once proper implementation is in place
	neWindowsMDM.POST(microsoft_mdm.MDE2ManagementPath, mdmMicrosoftManagementEndpoint, SyncMLReqMsgContainer{})

	// This endpoint is unauthenticated and is used by to retrieve the MDM enrollment Terms of Use
	neWindowsMDM.GET(microsoft_mdm.MDE2TOSPath, mdmMicrosoftTOSEndpoint, MDMWebContainer{})

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
	ne.UsePathPrefix().PathHandler("GET", "/api/_version_/fleet/results/", makeStreamDistributedQueryCampaignResultsHandler(config.Server, svc, logger))

	quota := throttled.RateQuota{MaxRate: throttled.PerHour(10), MaxBurst: forgotPasswordRateLimitMaxBurst}
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

	ne.HEAD("/api/fleet/device/ping", devicePingEndpoint, devicePingRequest{})

	ne.HEAD("/api/fleet/orbit/ping", orbitPingEndpoint, orbitPingRequest{})

	neAppleMDM.WithCustomMiddleware(limiter.Limit("login", throttled.RateQuota{MaxRate: loginRateLimit, MaxBurst: 9})).
		POST("/api/_version_/fleet/mdm/sso", initiateMDMAppleSSOEndpoint, initiateMDMAppleSSORequest{})

	neAppleMDM.WithCustomMiddleware(limiter.Limit("login", throttled.RateQuota{MaxRate: loginRateLimit, MaxBurst: 9})).
		POST("/api/_version_/fleet/mdm/sso/callback", callbackMDMAppleSSOEndpoint, callbackMDMAppleSSORequest{})
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

// RegisterAppleMDMProtocolServices registers the HTTP handlers that serve
// the MDM services to Apple devices.
func RegisterAppleMDMProtocolServices(
	mux *http.ServeMux,
	scepConfig config.MDMConfig,
	mdmStorage nanomdm_storage.AllStorage,
	scepStorage scep_depot.Depot,
	logger kitlog.Logger,
	checkinAndCommandService nanomdm_service.CheckinAndCommandService,
	ddmService nanomdm_service.DeclarativeManagement,
) error {
	scepCACerts, scepCAKey, err := scepStorage.CA([]byte{})
	if err != nil {
		return fmt.Errorf("load SCEP CA certificates and key: %w", err)
	}
	if err := registerSCEP(mux, scepConfig, scepCACerts[0], scepCAKey, scepStorage, logger); err != nil {
		return fmt.Errorf("scep: %w", err)
	}
	if err := registerMDM(mux, scepCACerts[0], mdmStorage, checkinAndCommandService, ddmService, logger); err != nil {
		return fmt.Errorf("mdm: %w", err)
	}
	return nil
}

// registerSCEP registers the HTTP handler for SCEP service needed for enrollment to MDM.
// Returns the SCEP CA certificate that can be used by verifiers.
func registerSCEP(
	mux *http.ServeMux,
	scepConfig config.MDMConfig,
	scepCert *x509.Certificate,
	scepKey *rsa.PrivateKey,
	scepStorage scep_depot.Depot,
	logger kitlog.Logger,
) error {
	var signer scepserver.CSRSigner = scep_depot.NewSigner(
		scepStorage,
		scep_depot.WithValidityDays(scepConfig.AppleSCEPSignerValidityDays),
		scep_depot.WithAllowRenewalDays(scepConfig.AppleSCEPSignerAllowRenewalDays),
	)
	scepChallenge := scepConfig.AppleSCEPChallenge
	if scepChallenge == "" {
		return errors.New("missing SCEP challenge")
	}

	signer = scepserver.ChallengeMiddleware(scepChallenge, signer)
	scepService, err := scepserver.NewService(scepCert, scepKey, signer,
		scepserver.WithLogger(kitlog.With(logger, "component", "mdm-apple-scep")),
	)
	if err != nil {
		return fmt.Errorf("initialize SCEP service: %w", err)
	}
	scepLogger := kitlog.With(logger, "component", "http-mdm-apple-scep")
	e := scepserver.MakeServerEndpoints(scepService)
	e.GetEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.GetEndpoint)
	e.PostEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.PostEndpoint)
	scepHandler := scepserver.MakeHTTPHandler(e, scepService, scepLogger)
	mux.Handle(apple_mdm.SCEPPath, scepHandler)
	return nil
}

// NanoMDMLogger is a logger adapter for nanomdm.
type NanoMDMLogger struct {
	logger kitlog.Logger
}

func NewNanoMDMLogger(logger kitlog.Logger) *NanoMDMLogger {
	return &NanoMDMLogger{
		logger: logger,
	}
}

func (l *NanoMDMLogger) Info(keyvals ...interface{}) {
	level.Info(l.logger).Log(keyvals...)
}

func (l *NanoMDMLogger) Debug(keyvals ...interface{}) {
	level.Debug(l.logger).Log(keyvals...)
}

func (l *NanoMDMLogger) With(keyvals ...interface{}) nanomdm_log.Logger {
	newLogger := kitlog.With(l.logger, keyvals...)
	return &NanoMDMLogger{
		logger: newLogger,
	}
}

// registerMDM registers the HTTP handlers that serve core MDM services (like checking in for MDM commands).
func registerMDM(
	mux *http.ServeMux,
	scepCACert *x509.Certificate,
	mdmStorage nanomdm_storage.AllStorage,
	checkinAndCommandService nanomdm_service.CheckinAndCommandService,
	ddmService nanomdm_service.DeclarativeManagement,
	logger kitlog.Logger,
) error {
	certVerifier, err := certverify.NewPoolVerifier(
		apple_mdm.EncodeCertPEM(scepCACert),
		x509.ExtKeyUsageClientAuth,
	)
	if err != nil {
		return fmt.Errorf("certificate pool verifier: %w", err)
	}
	mdmLogger := NewNanoMDMLogger(kitlog.With(logger, "component", "http-mdm-apple-mdm"))

	// As usual, handlers are applied from bottom to top:
	// 1. Extract and verify MDM signature.
	// 2. Verify signer certificate with CA.
	// 3. Verify new or enrolled certificate (certauth.CertAuth which wraps the MDM service).
	// 4. Pass a copy of the request to Fleet middleware that ingests new hosts from pending MDM
	// enrollments and updates the Fleet hosts table accordingly with the UDID and serial number of
	// the device.
	// 5. Run actual MDM service operation (checkin handler or command and results handler).
	coreMDMService := nanomdm.New(mdmStorage, nanomdm.WithLogger(mdmLogger), nanomdm.WithDeclarativeManagement(ddmService))
	// NOTE: it is critical that the coreMDMService runs first, as the first
	// service in the multi-service feature is run to completion _before_ running
	// the other ones in parallel. This way, subsequent services have access to
	// the result of the core service, e.g. the device is enrolled, etc.
	var mdmService nanomdm_service.CheckinAndCommandService = multi.New(mdmLogger, coreMDMService, checkinAndCommandService)

	mdmService = certauth.New(mdmService, mdmStorage)
	var mdmHandler http.Handler = httpmdm.CheckinAndCommandHandler(mdmService, mdmLogger.With("handler", "checkin-command"))
	mdmHandler = httpmdm.CertVerifyMiddleware(mdmHandler, certVerifier, mdmLogger.With("handler", "cert-verify"))
	mdmHandler = httpmdm.CertExtractMdmSignatureMiddleware(mdmHandler, mdmLogger.With("handler", "cert-extract"))
	mux.Handle(apple_mdm.MDMPath, mdmHandler)
	return nil
}
