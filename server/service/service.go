// Package service holds the implementation of the fleet interface and HTTP
// endpoints for the API
package service

import (
	"html/template"
	"strings"
	"sync"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/sso"
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kit/version"
	"github.com/pkg/errors"
)

// Service is the struct implementing fleet.Service. Create a new one with NewService.
type Service struct {
	ds             fleet.Datastore
	carveStore     fleet.CarveStore
	resultStore    fleet.QueryResultStore
	liveQueryStore fleet.LiveQueryStore
	logger         kitlog.Logger
	config         config.FleetConfig
	clock          clock.Clock
	license        fleet.LicenseInfo

	osqueryLogWriter *logging.OsqueryLogger

	mailService     fleet.MailService
	ssoSessionStore sso.SessionStore

	seenHostSet *seenHostSet

	authz *authz.Authorizer
}

// NewService creates a new service from the config struct
func NewService(ds fleet.Datastore, resultStore fleet.QueryResultStore,
	logger kitlog.Logger, osqueryLogger *logging.OsqueryLogger, config config.FleetConfig, mailService fleet.MailService,
	c clock.Clock, sso sso.SessionStore, lq fleet.LiveQueryStore, carveStore fleet.CarveStore,
	license fleet.LicenseInfo) (fleet.Service, error) {
	var svc fleet.Service

	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, errors.Wrap(err, "new authorizer")
	}

	svc = &Service{
		ds:               ds,
		carveStore:       carveStore,
		resultStore:      resultStore,
		liveQueryStore:   lq,
		logger:           logger,
		config:           config,
		clock:            c,
		osqueryLogWriter: osqueryLogger,
		mailService:      mailService,
		ssoSessionStore:  sso,
		seenHostSet:      newSeenHostSet(),
		license:          license,
		authz:            authorizer,
	}
	svc = validationMiddleware{svc, ds, sso}
	return svc, nil
}

func (s Service) SendEmail(mail fleet.Email) error {
	return s.mailService.SendEmail(mail)
}

type validationMiddleware struct {
	fleet.Service
	ds              fleet.Datastore
	ssoSessionStore sso.SessionStore
}

// getAssetURL gets the URL prefix used for retrieving assets from Github. This
// function will determine the appropriate version to use, and create a URL
// prefix for retrieving assets from that tag.
func getAssetURL() template.URL {
	v := version.Version().Version
	tag := strings.Split(v, "-")[0]
	if tag == "unknown" {
		tag = "main"
	}

	return template.URL("https://github.com/fleetdm/fleet/blob/" + tag)
}

// seenHostSet implements synchronized storage for the set of seen hosts.
type seenHostSet struct {
	mutex   sync.Mutex
	hostIDs map[uint]bool
}

func newSeenHostSet() *seenHostSet {
	return &seenHostSet{
		mutex:   sync.Mutex{},
		hostIDs: make(map[uint]bool),
	}
}

// addHostID adds the host identified by ID to the set
func (m *seenHostSet) addHostID(id uint) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.hostIDs[id] = true
}

// getAndClearHostIDs gets the list of unique host IDs from the set and empties
// the set.
func (m *seenHostSet) getAndClearHostIDs() []uint {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	var ids []uint
	for id := range m.hostIDs {
		ids = append(ids, id)
	}
	m.hostIDs = make(map[uint]bool)
	return ids
}
