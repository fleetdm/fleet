// Package service holds the implementation of the fleet interface and HTTP
// endpoints for the API
package service

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WatchBeam/clock"
	gocache "github.com/patrickmn/go-cache"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_storage "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/fleetdm/fleet/v4/server/sso"
)

var _ fleet.Service = (*Service)(nil)

// Service is the struct implementing fleet.Service. Create a new one with NewService.
type Service struct {
	ds             fleet.Datastore
	task           *async.Task
	carveStore     fleet.CarveStore
	resultStore    fleet.QueryResultStore
	liveQueryStore fleet.LiveQueryStore
	logger         *slog.Logger
	config         config.FleetConfig
	clock          clock.Clock

	osqueryLogWriter *OsqueryLogger

	mailService     fleet.MailService
	ssoSessionStore sso.SessionStore

	failingPolicySet  fleet.FailingPolicySet
	enrollHostLimiter fleet.EnrollHostLimiter

	authz *authz.Authorizer

	jitterMu *sync.RWMutex
	jitterH  map[time.Duration]*jitterHashTable

	geoIP fleet.GeoIP

	*fleet.EnterpriseOverrides

	depStorage        nanodep_storage.AllDEPStorage
	mdmStorage        nanomdm_storage.AllStorage
	mdmPushService    nanomdm_push.Pusher
	mdmAppleCommander *apple_mdm.MDMAppleCommander

	cronSchedulesService fleet.CronSchedulesService

	wstepCertManager  microsoft_mdm.CertManager
	scepConfigService fleet.SCEPConfigService
	digiCertService   fleet.DigiCertService

	conditionalAccessMicrosoftProxy ConditionalAccessMicrosoftProxy

	keyValueStore         fleet.KeyValueStore
	installAttemptCounter fleet.SoftwareInstallAttemptCounter

	// configETagStore powers the osquery config ETag SHORT CIRCUIT (see
	// GetClientConfigWithETag in osquery.go). It is nil unless the
	// osquery.redis_config_etags feature flag is enabled AND Redis is
	// configured — nil is what turns the short circuit off, there is no other
	// gate at request time.
	configETagStore fleet.ConfigETagStore
	// configETagStateOnce bounds the "optimization state first observed" log
	// to once per Fleet container (see GetClientConfigWithETag). A pointer,
	// because some Service methods use value receivers and sync.Once must
	// not be copied.
	configETagStateOnce *sync.Once
	// configETagErrLast rate-limits config-ETag error logging (unix seconds
	// of the last emitted error; see logConfigETagError). A pointer for the
	// same no-copy reason.
	configETagErrLast *atomic.Int64

	androidSvc android.Service

	// activitySvc is the activity bounded context service for write operations.
	activitySvc fleet.ActivityWriteService

	// acmeSvc is the ACME service module for write operations.
	acmeSvc fleet.ACMEWriteService

	// orgLogoStore stores the bytes of customer-uploaded org logos.
	orgLogoStore fleet.OrgLogoStore

	// agentNotifier publishes check-in wake-ups for agents connected over the
	// WebSocket transport; nil when the transport is disabled.
	agentNotifier fleet.AgentCheckInNotifier

	// packConfigCache caches marshaled pack config JSON per (teamID, queryReportsDisabled).
	// Avoids redundant DB queries and JSON marshaling for identical pack configs.
	// Nil when osquery.config_in_memory_cache is disabled.
	packConfigCache *gocache.Cache
}

// ConditionalAccessMicrosoftProxy is the interface of the Microsoft compliance proxy.
type ConditionalAccessMicrosoftProxy interface {
	// Create creates the integration on the MS proxy and returns the consent URL.
	Create(ctx context.Context, tenantID string) (*conditional_access_microsoft_proxy.CreateResponse, error)
	// Get returns the integration settings.
	Get(ctx context.Context, tenantID string, secret string) (*conditional_access_microsoft_proxy.GetResponse, error)
	// Delete deprovisions the tenant on Microsoft and deletes the integration in the proxy service.
	// Returns a fleet.IsNotFound error if the integration doesn't exist.
	Delete(ctx context.Context, tenantID string, secret string) (*conditional_access_microsoft_proxy.DeleteResponse, error)
	// SetComplianceStatus sets the inventory and compliance status of a host.
	// Returns the message ID to query the status of the operation (MS has an asynchronous API).
	SetComplianceStatus(
		ctx context.Context,
		tenantID string, secret string,
		deviceID string,
		userPrincipalName string,
		mdmEnrolled bool,
		deviceName, osName, osVersion string,
		compliant bool,
		lastCheckInTime time.Time,
	) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error)
	// GetMessageStatusResponse returns the status of a "compliance set" operation.
	GetMessageStatus(ctx context.Context, tenantID string, secret string, messageID string) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error)
}

func (svc *Service) LookupGeoIP(ctx context.Context, ip string) *fleet.GeoLocation {
	return svc.geoIP.Lookup(ctx, ip)
}

func (svc *Service) SetEnterpriseOverrides(overrides fleet.EnterpriseOverrides) {
	svc.EnterpriseOverrides = &overrides
}

// OsqueryLogger holds osqueryd's status and result loggers.
type OsqueryLogger struct {
	// Status holds the osqueryd's status logger.
	//
	// See https://osquery.readthedocs.io/en/stable/deployment/logging/#status-logs
	Status fleet.JSONLogger
	// Result holds the osqueryd's result logger.
	//
	// See https://osquery.readthedocs.io/en/stable/deployment/logging/#results-logs
	Result fleet.JSONLogger
}

