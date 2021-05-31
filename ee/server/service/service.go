package service

import (
	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/kolide"
	kitlog "github.com/go-kit/kit/log"
)

type Service struct {
	kolide.Service

	ds      kolide.Datastore
	logger  kitlog.Logger
	config  config.KolideConfig
	clock   clock.Clock
	license *kolide.LicenseInfo
}

func NewService(
	svc kolide.Service,
	ds kolide.Datastore,
	logger kitlog.Logger,
	config config.KolideConfig,
	mailService kolide.MailService,
	c clock.Clock,
	license *kolide.LicenseInfo,
) (*Service, error) {
	return &Service{
		Service: svc,
		ds:      ds,
		logger:  logger,
		config:  config,
		clock:   c,
		license: license,
	}, nil
}
