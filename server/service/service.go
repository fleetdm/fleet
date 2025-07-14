// Package service holds the implementation of the fleet interface and HTTP
// endpoints for the API
package service

import (
	"context"
	"fmt"
	"html/template"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_storage "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/fleetdm/fleet/v4/server/sso"
	kitlog "github.com/go-kit/log"
)

var _ fleet.Service = (*Service)(nil)

// Service is the struct implementing fleet.Service. Create a new one with NewService.
type Service struct {
	ds             fleet.Datastore
	task           *async.Task
	carveStore     fleet.CarveStore
	resultStore    fleet.QueryResultStore
	liveQueryStore fleet.LiveQueryStore
	logger         kitlog.Logger
	config         config.FleetConfig
	clock          clock.Clock

	osqueryLogWriter *OsqueryLogger

	mailService     fleet.MailService
	ssoSessionStore sso.SessionStore

	failingPolicySet  fleet.FailingPolicySet
	enrollHostLimiter fleet.EnrollHostLimiter

	authz *authz.Authorizer

	jitterMu *sync.Mutex
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

// NewService creates a new service from the config struct
func NewService(
	ctx context.Context,
	ds fleet.Datastore,
	task *async.Task,
	resultStore fleet.QueryResultStore,
	logger kitlog.Logger,
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
) (fleet.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
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
		jitterMu:          new(sync.Mutex),
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
	}
	return validationMiddleware{svc, ds, sso}, nil
}

func (svc *Service) SendEmail(ctx context.Context, mail fleet.Email) error {
	return svc.mailService.SendEmail(ctx, mail)
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