// PackConfigCacheTTL is how long a marshaled pack config stays in packConfigCache
const PackConfigCacheTTL = 1 * time.Minute

// NewService creates a new service from the config struct
func NewService(
	ctx context.Context,
	ds fleet.Datastore,
	task *async.Task,
	resultStore fleet.QueryResultStore,
	logger *slog.Logger,
	osqueryLogger *OsqueryLogger,
	config config.FleetConfig,
	mailService fleet.MailService,
	c clock.Clock,
	sso sso.SessionStore,
	lq fleet.LiveQueryStore,
	carveStore fleet.CarveStore,
	failingPolicySet fleet.FailingPolicySet,
	geoIP fleet.GeoIP,
	enrollHostLimiter fleet.EnrollHostLimiter,
	depStorage nanodep_storage.AllDEPStorage,
	mdmStorage fleet.MDMAppleStore,
	mdmPushService nanomdm_push.Pusher,
	cronSchedulesService fleet.CronSchedulesService,
	wstepCertManager microsoft_mdm.CertManager,
	scepConfigService fleet.SCEPConfigService,
	digiCertService fleet.DigiCertService,
	conditionalAccessProxy ConditionalAccessMicrosoftProxy,
	keyValueStore fleet.KeyValueStore,
	installAttemptCounter fleet.SoftwareInstallAttemptCounter,
	androidSvc android.Service,
	orgLogoStore fleet.OrgLogoStore,
) (fleet.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	var packConfigCache *gocache.Cache
	if config.Osquery.ConfigInMemoryCache {
		packConfigCache = gocache.New(PackConfigCacheTTL, 30*time.Second)
	}

	svc := &Service{
		ds:                ds,
		task:              task,
		carveStore:        carveStore,
		resultStore:       resultStore,
		liveQueryStore:    lq,
		logger:            logger,
		config:            config,
		clock:             c,
		osqueryLogWriter:  osqueryLogger,
		mailService:       mailService,
		ssoSessionStore:   sso,
		failingPolicySet:  failingPolicySet,
		authz:             authorizer,
		jitterH:           make(map[time.Duration]*jitterHashTable),
		jitterMu:          new(sync.RWMutex),
		geoIP:             geoIP,
		enrollHostLimiter: enrollHostLimiter,
		depStorage:        depStorage,
		// TODO: remove mdmStorage and mdmPushService when
		// we remove deprecated top-level service methods
		// from the prototype.
		mdmStorage:           mdmStorage,
		mdmPushService:       mdmPushService,
		mdmAppleCommander:    apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService),
		cronSchedulesService: cronSchedulesService,
		wstepCertManager:     wstepCertManager,
		scepConfigService:    scepConfigService,
		digiCertService:      digiCertService,

		conditionalAccessMicrosoftProxy: conditionalAccessProxy,
		keyValueStore:                   keyValueStore,
		configETagStateOnce:             new(sync.Once),
		configETagErrLast:               new(atomic.Int64),
		installAttemptCounter:           installAttemptCounter,
		androidSvc:                      androidSvc,
		orgLogoStore:                    orgLogoStore,
		packConfigCache:                 packConfigCache,
	}
	return validationMiddleware{svc, ds, sso}, nil
}

func (svc *Service) SendEmail(ctx context.Context, mail fleet.Email) error {
	return svc.mailService.SendEmail(ctx, mail)
}

// SetConfigETagStore injects the Redis-backed osquery config ETag store,
// enabling the config SHORT CIRCUIT (see GetClientConfigWithETag in
// osquery.go). Called after NewService, and ONLY when the
// osquery.redis_config_etags feature flag is enabled — leaving the store nil
// is what keeps the short circuit off.
func (svc *Service) SetConfigETagStore(store fleet.ConfigETagStore) {
	svc.configETagStore = store
}

// SetActivityService sets the activity bounded context service for write operations.
// This should be called after NewService to inject the activity service dependency.
func (svc *Service) SetActivityService(activitySvc fleet.ActivityWriteService) {
	svc.activitySvc = activitySvc
}

// SetACMEService sets the ACME service module service for write operations.
// This should be called after NewService to inject the ACME service dependency.
func (svc *Service) SetACMEService(acmeSvc fleet.ACMEWriteService) {
	svc.acmeSvc = acmeSvc
}

// SetAgentCheckInNotifier sets the notifier used to wake up agents connected
// over the WebSocket transport; when unset, no notifications are published.
func (svc *Service) SetAgentCheckInNotifier(notifier fleet.AgentCheckInNotifier) {
	svc.agentNotifier = notifier
}

type validationMiddleware struct {
	fleet.Service
	ds              fleet.Datastore
	ssoSessionStore sso.SessionStore
}

// getAssetURL simply returns the base url used for retrieving image assets from fleetdm.com.
func getAssetURL() template.URL {
	return template.URL("https://fleetdm.com/images/permanent")
}

// emailLinkBaseURL returns the base URL used to build links in transactional
// emails. The server URL is the source of truth; the URL prefix is appended
// only when the server URL does not already carry it. This keeps links correct
// whether an operator configures the subpath in the server URL, in the URL
// prefix, or both, instead of duplicating it (e.g. https://host/p/p/login).
func emailLinkBaseURL(serverURL, urlPrefix string) template.URL {
	if urlPrefix != "" && !strings.HasSuffix(strings.TrimSuffix(serverURL, "/"), urlPrefix) {
		if joined, err := url.JoinPath(serverURL, urlPrefix); err == nil {
			serverURL = joined
		}
	}
	return template.URL(serverURL) //nolint:gosec // G203: operator-configured URL, not user input
}
