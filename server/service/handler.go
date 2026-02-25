package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/server/config"
	carvestorectx "github.com/fleetdm/fleet/v4/server/contexts/carvestore"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	mdmcrypto "github.com/fleetdm/fleet/v4/server/mdm/crypto"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	httpmdm "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http/mdm"
	nanomdm_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/multi"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/nanomdm"
	scep_depot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/fleetdm/fleet/v4/server/platform/middleware/ratelimit"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/log"
	"github.com/fleetdm/fleet/v4/server/service/middleware/mdmconfigured"
	"github.com/fleetdm/fleet/v4/server/service/middleware/otel"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/klauspost/compress/gzhttp"
	nanomdm_log "github.com/micromdm/nanolib/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/throttled/throttled/v2"
	"go.elastic.co/apm/module/apmgorilla/v2"
	otmiddleware "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

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
	loginRateLimit  *throttled.Rate
	mdmSsoRateLimit *throttled.Rate
	httpSigVerifier mux.MiddlewareFunc
}

// ExtraHandlerOption allows adding extra configuration to the HTTP handler.
type ExtraHandlerOption func(*extraHandlerOpts)

// WithLoginRateLimit configures the rate limit for the login endpoints.
func WithLoginRateLimit(r throttled.Rate) ExtraHandlerOption {
	return func(o *extraHandlerOpts) {
		o.loginRateLimit = &r
	}
}

// WithMdmSsoRateLimit configures the rate limit for the MDM SSO endpoints (falls back to login rate limit otherwise).
func WithMdmSsoRateLimit(r throttled.Rate) ExtraHandlerOption {
	return func(o *extraHandlerOpts) {
		o.mdmSsoRateLimit = &r
	}
}

func WithHTTPSigVerifier(m mux.MiddlewareFunc) ExtraHandlerOption {
	return func(o *extraHandlerOpts) {
		o.httpSigVerifier = m
	}
}

func setCarveStoreInRequestContext(carveStore fleet.CarveStore) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		ctx = carvestorectx.NewContext(ctx, carveStore)
		return ctx
	}
}

// MakeHandler creates an HTTP handler for the Fleet server endpoints.
func MakeHandler(
	svc fleet.Service,
	config config.FleetConfig,
	logger *logging.Logger,
	limitStore throttled.GCRAStore,
	redisPool fleet.RedisPool,
	carveStore fleet.CarveStore,
	featureRoutes []endpointer.HandlerRoutesFunc,
	extra ...ExtraHandlerOption,
) http.Handler {
	var eopts extraHandlerOpts
	for _, fn := range extra {
		fn(&eopts)
	}

	// Create the client IP extraction strategy based on config.
	ipStrategy, err := endpointer.NewClientIPStrategy(config.Server.TrustedProxies)
	if err != nil {
		panic(fmt.Sprintf("invalid server.trusted_proxies configuration: %v", err))
	}

	fleetAPIOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(
			kithttp.PopulateRequestContext, // populate the request context with common fields
			auth.SetRequestsContexts(svc),
			setCarveStoreInRequestContext(carveStore),
		),
		kithttp.ServerErrorHandler(&endpointer.ErrorHandler{Logger: logger.SlogLogger()}),
		kithttp.ServerErrorEncoder(fleetErrorEncoder),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
			log.LogRequestEnd(logger.SlogLogger()),
			checkLicenseExpiration(svc),
		),
	}

	r := mux.NewRouter()
	if config.Logging.TracingEnabled {
		if config.OTELEnabled() {
			r.Use(otmiddleware.Middleware(
				"service",
				otmiddleware.WithSpanNameFormatter(func(route string, r *http.Request) string {
					// Use the guideline for span names: {method} {target}
					// See https://opentelemetry.io/docs/specs/semconv/http/http-spans/
					return r.Method + " " + route
				})))
		} else {
			apmgorilla.Instrument(r)
		}
	}

	if config.Server.GzipResponses {
		r.Use(func(h http.Handler) http.Handler {
			return gzhttp.GzipHandler(h)
		})
	}

	// Add middleware to extract the client IP and set it in the request context.
	r.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ipStrategy.ClientIP(r.Header, r.RemoteAddr)
			if ip != "" {
				r.RemoteAddr = ip
			}
			handler.ServeHTTP(w, r.WithContext(publicip.NewContext(r.Context(), ip)))
		})
	})

	if eopts.httpSigVerifier != nil {
		r.Use(eopts.httpSigVerifier)
	}

	attachFleetAPIRoutes(r, svc, config, logger, limitStore, redisPool, fleetAPIOptions, eopts)
	for _, featureRoute := range featureRoutes {
		featureRoute(r, fleetAPIOptions)
	}
	addMetrics(r)

	return r
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
	forgotPasswordRateLimitMaxBurst = 9 // Max burst used for rate limiting on the the forgot_password endpoint.

	// Fleet Desktop API endpoints rate limiting:
	//
	// Allow up to 1_000 consecutive failing requests per minute.
	// If the threshold of 1_000 consecutive failures is reached for an IP,
	// ban requests from such IP for a duration of 1 minute.
	//

	deviceIPAllowedConsecutiveFailingRequestsCount      = 1_000
	deviceIPAllowedConsecutiveFailingRequestsTimeWindow = 1 * time.Minute
	deviceIPBanTime                                     = 1 * time.Minute
)

