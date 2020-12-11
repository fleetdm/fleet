// Package service holds the implementation of the kolide service interface and the HTTP endpoints
// for the API
package service

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/logging"
	"github.com/fleetdm/fleet/server/sso"
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kit/version"
	"github.com/pkg/errors"
)

// NewService creates a new service from the config struct
func NewService(ds kolide.Datastore, resultStore kolide.QueryResultStore,
	logger kitlog.Logger, config config.KolideConfig, mailService kolide.MailService,
	c clock.Clock, sso sso.SessionStore, lq kolide.LiveQueryStore, carveds kolide.CarveStore) (kolide.Service, error) {
	var svc kolide.Service

	osqueryLogger, err := logging.New(config, logger)
	if err != nil {
		return nil, errors.Wrap(err, "initializing osquery logging")
	}

	svc = service{
		ds:               ds,
		carveds:          carveds,
		resultStore:      resultStore,
		liveQueryStore:   lq,
		logger:           logger,
		config:           config,
		clock:            c,
		osqueryLogWriter: osqueryLogger,
		mailService:      mailService,
		ssoSessionStore:  sso,
		metaDataClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
	svc = validationMiddleware{svc, ds, sso}
	return svc, nil
}

type service struct {
	ds             kolide.Datastore
	carveds        kolide.CarveStore
	resultStore    kolide.QueryResultStore
	liveQueryStore kolide.LiveQueryStore
	logger         kitlog.Logger
	config         config.KolideConfig
	clock          clock.Clock

	osqueryLogWriter *logging.OsqueryLogger

	mailService     kolide.MailService
	ssoSessionStore sso.SessionStore
	metaDataClient  *http.Client
}

func (s service) SendEmail(mail kolide.Email) error {
	return s.mailService.SendEmail(mail)
}

func (s service) Clock() clock.Clock {
	return s.clock
}

type validationMiddleware struct {
	kolide.Service
	ds              kolide.Datastore
	ssoSessionStore sso.SessionStore
}

// getAssetURL gets the URL prefix used for retrieving assets from Github. This
// function will determine the appropriate version to use, and create a URL
// prefix for retrieving assets from that tag.
func getAssetURL() template.URL {
	v := version.Version().Version
	tag := strings.Split(v, "-")[0]
	if tag == "unknown" {
		tag = "master"
	}

	return template.URL("https://github.com/fleetdm/fleet/blob/" + tag)
}