func attachFleetAPIRoutes(r *mux.Router, svc fleet.Service, config config.FleetConfig,
	logger *logging.Logger, limitStore throttled.GCRAStore, redisPool fleet.RedisPool, opts []kithttp.ServerOption,
	extra extraHandlerOpts,
) {
	apiVersions := []string{"v1", "2022-04"}

	// user-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(svc, opts, r, apiVersions...)

	ue.POST("/api/_version_/fleet/trigger", triggerEndpoint, fleet.TriggerRequest{})

	ue.GET("/api/_version_/fleet/me", meEndpoint, fleet.GetMeRequest{})
	ue.GET("/api/_version_/fleet/sessions/{id:[0-9]+}", getInfoAboutSessionEndpoint, fleet.GetInfoAboutSessionRequest{})
	ue.DELETE("/api/_version_/fleet/sessions/{id:[0-9]+}", deleteSessionEndpoint, fleet.DeleteSessionRequest{})

	ue.GET("/api/_version_/fleet/config/certificate", getCertificateEndpoint, nil)
	ue.GET("/api/_version_/fleet/config", getAppConfigEndpoint, nil)
	ue.PATCH("/api/_version_/fleet/config", modifyAppConfigEndpoint, fleet.ModifyAppConfigRequest{})
	ue.POST("/api/_version_/fleet/spec/enroll_secret", applyEnrollSecretSpecEndpoint, fleet.ApplyEnrollSecretSpecRequest{})
	ue.GET("/api/_version_/fleet/spec/enroll_secret", getEnrollSecretSpecEndpoint, nil)
	ue.GET("/api/_version_/fleet/version", versionEndpoint, nil)

	ue.POST("/api/_version_/fleet/users/roles/spec", applyUserRoleSpecsEndpoint, fleet.ApplyUserRoleSpecsRequest{})
	ue.POST("/api/_version_/fleet/translate", translatorEndpoint, fleet.TranslatorRequest{})
	ue.WithAltPaths("/api/_version_/fleet/spec/teams").WithRequestBodySizeLimit(5*units.MiB).POST("/api/_version_/fleet/spec/fleets", applyTeamSpecsEndpoint, fleet.ApplyTeamSpecsRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{fleet_id:[0-9]+}/secrets").PATCH("/api/_version_/fleet/fleets/{fleet_id:[0-9]+}/secrets", modifyTeamEnrollSecretsEndpoint, fleet.ModifyTeamEnrollSecretsRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams").POST("/api/_version_/fleet/fleets", createTeamEndpoint, fleet.CreateTeamRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams").GET("/api/_version_/fleet/fleets", listTeamsEndpoint, fleet.ListTeamsRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{id:[0-9]+}").GET("/api/_version_/fleet/fleets/{id:[0-9]+}", getTeamEndpoint, fleet.GetTeamRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{id:[0-9]+}").PATCH("/api/_version_/fleet/fleets/{id:[0-9]+}", modifyTeamEndpoint, fleet.ModifyTeamRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{id:[0-9]+}").DELETE("/api/_version_/fleet/fleets/{id:[0-9]+}", deleteTeamEndpoint, fleet.DeleteTeamRequest{})
	ue.WithRequestBodySizeLimit(2*units.MiB).WithAltPaths("/api/_version_/fleet/teams/{id:[0-9]+}/agent_options").POST("/api/_version_/fleet/fleets/{id:[0-9]+}/agent_options", modifyTeamAgentOptionsEndpoint, fleet.ModifyTeamAgentOptionsRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{id:[0-9]+}/users").GET("/api/_version_/fleet/fleets/{id:[0-9]+}/users", listTeamUsersEndpoint, fleet.ListTeamUsersRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{id:[0-9]+}/users").PATCH("/api/_version_/fleet/fleets/{id:[0-9]+}/users", addTeamUsersEndpoint, fleet.ModifyTeamUsersRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{id:[0-9]+}/users").DELETE("/api/_version_/fleet/fleets/{id:[0-9]+}/users", deleteTeamUsersEndpoint, fleet.ModifyTeamUsersRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{id:[0-9]+}/secrets").GET("/api/_version_/fleet/fleets/{id:[0-9]+}/secrets", teamEnrollSecretsEndpoint, fleet.TeamEnrollSecretsRequest{})

	ue.GET("/api/_version_/fleet/users", listUsersEndpoint, fleet.ListUsersRequest{})
	ue.POST("/api/_version_/fleet/users/admin", createUserEndpoint, fleet.CreateUserRequest{})
	ue.GET("/api/_version_/fleet/users/{id:[0-9]+}", getUserEndpoint, fleet.GetUserRequest{})
	ue.PATCH("/api/_version_/fleet/users/{id:[0-9]+}", modifyUserEndpoint, fleet.ModifyUserRequest{})
	ue.DELETE("/api/_version_/fleet/users/{id:[0-9]+}", deleteUserEndpoint, fleet.DeleteUserRequest{})
	ue.POST("/api/_version_/fleet/users/{id:[0-9]+}/require_password_reset", requirePasswordResetEndpoint, fleet.RequirePasswordResetRequest{})
	ue.GET("/api/_version_/fleet/users/{id:[0-9]+}/sessions", getInfoAboutSessionsForUserEndpoint, fleet.GetInfoAboutSessionsForUserRequest{})
	ue.DELETE("/api/_version_/fleet/users/{id:[0-9]+}/sessions", deleteSessionsForUserEndpoint, fleet.DeleteSessionsForUserRequest{})
	ue.POST("/api/_version_/fleet/change_password", changePasswordEndpoint, fleet.ChangePasswordRequest{})

	ue.GET("/api/_version_/fleet/email/change/{token}", changeEmailEndpoint, fleet.ChangeEmailRequest{})
	// TODO: searchTargetsEndpoint will be removed in Fleet 5.0
	ue.POST("/api/_version_/fleet/targets", searchTargetsEndpoint, fleet.SearchTargetsRequest{})
	ue.POST("/api/_version_/fleet/targets/count", countTargetsEndpoint, fleet.CountTargetsRequest{})

	ue.POST("/api/_version_/fleet/invites", createInviteEndpoint, fleet.CreateInviteRequest{})
	ue.GET("/api/_version_/fleet/invites", listInvitesEndpoint, fleet.ListInvitesRequest{})
	ue.DELETE("/api/_version_/fleet/invites/{id:[0-9]+}", deleteInviteEndpoint, fleet.DeleteInviteRequest{})
	ue.PATCH("/api/_version_/fleet/invites/{id:[0-9]+}", updateInviteEndpoint, fleet.UpdateInviteRequest{})

	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/global/policies", globalPolicyEndpoint, fleet.GlobalPolicyRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/policies", globalPolicyEndpoint, fleet.GlobalPolicyRequest{})
	ue.EndingAtVersion("v1").GET("/api/_version_/fleet/global/policies", listGlobalPoliciesEndpoint, fleet.ListGlobalPoliciesRequest{})
	ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/policies", listGlobalPoliciesEndpoint, fleet.ListGlobalPoliciesRequest{})
	ue.GET("/api/_version_/fleet/policies/count", countGlobalPoliciesEndpoint, fleet.CountGlobalPoliciesRequest{})
	ue.EndingAtVersion("v1").GET("/api/_version_/fleet/global/policies/{policy_id}", getPolicyByIDEndpoint, fleet.GetPolicyByIDRequest{})
	ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/policies/{policy_id}", getPolicyByIDEndpoint, fleet.GetPolicyByIDRequest{})
	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/global/policies/delete", deleteGlobalPoliciesEndpoint, fleet.DeleteGlobalPoliciesRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/policies/delete", deleteGlobalPoliciesEndpoint, fleet.DeleteGlobalPoliciesRequest{})
	ue.EndingAtVersion("v1").PATCH("/api/_version_/fleet/global/policies/{policy_id}", modifyGlobalPolicyEndpoint, fleet.ModifyGlobalPolicyRequest{})
	ue.StartingAtVersion("2022-04").PATCH("/api/_version_/fleet/policies/{policy_id}", modifyGlobalPolicyEndpoint, fleet.ModifyGlobalPolicyRequest{})
	ue.POST("/api/_version_/fleet/automations/reset", resetAutomationEndpoint, fleet.ResetAutomationRequest{})

	// Alias /api/_version_/fleet/team/ -> /api/_version_/fleet/teams/
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/policies", "/api/_version_/fleet/teams/{fleet_id}/policies").
		POST("/api/_version_/fleet/fleets/{fleet_id}/policies", teamPolicyEndpoint, fleet.TeamPolicyRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/policies", "/api/_version_/fleet/teams/{fleet_id}/policies").
		GET("/api/_version_/fleet/fleets/{fleet_id}/policies", listTeamPoliciesEndpoint, fleet.ListTeamPoliciesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/policies/count", "/api/_version_/fleet/teams/{fleet_id}/policies/count").
		GET("/api/_version_/fleet/fleets/{fleet_id}/policies/count", countTeamPoliciesEndpoint, fleet.CountTeamPoliciesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/policies/{policy_id}", "/api/_version_/fleet/teams/{fleet_id}/policies/{policy_id}").
		GET("/api/_version_/fleet/fleets/{fleet_id}/policies/{policy_id}", getTeamPolicyByIDEndpoint, fleet.GetTeamPolicyByIDRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/policies/delete", "/api/_version_/fleet/teams/{fleet_id}/policies/delete").
		POST("/api/_version_/fleet/fleets/{fleet_id}/policies/delete", deleteTeamPoliciesEndpoint, fleet.DeleteTeamPoliciesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/teams/{fleet_id}/policies/{policy_id}").PATCH("/api/_version_/fleet/fleets/{fleet_id}/policies/{policy_id}", modifyTeamPolicyEndpoint, fleet.ModifyTeamPolicyRequest{})
	ue.WithRequestBodySizeLimit(fleet.MaxSpecSize).POST("/api/_version_/fleet/spec/policies", applyPolicySpecsEndpoint, fleet.ApplyPolicySpecsRequest{})

	ue.POST("/api/_version_/fleet/certificates", createCertificateTemplateEndpoint, fleet.CreateCertificateTemplateRequest{})
	ue.GET("/api/_version_/fleet/certificates", listCertificateTemplatesEndpoint, fleet.ListCertificateTemplatesRequest{})
	ue.GET("/api/_version_/fleet/certificates/{id:[0-9]+}", getCertificateTemplateEndpoint, fleet.GetCertificateTemplateRequest{})
	ue.DELETE("/api/_version_/fleet/certificates/{id:[0-9]+}", deleteCertificateTemplateEndpoint, fleet.DeleteCertificateTemplateRequest{})
	ue.POST("/api/_version_/fleet/spec/certificates", applyCertificateTemplateSpecsEndpoint, fleet.ApplyCertificateTemplateSpecsRequest{})
	ue.DELETE("/api/_version_/fleet/spec/certificates", deleteCertificateTemplateSpecsEndpoint, fleet.DeleteCertificateTemplateSpecsRequest{})

	ue.WithAltPaths("/api/_version_/fleet/queries/{id:[0-9]+}").GET("/api/_version_/fleet/reports/{id:[0-9]+}", getQueryEndpoint, fleet.GetQueryRequest{})
	ue.WithAltPaths("/api/_version_/fleet/queries").GET("/api/_version_/fleet/reports", listQueriesEndpoint, fleet.ListQueriesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/queries/{id:[0-9]+}/report").GET("/api/_version_/fleet/reports/{id:[0-9]+}/report", getQueryReportEndpoint, fleet.GetQueryReportRequest{})
	ue.WithAltPaths("/api/_version_/fleet/queries").POST("/api/_version_/fleet/reports", createQueryEndpoint, fleet.CreateQueryRequest{})
	ue.WithAltPaths("/api/_version_/fleet/queries/{id:[0-9]+}").PATCH("/api/_version_/fleet/reports/{id:[0-9]+}", modifyQueryEndpoint, fleet.ModifyQueryRequest{})
	ue.WithAltPaths("/api/_version_/fleet/queries/{name}").DELETE("/api/_version_/fleet/reports/{name}", deleteQueryEndpoint, fleet.DeleteQueryRequest{})
	ue.WithAltPaths("/api/_version_/fleet/queries/id/{id:[0-9]+}").DELETE("/api/_version_/fleet/reports/id/{id:[0-9]+}", deleteQueryByIDEndpoint, fleet.DeleteQueryByIDRequest{})
	ue.WithAltPaths("/api/_version_/fleet/queries/delete").POST("/api/_version_/fleet/reports/delete", deleteQueriesEndpoint, fleet.DeleteQueriesRequest{})
	ue.WithAltPaths("/api/_version_/fleet/spec/queries").WithRequestBodySizeLimit(fleet.MaxSpecSize).POST("/api/_version_/fleet/spec/reports", applyQuerySpecsEndpoint, fleet.ApplyQuerySpecsRequest{})
	ue.WithAltPaths("/api/_version_/fleet/spec/queries").GET("/api/_version_/fleet/spec/reports", getQuerySpecsEndpoint, fleet.GetQuerySpecsRequest{})
	ue.WithAltPaths("/api/_version_/fleet/spec/queries/{name}").GET("/api/_version_/fleet/spec/reports/{name}", getQuerySpecEndpoint, fleet.GetQuerySpecRequest{})

	ue.GET("/api/_version_/fleet/packs/{id:[0-9]+}", getPackEndpoint, fleet.GetPackRequest{})
	ue.POST("/api/_version_/fleet/packs", createPackEndpoint, fleet.CreatePackRequest{})
	ue.PATCH("/api/_version_/fleet/packs/{id:[0-9]+}", modifyPackEndpoint, fleet.ModifyPackRequest{})
	ue.GET("/api/_version_/fleet/packs", listPacksEndpoint, fleet.ListPacksRequest{})
	ue.DELETE("/api/_version_/fleet/packs/{name}", deletePackEndpoint, fleet.DeletePackRequest{})
	ue.DELETE("/api/_version_/fleet/packs/id/{id:[0-9]+}", deletePackByIDEndpoint, fleet.DeletePackByIDRequest{})
	ue.WithRequestBodySizeLimit(fleet.MaxSpecSize).POST("/api/_version_/fleet/spec/packs", applyPackSpecsEndpoint, fleet.ApplyPackSpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/packs", getPackSpecsEndpoint, nil)
	ue.GET("/api/_version_/fleet/spec/packs/{name}", getPackSpecEndpoint, fleet.GetGenericSpecRequest{})

	ue.GET("/api/_version_/fleet/software/versions", listSoftwareVersionsEndpoint, fleet.ListSoftwareRequest{})
	ue.GET("/api/_version_/fleet/software/versions/{id:[0-9]+}", getSoftwareEndpoint, fleet.GetSoftwareRequest{})

	// DEPRECATED: use /api/_version_/fleet/software/versions instead
	ue.GET("/api/_version_/fleet/software", listSoftwareEndpoint, fleet.ListSoftwareRequest{})
	// DEPRECATED: use /api/_version_/fleet/software/versions{id:[0-9]+} instead
	ue.GET("/api/_version_/fleet/software/{id:[0-9]+}", getSoftwareEndpoint, fleet.GetSoftwareRequest{})
	// DEPRECATED: software version counts are now included directly in the software version list
	ue.GET("/api/_version_/fleet/software/count", countSoftwareEndpoint, fleet.CountSoftwareRequest{})

	ue.GET("/api/_version_/fleet/software/titles", listSoftwareTitlesEndpoint, fleet.ListSoftwareTitlesRequest{})
	ue.GET("/api/_version_/fleet/software/titles/{id:[0-9]+}", getSoftwareTitleEndpoint, fleet.GetSoftwareTitleRequest{})
	ue.POST("/api/_version_/fleet/hosts/{host_id:[0-9]+}/software/{software_title_id:[0-9]+}/install", installSoftwareTitleEndpoint,
		fleet.InstallSoftwareRequest{})
	ue.POST("/api/_version_/fleet/hosts/{host_id:[0-9]+}/software/{software_title_id:[0-9]+}/uninstall", uninstallSoftwareTitleEndpoint,
		fleet.UninstallSoftwareRequest{})

	// Software installers
	ue.GET("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/package", getSoftwareInstallerEndpoint, fleet.GetSoftwareInstallerRequest{})
	ue.POST("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/package/token", getSoftwareInstallerTokenEndpoint,
		fleet.GetSoftwareInstallerRequest{})
	// Software package endpoints are already limited to max installer size in serve.go
	ue.SkipRequestBodySizeLimit().POST("/api/_version_/fleet/software/package", uploadSoftwareInstallerEndpoint, decodeUploadSoftwareInstallerRequest{})
	ue.PATCH("/api/_version_/fleet/software/titles/{id:[0-9]+}/name", updateSoftwareNameEndpoint, fleet.UpdateSoftwareNameRequest{})
	// Software package endpoints are already limited to max installer size in serve.go
	ue.SkipRequestBodySizeLimit().PATCH("/api/_version_/fleet/software/titles/{id:[0-9]+}/package", updateSoftwareInstallerEndpoint, decodeUpdateSoftwareInstallerRequest{})
	ue.DELETE("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/available_for_install", deleteSoftwareInstallerEndpoint, fleet.DeleteSoftwareInstallerRequest{})
	ue.GET("/api/_version_/fleet/software/install/{install_uuid}/results", getSoftwareInstallResultsEndpoint,
		fleet.GetSoftwareInstallResultsRequest{})
	// POST /api/_version_/fleet/software/batch is asynchronous, meaning it will start the process of software download+upload in the background
	// and will return a request UUID to be used in GET /api/_version_/fleet/software/batch/{request_uuid} to query for the status of the operation.
	ue.POST("/api/_version_/fleet/software/batch", batchSetSoftwareInstallersEndpoint, fleet.BatchSetSoftwareInstallersRequest{})
	ue.GET("/api/_version_/fleet/software/batch/{request_uuid}", batchSetSoftwareInstallersResultEndpoint, fleet.BatchSetSoftwareInstallersResultRequest{})

	// software title custom icons
	ue.GET("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/icon", getSoftwareTitleIconsEndpoint, fleet.GetSoftwareTitleIconsRequest{})
	ue.PUT("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/icon", putSoftwareTitleIconEndpoint, decodePutSoftwareTitleIconRequest{})
	ue.DELETE("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/icon", deleteSoftwareTitleIconEndpoint, fleet.DeleteSoftwareTitleIconRequest{})

	// App store software
	ue.GET("/api/_version_/fleet/software/app_store_apps", getAppStoreAppsEndpoint, fleet.GetAppStoreAppsRequest{})
	ue.POST("/api/_version_/fleet/software/app_store_apps", addAppStoreAppEndpoint, fleet.AddAppStoreAppRequest{})
	ue.PATCH("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/app_store_app", updateAppStoreAppEndpoint, fleet.UpdateAppStoreAppRequest{})

	// Setup Experience
	//
	// Setup experience software endpoints:
	ue.PUT("/api/_version_/fleet/setup_experience/software", putSetupExperienceSoftware, fleet.PutSetupExperienceSoftwareRequest{})
	ue.GET("/api/_version_/fleet/setup_experience/software", getSetupExperienceSoftware, fleet.GetSetupExperienceSoftwareRequest{})

	// Setup experience script endpoints:
	ue.GET("/api/_version_/fleet/setup_experience/script", getSetupExperienceScriptEndpoint, fleet.GetSetupExperienceScriptRequest{})
	ue.WithRequestBodySizeLimit(fleet.MaxScriptSize).POST("/api/_version_/fleet/setup_experience/script", setSetupExperienceScriptEndpoint, decodeSetSetupExperienceScriptRequest{})
	ue.DELETE("/api/_version_/fleet/setup_experience/script", deleteSetupExperienceScriptEndpoint, fleet.DeleteSetupExperienceScriptRequest{})

	// Fleet-maintained apps
	ue.WithRequestBodySizeLimit(fleet.MaxMultiScriptQuerySize).POST("/api/_version_/fleet/software/fleet_maintained_apps", addFleetMaintainedAppEndpoint, decodeAddFleetMaintainedAppRequest{})
	ue.GET("/api/_version_/fleet/software/fleet_maintained_apps", listFleetMaintainedAppsEndpoint, fleet.ListFleetMaintainedAppsRequest{})
	ue.GET("/api/_version_/fleet/software/fleet_maintained_apps/{app_id}", getFleetMaintainedApp, fleet.GetFleetMaintainedAppRequest{})

	// Vulnerabilities
	ue.GET("/api/_version_/fleet/vulnerabilities", listVulnerabilitiesEndpoint, fleet.ListVulnerabilitiesRequest{})
	ue.GET("/api/_version_/fleet/vulnerabilities/{cve}", getVulnerabilityEndpoint, fleet.GetVulnerabilityRequest{})

	// Hosts
	ue.GET("/api/_version_/fleet/host_summary", getHostSummaryEndpoint, fleet.GetHostSummaryRequest{})
	ue.GET("/api/_version_/fleet/hosts", listHostsEndpoint, fleet.ListHostsRequest{})
	ue.POST("/api/_version_/fleet/hosts/delete", deleteHostsEndpoint, fleet.DeleteHostsRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}", getHostEndpoint, fleet.GetHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/count", countHostsEndpoint, fleet.CountHostsRequest{})
	ue.POST("/api/_version_/fleet/hosts/search", searchHostsEndpoint, fleet.SearchHostsRequest{})
	ue.GET("/api/_version_/fleet/hosts/identifier/{identifier}", hostByIdentifierEndpoint, fleet.HostByIdentifierRequest{})
	ue.POST("/api/_version_/fleet/hosts/identifier/{identifier}/query", runLiveQueryOnHostEndpoint, fleet.RunLiveQueryOnHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/query", runLiveQueryOnHostByIDEndpoint, fleet.RunLiveQueryOnHostByIDRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}", deleteHostEndpoint, fleet.DeleteHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/transfer", addHostsToTeamEndpoint, fleet.AddHostsToTeamRequest{})
	ue.POST("/api/_version_/fleet/hosts/transfer/filter", addHostsToTeamByFilterEndpoint, fleet.AddHostsToTeamByFilterRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/refetch", refetchHostEndpoint, fleet.RefetchHostRequest{})
	// Deprecated: Device mappings are included in the host details endpoint: /api/_version_/fleet/hosts/{id}
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/device_mapping", listHostDeviceMappingEndpoint, fleet.ListHostDeviceMappingRequest{})
	ue.PUT("/api/_version_/fleet/hosts/{id:[0-9]+}/device_mapping", putHostDeviceMappingEndpoint, fleet.PutHostDeviceMappingRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}/device_mapping/idp", deleteHostIDPEndpoint, fleet.DeleteHostIDPRequest{})
	ue.GET("/api/_version_/fleet/hosts/report", hostsReportEndpoint, fleet.HostsReportRequest{})
	ue.GET("/api/_version_/fleet/os_versions", osVersionsEndpoint, fleet.OsVersionsRequest{})
	ue.GET("/api/_version_/fleet/os_versions/{id:[0-9]+}", getOSVersionEndpoint, fleet.GetOSVersionRequest{})
	ue.WithAltPaths("/api/_version_/fleet/hosts/{id:[0-9]+}/queries/{report_id:[0-9]+}").GET("/api/_version_/fleet/hosts/{id:[0-9]+}/reports/{report_id:[0-9]+}", getHostQueryReportEndpoint, fleet.GetHostQueryReportRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/health", getHostHealthEndpoint, fleet.GetHostHealthRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/labels", addLabelsToHostEndpoint, fleet.AddLabelsToHostRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}/labels", removeLabelsFromHostEndpoint, fleet.RemoveLabelsFromHostRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/software", getHostSoftwareEndpoint, getHostSoftwareDecoder{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/certificates", listHostCertificatesEndpoint, fleet.ListHostCertificatesRequest{})

	ue.GET("/api/_version_/fleet/hosts/summary/mdm", getHostMDMSummary, fleet.GetHostMDMSummaryRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/mdm", getHostMDM, fleet.GetHostMDMRequest{})

	ue.POST("/api/_version_/fleet/labels", createLabelEndpoint, fleet.CreateLabelRequest{})
	ue.PATCH("/api/_version_/fleet/labels/{id:[0-9]+}", modifyLabelEndpoint, fleet.ModifyLabelRequest{})
	ue.GET("/api/_version_/fleet/labels/{id:[0-9]+}", getLabelEndpoint, fleet.GetLabelRequest{})
	ue.GET("/api/_version_/fleet/labels", listLabelsEndpoint, fleet.ListLabelsRequest{})
	ue.GET("/api/_version_/fleet/labels/summary", getLabelsSummaryEndpoint, fleet.GetLabelsSummaryRequest{})
	ue.GET("/api/_version_/fleet/labels/{id:[0-9]+}/hosts", listHostsInLabelEndpoint, fleet.ListHostsInLabelRequest{})
	ue.DELETE("/api/_version_/fleet/labels/{name}", deleteLabelEndpoint, fleet.DeleteLabelRequest{})
	ue.DELETE("/api/_version_/fleet/labels/id/{id:[0-9]+}", deleteLabelByIDEndpoint, fleet.DeleteLabelByIDRequest{})
	ue.WithRequestBodySizeLimit(fleet.MaxSpecSize).POST("/api/_version_/fleet/spec/labels", applyLabelSpecsEndpoint, fleet.ApplyLabelSpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/labels", getLabelSpecsEndpoint, fleet.GetLabelSpecsRequest{})
	ue.GET("/api/_version_/fleet/spec/labels/{name}", getLabelSpecEndpoint, fleet.GetGenericSpecRequest{})

	// This endpoint runs live queries synchronously (with a configured timeout).
	ue.WithAltPaths("/api/_version_/fleet/queries/{id:[0-9]+}/run").POST("/api/_version_/fleet/reports/{id:[0-9]+}/run", runOneLiveQueryEndpoint, fleet.RunOneLiveQueryRequest{})
	// Old endpoint, removed from docs. This GET endpoint runs live queries synchronously (with a configured timeout).
	ue.WithAltPaths("/api/_version_/fleet/queries/run").GET("/api/_version_/fleet/reports/run", runLiveQueryEndpoint, fleet.RunLiveQueryRequest{})
	// The following two POST APIs are the asynchronous way to run live queries.
	// The live queries are created with these two endpoints and their results can be queried via
	// websockets via the `GET /api/_version_/fleet/results/` endpoint.
	ue.WithAltPaths("/api/_version_/fleet/queries/run").POST("/api/_version_/fleet/reports/run", createDistributedQueryCampaignEndpoint, fleet.CreateDistributedQueryCampaignRequest{})
	ue.WithAltPaths("/api/_version_/fleet/queries/run_by_identifiers").POST("/api/_version_/fleet/reports/run_by_identifiers", createDistributedQueryCampaignByIdentifierEndpoint, fleet.CreateDistributedQueryCampaignByIdentifierRequest{})
	// This endpoint is deprecated and maintained for backwards compatibility. This and above endpoint are functionally equivalent
	ue.WithAltPaths("/api/_version_/fleet/queries/run_by_names").POST("/api/_version_/fleet/reports/run_by_names", createDistributedQueryCampaignByIdentifierEndpoint, fleet.CreateDistributedQueryCampaignByIdentifierRequest{})

	ue.GET("/api/_version_/fleet/packs/{id:[0-9]+}/scheduled", getScheduledQueriesInPackEndpoint, fleet.GetScheduledQueriesInPackRequest{})
	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/schedule", scheduleQueryEndpoint, fleet.ScheduleQueryRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/packs/schedule", scheduleQueryEndpoint, fleet.ScheduleQueryRequest{})
	ue.GET("/api/_version_/fleet/schedule/{id:[0-9]+}", getScheduledQueryEndpoint, fleet.GetScheduledQueryRequest{})
	ue.EndingAtVersion("v1").PATCH("/api/_version_/fleet/schedule/{id:[0-9]+}", modifyScheduledQueryEndpoint, fleet.ModifyScheduledQueryRequest{})
	ue.StartingAtVersion("2022-04").PATCH("/api/_version_/fleet/packs/schedule/{id:[0-9]+}", modifyScheduledQueryEndpoint, fleet.ModifyScheduledQueryRequest{})
	ue.EndingAtVersion("v1").DELETE("/api/_version_/fleet/schedule/{id:[0-9]+}", deleteScheduledQueryEndpoint, fleet.DeleteScheduledQueryRequest{})
	ue.StartingAtVersion("2022-04").DELETE("/api/_version_/fleet/packs/schedule/{id:[0-9]+}", deleteScheduledQueryEndpoint, fleet.DeleteScheduledQueryRequest{})

	ue.EndingAtVersion("v1").GET("/api/_version_/fleet/global/schedule", getGlobalScheduleEndpoint, fleet.GetGlobalScheduleRequest{})
	ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/schedule", getGlobalScheduleEndpoint, fleet.GetGlobalScheduleRequest{})
	ue.EndingAtVersion("v1").POST("/api/_version_/fleet/global/schedule", globalScheduleQueryEndpoint, fleet.GlobalScheduleQueryRequest{})
	ue.StartingAtVersion("2022-04").POST("/api/_version_/fleet/schedule", globalScheduleQueryEndpoint, fleet.GlobalScheduleQueryRequest{})
	ue.EndingAtVersion("v1").PATCH("/api/_version_/fleet/global/schedule/{id:[0-9]+}", modifyGlobalScheduleEndpoint, fleet.ModifyGlobalScheduleRequest{})
	ue.StartingAtVersion("2022-04").PATCH("/api/_version_/fleet/schedule/{id:[0-9]+}", modifyGlobalScheduleEndpoint, fleet.ModifyGlobalScheduleRequest{})
	ue.EndingAtVersion("v1").DELETE("/api/_version_/fleet/global/schedule/{id:[0-9]+}", deleteGlobalScheduleEndpoint, fleet.DeleteGlobalScheduleRequest{})
	ue.StartingAtVersion("2022-04").DELETE("/api/_version_/fleet/schedule/{id:[0-9]+}", deleteGlobalScheduleEndpoint, fleet.DeleteGlobalScheduleRequest{})

	// Alias /api/_version_/fleet/team/ -> /api/_version_/fleet/teams/
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/schedule", "/api/_version_/fleet/teams/{fleet_id}/schedule").
		GET("/api/_version_/fleet/fleets/{fleet_id}/schedule", getTeamScheduleEndpoint, fleet.GetTeamScheduleRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/schedule", "/api/_version_/fleet/teams/{fleet_id}/schedule").
		POST("/api/_version_/fleet/fleets/{fleet_id}/schedule", teamScheduleQueryEndpoint, fleet.TeamScheduleQueryRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/schedule/{report_id}", "/api/_version_/fleet/teams/{fleet_id}/schedule/{report_id}").
		PATCH("/api/_version_/fleet/fleets/{fleet_id}/schedule/{report_id}", modifyTeamScheduleEndpoint, fleet.ModifyTeamScheduleRequest{})
	ue.WithAltPaths("/api/_version_/fleet/team/{fleet_id}/schedule/{report_id}", "/api/_version_/fleet/teams/{fleet_id}/schedule/{report_id}").
		DELETE("/api/_version_/fleet/fleets/{fleet_id}/schedule/{report_id}", deleteTeamScheduleEndpoint, fleet.DeleteTeamScheduleRequest{})

	ue.GET("/api/_version_/fleet/carves", listCarvesEndpoint, fleet.ListCarvesRequest{})
	ue.GET("/api/_version_/fleet/carves/{id:[0-9]+}", getCarveEndpoint, fleet.GetCarveRequest{})
	ue.GET("/api/_version_/fleet/carves/{id:[0-9]+}/block/{block_id}", getCarveBlockEndpoint, fleet.GetCarveBlockRequest{})

	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/macadmins", getMacadminsDataEndpoint, fleet.GetMacadminsDataRequest{})
	ue.GET("/api/_version_/fleet/macadmins", getAggregatedMacadminsDataEndpoint, fleet.GetAggregatedMacadminsDataRequest{})

	ue.GET("/api/_version_/fleet/status/result_store", statusResultStoreEndpoint, nil)
	ue.GET("/api/_version_/fleet/status/live_query", statusLiveQueryEndpoint, nil)

	ue.WithRequestBodySizeLimit(fleet.MaxScriptSize).POST("/api/_version_/fleet/scripts/run", runScriptEndpoint, fleet.RunScriptRequest{})
	ue.WithRequestBodySizeLimit(fleet.MaxScriptSize).POST("/api/_version_/fleet/scripts/run/sync", runScriptSyncEndpoint, fleet.RunScriptSyncRequest{})
	ue.POST("/api/_version_/fleet/scripts/run/batch", batchScriptRunEndpoint, fleet.BatchScriptRunRequest{})
	ue.GET("/api/_version_/fleet/scripts/results/{execution_id}", getScriptResultEndpoint, fleet.GetScriptResultRequest{})
	ue.WithRequestBodySizeLimit(fleet.MaxScriptSize).POST("/api/_version_/fleet/scripts", createScriptEndpoint, decodeCreateScriptRequest{})
	ue.GET("/api/_version_/fleet/scripts", listScriptsEndpoint, fleet.ListScriptsRequest{})
	ue.GET("/api/_version_/fleet/scripts/{script_id:[0-9]+}", getScriptEndpoint, fleet.GetScriptRequest{})
	ue.WithRequestBodySizeLimit(fleet.MaxScriptSize).PATCH("/api/_version_/fleet/scripts/{script_id:[0-9]+}", updateScriptEndpoint, decodeUpdateScriptRequest{})
	ue.DELETE("/api/_version_/fleet/scripts/{script_id:[0-9]+}", deleteScriptEndpoint, fleet.DeleteScriptRequest{})
	ue.WithRequestBodySizeLimit(fleet.MaxBatchScriptSize).POST("/api/_version_/fleet/scripts/batch", batchSetScriptsEndpoint, fleet.BatchSetScriptsRequest{})
	ue.POST("/api/_version_/fleet/scripts/batch/{batch_execution_id:[a-zA-Z0-9-]+}/cancel", batchScriptCancelEndpoint, fleet.BatchScriptCancelRequest{})
	// Deprecated, will remove in favor of batchScriptExecutionStatusEndpoint when batch script details page is ready.
	ue.GET("/api/_version_/fleet/scripts/batch/summary/{batch_execution_id:[a-zA-Z0-9-]+}", batchScriptExecutionSummaryEndpoint, batchScriptExecutionSummaryRequest{})
	ue.GET("/api/_version_/fleet/scripts/batch/{batch_execution_id:[a-zA-Z0-9-]+}/host-results", batchScriptExecutionHostResultsEndpoint, fleet.BatchScriptExecutionHostResultsRequest{})
	ue.GET("/api/_version_/fleet/scripts/batch/{batch_execution_id:[a-zA-Z0-9-]+}", batchScriptExecutionStatusEndpoint, fleet.BatchScriptExecutionStatusRequest{})
	ue.GET("/api/_version_/fleet/scripts/batch", batchScriptExecutionListEndpoint, fleet.BatchScriptExecutionListRequest{})

	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/scripts", getHostScriptDetailsEndpoint, fleet.GetHostScriptDetailsRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/activities/upcoming", listHostUpcomingActivitiesEndpoint, fleet.ListHostUpcomingActivitiesRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}/activities/upcoming/{activity_id}", cancelHostUpcomingActivityEndpoint, fleet.CancelHostUpcomingActivityRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/lock", lockHostEndpoint, fleet.LockHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/unlock", unlockHostEndpoint, fleet.UnlockHostRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/wipe", wipeHostEndpoint, fleet.WipeHostRequest{})

	// Generative AI
	ue.POST("/api/_version_/fleet/autofill/policy", autofillPoliciesEndpoint, fleet.AutofillPoliciesRequest{})

	// Secret variables
	ue.PUT("/api/_version_/fleet/spec/secret_variables", createSecretVariablesEndpoint, fleet.CreateSecretVariablesRequest{})
	ue.POST("/api/_version_/fleet/custom_variables", createSecretVariableEndpoint, fleet.CreateSecretVariableRequest{})
	ue.GET("/api/_version_/fleet/custom_variables", listSecretVariablesEndpoint, fleet.ListSecretVariablesRequest{})
	ue.DELETE("/api/_version_/fleet/custom_variables/{id:[0-9]+}", deleteSecretVariableEndpoint, fleet.DeleteSecretVariableRequest{})

	// Scim details
	ue.GET("/api/_version_/fleet/scim/details", getScimDetailsEndpoint, nil)

	// Microsoft Compliance Partner
	ue.POST("/api/_version_/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateEndpoint, fleet.ConditionalAccessMicrosoftCreateRequest{})
	ue.POST("/api/_version_/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmEndpoint, fleet.ConditionalAccessMicrosoftConfirmRequest{})
	ue.DELETE("/api/_version_/fleet/conditional-access/microsoft", conditionalAccessMicrosoftDeleteEndpoint, fleet.ConditionalAccessMicrosoftDeleteRequest{})

	// Okta Conditional Access
	ue.GET("/api/_version_/fleet/conditional_access/idp/signing_cert", conditionalAccessGetIdPSigningCertEndpoint, fleet.ConditionalAccessGetIdPSigningCertRequest{})
	ue.GET("/api/_version_/fleet/conditional_access/idp/apple/profile", conditionalAccessGetIdPAppleProfileEndpoint, nil)

	// Deprecated: PATCH /mdm/apple/setup is now deprecated, replaced by the
	// PATCH /setup_experience endpoint.
	ue.PATCH("/api/_version_/fleet/mdm/apple/setup", updateMDMAppleSetupEndpoint, fleet.UpdateMDMAppleSetupRequest{})
	ue.PATCH("/api/_version_/fleet/setup_experience", updateMDMAppleSetupEndpoint, fleet.UpdateMDMAppleSetupRequest{})

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
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/enqueue", enqueueMDMAppleCommandEndpoint, fleet.EnqueueMDMAppleCommandRequest{})
	// Deprecated: POST /mdm/apple/commandresults is now deprecated, replaced by the
	// platform-agnostic POST /mdm/commands/commandresults. It is still supported
	// indefinitely for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/commandresults", getMDMAppleCommandResultsEndpoint, fleet.GetMDMAppleCommandResultsRequest{})
	// Deprecated: POST /mdm/apple/commands is now deprecated, replaced by the
	// platform-agnostic POST /mdm/commands/commands. It is still supported
	// indefinitely for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/commands", listMDMAppleCommandsEndpoint, fleet.ListMDMAppleCommandsRequest{})
	// Deprecated: those /mdm/apple/profiles/... endpoints are now deprecated,
	// replaced by the platform-agnostic /mdm/profiles/... It is still supported
	// indefinitely for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/profiles/{profile_id:[0-9]+}", getMDMAppleConfigProfileEndpoint, fleet.GetMDMAppleConfigProfileRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/profiles/{profile_id:[0-9]+}", deleteMDMAppleConfigProfileEndpoint, fleet.DeleteMDMAppleConfigProfileRequest{})
	mdmAppleMW.WithRequestBodySizeLimit(fleet.MaxProfileSize).POST("/api/_version_/fleet/mdm/apple/profiles", newMDMAppleConfigProfileEndpoint, decodeNewMDMAppleConfigProfileRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesEndpoint, fleet.ListMDMAppleConfigProfilesRequest{})

	// Deprecated: GET /mdm/apple/filevault/summary is now deprecated, replaced by the
	// platform-agnostic GET /mdm/disk_encryption/summary. It is still supported indefinitely
	// for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/filevault/summary", getMdmAppleFileVaultSummaryEndpoint, fleet.GetMDMAppleFileVaultSummaryRequest{})

	// Deprecated: GET /mdm/apple/profiles/summary is now deprecated, replaced by the
	// platform-agnostic GET /mdm/profiles/summary. It is still supported indefinitely
	// for backwards compatibility.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/profiles/summary", getMDMAppleProfilesSummaryEndpoint, fleet.GetMDMAppleProfilesSummaryRequest{})

	// Deprecated: POST /mdm/apple/enrollment_profile is now deprecated, replaced by the
	// POST /enrollment_profiles/automatic endpoint.
	mdmAppleMW.WithRequestBodySizeLimit(fleet.MaxProfileSize).POST("/api/_version_/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantEndpoint, fleet.CreateMDMAppleSetupAssistantRequest{})
	mdmAppleMW.WithRequestBodySizeLimit(fleet.MaxProfileSize).POST("/api/_version_/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantEndpoint, fleet.CreateMDMAppleSetupAssistantRequest{})

	// Deprecated: GET /mdm/apple/enrollment_profile is now deprecated, replaced by the
	// GET /enrollment_profiles/automatic endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/enrollment_profile", getMDMAppleSetupAssistantEndpoint, fleet.GetMDMAppleSetupAssistantRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/enrollment_profiles/automatic", getMDMAppleSetupAssistantEndpoint, fleet.GetMDMAppleSetupAssistantRequest{})

	// Deprecated: DELETE /mdm/apple/enrollment_profile is now deprecated, replaced by the
	// DELETE /enrollment_profiles/automatic endpoint.
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/enrollment_profile", deleteMDMAppleSetupAssistantEndpoint, fleet.DeleteMDMAppleSetupAssistantRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/enrollment_profiles/automatic", deleteMDMAppleSetupAssistantEndpoint, fleet.DeleteMDMAppleSetupAssistantRequest{})

	// TODO: are those undocumented endpoints still needed? I think they were only used
	// by 'fleetctl apple-mdm' sub-commands.
	// Generous limit for these unknown old unused endpoints-
	mdmAppleMW.WithRequestBodySizeLimit(512*units.MiB).POST("/api/_version_/fleet/mdm/apple/installers", uploadAppleInstallerEndpoint, decodeUploadAppleInstallerRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/installers/{installer_id:[0-9]+}", getAppleInstallerEndpoint, fleet.GetAppleInstallerDetailsRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/installers/{installer_id:[0-9]+}", deleteAppleInstallerEndpoint, fleet.DeleteAppleInstallerDetailsRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/installers", listMDMAppleInstallersEndpoint, fleet.ListMDMAppleInstallersRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/devices", listMDMAppleDevicesEndpoint, fleet.ListMDMAppleDevicesRequest{})

	// Deprecated: GET /mdm/manual_enrollment_profile is now deprecated, replaced by the
	// GET /enrollment_profiles/manual endpoint.
	// Ref: https://github.com/fleetdm/fleet/issues/16252
	mdmAppleMW.GET("/api/_version_/fleet/mdm/manual_enrollment_profile", getManualEnrollmentProfileEndpoint, fleet.GetManualEnrollmentProfileRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/enrollment_profiles/manual", getManualEnrollmentProfileEndpoint, fleet.GetManualEnrollmentProfileRequest{})

	// bootstrap-package routes

	// Deprecated: POST /mdm/bootstrap is now deprecated, replaced by the
	// POST /bootstrap endpoint.
	// Bootstrap endpoints are already max size limited to installer size in serve.go
	mdmAppleMW.SkipRequestBodySizeLimit().POST("/api/_version_/fleet/mdm/bootstrap", uploadBootstrapPackageEndpoint, decodeUploadBootstrapPackageRequest{})
	mdmAppleMW.SkipRequestBodySizeLimit().POST("/api/_version_/fleet/bootstrap", uploadBootstrapPackageEndpoint, decodeUploadBootstrapPackageRequest{})

	// Deprecated: GET /mdm/bootstrap/:team_id/metadata is now deprecated, replaced by the
	// GET /bootstrap/:team_id/metadata endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/bootstrap/{fleet_id:[0-9]+}/metadata", bootstrapPackageMetadataEndpoint, fleet.BootstrapPackageMetadataRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/bootstrap/{fleet_id:[0-9]+}/metadata", bootstrapPackageMetadataEndpoint, fleet.BootstrapPackageMetadataRequest{})

	// Deprecated: DELETE /mdm/bootstrap/:team_id is now deprecated, replaced by the
	// DELETE /bootstrap/:team_id endpoint.
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/bootstrap/{fleet_id:[0-9]+}", deleteBootstrapPackageEndpoint, fleet.DeleteBootstrapPackageRequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/bootstrap/{fleet_id:[0-9]+}", deleteBootstrapPackageEndpoint, fleet.DeleteBootstrapPackageRequest{})

	// Deprecated: GET /mdm/bootstrap/summary is now deprecated, replaced by the
	// GET /bootstrap/summary endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/bootstrap/summary", getMDMAppleBootstrapPackageSummaryEndpoint, fleet.GetMDMAppleBootstrapPackageSummaryRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/bootstrap/summary", getMDMAppleBootstrapPackageSummaryEndpoint, fleet.GetMDMAppleBootstrapPackageSummaryRequest{})

	// Deprecated: POST /mdm/apple/bootstrap is now deprecated, replaced by the platform agnostic /mdm/bootstrap
	// Bootstrap endpoints are already max size limited to installer size in serve.go
	mdmAppleMW.SkipRequestBodySizeLimit().POST("/api/_version_/fleet/mdm/apple/bootstrap", uploadBootstrapPackageEndpoint, decodeUploadBootstrapPackageRequest{})
	// Deprecated: GET /mdm/apple/bootstrap/:team_id/metadata is now deprecated, replaced by the platform agnostic /mdm/bootstrap/:team_id/metadata
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/bootstrap/{fleet_id:[0-9]+}/metadata", bootstrapPackageMetadataEndpoint, fleet.BootstrapPackageMetadataRequest{})
	// Deprecated: DELETE /mdm/apple/bootstrap/:team_id is now deprecated, replaced by the platform agnostic /mdm/bootstrap/:team_id
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/bootstrap/{fleet_id:[0-9]+}", deleteBootstrapPackageEndpoint, fleet.DeleteBootstrapPackageRequest{})
	// Deprecated: GET /mdm/apple/bootstrap/summary is now deprecated, replaced by the platform agnostic /mdm/bootstrap/summary
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/bootstrap/summary", getMDMAppleBootstrapPackageSummaryEndpoint, fleet.GetMDMAppleBootstrapPackageSummaryRequest{})

	// host-specific mdm routes

	// Deprecated: POST /mdm/hosts/:id/lock is now deprecated, replaced by
	// POST /hosts/:id/lock.
	mdmAppleMW.POST("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/lock", deviceLockEndpoint, fleet.DeviceLockRequest{})
	mdmAppleMW.POST("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/wipe", deviceWipeEndpoint, fleet.DeviceWipeRequest{})

	// Deprecated: GET /mdm/hosts/:id/profiles is now deprecated, replaced by
	// GET /hosts/:id/configuration_profiles.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/profiles", getHostProfilesEndpoint, fleet.GetHostProfilesRequest{})
	// TODO: Confirm if response should be updated to include Windows profiles and use mdmAnyMW
	mdmAppleMW.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/configuration_profiles", getHostProfilesEndpoint, fleet.GetHostProfilesRequest{})

	// Deprecated: GET /mdm/apple is now deprecated, replaced by the
	// GET /apns endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple", getAppleMDMEndpoint, nil)
	mdmAppleMW.GET("/api/_version_/fleet/apns", getAppleMDMEndpoint, nil)

	// EULA routes

	// Deprecated: POST /mdm/setup/eula is now deprecated, replaced by the
	// POST /setup_experience/eula endpoint.
	mdmAppleMW.WithRequestBodySizeLimit(fleet.MaxEULASize).POST("/api/_version_/fleet/mdm/setup/eula", createMDMEULAEndpoint, decodeCreateMDMEULARequest{})
	mdmAppleMW.WithRequestBodySizeLimit(fleet.MaxEULASize).POST("/api/_version_/fleet/setup_experience/eula", createMDMEULAEndpoint, decodeCreateMDMEULARequest{})

	// Deprecated: GET /mdm/setup/eula/metadata is now deprecated, replaced by the
	// GET /setup_experience/eula/metadata endpoint.
	mdmAppleMW.GET("/api/_version_/fleet/mdm/setup/eula/metadata", getMDMEULAMetadataEndpoint, fleet.GetMDMEULAMetadataRequest{})
	mdmAppleMW.GET("/api/_version_/fleet/setup_experience/eula/metadata", getMDMEULAMetadataEndpoint, fleet.GetMDMEULAMetadataRequest{})

	// Deprecated: DELETE /mdm/setup/eula/:token is now deprecated, replaced by the
	// DELETE /setup_experience/eula/:token endpoint.
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/setup/eula/{token}", deleteMDMEULAEndpoint, fleet.DeleteMDMEULARequest{})
	mdmAppleMW.DELETE("/api/_version_/fleet/setup_experience/eula/{token}", deleteMDMEULAEndpoint, fleet.DeleteMDMEULARequest{})

	// Deprecated: POST /mdm/apple/setup/eula is now deprecated, replaced by the platform agnostic /mdm/setup/eula
	mdmAppleMW.WithRequestBodySizeLimit(fleet.MaxEULASize).POST("/api/_version_/fleet/mdm/apple/setup/eula", createMDMEULAEndpoint, decodeCreateMDMEULARequest{})
	// Deprecated: GET /mdm/apple/setup/eula/metadata is now deprecated, replaced by the platform agnostic /mdm/setup/eula/metadata
	mdmAppleMW.GET("/api/_version_/fleet/mdm/apple/setup/eula/metadata", getMDMEULAMetadataEndpoint, fleet.GetMDMEULAMetadataRequest{})
	// Deprecated: DELETE /mdm/apple/setup/eula/:token is now deprecated, replaced by the platform agnostic /mdm/setup/eula/:token
	mdmAppleMW.DELETE("/api/_version_/fleet/mdm/apple/setup/eula/{token}", deleteMDMEULAEndpoint, fleet.DeleteMDMEULARequest{})

	mdmAppleMW.WithRequestBodySizeLimit(fleet.MaxProfileSize).POST("/api/_version_/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileEndpoint, fleet.PreassignMDMAppleProfileRequest{})
	mdmAppleMW.POST("/api/_version_/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentEndpoint, fleet.MatchMDMApplePreassignmentRequest{})

	mdmAnyMW := ue.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyAnyMDM())

	// Deprecated: POST /mdm/commands/run is now deprecated, replaced by the
	// POST /commands/run endpoint.
	mdmAnyMW.WithRequestBodySizeLimit(fleet.MaxMDMCommandSize).POST("/api/_version_/fleet/mdm/commands/run", runMDMCommandEndpoint, fleet.RunMDMCommandRequest{})
	mdmAnyMW.WithRequestBodySizeLimit(fleet.MaxMDMCommandSize).POST("/api/_version_/fleet/commands/run", runMDMCommandEndpoint, fleet.RunMDMCommandRequest{})

	// Deprecated: GET /mdm/commandresults is now deprecated, replaced by the
	// GET /commands/results endpoint.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/commandresults", getMDMCommandResultsEndpoint, fleet.GetMDMCommandResultsRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/commands/results", getMDMCommandResultsEndpoint, fleet.GetMDMCommandResultsRequest{})

	// Deprecated: GET /mdm/commands is now deprecated, replaced by the
	// GET /commands endpoint.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/commands", listMDMCommandsEndpoint, fleet.ListMDMCommandsRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/commands", listMDMCommandsEndpoint, fleet.ListMDMCommandsRequest{})

	// Deprecated: PATCH /mdm/hosts/:id/unenroll is now deprecated, replaced by
	// DELETE /hosts/:id/mdm.
	mdmAnyMW.PATCH("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/unenroll", mdmUnenrollEndpoint, fleet.MdmUnenrollRequest{})
	mdmAnyMW.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}/mdm", mdmUnenrollEndpoint, fleet.MdmUnenrollRequest{})

	// Deprecated: GET /mdm/disk_encryption/summary is now deprecated, replaced by the
	// GET /disk_encryption endpoint.
	ue.GET("/api/_version_/fleet/mdm/disk_encryption/summary", getMDMDiskEncryptionSummaryEndpoint, fleet.GetMDMDiskEncryptionSummaryRequest{})
	ue.GET("/api/_version_/fleet/disk_encryption", getMDMDiskEncryptionSummaryEndpoint, fleet.GetMDMDiskEncryptionSummaryRequest{})

	// Deprecated: GET /mdm/hosts/:id/encryption_key is now deprecated, replaced by
	// GET /hosts/:id/encryption_key.
	ue.GET("/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/encryption_key", getHostEncryptionKey, fleet.GetHostEncryptionKeyRequest{})
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/encryption_key", getHostEncryptionKey, fleet.GetHostEncryptionKeyRequest{})

	// Deprecated: GET /mdm/profiles/summary is now deprecated, replaced by the
	// GET /configuration_profiles/summary endpoint.
	ue.GET("/api/_version_/fleet/mdm/profiles/summary", getMDMProfilesSummaryEndpoint, fleet.GetMDMProfilesSummaryRequest{})
	ue.GET("/api/_version_/fleet/configuration_profiles/summary", getMDMProfilesSummaryEndpoint, fleet.GetMDMProfilesSummaryRequest{})

	// Deprecated: GET /mdm/profiles/:profile_uuid is now deprecated, replaced by
	// GET /configuration_profiles/:profile_uuid.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/profiles/{profile_uuid}", getMDMConfigProfileEndpoint, fleet.GetMDMConfigProfileRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/configuration_profiles/{profile_uuid}", getMDMConfigProfileEndpoint, fleet.GetMDMConfigProfileRequest{})

	// Deprecated: DELETE /mdm/profiles/:profile_uuid is now deprecated, replaced by
	// DELETE /configuration_profiles/:profile_uuid.
	ue.DELETE("/api/_version_/fleet/mdm/profiles/{profile_uuid}", deleteMDMConfigProfileEndpoint, fleet.DeleteMDMConfigProfileRequest{})
	ue.DELETE("/api/_version_/fleet/configuration_profiles/{profile_uuid}", deleteMDMConfigProfileEndpoint, fleet.DeleteMDMConfigProfileRequest{})

	// Deprecated: GET /mdm/profiles is now deprecated, replaced by the
	// GET /configuration_profiles endpoint.
	mdmAnyMW.GET("/api/_version_/fleet/mdm/profiles", listMDMConfigProfilesEndpoint, fleet.ListMDMConfigProfilesRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/configuration_profiles", listMDMConfigProfilesEndpoint, fleet.ListMDMConfigProfilesRequest{})

	// Deprecated: POST /mdm/profiles is now deprecated, replaced by the
	// POST /configuration_profiles endpoint.
	mdmAnyMW.WithRequestBodySizeLimit(fleet.MaxProfileSize).POST("/api/_version_/fleet/mdm/profiles", newMDMConfigProfileEndpoint, decodeNewMDMConfigProfileRequest{})
	mdmAnyMW.WithRequestBodySizeLimit(fleet.MaxProfileSize).POST("/api/_version_/fleet/configuration_profiles", newMDMConfigProfileEndpoint, decodeNewMDMConfigProfileRequest{})
	// Batch needs to allow being called without any MDM enabled, to support deleting profiles, but will fail later if trying to add
	ue.WithRequestBodySizeLimit(fleet.MaxBatchProfileSize).POST("/api/_version_/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesEndpoint, fleet.BatchModifyMDMConfigProfilesRequest{})

	// Deprecated: POST /hosts/{host_id:[0-9]+}/configuration_profiles/resend/{profile_uuid} is now deprecated, replaced by the
	// POST /hosts/{host_id:[0-9]+}/configuration_profiles/{profile_uuid}/resend endpoint.
	mdmAnyMW.POST("/api/_version_/fleet/hosts/{host_id:[0-9]+}/configuration_profiles/resend/{profile_uuid}", resendHostMDMProfileEndpoint, fleet.ResendHostMDMProfileRequest{})
	mdmAnyMW.POST("/api/_version_/fleet/hosts/{host_id:[0-9]+}/configuration_profiles/{profile_uuid}/resend", resendHostMDMProfileEndpoint, fleet.ResendHostMDMProfileRequest{})
	mdmAnyMW.POST("/api/_version_/fleet/configuration_profiles/resend/batch", batchResendMDMProfileToHostsEndpoint, fleet.BatchResendMDMProfileToHostsRequest{})
	mdmAnyMW.GET("/api/_version_/fleet/configuration_profiles/{profile_uuid}/status", getMDMConfigProfileStatusEndpoint, fleet.GetMDMConfigProfileStatusRequest{})

	// Deprecated: PATCH /mdm/apple/settings is deprecated, replaced by POST /disk_encryption.
	// It was only used to set disk encryption.
	mdmAnyMW.PATCH("/api/_version_/fleet/mdm/apple/settings", updateMDMAppleSettingsEndpoint, fleet.UpdateMDMAppleSettingsRequest{})
	ue.POST("/api/_version_/fleet/disk_encryption", updateDiskEncryptionEndpoint, fleet.UpdateDiskEncryptionRequest{})

	// the following set of mdm endpoints must always be accessible (even
	// if MDM is not configured) as it bootstraps the setup of MDM
	// (generates CSR request for APNs, plus the SCEP and ABM keypairs).
	// Deprecated: this endpoint shouldn't be used anymore in favor of the
	// new flow described in https://github.com/fleetdm/fleet/issues/10383
	ue.POST("/api/_version_/fleet/mdm/apple/request_csr", requestMDMAppleCSREndpoint, fleet.RequestMDMAppleCSRRequest{})
	// Deprecated: this endpoint shouldn't be used anymore in favor of the
	// new flow described in https://github.com/fleetdm/fleet/issues/10383
	ue.POST("/api/_version_/fleet/mdm/apple/dep/key_pair", newMDMAppleDEPKeyPairEndpoint, nil)
	ue.GET("/api/_version_/fleet/mdm/apple/abm_public_key", generateABMKeyPairEndpoint, nil)
	ue.POST("/api/_version_/fleet/abm_tokens", uploadABMTokenEndpoint, decodeUploadABMTokenRequest{})
	ue.DELETE("/api/_version_/fleet/abm_tokens/{id:[0-9]+}", deleteABMTokenEndpoint, fleet.DeleteABMTokenRequest{})
	ue.GET("/api/_version_/fleet/abm_tokens", listABMTokensEndpoint, nil)
	ue.GET("/api/_version_/fleet/abm_tokens/count", countABMTokensEndpoint, nil)
	ue.WithAltPaths("/api/_version_/fleet/abm_tokens/{id:[0-9]+}/teams").PATCH("/api/_version_/fleet/abm_tokens/{id:[0-9]+}/fleets", updateABMTokenTeamsEndpoint, fleet.UpdateABMTokenTeamsRequest{})
	ue.PATCH("/api/_version_/fleet/abm_tokens/{id:[0-9]+}/renew", renewABMTokenEndpoint, decodeRenewABMTokenRequest{})

	ue.GET("/api/_version_/fleet/mdm/apple/request_csr", getMDMAppleCSREndpoint, fleet.GetMDMAppleCSRRequest{})
	ue.POST("/api/_version_/fleet/mdm/apple/apns_certificate", uploadMDMAppleAPNSCertEndpoint, decodeUploadMDMAppleAPNSCertRequest{})
	ue.DELETE("/api/_version_/fleet/mdm/apple/apns_certificate", deleteMDMAppleAPNSCertEndpoint, fleet.DeleteMDMAppleAPNSCertRequest{})

	// VPP Tokens
	ue.GET("/api/_version_/fleet/vpp_tokens", getVPPTokens, fleet.GetVPPTokensRequest{})
	ue.POST("/api/_version_/fleet/vpp_tokens", uploadVPPTokenEndpoint, decodeUploadVPPTokenRequest{})
	ue.WithAltPaths("/api/_version_/fleet/vpp_tokens/{id}/teams").PATCH("/api/_version_/fleet/vpp_tokens/{id}/fleets", patchVPPTokensTeams, fleet.PatchVPPTokensTeamsRequest{})
	ue.PATCH("/api/_version_/fleet/vpp_tokens/{id}/renew", patchVPPTokenRenewEndpoint, decodePatchVPPTokenRenewRequest{})
	ue.DELETE("/api/_version_/fleet/vpp_tokens/{id}", deleteVPPToken, fleet.DeleteVPPTokenRequest{})

	// Batch VPP Associations
	ue.POST("/api/_version_/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsEndpoint, fleet.BatchAssociateAppStoreAppsRequest{})

	// Deprecated: GET /mdm/apple_bm is now deprecated, replaced by the
	// GET /abm endpoint.
	ue.GET("/api/_version_/fleet/mdm/apple_bm", getAppleBMEndpoint, nil)
	// Deprecated: GET /abm is now deprecated, replaced by the GET /abm_tokens endpoint.
	ue.GET("/api/_version_/fleet/abm", getAppleBMEndpoint, nil)

	// Deprecated: POST /mdm/apple/profiles/batch is now deprecated, replaced by the
	// platform-agnostic POST /mdm/profiles/batch. It is still supported
	// indefinitely for backwards compatibility.
	//
	// batch-apply is accessible even though MDM is not enabled, it needs
	// to support the case where `fleetctl get config`'s output is used as
	// input to `fleetctl apply`
	ue.WithRequestBodySizeLimit(fleet.MaxBatchProfileSize).POST("/api/_version_/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesEndpoint, fleet.BatchSetMDMAppleProfilesRequest{})

	// batch-apply is accessible even though MDM is not enabled, it needs
	// to support the case where `fleetctl get config`'s output is used as
	// input to `fleetctl apply`
	ue.WithRequestBodySizeLimit(fleet.MaxBatchProfileSize).POST("/api/_version_/fleet/mdm/profiles/batch", batchSetMDMProfilesEndpoint, fleet.BatchSetMDMProfilesRequest{})

	// Certificate Authority endpoints
	ue.POST("/api/_version_/fleet/certificate_authorities", createCertificateAuthorityEndpoint, fleet.CreateCertificateAuthorityRequest{})
	ue.GET("/api/_version_/fleet/certificate_authorities", listCertificateAuthoritiesEndpoint, fleet.ListCertificateAuthoritiesRequest{})
	ue.GET("/api/_version_/fleet/certificate_authorities/{id:[0-9]+}", getCertificateAuthorityEndpoint, fleet.GetCertificateAuthorityRequest{})
	ue.DELETE("/api/_version_/fleet/certificate_authorities/{id:[0-9]+}", deleteCertificateAuthorityEndpoint, fleet.DeleteCertificateAuthorityRequest{})
	ue.PATCH("/api/_version_/fleet/certificate_authorities/{id:[0-9]+}", updateCertificateAuthorityEndpoint, fleet.UpdateCertificateAuthorityRequest{})
	ue.POST("/api/_version_/fleet/certificate_authorities/{id:[0-9]+}/request_certificate", requestCertificateEndpoint, fleet.RequestCertificateRequest{})
	ue.POST("/api/_version_/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesEndpoint, fleet.BatchApplyCertificateAuthoritiesRequest{})
	ue.GET("/api/_version_/fleet/spec/certificate_authorities", getCertificateAuthoritiesSpecEndpoint, fleet.GetCertificateAuthoritiesSpecRequest{})

	ipBanner := redis.NewIPBanner(redisPool, "ipbanner::",
		deviceIPAllowedConsecutiveFailingRequestsCount,
		deviceIPAllowedConsecutiveFailingRequestsTimeWindow,
		deviceIPBanTime,
	)
	errorLimiter := ratelimit.NewErrorMiddleware(ipBanner).Limit(logger.SlogLogger())

	// Device-authenticated endpoints.
	de := newDeviceAuthenticatedEndpointer(svc, logger, opts, r, apiVersions...)
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}", getDeviceHostEndpoint, fleet.GetDeviceHostRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/desktop", getFleetDesktopEndpoint, fleet.GetFleetDesktopRequest{})
	de.WithCustomMiddleware(errorLimiter).HEAD("/api/_version_/fleet/device/{token}/ping", devicePingEndpoint, fleet.DeviceAuthPingRequest{})
	de.WithCustomMiddleware(errorLimiter).POST("/api/_version_/fleet/device/{token}/refetch", refetchDeviceHostEndpoint, fleet.RefetchDeviceHostRequest{})
	// Deprecated: Device mapping data is now included in host details endpoint
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/device_mapping", listDeviceHostDeviceMappingEndpoint, fleet.ListDeviceHostDeviceMappingRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/macadmins", getDeviceMacadminsDataEndpoint, fleet.GetDeviceMacadminsDataRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/policies", listDevicePoliciesEndpoint, fleet.ListDevicePoliciesRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/transparency", transparencyURL, fleet.TransparencyURLRequest{})
	de.WithCustomMiddleware(errorLimiter).WithRequestBodySizeLimit(fleet.MaxFleetdErrorReportSize).POST("/api/_version_/fleet/device/{token}/debug/errors", fleetdError, fleet.FleetdErrorRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/software", getDeviceSoftwareEndpoint, fleet.GetDeviceSoftwareRequest{})
	de.WithCustomMiddleware(errorLimiter).POST("/api/_version_/fleet/device/{token}/software/install/{software_title_id}", submitSelfServiceSoftwareInstall, fleet.FleetSelfServiceSoftwareInstallRequest{})
	de.WithCustomMiddleware(errorLimiter).POST("/api/_version_/fleet/device/{token}/software/uninstall/{software_title_id}", submitDeviceSoftwareUninstall, fleet.FleetDeviceSoftwareUninstallRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/software/install/{install_uuid}/results", getDeviceSoftwareInstallResultsEndpoint, fleet.GetDeviceSoftwareInstallResultsRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/software/uninstall/{execution_id}/results", getDeviceSoftwareUninstallResultsEndpoint, fleet.GetDeviceSoftwareUninstallResultsRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/certificates", listDeviceCertificatesEndpoint, fleet.ListDeviceCertificatesRequest{})
	de.WithCustomMiddleware(errorLimiter).POST("/api/_version_/fleet/device/{token}/setup_experience/status", getDeviceSetupExperienceStatusEndpoint, fleet.GetDeviceSetupExperienceStatusRequest{})
	de.WithCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/software/titles/{software_title_id}/icon", getDeviceSoftwareIconEndpoint, fleet.GetDeviceSoftwareIconRequest{})
	de.WithCustomMiddleware(errorLimiter).POST("/api/_version_/fleet/device/{token}/mdm/linux/trigger_escrow", triggerLinuxDiskEncryptionEscrowEndpoint, fleet.TriggerLinuxDiskEncryptionEscrowRequest{})
	de.WithCustomMiddleware(errorLimiter).POST("/api/_version_/fleet/device/{token}/bypass_conditional_access", bypassConditionalAccessEndpoint, fleet.BypassConditionalAccessRequest{})
	// Device authenticated, Apple MDM endpoints.
	demdm := de.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyAppleMDM())
	demdm.AppendCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/mdm/apple/manual_enrollment_profile", getDeviceMDMManualEnrollProfileEndpoint, fleet.GetDeviceMDMManualEnrollProfileRequest{})
	demdm.AppendCustomMiddleware(errorLimiter).GET("/api/_version_/fleet/device/{token}/software/commands/{command_uuid}/results", getDeviceMDMCommandResultsEndpoint, fleet.GetDeviceMDMCommandResultsRequest{})
	demdm.AppendCustomMiddleware(errorLimiter).POST("/api/_version_/fleet/device/{token}/configuration_profiles/{profile_uuid}/resend", resendDeviceConfigurationProfileEndpoint, fleet.ResendDeviceConfigurationProfileRequest{})
	demdm.AppendCustomMiddleware(errorLimiter).POST("/api/_version_/fleet/device/{token}/migrate_mdm", migrateMDMDeviceEndpoint, fleet.DeviceMigrateMDMRequest{})

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
		POST("/api/osquery/config", getClientConfigEndpoint, fleet.GetClientConfigRequest{})
	he.WithAltPaths("/api/v1/osquery/distributed/read").
		POST("/api/osquery/distributed/read", getDistributedQueriesEndpoint, fleet.GetDistributedQueriesRequest{})
	he.WithRequestBodySizeLimit(fleet.MaxOsqueryDistributedWriteSize).WithAltPaths("/api/v1/osquery/distributed/write").
		POST("/api/osquery/distributed/write", submitDistributedQueryResultsEndpoint, submitDistributedQueryResultsRequestShim{})
	he.WithAltPaths("/api/v1/osquery/carve/begin").
		POST("/api/osquery/carve/begin", carveBeginEndpoint, fleet.CarveBeginRequest{})
	he.WithAltPaths("/api/v1/osquery/log").
		POST("/api/osquery/log", submitLogsEndpoint, fleet.SubmitLogsRequest{})
	he.WithAltPaths("/api/v1/osquery/yara/{name}").
		POST("/api/osquery/yara/{name}", getYaraEndpoint, fleet.GetYaraRequest{})

	// android authenticated end-points
	// Authentication is implemented using the orbit_node_key from the 'Authentication' header.
	// The 'orbit_node_key' is used because it's the only thing we have available when the device gets enrolled
	// after the MDM setup is complete.
	androidEndpoints := androidAuthenticatedEndpointer(svc, logger, opts, r, apiVersions...)
	androidEndpoints.GET("/api/fleetd/certificates/{id:[0-9]+}", getDeviceCertificateTemplateEndpoint, fleet.GetDeviceCertificateTemplateRequest{})
	androidEndpoints.PUT("/api/fleetd/certificates/{id:[0-9]+}/status", updateCertificateStatusEndpoint, fleet.UpdateCertificateStatusRequest{})

	// orbit authenticated endpoints
	oe := newOrbitAuthenticatedEndpointer(svc, logger, opts, r, apiVersions...)
	oe.POST("/api/fleet/orbit/device_token", setOrUpdateDeviceTokenEndpoint, fleet.SetOrUpdateDeviceTokenRequest{})
	oe.POST("/api/fleet/orbit/config", getOrbitConfigEndpoint, fleet.OrbitGetConfigRequest{})
	// using POST to get a script execution request since all authenticated orbit
	// endpoints are POST due to passing the device token in the JSON body.
	oe.POST("/api/fleet/orbit/scripts/request", getOrbitScriptEndpoint, fleet.OrbitGetScriptRequest{})
	oe.POST("/api/fleet/orbit/scripts/result", postOrbitScriptResultEndpoint, fleet.OrbitPostScriptResultRequest{})
	oe.PUT("/api/fleet/orbit/device_mapping", putOrbitDeviceMappingEndpoint, fleet.OrbitPutDeviceMappingRequest{})
	oe.WithRequestBodySizeLimit(fleet.MaxMultiScriptQuerySize).POST("/api/fleet/orbit/software_install/result", postOrbitSoftwareInstallResultEndpoint, fleet.OrbitPostSoftwareInstallResultRequest{})
	oe.POST("/api/fleet/orbit/software_install/package", orbitDownloadSoftwareInstallerEndpoint, fleet.OrbitDownloadSoftwareInstallerRequest{})
	oe.POST("/api/fleet/orbit/software_install/details", getOrbitSoftwareInstallDetails, fleet.OrbitGetSoftwareInstallRequest{})
	oe.POST("/api/fleet/orbit/setup_experience/init", orbitSetupExperienceInitEndpoint, fleet.OrbitSetupExperienceInitRequest{})

	// POST /api/fleet/orbit/setup_experience/status is used by macOS and Linux hosts.
	// For macOS hosts we verify Apple MDM is enabled and configured.
	oeAppleMDM := oe.WithCustomMiddlewareAfterAuth(mdmConfiguredMiddleware.VerifyAppleMDMOnMacOSHosts())
	oeAppleMDM.POST("/api/fleet/orbit/setup_experience/status", getOrbitSetupExperienceStatusEndpoint, fleet.GetOrbitSetupExperienceStatusRequest{})

	oeWindowsMDM := oe.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyWindowsMDM())
	oeWindowsMDM.POST("/api/fleet/orbit/disk_encryption_key", postOrbitDiskEncryptionKeyEndpoint, fleet.OrbitPostDiskEncryptionKeyRequest{})

	oe.POST("/api/fleet/orbit/luks_data", postOrbitLUKSEndpoint, fleet.OrbitPostLUKSRequest{})

	// unauthenticated endpoints - most of those are either login-related,
	// invite-related or host-enrolling. So they typically do some kind of
	// one-time authentication by verifying that a valid secret token is provided
	// with the request.
	ne := newNoAuthEndpointer(svc, opts, r, apiVersions...)
	ne.WithAltPaths("/api/v1/osquery/enroll").
		POST("/api/osquery/enroll", enrollAgentEndpoint, contract.EnrollOsqueryAgentRequest{})

	// These endpoint are token authenticated.
	// NOTE: remember to update
	// `service.mdmConfigurationRequiredEndpoints` when you add an
	// endpoint that's behind the mdmConfiguredMiddleware, this applies
	// both to this set of endpoints and to any user authenticated
	// endpoints using `mdmAppleMW.*` above in this file.
	neAppleMDM := ne.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyAppleMDM())
	neAppleMDM.GET(apple_mdm.EnrollPath, mdmAppleEnrollEndpoint, decodeMdmAppleEnrollRequest{})
	neAppleMDM.POST(apple_mdm.EnrollPath, mdmAppleEnrollEndpoint, decodeMdmAppleEnrollRequest{})

	neAppleMDM.GET(apple_mdm.InstallerPath, mdmAppleGetInstallerEndpoint, fleet.MdmAppleGetInstallerRequest{})
	neAppleMDM.HEAD(apple_mdm.InstallerPath, mdmAppleHeadInstallerEndpoint, fleet.MdmAppleHeadInstallerRequest{})
	neAppleMDM.POST("/api/_version_/fleet/ota_enrollment", mdmAppleOTAEndpoint, decodeMdmAppleOTARequest{})

	// Deprecated: GET /mdm/bootstrap is now deprecated, replaced by the
	// GET /bootstrap endpoint.
	neAppleMDM.GET("/api/_version_/fleet/mdm/bootstrap", downloadBootstrapPackageEndpoint, fleet.DownloadBootstrapPackageRequest{})
	neAppleMDM.GET("/api/_version_/fleet/bootstrap", downloadBootstrapPackageEndpoint, fleet.DownloadBootstrapPackageRequest{})

	// Deprecated: GET /mdm/apple/bootstrap is now deprecated, replaced by the platform agnostic /mdm/bootstrap
	neAppleMDM.GET("/api/_version_/fleet/mdm/apple/bootstrap", downloadBootstrapPackageEndpoint, fleet.DownloadBootstrapPackageRequest{})

	// Deprecated: GET /mdm/setup/eula/:token is now deprecated, replaced by the
	// GET /setup_experience/eula/:token endpoint.
	neAppleMDM.GET("/api/_version_/fleet/mdm/setup/eula/{token}", getMDMEULAEndpoint, fleet.GetMDMEULARequest{})
	neAppleMDM.GET("/api/_version_/fleet/setup_experience/eula/{token}", getMDMEULAEndpoint, fleet.GetMDMEULARequest{})

	// Deprecated: GET /mdm/apple/setup/eula/:token is now deprecated, replaced by the platform agnostic /mdm/setup/eula/:token
	neAppleMDM.GET("/api/_version_/fleet/mdm/apple/setup/eula/{token}", getMDMEULAEndpoint, fleet.GetMDMEULARequest{})

	// Get OTA profile
	neAppleMDM.GET("/api/_version_/fleet/enrollment_profiles/ota", getOTAProfileEndpoint, decodeGetOTAProfileRequest{})

	// This is the account-driven enrollment endpoint for BYoD Apple devices, also known as User Enrollment.
	neAppleMDM.POST(apple_mdm.AccountDrivenEnrollPath, mdmAppleAccountEnrollEndpoint, decodeMdmAppleAccountEnrollRequest{})
	// This is for OAUTH2 token based auth
	// ne.POST(apple_mdm.EnrollPath+"/token", mdmAppleAccountEnrollTokenEndpoint, mdmAppleAccountEnrollTokenRequest{})

	// These endpoint are used by Microsoft devices during MDM device enrollment phase
	neWindowsMDM := ne.WithCustomMiddleware(mdmConfiguredMiddleware.VerifyWindowsMDM())

	// Microsoft MS-MDE2 Endpoints
	// This endpoint is unauthenticated and is used by Microsoft devices to discover the MDM server endpoints
	neWindowsMDM.WithRequestBodySizeLimit(fleet.MaxMicrosoftMDMSize).POST(microsoft_mdm.MDE2DiscoveryPath, mdmMicrosoftDiscoveryEndpoint, SoapRequestContainer{})

	// This endpoint is unauthenticated and is used by Microsoft devices to retrieve the opaque STS auth token
	neWindowsMDM.WithRequestBodySizeLimit(fleet.MaxMicrosoftMDMSize).GET(microsoft_mdm.MDE2AuthPath, mdmMicrosoftAuthEndpoint, SoapRequestContainer{})

	// This endpoint is authenticated using the BinarySecurityToken header field
	neWindowsMDM.WithRequestBodySizeLimit(fleet.MaxMicrosoftMDMSize).POST(microsoft_mdm.MDE2PolicyPath, mdmMicrosoftPolicyEndpoint, SoapRequestContainer{})

	// This endpoint is authenticated using the BinarySecurityToken header field
	neWindowsMDM.WithRequestBodySizeLimit(fleet.MaxMicrosoftMDMSize).POST(microsoft_mdm.MDE2EnrollPath, mdmMicrosoftEnrollEndpoint, SoapRequestContainer{})

	// This endpoint is unauthenticated for now
	// It should be authenticated through TLS headers once proper implementation is in place
	neWindowsMDM.WithRequestBodySizeLimit(fleet.MaxMicrosoftMDMSize).POST(microsoft_mdm.MDE2ManagementPath, mdmMicrosoftManagementEndpoint, SyncMLReqMsgContainer{})

	// This endpoint is unauthenticated and is used by to retrieve the MDM enrollment Terms of Use
	neWindowsMDM.WithRequestBodySizeLimit(fleet.MaxMicrosoftMDMSize).GET(microsoft_mdm.MDE2TOSPath, mdmMicrosoftTOSEndpoint, MDMWebContainer{})

	// These endpoints are unauthenticated and made from orbit, and add the orbit capabilities header.
	neOrbit := newOrbitNoAuthEndpointer(svc, opts, r, apiVersions...)
	neOrbit.POST("/api/fleet/orbit/enroll", enrollOrbitEndpoint, contract.EnrollOrbitRequest{})

	ne.GET("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/in_house_app", getInHouseAppPackageEndpoint, fleet.GetInHouseAppPackageRequest{})
	ne.GET("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/in_house_app/manifest", getInHouseAppManifestEndpoint, fleet.GetInHouseAppManifestRequest{})

	// For some reason osquery does not provide a node key with the block data.
	// Instead the carve session ID should be verified in the service method.
	// Since []byte slices is encoded as base64 in JSON, increase the limit to 1.5x
	ne.SkipRequestBodySizeLimit().WithAltPaths("/api/v1/osquery/carve/block").
		POST("/api/osquery/carve/block", carveBlockEndpoint, decodeCarveBlockRequest{})

	ne.GET("/api/_version_/fleet/software/titles/{title_id:[0-9]+}/package/token/{token}", downloadSoftwareInstallerEndpoint,
		fleet.DownloadSoftwareInstallerRequest{})

	ne.POST("/api/_version_/fleet/perform_required_password_reset", performRequiredPasswordResetEndpoint, fleet.PerformRequiredPasswordResetRequest{})
	ne.POST("/api/_version_/fleet/users", createUserFromInviteEndpoint, fleet.CreateUserRequest{})
	ne.GET("/api/_version_/fleet/invites/{token}", verifyInviteEndpoint, fleet.VerifyInviteRequest{})
	ne.POST("/api/_version_/fleet/reset_password", resetPasswordEndpoint, fleet.ResetPasswordRequest{})
	ne.POST("/api/_version_/fleet/logout", logoutEndpoint, nil)
	ne.POST("/api/v1/fleet/sso", initiateSSOEndpoint, fleet.InitiateSSORequest{})
	ne.POST("/api/v1/fleet/sso/callback", makeCallbackSSOEndpoint(config.Server.URLPrefix), callbackSSODecoder{})
	ne.GET("/api/v1/fleet/sso", settingsSSOEndpoint, nil)

	// the websocket distributed query results endpoint is a bit different - the
	// provided path is a prefix, not an exact match, and it is not a go-kit
	// endpoint but a raw http.Handler. It uses the NoAuthEndpointer because
	// authentication is done when the websocket session is established, inside
	// the handler.
	ne.UsePathPrefix().PathHandler("GET", "/api/_version_/fleet/results/",
		makeStreamDistributedQueryCampaignResultsHandler(config.Server, svc, logger))

	quota := throttled.RateQuota{MaxRate: throttled.PerHour(10), MaxBurst: forgotPasswordRateLimitMaxBurst}
	limiter := ratelimit.NewMiddleware(limitStore)
	ne.
		WithCustomMiddleware(limiter.Limit("forgot_password", quota)).
		POST("/api/_version_/fleet/forgot_password", forgotPasswordEndpoint, fleet.ForgotPasswordRequest{})

	// By default, MDM SSO shares the login rate limit bucket; if MDM SSO limit is overridden, MDM SSO gets its
	// own rate limit bucket.
	loginRateLimit := throttled.PerMin(10)
	if extra.loginRateLimit != nil {
		loginRateLimit = *extra.loginRateLimit
	}
	loginLimiter := limiter.Limit("login", throttled.RateQuota{MaxRate: loginRateLimit, MaxBurst: 9})
	mdmSsoLimiter := loginLimiter
	if extra.mdmSsoRateLimit != nil {
		mdmSsoLimiter = limiter.Limit("mdm_sso", throttled.RateQuota{MaxRate: *extra.mdmSsoRateLimit, MaxBurst: 9})
	}

	ne.WithCustomMiddleware(loginLimiter).
		POST("/api/_version_/fleet/login", loginEndpoint, contract.LoginRequest{})
	ne.WithCustomMiddleware(limiter.Limit("mfa", throttled.RateQuota{MaxRate: loginRateLimit, MaxBurst: 9})).
		POST("/api/_version_/fleet/sessions", sessionCreateEndpoint, fleet.SessionCreateRequest{})

	ne.HEAD("/api/fleet/device/ping", devicePingEndpoint, fleet.DevicePingRequest{})

	ne.HEAD("/api/fleet/orbit/ping", orbitPingEndpoint, fleet.OrbitPingRequest{})

	// This is a callback endpoint for calendar integration -- it is called to notify an event change in a user calendar
	ne.POST("/api/_version_/fleet/calendar/webhook/{event_uuid}", calendarWebhookEndpoint, decodeCalendarWebhookRequest{})

	neAppleMDM.WithCustomMiddleware(mdmSsoLimiter).
		POST("/api/_version_/fleet/mdm/sso", initiateMDMSSOEndpoint, fleet.InitiateMDMSSORequest{})
	ne.WithCustomMiddleware(mdmSsoLimiter).
		POST("/api/_version_/fleet/mdm/sso/callback", callbackMDMSSOEndpoint, callbackMDMSSODecoder{})
}

// WithSetup is an http middleware that checks if setup procedures have been completed.
// If setup hasn't been completed it serves the API with a setup middleware.
// If the server is already configured, the default API handler is exposed.
func WithSetup(svc fleet.Service, logger *logging.Logger, next http.Handler) http.HandlerFunc {
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
func RedirectLoginToSetup(svc fleet.Service, logger *logging.Logger, next http.Handler, urlPrefix string) http.HandlerFunc {
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
func RedirectSetupToLogin(svc fleet.Service, logger *logging.Logger, next http.Handler, urlPrefix string) http.HandlerFunc {
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
	mdmStorage fleet.MDMAppleStore,
	scepStorage scep_depot.Depot,
	logger *logging.Logger,
	checkinAndCommandService nanomdm_service.CheckinAndCommandService,
	ddmService nanomdm_service.DeclarativeManagement,
	profileService nanomdm_service.ProfileService,
	serverURLPrefix string,
	fleetConfig config.FleetConfig,
) error {
	if err := registerSCEP(mux, scepConfig, scepStorage, mdmStorage, logger, fleetConfig); err != nil {
		return fmt.Errorf("scep: %w", err)
	}
	if err := registerMDM(mux, mdmStorage, checkinAndCommandService, ddmService, profileService, logger, fleetConfig); err != nil {
		return fmt.Errorf("mdm: %w", err)
	}
	if err := registerMDMServiceDiscovery(mux, logger, serverURLPrefix, fleetConfig); err != nil {
		return fmt.Errorf("service discovery: %w", err)
	}
	return nil
}

func registerMDMServiceDiscovery(
	mux *http.ServeMux,
	logger *logging.Logger,
	serverURLPrefix string,
	fleetConfig config.FleetConfig,
) error {
	serviceDiscoveryLogger := logger.With("component", "mdm-apple-service-discovery")
	fullMDMEnrollmentURL := fmt.Sprintf("%s%s", serverURLPrefix, apple_mdm.AccountDrivenEnrollPath)
	serviceDiscoveryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceDiscoveryLogger.Log("msg", "serving MDM service discovery response", "url", fullMDMEnrollmentURL)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, `{"Servers":[{"Version": "mdm-byod", "BaseURL": "%s"}]}`, fullMDMEnrollmentURL)
		if err != nil {
			serviceDiscoveryLogger.Log("err", "error writing service discovery response", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
	mux.Handle(apple_mdm.ServiceDiscoveryPath, otel.WrapHandler(serviceDiscoveryHandler, apple_mdm.ServiceDiscoveryPath, fleetConfig))
	return nil
}

// registerSCEP registers the HTTP handler for SCEP service needed for enrollment to MDM.
// Returns the SCEP CA certificate that can be used by verifiers.
func registerSCEP(
	mux *http.ServeMux,
	scepConfig config.MDMConfig,
	scepStorage scep_depot.Depot,
	mdmStorage fleet.MDMAppleStore,
	logger *logging.Logger,
	fleetConfig config.FleetConfig,
) error {
	var signer scepserver.CSRSignerContext = scepserver.SignCSRAdapter(scep_depot.NewSigner(
		scepStorage,
		scep_depot.WithValidityDays(scepConfig.AppleSCEPSignerValidityDays),
		scep_depot.WithAllowRenewalDays(scepConfig.AppleSCEPSignerAllowRenewalDays),
	))
	assets, err := mdmStorage.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{fleet.MDMAssetSCEPChallenge}, nil)
	if err != nil {
		return fmt.Errorf("retrieving SCEP challenge: %w", err)
	}

	scepChallenge := string(assets[fleet.MDMAssetSCEPChallenge].Value)
	signer = scepserver.StaticChallengeMiddleware(scepChallenge, signer)
	scepService := NewSCEPService(
		mdmStorage,
		signer,
		logger.With("component", "mdm-apple-scep"),
	)

	scepLogger := logger.With("component", "http-mdm-apple-scep")
	e := scepserver.MakeServerEndpoints(scepService)
	e.GetEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.GetEndpoint)
	e.PostEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.PostEndpoint)
	scepHandler := scepserver.MakeHTTPHandler(e, scepService, scepLogger)
	mux.Handle(apple_mdm.SCEPPath, otel.WrapHandler(scepHandler, apple_mdm.SCEPPath, fleetConfig))
	return nil
}

func RegisterSCEPProxy(
	rootMux *http.ServeMux,
	ds fleet.Datastore,
	logger *logging.Logger,
	timeout *time.Duration,
	fleetConfig *config.FleetConfig,
) error {
	if fleetConfig == nil {
		return errors.New("fleet config is nil")
	}
	scepService := eeservice.NewSCEPProxyService(
		ds,
		logger.With("component", "scep-proxy-service"),
		timeout,
	)
	scepLogger := logger.With("component", "http-scep-proxy")
	e := scepserver.MakeServerEndpointsWithIdentifier(scepService)
	e.GetEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.GetEndpoint)
	e.PostEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.PostEndpoint)
	scepHandler := scepserver.MakeHTTPHandlerWithIdentifier(e, apple_mdm.SCEPProxyPath, scepLogger)
	// Not using OTEL dynamic wrapper so as not to expose {identifier} in the span name
	scepHandler = otel.WrapHandler(scepHandler, apple_mdm.SCEPProxyPath, *fleetConfig)
	rootMux.Handle(apple_mdm.SCEPProxyPath, scepHandler)
	return nil
}

// NanoMDMLogger is a logger adapter for nanomdm.
type NanoMDMLogger struct {
	logger *logging.Logger
}

func NewNanoMDMLogger(logger *logging.Logger) *NanoMDMLogger {
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
	return &NanoMDMLogger{
		logger: l.logger.With(keyvals...),
	}
}

// registerMDM registers the HTTP handlers that serve core MDM services (like checking in for MDM commands).
func registerMDM(
	mux *http.ServeMux,
	mdmStorage fleet.MDMAppleStore,
	checkinAndCommandService nanomdm_service.CheckinAndCommandService,
	ddmService nanomdm_service.DeclarativeManagement,
	profileService nanomdm_service.ProfileService,
	logger *logging.Logger,
	fleetConfig config.FleetConfig,
) error {
	certVerifier := mdmcrypto.NewSCEPVerifier(mdmStorage)
	mdmLogger := NewNanoMDMLogger(logger.With("component", "http-mdm-apple-mdm"))

	// As usual, handlers are applied from bottom to top:
	// 1. Extract and verify MDM signature.
	// 2. Verify signer certificate with CA.
	// 3. Verify new or enrolled certificate (certauth.CertAuth which wraps the MDM service).
	// 4. Pass a copy of the request to Fleet middleware that ingests new hosts from pending MDM
	// enrollments and updates the Fleet hosts table accordingly with the UDID and serial number of
	// the device.
	// 5. Run actual MDM service operation (checkin handler or command and results handler).
	coreMDMService := nanomdm.New(mdmStorage, nanomdm.WithLogger(mdmLogger), nanomdm.WithDeclarativeManagement(ddmService),
		nanomdm.WithProfileService(profileService), nanomdm.WithUserAuthenticate(checkinAndCommandService))
	// NOTE: it is critical that the coreMDMService runs first, as the first
	// service in the multi-service feature is run to completion _before_ running
	// the other ones in parallel. This way, subsequent services have access to
	// the result of the core service, e.g. the device is enrolled, etc.
	var mdmService nanomdm_service.CheckinAndCommandService = multi.New(mdmLogger, coreMDMService, checkinAndCommandService)

	mdmService = certauth.New(mdmService, mdmStorage, certauth.WithLogger(mdmLogger.With("handler", "cert-auth")))
	var mdmHandler http.Handler = httpmdm.CheckinAndCommandHandler(mdmService, mdmLogger.With("handler", "checkin-command"))
	verifyDisable, exists := os.LookupEnv("FLEET_MDM_APPLE_SCEP_VERIFY_DISABLE")
	if exists && (strings.EqualFold(verifyDisable, "true") || verifyDisable == "1") {
		level.Info(logger).Log("msg",
			"disabling verification of macOS SCEP certificates as FLEET_MDM_APPLE_SCEP_VERIFY_DISABLE is set to true")
	} else {
		mdmHandler = httpmdm.CertVerifyMiddleware(mdmHandler, certVerifier, mdmLogger.With("handler", "cert-verify"))
	}
	mdmHandler = httpmdm.CertExtractMdmSignatureMiddleware(mdmHandler, httpmdm.MdmSignatureVerifierFunc(cryptoutil.VerifyMdmSignature),
		httpmdm.SigLogWithLogger(mdmLogger.With("handler", "cert-extract")))
	mux.Handle(apple_mdm.MDMPath, otel.WrapHandler(mdmHandler, apple_mdm.MDMPath, fleetConfig))
	return nil
}

func WithMDMEnrollmentMiddleware(svc fleet.Service, logger *logging.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mdm/sso" && r.URL.Path != "/account_driven_enroll/sso" {
			// TODO: redirects for non-SSO config web url?
			next.ServeHTTP(w, r)
			return
		}

		// if x-apple-aspen-deviceinfo custom header is present, we need to check for minimum os version
		di := r.Header.Get("x-apple-aspen-deviceinfo")
		if di != "" {
			parsed, err := apple_mdm.ParseDeviceinfo(di, false) // FIXME: use verify=true when we have better parsing for various Apple certs (https://github.com/fleetdm/fleet/issues/20879)
			if err != nil {
				// just log the error and continue to next
				level.Error(logger).Log("msg", "parsing x-apple-aspen-deviceinfo", "err", err)
				next.ServeHTTP(w, r)
				return
			}

			// TODO: skip os version check if deviceinfo query param is present? or find another way
			// to avoid polling the DB and Apple endpoint twice for each enrollment.

			sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(r.Context(), parsed)
			if err != nil {
				// just log the error and continue to next
				level.Error(logger).Log("msg", "checking minimum os version for mdm", "err", err)
				next.ServeHTTP(w, r)
				return
			}

			if sur != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				if err := json.NewEncoder(w).Encode(sur); err != nil {
					level.Error(logger).Log("msg", "failed to encode software update required", "err", err)
					http.Redirect(w, r, r.URL.String()+"?error=true", http.StatusSeeOther)
				}
				return
			}

			// TODO: Do non-Apple devices ever use this route? If so, we probably need to change the
			// approach below so we don't endlessly redirect non-Apple clients to the same URL.

			// if we get here, the minimum os version is satisfied, so we continue with SSO flow
			q := r.URL.Query()
			v, ok := q["deviceinfo"]
			if !ok || len(v) == 0 {
				// If the deviceinfo query param is empty, we add the deviceinfo to the URL and
				// redirect.
				//
				// Note: We'll apply this redirect only if query params are empty because want to
				// redirect to the same URL with added query params after parsing the x-apple-aspen-deviceinfo
				// header. Whenever we see a request with any query params already present, we'll
				// skip this step and just continue to the next handler.
				newURL := *r.URL
				q.Set("deviceinfo", di)
				newURL.RawQuery = q.Encode()
				level.Info(logger).Log("msg", "handling mdm sso: redirect with deviceinfo", "host_uuid", parsed.UDID, "serial", parsed.Serial)
				http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
				return
			}
			if len(v) > 0 && v[0] != di {
				// something is wrong, the device info in the query params does not match
				// the one in the header, so we just log the error and continue to next
				level.Error(logger).Log("msg", "device info in query params does not match header", "header", di, "query", v[0])
			}
			level.Info(logger).Log("msg", "handling mdm sso: proceed to next", "host_uuid", parsed.UDID, "serial", parsed.Serial)
		}

		next.ServeHTTP(w, r)
	}
}
